# Feature Specification: Core Processing Engine

**Feature Branch**: `001-core-engine`

**Created**: 2026-07-11

**Status**: Draft

**Input**: User description: "Build the Varuna core processing engine — Teammate 1 scope: batched-tick ingestion, grid-hash spatial index, geofence engine with hysteresis, absence engine with zone-aware thresholds, identity-conflict engine, loitering detection, alert construction with reasoning traces, embedded persistence, and metrics — as defined in Prompt_1_Core_Engine.md and Varuna_PRD_v4_FINAL.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Real-Time Geofence Breach Detection (Priority: P1)

A maritime surveillance operator monitors vessel traffic across 8 protected zones. When a vessel crosses a zone boundary, the system fires exactly one alert per genuine transition, with full reasoning trace. Vessels jittering near a boundary due to GPS noise do NOT produce duplicate alerts thanks to hysteresis buffering.

**Why this priority**: Core function. Without geofence detection, the demo fails entirely.

**Independent Test**: Feed a synthetic vessel across a boundary — confirm one `geofence_breach` alert fires. Feed a vessel jittering back and forth within the hysteresis margin — confirm zero alerts.

**Acceptance Scenarios**:

1. **Given** a vessel outside Zone A, **When** it crosses into Zone A by more than the hysteresis margin, **Then** exactly one `geofence_breach` alert fires with type, position, zone name, confidence, and reasoning trace.
2. **Given** a vessel near Zone A's boundary, **When** its position jitters back and forth within the hysteresis buffer, **Then** zero alerts fire.
3. **Given** a vessel inside Zone A, **When** it exits by more than the hysteresis margin, **Then** exactly one exit alert fires.

---

### User Story 2 - Dark Transit Detection (Priority: P1)

A vessel near a protected zone boundary goes silent (stops broadcasting AIS). The absence engine detects the silence gap exceeds the zone-dependent tolerance and fires a `suspected_dark_transit` alert BEFORE the vessel reappears, with confidence score and reasoning trace.

**Why this priority**: Key differentiator vs. geofence-only systems. The demo script depends on this firing before reappearance.

**Independent Test**: Feed a vessel with positions near a boundary, then stop feeding for longer than the zone tolerance. Confirm `suspected_dark_transit` fires with non-zero confidence and full reasoning trace.

**Acceptance Scenarios**:

1. **Given** a vessel last seen near Zone B boundary, **When** silence exceeds the zone's tolerance threshold, **Then** `suspected_dark_transit` fires with confidence > 0, reasoning trace showing silence_ratio, boundary_proximity, and thresholds used.
2. **Given** a vessel in open ocean with no nearby boundaries, **When** silence exceeds coastal tolerance but not open-ocean tolerance, **Then** no alert fires (zone-dependent thresholds respected).
3. **Given** a vessel that went dark and reappears on the far side of the boundary, **Then** confidence is escalated and `suspected_zone_crossing` flag is set.

---

### User Story 3 - Identity Conflict Detection (Priority: P2)

Two AIS messages arrive under the same MMSI but from positions that are physically unreachable given the elapsed time and a generous max-vessel-speed bound. The system fires `identity_conflict` with both positions as evidence.

**Why this priority**: Cheap to build (reuses existing state), high credibility, closes a gap no geofence/absence system catches.

**Independent Test**: Feed two messages with the same MMSI, 100 km apart, 10 seconds apart. Confirm `identity_conflict` fires. Feed two messages with the same MMSI at plausible speed — confirm no alert.

**Acceptance Scenarios**:

1. **Given** MMSI "123456789" last seen at (10.0, 72.0), **When** a new message for the same MMSI arrives at (12.0, 74.0) 30 seconds later (impossible at max vessel speed), **Then** `identity_conflict` fires with both positions in evidence.
2. **Given** MMSI "123456789" last seen at (10.0, 72.0), **When** a new message arrives at (10.001, 72.001) 60 seconds later (plausible), **Then** no alert fires.

---

### User Story 4 - Sustained High-Throughput Ingestion (Priority: P1)

The engine sustains ≥ 50,000 messages/sec for 120 seconds with p99 end-to-end latency < 50ms, processing batched ticks of ~20ms.

**Why this priority**: Non-negotiable benchmark requirement. Demo fails without it.

**Independent Test**: Run benchmark feeding 50k+ msgs/sec for 120s. Measure throughput and p50/p95/p99 latency. Confirm sustained rate and latency targets.

**Acceptance Scenarios**:

1. **Given** a synthetic AIS feed at 50,000+ msgs/sec, **When** running for 120 seconds, **Then** throughput counter shows ≥ 50k sustained, p99 latency < 50ms.
2. **Given** 8 active zone polygons and batched-tick processing, **When** under sustained load, **Then** zero dropped messages, zero missed state transitions.

---

### User Story 5 - Loitering Detection (Priority: P3)

A vessel maintains low speed within a tight radius near a sensitive zone for an extended period. The system fires `suspected_illegal_fishing`.

**Why this priority**: Nice-to-have beyond core detection engines. Pure stateful rule on existing ring buffer.

**Independent Test**: Feed a vessel at < 3 knots within a small radius near a zone for longer than the configured time window. Confirm `suspected_illegal_fishing` fires.

**Acceptance Scenarios**:

1. **Given** a vessel inside Zone C at 2 knots, **When** it stays within 500m radius for 30+ minutes, **Then** `suspected_illegal_fishing` fires with zone and reasoning trace.

