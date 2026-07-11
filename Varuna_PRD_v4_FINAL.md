# Varuna — Real-Time Maritime Surveillance Engine
### Product Requirements Document · Version 4.0 · National Hackathon Final

**Problem Statement 5 — Real-Time Maritime Surveillance**
**Build window:** 24-hour hackathon
**Delivery format:** Web application (browser-native, zero install friction for judges)

---

## QUICK REFERENCE — What judges will evaluate us on

| Constraint | What it actually asks | How Varuna answers it |
|---|---|---|
| No heavy distributed pipelines | No Kafka, Flink, Spark, or broker overhead | Single Go process, in-memory channel buffer, no external broker |
| ≥ 50,000 messages/sec throughput | Sustained, not burst | Grid-hash O(1) spatial index + batched ingestion; demonstrated live with hardware spec |
| Millisecond alert latency | End-to-end, not just geometry check | p99 < 50ms ingestion → WebSocket delivery, measured and displayed live |

**Where Varuna actually differentiates:** every team here will clear 50k/sec — it's a baseline, not a headline. The real gap in this space isn't throughput. It's that every production dark-fleet system (Global Fishing Watch/Skylight, Windward, Starboard) runs on an hours-to-days latency budget by design, because their sensors — nighttime-light satellites, SAR — are inherently slow. Nobody occupies the real-time reflex layer sitting directly on the live AIS feed. That's what Varuna is. We are the fast triage layer that fires in milliseconds and hands off to slow, high-confidence corroboration sources when they arrive — not a competitor to those platforms, the missing piece underneath them.

---

## 1. Problem & Positioning

### 1.1 What the problem statement asks
Maritime authorities monitor millions of vessel location updates daily to protect ecosystems and detect illegal fishing. Traditional systems bottleneck under load. Build a scalable monitoring platform that identifies maritime violations instantly — no heavy distributed pipelines, minimum 50,000 messages/sec, millisecond alert latency.

### 1.2 What every competing team will build
**Ingest AIS → check geofence → alert on breach.** Correct, and table stakes. It is also blind to the standard evasion technique: a vessel disables its AIS transponder before crossing a protected boundary, transits dark, and reappears inside. No message arrives during the gap, so a geofence-only system has nothing to check.

### 1.3 Why "alert on absence" alone is not a differentiator — and what is
Silence-based dark-vessel alerting is not a novel idea. It's the first thing every real vendor in this space tries, and the published literature is explicit that it fails on its own: a system that treats every AIS gap as suspicious generates unworkable false-positive rates, because coverage gaps, equipment failure, and legitimate silence in open ocean are common and indistinguishable from evasion using AIS data alone. Real systems compensate with multi-sensor fusion — combining AIS with RF, optical, and SAR — because both SAR and optical satellite passes independently produce false positives and false negatives, so no single sensor is trusted alone.

Varuna does not claim to have solved that fusion problem in 24 hours. What Varuna claims — and can prove — is narrower and true:

1. **Latency class nobody else occupies.** Skylight's VIIRS nighttime-detection pipeline, the current state of the art for spotting non-broadcasting vessels, runs at roughly 2–3 hours average latency from image capture to detection, at 750-meter resolution, and only works at night under clear sky. Sentinel-1 SAR has a multi-day revisit cycle. These are analyst-facing intelligence platforms. Varuna is a real-time enforcement-facing layer: it turns an AIS gap into an actionable, timestamped, evidence-backed alert in milliseconds, at the moment the gap crosses a zone-calibrated threshold — not hours later in a dashboard.
2. **A fusion-ready architecture, not a fusion claim.** Every alert carries a slot for corroborating evidence (SAR tile, VIIRS hit, RF ping) that upgrades its confidence when a slow signal arrives later. We built and demoed one such slot end-to-end (SAR). We are honest that we did not build all of them.
3. **Two cheap, real gaps closed that most hackathon teams miss entirely:** duplicate-identity (MMSI spoofing) detection, and a machine-auditable reasoning trace on every alert — both detailed in §3.

### 1.4 Honest scope
Varuna is a **decision-support tool for human investigators.** Every alert is a confidence-scored, evidence-backed lead to be triaged by a watch officer — never an autonomous verdict. Stating this proactively is engineering maturity, not weakness.

---

## 2. Architecture

### 2.1 System diagram

