# Tasks: Core Processing Engine

**Input**: Design documents from `specs/001-core-engine/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included — the spec explicitly requires test pass for definition of done.

**Organization**: Tasks grouped by user story. Ponytail mode: fewest tasks that cover everything.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Go module init, shared types, zone definitions

- [x] T001 Initialize Go module with `go.mod` and add bbolt dependency in `go.mod`
- [x] T002 [P] Define all shared types (AISMessage, Alert, PositionUpdate, Config, Zone, Metrics, VesselState) in `engine/types.go`
- [x] T003 [P] Define 8 hardcoded zone polygons (Indian Ocean EEZs, marine protected areas) with zone type, hysteresis margin, silence tolerance in `engine/zones.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Grid-hash spatial index + engine scaffolding. ALL user stories depend on these.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Implement grid-hash spatial index: CellID computation, zone-to-cell precomputation, `map[CellID][]*Zone` lookup in `engine/grid.go`
- [x] T005 Implement ray-cast point-in-polygon algorithm in `engine/grid.go`
- [x] T006 Implement core Engine struct, NewEngine constructor (precomputes grid from zones), batched-tick ingestion loop (~20ms ticks), Alerts()/Positions() output channels in `engine/engine.go`
- [x] T007 [P] Implement async bbolt persistence: store goroutine, buffered write channel, alert/track writes in `engine/store.go`

**Checkpoint**: Engine boots, ingests messages in batches, grid-hash resolves cells. No detection logic yet.

---

## Phase 3: User Story 1 - Real-Time Geofence Breach Detection (Priority: P1) 🎯 MVP

**Goal**: Two-tier geofence check with hysteresis. One alert per genuine transition, zero on noise.

**Independent Test**: Feed vessel across boundary → one alert. Jitter within hysteresis → zero alerts.

### Implementation for User Story 1

- [x] T008 [US1] Implement two-tier geofence check: grid-hash cell lookup → ray-cast PIP for boundary cells only, in `engine/geofence.go`
- [x] T009 [US1] Implement zone membership state machine (`map[vesselID]map[zoneID]membershipState`) with hysteresis buffer logic in `engine/geofence.go`
- [x] T010 [US1] Implement geofence alert construction with full reasoning trace (inputs_evaluated, thresholds_used, modalities_available, engine_version) in `engine/geofence.go`
- [x] T011 [US1] Wire geofence engine into batched-tick processing loop in `engine/engine.go`
- [x] T012 [US1] Write geofence tests: one alert per crossing, zero from jitter within hysteresis margin, correct reasoning trace on every alert in `engine/engine_test.go`

**Checkpoint**: Geofence detection works end-to-end. Hysteresis prevents flapping. Every alert has reasoning trace.

---

## Phase 4: User Story 4 - Sustained High-Throughput Ingestion (Priority: P1) 🎯 MVP

**Goal**: ≥ 50k msgs/sec sustained for 120s, p99 < 50ms.

**Independent Test**: Run benchmark, confirm throughput and latency targets.

### Implementation for User Story 4

- [x] T013 [US4] Implement Metrics() method: rolling throughput counter (ring buffer window), p50/p95/p99 latency calculation from latency ring buffer in `engine/engine.go`
- [x] T014 [US4] Add ingestion timestamp tracking and latency measurement (time.Now at ingest → time.Now at channel emit) in `engine/engine.go`
- [x] T015 [US4] Write throughput benchmark: BenchmarkThroughput feeding 50k+ msgs/sec for 120s, assert sustained rate and p99 < 50ms in `engine/engine_test.go`

**Checkpoint**: Engine hits 50k/sec sustained. Metrics are accurate and queryable.

---

## Phase 5: User Story 2 - Dark Transit Detection (Priority: P1)

**Goal**: Absence engine detects silence gaps per zone tolerance. Fires `suspected_dark_transit` before reappearance.

**Independent Test**: Feed vessel near boundary, stop feeding, confirm alert fires before reappearance.

### Implementation for User Story 2

- [x] T016 [US2] Implement per-vessel state: ring buffer (last 32 positions), last-seen timestamp, absence state machine (PRESENT → SUSPICIOUS_DARK → UNRESOLVED) in `engine/absence.go`
- [x] T017 [US2] Implement zone-dependent silence thresholds (coastal: cadence+60s, offshore: cadence×3, open-ocean: conservative) in `engine/absence.go`
- [x] T018 [US2] Implement confidence scoring: silence_ratio, boundary_proximity, historical_gap_pattern, time_of_day in `engine/absence.go`
- [x] T019 [US2] Implement reappearance handling: plausible/implausible position jump, far-side-of-boundary detection, unresolved promotion in `engine/absence.go`
- [x] T020 [US2] Implement absence alert construction with reasoning trace in `engine/absence.go`
- [x] T021 [US2] Wire absence engine into batched-tick loop (run absence checks on each tick) in `engine/engine.go`
- [x] T022 [US2] Write absence engine tests: alert fires before reappearance, zone-dependent thresholds respected, confidence > 0, reasoning trace complete in `engine/engine_test.go`

**Checkpoint**: Absence detection works. Fires before reappearance. Zone-aware thresholds prevent open-ocean false positives.

---

## Phase 6: User Story 3 - Identity Conflict Detection (Priority: P2)

**Goal**: Detect same MMSI broadcasting from kinematically impossible positions.

**Independent Test**: Feed two messages with same MMSI, impossible positions → alert. Plausible positions → no alert.

### Implementation for User Story 3

