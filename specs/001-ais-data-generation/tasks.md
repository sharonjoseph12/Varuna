# Tasks: AIS Data Generation, Corroboration Jobs, Benchmark & Demo

**Input**: Design documents from `specs/001-ais-data-generation/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story label (US1–US5)

---

## Phase 1: Setup

**Purpose**: Go module, directory structure, .gitignore

- [X] T001 Initialize Go module `varuna` at repo root: `go.mod` with `module varuna`
- [X] T002 Create directory skeleton: `generator/`, `zones/`, `corroboration/`, `benchmark/`, `cmd/varuna/`, `data/sar/`, `data/models/`
- [X] T003 [P] Add `.gitignore` entries for `data/sar/*.tiff`, `data/models/*.onnx`, `benchmark-results.json`, `*.exe`, `*.out`

---

## Phase 2: Foundational — Shared Types & Zone File

**Purpose**: AISMessage struct and zones.geojson must exist before any other package compiles.

- [X] T004 Create `generator/types.go` — define `AISMessage`, `VesselState`, `ScriptedEvent`, `Config` structs
- [X] T005 [P] Create `corroboration/types.go` — define `SARTile`, `CorroborationEvidence`, `AlertRef`, `CorroborateFunc`
- [X] T006 [P] Create `benchmark/results.go` — define `BenchmarkResult`, `BenchmarkConfig` structs and `WriteResults`
- [X] T007 Create `zones/zones.geojson` — 8 hardcoded GeoJSON Polygon features with `name`, `zone_type`, `silence_tolerance_s`, `boundary_buffer_km`

**Checkpoint**: ✅ Types and zone file created.

---

## Phase 3: User Story 1 — Synthetic AIS Generator with Scripted Events 🎯 MVP

**Goal**: Standalone generator sustaining ≥ 50k msg/sec with 4 HTTP-triggered scripted events.

**Independent Test**: `go test ./generator/... -run TestGeneratorThroughput`

- [X] T008 [US1] Create `generator/vessel.go` — `newVessel`, `advance` (Haversine great-circle), `cadenceForZone`
- [X] T009 [US1] Create `generator/events.go` — `applyDarkTransit`, `applyLoitering`, `applyDuplicateMMSI`, `applyGeofenceCrossing`, `resetEvent`, `tickEvent`, `bearingTo`
- [X] T010 [US1] Create `generator/trigger_server.go` — `net/http` server, POST `/trigger/{event}`, GET `/status`
- [X] T011 [US1] Create `generator/generator.go` — `NewGenerator`, `Run` (one goroutine per vessel), `ApplyEvent`, `AllVesselIDs`
- [X] T012 [US1] Create `generator/config.go` — `DefaultConfig()` (200 vessels, 10000 buffer, port 8081)
- [X] T013 [US1] Create `generator/generator_test.go` — `TestGeneratorThroughput`, `TestDarkTransitEvent`, `TestDuplicateMMSI`

**Checkpoint**: ✅ All generator files written.

---

## Phase 4: User Story 2 — Zone GeoJSON

**Goal**: zones.geojson shared file, loadable by engine and frontend.

**Independent Test**: `go test ./zones/... -run TestZonesLoad`

- [X] T014 [P] [US2] Create `zones/loader.go` — `LoadZones`, `ZoneLookup`, `polygonCentroid`
- [X] T015 [P] [US2] Create `zones/zones_test.go` — `TestZonesLoad`, `TestZoneProperties`

**Checkpoint**: ✅ Zone loader and tests written.

---

## Phase 5: User Story 3 — SAR Corroboration Job

**Goal**: Isolated SAR goroutine calling engine.Corroborate when tile covers alert position.

**Independent Test**: `go test ./corroboration/... -run TestSAR`

- [X] T016 [US3] Create `corroboration/sar_job.go` — `RunSARJob`, `defaultSARInfer`, `sarInferFn` injectable var
- [X] T017 [US3] Create `data/models/sar_infer.py` — ONNX inference via Python subprocess, JSON stdout
- [X] T018 [US3] Create `corroboration/sar_job_test.go` — `TestSARJobHit`, `TestSARJobMiss`

**Checkpoint**: ✅ SAR job and tests written.

---

## Phase 6: User Story 4 — VIIRS Night-Lights Stub

**Goal**: Stub second corroboration modality, same interface as SAR, explicit stub label.

**Independent Test**: `go test ./corroboration/... -run TestVIIRS`

- [X] T019 [P] [US4] Create `corroboration/viirs_stub.go` — `RunVIIRSStub`, stub evidence with `stub:true`
- [X] T020 [P] [US4] Create `corroboration/viirs_stub_test.go` — `TestVIIRSStub`

**Checkpoint**: ✅ VIIRS stub and test written.

---

## Phase 7: User Story 5 — Benchmark Harness

**Goal**: Reproducible benchmark producing benchmark-results.json with real p99 < 50ms.

- [X] T021 [US5] Create `benchmark/harness.go` — `RunBenchmark`, latency sampling, percentile computation
- [X] T022 [US5] Update `benchmark/results.go` — `BenchmarkConfig`, `WriteResults`
- [X] T023 [US5] Create `cmd/benchmark/main.go` — CLI flags, calls `RunBenchmark`, prints summary

**Checkpoint**: ✅ Benchmark harness written.

---

## Phase 8: Wire Everything — cmd/varuna/main.go

- [X] T024 Create `cmd/varuna/main.go` — wires generator + engine + SAR job + VIIRS stub + trigger server; graceful SIGINT shutdown
- [X] T025 Create `cmd/varuna/interfaces.go` — `EngineInterface`, `stubEngine` (no-op for solo testing)

---

## Phase 9: Polish & Cross-Cutting

- [X] T026 [P] Add `README.md` — run instructions, curl trigger examples, benchmark, SAR model setup
- [X] T027 [P] Create `data/models/requirements.txt` — `onnxruntime`, `numpy`, `Pillow`
- [X] T028 [P] Create `data/sar/README.md` — Sentinel-1 tile download instructions + demo framing note
- [ ] T029 Run `go vet ./...` and fix all warnings — **requires Go installation on demo machine**
- [ ] T030 [P] Verify `go build ./cmd/varuna` and `go build ./cmd/benchmark` — **requires Go installation**

---

## Dependencies & Execution Order

- **Phase 1** (Setup): No deps — start immediately
- **Phase 2** (Foundational): Depends on Phase 1 — BLOCKS all phases
- **Phase 3** (US1 Generator): Depends on Phase 2 — highest priority ✅
- **Phase 4** (US2 Zones): Depends on Phase 2 ✅
- **Phase 5** (US3 SAR): Depends on Phase 2 ✅
- **Phase 6** (US4 VIIRS): Depends on Phase 2 ✅
- **Phase 7** (US5 Benchmark): Depends on Phase 3 ✅
- **Phase 8** (Wire): Depends on Phases 3–6 ✅
- **Phase 9** (Polish): T029/T030 require Go on demo machine

## MVP Status

Phases 1–8 complete. Run `go build ./...` on the demo machine to verify.
