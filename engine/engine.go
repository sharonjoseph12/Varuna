package engine

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Engine is the core processing engine.
type Engine struct {
	cfg    Config
	zones  []Zone
	grid   *Grid
	store  *Store

	vessels   map[string]*VesselState // vesselID → state
	mmsiIndex map[string]*VesselState // mmsi → latest state
	vesselsMu sync.RWMutex

	// Alert storage for corroboration lookups
	alertStore   map[string]*Alert
	alertStoreMu sync.RWMutex

	// Trust engine
	trustScores map[string]*TrustScore
	trustMu     sync.RWMutex

	// Dark intent model (simulated ONNX)
	darkModel *DarkIntentModel

	// Rendezvous / STS tracking
	rendezvous   map[string]*RendezvousState
	rendezvousMu sync.Mutex

	alertCh    chan Alert
	positionCh chan PositionUpdate

	// Metrics tracking
	totalProcessed atomic.Int64
	totalAlerts    atomic.Int64
	latencies      []float64 // ring buffer of latencies in ms
	latIdx         int
	latMu          sync.Mutex
	throughputLog  []throughputSample
	tpMu           sync.Mutex

	startTime time.Time
}

type throughputSample struct {
	timestamp time.Time
	count     int64
}

const latencyBufSize = 10000

// NewEngine creates and initializes the engine with precomputed grid.
func NewEngine(cfg Config, zones []Zone) *Engine {
	e := &Engine{
		cfg:         cfg,
		zones:       zones,
		grid:        NewGrid(cfg.GridCellSizeDeg, zones),
		vessels:     make(map[string]*VesselState),
		mmsiIndex:   make(map[string]*VesselState),
		alertStore:  make(map[string]*Alert),
		trustScores: make(map[string]*TrustScore),
		darkModel:   NewDarkIntentModel(),
		rendezvous:  make(map[string]*RendezvousState),
		alertCh:     make(chan Alert, cfg.AlertChannelSize),
		positionCh: make(chan PositionUpdate, cfg.PositionChannelSize),
		latencies:  make([]float64, latencyBufSize),
		startTime:  time.Now(),
	}

	// Initialize store (async, non-blocking)
	if cfg.StorePath != "" {
		var err error
		e.store, err = NewStore(cfg.StorePath, cfg.StoreWriteBufferSize)
		if err != nil {
			log.Printf("warning: store init failed: %v (continuing without persistence)", err)
		}
	}

	return e
}

// Ingest consumes AIS messages from the input channel using concurrent workers.
// Blocks until the input channel is closed.
func (e *Engine) Ingest(in <-chan AISMessage) {
	// Start periodic checks
	ticker := time.NewTicker(time.Duration(e.cfg.TickIntervalMs) * time.Millisecond)
	go func() {
		for range ticker.C {
			e.runAbsenceChecks()
			e.checkRendezvous()
		}
	}()

	// Start a fixed number of workers to process messages concurrently
	const numWorkers = 8
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for msg := range in {
				e.processMessage(msg, time.Now())
			}
		}()
	}

	wg.Wait()
	ticker.Stop()
}

func (e *Engine) processMessage(msg AISMessage, ingestTime time.Time) {
	e.totalProcessed.Add(1)

	// Get or create vessel state
	vs := e.getOrCreateVessel(msg.VesselID, msg.MMSI)

	// Lock the vessel state for concurrent workers
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Record gap for absence engine
	if vs.LastSeen > 0 && msg.TimestampMs > vs.LastSeen {
		gap := msg.TimestampMs - vs.LastSeen
		vs.GapHistory = append(vs.GapHistory, gap)
		// ponytail: cap gap history at 100, trim oldest
		if len(vs.GapHistory) > 100 {
			vs.GapHistory = vs.GapHistory[len(vs.GapHistory)-100:]
		}
	}

	// Update vessel state
	vs.AddPosition(msg)
	vs.LastSeen = msg.TimestampMs
	vs.LastLat = msg.Lat
	vs.LastLon = msg.Lon
	vs.MMSI = msg.MMSI

	// Emit position update
	e.emitPosition(msg, ingestTime)

	// Identity conflict check (runs before geofence, uses MMSI state)
	e.checkIdentityConflict(msg, ingestTime)

	// Geofence check
	e.checkGeofence(vs, msg, ingestTime)

	// Loitering check
	e.checkLoitering(vs, msg, ingestTime)

	// Trust evaluation
	e.evaluateTrust(msg, ingestTime)

	// Handle reappearance for absence engine
	e.handleReappearance(vs, msg, ingestTime)

	// Persist track
	if e.store != nil {
		e.store.WriteTrack(msg)
	}
}

