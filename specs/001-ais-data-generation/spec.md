# Feature Specification: AIS Data Generation, Corroboration Jobs, Benchmark & Demo

**Feature Branch**: `001-ais-data-generation`

**Created**: 2026-07-11

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Synthetic AIS Generator with Scripted Events (Priority: P1)

A judge or operator launches Varuna's demo and sees N vessels moving realistically on the map, with throughput already past 50,000 messages/sec. They can trigger scripted events (dark-transit, loitering, duplicate-MMSI) on demand via HTTP call or keypress — not on a fixed timer — and the events are independently repeatable without restarting the system.

**Why this priority**: The generator is the foundation every other component depends on. Without it, the backend engines have nothing to process, the benchmark has nothing to measure, and the demo has nothing to show.

**Independent Test**: Run the generator standalone (no frontend, no engine); confirm it emits correctly-shaped AISMessage structs on a Go channel at ≥ 50,000 msg/sec sustained for 60 seconds. Trigger each scripted event via HTTP and confirm the correct vessel state changes appear in the output stream.

**Acceptance Scenarios**:

1. **Given** the generator is started with N=200 vessels, **When** it runs for 60 seconds, **Then** throughput on the output channel is ≥ 50,000 msg/sec sustained with zero message drops.
2. **Given** the generator is running, **When** `/trigger/dark-transit` is called, **Then** the named vessel stops emitting messages for the configured silence window, then reappears either plausibly or with a kinematic jump (configurable).
3. **Given** the generator is running, **When** `/trigger/loitering` is called, **Then** the named vessel holds speed < 3 knots within a tight radius near a sensitive zone for the configured window.
4. **Given** the generator is running, **When** `/trigger/duplicate-mmsi` is called, **Then** two separate vessel streams begin broadcasting the same MMSI from physically incompatible positions.
5. **Given** the generator is running, **When** `/trigger/geofence-crossing` is called, **Then** the named vessel crosses a zone boundary cleanly.
6. **Given** any event has been triggered, **When** the same trigger endpoint is called again, **Then** the event replays correctly without requiring a restart.

---

### User Story 2 - Hardcoded Zone GeoJSON (Priority: P2)

The system has 8 pre-defined zone polygons as GeoJSON that are shared between the backend engine (teammate 1) and the frontend map (teammate 2), so both agree on identical boundaries without any zone-authoring UI.

**Why this priority**: Zones must exist before any geofence or absence logic can be tested. They are a shared contract between all three teammates.

**Independent Test**: Load the zones.geojson file; confirm 8 valid GeoJSON Polygon features are present, each with a `name`, `zone_type` (coastal/offshore/open_ocean), and `silence_tolerance_s` property. Visually verify they render on a MapLibre map.

**Acceptance Scenarios**:

1. **Given** zones.geojson exists, **When** parsed as GeoJSON, **Then** it contains exactly 8 Polygon features with required properties.
2. **Given** zones.geojson is loaded by the engine, **When** the grid-hash index is built, **Then** all 8 zones are indexed with no errors.

---

### User Story 3 - SAR Corroboration Job (Priority: P3)

When a `suspected_dark_transit` alert fires and the vessel's last-known position falls under a pre-downloaded Sentinel-1 GRD tile, an offline background goroutine runs a SAR ship-detector model and upgrades the alert's corroboration status to `corroborated` with the detection evidence attached.

**Why this priority**: This is the key differentiator for the corroboration demo. It must run completely isolated from the hot ingestion path.

**Independent Test**: Given a known alert ID and a pre-downloaded Sentinel-1 tile, run the SAR job in isolation; confirm it calls `engine.Corroborate(alertID, "sar", evidence)` and returns a detection result.

**Acceptance Scenarios**:

1. **Given** a `suspected_dark_transit` alert exists with a position covered by the pre-downloaded tile, **When** the SAR goroutine ticks, **Then** it runs the ship detector and calls `engine.Corroborate(alertID, "sar", evidence)`.
2. **Given** the SAR job runs, **When** no tile covers the alert position, **Then** no corroboration call is made and no error is raised.
3. **Given** the SAR job runs, **When** it finds a ship detection, **Then** the evidence includes `tile_id`, `detection_confidence`, `bounding_box_pixels`, and `model_version`.

---

### User Story 4 - VIIRS Night-Lights Stub (Priority: P4)

A stubbed second corroboration modality (VIIRS) uses the exact same call signature as the SAR job (`engine.Corroborate(alertID, "viirs", evidence)`), demonstrating that the alert object accepts multiple independent corroboration sources without any code change to the core engines. It is labeled explicitly as a stub in the code and in the demo.

**Why this priority**: Proves the fusion-ready architecture without fabricating a capability. Lower priority because the SAR job already demonstrates the pattern.

**Acceptance Scenarios**:

1. **Given** a matching alert exists, **When** the VIIRS stub goroutine ticks, **Then** it calls `engine.Corroborate(alertID, "viirs", evidence)` with a mocked detection record.
2. **Given** the stub is running, **When** inspecting its output, **Then** the evidence clearly contains a `stub: true` marker.

---

### User Story 5 - Benchmark Harness (Priority: P5)

