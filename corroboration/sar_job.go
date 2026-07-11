package corroboration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// sarInferFn is the inference function used by RunSARJob.
// Tests swap this var to inject a mock without changing production code.
// ponytail: var injection — no interface needed for a single implementation.
var sarInferFn = defaultSARInfer

// sarInferResult is the JSON shape printed to stdout by sar_infer.py.
type sarInferResult struct {
	Found        bool    `json:"found"`
	Confidence   float64 `json:"confidence"`
	BBox         []int   `json:"bbox"`          // [x, y, w, h] in pixels
	ModelVersion string  `json:"model_version"`
}

// RunSARJob is a blocking goroutine that polls pending alerts every tickInterval,
// checks each alert's position against the tile bbox, runs SAR inference when
// there is a match, and calls corroborate on a successful detection.
//
// ISOLATION: Never call on the ingestion hot path. Always launch as:
//
//	go RunSARJob(ctx, alerts, corroborate, tile, modelPath, tickInterval)
func RunSARJob(
	ctx context.Context,
	alerts <-chan AlertRef,
	corroborate CorroborateFunc,
	tile SARTile,
	modelPath string,
	tickInterval time.Duration,
) {
	if tickInterval <= 0 {
		tickInterval = 30 * time.Second
	}
	pending := make(map[string]AlertRef)
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ref, ok := <-alerts:
			if !ok {
				return
			}
			pending[ref.AlertID] = ref
		case <-ticker.C:
			for id, ref := range pending {
				if !tile.Contains(ref.Lat, ref.Lon) {
					continue
				}
				evidence, err := sarInferFn(tile, ref, modelPath)
				if err != nil {
					log.Printf("[sar] inference error for alert %s: %v", id, err)
					continue
				}
				if evidence == nil {
					continue // no ship detected in tile
				}
				corroborate(id, "sar", evidence)
				delete(pending, id)
			}
		}
	}
}

// defaultSARInfer execs data/models/sar_infer.py and returns evidence if a ship
// is found, or (nil, nil) if not found.
func defaultSARInfer(tile SARTile, ref AlertRef, modelPath string) (map[string]interface{}, error) {
	// ponytail: Python subprocess boundary keeps CGo out of the hot path.
	// Upgrade to onnxruntime_go native binding if cross-process overhead matters.
	cmd := exec.Command("python", modelPath,
		"--tile", tile.FilePath,
		"--lat", fmt.Sprintf("%f", ref.Lat),
		"--lon", fmt.Sprintf("%f", ref.Lon),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var result sarInferResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	if !result.Found {
		return nil, nil
	}
	e := CorroborationEvidence{
		Source:              "sar",
		TileID:              tile.TileID,
		DetectionConfidence: result.Confidence,
		BoundingBoxPixels:   result.BBox,
		ModelVersion:        result.ModelVersion,
		DetectedAt:          time.Now().UTC(),
	}
	return evidenceToMap(e), nil
}

func evidenceToMap(e CorroborationEvidence) map[string]interface{} {
	b, _ := json.Marshal(e)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	return m
}
