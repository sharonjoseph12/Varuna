package generator

import (
	"math"
	"math/rand"
	"time"
)

// applyDarkTransit starts a dark-transit event on vessel v.
// silenceS is how long the vessel will be silent. variant controls reappearance.
func applyDarkTransit(v *VesselState, silenceS int, variant DarkVariant) {
	if silenceS <= 0 {
		silenceS = 45 // default silence window matches demo script
	}
	v.ActiveEvent = EventDarkTransit
	v.DarkUntil = time.Now().Add(time.Duration(silenceS) * time.Second)
	v.DarkVariant = variant
	v.SilenceS = silenceS
}

// applyLoitering starts a loitering event: vessel holds < 3 knots in tight radius.
func applyLoitering(v *VesselState, durationS int) {
	if durationS <= 0 {
		durationS = 120
	}
	v.ActiveEvent = EventLoitering
	v.LoiterCenterLat = v.Lat
	v.LoiterCenterLon = v.Lon
	v.LoiterUntil = time.Now().Add(time.Duration(durationS) * time.Second)
	v.SpeedKnots = 1.5 + rand.Float64()*1.0 // 1.5–2.5 knots
}

// applyDuplicateMMSI starts broadcasting targetMMSI from this vessel's position.
func applyDuplicateMMSI(v *VesselState, targetMMSI string) {
	v.ActiveEvent = EventDuplicateMMSI
	v.DuplicateMMSI = targetMMSI
}

// applyGeofenceCrossing steers the vessel toward the named zone's centroid.
func applyGeofenceCrossing(v *VesselState, zoneName string, zoneLat, zoneLon float64) {
	v.ActiveEvent = EventGeofenceCrossing
	v.TargetZone = zoneName
	v.HeadingDeg = bearingTo(v.Lat, v.Lon, zoneLat, zoneLon)
	v.SpeedKnots = 18 // full speed toward zone
}

// resetEvent cancels any active event and restores normal operation.
func resetEvent(v *VesselState) {
	v.ActiveEvent = EventNone
	v.DarkUntil = time.Time{}
	v.DuplicateMMSI = ""
	v.TargetZone = ""
	v.SpeedKnots = 8 + rand.Float64()*10
}

// tickEvent is called each update cycle to expire events that have finished.
func tickEvent(v *VesselState) {
	switch v.ActiveEvent {
	case EventDarkTransit:
		if time.Now().After(v.DarkUntil) {
			if v.DarkVariant == VariantJump {
				// Kinematic jump: teleport to physically implausible position
				v.Lat += 5.0 + rand.Float64()*5.0
				v.Lon += 5.0 + rand.Float64()*5.0
			}
			v.ActiveEvent = EventNone
			v.DarkUntil = time.Time{}
		}
	case EventLoitering:
		if time.Now().After(v.LoiterUntil) {
			resetEvent(v)
		} else {
			// Drift in tiny circle around loiter centre (radius ~50m)
			jitter := 0.0005
			v.Lat = v.LoiterCenterLat + (rand.Float64()-0.5)*jitter
			v.Lon = v.LoiterCenterLon + (rand.Float64()-0.5)*jitter
		}
	case EventDuplicateMMSI:
		// Persists until explicitly reset via /trigger/reset
	case EventGeofenceCrossing:
		// Persists until reset manually after demo
	}
}

// bearingTo returns the initial bearing (degrees true) from point 1 to point 2.
func bearingTo(lat1, lon1, lat2, lon2 float64) float64 {
	φ1 := deg2rad(lat1)
	φ2 := deg2rad(lat2)
	Δλ := deg2rad(lon2 - lon1)
	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)
	b := rad2deg(math.Atan2(y, x))
	if b < 0 {
		b += 360
	}
	return b
}