---

### User Story 6 - Alert Corroboration Upgrade (Priority: P3)

Teammate 3's SAR/VIIRS job calls `Corroborate(alertID, source, evidence)` to upgrade an existing alert's corroboration status from "none" to "corroborated".

**Why this priority**: Interface contract for teammate integration. Simple to build.

**Independent Test**: Create an alert, call `Corroborate` with SAR evidence, confirm alert's corroboration status updates to "corroborated" with source.

**Acceptance Scenarios**:

1. **Given** an alert with corroboration status "none", **When** `Corroborate("alert-123", "SAR", evidence)` is called, **Then** alert status updates to "corroborated" with source "SAR".

---

### Edge Cases

- What happens when a vessel enters two overlapping zones simultaneously? → Two separate alerts, one per zone.
- What happens when the same vessel crosses the same boundary twice in rapid succession but beyond hysteresis? → Two alerts, one per genuine crossing.
- What happens when AIS messages arrive out of timestamp order? → Use message timestamp, not arrival time. Skip stale messages older than last-seen.
- What happens when the engine receives a message for a vessel it has never seen? → Initialize state, no alert on first sighting.
- What happens when the bbolt/sqlite write goroutine falls behind? → Bounded channel, drop oldest writes rather than blocking the hot path.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST process incoming AIS messages in batched ticks of ~20ms, not one at a time.
- **FR-002**: System MUST maintain a grid-hash spatial index with O(1) cell lookup per position.
- **FR-003**: System MUST perform two-tier geofence checks: grid-hash filter first, ray-cast PIP only for boundary cells.
- **FR-004**: System MUST implement boundary hysteresis buffering — zone transitions commit only after clearing a configurable margin past the raw polygon edge.
- **FR-005**: System MUST track zone membership per vessel as a state machine, firing exactly one alert per genuine transition.
- **FR-006**: System MUST implement per-vessel absence detection with zone-dependent silence thresholds (coastal: cadence + 60s, offshore: cadence × 3, open ocean: conservative).
- **FR-007**: System MUST score absence-alert confidence using silence ratio, boundary proximity, historical gap pattern, and time-of-day.
- **FR-008**: System MUST detect identity conflicts when the same MMSI reports kinematically impossible positions.
- **FR-009**: System MUST detect loitering (low speed + tight radius + extended duration near a sensitive zone).
- **FR-010**: System MUST attach a complete reasoning trace to every alert: inputs evaluated, thresholds used, modalities available, engine version.
- **FR-011**: System MUST persist alerts and tracks asynchronously to embedded storage (bbolt or sqlite), never blocking the hot path.
- **FR-012**: System MUST expose `Alerts()` and `Positions()` output channels for teammate 2's WebSocket layer.
- **FR-013**: System MUST expose `Corroborate(alertID, source, evidence)` for teammate 3's SAR/VIIRS jobs.
- **FR-014**: System MUST expose `Metrics()` returning sustained throughput and p50/p95/p99 latency.
- **FR-015**: System MUST handle vessel reappearance after dark transit: plausible position → reduce confidence; implausible jump → escalate; far side of boundary → flag suspected_zone_crossing; no reappearance → promote to unresolved_dark_vessel.

### Key Entities

- **AISMessage**: Inbound vessel position report (vessel_id, mmsi, lat, lon, heading, speed_knots, timestamp_ms)
- **Alert**: Outbound detection result with type, confidence, evidence, reasoning trace, corroboration status
- **PositionUpdate**: Live vessel position for map rendering
- **Zone**: Geographic polygon with precomputed grid cells, zone type (coastal/offshore/open-ocean), hysteresis margin
- **VesselState**: Per-vessel tracking: ring buffer of positions, zone membership set, absence state, last-seen timestamp, historical gap pattern
- **Metrics**: Rolling throughput, p50/p95/p99 latency histograms

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sustained throughput ≥ 50,000 messages/sec for 120 seconds on target hardware.
- **SC-002**: End-to-end p99 latency (ingestion → alert/position emitted to output channel) < 50ms.
- **SC-003**: Exactly 1 alert per scripted geofence crossing. Zero duplicates, zero misses.
- **SC-004**: Zero duplicate alerts when a vessel jitters within the hysteresis buffer.
- **SC-005**: `suspected_dark_transit` fires before vessel reappears, with confidence > 0 and non-empty reasoning trace.
- **SC-006**: `identity_conflict` fires on every scripted impossible-position pair, never on normal traffic.
- **SC-007**: Every emitted alert has a non-empty, complete reasoning trace.
- **SC-008**: `Alerts()` and `Positions()` channels are stable — teammate 2 can wire them without coordination.
- **SC-009**: All automated tests pass: `go test ./...`

## Assumptions

- Go 1.21+ is available on the build machine.
- Only external dependency is `go.etcd.io/bbolt` for embedded persistence; everything else is stdlib.
- 8 hardcoded GeoJSON zones are sufficient (no zone-authoring UI needed).
- Teammate 3 provides AIS messages via a Go channel implementing the `AISMessage` contract. A built-in mini-generator exists for standalone testing.
- Teammate 2 consumes `Alerts()` and `Positions()` channels via WebSocket fan-out (not our scope).
- No Docker, no message broker, no auth, no live public AIS feed.
- Ponytail mode: laziest correct solution, stdlib first, no unnecessary abstractions.
