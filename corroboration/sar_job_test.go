package corroboration

import (
	"context"
	"testing"
	"time"
)

// TestSARJobHit: alert position inside tile bbox → corroborate called once.
func TestSARJobHit(t *testing.T) {
	tile := SARTile{
		TileID: "TEST-TILE-001", FilePath: "fake_tile.tiff",
		MinLat: 10.0, MaxLat: 20.0,
		MinLon: 70.0, MaxLon: 80.0,
	}

	// Inject mock inference that always reports a ship found
	orig := sarInferFn
	sarInferFn = func(tile SARTile, ref AlertRef, modelPath string) (map[string]interface{}, error) {
		e := CorroborationEvidence{
			Source: "sar", TileID: tile.TileID,
			DetectionConfidence: 0.85,
			BoundingBoxPixels:   []int{100, 200, 50, 50},
			ModelVersion:        "yolov8n-sar-ssdd-v1",
			DetectedAt:          time.Now().UTC(),
		}
		return evidenceToMap(e), nil
	}
	defer func() { sarInferFn = orig }()

	alerts := make(chan AlertRef, 1)
	alerts <- AlertRef{AlertID: "alert-001", Lat: 15.0, Lon: 75.0}

	var calledID, calledSource string
	var called bool
	corroborate := func(id, source string, evidence map[string]interface{}) {
		called = true
		calledID = id
		calledSource = source
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go RunSARJob(ctx, alerts, corroborate, tile, "fake_model.onnx", 50*time.Millisecond)

	// Wait up to 500ms for corroborate to fire
	deadline := time.Now().Add(500 * time.Millisecond)
	for !called && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if !called {
		t.Fatal("expected corroborate to be called for in-bbox alert")
	}
	if calledID != "alert-001" {
		t.Errorf("expected alertID alert-001, got %s", calledID)
	}
	if calledSource != "sar" {
		t.Errorf("expected source sar, got %s", calledSource)
	}
}

// TestSARJobMiss: alert position outside tile bbox → corroborate must NOT be called.
func TestSARJobMiss(t *testing.T) {
	tile := SARTile{
		TileID: "TEST-TILE-001", FilePath: "fake_tile.tiff",
		MinLat: 10.0, MaxLat: 20.0,
		MinLon: 70.0, MaxLon: 80.0,
	}

	alerts := make(chan AlertRef, 1)
	alerts <- AlertRef{AlertID: "alert-002", Lat: 50.0, Lon: 10.0} // clearly outside bbox

	var called bool
	corroborate := func(id, source string, evidence map[string]interface{}) {
		called = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go RunSARJob(ctx, alerts, corroborate, tile, "fake_model.onnx", 50*time.Millisecond)
	<-ctx.Done()

	if called {
		t.Error("corroborate was called for out-of-bbox alert — should not have been")
	}
}
