# HTTP Contract: Metrics Endpoint

**Endpoint**: `GET /metrics`
**Direction**: Client polls server
**Frequency**: Every 1-2 seconds

## Response Format

```json
{
  "throughput_per_sec": 0,
  "p50_latency_ms": 0,
  "p99_latency_ms": 0
}
```

## Client Requirements

- Poll every 1-2 seconds via `setInterval` + `fetch`
- Display latest values prominently
- Maintain trailing array of last 60 readings for latency chart
