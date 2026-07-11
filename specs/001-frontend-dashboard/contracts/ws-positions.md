# WebSocket Contract: Positions Stream

**Endpoint**: `ws://localhost:8080/ws/positions`
**Direction**: Server → Client (push only)
**Volume**: Up to 50,000+ messages/second

## Message Format

```json
{
  "vessel_id": "string",
  "lat": 0.0,
  "lon": 0.0,
  "heading": 0.0,
  "speed_knots": 0.0,
  "timestamp_ms": 0
}
```

## Client Requirements

- Buffer incoming messages in a `Map<vessel_id, position>`
- Flush to map render via `requestAnimationFrame` batching
- Never re-render per individual message
