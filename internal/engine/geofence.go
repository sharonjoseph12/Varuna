package engine

import "time"

// checkGeofence performs the two-tier geofence check with hysteresis.
func (e *Engine) checkGeofence(vs *VesselState, msg AISMessage, ingestTime time.Time) {
	// Tier 1: grid-hash cell lookup — which zones might this position overlap?
	candidates := e.grid.ZonesAt(msg.Lat, msg.Lon)

	// Track which zones this vessel is currently in
	currentlyInside := make(map[string]bool)

	for _, zone := range candidates {
		// Tier 2: ray-cast PIP for candidate zones
		inside := PointInPolygon(msg.Lat, msg.Lon, zone.Polygon)
		distToEdge := DistToPolygonEdgeDeg(msg.Lat, msg.Lon, zone.Polygon)

		prevState, exists := vs.ZoneMembership[zone.ID]
		if !exists {
			prevState = MembershipOutside
		}

		newState := e.computeTransition(prevState, inside, distToEdge, zone.HysteresisMarginDeg)
		vs.ZoneMembership[zone.ID] = newState

		if inside {
			currentlyInside[zone.ID] = true
		}

		// Fire alert only on genuine transition (outside→inside committed)
		if prevState != MembershipInside && newState == MembershipInside {
			alert := Alert{
				AlertID:   newAlertID(),
				Type:      "geofence_breach",
				VesselID:  msg.VesselID,
				Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
				Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
				Zone:      zone.Name,
				Confidence: 1.0, // geofence is deterministic
				Evidence:   map[string]interface{}{
					"distance_to_edge_deg": distToEdge,
					"hysteresis_margin_deg": zone.HysteresisMarginDeg,
				},
				ReasoningTrace: ReasoningTrace{
					InputsEvaluated:     []string{"zone_membership_transition", "position", "hysteresis_check"},
					ThresholdsUsed:      map[string]float64{"hysteresis_margin_deg": zone.HysteresisMarginDeg},
					ModalitiesAvailable: []string{"ais"},
					EngineVersion:       "geofence-v1",
				},
				Corroboration: Corroboration{Status: "none"},
			}
			e.emitAlert(alert, ingestTime)
		}

		// Fire exit alert
		if prevState == MembershipInside && newState == MembershipOutside {
			alert := Alert{
				AlertID:   newAlertID(),
				Type:      "geofence_breach",
				VesselID:  msg.VesselID,
				Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
				Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
				Zone:      zone.Name,
				Confidence: 1.0,
				Evidence:   map[string]interface{}{
					"transition":          "exit",
					"distance_to_edge_deg": distToEdge,
				},
				ReasoningTrace: ReasoningTrace{
					InputsEvaluated:     []string{"zone_membership_transition", "position", "hysteresis_check"},
					ThresholdsUsed:      map[string]float64{"hysteresis_margin_deg": zone.HysteresisMarginDeg},
					ModalitiesAvailable: []string{"ais"},
					EngineVersion:       "geofence-v1",
				},
				Corroboration: Corroboration{Status: "none"},
			}
			e.emitAlert(alert, ingestTime)
		}
	}

	// For zones not in candidates, if vessel was inside, check if they've exited
	for zoneID, state := range vs.ZoneMembership {
		if state == MembershipInside && !currentlyInside[zoneID] {
			// Not even in a candidate cell anymore — definitely exited
			inCandidate := false
			for _, z := range candidates {
				if z.ID == zoneID {
					inCandidate = true
					break
				}
			}
			if !inCandidate {
				vs.ZoneMembership[zoneID] = MembershipOutside
				// Find zone for alert
				for i := range e.zones {
					if e.zones[i].ID == zoneID {
						alert := Alert{
							AlertID:   newAlertID(),
							Type:      "geofence_breach",
							VesselID:  msg.VesselID,
							Timestamp: time.UnixMilli(msg.TimestampMs).UTC().Format(time.RFC3339),
							Position:  LatLon{Lat: msg.Lat, Lon: msg.Lon},
							Zone:      e.zones[i].Name,
							Confidence: 1.0,
							Evidence:   map[string]interface{}{"transition": "exit"},
							ReasoningTrace: ReasoningTrace{
								InputsEvaluated:     []string{"zone_membership_transition", "grid_cell_change"},
								ThresholdsUsed:      map[string]float64{},
								ModalitiesAvailable: []string{"ais"},
								EngineVersion:       "geofence-v1",
							},
							Corroboration: Corroboration{Status: "none"},
						}
						e.emitAlert(alert, ingestTime)
						break
					}
				}
			}
		}
	}
}

// computeTransition implements the hysteresis state machine.
// A transition only commits once the position clears the margin past the polygon edge.
func (e *Engine) computeTransition(prev MembershipState, inside bool, distToEdge, margin float64) MembershipState {
	switch prev {
	case MembershipOutside:
		if inside && distToEdge > margin {
			return MembershipInside // genuine entry — cleared margin
		}
		if inside {
			return MembershipPendingEntry // inside but within margin — wait
		}
		return MembershipOutside

	case MembershipInside:
		if !inside && distToEdge > margin {
			return MembershipOutside // genuine exit — cleared margin
		}
		if !inside {
			return MembershipPendingExit // outside but within margin — wait
		}
		return MembershipInside

	case MembershipPendingEntry:
		if inside && distToEdge > margin {
			return MembershipInside // now cleared margin
		}
		if !inside {
			return MembershipOutside // retreated
		}
		return MembershipPendingEntry // still within margin

	case MembershipPendingExit:
		if !inside && distToEdge > margin {
			return MembershipOutside // now cleared margin
		}
		if inside {
			return MembershipInside // re-entered
		}
		return MembershipPendingExit // still within margin
	}
	return prev
}
