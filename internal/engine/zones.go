package engine

// DefaultZones returns 8 hardcoded zones covering Indian Ocean areas.
// ponytail: hardcoded GeoJSON-style polygons, no zone-authoring UI
func DefaultZones() []Zone {
	return []Zone{
		{
			ID: "gulf-of-mannar", Name: "Gulf of Mannar MPA", Type: "coastal",
			Polygon: [][2]float64{
				{8.5, 78.0}, {9.5, 78.0}, {9.5, 79.5}, {8.5, 79.5},
			},
			HysteresisMarginDeg: 0.001, SilenceToleranceSec: 120,
			BoundaryBufferKm: 5,
		},
		{
			ID: "palk-bay", Name: "Palk Bay Restricted Zone", Type: "coastal",
			Polygon: [][2]float64{
				{9.5, 79.0}, {10.5, 79.0}, {10.5, 80.0}, {9.5, 80.0},
			},
			HysteresisMarginDeg: 0.001, SilenceToleranceSec: 120,
			BoundaryBufferKm: 5,
		},
		{
			ID: "lakshadweep", Name: "Lakshadweep Islands EEZ", Type: "offshore",
			Polygon: [][2]float64{
				{10.0, 71.0}, {12.5, 71.0}, {12.5, 74.0}, {10.0, 74.0},
			},
			HysteresisMarginDeg: 0.002, SilenceToleranceSec: 2700, // 45 min
			BoundaryBufferKm: 10,
		},
		{
			ID: "andaman-north", Name: "Andaman North Marine Reserve", Type: "offshore",
			Polygon: [][2]float64{
				{12.0, 92.0}, {14.0, 92.0}, {14.0, 94.0}, {12.0, 94.0},
			},
			HysteresisMarginDeg: 0.002, SilenceToleranceSec: 2700,
			BoundaryBufferKm: 10,
		},
		{
			ID: "andaman-south", Name: "Andaman South Protected Area", Type: "offshore",
			Polygon: [][2]float64{
				{10.0, 92.0}, {12.0, 92.0}, {12.0, 94.0}, {10.0, 94.0},
			},
			HysteresisMarginDeg: 0.002, SilenceToleranceSec: 2700,
			BoundaryBufferKm: 10,
		},
		{
			ID: "kochi-anchorage", Name: "Kochi Port Anchorage Zone", Type: "coastal",
			Polygon: [][2]float64{
				{9.8, 75.8}, {10.2, 75.8}, {10.2, 76.4}, {9.8, 76.4},
			},
			HysteresisMarginDeg: 0.001, SilenceToleranceSec: 120,
			BoundaryBufferKm: 3,
		},
		{
			ID: "arabian-deep", Name: "Arabian Sea Deep Water Zone", Type: "open_ocean",
			Polygon: [][2]float64{
				{8.0, 65.0}, {15.0, 65.0}, {15.0, 72.0}, {8.0, 72.0},
			},
			HysteresisMarginDeg: 0.005, SilenceToleranceSec: 21600, // 6 hours — open ocean is conservative
			BoundaryBufferKm: 20,
		},
		{
			ID: "sri-lanka-eez", Name: "Sri Lanka EEZ Northern Boundary", Type: "offshore",
			Polygon: [][2]float64{
				{8.0, 79.5}, {10.0, 79.5}, {10.0, 82.0}, {8.0, 82.0},
			},
			HysteresisMarginDeg: 0.002, SilenceToleranceSec: 2700,
			BoundaryBufferKm: 10,
		},
	}
}
