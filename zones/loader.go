// Package zones loads and exposes the shared zone GeoJSON used by both
// the engine (geofence/absence logic) and the frontend (map rendering).
package zones

import (
	"encoding/json"
	"fmt"
	"os"
)

// Zone represents a single maritime enforcement zone.
type Zone struct {
	Name              string      `json:"name"`
	ZoneType          string      `json:"zone_type"` // "coastal" | "offshore" | "open_ocean"
	SilenceToleranceS int         `json:"silence_tolerance_s"`
	BoundaryBufferKm  float64     `json:"boundary_buffer_km"`
	Coordinates       [][][2]float64 // [ring][point][lon,lat]
	CentroidLat       float64
	CentroidLon       float64
}

type geoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []geoJSONFeature `json:"features"`
}

type geoJSONFeature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   geoJSONGeometry        `json:"geometry"`
}

type geoJSONGeometry struct {
	Type        string           `json:"type"`
	Coordinates [][][][2]float64 `json:"coordinates"` // Polygon = [ring][point]
}

// LoadZones parses the GeoJSON file at path and returns typed Zone values.
func LoadZones(path string) ([]Zone, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("zones: read %s: %w", path, err)
	}
	var fc geoJSONFeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("zones: parse GeoJSON: %w", err)
	}
	zones := make([]Zone, 0, len(fc.Features))
	for _, f := range fc.Features {
		if f.Geometry.Type != "Polygon" {
			continue
		}
		z := Zone{}
		if v, ok := f.Properties["name"].(string); ok {
			z.Name = v
		}
		if v, ok := f.Properties["zone_type"].(string); ok {
			z.ZoneType = v
		}
		if v, ok := f.Properties["silence_tolerance_s"].(float64); ok {
			z.SilenceToleranceS = int(v)
		}
		if v, ok := f.Properties["boundary_buffer_km"].(float64); ok {
			z.BoundaryBufferKm = v
		}
		if len(f.Geometry.Coordinates) > 0 && len(f.Geometry.Coordinates[0]) > 0 {
			ring := f.Geometry.Coordinates[0][0]
			coords := make([][2]float64, len(ring))
			copy(coords, ring)
			z.Coordinates = [][][2]float64{coords}
			z.CentroidLon, z.CentroidLat = polygonCentroid(ring)
		}
		zones = append(zones, z)
	}
	return zones, nil
}

// polygonCentroid returns the arithmetic mean of a ring's vertices (lon, lat).
func polygonCentroid(ring [][2]float64) (lon, lat float64) {
	n := len(ring)
	if n == 0 {
		return 0, 0
	}
	// Skip closing vertex if it duplicates the first
	end := n
	if n > 1 && ring[0][0] == ring[n-1][0] && ring[0][1] == ring[n-1][1] {
		end = n - 1
	}
	for i := 0; i < end; i++ {
		lon += ring[i][0]
		lat += ring[i][1]
	}
	return lon / float64(end), lat / float64(end)
}

// ZoneLookup returns a lookup function suitable for TriggerServer.
// It maps zone name → (centroid lat, centroid lon, found).
func ZoneLookup(zones []Zone) func(string) (float64, float64, bool) {
	index := make(map[string]Zone, len(zones))
	for _, z := range zones {
		index[z.Name] = z
	}
	return func(name string) (float64, float64, bool) {
		z, ok := index[name]
		return z.CentroidLat, z.CentroidLon, ok
	}
}