func (e *Engine) getOrCreateVessel(vesselID, mmsi string) *VesselState {
	e.vesselsMu.RLock()
	vs, ok := e.vessels[vesselID]
	e.vesselsMu.RUnlock()
	if ok {
		return vs
	}

	e.vesselsMu.Lock()
	// Double-check after write lock
	vs, ok = e.vessels[vesselID]
	if !ok {
		vs = &VesselState{
			VesselID:       vesselID,
			MMSI:           mmsi,
			ZoneMembership: make(map[string]MembershipState),
			AbsState:       AbsencePresent,
		}
		e.vessels[vesselID] = vs
		if mmsi != "" {
			e.mmsiIndex[mmsi] = vs
		}
	}
	e.vesselsMu.Unlock()
	return vs
}

func (e *Engine) emitPosition(msg AISMessage, ingestTime time.Time) {
	pu := PositionUpdate{
		VesselID:    msg.VesselID,
		Lat:         msg.Lat,
		Lon:         msg.Lon,
		HeadingDeg:  msg.HeadingDeg,
		SpeedKnots:  msg.SpeedKnots,
		TimestampMs: msg.TimestampMs,
	}
	select {
	case e.positionCh <- pu:
	default:
		// ponytail: drop if channel full, never block hot path
	}
	e.recordLatency(ingestTime)
}

func (e *Engine) emitAlert(alert Alert, ingestTime time.Time) {
	e.totalAlerts.Add(1)

	// Store for corroboration lookup
	e.alertStoreMu.Lock()
	alertCopy := alert
	e.alertStore[alert.AlertID] = &alertCopy
	e.alertStoreMu.Unlock()

	select {
	case e.alertCh <- alert:
	default:
		// ponytail: drop if channel full, never block hot path
	}

	// Persist alert
	if e.store != nil {
		e.store.WriteAlert(alert)
	}

	e.recordLatency(ingestTime)
}

func (e *Engine) recordLatency(ingestTime time.Time) {
	latMs := float64(time.Since(ingestTime).Microseconds()) / 1000.0
	e.latMu.Lock()
	e.latencies[e.latIdx] = latMs
	e.latIdx = (e.latIdx + 1) % latencyBufSize
	e.latMu.Unlock()
}

// Alerts returns the read-only alert channel for teammate 2.
func (e *Engine) Alerts() <-chan Alert {
	return e.alertCh
}

// Positions returns the read-only position channel for teammate 2.
func (e *Engine) Positions() <-chan PositionUpdate {
	return e.positionCh
}

// Corroborate upgrades an alert's corroboration status. Thread-safe.
func (e *Engine) Corroborate(alertID string, source string, evidence map[string]interface{}) {
	e.alertStoreMu.Lock()
	defer e.alertStoreMu.Unlock()
	a, ok := e.alertStore[alertID]
	if !ok {
		return
	}
	a.Corroboration.Status = "corroborated"
	a.Corroboration.Source = &source
	for k, v := range evidence {
		if a.Evidence == nil {
			a.Evidence = make(map[string]interface{})
		}
		a.Evidence[k] = v
	}
}

// GetAlert returns an alert by ID (for corroboration verification).
func (e *Engine) GetAlert(alertID string) (Alert, bool) {
	e.alertStoreMu.RLock()
	defer e.alertStoreMu.RUnlock()
	a, ok := e.alertStore[alertID]
	if !ok {
		return Alert{}, false
	}
	return *a, true
}

// Metrics returns a snapshot of current throughput and latency stats.
func (e *Engine) Metrics() Metrics {
	total := e.totalProcessed.Load()
	elapsed := time.Since(e.startTime).Seconds()
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(total) / elapsed
	}

	p50, p95, p99 := e.computeLatencyPercentiles()

	return Metrics{
		ThroughputMsgsSec: throughput,
		LatencyP50Ms:      p50,
		LatencyP95Ms:      p95,
		LatencyP99Ms:      p99,
		TotalProcessed:    total,
		TotalAlerts:       e.totalAlerts.Load(),
	}
}

func (e *Engine) computeLatencyPercentiles() (p50, p95, p99 float64) {
	e.latMu.Lock()
	// Collect non-zero latencies
	var lats []float64
	for _, l := range e.latencies {
		if l > 0 {
			lats = append(lats, l)
		}
	}
	e.latMu.Unlock()

	if len(lats) == 0 {
		return 0, 0, 0
	}

	sort.Float64s(lats)
	p50 = percentile(lats, 0.50)
	p95 = percentile(lats, 0.95)
	p99 = percentile(lats, 0.99)
	return
}

func percentile(sorted []float64, p float64) float64 {
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

var alertCounter atomic.Uint64

func newAlertID() string {
	return fmt.Sprintf("alert-%d-%d", time.Now().UnixNano(), alertCounter.Add(1))
}

// Close shuts down the engine and store.
func (e *Engine) Close() {
	if e.store != nil {
		e.store.Close()
	}
}
