# Research: Core Processing Engine

## Decision: Spatial Index Approach

**Decision**: Fixed-size grid-hash with `map[CellID][]*Zone`

**Rationale**: O(1) cell lookup via integer division on lat/lon. Each zone precomputes overlapping cells at startup. No external GIS library needed. For 8 zones with moderate polygon complexity, this is the correct tradeoff — R-tree adds complexity without measurable benefit at this scale.

**Alternatives considered**:
- R-tree (e.g., `tidwall/rtree`): More general, but overkill for 8 static zones. Adds a dependency for no throughput gain.
- Brute-force PIP on every zone: O(zones × vertices) per message. Works at 8 zones but wastes CPU on messages nowhere near any zone.

## Decision: Point-in-Polygon Algorithm

**Decision**: Ray-cast (crossing number) algorithm, stdlib implementation (~20 lines).

**Rationale**: Standard, correct on all edge cases, no dependency. Winding number is equivalent; ray-cast is simpler to implement and debug.

**Alternatives considered**:
- `paulmach/orb` library: Full GIS library, massive dependency for one function we need.
- Pre-rasterized bitmaps: Fast lookup but complex setup, memory-heavy, and precision-lossy.

## Decision: Embedded Persistence

**Decision**: `go.etcd.io/bbolt` (pure-Go embedded KV store)

**Rationale**: Pure Go, no CGO, no external process, battle-tested (etcd lineage). Async writes via buffered channel + goroutine. Simplest correct choice.

**Alternatives considered**:
- `mattn/go-sqlite3`: Requires CGO, complicates cross-compilation. More features than needed — we're doing key-value writes.
- Write to flat JSON files: Simplest, but no crash safety, no read-back indexing.

## Decision: Batched-Tick Interval

**Decision**: ~20ms tick (configurable). Accumulate messages in a buffered channel, drain on each tick.

**Rationale**: Amortizes per-message overhead. At 50k msgs/sec, each tick processes ~1000 messages. Keeps latency < 20ms per tick + processing time.

**Alternatives considered**:
- Per-message processing: Higher CPU overhead from goroutine scheduling per message. Won't reliably hit 50k/sec.
- Larger batch windows (100ms+): Better throughput but p99 latency suffers.

## Decision: Hysteresis Implementation

**Decision**: Per-zone configurable margin (default ~100m ≈ 3× typical AIS position error of ~30m). State transition requires position to clear margin on both sides.

**Rationale**: GPS/AIS position error is well-documented at 10-30m. A margin of 3× covers >99% of noise. Simple distance check from polygon edge.

**Alternatives considered**:
- Time-based debounce: Doesn't address the root cause (position noise). A vessel sitting on the boundary would still spam alerts after the debounce window.
- Probabilistic filtering: Overly complex for a deterministic check.

## Decision: Latency Measurement

**Decision**: Record `time.Now()` at ingestion, `time.Now()` when alert/position hits the output channel. Difference is end-to-end latency. Store in a ring buffer for p50/p95/p99 calculation.

**Rationale**: Stdlib `time` package is sufficient. No external metrics library needed. Ring buffer avoids unbounded memory growth.

## Decision: Max Vessel Speed Bound for Identity Conflict

**Decision**: 50 knots (≈ 92.6 km/h). Generous upper bound — fastest commercial vessels are ~30 knots; military vessels rarely exceed 40.

**Rationale**: Conservative to avoid false positives. Only fires when the position pair is physically impossible, not merely unlikely.

All NEEDS CLARIFICATION items resolved. No unknowns remain.
