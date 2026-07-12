package engine

import "time"

// mmsiState tracks the last known position per MMSI for identity-conflict detection.
type mmsiState struct {
	lat, lon    float64
	timestampMs int64
	vesselID    string
}

// checkIdentityConflict detects same MMSI broadcasting from kinematically impossible positions.
// ponytail: few lines of comparison logic, reuses existing per-vessel state
func (e *Engine) checkIdentityConflict(msg AISMessage, ingestTime time.Time) {
	if msg.MMSI == "" {
		return
	}

	e.vesselsMu.RLock()
	idxVs := e.mmsiIndex[msg.MMSI]
	var prev *mmsiState
	if idxVs != nil && idxVs.VesselID != msg.VesselID && idxVs.LastSeen > 0 {
		prev = &mmsiState{
			lat:         idxVs.LastLat,
			lon:         idxVs.LastLon,
			timestampMs: idxVs.LastSeen,
			vesselID:    idxVs.VesselID,
		}
	}
	e.vesselsMu.RUnlock()

	if prev == nil {
		return
	}

	// Check kinematic impossibility
	distKm := haversineKm(prev.lat, prev.lon, msg.Lat, msg.Lon)
	elapsedMs := msg.TimestampMs - prev.timestampMs
	if elapsedMs <= 0 {
		return // same timestamp or out of order
	}

	elapsedHrs := float64(elapsedMs) / 3600000.0
	maxPossibleKm := e.cfg.MaxVesselSpeedKnots * 1.852 * elapsedHrs

	if distKm <= maxPossibleKm {
		return // plausible — no conflict
	}

	alert := Alert{
		AlertID:   newAlertID(),
		Type:      "identity_conflict",
		VesselID:  msg.VesselID,
		Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
		Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
		Zone:      "", // identity conflict is zone-independent
		Confidence: 0.95, // near-certain — it's a physical impossibility
		Evidence: map[string]interface{}{
			"mmsi":                    msg.MMSI,
			"conflicting_vessel_id":   prev.vesselID,
			"conflicting_position":    LatLon{Lat: prev.lat, Lon: prev.lon},
			"distance_km":            distKm,
			"elapsed_seconds":        float64(elapsedMs) / 1000,
			"max_possible_km":        maxPossibleKm,
			"max_vessel_speed_knots": e.cfg.MaxVesselSpeedKnots,
		},
		ReasoningTrace: ReasoningTrace{
			InputsEvaluated: []string{
				"mmsi_match", "position_pair", "kinematic_feasibility",
			},
			ThresholdsUsed: map[string]float64{
				"max_vessel_speed_knots": e.cfg.MaxVesselSpeedKnots,
				"distance_km":           distKm,
				"max_possible_km":       maxPossibleKm,
			},
			ModalitiesAvailable: []string{"ais"},
			EngineVersion:       "identity-v1",
		},
		Corroboration: Corroboration{Status: "none"},
	}
	e.emitAlert(alert, ingestTime)
}
