# Prompt for Teammate 3 — Data Generation, Corroboration Jobs, Benchmark & Demo

Paste this whole file into your AI coding assistant at the start of the build. The full PRD (`Varuna_PRD_v4_FINAL.md`) is your source of truth for anything not covered here.

---

## Your role
You own three things that don't depend on teammate 1 or 2 finishing first, so you can work fully in parallel from hour zero: the **synthetic AIS generator with scripted events**, the **corroboration jobs** (SAR real, VIIRS stubbed), and the **benchmark harness + demo/Q&A rehearsal**. You are also the person who owns the actual live demo delivery and the team's ability to survive adversarial questions — treat the Q&A table in the PRD (§7) as something to internalize, not just read.

## Shared contract — what you must produce for teammate 1

**Outbound AIS message, exactly matching teammate 1's ingestion type:**
```go
type AISMessage struct {
    VesselID    string  `json:"vessel_id"`
    MMSI        string  `json:"mmsi"`
    Lat         float64 `json:"lat"`
    Lon         float64 `json:"lon"`
    HeadingDeg  float64 `json:"heading"`
    SpeedKnots  float64 `json:"speed_knots"`
    TimestampMs int64   `json:"timestamp_ms"`
}
```
Your generator writes these onto a Go channel that teammate 1's engine reads from (`engine.Ingest(in <-chan AISMessage)`). Agree the exact package/function name with teammate 1 in the first hour so neither of you blocks the other — suggested: `package generator`, `func NewGenerator(cfg Config) *Generator`, `func (g *Generator) Run(out chan<- AISMessage)`.

**Corroboration call into teammate 1's engine, once an alert is ready to upgrade:**
```go
engine.Corroborate(alertID string, source string, evidence map[string]interface{})
```

## What to build, in order
1. **Baseline generator**: N vessels (configurable count, aim for enough to comfortably clear 50k msgs/sec with the batching), great-circle movement, realistic heading/speed/cadence per zone type (coastal 2–10s, offshore 10–30min, open ocean hours — match teammate 1's zone tiers).
2. **Scripted dark-transit event**: one or more vessels that, on a judge-controllable trigger (a keypress or HTTP call, not a fixed timer — you want to control exactly when this fires live), go silent near a zone boundary for a configurable window, then reappear either plausibly or with a kinematic jump. Build both variants — you'll want the plausible one for the main demo and the jump variant as an optional "what if it's not plausible" follow-up if a judge asks.
3. **Scripted loitering event**: a vessel that holds low speed in a tight radius near a sensitive zone.
4. **Scripted duplicate-MMSI event**: two synthetic position streams broadcasting the same MMSI from physically incompatible locations, triggerable independently of the dark-transit event. Keep this cheap and deterministic — it's the newest, highest-credibility piece of the whole build, don't let it be an afterthought.
5. **Hardcoded zone GeoJSON**: 8 zones is the target from the benchmark methodology (§5) — write these once, share the file with teammate 2 so the map and the engine agree on the same boundaries.
6. **SAR corroboration job (real)**: pre-download 1–2 Sentinel-1 GRD tiles from Copernicus Open Access Hub covering your scripted vessel's last-dark position. Run a SAR-specific ship detector (SSDD/HRSID-fine-tuned YOLO, or a pretrained SAR detector from HuggingFace via onnxruntime) — do not use a COCO-pretrained checkpoint, it has never seen a ship in SAR imagery and will detect nothing. Runs on a slow ticker goroutine, isolated from the hot path, calls `engine.Corroborate(...)` when it finds a match.
7. **VIIRS stub**: a mocked corroboration hit using the same call signature as the SAR job. Be explicit internally and in the demo that this is a stub proving the interface, not a working nighttime-detection model — do not let this get oversold in the pitch.
8. **Benchmark harness**: script that drives the generator at sustained 50k msgs/sec for 120 seconds and logs hardware spec, message size, zone count, WebSocket client count, throughput, p50/p95/p99 latency, and alert correctness (scripted crossings vs. alerts fired, must be 1:1). This produces the exact numbers quoted in PRD §5 — get real numbers early, don't wait until the last hour to discover the real p99 isn't 38ms.

## Demo & Q&A ownership (do this in parallel with the build, not after)
- Time the 2:45 demo script (PRD §6) out loud against the actual running system, not a hypothetical one, as soon as items 1–4 above are functional even with placeholder frontend.
- Read the Q&A table (PRD §7) with the team and make sure all three of you could each answer any question in it cold, not just the person who wrote the PRD.
- Prepare the live boundary-jitter test (dragging or scripting a vessel back and forth across a zone edge) as a rehearsed, reliable action — this is explicitly something a judge may ask to see, and it needs to work first try.

## Explicitly do not build
No live public AIS feed on stage. No Docker orchestration. No zone-authoring UI. If you're short on time, SAR/VIIRS enrichment (items 6–7) are the first things to cut — items 1–5 and 8 are what the demo cannot survive without.

## Definition of done for your piece
- Generator sustains 50k msgs/sec on its own, independent of whether the engine or frontend are ready, so you can hand teammate 1 real load early.
- All four scripted events (dark-transit, loitering, duplicate-MMSI, plus the plain geofence crossing) are individually triggerable on demand, not on a fixed timer, and repeatable — you should be able to run the whole demo sequence twice in a row without restarting anything.
- You can deliver the full demo script from memory, and you and your teammates can each field at least half the Q&A table without looking at the document.

If you're blocked on backend or frontend integration details, check `Varuna_PRD_v4_FINAL.md` §2 (architecture diagram) for how the pieces connect, or ask teammates 1 and 2 directly.
