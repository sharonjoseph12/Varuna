# Engine API Contract

This is the public surface teammates 2 and 3 import and call.

## Package: `engine`

### Constructor

```go
func NewEngine(cfg Config, zones []Zone) *Engine
```

Creates and initializes the engine. Precomputes grid-hash spatial index from zones. Does not start processing — call `Ingest()` to begin.

### Ingestion (Teammate 3 → Engine)

```go
func (e *Engine) Ingest(in <-chan AISMessage)
```

Blocks, reading from `in` until the channel is closed. Internally batches messages on ~20ms ticks. Call in a goroutine.

### Alert Stream (Engine → Teammate 2)

```go
func (e *Engine) Alerts() <-chan Alert
```

Returns a read-only channel of alerts. Teammate 2 subscribes here for WebSocket fan-out. Buffered (10k default). If consumer falls behind, oldest alerts are dropped (engine never blocks).

### Position Stream (Engine → Teammate 2)

```go
func (e *Engine) Positions() <-chan PositionUpdate
```

Returns a read-only channel of live vessel positions. Teammate 2 subscribes here for map rendering. Buffered (50k default).

### Corroboration (Teammate 3 → Engine)

```go
func (e *Engine) Corroborate(alertID string, source string, evidence map[string]interface{})
```

Thread-safe. Upgrades an existing alert's corroboration status to "corroborated" with the given source and evidence. Called by teammate 3's SAR/VIIRS background jobs.

### Metrics

```go
func (e *Engine) Metrics() Metrics
```

Returns current throughput and latency statistics. Thread-safe snapshot.

## HTTP Endpoints (cmd/varuna)

These are served by `cmd/varuna/main.go` for standalone testing and teammate integration:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ws/alerts` | WebSocket stream of Alert JSON |
| GET | `/ws/positions` | WebSocket stream of PositionUpdate JSON |
| GET | `/api/metrics` | JSON snapshot of Metrics |
| POST | `/api/trigger/{event}` | Trigger scripted demo events: `dark_transit`, `boundary_jitter`, `mmsi_conflict`, `loiter` |

## JSON Wire Format

Alert JSON matches the struct tags in `types.go`:

```json
{
  "alert_id": "uuid",
  "type": "geofence_breach",
  "vessel_id": "V-001",
  "timestamp": "2026-07-11T14:15:00Z",
  "position": {"lat": 10.5, "lon": 72.3},
  "zone": "Gulf of Mannar MPA",
  "confidence": 0.95,
  "evidence": {},
  "reasoning_trace": {
    "inputs_evaluated": ["zone_membership_transition"],
    "thresholds_used": {"hysteresis_margin_deg": 0.001},
    "modalities_available": ["ais"],
    "engine_version": "geofence-v1"
  },
  "corroboration": {"status": "none", "source": null}
}
```