```
┌──────────────────────────────────────────────────────┐
│              SYNTHETIC AIS GENERATOR                  │
│  N vessels · great-circle movement · scripted events  │
│  (dark-transit, boundary approach, loitering,          │
│   duplicate-MMSI conflict)                             │
└─────────────────────┬────────────────────────────────┘
                      │  raw messages
                      ▼
┌──────────────────────────────────────────────────────┐
│              IN-MEMORY CHANNEL BUFFER                 │
│  Batched ticks (~20ms) — amortizes per-message cost   │
│  No broker. No serialization hop. No network round-   │
│  trip. This is why we have no distributed pipeline.   │
└─────────────────────┬────────────────────────────────┘
                      │  batches
                      ▼
┌────────────────────────────────────────────────────────────┐
│              PROCESSING CORE  (Go)                           │
│                                                               │
│ ┌──────────────┐ ┌───────────────┐ ┌────────────────────┐   │
│ │  GEOFENCE    │ │   ABSENCE      │ │  IDENTITY-CONFLICT  │   │
│ │  ENGINE      │ │   ENGINE       │ │  ENGINE             │   │
│ │              │ │                │ │                     │   │
│ │ Grid-hash    │ │ Per-vessel     │ │ Same MMSI, two      │   │
│ │ O(1) lookup  │ │ ring buffer    │ │ kinematically-       │   │
│ │ → ray-cast   │ │ Zone-aware     │ │ impossible positions │   │
│ │ PIP test     │ │ tolerance      │ │ within elapsed time  │   │
│ │ State:       │ │ State:         │ │ → identity_conflict  │   │
│ │ outside→     │ │ present→       │ │                      │   │
│ │ inside       │ │ absent→        │ │                      │   │
│ │ fires ONE    │ │ reappeared     │ │                      │   │
│ │ alert        │ │                │ │                      │   │
│ └──────┬───────┘ └───────┬────────┘ └──────────┬──────────┘   │
│        │                 │                     │              │
│        └────────┬────────┴──────────┬──────────┘              │
│                  ▼                              ▼              │
│      ┌───────────────────────┐   ┌─────────────────────────┐  │
│      │   ALERT OBJECT        │   │  REASONING TRACE         │  │
│      │  vessel_id · type     │   │  inputs fired · values   │  │
│      │  position · zone      │   │  thresholds compared     │  │
│      │  confidence · ts      │   │  modality availability   │  │
│      │  evidence slot(s)     │   │  engine version           │  │
│      └───────────┬───────────┘   └─────────────────────────┘  │
└──────────────────┼─────────────────────────────────────────────┘
                    │
       ┌────────────┼────────────────┐
       ▼                             ▼
┌──────────────────┐      ┌───────────────────────┐
│  EMBEDDED STORE  │      │  WEBSOCKET FAN-OUT    │
│  SQLite / BoltDB │      │  → All connected      │
│  alerts + tracks │      │    dashboards          │
└──────────┬───────┘      │  Backpressure handled │
           │               │  Slow consumers don't │
           │ slow async    │  block ingestion       │
           ▼               └───────────────────────┘
┌──────────────────────────────────────────────────────┐
│  CORROBORATION JOBS  (isolated — never on hot path)   │
│  SAR: pre-downloaded Sentinel-1 tile, SAR-trained model│
│  VIIRS (stub): mocked night-lights hit, same interface │
│  Upgrades matching alerts to "corroborated"            │
│  Labeled explicitly as offline/retrospective in demo   │
└──────────────────────────────────────────────────────┘
```

### 2.2 Why this satisfies the "no heavy pipeline" constraint
The constraint eliminates Kafka, Flink, Spark, and any broker-mediated architecture. Varuna's answer: the synthetic generator and the processing core communicate through a **Go channel with batched ticks** — an in-process ring buffer. No network hop, no serialization cost, no broker process. The entire ingestion-to-alert path runs as goroutines inside a single binary. For a single-machine 50k/sec target, a broker adds operational overhead with no throughput benefit — this is the correct design, not a shortcut.

### 2.3 Why Go
Go was chosen for concurrency ergonomics, not because the throughput target demanded a systems fight. Goroutines map cleanly onto per-client WebSocket fan-out, and the channel-based pipeline needs no external synchronization primitives. Honest answer if asked "why not Python/Node": goroutines make per-client fan-out structurally clean, and Go's GC profile under sustained write load is more predictable than a Node.js event loop under the same pressure. Rust would give tighter latency control; Go's concurrency model was faster to build correctly inside 24 hours.

---

## 3. Functional Requirements

