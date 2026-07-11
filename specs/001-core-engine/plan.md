# Implementation Plan: Core Processing Engine

**Branch**: `001-core-engine` | **Date**: 2026-07-11 | **Spec**: [spec.md](file:///c:/Users/Sharon/OneDrive/Desktop/Varuna/specs/001-core-engine/spec.md)

**Input**: Feature specification from `specs/001-core-engine/spec.md`

## Summary

Build the Varuna real-time maritime surveillance processing core in Go. Single binary, single `engine` package. Batched-tick ingestion at 50k+ msgs/sec, grid-hash O(1) spatial index, two-tier geofence with hysteresis, absence detection with zone-aware thresholds, identity-conflict detection, loitering detection, reasoning traces on every alert, and async embedded persistence via bbolt.

## Technical Context

**Language/Version**: Go 1.21+

**Primary Dependencies**: `go.etcd.io/bbolt` (embedded KV store). Everything else is stdlib.

**Storage**: bbolt — pure-Go embedded key-value store. Async writes, never on hot path.

**Testing**: `go test` (stdlib). Benchmarks via `testing.B`.

**Target Platform**: Single-machine (Windows/Linux/macOS). 4+ cores, 8GB+ RAM.

**Project Type**: Library (engine package) + CLI entry point

**Performance Goals**: ≥ 50,000 msgs/sec sustained for 120s. p99 < 50ms ingestion→output-channel.

**Constraints**: No external broker, no Docker, no network hops in hot path. Single Go process.

**Scale/Scope**: 8 zones, ~10k concurrent vessels, 5 WebSocket clients.

## Constitution Check

*GATE: Constitution is unfilled template — no constraints to check. Proceeding.*

## Project Structure

### Documentation (this feature)

```text
specs/001-core-engine/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── engine-api.md    # Public API surface
└── tasks.md             # Phase 2 output (via /speckit-tasks)
```

### Source Code (repository root)

```text
engine/
├── types.go          # All shared types (AISMessage, Alert, Config, Zone, etc.)
├── grid.go           # Grid-hash spatial index
├── geofence.go       # Two-tier geofence engine with hysteresis
├── absence.go        # Absence engine with zone-aware thresholds
├── identity.go       # Identity-conflict engine
├── loitering.go      # Loitering detection
├── engine.go         # Core orchestrator: batched-tick ingestion, channels
├── store.go          # Embedded bbolt persistence
├── zones.go          # 8 hardcoded zone definitions
└── engine_test.go    # All tests + throughput benchmark

cmd/varuna/
└── main.go           # Entry point: HTTP server, mini-generator, WS endpoints
```

**Structure Decision**: Single `engine` package — no internal packages, no interfaces with one implementation, no factory patterns. Ponytail: fewest files that make sense.

## Complexity Tracking

No constitution violations. No complexity to justify.
