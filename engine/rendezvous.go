package engine

import (
	"fmt"
	"sort"
	"time"
)

// RendezvousState tracks an ongoing proximity event between two vessels.
type RendezvousState struct {
	VesselA     string
	VesselB     string
	StartMs     int64
	LastCheckMs int64
	Alerted     bool // prevent duplicate alerts for same event
}

// checkRendezvous detects two vessels maintaining <500m distance at <2 knots for >30 min.
// ponytail: O(n²) on slow-vessel candidates only; ceiling is fine for demo's ~10 vessels.
// Upgrade path: spatial index to pre-filter by grid cell.
func (e *Engine) checkRendezvous() {
	const (
		maxSpeedKnots  = 2.0
		maxDistanceM   = 500.0
		minDurationMin = 30.0
	)

	now := time.Now().UnixMilli()

	// Build candidate set: vessels currently going slow
	e.vesselsMu.RLock()
	type candidate struct {
		id       string
		lat, lon float64
		speed    float64
	}
	var slow []candidate
	for _, vs := range e.vessels {
		if vs.LastSeen == 0 {
			continue
		}
		spd := vs.lastSpeed()
		if spd < maxSpeedKnots {
			slow = append(slow, candidate{vs.VesselID, vs.LastLat, vs.LastLon, spd})
		}
	}
	e.vesselsMu.RUnlock()

	if len(slow) < 2 {
		return
	}

	// Pairwise distance check
	for i := 0; i < len(slow); i++ {
		for j := i + 1; j < len(slow); j++ {
			a, b := slow[i], slow[j]
			distM := haversineKm(a.lat, a.lon, b.lat, b.lon) * 1000

			key := rendezvousKey(a.id, b.id)

			if distM > maxDistanceM {
				// Too far apart — clear any tracking
				e.rendezvousMu.Lock()
				delete(e.rendezvous, key)
				e.rendezvousMu.Unlock()
				continue
			}

			e.rendezvousMu.Lock()
			rs, exists := e.rendezvous[key]
			if !exists {
				e.rendezvous[key] = &RendezvousState{
					VesselA:     a.id,
					VesselB:     b.id,
					StartMs:     now,
					LastCheckMs: now,
				}
				e.rendezvousMu.Unlock()
				continue
			}

			rs.LastCheckMs = now
			durationMin := float64(now-rs.StartMs) / 60000.0

			if durationMin >= minDurationMin && !rs.Alerted {
				rs.Alerted = true
				e.rendezvousMu.Unlock()

				// Emit STS alert
				midLat := (a.lat + b.lat) / 2
				midLon := (a.lon + b.lon) / 2

				alert := Alert{
					AlertID:   newAlertID(),
					Type:      "suspected_sts_transfer",
					VesselID:  a.id, // primary vessel
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Position:  LatLon{Lat: midLat, Lon: midLon},
					Zone:      "",
					Confidence: 0.8,
					Evidence: map[string]interface{}{
						"vessel_a":         a.id,
						"vessel_b":         b.id,
						"distance_m":       distM,
						"duration_min":     durationMin,
						"avg_speed_a":      a.speed,
						"avg_speed_b":      b.speed,
						"rendezvous_start": time.UnixMilli(rs.StartMs).UTC().Format(time.RFC3339),
					},
					ReasoningTrace: ReasoningTrace{
						InputsEvaluated: []string{
							"pairwise_distance", "speed_threshold", "duration_threshold",
						},
						ThresholdsUsed: map[string]float64{
							"max_distance_m":    maxDistanceM,
							"max_speed_knots":   maxSpeedKnots,
							"min_duration_min":  minDurationMin,
						},
						ModalitiesAvailable: []string{"ais"},
						EngineVersion:       "rendezvous-v1",
					},
					Corroboration: Corroboration{Status: "none"},
				}
				e.emitAlert(alert, time.Now())
			} else {
				e.rendezvousMu.Unlock()
			}
		}
	}
}

// rendezvousKey produces a stable key from two vessel IDs.
func rendezvousKey(a, b string) string {
	pair := []string{a, b}
	sort.Strings(pair)
	return fmt.Sprintf("%s<>%s", pair[0], pair[1])
}
