# Tasks: Frontend Maritime Surveillance Dashboard

**Input**: Design documents from `specs/001-frontend-dashboard/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested — manual demo verification only.

**Organization**: Tasks grouped by user story for independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: React + Vite project initialization, mock server, design system

- [ ] T001 Initialize React + Vite project in `dashboard/` with `npx -y create-vite@latest ./ -- --template react`
- [ ] T002 Install dependencies: `maplibre-gl`, `chart.js`, `react-chartjs-2`, `ws` (dev dep for mock server)
- [ ] T003 [P] Create design system tokens and global styles in `dashboard/src/styles/index.css`
- [ ] T004 [P] Create hardcoded GeoJSON zone data in `dashboard/src/utils/zones.js`
- [ ] T005 [P] Create reasoning trace translator in `dashboard/src/utils/reasoningTrace.js`
- [ ] T006 [P] Create lead package export utility in `dashboard/src/utils/exportLead.js`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: WebSocket/HTTP hooks and mock server — MUST be complete before any UI story

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T007 Create generic WebSocket hook with reconnect + exponential backoff in `dashboard/src/hooks/useWebSocket.js`
- [ ] T008 [P] Create batched positions hook using rAF in `dashboard/src/hooks/usePositions.js` (depends on T007)
- [ ] T009 [P] Create alerts hook with dropped-alert handling in `dashboard/src/hooks/useAlerts.js` (depends on T007)
- [ ] T010 [P] Create metrics polling hook in `dashboard/src/hooks/useMetrics.js`
- [ ] T011 Create standalone mock server serving both WebSockets and `/metrics` in `dashboard/mock-server.js`
- [ ] T012 Create root App layout (map + side panels) in `dashboard/src/App.jsx` and `dashboard/src/App.css`

**Checkpoint**: Mock server running, hooks connecting, empty layout renders

---

## Phase 3: User Story 1 — Live Maritime Map with Zone Boundaries (Priority: P1) 🎯 MVP

**Goal**: Map loads with zone polygons, vessels moving, performance metrics visible — zero clicks required

**Independent Test**: Open http://localhost:5173. Within 2 seconds: map renders, zones visible as translucent polygons, vessel markers moving, throughput >50k, latency updating.

### Implementation for User Story 1

- [ ] T013 [US1] Create MapView component with MapLibre GL JS initialization in `dashboard/src/components/MapView.jsx`
- [ ] T014 [US1] Add zone polygon rendering layer to MapView from zones.js GeoJSON data
- [ ] T015 [US1] Add vessel marker layer to MapView — circle/triangle markers oriented by heading, updated via GeoJSON source setData() per rAF tick
- [ ] T016 [US1] Integrate MapView into App.jsx layout with usePositions hook

**Checkpoint**: Map shows zones and moving vessels from mock data. No alert panel or metrics yet, but the core visual is live.

---

## Phase 4: User Story 3 — Throughput + Latency Performance Panel (Priority: P1)

**Goal**: Always-visible performance panel showing throughput and p50/p99 latency, updating every 1-2s

**Independent Test**: Dashboard loads, MetricsPanel shows throughput >50,000 and latency values updating every 1-2s. No user interaction required.

### Implementation for User Story 3

- [ ] T017 [US3] Create MetricsPanel component with throughput counter and Chart.js latency chart in `dashboard/src/components/MetricsPanel.jsx`
- [ ] T018 [US3] Integrate MetricsPanel into App.jsx layout with useMetrics hook — always visible on screen

**Checkpoint**: Performance metrics visible from first load. Throughput climbs past 50k.

---

## Phase 5: User Story 2 — Alert Panel with Reasoning Trace (Priority: P1)

**Goal**: Real-time alert list, color-coded, expandable with human-readable reasoning trace and dotted projected path

**Independent Test**: Alerts stream in color-coded. Click any alert — it expands with readable reasoning text and dotted projected path.

### Implementation for User Story 2

- [ ] T019 [US2] Create AlertPanel component — scrollable list, newest first, color-coded by type in `dashboard/src/components/AlertPanel.jsx`
- [ ] T020 [US2] Create AlertDetail component — expanded view with reasoning trace, track history, dotted path, corroboration status in `dashboard/src/components/AlertDetail.jsx`
- [ ] T021 [US2] Integrate human-readable reasoning trace translation (from reasoningTrace.js) into AlertDetail
- [ ] T022 [US2] Add dotted projected path rendering to MapView when an alert is expanded (silent period = dotted line, never solid)
- [ ] T023 [US2] Show both conflicting positions on map for identity_conflict alerts
- [ ] T024 [US2] Create Toast component for alerts_dropped notifications in `dashboard/src/components/Toast.jsx`
- [ ] T025 [US2] Integrate AlertPanel + AlertDetail + Toast into App.jsx with useAlerts hook

**Checkpoint**: Full alert workflow — streaming, color-coded, expandable with reasoning trace, dotted paths on map.

---

## Phase 6: User Story 6 — Boundary-Jitter Demo Control (Priority: P2)

**Goal**: Debug slider/marker lets team nudge a test vessel across zone edges live on stage

**Independent Test**: Use control to jitter a test vessel across a zone edge 10 times. Exactly one alert per genuine crossing, zero duplicates.

### Implementation for User Story 6

- [ ] T026 [US6] Create JitterControl component — slider or draggable marker that sends synthetic position updates in `dashboard/src/components/JitterControl.jsx`
- [ ] T027 [US6] Integrate JitterControl into App.jsx — small debug control in bottom-right corner

**Checkpoint**: Boundary jitter demo works live — judges can drag a vessel across a zone edge.

---

## Phase 7: User Story 4 — Lead Package Export (Priority: P2)

**Goal**: One-click JSON export from any expanded alert, labeled "investigative lead — not legal evidence"

**Independent Test**: Expand alert, click export button, JSON file downloads with disclaimer header.

### Implementation for User Story 4

- [ ] T028 [US4] Add export button to AlertDetail component, wired to exportLead.js utility
- [ ] T029 [US4] Verify exported JSON includes full alert object + `"disclaimer": "investigative lead — not legal evidence"` header

**Checkpoint**: Export works from any expanded alert.

---

## Phase 8: User Story 5 — Re-Entry Cone Projection (Priority: P3)

**Goal**: Translucent projected cone on map for dark-transit alerts

**Independent Test**: Open a dark-transit alert. Cone renders from last known position in direction of last heading.

### Implementation for User Story 5

- [ ] T030 [US5] Add re-entry cone rendering to MapView for selected suspected_dark_transit alerts — translucent polygon from last position/heading/speed

**Checkpoint**: Cone visible for dark-transit alerts. Nice-to-have complete.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Visual polish, edge cases, demo readiness

- [ ] T031 [P] Add WebSocket reconnecting indicator to App.jsx
- [ ] T032 [P] Add "Connecting to server..." loading state before mock server connects
- [ ] T033 Verify UI stays at ≥30 FPS with 50k msgs/sec (profile and fix if needed)
- [ ] T034 Run full quickstart.md validation — standalone demo against mock data

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 Map (Phase 3)**: Depends on Phase 2 — core visual MVP
- **US3 Metrics (Phase 4)**: Depends on Phase 2 — can parallel with US1
- **US2 Alerts (Phase 5)**: Depends on Phase 2 — can parallel with US1/US3
- **US6 Jitter (Phase 6)**: Depends on Phase 2 + mock server from Phase 2
- **US4 Export (Phase 7)**: Depends on US2 (needs AlertDetail)
- **US5 Cone (Phase 8)**: Depends on US1 (needs MapView) + US2 (needs alert selection)
- **Polish (Phase 9)**: Depends on all desired stories complete

### User Story Dependencies

- **US1 (Map)**: After Phase 2 only — fully independent
- **US3 (Metrics)**: After Phase 2 only — fully independent
- **US2 (Alerts)**: After Phase 2 only — fully independent
- **US6 (Jitter)**: After Phase 2 — independent but needs mock server
- **US4 (Export)**: After US2 — needs AlertDetail component
- **US5 (Cone)**: After US1 + US2 — needs MapView + alert selection state

### Parallel Opportunities

- T003, T004, T005, T006 can all run in parallel (Phase 1)
- T008, T009, T010 can all run in parallel (Phase 2, after T007)
- US1 (Phase 3), US3 (Phase 4), US2 (Phase 5) can all start in parallel after Phase 2
- T031, T032 can run in parallel (Phase 9)

---

## Parallel Example: After Foundational Phase

```bash
# All three P1 stories can launch simultaneously:
Task: "T013 [US1] Create MapView component"
Task: "T017 [US3] Create MetricsPanel component"
Task: "T019 [US2] Create AlertPanel component"
```

---

## Implementation Strategy

### MVP First (US1 + US3 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: US1 Map + Phase 4: US3 Metrics
4. **STOP and VALIDATE**: Map shows vessels + zones, metrics visible, mock data flowing
5. Demo-ready with just the opening shot of the pitch

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US1 Map + US3 Metrics → Opening demo shot works (MVP!)
3. US2 Alerts → Full alert workflow with reasoning traces
4. US6 Jitter → Live boundary-jitter demo moment
5. US4 Export → Lead package export
6. US5 Cone → Re-entry cone visual polish
7. Polish → Demo-ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- Each story is independently testable after Phase 2
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
- **Ponytail**: No test tasks generated — manual demo verification per spec. No unrequested abstractions. Shortest working diff.
