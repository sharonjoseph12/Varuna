package engine

import (
	"encoding/json"
	"fmt"
)

// HeatmapGeoJSON generates a GeoJSON FeatureCollection from heatmap cells.
// Each cell becomes a small square polygon with a weight property.
func HeatmapGeoJSON(cells []HeatmapCell) json.RawMessage {
	const halfCell = 0.025 // half of 0.05° ≈ 2.75km

	type geometry struct {
		Type        string        `json:"type"`
		Coordinates [][][2]float64 `json:"coordinates"`
	}
	type properties struct {
		Weight float64 `json:"weight"`
		Type   string  `json:"type"`
	}
	type feature struct {
		Type       string     `json:"type"`
		Geometry   geometry   `json:"geometry"`
		Properties properties `json:"properties"`
	}
	type featureCollection struct {
		Type     string    `json:"type"`
		Features []feature `json:"features"`
	}

	fc := featureCollection{
		Type:     "FeatureCollection",
		Features: make([]feature, 0, len(cells)),
	}

	for _, c := range cells {
		ring := [5][2]float64{
			{c.Lon - halfCell, c.Lat - halfCell},
			{c.Lon + halfCell, c.Lat - halfCell},
			{c.Lon + halfCell, c.Lat + halfCell},
			{c.Lon - halfCell, c.Lat + halfCell},
			{c.Lon - halfCell, c.Lat - halfCell}, // close ring
		}
		f := feature{
			Type: "Feature",
			Geometry: geometry{
				Type:        "Polygon",
				Coordinates: [][][2]float64{ring[:]},
			},
			Properties: properties{
				Weight: c.Weight,
				Type:   "dark_ship_probability",
			},
		}
		fc.Features = append(fc.Features, f)
	}

	data, err := json.Marshal(fc)
	if err != nil {
		// ponytail: should never happen with known types
		return json.RawMessage(fmt.Sprintf(`{"type":"FeatureCollection","features":[],"error":"%s"}`, err))
	}
	return data
}

// HeatmapPointsGeoJSON generates a simpler point-based GeoJSON for MapLibre's native heatmap layer.
// This is for the risk heatmap (alert clustering), not the dark ship probability heatmap.
func HeatmapPointsGeoJSON(alerts []Alert) json.RawMessage {
	type geometry struct {
		Type        string     `json:"type"`
		Coordinates [2]float64 `json:"coordinates"`
	}
	type properties struct {
		Weight float64 `json:"weight"`
		Type   string  `json:"type"`
	}
	type feature struct {
		Type       string     `json:"type"`
		Geometry   geometry   `json:"geometry"`
		Properties properties `json:"properties"`
	}
	type featureCollection struct {
		Type     string    `json:"type"`
		Features []feature `json:"features"`
	}

	fc := featureCollection{
		Type:     "FeatureCollection",
		Features: make([]feature, 0, len(alerts)),
	}

	for _, a := range alerts {
		weight := a.Confidence
		fc.Features = append(fc.Features, feature{
			Type: "Feature",
			Geometry: geometry{
				Type:        "Point",
				Coordinates: [2]float64{a.Position.Lon, a.Position.Lat},
			},
			Properties: properties{
				Weight: weight,
				Type:   a.Type,
			},
		})
	}

	data, _ := json.Marshal(fc)
	return data
}
