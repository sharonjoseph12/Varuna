# Varuna — Real-Time Maritime Surveillance Engine

**Problem Statement 5 · National Hackathon Final**

One Go binary. No broker. 50,000+ vessel messages/sec. Millisecond alert latency.

---

## Quick Start

### Prerequisites

- Go 1.22+
- Python 3.11+ (for SAR inference only)
- `pip install -r data/models/requirements.txt`
- A pre-downloaded Sentinel-1 GRD tile in `data/sar/` (see `data/sar/README.md`)
- SAR ONNX model at `data/models/yolov8n_sar_ssdd.onnx` (see below)

### Run the generator + engine (standalone demo)

```bash
# Run with stub engine (no teammate 1 dependency)
go run ./cmd/varuna -stub -vessels 200

# Run with real engine (teammate 1's package wired in)
go run ./cmd/varuna -vessels 200
```

### Run the benchmark

```bash
go run ./cmd/benchmark -duration 120 -vessels 200 -clients 5
# Writes benchmark-results.json at repo root
```

### Run tests

```bash
go test ./generator/... -v -run Test
go test ./zones/...
go test ./corroboration/...
```

---

## Triggering Demo Events

All events are HTTP POST, repeatable without restart.

```bash
# Dark transit — vessel goes silent near a zone boundary
curl -X POST "http://localhost:8081/trigger/dark-transit?vessel=vessel-0000&silence_s=45&variant=plausible"

# Dark transit with kinematic jump (impossible reappearance)
curl -X POST "http://localhost:8081/trigger/dark-transit?vessel=vessel-0000&silence_s=45&variant=jump"

# Duplicate MMSI — vessel broadcasts another vessel's identity
curl -X POST "http://localhost:8081/trigger/duplicate-mmsi?vessel=vessel-0001&target_mmsi=200000000"

# Loitering — low speed in tight radius near sensitive zone
curl -X POST "http://localhost:8081/trigger/loitering?vessel=vessel-0002&duration_s=120"

# Geofence crossing — vessel steers directly toward zone boundary
curl -X POST "http://localhost:8081/trigger/geofence-crossing?vessel=vessel-0003&zone=Strait+of+Malacca"

# Reset any active event
curl -X POST "http://localhost:8081/trigger/reset?vessel=vessel-0003"

# Check active events
curl http://localhost:8081/status
```

**Available zone names** (case-sensitive):
- `Strait of Malacca`, `Persian Gulf`, `Bay of Bengal EEZ`, `South China Sea`
- `Gulf of Guinea`, `Mediterranean EEZ West`, `North Sea`, `Barents Sea Approach`

---

## SAR Model Setup

```bash
# Option A — download pre-trained SAR detector from HuggingFace
# Search: https://huggingface.co/models?search=sar+ship+detection+yolov8
# Place ONNX file at: data/models/yolov8n_sar_ssdd.onnx

# Option B — fine-tune yourself (30 min on GPU)
pip install ultralytics
# Download SSDD dataset: https://github.com/CAESAR-Radi/SAR-Ship-Dataset
yolo train model=yolov8n.pt data=ssdd.yaml epochs=50 imgsz=640
yolo export model=runs/detect/train/weights/best.pt format=onnx
cp runs/detect/train/weights/best.onnx data/models/yolov8n_sar_ssdd.onnx
```

**Do NOT use a COCO-pretrained checkpoint without SAR fine-tuning.**
COCO has never seen SAR imagery; it will detect nothing.

---

## Architecture Summary

```
generator/ ──chan AISMessage──▶ engine.Ingest()
                                    │
                              alerts channel
                                    │
              ┌─────────────────────┴──────────────────────┐
              ▼                                             ▼
    corroboration/sar_job.go                   corroboration/viirs_stub.go
    (30s ticker, isolated goroutine)           (60s ticker, stub, isolated)
              │                                             │
              └──────────────── engine.Corroborate() ──────┘

generator/trigger_server.go  ← HTTP :8081  (judge-controllable events)
zones/zones.geojson           ← shared between engine + frontend map
benchmark/harness.go          ← standalone benchmark driver
```

No Kafka. No Docker. No external broker. Single binary.

---

## Project Structure

```
generator/          Synthetic AIS vessel generator + HTTP trigger server
zones/              8 hardcoded zone polygons (GeoJSON) — shared with engine + frontend
corroboration/      SAR job (real) + VIIRS stub (architecture demo)
benchmark/          Benchmark harness + BenchmarkResult JSON writer
cmd/varuna/         Main binary entry point
cmd/benchmark/      Benchmark runner entry point
data/sar/           Pre-downloaded Sentinel-1 GRD tile (not in git)
data/models/        SAR ONNX model + sar_infer.py Python script
specs/              speckit spec, plan, tasks, research docs
```

---

## Benchmark Methodology (PRD §5)

Run on a 4-core 2.4GHz laptop / 8GB RAM / 5 WS clients / 8 zones / 64-byte messages.
Sustained 120 seconds. Not a burst measurement.

Expected:
- **Throughput**: ≥ 50,000 msg/sec sustained
- **p99 latency**: < 50ms end-to-end (ingestion → WebSocket delivery)
- **Alert correctness**: 1:1 (4 scripted crossings → 4 alerts, 0 false positives)

---

## Member 3 Scope Checklist

- [x] Synthetic AIS generator (N vessels, great-circle movement, zone-aware cadence)
- [x] 4 judge-triggerable scripted events (dark-transit ×2 variants, loitering, duplicate-MMSI, geofence-crossing)
- [x] HTTP trigger server (repeatable without restart)
- [x] 8-zone GeoJSON (shared with engine + frontend)
- [x] SAR corroboration job (isolated goroutine, Python subprocess, ONNX model)
- [x] VIIRS stub (same interface, explicit stub label)
- [x] Benchmark harness (p50/p95/p99, alert correctness, benchmark-results.json)
- [x] All interfaces match teammate 1's engine contract exactly
