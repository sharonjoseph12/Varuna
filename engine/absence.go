package engine

import (
	"math"
	"time"
)

// runAbsenceChecks scans all vessels for silence gaps on each tick.
func (e *Engine) runAbsenceChecks() {
	now := time.Now().UnixMilli()

	e.vesselsMu.RLock()
	vessels := make([]*VesselState, 0, len(e.vessels))
	for _, vs := range e.vessels {
		vessels = append(vessels, vs)
	}
	e.vesselsMu.RUnlock()

	for _, vs := range vessels {
		if vs.LastSeen == 0 || vs.AbsState != AbsencePresent {
			continue
		}

		silenceMs := now - vs.LastSeen
		zone := e.findNearestZone(vs.LastLat, vs.LastLon)
		if zone == nil {
			continue // no zone nearby — skip
		}

		toleranceMs := zone.SilenceToleranceSec * 1000
		if silenceMs <= toleranceMs {
			continue
		}

		// Check boundary proximity
		boundaryProximityKm := e.boundaryProximityKm(vs.LastLat, vs.LastLon, zone)
		if boundaryProximityKm > zone.BoundaryBufferKm {
			continue // not near boundary — less suspicious
		}

		// Score confidence
		confidence := e.scoreAbsenceConfidence(vs, silenceMs, toleranceMs, boundaryProximityKm, zone)

		vs.AbsState = AbsenceSuspiciousDark

		alert := Alert{
			AlertID:   newAlertID(),
			Type:      "suspected_dark_transit",
			VesselID:  vs.VesselID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Position:  LatLon{Lat: vs.LastLat, Lon: vs.LastLon},
			Zone:      zone.Name,
			Confidence: confidence,
			Evidence: map[string]interface{}{
				"silence_duration_s":    float64(silenceMs) / 1000,
				"boundary_proximity_km": boundaryProximityKm,
				"zone_tolerance_s":      zone.SilenceToleranceSec,
				"last_heading":          vs.lastHeading(),
				"last_speed_knots":      vs.lastSpeed(),
			},
			ReasoningTrace: ReasoningTrace{
				InputsEvaluated: []string{
					"silence_ratio", "boundary_proximity",
					"historical_gap_pattern", "time_of_day",
				},
				ThresholdsUsed: map[string]float64{
					"zone_tolerance_s":      float64(zone.SilenceToleranceSec),
					"boundary_buffer_km":    zone.BoundaryBufferKm,
					"silence_duration_s":    float64(silenceMs) / 1000,
				},
				ModalitiesAvailable: []string{"ais"},
				EngineVersion:       "absence-v1",
			},
			Corroboration: Corroboration{Status: "none"},
		}
		e.emitAlert(alert, time.Now())
	}
}

// handleReappearance processes a vessel reappearing after a dark period.
func (e *Engine) handleReappearance(vs *VesselState, msg AISMessage, ingestTime time.Time) {
	if vs.AbsState == AbsencePresent {
		return
	}

	if vs.AbsState == AbsenceSuspiciousDark || vs.AbsState == AbsenceUnresolved {
		// Check if reappearance is plausible
		prevPos, ok := vs.LastPosition()
		if !ok {
			vs.AbsState = AbsencePresent
			return
		}

		distKm := haversineKm(prevPos.Lat, prevPos.Lon, msg.Lat, msg.Lon)
		elapsedHrs := float64(msg.TimestampMs-prevPos.TimestampMs) / 3600000.0
		maxPossibleKm := e.cfg.MaxVesselSpeedKnots * 1.852 * elapsedHrs

		zone := e.findNearestZone(prevPos.Lat, prevPos.Lon, msg.Lat, msg.Lon)
		crossedBoundary := false
		if zone != nil {
			// Check if vessel was outside and is now inside (or vice versa)
			wasInside := PointInPolygon(prevPos.Lat, prevPos.Lon, zone.Polygon)
			nowInside := PointInPolygon(msg.Lat, msg.Lon, zone.Polygon)
			crossedBoundary = wasInside != nowInside
		}

		if distKm > maxPossibleKm*1.5 {
			// Implausible jump — escalate
			alert := Alert{
				AlertID:   newAlertID(),
				Type:      "suspected_dark_transit",
				VesselID:  msg.VesselID,
				Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
				Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
				Zone:      e.zoneName(zone),
				Confidence: 0.9,
				Evidence: map[string]interface{}{
					"kinematic_anomaly":    true,
					"distance_km":         distKm,
					"max_possible_km":     maxPossibleKm,
					"previous_position":    LatLon{Lat: prevPos.Lat, Lon: prevPos.Lon},
				},
				ReasoningTrace: ReasoningTrace{
					InputsEvaluated: []string{"reappearance_plausibility", "kinematic_check"},
					ThresholdsUsed:  map[string]float64{"max_vessel_speed_knots": e.cfg.MaxVesselSpeedKnots},
					ModalitiesAvailable: []string{"ais"},
					EngineVersion:       "absence-v1",
				},
				Corroboration: Corroboration{Status: "none"},
			}
			e.emitAlert(alert, ingestTime)
		}

		if crossedBoundary {
			alert := Alert{
				AlertID:   newAlertID(),
				Type:      "suspected_dark_transit",
				VesselID:  msg.VesselID,
				Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
				Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
				Zone:      e.zoneName(zone),
				Confidence: 0.85,
				Evidence: map[string]interface{}{
					"suspected_zone_crossing": true,
					"previous_position":       LatLon{Lat: prevPos.Lat, Lon: prevPos.Lon},
					"crossed_boundary":        true,
				},
				ReasoningTrace: ReasoningTrace{
					InputsEvaluated: []string{"boundary_crossing_during_dark", "reappearance_position"},
					ThresholdsUsed:  map[string]float64{},
					ModalitiesAvailable: []string{"ais"},
					EngineVersion:       "absence-v1",
				},
				Corroboration: Corroboration{Status: "none"},
			}
			e.emitAlert(alert, ingestTime)
		}

		vs.AbsState = AbsencePresent
	}
}

