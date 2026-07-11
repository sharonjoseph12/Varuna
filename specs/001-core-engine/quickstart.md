# Quickstart: Core Processing Engine

## Prerequisites

- Go 1.21+
- No external services needed

## Setup

```bash
cd Varuna
go mod tidy
```

## Run Standalone

```bash
go run ./cmd/varuna/
```

Starts the engine with a built-in mini-generator (8 vessels, scripted events). Serves on `:8080`.

## Validate

### 1. Check alerts stream

```bash
# In another terminal — connect to WebSocket
# Or use a browser WebSocket client at ws://localhost:8080/ws/alerts
curl http://localhost:8080/api/metrics
```

Expected: JSON with `throughput_msgs_sec`, `latency_p50_ms`, etc.

### 2. Trigger scripted events

```bash
# Dark transit near boundary
curl -X POST http://localhost:8080/api/trigger/dark_transit

# Boundary jitter (should produce zero alerts)
curl -X POST http://localhost:8080/api/trigger/boundary_jitter

# MMSI identity conflict
curl -X POST http://localhost:8080/api/trigger/mmsi_conflict

# Loitering
curl -X POST http://localhost:8080/api/trigger/loiter
```

### 3. Run tests

```bash
go test ./engine/... -v -count=1
```

Expected: All pass. Key assertions:
- One alert per scripted crossing
- Zero alerts from boundary jitter within hysteresis margin
- `identity_conflict` fires only on impossible position pairs
- Every alert has non-empty reasoning trace

### 4. Run throughput benchmark

```bash
go test ./engine/... -bench=BenchmarkThroughput -benchtime=120s
```

Expected: ≥ 50,000 msgs/sec sustained. p99 < 50ms.

## Integration Points

- **Teammate 2**: Connect WebSocket to `ws://localhost:8080/ws/alerts` and `ws://localhost:8080/ws/positions`
- **Teammate 3**: Import `engine` package, feed AIS messages to `e.Ingest(channel)`, call `e.Corroborate()` from SAR/VIIRS jobs