### 3.1 Ingestion
- **Synthetic AIS generator:** configurable vessel count, great-circle movement, realistic heading/speed/cadence. Scripted events: dark-transit near boundary, loitering, and duplicate-MMSI conflict — all judge-triggerable on demand.
- **Batched-tick processing:** messages accumulate for ~20ms per tick and process as a batch. This single design choice — not language choice — is the primary lever for clearing the throughput floor.
- **Message schema:** `{vessel_id, mmsi, lat, lon, heading, speed_knots, timestamp_ms}`.
- **Real-data note:** a production deployment sits on a commercial AIS feed (Spire, MarineTraffic, Kpler). Varuna is the processing and inference layer, not the data source.

### 3.2 Geofence Engine
**Two-tier spatial check:**
- Tier 1 — grid-hash cell lookup: the ocean is divided into fixed-size lat/lon cells; each zone polygon precomputes overlapping cells at startup; every incoming position maps to one cell via integer arithmetic. No zone overlap → skip.
- Tier 2 — precise ray-cast PIP, only for vessels in a boundary cell (typically 5–10% of checks).

**State machine alerting:** `{vessel_id → zone_membership_set}` per vessel. An alert fires on transition (outside→inside / inside→outside), not steady-state — a vessel inside a zone for 10 minutes generates one alert, not thousands.

**Boundary hysteresis — new in v4, closes a real gap most teams miss entirely:** raw GPS/AIS position noise near a boundary line is a well-documented cause of alert flapping — a vessel sitting near the edge of a zone can appear to cross in and out repeatedly due to normal position error, and each spurious crossing fires a duplicate alert unless the system compensates. Varuna adds a small inward/outward hysteresis buffer (configurable per zone, defaulting to a few times the expected AIS position error) around each boundary: a state transition only commits once a position clears the buffer, not the raw polygon edge. This is a handful of lines on top of existing zone geometry, but it is the difference between a demo that survives a judge manually nudging a vessel back and forth across a boundary live, and one that visibly spams duplicate alerts under the exact test a judge is most likely to try.

**If asked "isn't PIP fast anyway?":** Yes — optimized PIP libraries clear millions of checks per second on a single core. The value here isn't that two-tier filtering heroically enables 50k/sec; it's that it keeps the hot path clean and avoids wasted work on messages that are obviously irrelevant.

### 3.3 Absence Engine
**Core insight:** if a vessel's last known position was within a configurable proximity buffer of a zone boundary, and it has gone silent beyond the zone-appropriate tolerance, that is a suspicious gap regardless of whether a subsequent message ever triggers a geofence check.

**Per-vessel state:** ring buffer of last N position records, last-seen timestamp, absence state (`PRESENT → SUSPICIOUS_DARK → CORROBORATED / UNRESOLVED`), and this vessel's own historical gap pattern in this zone (distinguishing "this vessel always goes quiet here" from "this is new behavior").

**Zone-dependent silence thresholds:**

| Zone type | Typical AIS cadence | Varuna silence tolerance |
|---|---|---|
| Coastal / terrestrial coverage | 2–10 sec | Cadence + 60 sec buffer |
| Offshore / satellite-only | 10–30 min | Cadence × 3 + coverage-gap estimate |
| Open ocean | Variable, often many hours | Conservative — stated as limitation, not tuned for demo optics |

Open-ocean AIS gaps of many hours are well documented as routine rather than anomalous (see IMO and academic AIS-coverage literature). A flat global threshold would flood any real deployment with false positives. The zone-dependent model is the correct response, and this is said proactively, before a judge has to ask.

**Alert trigger logic:**
```
IF (now - vessel.last_seen) > zone_tolerance(vessel.current_zone)
AND vessel.last_known_position WITHIN boundary_buffer_km(vessel.current_zone)
AND vessel.absence_state == PRESENT
THEN:
    raise suspected_dark_transit(
        vessel_id,
        confidence = score(silence_ratio, boundary_proximity, historical_gap_pattern),
        projected_reentry_cone = project(last_heading, last_speed, elapsed_time)
    )
    vessel.absence_state = SUSPICIOUS_DARK
```

**Confidence score inputs:** silence duration as a ratio of zone tolerance; boundary proximity at last signal; whether this gap is anomalous for this vessel's own history; time-of-day factor (dark transits concentrate at night, empirically observed in enforcement literature).

