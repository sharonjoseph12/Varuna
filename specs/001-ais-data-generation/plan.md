# Implementation Plan: AIS Data Generation, Corroboration Jobs, Benchmark & Demo

**Branch**: `001-ais-data-generation` | **Date**: 2026-07-11 | **Spec**: [spec.md](./spec.md)

## Summary

Build the synthetic AIS generator with four judge-triggerable scripted events, a shared zones GeoJSON file, offline SAR and VIIRS corroboration jobs, and a benchmark harness — all as Go packages inside the Varuna monorepo. The generator feeds teammate 1's ingestion engine via a typed Go channel. Corroboration jobs run on isolated goroutines on slow tickers, calling `engine.Corroborate(...)` when a match is found.

## Technical Context

**Language/Version**: Go 1.22+, Python 3.11 (SAR model inference via onnxruntime)

**Primary Dependencies**:
- Go stdlib: `net/http`, `encoding/json`, `sync`, `time`, `math`
- `github.com/urfave/cli/v2` or plain `flag` for config (ponytail: use flag, no CLI framework needed)
- `onnxruntime_go` or Python subprocess for SAR inference
- `ultralytics` / `onnxruntime` Python for SAR ship detector

**Storage**: `zones.geojson` (static file), `data/sar/` (Sentinel-1 tile), `data/models/` (ONNX model), `benchmark-results.json`

**Testing**: Go `testing` package + `go test -bench`; no external test framework

**Target Platform**: Single laptop, Linux/Windows/macOS compatible

**Performance Goals**: ≥ 50,000 AISMessage/sec sustained for 120 seconds on 4-core 2.4GHz / 8GB RAM

**Constraints**: Zero external brokers; single binary; SAR job never on hot path; all events HTTP-triggerable without restart

**Scale/Scope**: ~200 vessels, 8 zones, 4 scripted event types, 1 SAR tile, 1 VIIRS stub

## Constitution Check

No formal constitution is defined for this project (constitution.md is a template). Applying Ponytail defaults:
- ✅ Stdlib first — no new dependencies if stdlib covers it
- ✅ No broker, no Docker, no zone UI
- ✅ Smallest working diff per task
- ✅ SAR job isolated from hot path (goroutine + slow ticker)
- ✅ Explicit stub labeling on VIIRS

## Project Structure

### Documentation (this feature)

```text
specs/001-ais-data-generation/
├── plan.md              ← this file
├── research.md          ← Phase 0 decisions
├── data-model.md        ← entity definitions
├── contracts/           ← Go channel + HTTP API contracts
└── tasks.md             ← task list
```

### Source Code (repository root)

```text
generator/
├── generator.go         ← NewGenerator, Run, vessel loop
├── vessel.go            ← VesselState, great-circle movement
├── events.go            ← scripted event handlers
├── trigger_server.go    ← HTTP trigger endpoints
└── config.go            ← Config struct

zones/
└── zones.geojson        ← 8 hardcoded zone polygons (shared with engine + frontend)

corroboration/
├── sar_job.go           ← SAR goroutine, ONNX inference call
├── viirs_stub.go        ← VIIRS mocked corroboration
└── types.go             ← CorroborationEvidence struct

benchmark/
├── harness.go           ← benchmark driver
└── results.go           ← BenchmarkResult struct + JSON write

data/
├── sar/                 ← pre-downloaded Sentinel-1 GRD tile(s)
└── models/              ← SAR ship detector ONNX model

cmd/
└── varuna/
    └── main.go          ← wires generator + engine + corroboration + benchmark

benchmark-results.json   ← written by benchmark harness at run time
```
