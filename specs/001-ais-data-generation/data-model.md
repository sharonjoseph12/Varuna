# Data Model: AIS Data Generation, Corroboration Jobs, Benchmark & Demo

## AISMessage (Go struct — shared contract with teammate 1)

```go
type AISMessage struct {
    VesselID    string  `json:"vessel_id"`
    MMSI        string  `json:"mmsi"`
    Lat         float64 `json:"lat"`
    Lon         float64 `json:"lon"`
    HeadingDeg  float64 `json:"heading"`
    SpeedKnots  float64 `json:"speed_knots"`
    TimestampMs int64   `json:"timestamp_ms"`
}
```

## VesselState (internal generator state)

```go
type VesselState struct {
    ID             string
    MMSI           string
    Lat, Lon       float64
    HeadingDeg     float64
    SpeedKnots     float64
    CadenceMs      int64         // ms between updates
    ZoneType       string        // "coastal" | "offshore" | "open_ocean"
    ActiveEvent    ScriptedEvent
    EventParams    map[string]interface{}
    DarkUntil      time.Time     // zero if not dark
    DuplicateMMSI  string        // non-empty if impersonating another vessel
}
```

## ScriptedEvent (enum)

```go
type ScriptedEvent int

const (
    EventNone ScriptedEvent = iota
    EventDarkTransit
    EventLoitering
    EventDuplicateMMSI
    EventGeofenceCrossing
)
```

## Zone (GeoJSON Feature properties)

```json
{
  "name": "string",
  "zone_type": "coastal | offshore | open_ocean",
  "silence_tolerance_s": 120,
  "boundary_buffer_km": 2.0
}
```

## SARTile

```go
type SARTile struct {
    TileID   string  `json:"tile_id"`
    FilePath string  `json:"file_path"`
    MinLat   float64 `json:"min_lat"`
    MaxLat   float64 `json:"max_lat"`
    MinLon   float64 `json:"min_lon"`
    MaxLon   float64 `json:"max_lon"`
}
```

## CorroborationEvidence

```go
type CorroborationEvidence struct {
    Source              string    `json:"source"`             // "sar" | "viirs"
    TileID              string    `json:"tile_id,omitempty"`
    DetectionConfidence float64   `json:"detection_confidence"`
    BoundingBoxPixels   []int     `json:"bounding_box_pixels,omitempty"`
    ModelVersion        string    `json:"model_version,omitempty"`
    Stub                bool      `json:"stub,omitempty"`
    DetectedAt          time.Time `json:"detected_at"`
}
```

## BenchmarkResult

```go
type BenchmarkResult struct {
    HardwareCPU      string    `json:"hardware_cpu"`
    HardwareCores    int       `json:"hardware_cores"`
    HardwareRAMGB    float64   `json:"hardware_ram_gb"`
    MessageSizeBytes int       `json:"message_size_bytes"`
    ZoneCount        int       `json:"zone_count"`
    WSClientCount    int       `json:"ws_client_count"`
    DurationS        int       `json:"duration_s"`
    ThroughputMsgSec float64   `json:"throughput_msg_sec"`
    P50LatencyMs     float64   `json:"p50_latency_ms"`
    P95LatencyMs     float64   `json:"p95_latency_ms"`
    P99LatencyMs     float64   `json:"p99_latency_ms"`
    AlertsExpected   int       `json:"alerts_expected"`
    AlertsFired      int       `json:"alerts_fired"`
    FalsePositives   int       `json:"false_positives"`
    DroppedMessages  int       `json:"dropped_messages"`
    RunAt            time.Time `json:"run_at"`
}
```

## State Transitions

### VesselState.ActiveEvent
```
None → DarkTransit   (HTTP trigger /trigger/dark-transit?vessel=X&variant=plausible|jump)
None → Loitering     (HTTP trigger /trigger/loitering?vessel=X)
None → DuplicateMMSI (HTTP trigger /trigger/duplicate-mmsi?vessel=X&target_mmsi=Y)
None → GeofenceCrossing (HTTP trigger /trigger/geofence-crossing?vessel=X&zone=Z)
Any → None           (event window expires or manual reset)
```