**Reappearance handling:** plausible position given elapsed time/heading/speed → confidence unchanged or reduced; implausible jump → confidence escalated, flag `kinematic_anomaly`; reappears on the far side of the boundary it went dark near → confidence escalated significantly, flag `suspected_zone_crossing`; no reappearance in a configurable window → promote to `unresolved_dark_vessel`.

**Stated limitation, said before it's asked:** the confidence score is an unvalidated first-pass heuristic — no labeled incident dataset exists in a 24-hour build window. A production system needs a corpus of real evasion incidents to calibrate it. The value demonstrated here is the architecture and the triage discipline around it, not a trained model.

### 3.4 Identity-Conflict Engine — new in v4
**The gap this closes:** real maritime-fraud research documents vessels broadcasting the same MMSI simultaneously from two different physical locations — a physical impossibility that signals spoofing or identity theft, but one that geofence and absence logic alone never catch, because each individual message looks perfectly valid in isolation.

**Logic:** for every incoming message, check the last known position for that MMSI. If a second message under the same MMSI reports a position that is physically unreachable within the elapsed time (distance / elapsed_time > a generous maximum-vessel-speed bound), raise `identity_conflict` with both conflicting positions attached as evidence. This runs on state Varuna already maintains per vessel — it is a few lines of comparison logic, not a new subsystem, and it is the single cheapest, highest-credibility addition in this document.

### 3.5 Loitering Detection
Rolling low-speed + tight-radius check: if a vessel maintains speed below `loiter_speed_threshold` (e.g., < 3 knots) within `loiter_radius_m` for longer than `loiter_time_window`, inside or near a sensitive zone, raise `suspected_illegal_fishing`. Pure stateful rule on the same ring buffer — no ML, no training data.

**Stated limitation:** legitimate anchoring, tidal waiting, or weather delay produce the same signature. This rule is deterministic and cheap, not proof of intent — it is a lead generator, same as every other alert type here.

### 3.6 Alerting & Reasoning Trace — expanded in v4
Alert object:
```json
{
  "alert_id": "uuid",
  "type": "geofence_breach | suspected_dark_transit | suspected_illegal_fishing | identity_conflict | unresolved_dark_vessel",
  "vessel_id": "MMSI or synthetic ID",
  "timestamp": "ISO8601",
  "position": { "lat": 0.0, "lon": 0.0 },
  "zone": "zone_name",
  "confidence": 0.0,
  "evidence": {
    "silence_duration_s": 0,
    "boundary_proximity_km": 0.0,
    "conflicting_position": null
  },
  "reasoning_trace": {
    "inputs_evaluated": ["silence_ratio", "boundary_proximity", "historical_gap_pattern", "time_of_day"],
    "thresholds_used": { "zone_tolerance_s": 0, "boundary_buffer_km": 0.0 },
    "modalities_available": ["ais"],
    "engine_version": "absence-engine-v1"
  },
  "corroboration": { "status": "none | pending | corroborated", "source": null }
}
```

**Why the reasoning trace exists:** published work on operational dark-vessel detection systems explicitly calls for models to be positioned as triage tools with an auditable trace — timestamps, which data modalities were available, model/engine versions, and the thresholds applied — rather than a bare confidence number a watch officer has to trust blindly. This turns "confidence: 0.74" into something an investigator can actually interrogate and defend in a report.

Pushed over WebSocket to all connected dashboards immediately, no polling. Backpressure handling: slow clients get a bounded queue; if they fall too far behind they receive an `alerts_dropped` notification and re-sync, so a slow browser tab never blocks ingestion.

### 3.7 Lead Package Export
Clicking any alert opens the evidence panel: full vessel track history up to the alert; silent period rendered as a **dotted projected path**, never a confirmed route; reappearance event annotated with plausibility score; full reasoning trace visible; one-click export as structured JSON, labeled **"investigative lead — not legal evidence"** in the export header.

### 3.8 Performance Dashboard (Always Visible)
1. **Throughput panel:** live messages/sec counter, climbing past 50,000 and holding. Frame it once, verbally, then move attention to latency — throughput is the floor, not the story.
2. **Latency panel:** p50 and p99 end-to-end latency histogram. End-to-end means: timestamp when a message enters ingestion → timestamp when the corresponding alert or position update reaches a connected WebSocket client. Not geometry-check duration alone — this is the number that's actually hard under sustained fan-out.

### 3.9 Corroboration Jobs (Offline, Isolated, Correctly Framed)
Runs as a separate goroutine on a slow ticker, never on the ingestion hot path.

