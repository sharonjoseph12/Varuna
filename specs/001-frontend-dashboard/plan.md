# Implementation Plan: Frontend Maritime Surveillance Dashboard

**Branch**: `001-frontend-dashboard` | **Date**: 2026-07-11 | **Spec**: [spec.md](file:///c:/Users/siddharth/Desktop/Varuna/specs/001-frontend-dashboard/spec.md)

**Input**: Feature specification from `specs/001-frontend-dashboard/spec.md`

## Summary

Build a React + Vite single-page dashboard that renders a live MapLibre map with vessel positions and geofence zones, a real-time alert panel with expandable reasoning traces, and a performance panel (throughput + latency). Consumes two WebSocket streams (`/ws/positions`, `/ws/alerts`) and one HTTP endpoint (`/metrics`). Ships with a standalone Node mock server so development is never blocked on the Go backend.

## Technical Context

**Language/Version**: JavaScript/JSX (ES2022), Node.js 18+

**Primary Dependencies**: React 18, Vite 5, MapLibre GL JS, Chart.js, react-chartjs-2

**Storage**: N/A — all state is ephemeral (WebSocket streams + in-memory)

**Testing**: Manual verification against mock server; visual demo verification

**Target Platform**: Modern desktop browser (Chrome/Edge, demo on stage)

**Project Type**: Web application (single-page, frontend-only)

**Performance Goals**: ≥30 FPS with 50k msgs/sec position stream; <500ms alert panel update; <200ms alert expansion

**Constraints**: No API keys, no auth, no build-time dependencies beyond npm. Must run standalone against mock data. Must load and display moving vessels + metrics within 2 seconds.

**Scale/Scope**: Single-page dashboard, ~8 components, 1 mock server script

## Constitution Check

*GATE: Constitution is unfilled template — no project-specific gates defined. Passes by default.*

## Project Structure

### Documentation (this feature)

```text
specs/001-frontend-dashboard/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
dashboard/
├── index.html
├── package.json
├── vite.config.js
├── mock-server.js          # Standalone Node WS + HTTP mock
├── src/
│   ├── main.jsx            # Entry point
│   ├── App.jsx             # Root layout: map + panels
│   ├── App.css             # Global styles
│   ├── hooks/
│   │   ├── useWebSocket.js     # Generic WS hook with reconnect
│   │   ├── usePositions.js     # Batched position state via rAF
│   │   ├── useAlerts.js        # Alert list state + dropped handling
│   │   └── useMetrics.js       # Polling /metrics
│   ├── components/
│   │   ├── MapView.jsx         # MapLibre map + zones + vessels + cones
│   │   ├── AlertPanel.jsx      # Alert list + expansion
│   │   ├── AlertDetail.jsx     # Expanded alert: trace, dotted path, export
│   │   ├── MetricsPanel.jsx    # Throughput + latency display
│   │   ├── JitterControl.jsx   # Debug slider/draggable for boundary test
│   │   └── Toast.jsx           # Non-blocking notification for dropped alerts
│   ├── utils/
│   │   ├── reasoningTrace.js   # Translate trace JSON → human-readable text
│   │   ├── exportLead.js       # JSON export with disclaimer header
│   │   └── zones.js            # Hardcoded GeoJSON zone data
│   └── styles/
│       └── index.css           # Design system tokens + component styles
└── public/
    └── favicon.ico
```

**Structure Decision**: Single `dashboard/` directory at repo root. No backend code — that's teammate 1's domain. Flat component structure, no routing (single page). Hooks encapsulate all data fetching and state management.

## Complexity Tracking

No constitution violations to justify.
