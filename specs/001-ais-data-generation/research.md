# Research: AIS Data Generation, Corroboration Jobs, Benchmark & Demo

## Great-Circle Movement

**Decision**: Use the Haversine/inverse-Haversine formula from stdlib `math` to advance vessel positions along great-circle arcs.
**Rationale**: Zero external GIS dependency; the formula is ~15 lines and sufficient for realistic vessel movement over 24-hour demo distances.
**Alternatives considered**: `go-geo` library — rejected, adds a dependency for what stdlib math covers cleanly.

## AIS Cadence Tiers

**Decision**: Match PRD §3.3 zone tiers:
- Coastal: 2–10s random interval
- Offshore: 10–30min random interval
- Open ocean: 30–120min random interval
**Rationale**: Zone-aware cadence is explicitly required by the PRD and the absence engine depends on it.

## Scripted Event Triggering

**Decision**: Lightweight `net/http` server on a separate port (default `:8081`) serving `/trigger/{event}` endpoints. Event state stored in a mutex-protected map per vessel.
**Rationale**: HTTP trigger is judge-controllable (curl, browser, Postman), repeatable without restart, and requires only stdlib.
**Alternatives considered**: CLI keypress — rejected, not remote-controllable during a live demo.

## Throughput Architecture

**Decision**: Generator maintains a pool of goroutines (one per vessel) each advancing state and writing to a single buffered `chan AISMessage` shared with the engine. Buffer size = 10,000 (roughly 200ms at 50k/s).
**Rationale**: Goroutine-per-vessel maps cleanly onto Go's scheduler; buffered channel decouples generator from consumer without an external broker.
**Alternatives considered**: Single goroutine batch-building — works but serializes vessel state updates; per-vessel goroutines parallelize naturally.

## SAR Ship Detector

**Decision**: Use a pre-trained SAR ship detector from HuggingFace (`HRSID`-trained YOLOv8n, exported to ONNX) called via a Python subprocess (`python sar_infer.py <tile_path> <lat> <lon>`). The Go SAR job execs the subprocess and parses its JSON stdout.
**Rationale**: Avoids CGo complexity of `onnxruntime_go` bindings; Python subprocess is a clean boundary; model never touches the hot path.
**Alternatives considered**: `onnxruntime_go` — more integrated but adds C library linkage complexity that's risky in a 24h window. CGo = rejected per ponytail.
**ponytail**: subprocess boundary, upgrade to native Go ONNX binding if inference latency matters.

## VIIRS Stub Design

**Decision**: Static JSON blob in `viirs_stub.go` returning a hardcoded detection record with `stub: true`. Same function signature as SAR job.
**Rationale**: Proves the fusion interface without fabricating capability. Explicitly labeled stub in code and demo.

## Benchmark Methodology

**Decision**: `benchmark/harness.go` uses `time.Now()` timestamps injected into each `AISMessage.TimestampMs` at generation, and records the WebSocket delivery timestamp at the client. p50/p95/p99 computed from a sliding window of latency samples.
**Rationale**: End-to-end latency (ingestion → WebSocket delivery) is the metric the PRD specifies, not just geometry-check duration.

## Zones GeoJSON

**Decision**: 8 hand-coded polygons covering realistic maritime enforcement areas (Bay of Bengal, Strait of Malacca, Persian Gulf, South China Sea, Gulf of Guinea, Mediterranean EEZ, North Sea, Barents Sea approaches). Each has `zone_type` and `silence_tolerance_s` matching PRD §3.3 tiers.
**Rationale**: Hardcoded GeoJSON is the PRD-explicit approach; no zone-authoring UI needed. File shared verbatim between engine and frontend.
