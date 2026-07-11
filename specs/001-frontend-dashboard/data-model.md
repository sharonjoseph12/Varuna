# Data Model: Frontend Maritime Surveillance Dashboard

All data is ephemeral — streamed via WebSocket or polled via HTTP. No database.

## Entities

### VesselPosition
Received from `ws://localhost:8080/ws/positions` at high volume.

| Field | Type | Description |
|-------|------|-------------|
| vessel_id | string | Unique vessel identifier |
| lat | float | Latitude |
| lon | float | Longitude |
| heading | float | Heading in degrees |
| speed_knots | float | Speed in knots |
| timestamp_ms | integer | Unix timestamp in milliseconds |

**Frontend state**: `Map<vessel_id, VesselPosition>` — only latest position per vessel retained. Flushed to GeoJSON FeatureCollection once per rAF tick.

### Alert
Received from `ws://localhost:8080/ws/alerts` individually.

| Field | Type | Description |
|-------|------|-------------|
| alert_id | string (UUID) | Unique alert identifier |
| type | enum | geofence_breach, suspected_dark_transit, suspected_illegal_fishing, identity_conflict, unresolved_dark_vessel |
| vessel_id | string | Associated vessel |
| timestamp | string (ISO8601) | When the alert was generated |
| position | {lat, lon} | Position at alert time |
| zone | string | Zone name that triggered the alert |
| confidence | float (0-1) | Confidence score |
| evidence | object | {silence_duration_s, boundary_proximity_km, conflicting_position} |
| reasoning_trace | object | {inputs_evaluated[], thresholds_used{}, modalities_available[], engine_version} |
| corroboration | object | {status: none/pending/corroborated, source: string?} |

**Frontend state**: `Alert[]` — newest first, capped at last 100 for UI performance.

### Metrics
Polled from `GET /metrics` every 1-2 seconds.

| Field | Type | Description |
|-------|------|-------------|
| throughput_per_sec | integer | Messages processed per second |
| p50_latency_ms | integer | Median end-to-end latency |
| p99_latency_ms | integer | 99th percentile end-to-end latency |

**Frontend state**: Latest snapshot + trailing array of last 60 readings for chart.

### Zone (Static)
Hardcoded GeoJSON FeatureCollection. Not streamed.

| Field | Type | Description |
|-------|------|-------------|
| name | string | Zone identifier |
| geometry | GeoJSON Polygon | Zone boundary |
| fill_color | string | Translucent fill color |
| stroke_color | string | Boundary line color |

### AlertsDropped (Control Message)
Received from WebSocket when client falls behind.

| Field | Type | Description |
|-------|------|-------------|
| type | "alerts_dropped" | Message type discriminator |
| count | integer | Number of alerts dropped |

## State Transitions

### Alert Expansion State
```
COLLAPSED → EXPANDED (click) → COLLAPSED (click or new selection)
```

### WebSocket Connection State
```
CONNECTING → CONNECTED → DISCONNECTED → RECONNECTING → CONNECTED
```