**SAR (built, demoed live):** one pre-downloaded Sentinel-1 GRD tile, matched to the scripted vessel's last known position before it went dark. Model: a ship detector fine-tuned on a SAR-specific dataset (SSDD, HRSID, or SAR-Ship-Dataset) — not a COCO-pretrained checkpoint, which has never seen a ship in SAR imagery and detects nothing.

**VIIRS night-lights (stub, architecture demoed):** a mocked corroboration hit using the same interface the SAR job uses, showing the alert object accepting a second independent modality without any code change to the absence or geofence engines. We are explicit in the demo that this is a stub, not a working nighttime-detection model — the point is the fusion-ready interface, not a fabricated capability.

**Demo framing (say once, plainly):** *"In production, satellite corroboration jobs run on a slow background ticker and mostly find nothing, because Sentinel-1's revisit cycle is measured in days and VIIRS nighttime detection runs at hours-scale latency even in the best production systems today. We're showing the one case where we happen to have a pre-downloaded tile that covers this vessel's last position, plus a stubbed second modality to show the alert object is built to accept corroboration from any source without re-architecting. This is how the system behaves once a real satellite pass or nighttime detection lands — not a claim of real-time satellite coverage."*

---

## 4. Tech Stack

### Backend — Go (single binary)
| Component | Library / Approach |
|---|---|
| HTTP + WebSocket server | `net/http` + `nhooyr.io/websocket` |
| Ingestion pipeline | Native goroutines + buffered channels |
| Spatial index | Custom `map[cellID][]*Zone` — no external GIS library needed |
| Persistence | `bbolt` (pure-Go embedded KV) or `mattn/go-sqlite3` |
| Background jobs | `time.Ticker` goroutines, isolated from hot path |

### Frontend — React + Vite
| Component | Library |
|---|---|
| Map | MapLibre GL JS (open-source, no API key required) |
| Vessel rendering at volume | deck.gl ScatterplotLayer on top of MapLibre (cut first under time pressure — MapLibre alone is sufficient) |
| Live charts | Chart.js |
| WebSocket client | Native browser WebSocket API — no socket.io |

### Corroboration jobs (offline only)
| Component | Approach |
|---|---|
| SAR imagery | 1–2 pre-downloaded Sentinel-1 GRD tiles from Copernicus Open Access Hub |
| SAR model | `ultralytics` YOLO fine-tuned on SSDD/HRSID, or a pre-trained SAR ship detector from HuggingFace, run via `onnxruntime` |
| VIIRS stub | Static mocked detection record, same JSON shape a real corroboration source would produce |
| Isolation | Separate goroutines, never touch the ingestion path |

### Explicitly excluded
No Kafka, NATS, RabbitMQ, or any message broker. No Docker orchestration for the demo. No auth layer. No zone-authoring UI (hardcoded GeoJSON is fine). No live public AIS feed on stage (unreliable — use pre-recorded/synthetic).

---

## 5. Benchmark Methodology

Every throughput claim is stated with context. "We hit 50k/sec" means nothing on its own. This does:

> *"Sustained 50,000 messages/sec for 120 seconds on a 4-core 2.4GHz laptop with 8GB RAM, 5 concurrent WebSocket clients connected. Message size: 64 bytes. Active zones: 8 polygons. p50 end-to-end latency: 12ms. p99 end-to-end latency: 38ms. Zero dropped alerts. Zero missed state transitions. Benchmark code is in the repo."*

**Measurements to capture and display:** hardware (CPU model, core count, RAM); message size; zone count and polygon complexity; WebSocket client count during benchmark; sustained throughput over 120 seconds (not peak burst); p50/p95/p99 end-to-end latency; alert correctness (scripted crossings vs. alerts fired, must be 1:1); false-positive rate (must be zero for geofence; stated and explained for absence/identity engines).

**If asked "isn't 50k/sec easy?":** *"Yes — optimized point-in-polygon clears millions of checks per second on a single core. We treat 50k as the baseline to clear cleanly, not the headline. What we're proud of is p99 latency under sustained multi-client fan-out, absence detection that fires before any message arrives, and an identity-conflict check most teams won't think to build at all."*

---

## 6. Demo Script — 2 Minutes 45 Seconds

**[0:00 – 0:20] Open on the live map**
Vessels already moving, throughput counter already past 50,000, latency histogram already live.
Say: *"This is Varuna. Fifty thousand vessel messages a second are spatially evaluated against eight geofence zones and delivered to every connected dashboard. p99 end-to-end — ingestion to your browser — is under 40 milliseconds. One Go process. No broker."*

