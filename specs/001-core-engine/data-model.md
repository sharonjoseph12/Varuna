# Data Model: Core Processing Engine

## Entities

### AISMessage (Inbound)
| Field | Type | Description |
|-------|------|-------------|
| VesselID | string | Unique vessel identifier |
| MMSI | string | Maritime Mobile Service Identity |
| Lat | float64 | Latitude in decimal degrees |
| Lon | float64 | Longitude in decimal degrees |
| HeadingDeg | float64 | Heading in degrees (0-360) |
| SpeedKnots | float64 | Speed over ground in knots |
| TimestampMs | int64 | Unix timestamp in milliseconds |

### Alert (Outbound)
| Field | Type | Description |
|-------|------|-------------|
| AlertID | string | UUID |
| Type | string | geofence_breach \| suspected_dark_transit \| suspected_illegal_fishing \| identity_conflict \| unresolved_dark_vessel |
| VesselID | string | Vessel that triggered the alert |
| Timestamp | string | ISO8601 |
| Position | {Lat, Lon float64} | Position at alert time |
| Zone | string | Zone name involved |
| Confidence | float64 | 0.0–1.0 |
| Evidence | map[string]interface{} | Alert-type-specific evidence |
| ReasoningTrace | ReasoningTrace | Full audit trail |
| Corroboration | Corroboration | Corroboration status |

### ReasoningTrace (Embedded in Alert)
| Field | Type | Description |
|-------|------|-------------|
| InputsEvaluated | []string | Which inputs were checked |
| ThresholdsUsed | map[string]float64 | Exact threshold values |
| ModalitiesAvailable | []string | Data sources available at fire time |
| EngineVersion | string | Engine version string |

### Corroboration (Embedded in Alert)
| Field | Type | Description |
|-------|------|-------------|
| Status | string | none \| pending \| corroborated |
| Source | *string | e.g., "SAR", "VIIRS" (nil if none) |

### PositionUpdate (Outbound)
| Field | Type | Description |
|-------|------|-------------|
| VesselID | string | Vessel identifier |
| Lat | float64 | Current latitude |
| Lon | float64 | Current longitude |
| HeadingDeg | float64 | Current heading |
| SpeedKnots | float64 | Current speed |
| TimestampMs | int64 | Position timestamp |

### Zone (Configuration)
| Field | Type | Description |
|-------|------|-------------|
| ID | string | Zone identifier |
| Name | string | Human-readable name |
| Type | string | coastal \| offshore \| open_ocean |
| Polygon | [][2]float64 | Ordered vertices (lat, lon) |
| HysteresisMarginDeg | float64 | Margin in degrees (~100m default) |
| SilenceToleranceSec | int64 | Zone-dependent absence threshold |
| BoundaryBufferKm | float64 | Proximity buffer for absence engine |
| GridCells | []CellID | Precomputed overlapping grid cells |

### VesselState (Internal)
| Field | Type | Description |
|-------|------|-------------|
| VesselID | string | Vessel identifier |
| MMSI | string | Current MMSI |
| Positions | [32]AISMessage | Ring buffer of last 32 positions |
| PosIdx | int | Current ring buffer index |
| PosCount | int | Number of positions stored |
| ZoneMembership | map[string]bool | Current zone membership set |
| AbsenceState | string | PRESENT \| SUSPICIOUS_DARK \| UNRESOLVED |
| LastSeen | int64 | Last-seen timestamp (ms) |
| GapHistory | []int64 | Historical gap durations for this vessel |

### CellID (Internal)
| Field | Type | Description |
|-------|------|-------------|
| LatCell | int | Integer cell index for latitude |
| LonCell | int | Integer cell index for longitude |

### Config
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| TickIntervalMs | int | 20 | Batch tick interval |
| GridCellSizeDeg | float64 | 0.1 | Grid cell size in degrees |
| MaxVesselSpeedKnots | float64 | 50 | Upper bound for identity conflict |
| DefaultHysteresisMarginDeg | float64 | 0.001 | ~100m hysteresis margin |
| LoiterSpeedThreshold | float64 | 3.0 | Knots below which loitering starts |
| LoiterRadiusM | float64 | 500 | Max radius for loitering |
| LoiterTimeWindowMin | int | 30 | Minutes to trigger loitering |
| AlertChannelSize | int | 10000 | Alert output channel buffer |
| PositionChannelSize | int | 50000 | Position output channel buffer |
| StoreWriteBufferSize | int | 1000 | Async store write buffer |

### Metrics (Outbound)
| Field | Type | Description |
|-------|------|-------------|
| ThroughputMsgsSec | float64 | Rolling throughput |
| LatencyP50Ms | float64 | p50 end-to-end latency |
| LatencyP95Ms | float64 | p95 end-to-end latency |
| LatencyP99Ms | float64 | p99 end-to-end latency |
| TotalProcessed | int64 | Total messages processed |
| TotalAlerts | int64 | Total alerts fired |

## State Transitions

### Zone Membership (per vessel per zone)
```
OUTSIDE --[position inside zone + cleared hysteresis margin]--> INSIDE  (fires geofence_breach)
INSIDE  --[position outside zone + cleared hysteresis margin]--> OUTSIDE (fires geofence_exit)
```

### Absence State (per vessel)
```
PRESENT --[silence > zone tolerance AND near boundary]--> SUSPICIOUS_DARK (fires suspected_dark_transit)
SUSPICIOUS_DARK --[reappears plausibly]--> PRESENT (confidence unchanged/reduced)
SUSPICIOUS_DARK --[reappears implausibly]--> PRESENT (confidence escalated, kinematic_anomaly flag)
SUSPICIOUS_DARK --[reappears far side of boundary]--> PRESENT (confidence escalated, suspected_zone_crossing)
SUSPICIOUS_DARK --[no reappearance within window]--> UNRESOLVED (fires unresolved_dark_vessel)
UNRESOLVED --[reappears]--> PRESENT
```
