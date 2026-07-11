// Package generator provides the synthetic AIS vessel generator for Varuna.
// It emits AISMessage structs onto a Go channel consumed by the engine via
// engine.Ingest(in <-chan AISMessage).
package generator

import "time"

// AISMessage is the shared contract with teammate 1's ingestion engine.
// Field names and JSON tags must match exactly.
type AISMessage struct {
	VesselID    string  `json:"vessel_id"`
	MMSI        string  `json:"mmsi"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	HeadingDeg  float64 `json:"heading"`
	SpeedKnots  float64 `json:"speed_knots"`
	TimestampMs int64   `json:"timestamp_ms"`
}

// ScriptedEvent enumerates the four judge-triggerable demo scenarios.
type ScriptedEvent int

const (
	EventNone            ScriptedEvent = iota
	EventDarkTransit                   // vessel stops emitting for a configurable window
	EventLoitering                     // vessel holds low speed in tight radius
	EventDuplicateMMSI                 // vessel broadcasts another vessel's MMSI
	EventGeofenceCrossing              // vessel steers toward a zone boundary
)

// DarkVariant selects the reappearance behaviour after a dark-transit event.
type DarkVariant string

const (
	VariantPlausible DarkVariant = "plausible" // reappear at projected position
	VariantJump      DarkVariant = "jump"      // reappear at a kinematically impossible position
)

// VesselState holds per-vessel simulation state. Protected by Generator.mu.
type VesselState struct {
	ID            string
	MMSI          string
	Lat           float64
	Lon           float64
	HeadingDeg    float64
	SpeedKnots    float64
	CadenceMs     int64         // milliseconds between position updates
	ZoneType      string        // "coastal" | "offshore" | "open_ocean"
	ActiveEvent   ScriptedEvent
	DarkUntil     time.Time // non-zero while vessel is intentionally silent
	DarkVariant   DarkVariant
	SilenceS      int    // duration of dark window in seconds
	LoiterCenterLat float64
	LoiterCenterLon float64
	LoiterUntil     time.Time
	DuplicateMMSI   string // the MMSI this vessel is currently spoofing
	TargetZone      string // zone name for geofence-crossing event
}

// Config configures the generator.
type Config struct {
	VesselCount  int    // number of simulated vessels
	OutputBuffer int    // buffered channel size
	TriggerPort  int    // HTTP trigger server port
	DataDir      string // path to data/ directory
}
