# WebSocket Contract: Alerts Stream

**Endpoint**: `ws://localhost:8080/ws/alerts`
**Direction**: Server → Client (push only)
**Volume**: Low (event-driven, one per alert)

## Message Format

```json
{
  "alert_id": "uuid",
  "type": "geofence_breach | suspected_dark_transit | suspected_illegal_fishing | identity_conflict | unresolved_dark_vessel",
  "vessel_id": "string",
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

## Control Messages

### Alerts Dropped
```json
{ "type": "alerts_dropped", "count": 5 }
```
Client must show a non-blocking toast and resync.

## Client Requirements

- Prepend new alerts to list (newest first)
- Cap list at 100 entries for UI performance
- Handle `alerts_dropped` control messages
