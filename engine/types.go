package engine

import "sync"

// AISMessage is the inbound vessel position report from teammate 3's generator.
type AISMessage struct {
	VesselID    string  `json:"vessel_id"`
	MMSI        string  `json:"mmsi"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	HeadingDeg  float64 `json:"heading"`
	SpeedKnots  float64 `json:"speed_knots"`
	TimestampMs int64   `json:"timestamp_ms"`
}

// Alert is the outbound detection result sent to teammate 2's WebSocket fan-out.
type Alert struct {
	AlertID    string  `json:"alert_id"`
	Type       string  `json:"type"` // geofence_breach | suspected_dark_transit | suspected_illegal_fishing | identity_conflict | unresolved_dark_vessel
	VesselID   string  `json:"vessel_id"`
	Timestamp  string  `json:"timestamp"` // ISO8601
	Position   LatLon  `json:"position"`
	Zone       string  `json:"zone"`
	Confidence float64 `json:"confidence"`
	Evidence   map[string]interface{} `json:"evidence"`
	ReasoningTrace ReasoningTrace `json:"reasoning_trace"`
	Corroboration  Corroboration  `json:"corroboration"`
}

type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type ReasoningTrace struct {
	InputsEvaluated     []string           `json:"inputs_evaluated"`
	ThresholdsUsed      map[string]float64 `json:"thresholds_used"`
	ModalitiesAvailable []string           `json:"modalities_available"`
	EngineVersion       string             `json:"engine_version"`
}

type Corroboration struct {
	Status string  `json:"status"` // none | pending | corroborated
	Source *string `json:"source"`
}

// PositionUpdate is sent to teammate 2 for live map rendering.
type PositionUpdate struct {
	VesselID    string  `json:"vessel_id"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	HeadingDeg  float64 `json:"heading"`
	SpeedKnots  float64 `json:"speed_knots"`
	TimestampMs int64   `json:"timestamp_ms"`
}

// Zone defines a geofence polygon with precomputed grid cells.
type Zone struct {
	ID                  string
	Name                string
	Type                string       // coastal | offshore | open_ocean
	Polygon             [][2]float64 // ordered vertices [lat, lon]
	HysteresisMarginDeg float64     // ~0.001 ≈ 100m
	SilenceToleranceSec int64
	BoundaryBufferKm    float64
	GridCells           []CellID // precomputed at startup
}

// CellID identifies a grid cell in the spatial index.
type CellID struct {
	LatCell int
	LonCell int
}

// Config holds all engine configuration.
type Config struct {
	TickIntervalMs        int
	GridCellSizeDeg       float64
	MaxVesselSpeedKnots   float64
	DefaultHysteresisMarginDeg float64
	LoiterSpeedThreshold  float64
	LoiterRadiusM         float64
	LoiterTimeWindowMin   int
	AlertChannelSize      int
	PositionChannelSize   int
	StoreWriteBufferSize  int
	StorePath             string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		TickIntervalMs:             20,
		GridCellSizeDeg:            0.1, // ~11km cells
		MaxVesselSpeedKnots:        50,
		DefaultHysteresisMarginDeg: 0.001, // ~100m
		LoiterSpeedThreshold:       3.0,
		LoiterRadiusM:              500,
		LoiterTimeWindowMin:        30,
		AlertChannelSize:           10000,
		PositionChannelSize:        50000,
		StoreWriteBufferSize:       1000,
		StorePath:                  "varuna.db",
	}
}

// Metrics holds throughput and latency stats.
type Metrics struct {
	ThroughputMsgsSec float64 `json:"throughput_msgs_sec"`
	LatencyP50Ms      float64 `json:"latency_p50_ms"`
	LatencyP95Ms      float64 `json:"latency_p95_ms"`
	LatencyP99Ms      float64 `json:"latency_p99_ms"`
	TotalProcessed    int64   `json:"total_processed"`
	TotalAlerts       int64   `json:"total_alerts"`
}

// AbsenceState tracks the absence detection state machine per vessel.
type AbsenceState int

const (
	AbsencePresent       AbsenceState = iota
	AbsenceSuspiciousDark
	AbsenceUnresolved
)

// ponytail: membership uses a simple enum, not a full state machine struct
type MembershipState int

const (
	MembershipOutside MembershipState = iota
	MembershipInside
	MembershipPendingEntry // within hysteresis margin, waiting to confirm
	MembershipPendingExit
)

const RingBufferSize = 32

// VesselState holds all per-vessel tracking state.
type VesselState struct {
	VesselID      string
	MMSI          string
	Positions     [RingBufferSize]AISMessage
	PosIdx        int
	PosCount      int
	ZoneMembership map[string]MembershipState // zoneID → state
	AbsState      AbsenceState
	LastSeen      int64 // timestamp ms
	LastLat       float64
	LastLon       float64
	GapHistory    []int64 // historical gap durations in ms
	LoiterStart   int64   // timestamp ms when loitering began, 0 if not loitering
	LoiterAnchor  LatLon  // center point of loitering
	mu            sync.Mutex
}

// AddPosition appends a position to the ring buffer.
func (v *VesselState) AddPosition(msg AISMessage) {
	v.Positions[v.PosIdx] = msg
	v.PosIdx = (v.PosIdx + 1) % RingBufferSize
	if v.PosCount < RingBufferSize {
		v.PosCount++
	}
}

// LastPosition returns the most recently added position.
func (v *VesselState) LastPosition() (AISMessage, bool) {
	if v.PosCount == 0 {
		return AISMessage{}, false
	}
	idx := (v.PosIdx - 1 + RingBufferSize) % RingBufferSize
	return v.Positions[idx], true
}
