# Prompt for Teammate 2 — Frontend Dashboard (React + MapLibre)

Paste this whole file into your AI coding assistant at the start of the build. The full PRD (`Varuna_PRD_v4_FINAL.md`) is your source of truth for anything not covered here.

---

## Your role
You own the **dashboard**: the live map, the throughput/latency panels, the alert panel with reasoning trace, and the lead-package export. You do NOT own the Go engine internals or the synthetic generator — those are teammates 1 and 3. You consume two WebSocket streams and one HTTP metrics endpoint that teammate 1 exposes. Build against these contracts so you can start immediately with mocked data and swap in the real backend the moment it's ready, without waiting on teammate 1.

## Shared contracts — build your mock data to match these exactly

**WebSocket: `ws://localhost:8080/ws/alerts`** — server pushes one JSON object per alert:
```json
{
  "alert_id": "uuid",
  "type": "geofence_breach | suspected_dark_transit | suspected_illegal_fishing | identity_conflict | unresolved_dark_vessel",
  "vessel_id": "string",
  "timestamp": "ISO8601",
  "position": { "lat": 0.0, "lon": 0.0 },
  "zone": "zone_name",
  "confidence": 0.0,
  "evidence": { "silence_duration_s": 0, "boundary_proximity_km": 0.0, "conflicting_position": null },
  "reasoning_trace": {
    "inputs_evaluated": ["silence_ratio", "boundary_proximity", "historical_gap_pattern", "time_of_day"],
    "thresholds_used": { "zone_tolerance_s": 0, "boundary_buffer_km": 0.0 },
    "modalities_available": ["ais"],
    "engine_version": "absence-engine-v1"
  },
  "corroboration": { "status": "none | pending | corroborated", "source": null }
}
```

**WebSocket: `ws://localhost:8080/ws/positions`** — server pushes live vessel position updates for map rendering: `{ "vessel_id": "string", "lat": 0.0, "lon": 0.0, "heading": 0.0, "speed_knots": 0.0, "timestamp_ms": 0 }`. Handle high volume gracefully — this stream is the 50k/sec one, so throttle your render loop (e.g. requestAnimationFrame batching), don't re-render per message.

**HTTP: `GET /metrics`** — polled every 1–2s for the performance panel: `{ "throughput_per_sec": 0, "p50_latency_ms": 0, "p99_latency_ms": 0 }`.

**Backpressure handling:** if the WebSocket connection falls behind, the server sends `{"type": "alerts_dropped", "count": N}` — show a small non-blocking toast, then resync (the server will re-send current state).

## What to build, in order
1. **Live map (MapLibre GL JS, no API key needed)** with hardcoded GeoJSON zone polygons rendered as translucent boundaries. Start with static demo zones so you're not blocked on teammate 3's data.
2. **Vessel rendering** on the map from the positions WebSocket. Use simple circle/triangle markers oriented by heading — cut deck.gl entirely, MapLibre native rendering is sufficient and faster to build correctly.
3. **Throughput + latency panel**, always visible, polling `/metrics`. This must be on screen from the moment the demo starts — it's the first thing said in the pitch.
4. **Alert panel**: live list from the alerts WebSocket, newest first, color-coded by `type`. Clicking an alert expands: track history, projected path during silence rendered as a **dotted line** (never solid — a dotted line signals "we don't know this, we're projecting it"), the full reasoning trace in human-readable form (not raw JSON — translate `inputs_evaluated` + `thresholds_used` into a sentence or two), and corroboration status.
5. **Re-entry cone**: when a `suspected_dark_transit` alert is open, render a projected cone on the map from last known heading/speed — this is a nice-to-have, build it after 1–4 are solid.
6. **Lead package export**: one-click button on any expanded alert, downloads the alert JSON with a visible header/watermark saying "investigative lead — not legal evidence."
7. **Boundary-jitter demo control**: a small debug control (a slider or draggable test-vessel marker) that lets the team manually nudge a synthetic vessel back and forth across a zone edge live, on stage, to prove the hysteresis buffer works. This is explicitly called for in the demo script — build it, it's a strong live moment.

## Explicitly do not build
No auth/login screens. No zone-authoring UI — zones are hardcoded GeoJSON, imported as a static file. No animation polish beyond what's needed for the demo to read clearly. No PDF export — raw JSON is enough.

## Definition of done for your piece
- Dashboard runs standalone against mocked WebSocket data (write a tiny local mock server or replay a recorded JSON sequence) so you're never blocked waiting on teammate 1.
- Can visibly demonstrate, on your own: a geofence breach alert firing exactly once per crossing, a dark-transit alert appearing with full reasoning trace, an identity-conflict alert with both positions shown, and zero duplicate alerts when you jitter a test vessel across a boundary.
- Throughput panel visibly climbs past 50,000 and holds, without visibly janking the UI (this means your position-stream renderer must be decoupled from raw message rate — do not naively re-render on every WebSocket message).

If you're blocked on backend details, don't guess — check `Varuna_PRD_v4_FINAL.md` §3.5–§3.7 for the full alert/lead-package spec, or ask teammate 1.
