package engine

import (
	"math"
	"time"
)

// SARArea represents the computed search and rescue area for a lost vessel.
type SARArea struct {
	CenterLat float64 `json:"center_lat"` // original last known position
	CenterLon float64 `json:"center_lon"`
	DriftLat  float64 `json:"drift_lat"`  // center after drift projection
	DriftLon  float64 `json:"drift_lon"`
	RadiusKm  float64 `json:"radius_km"`  // search area radius
	VesselID  string  `json:"vessel_id"`
	ElapsedH  float64 `json:"elapsed_h"`  // hours since last contact
}

// Simulated ocean current: 0.5 knots at 135° (SE monsoon, Indian Ocean).
const (
	currentSpeedKnots = 0.5
	currentHeadingDeg = 135.0
)

// ComputeSARArea calculates the primary search area for a vessel that has gone dark.
func (e *Engine) ComputeSARArea(vesselID string) (SARArea, bool) {
	e.vesselsMu.RLock()
	vs, ok := e.vessels[vesselID]
	e.vesselsMu.RUnlock()
	if !ok || vs.LastSeen == 0 {
		return SARArea{}, false
	}

	// Time since last contact
	elapsedMs := time.Now().UnixMilli() - vs.LastSeen
	elapsedH := float64(elapsedMs) / 3600000.0
	if elapsedH < 0.01 {
		elapsedH = 0.5 // minimum projection: 30 min
	}

	lastSpeed := vs.lastSpeed()
	lastHeading := vs.lastHeading()
	if lastSpeed < 0.5 {
		lastSpeed = 2.0 // assume minimum drift speed
	}

	// Project vessel vector
	headRad := lastHeading * math.Pi / 180
	vesselDriftKm := lastSpeed * 1.852 * elapsedH
	driftLat := vs.LastLat + (vesselDriftKm/111.0)*math.Cos(headRad)
	driftLon := vs.LastLon + (vesselDriftKm/(111.0*math.Cos(vs.LastLat*math.Pi/180)))*math.Sin(headRad)

	// Add current vector
	currentRad := currentHeadingDeg * math.Pi / 180
	currentDriftKm := currentSpeedKnots * 1.852 * elapsedH
	driftLat += (currentDriftKm / 111.0) * math.Cos(currentRad)
	driftLon += (currentDriftKm / (111.0 * math.Cos(vs.LastLat*math.Pi/180))) * math.Sin(currentRad)

	// Search radius: expands with time uncertainty
	// ponytail: simple expanding circle, not a Leeway model
	avgSpeedKm := (lastSpeed*1.852 + currentSpeedKnots*1.852) / 2
	radiusKm := 3 * elapsedH * avgSpeedKm
	if radiusKm < 5 {
		radiusKm = 5 // minimum 5km search radius
	}
	if radiusKm > 200 {
		radiusKm = 200 // cap for sanity
	}

	return SARArea{
		CenterLat: vs.LastLat,
		CenterLon: vs.LastLon,
		DriftLat:  driftLat,
		DriftLon:  driftLon,
		RadiusKm:  math.Round(radiusKm*10) / 10,
		VesselID:  vesselID,
		ElapsedH:  math.Round(elapsedH*100) / 100,
	}, true
}
