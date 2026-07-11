# Feature Specification: Frontend Maritime Surveillance Dashboard

**Feature Branch**: `001-frontend-dashboard`

**Created**: 2026-07-11

**Status**: Draft

**Input**: User description: "Build the Varuna frontend dashboard: live map (MapLibre GL JS), vessel rendering from positions WebSocket, throughput + latency performance panel, alert panel with reasoning trace expansion, lead-package JSON export, re-entry cone projection, and boundary-jitter demo control. Consumes ws://localhost:8080/ws/alerts, ws://localhost:8080/ws/positions, and GET /metrics. Must run standalone against mocked data."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Live Maritime Map with Zone Boundaries (Priority: P1)

A demo operator opens the dashboard and immediately sees a live map of the ocean with translucent geofence zone boundaries rendered. Vessels appear as oriented markers moving in real-time. The throughput counter is already past 50,000 and the latency histogram is live. This is the opening shot of the 2:45 pitch — it must work on first load, no clicks required.

**Why this priority**: This is literally the first thing shown in the demo script (0:00–0:20). If the map doesn't load with vessels moving and metrics visible, the demo fails before it starts.

**Independent Test**: Open the dashboard URL. Within 2 seconds, the map renders with zone polygons visible, vessel markers moving, throughput counter climbing, and latency panel updating. No user interaction required.

**Acceptance Scenarios**:

1. **Given** the dashboard is loaded, **When** the positions WebSocket connects, **Then** vessel markers appear on the map oriented by heading within 1 second
2. **Given** 50,000+ position messages/sec are streaming, **When** the user looks at the map, **Then** the UI remains responsive (no visible jank, ≥30 FPS) because the render loop is decoupled from raw message rate via requestAnimationFrame batching
3. **Given** the map is loaded, **When** the user looks at zone boundaries, **Then** hardcoded GeoJSON zones render as translucent colored polygons with visible boundary lines

---

### User Story 2 - Alert Panel with Reasoning Trace (Priority: P1)

A watch officer sees alerts appearing in real-time in a side panel, newest first, color-coded by type (geofence_breach, suspected_dark_transit, suspected_illegal_fishing, identity_conflict, unresolved_dark_vessel). Clicking an alert expands it to show: track history, projected path during silence as a **dotted line** (never solid), the reasoning trace in human-readable form (not raw JSON), and corroboration status.

**Why this priority**: The reasoning trace is the core differentiator — it proves Varuna is auditable, not a black box. The demo script explicitly clicks an alert to show "exactly why it fired" at 0:40–1:20.

**Independent Test**: With mock alert data streaming, alerts appear color-coded. Click any alert — it expands showing readable reasoning text (e.g., "Fired because silence exceeded zone tolerance by 3.2x, vessel was 0.8km from boundary, and this gap is anomalous for this vessel's history") and projected path as dotted line.

**Acceptance Scenarios**:

1. **Given** alerts are streaming via WebSocket, **When** a new alert arrives, **Then** it appears at the top of the list within 500ms, color-coded by its `type` field
2. **Given** an alert is in the list, **When** the user clicks it, **Then** it expands to show: track history, projected dotted path during silence, human-readable reasoning trace (translated from `inputs_evaluated` + `thresholds_used`), and corroboration status
3. **Given** an `identity_conflict` alert is expanded, **When** the user views it, **Then** both conflicting positions are shown on the map simultaneously

---

### User Story 3 - Throughput + Latency Performance Panel (Priority: P1)

The throughput counter and p50/p99 latency values are always visible on screen, updating every 1–2 seconds by polling `/metrics`. The throughput panel visibly shows the count climbing past 50,000 and holding.

**Why this priority**: The demo script opens with this visible — "it's the first thing said in the pitch." Without visible performance metrics on first load, the throughput claim is unsubstantiated.

**Independent Test**: Dashboard loads, performance panel polls `/metrics` every 1–2s, throughput counter shows a number >50,000, p50 and p99 latency values update live.

**Acceptance Scenarios**:

1. **Given** the dashboard is loaded, **When** the metrics endpoint responds, **Then** throughput, p50 latency, and p99 latency are displayed and update every 1–2 seconds
2. **Given** the backend reports throughput_per_sec > 50000, **When** the user looks at the panel, **Then** the counter displays the value prominently and without flicker

---

### User Story 4 - Lead Package Export (Priority: P2)

A watch officer viewing an expanded alert clicks a single button and downloads the alert as structured JSON. The downloaded file contains a visible header/watermark saying "investigative lead — not legal evidence."

**Why this priority**: Demo'd at 1:40–2:00, but the core demo works without it. One-click export, simple JSON blob.

**Independent Test**: Expand any alert, click "Export Lead Package" button, a JSON file downloads. Open the file — it contains the full alert object plus a header field stating "investigative lead — not legal evidence."

**Acceptance Scenarios**:

1. **Given** an alert is expanded, **When** the user clicks the export button, **Then** a JSON file downloads containing the full alert object with `"disclaimer": "investigative lead — not legal evidence"` in the header
2. **Given** the exported JSON, **When** a user opens it, **Then** all fields (reasoning trace, evidence, corroboration status) are present and correctly formatted

---

### User Story 5 - Re-Entry Cone Projection (Priority: P3)

When a `suspected_dark_transit` alert is expanded, a projected cone appears on the map from the vessel's last known position, heading, and speed — showing where it might reappear.

**Why this priority**: Nice-to-have visual that strengthens the dark-transit demo, but items 1–4 are the core. Build after those are solid.

**Independent Test**: Open a dark-transit alert. A translucent cone renders on the map from last known position, oriented by last heading, with radius growing by speed × elapsed time.

