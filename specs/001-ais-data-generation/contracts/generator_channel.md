# Contract: Generator → Engine Channel

## Interface

```go
// package generator
type Config struct {
    VesselCount    int
    OutputBuffer   int    // channel buffer size, default 10000
    TriggerPort    int    // HTTP trigger server port, default 8081
    DataDir        string // path to data/sar and data/models
}

func NewGenerator(cfg Config) *Generator
func (g *Generator) Run(out chan<- AISMessage)
// Run is blocking; call as: go g.Run(ch)

// Teammate 1 engine contract (must match exactly):
// engine.Ingest(in <-chan AISMessage)
// engine.Corroborate(alertID string, source string, evidence map[string]interface{})
```

## HTTP Trigger Endpoints (port 8081)

| Method | Path | Query Params | Description |
|--------|------|-------------|-------------|
| POST | /trigger/dark-transit | vessel=ID, variant=plausible\|jump, silence_s=N | Start dark transit for vessel |
| POST | /trigger/loitering | vessel=ID, duration_s=N | Start loitering event |
| POST | /trigger/duplicate-mmsi | vessel=ID, target_mmsi=MMSI | Start MMSI impersonation |
| POST | /trigger/geofence-crossing | vessel=ID, zone=NAME | Direct vessel toward zone boundary |
| POST | /trigger/reset | vessel=ID | Cancel active event for vessel |
| GET | /status | — | JSON status of all active events |

## Response Format

All trigger endpoints return:
```json
{
  "ok": true,
  "vessel_id": "vessel-042",
  "event": "dark_transit",
  "params": { "silence_s": 45, "variant": "plausible" }
}
```
