# Research: Frontend Maritime Surveillance Dashboard

## R1: MapLibre GL JS for High-Volume Vessel Rendering

**Decision**: Use MapLibre GL JS with GeoJSON sources updated via `setData()` inside a `requestAnimationFrame` callback. No deck.gl.

**Rationale**: MapLibre natively supports circle and symbol layers with rotation. Updating a single GeoJSON source with all vessel positions in one `setData()` call per frame is the simplest approach that handles 50k msgs/sec — the bottleneck is render, not data. Batching WS messages into a Map keyed by vessel_id, then flushing to GeoJSON once per rAF tick, means the map only re-renders at ~60Hz regardless of message volume.

**Alternatives considered**:
- deck.gl ScatterplotLayer: PRD says "cut first under time pressure." More complex setup, separate coordinate system. Not needed.
- Per-message marker updates: Would choke at 50k/sec. Rejected.

## R2: WebSocket Reconnection Strategy

**Decision**: Native browser WebSocket API with manual reconnect on `close`/`error`. Exponential backoff starting at 1s, max 10s.

**Rationale**: socket.io is explicitly excluded by the PRD. Native WS is one constructor call. Reconnect is ~15 lines.

**Alternatives considered**:
- socket.io: PRD explicitly excludes it.
- No reconnect: Demo could break on flaky connection. Rejected.

## R3: Performance Panel Rendering

**Decision**: Simple numeric display with CSS transitions for the throughput counter. Chart.js line chart for latency history (last 60 data points).

**Rationale**: Chart.js is already in the PRD tech stack. The throughput counter is just a big number that updates — no chart needed. Latency benefits from a small trailing chart to show stability over time.

**Alternatives considered**:
- Full histogram: Overkill for demo. A p50/p99 line chart tells the story.
- No chart library (pure CSS): Harder to read at a glance. Chart.js is already a dependency.

## R4: Alert Color Coding

**Decision**: Fixed color map by alert type:
- `geofence_breach`: #FF6B6B (red)
- `suspected_dark_transit`: #FFB347 (orange)
- `suspected_illegal_fishing`: #9B59B6 (purple)
- `identity_conflict`: #E74C3C (crimson)
- `unresolved_dark_vessel`: #95A5A6 (gray)

**Rationale**: High-contrast, colorblind-distinguishable palette. Fixed mapping = no logic, just a CSS class per type.

## R5: Reasoning Trace Translation

**Decision**: Template-based string builder. Map each `inputs_evaluated` key to a sentence fragment, fill from `thresholds_used` and `evidence`.

**Rationale**: The spec says "not raw JSON — translate into a sentence or two." A simple switch/map over known input keys produces readable English. No NLP, no LLM — deterministic string templates.

**Example output**: "Alert fired because the vessel was silent for 340 seconds (3.2× the zone tolerance of 106s), was 0.8km from the zone boundary (within the 2.0km buffer), and this gap is anomalous compared to this vessel's history in this zone."

## R6: Mock Server

**Decision**: Single Node.js script (`mock-server.js`) using `ws` npm package. Serves both WebSocket endpoints and the HTTP `/metrics` endpoint. Generates synthetic positions at configurable rate, fires scripted alert sequences.

**Rationale**: Must run standalone per spec. A single file mock is the laziest correct solution — no separate package, no docker, no process manager.