**[0:20 – 0:40] Baseline: geofence works as expected**
Point to a vessel approaching a boundary. Say: *"Standard system: vessel enters a zone, alert fires. [Crosses. Alert fires.] That's table stakes. Here's what a geofence-only system structurally can't do."*

**[0:40 – 1:20] Differentiator: dark transit**
Trigger the scripted event. Say: *"This vessel just went dark near the boundary. A geofence-only system has nothing to check right now — it's silent. Watch the absence engine."* After the scripted silence window, `suspected_dark_transit` fires with confidence score, reasoning trace, and projected re-entry cone.
Say: *"This fired before the vessel reappeared, because the evasion happens during the silence, not after. Click it — here's exactly why it fired: silence ratio, boundary proximity, this vessel's own gap history. Nothing here is a black box."*

**[1:20 – 1:40] Identity conflict**
Trigger the scripted duplicate-MMSI event. Say: *"Second failure mode nobody else in this room is checking for: this MMSI just reported two positions that are physically impossible to reach in the time between them. That's not a gap — that's spoofing or identity theft. Alert fires instantly, both positions attached as evidence."*

**[1:40 – 2:00] Vessel reappears, lead package**
Say: *"It reappeared inside the zone. Our alert was already open. [Click.] Track before the gap, projected path during silence as a dotted line, reappearance plausibility score, full reasoning trace. One-click export, labeled investigative lead, not legal evidence."*

**[2:00 – 2:20] Corroboration**
Say: *"This alert falls under a Sentinel-1 pass we pre-downloaded — in production, most alerts never get a satellite match, because SAR revisit is days and VIIRS nighttime detection runs hours behind even in production systems today. Today we have one match. It ran offline and upgraded this alert to corroborated. We also stub a second modality slot — VIIRS — to show the alert object accepts any corroboration source without touching the core engines."*

**[2:20 – 2:45] Close on limitations, own them**
Say: *"Four things Varuna does not do, and we know exactly why: it doesn't detect vessels that never broadcast AIS — that needs constant-coverage radar or optical fusion, a different sensor stack entirely. The absence-engine confidence score is heuristic, not calibrated against a labeled incident dataset — that's the production next step. Satellite corroboration is retrospective, not real-time. And we are not claiming to have solved multi-sensor fusion — we built one working corroboration path and one stubbed interface to prove the architecture accepts more. We built the real-time processing layer. The sensor infrastructure is the deployment problem, and it's a solved one — we're the piece that was missing underneath it."*

Sit down.

---

## 7. Q&A Preparation

