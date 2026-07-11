# Prompt for Teammate 1 — Core Processing Engine (Go)

Paste this whole file into your AI coding assistant (Claude Code or similar) at the start of the build. The full PRD (`Varuna_PRD_v4_FINAL.md`) is your source of truth for anything not covered here — read it first if your assistant supports file upload.

---

## Your role
You own the **processing core**: ingestion, the geofence engine, the absence engine, the identity-conflict engine, alert construction with reasoning trace, and the embedded persistence layer. You do NOT own the frontend, the synthetic generator's internals, or the SAR/VIIRS corroboration jobs — those are teammates 2 and 3. You consume synthetic data through a channel interface teammate 3 will feed, and you expose a channel of alerts + position updates that teammate 2's WebSocket layer will consume. Build to those interfaces so the three of you can work in parallel without touching each other's files.

## Shared contracts — do not change these without telling both teammates

**Inbound AIS message (from the generator, teammate 3):**
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

**Outbound alert object (to teammate 2's WebSocket fan-out):**
```go
type Alert struct {
    AlertID   string  `json:"alert_id"`
    Type      string  `json:"type"` // geofence_breach | suspected_dark_transit | suspected_illegal_fishing | identity_conflict | unresolved_dark_vessel
    VesselID  string  `json:"vessel_id"`
    Timestamp string  `json:"timestamp"` // ISO8601
    Position  struct{ Lat, Lon float64 } `json:"position"`
    Zone      string  `json:"zone"`
    Confidence float64 `json:"confidence"`
    Evidence  map[string]interface{} `json:"evidence"`
    ReasoningTrace struct {
        InputsEvaluated  []string          `json:"inputs_evaluated"`
        ThresholdsUsed   map[string]float64 `json:"thresholds_used"`
        ModalitiesAvailable []string       `json:"modalities_available"`
        EngineVersion    string            `json:"engine_version"`
    } `json:"reasoning_trace"`
    Corroboration struct {
        Status string  `json:"status"` // none | pending | corroborated
        Source *string `json:"source"`
    } `json:"corroboration"`
}
```

**Your package's public surface (what teammates 2 and 3 will import/call):**
```go
// package engine
func NewEngine(cfg Config, zones []Zone) *Engine
func (e *Engine) Ingest(in <-chan AISMessage)          // consume from generator
func (e *Engine) Alerts() <-chan Alert                  // teammate 2 subscribes here
func (e *Engine) Positions() <-chan PositionUpdate       // teammate 2 subscribes here for live map
func (e *Engine) Corroborate(alertID string, source string, evidence map[string]interface{}) // teammate 3's SAR/VIIRS job calls this to upgrade an alert
func (e *Engine) Metrics() Metrics                       // throughput + latency histogram for the dashboard panel
```

## What to build, in order (non-negotiable — do not skip or reorder)
1. Batched-tick ingestion: accumulate messages for ~20ms per tick, process as a batch, not one at a time. This is the primary lever for the 50k/sec floor.
2. Grid-hash spatial index: fixed-size lat/lon cells, each zone polygon precomputes overlapping cells at startup. O(1) lookup per incoming position.
3. Two-tier geofence check: grid-hash filter first, ray-cast point-in-polygon only for vessels in a boundary cell.
4. **Boundary hysteresis buffer**: a zone-membership transition only commits once a position clears a configurable margin past the raw polygon edge, not on raw crossing. This prevents duplicate alerts from GPS/AIS position noise — test it by feeding a synthetic vessel jittering back and forth across a boundary and confirming exactly one alert fires per real crossing, zero on noise.
5. State-machine alerting: `{vessel_id → zone_membership_set}`. Fire exactly one alert per genuine transition.
6. Absence engine: per-vessel ring buffer, last-seen timestamp, zone-dependent silence tolerance (coastal/offshore/open-ocean tiers — see PRD §3.3 for the threshold table), confidence scoring from silence ratio + boundary proximity + historical gap pattern.
7. Identity-conflict engine: track last known position per MMSI; if a new message under the same MMSI reports a position unreachable within the elapsed time given a generous max-vessel-speed bound, fire `identity_conflict` with both positions as evidence. This reuses state you already have — should take under an hour.
8. Reasoning trace: every alert must carry which inputs were evaluated, what thresholds they were compared against, and which modalities were available at fire-time. Not optional — this is graded in the demo.
9. Embedded persistence: bbolt or sqlite, write alerts + tracks async, never blocking the hot path.
10. `Metrics()`: sustained throughput (rolling window), p50/p95/p99 end-to-end latency (ingestion timestamp → alert/position emitted onto your output channel — teammate 2 adds the WebSocket-delivery leg on top of this).

## Explicitly do not build
No message broker, no Docker, no auth, no zone-authoring UI (hardcoded GeoJSON zones are fine — teammate 3 or you can hand-write 8 zones). No live public AIS feed.

## Definition of done for your piece
- `go test ./...` passes, including: one alert per scripted crossing, zero alerts from boundary jitter within the hysteresis margin, `identity_conflict` fires only on genuinely impossible position pairs, every emitted alert has a non-empty reasoning trace.
- Sustained 50k synthetic messages/sec for 120 seconds on your own machine, with throughput and p99 latency numbers you can quote from memory before the demo.
- Your two output channels (`Alerts()`, `Positions()`) are stable and documented so teammate 2 can wire them up without needing you in the room.

If you're blocked on anything outside this scope, don't guess at teammate 2 or 3's internals — ask them, or check `Varuna_PRD_v4_FINAL.md` §2–§3 for the full spec.
