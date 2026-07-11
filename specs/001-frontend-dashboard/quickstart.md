# Quickstart: Varuna Frontend Dashboard

## Prerequisites
- Node.js 18+ and npm

## Setup
```bash
cd dashboard
npm install
```

## Run (standalone with mock data)
```bash
# Terminal 1: start mock server
node mock-server.js

# Terminal 2: start dev server
npm run dev
```

Open http://localhost:5173 — vessels should be moving, metrics updating, alerts firing within 2 seconds.

## Run (against real backend)
```bash
# Just start dev server — backend must be running on localhost:8080
npm run dev
```

## What you should see on first load
1. Map with translucent zone boundaries
2. Vessel markers moving and rotating by heading
3. Throughput counter past 50,000
4. Latency panel showing p50/p99
5. Alerts appearing in the side panel

## Debug Controls
- **Boundary Jitter Slider**: Bottom-right corner. Drag to move a test vessel across zone edges. Proves hysteresis works.