| Question | Answer |
|---|---|
| "Isn't 50k/sec easy on modern hardware?" | "Yes — PIP clears millions of checks/sec on a single core. It's the baseline we had to clear first. Our headline is p99 end-to-end latency under sustained fan-out, absence detection, and identity-conflict detection." |
| "Isn't silence-based dark-vessel detection just what everyone already does?" | "Yes, and we say so upfront — Windward, GFW, and Starboard all do it, and their own published work says single-modality gap detection alone produces unworkable false-positive rates. What's actually underserved is sub-second reflex alerting at the point of ingestion — their systems run hours-to-days behind by sensor design. We're that missing real-time layer, built to hand off to their slower, higher-confidence signals when they land." |
| "Why Go instead of Python/Node?" | "Goroutines make per-client WebSocket fan-out structurally clean, and Go's GC profile under sustained write load is more predictable than Node's event loop under the same pressure. Rust would give tighter latency control; Go's concurrency ergonomics were faster to build correctly in 24 hours." |
| "How did you validate the confidence score?" | "We didn't — no labeled incident dataset exists in a 24-hour build. The score is a heuristic based on silence ratio, boundary proximity, and vessel-specific gap history, and every alert carries the full reasoning trace so an investigator can see exactly why it fired rather than trusting a number blindly. A production system needs a corpus of known evasion incidents to calibrate it." |
| "What about vessels that never broadcast AIS?" | "Out of scope by design — that needs constant-coverage radar or optical satellite fusion, a fundamentally different sensor stack. We process cooperative AIS data and say so." |
| "What's your false positive rate?" | "Geofence breach: zero, it's deterministic. Absence detection: tunable — demo threshold is 3x this vessel's observed gap cadence in this zone; open ocean needs a far more conservative threshold since multi-hour gaps are normal there. Identity conflict: near-zero by construction, since it only fires on a physical impossibility, not a heuristic." |
| "Can this scale beyond one machine?" | "The grid-hash + channel model partitions cleanly by geographic region — one instance per sea area, aggregate alerts upstream. We haven't built that because the problem statement explicitly rules out distributed pipelines. Correct scoping for this brief." |
| "Why not use Windward or Global Fishing Watch?" | "We're not replacing them — they're deep intelligence platforms with sensors we don't have. GFW's own nighttime-detection pipeline runs at roughly 2–3 hours latency; Sentinel-1 SAR's revisit is measured in days. Neither is built for millisecond enforcement-facing alerting on a live feed. That's the gap Varuna fills — and in a real deployment we'd treat their outputs as corroboration inputs, not competitors." |
| "Your data is all synthetic — how do we trust it?" | "Geofence logic is deterministic and verifiable — script a crossing, confirm the alert. The absence and identity engines are heuristic/rule-based and we say so plainly. Benchmark numbers are on real hardware with reproducible methodology in the repo. What's synthetic is vessel positions — in deployment those come from a commercial AIS feed upstream." |
| "What happens if a vessel sits right on the boundary line?" | "That's the case that breaks most geofence demos — GPS/AIS noise makes the vessel appear to cross in and out repeatedly, and a naive state machine fires a duplicate alert on every flicker. We added a hysteresis buffer around every boundary: a transition only commits once the position clears the buffer, not the raw polygon edge. You're welcome to drag a vessel back and forth across a boundary right now — it won't spam alerts." |
| "Why add identity-conflict detection — wasn't that scope creep?" | "It cost us under an hour because it reuses state we already track per vessel, and it closes a documented real-world gap — MMSI spoofing via duplicate broadcast — that neither geofencing nor absence detection touches at all. Cheapest, highest-credibility addition in the build." |

---

## 8. Build Priority — 24-Hour Window

**Non-negotiable (demo fails without these):**
1. Synthetic generator with scripted dark-transit, loitering, and duplicate-MMSI events
2. Batched ingestion pipeline (Go channels, no broker)
3. Grid-hash geofence engine + state-machine breach detection + boundary hysteresis buffer
4. WebSocket fan-out to React dashboard with live map (MapLibre)
5. Throughput counter + end-to-end latency histogram (always visible)
6. Absence engine with zone-aware tolerance + confidence score
7. Identity-conflict engine (duplicate-MMSI kinematic check)
8. Alert reasoning trace attached to every alert
9. Basic alert panel with type, confidence, vessel ID, reasoning trace

**Build if time remains (in this order):**
10. Lead package export (JSON, labeled correctly)
11. Projected re-entry cone on the map during dark period
12. Loitering detection panel
13. Reappearance plausibility scoring + track replay
14. VIIRS stub corroboration slot (proves fusion-ready interface)
15. SAR enrichment segment (offline tile + SAR-trained model)