A benchmark script drives the generator at sustained ≥ 50,000 msg/sec for 120 seconds and logs hardware spec, message size, zone count, WebSocket client count, throughput, p50/p95/p99 end-to-end latency, and alert correctness (scripted crossings vs alerts fired, must be 1:1).

**Why this priority**: These are the exact numbers quoted in the demo. They must be real, reproducible, and captured early — not assumed at the last hour.

**Acceptance Scenarios**:

1. **Given** the harness runs, **When** complete, **Then** a benchmark-results.json file is written containing all required metrics.
2. **Given** 4 scripted geofence crossings are triggered during the benchmark, **When** the run ends, **Then** `alerts_fired == 4` and `false_positives == 0`.
3. **Given** the benchmark runs on the target machine, **When** p99 latency is measured, **Then** it is < 50ms end-to-end.

---

### Edge Cases

- What happens when a vessel's AIS cadence is indistinguishable from a real coverage gap (open ocean)?
- How does the generator handle a duplicate-MMSI trigger while a dark-transit is already active for that vessel?
- What happens if the SAR tile file is missing or corrupted at startup?
- How does the benchmark harness handle a slow WebSocket client that falls behind?

## Requirements

### Functional Requirements

- **FR-001**: Generator MUST emit `AISMessage` structs (`vessel_id`, `mmsi`, `lat`, `lon`, `heading`, `speed_knots`, `timestamp_ms`) onto a Go channel consumed by `engine.Ingest(in <-chan AISMessage)`.
- **FR-002**: Generator MUST sustain ≥ 50,000 messages/sec on the output channel for at least 120 seconds without dropping messages.
- **FR-003**: Generator MUST support great-circle movement with realistic heading/speed/cadence per zone tier (coastal 2–10s, offshore 10–30min, open ocean variable).
- **FR-004**: All four scripted events MUST be triggerable on demand via HTTP endpoint (not fixed timer) and repeatable without restart.
- **FR-005**: Scripted dark-transit MUST support both plausible-reappearance and kinematic-jump variants, selectable per trigger call.
- **FR-006**: System MUST provide a `zones.geojson` file with exactly 8 polygon zones, each with `name`, `zone_type`, and `silence_tolerance_s` properties.
- **FR-007**: SAR corroboration job MUST run in an isolated goroutine on a slow ticker, never touching the ingestion hot path.
- **FR-008**: SAR job MUST use a SAR-specific ship detector (SSDD/HRSID-fine-tuned YOLO or equivalent via onnxruntime) — NOT a COCO-pretrained checkpoint.
- **FR-009**: VIIRS stub MUST use the same `engine.Corroborate` call signature as the SAR job and MUST include a `stub: true` evidence marker.
- **FR-010**: Benchmark harness MUST record hardware spec, message size, zone count, WS client count, sustained throughput, p50/p95/p99 latency, and 1:1 alert correctness.
- **FR-011**: Benchmark output MUST be written to `benchmark-results.json` in the repo root.

### Key Entities

- **AISMessage**: Core data unit — `vessel_id string`, `mmsi string`, `lat/lon float64`, `heading float64`, `speed_knots float64`, `timestamp_ms int64`.
- **VesselState**: Per-vessel simulation state — position, heading, speed, cadence, active scripted event, MMSI.
- **ScriptedEvent**: Enum — `dark_transit`, `loitering`, `duplicate_mmsi`, `geofence_crossing`.
- **Zone**: GeoJSON Polygon with `name`, `zone_type`, `silence_tolerance_s`, `boundary_buffer_km`.
- **SARTile**: Pre-downloaded Sentinel-1 GRD tile metadata — `tile_id`, `file_path`, `bbox` (lat/lon bounds).
- **CorroborationEvidence**: `source string`, `tile_id string`, `detection_confidence float64`, `bounding_box_pixels []int`, `model_version string`, `stub bool`.
- **BenchmarkResult**: All metrics captured by the benchmark harness.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Generator sustains ≥ 50,000 msg/sec for 120 seconds on a 4-core 2.4GHz laptop (8GB RAM) — standalone, independent of engine or frontend readiness.
- **SC-002**: All four scripted events are individually triggerable on demand and repeatable without restart; end-to-end demo sequence runs twice in a row cleanly.
- **SC-003**: SAR corroboration job correctly upgrades a matching alert with detection evidence within one ticker interval of the alert being raised.
- **SC-004**: Benchmark harness produces a `benchmark-results.json` with p99 end-to-end latency < 50ms and 1:1 alert correctness for all scripted crossings.
- **SC-005**: zones.geojson passes JSON schema validation with 8 valid polygon zones sharing the same format used by both the engine and the frontend map.

## Assumptions

- Teammate 1's engine exposes `engine.Ingest(in <-chan AISMessage)` and `engine.Corroborate(alertID string, source string, evidence map[string]interface{})` — exact signatures agreed at hour 1.
- A pre-downloaded Sentinel-1 GRD tile covering the scripted vessel's last-dark position is available in the repo under `data/sar/`.
- The SAR ship detector model (ONNX format) is available from HuggingFace or bundled in the repo under `data/models/`.
- No live public AIS feed is used — all vessel positions are synthetic.
- No Docker orchestration is needed; everything runs as goroutines in a single Go binary.
- The benchmark is run on the team's demo laptop; hardware spec is captured at runtime.
- Mobile support and zone-authoring UI are explicitly out of scope.
