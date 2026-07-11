package engine

import (
	"math"
	"time"
)

// checkLoitering detects low-speed + tight-radius behavior near sensitive zones.
// ponytail: pure stateful rule on ring buffer, no ML
func (e *Engine) checkLoitering(vs *VesselState, msg AISMessage, ingestTime time.Time) {
	if msg.SpeedKnots >= e.cfg.LoiterSpeedThreshold {
		// Moving too fast — reset loiter state
		vs.LoiterStart = 0
		return
	}

	// Check if near a sensitive zone
	zone := e.findNearestZone(msg.Lat, msg.Lon)
	if zone == nil {
		// Also check if inside any zone
		candidates := e.grid.ZonesAt(msg.Lat, msg.Lon)
		for _, z := range candidates {
			if PointInPolygon(msg.Lat, msg.Lon, z.Polygon) {
				zone = z
				break
			}
		}
		if zone == nil {
			vs.LoiterStart = 0
			return
		}
	}

	if vs.LoiterStart == 0 {
		// Start tracking loitering
		vs.LoiterStart = msg.TimestampMs
		vs.LoiterAnchor = LatLon{Lat: msg.Lat, Lon: msg.Lon}
		return
	}

	// Check radius — is the vessel still within the loiter radius?
	distM := haversineKm(vs.LoiterAnchor.Lat, vs.LoiterAnchor.Lon, msg.Lat, msg.Lon) * 1000
	if distM > e.cfg.LoiterRadiusM {
		// Moved too far — reset
		vs.LoiterStart = msg.TimestampMs
		vs.LoiterAnchor = LatLon{Lat: msg.Lat, Lon: msg.Lon}
		return
	}

	// Check duration
	durationMin := float64(msg.TimestampMs-vs.LoiterStart) / 60000.0
	if durationMin < float64(e.cfg.LoiterTimeWindowMin) {
		return // not long enough yet
	}

	// Fire loitering alert
	alert := Alert{
		AlertID:   newAlertID(),
		Type:      "suspected_illegal_fishing",
		VesselID:  msg.VesselID,
		Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
		Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
		Zone:      zone.Name,
		Confidence: 0.6, // ponytail: loitering is low-confidence — anchoring/weather indistinguishable
		Evidence: map[string]interface{}{
			"loiter_duration_min": math.Round(durationMin*10) / 10,
			"loiter_radius_m":    math.Round(distM*10) / 10,
			"avg_speed_knots":    msg.SpeedKnots,
			"anchor_position":    vs.LoiterAnchor,
		},
		ReasoningTrace: ReasoningTrace{
			InputsEvaluated: []string{"speed_threshold", "radius_check", "duration_check", "zone_proximity"},
			ThresholdsUsed: map[string]float64{
				"loiter_speed_threshold_knots": e.cfg.LoiterSpeedThreshold,
				"loiter_radius_m":             e.cfg.LoiterRadiusM,
				"loiter_time_window_min":       float64(e.cfg.LoiterTimeWindowMin),
			},
			ModalitiesAvailable: []string{"ais"},
			EngineVersion:       "loitering-v1",
		},
		Corroboration: Corroboration{Status: "none"},
	}
	e.emitAlert(alert, ingestTime)

	// Reset loiter tracking after alert
	vs.LoiterStart = msg.TimestampMs
}