**Cut immediately under time pressure:**
- deck.gl (MapLibre is sufficient)
- Zone-authoring UI (hardcoded GeoJSON is fine)
- SAR/VIIRS enrichment (cut entirely if items 1–9 aren't solid first)
- Any animation polish
- PDF export (raw JSON is enough)

**Rule:** a working, benchmarked, demo-ready 1–9 beats a half-finished 1–15. Every time. Items 7–8 (identity-conflict, reasoning trace) are cheap enough that they should never be the thing you cut — they're what separates this submission from every other team's geofence-plus-silence build.

---

## 9. Success Metrics

| Metric | Target | How measured |
|---|---|---|
| Sustained throughput | ≥ 50,000 msgs/sec for 120 seconds | Live counter on dashboard + benchmark log |
| End-to-end p99 alert latency | < 50ms | Ingestion timestamp → WS delivery timestamp, displayed live |
| Alert correctness | 1 alert per scripted geofence crossing | Judge can count; zero duplicates, zero misses |
| Boundary flapping resistance | Zero duplicate alerts when a vessel is manually jittered back and forth across a zone edge within the hysteresis buffer | Live: judge (or team) drags a vessel across the boundary repeatedly during Q&A |
| Absence alert timing | `suspected_dark_transit` fires before vessel reappears | Visible on timeline in demo |
| Identity-conflict correctness | `identity_conflict` fires on every scripted duplicate-MMSI event, never otherwise | Judge can count; zero false triggers on normal traffic |
| Reasoning-trace completeness | Every alert has a non-empty, human-readable trace | Judge clicks any alert during demo |
| Q&A survivability | Every claim traceable to a build decision, live demo, or named limitation | Team knows this document |

---

## 10. Limitations — Own These First

1. **Coverage physics causes most AIS silence, not evasion.** Multi-hour gaps in open ocean are normal. Zone-dependent thresholds mitigate but don't eliminate this. Production calibration needs real incident data.
2. **Absence confidence score is unvalidated.** First-pass heuristic, not a calibrated probability. Named in the pitch.
3. **Does not detect vessels that never broadcast AIS.** Universal dark-fleet detection is a different sensor problem requiring constant-coverage radar or optical fusion.
4. **Position spoofing is not caught unless kinematically impossible or a duplicate-MMSI conflict.** A smooth, single-identity fabricated track passes both engines.
5. **Corroboration is retrospective and partial.** Sentinel-1 revisit is days-scale; VIIRS nighttime detection is hours-scale even in production systems. Most alerts never receive a match. We show one real SAR match and one stubbed second-modality interface — not a working fusion pipeline.
6. **All demo data is synthetic.** The architecture is real. The benchmark is real. The vessel positions are not. A production deployment needs a commercial AIS upstream.
7. **Alerts are leads, not verdicts.** Nothing here is legal-grade evidence or admissible record.
8. **Loitering detection cannot distinguish intent.** Anchoring, tidal waits, and weather delays trigger the same signature as suspicious loitering.

---

## 11. Future Scope

- **Calibrated ML confidence:** replace the heuristic absence score with a model trained on labeled dark-transit incident reports from coast guard logs
- **Real spoofing detection beyond duplicate-MMSI:** kinematic consistency checks flagging vessels whose reported speed/heading are physically implausible given prior track, even under a single identity
- **Working VIIRS integration:** replace the stubbed corroboration slot with a real nighttime-light detection call, following the same pattern Global Fishing Watch's own Skylight integration uses
- **Multi-region partitioning:** the grid-hash + channel model partitions cleanly by sea area — the horizontal scaling path without distributed-pipeline overhead
- **Real AIS ingestion:** swap the synthetic generator for a Spire or AISHub stream — the processing core doesn't change
- **India deployment:** India's coastline spans roughly 7,500 km with numerous islands and creeks that make comprehensive small-vessel surveillance difficult; this system is deployable on a single server at an existing coastal command node as a real-time layer beneath NC3I, not a replacement for it

---

## 12. Glossary

| Term | Definition |
|---|---|
| AIS | Automatic Identification System — vessel transponder broadcasting position, heading, speed over VHF |
| Dark transit | A vessel that disables its AIS transponder, typically near a boundary it is not authorized to cross |
| Identity conflict | Same MMSI broadcasting two physically-impossible-to-reconcile positions — a spoofing/identity-theft signal |
| EEZ | Exclusive Economic Zone — the 200 nautical mile zone where a coastal state has sovereign rights over resources |
| Geofence | A virtual polygon boundary over a real geographic area; crossing triggers an alert |
| Hysteresis buffer | A margin around a boundary that a position must clear before a zone-membership state change commits, preventing duplicate alerts from GPS/position noise near the edge |
| Grid-hash | A spatial index dividing the map into fixed cells; O(1) lookup maps any lat/lon to its cell |
| PIP | Point-in-polygon — determines whether a coordinate is inside a polygon boundary |
| SAR | Synthetic Aperture Radar — active satellite sensor that works day/night and through cloud cover |
| VIIRS | Visible Infrared Imaging Radiometer Suite — satellite sensor used for nighttime vessel-light detection |
| Reasoning trace | A structured record attached to each alert showing which inputs, thresholds, and modalities produced it |
| Ring buffer | Fixed-size circular buffer holding the last N vessel positions; old entries overwritten |
| Fan-out | Broadcasting one alert to all connected WebSocket clients simultaneously |
| p99 latency | The latency value below which 99% of all measurements fall — the "worst realistic case" |
| Lead package | A structured export of alert evidence for investigator handoff; explicitly not legal evidence |

---

*Version 4.0 — every specific number in this document is either demonstrated live, sourced from published research, or explicitly labeled as a limitation. No unsourced statistics, no invented pricing figures, no unverifiable claims about existing government infrastructure. If a claim cannot be defended under adversarial Q&A, it is not in this document.*
