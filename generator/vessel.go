package generator

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

const earthRadiusKm = 6371.0

// newVessel creates a vessel with a random starting position and cadence
// appropriate for the given zone type.
func newVessel(index int, zoneType string) *VesselState {
	r := rand.New(rand.NewSource(int64(index) * 7919))
	v := &VesselState{
		ID:         fmt.Sprintf("vessel-%04d", index),
		MMSI:       fmt.Sprintf("%09d", 200000000+index),
		Lat:        r.Float64()*160 - 80,  // -80..80
		Lon:        r.Float64()*360 - 180, // -180..180
		HeadingDeg: r.Float64() * 360,
		SpeedKnots: 8 + r.Float64()*10, // 8–18 knots
		ZoneType:   zoneType,
		CadenceMs:  cadenceForZone(zoneType, r),
	}
	return v
}

// cadenceForZone returns a randomised update interval in milliseconds
// matching the zone-tier AIS cadence from PRD §3.3.
func cadenceForZone(zoneType string, r *rand.Rand) int64 {
	switch zoneType {
	case "coastal":
		// 2–10 seconds
		return int64((2 + r.Intn(9)) * 1000)
	case "offshore":
		// 10–30 minutes
		return int64((10 + r.Intn(21)) * 60 * 1000)
	default: // open_ocean
		// 30–120 minutes
		return int64((30 + r.Intn(91)) * 60 * 1000)
	}
}

// advance moves the vessel along its great-circle heading by elapsed time.
// It does NOT check event state — that is handled by the generator loop.
func (v *VesselState) advance(elapsed time.Duration) {
	distanceKm := v.SpeedKnots * 1.852 * elapsed.Hours()
	v.Lat, v.Lon = haversineProject(v.Lat, v.Lon, v.HeadingDeg, distanceKm)
	// Clamp latitude
	if v.Lat > 90 {
		v.Lat = 90 - (v.Lat - 90)
		v.HeadingDeg = math.Mod(v.HeadingDeg+180, 360)
	}
	if v.Lat < -90 {
		v.Lat = -90 - (v.Lat + 90)
		v.HeadingDeg = math.Mod(v.HeadingDeg+180, 360)
	}
	// Wrap longitude
	for v.Lon > 180 {
		v.Lon -= 360
	}
	for v.Lon < -180 {
		v.Lon += 360
	}
}

// haversineProject returns the lat/lon reached by travelling distanceKm
// from (lat, lon) along the given bearing (degrees true).
func haversineProject(lat, lon, bearingDeg, distanceKm float64) (float64, float64) {
	δ := distanceKm / earthRadiusKm
	θ := deg2rad(bearingDeg)
	φ1 := deg2rad(lat)
	λ1 := deg2rad(lon)

	φ2 := math.Asin(math.Sin(φ1)*math.Cos(δ) +
		math.Cos(φ1)*math.Sin(δ)*math.Cos(θ))
	λ2 := λ1 + math.Atan2(
		math.Sin(θ)*math.Sin(δ)*math.Cos(φ1),
		math.Cos(δ)-math.Sin(φ1)*math.Sin(φ2),
	)
	return rad2deg(φ2), rad2deg(λ2)
}

// HaversineDistanceKm returns the great-circle distance in km between two points.
func HaversineDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	φ1, φ2 := deg2rad(lat1), deg2rad(lat2)
	Δφ := deg2rad(lat2 - lat1)
	Δλ := deg2rad(lon2 - lon1)
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func deg2rad(d float64) float64 { return d * math.Pi / 180 }
func rad2deg(r float64) float64 { return r * 180 / math.Pi }