- [x] T023 [US3] Implement identity-conflict check: last known position per MMSI, distance/elapsed_time vs max vessel speed (50 knots), fire `identity_conflict` with both positions as evidence in `engine/identity.go`
- [x] T024 [US3] Wire identity-conflict check into batched-tick loop in `engine/engine.go`
- [x] T025 [US3] Write identity-conflict tests: fires on impossible pairs, silent on plausible pairs, correct evidence in alert in `engine/engine_test.go`

**Checkpoint**: MMSI spoofing detection works. Cheapest, highest-credibility addition.

---

## Phase 7: User Story 5 - Loitering Detection (Priority: P3)

**Goal**: Detect low-speed + tight-radius behavior near sensitive zones.

**Independent Test**: Feed vessel at < 3 knots within tight radius for extended period → alert.

### Implementation for User Story 5

- [x] T026 [US5] Implement loitering check: rolling speed/radius check on ring buffer, configurable thresholds, fire `suspected_illegal_fishing` with reasoning trace in `engine/loitering.go`
- [x] T027 [US5] Wire loitering check into batched-tick loop in `engine/engine.go`
- [x] T028 [US5] Write loitering test: fires on sustained low-speed loiter, silent on brief slow-downs in `engine/engine_test.go`

**Checkpoint**: Loitering detection works. Pure stateful rule, no ML.

---

## Phase 8: User Story 6 - Alert Corroboration Upgrade (Priority: P3)

**Goal**: `Corroborate()` upgrades alert status for teammate 3's SAR/VIIRS jobs.

**Independent Test**: Create alert, call Corroborate, confirm status update.

### Implementation for User Story 6

- [x] T029 [US6] Implement Corroborate(alertID, source, evidence) method: thread-safe alert lookup + status update in `engine/engine.go`
- [x] T030 [US6] Write corroboration test: status upgrades from "none" to "corroborated" with correct source in `engine/engine_test.go`

**Checkpoint**: Corroboration interface ready for teammate 3.

---

## Phase 9: Entry Point & Integration

**Purpose**: HTTP server with WebSocket endpoints and mini-generator for standalone testing.

- [x] T031 Implement HTTP server with WebSocket fan-out for alerts and positions in `cmd/varuna/main.go`
- [x] T032 Implement built-in mini-generator: 8 vessels, scripted events (dark_transit, boundary_jitter, mmsi_conflict, loiter) in `cmd/varuna/main.go`
- [x] T033 Implement `/api/metrics` JSON endpoint and `/api/trigger/{event}` demo event endpoints in `cmd/varuna/main.go`
- [x] T034 Run `go test ./engine/... -v -count=1` and confirm all tests pass
- [x] T035 Run `go test ./engine/... -bench=BenchmarkThroughput -benchtime=30s` and confirm ≥ 50k msgs/sec

**Checkpoint**: Full standalone binary works. Teammates 2 and 3 can integrate.

---

## Phase 10: Polish & Cross-Cutting Concerns

- [x] T036 [P] Review all alerts have non-empty reasoning traces — add any missing fields
- [x] T037 [P] Add `ponytail:` comments to all deliberate simplifications
- [x] T038 Run quickstart.md validation end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 Geofence (Phase 3)**: Depends on Phase 2 — MVP
- **US4 Throughput (Phase 4)**: Depends on Phase 2 — can parallel with US1
- **US2 Absence (Phase 5)**: Depends on Phase 2 — can parallel with US1/US4
- **US3 Identity (Phase 6)**: Depends on Phase 2 — can parallel with US1/US2/US4
- **US5 Loitering (Phase 7)**: Depends on Phase 2 — can parallel
- **US6 Corroboration (Phase 8)**: Depends on Phase 2
- **Entry Point (Phase 9)**: Depends on all user story phases
- **Polish (Phase 10)**: Depends on Phase 9

### User Story Dependencies

- **US1 (P1)**: Independent after Phase 2
- **US4 (P1)**: Independent after Phase 2
- **US2 (P1)**: Independent after Phase 2 (uses ring buffer from types, not from US1)
- **US3 (P2)**: Independent after Phase 2
- **US5 (P3)**: Independent after Phase 2
- **US6 (P3)**: Independent after Phase 2

### Parallel Opportunities

All user story phases (3-8) can run in parallel after Phase 2 completes. Within each:
- T002/T003 can run in parallel (Phase 1)
- T004/T007 can run in parallel (Phase 2)

---

## Parallel Example: Phase 2 → Phase 3-8

```bash
# After Phase 2 completes, launch all user stories in parallel:
# Stream 1: US1 Geofence (T008-T012)
# Stream 2: US2 Absence (T016-T022)
# Stream 3: US3 Identity (T023-T025)
# Stream 4: US4 Throughput (T013-T015)
# Stream 5: US5 Loitering (T026-T028)
# Stream 6: US6 Corroboration (T029-T030)
```

---

## Implementation Strategy

### MVP First (US1 + US4)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T007)
3. Complete Phase 3: Geofence (T008-T012)
4. Complete Phase 4: Throughput (T013-T015)
5. **STOP and VALIDATE**: Geofence works, 50k/sec sustained

### Full Build

6. Complete Phase 5: Absence (T016-T022)
7. Complete Phase 6: Identity (T023-T025)
8. Complete Phase 7: Loitering (T026-T028)
9. Complete Phase 8: Corroboration (T029-T030)
10. Complete Phase 9: Entry point (T031-T035)
11. Complete Phase 10: Polish (T036-T038)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each user story is independently completable and testable
- Commit after each phase checkpoint
- 38 total tasks. Ponytail: no unnecessary tasks, no scaffolding "for later"