func (e *Engine) scoreAbsenceConfidence(vs *VesselState, silenceMs, toleranceMs int64, boundaryProximityKm float64, zone *Zone) float64 {
	score := 0.0

	// Silence ratio: how much beyond tolerance
	silenceRatio := float64(silenceMs) / float64(toleranceMs)
	if silenceRatio > 3 {
		score += 0.3
	} else if silenceRatio > 2 {
		score += 0.2
	} else {
		score += 0.1
	}

	// Boundary proximity: closer = more suspicious
	if boundaryProximityKm < 1 {
		score += 0.3
	} else if boundaryProximityKm < 3 {
		score += 0.2
	} else {
		score += 0.1
	}

	// Historical gap pattern: is this anomalous?
	if len(vs.GapHistory) > 3 {
		avgGap := avgInt64(vs.GapHistory)
		if silenceMs > avgGap*3 {
			score += 0.25 // anomalous gap
		} else {
			score += 0.05 // normal for this vessel
		}
	} else {
		score += 0.15 // not enough history
	}

	// Time of day: dark transits concentrate at night (empirical)
	hour := time.Now().UTC().Hour()
	if hour >= 20 || hour < 6 {
		score += 0.15
	} else {
		score += 0.05
	}

	// ponytail: cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	return score
}

func (e *Engine) findNearestZone(coords ...float64) *Zone {
	// ponytail: simple — check all zones, return the closest one whose boundary buffer covers any of the coords
	var nearest *Zone
	minDist := math.MaxFloat64

	for i := range e.zones {
		for j := 0; j < len(coords)-1; j += 2 {
			lat, lon := coords[j], coords[j+1]
			dist := DistToPolygonEdgeDeg(lat, lon, e.zones[i].Polygon)
			distKm := dist * 111.0 // rough deg → km
			if distKm < e.zones[i].BoundaryBufferKm && distKm < minDist {
				minDist = distKm
				nearest = &e.zones[i]
			}
		}
	}
	return nearest
}

func (e *Engine) boundaryProximityKm(lat, lon float64, zone *Zone) float64 {
	dist := DistToPolygonEdgeDeg(lat, lon, zone.Polygon)
	return dist * 111.0 // rough deg → km
}

func (e *Engine) zoneName(zone *Zone) string {
	if zone == nil {
		return "unknown"
	}
	return zone.Name
}

func (vs *VesselState) lastHeading() float64 {
	if pos, ok := vs.LastPosition(); ok {
		return pos.HeadingDeg
	}
	return 0
}

func (vs *VesselState) lastSpeed() float64 {
	if pos, ok := vs.LastPosition(); ok {
		return pos.SpeedKnots
	}
	return 0
}

func avgInt64(vals []int64) int64 {
	if len(vals) == 0 {
		return 0
	}
	var sum int64
	for _, v := range vals {
		sum += v
	}
	return sum / int64(len(vals))
}

// haversineKm calculates the great-circle distance between two lat/lon points in km.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
