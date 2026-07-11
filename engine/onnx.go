package engine

import "math"

// DarkIntentModel wraps ONNX inference for dark ship trajectory prediction.
// ponytail: simulated for hackathon — swap in real onnxruntime CGO bindings when available.
// Real integration point: replace Predict() and ReconstructionError() bodies.
type DarkIntentModel struct{}

// NewDarkIntentModel creates a simulated model instance.
// In production: loads an ONNX model file via onnxruntime.
func NewDarkIntentModel() *DarkIntentModel {
	return &DarkIntentModel{}
}

// HeatmapCell represents a single probability cell in the dark ship heatmap.
type HeatmapCell struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Weight float64 `json:"weight"` // 0.0–1.0 probability
}

// Predict generates a probability heatmap of where a dark vessel is likely heading.
// Uses last known position, heading, speed, and proximity to sensitive zones.
func (m *DarkIntentModel) Predict(lastLat, lastLon, heading, speed float64, zones []Zone) []HeatmapCell {
	const (
		gridStep  = 0.05 // ~5.5 km per cell
		gridRange = 10   // cells in each direction along heading
		spreadDeg = 30.0 // cone half-angle in degrees
	)

	var cells []HeatmapCell
	headRad := heading * math.Pi / 180

	for i := 1; i <= gridRange; i++ {
		dist := float64(i) * gridStep
		// Base weight decays with distance
		baseWeight := 1.0 - (float64(i) / float64(gridRange+1))

		// Generate cells across the spread cone
		for angle := -spreadDeg; angle <= spreadDeg; angle += 10 {
			angleRad := (heading + angle) * math.Pi / 180
			cellLat := lastLat + dist*math.Cos(angleRad)
			cellLon := lastLon + dist*math.Sin(angleRad)

			// Boost weight for cells near sensitive zones (fishing grounds / MPAs)
			zoneBoost := 0.0
			for _, z := range zones {
				centLat, centLon := zoneCentroid(z)
				distToZone := haversineKm(cellLat, cellLon, centLat, centLon)
				if distToZone < z.BoundaryBufferKm*2 {
					zoneBoost = 0.3 * (1 - distToZone/(z.BoundaryBufferKm*2))
				}
			}

			// Angular weight: center of cone is highest
			angularWeight := 1.0 - math.Abs(angle)/spreadDeg*0.5

			weight := baseWeight*angularWeight + zoneBoost
			if weight > 1.0 {
				weight = 1.0
			}
			if weight < 0.05 {
				continue // skip near-zero cells
			}

			cells = append(cells, HeatmapCell{
				Lat:    cellLat,
				Lon:    cellLon,
				Weight: math.Round(weight*1000) / 1000,
			})
		}
		_ = headRad // used implicitly via heading
	}

	return cells
}

// ReconstructionError simulates the autoencoder reconstruction error for a trajectory.
// Returns 0.0–1.0 where higher = more anomalous.
func (m *DarkIntentModel) ReconstructionError(lat, lon, heading, speed float64) float64 {
	// ponytail: deterministic heuristic simulating ML output.
	// Anomalous = high speed + unusual heading (not aligned with common shipping lanes).
	// Real ONNX model would take a trajectory window as input tensor.
	speedFactor := 0.0
	if speed > 20 {
		speedFactor = (speed - 20) / 30 // 0.0 at 20kn, 1.0 at 50kn
	}

	// Heading anomaly: headings near 45°, 135°, 225°, 315° are "off-lane"
	headingNorm := math.Mod(heading, 90)
	if headingNorm > 45 {
		headingNorm = 90 - headingNorm
	}
	headingFactor := headingNorm / 45 // 0.0 for cardinal, 1.0 for diagonal

	err := speedFactor*0.6 + headingFactor*0.4
	if err > 1.0 {
		err = 1.0
	}
	return err
}

// zoneCentroid returns the rough centroid of a zone polygon.
func zoneCentroid(z Zone) (float64, float64) {
	if len(z.Polygon) == 0 {
		return 0, 0
	}
	var sumLat, sumLon float64
	for _, pt := range z.Polygon {
		sumLat += pt[0]
		sumLon += pt[1]
	}
	n := float64(len(z.Polygon))
	return sumLat / n, sumLon / n
}