**Acceptance Scenarios**:

1. **Given** a `suspected_dark_transit` alert is selected, **When** the alert is expanded, **Then** a translucent cone renders on the map from last known position in the direction of last heading

---

### User Story 6 - Boundary-Jitter Demo Control (Priority: P2)

A small debug control (slider or draggable marker) lets the team manually nudge a synthetic vessel back and forth across a zone edge live on stage. This proves the hysteresis buffer works — zero duplicate alerts despite repeated boundary crossings.

**Why this priority**: Explicitly called for in the demo script and PRD. It's a strong live moment that proves boundary flapping resistance. Judges will try exactly this test.

**Independent Test**: Use the control to move a test vessel back and forth across a zone edge 10 times rapidly. Exactly one alert fires per genuine crossing (entering the zone), zero duplicates from jitter within the hysteresis buffer.

**Acceptance Scenarios**:

1. **Given** the debug control is visible, **When** the user drags a test vessel across a zone boundary, **Then** the vessel position updates on the map in real-time
2. **Given** the test vessel is jittered back and forth within the hysteresis buffer, **When** the user observes the alert panel, **Then** zero duplicate geofence alerts fire

---

### Edge Cases

- What happens when the WebSocket connection drops? → Show a reconnecting indicator, auto-reconnect with exponential backoff
- What happens when the server sends `{"type": "alerts_dropped", "count": N}`? → Show a small non-blocking toast with the count, then resync
- What happens when the positions stream delivers 50k+ msgs/sec? → Render loop uses requestAnimationFrame batching; never re-render per individual message
- What happens when an alert has `corroboration.status: "none"`? → Display "No corroboration available" rather than leaving the field blank
- What happens when the user opens the dashboard before the backend is running? → Show "Connecting to server..." state rather than crashing

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Dashboard MUST render a full-screen interactive map using MapLibre GL JS with no API key required
- **FR-002**: Dashboard MUST display hardcoded GeoJSON zone polygons as translucent boundaries on the map
- **FR-003**: Dashboard MUST connect to `ws://localhost:8080/ws/positions` and render vessel markers (circle/triangle) oriented by heading
- **FR-004**: Dashboard MUST batch position updates via requestAnimationFrame — never re-render per individual WebSocket message
- **FR-005**: Dashboard MUST connect to `ws://localhost:8080/ws/alerts` and display incoming alerts in a panel, newest first, color-coded by `type`
- **FR-006**: Dashboard MUST expand an alert on click showing: track history, projected dotted path, human-readable reasoning trace, and corroboration status
- **FR-007**: Dashboard MUST translate `reasoning_trace.inputs_evaluated` and `thresholds_used` into human-readable sentences, not display raw JSON
- **FR-008**: Dashboard MUST render silent-period projected paths as dotted lines, never solid lines
- **FR-009**: Dashboard MUST poll `GET /metrics` every 1–2 seconds and display throughput_per_sec, p50_latency_ms, and p99_latency_ms
- **FR-010**: Dashboard MUST show throughput and latency panels on screen from initial load without user interaction
- **FR-011**: Dashboard MUST provide a one-click export button on expanded alerts that downloads JSON with `"investigative lead — not legal evidence"` header
- **FR-012**: Dashboard MUST handle `alerts_dropped` messages with a non-blocking toast notification
- **FR-013**: Dashboard MUST include a boundary-jitter debug control (slider or draggable marker) for live demo
- **FR-014**: Dashboard MUST run standalone against mocked WebSocket data with a local mock server
- **FR-015**: Dashboard SHOULD render a projected re-entry cone on the map for expanded `suspected_dark_transit` alerts

### Key Entities

- **Vessel Position**: vessel_id, lat, lon, heading, speed_knots, timestamp_ms — streamed at high volume from positions WebSocket
- **Alert**: alert_id, type, vessel_id, timestamp, position, zone, confidence, evidence, reasoning_trace, corroboration — streamed from alerts WebSocket
- **Zone**: GeoJSON polygon with name — static, hardcoded
- **Metrics**: throughput_per_sec, p50_latency_ms, p99_latency_ms — polled from HTTP endpoint
- **Lead Package**: Full alert JSON plus disclaimer header — exported on demand

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Dashboard loads and displays moving vessels + performance metrics within 2 seconds of opening, no clicks required
- **SC-002**: UI remains responsive (≥30 FPS) while receiving 50,000+ position messages per second
- **SC-003**: New alerts appear in the panel within 500ms of WebSocket delivery
- **SC-004**: Clicking any alert expands a human-readable reasoning trace within 200ms
- **SC-005**: Boundary-jitter demo control produces zero duplicate alerts when a test vessel is jittered across a zone edge
- **SC-006**: Lead package export downloads a valid JSON file in under 1 second
- **SC-007**: Dashboard operates fully standalone against mocked data — never blocked on backend availability

## Assumptions

- Backend WebSocket and HTTP endpoints follow the contracts specified in the PRD (§3.5–§3.7) and the teammate prompt
- MapLibre GL JS is used for map rendering — no API key needed, free and open-source
- React + Vite is the frontend framework (per PRD §4)
- Chart.js is used for performance charts (per PRD §4)
- Native browser WebSocket API is used — no socket.io dependency
- Zone polygons are hardcoded GeoJSON, not authored via UI
- No authentication or login screens are needed
- A local mock server (or replay file) provides mock WebSocket and HTTP data for standalone development
- deck.gl is explicitly cut — MapLibre native rendering is sufficient
- No PDF export — raw JSON is enough
- Teammate 1 owns the Go backend; this dashboard consumes contracts, never modifies backend
