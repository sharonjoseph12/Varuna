// Package corroboration implements offline satellite corroboration jobs
// that run on slow background tickers, completely isolated from the hot
// ingestion path. Jobs call CorroborateFunc when a match is found.
package corroboration

import "time"

// SARTile describes a pre-downloaded Sentinel-1 GRD tile.
type SARTile struct {
	TileID   string  `json:"tile_id"`
	FilePath string  `json:"file_path"`
	MinLat   float64 `json:"min_lat"`
	MaxLat   float64 `json:"max_lat"`
	MinLon   float64 `json:"min_lon"`
	MaxLon   float64 `json:"max_lon"`
}

// Contains returns true if (lat, lon) falls within this tile's bounding box.
func (t SARTile) Contains(lat, lon float64) bool {
	return lat >= t.MinLat && lat <= t.MaxLat && lon >= t.MinLon && lon <= t.MaxLon
}

// CorroborationEvidence is the evidence payload passed to CorroborateFunc.
type CorroborationEvidence struct {
	Source              string    `json:"source"`                        // "sar" | "viirs"
	TileID              string    `json:"tile_id,omitempty"`
	DetectionConfidence float64   `json:"detection_confidence"`
	BoundingBoxPixels   []int     `json:"bounding_box_pixels,omitempty"` // [x, y, w, h]
	ModelVersion        string    `json:"model_version,omitempty"`
	Stub                bool      `json:"stub,omitempty"`
	DetectedAt          time.Time `json:"detected_at"`
}

// AlertRef carries the minimum fields a corroboration job needs to match an alert.
type AlertRef struct {
	AlertID string
	Lat     float64
	Lon     float64
}

// CorroborateFunc is the callback a job calls when it finds a match.
// Signature matches engine.Corroborate(alertID, source, evidence).
type CorroborateFunc func(alertID string, source string, evidence map[string]interface{})
