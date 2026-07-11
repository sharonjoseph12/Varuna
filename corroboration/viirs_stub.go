package corroboration

import (
	"context"
	"log"
	"time"
)

// RunVIIRSStub is a corroboration job that demonstrates the fusion-ready
// interface using a mocked VIIRS night-lights detection.
//
// THIS IS A STUB — it does not run a real nighttime-detection model.
// Its only purpose is to prove the alert object accepts a second independent
// corroboration modality without any change to the core engines.
//
// In production, replace this with a real VIIRS nighttime-detection pipeline
// (e.g. NASA Black Marble VNP46A1 daily composite, ~3h processing latency).
//
// Never call on the ingestion hot path — always launch as:
//
//	go RunVIIRSStub(ctx, alerts, corroborate, tickInterval)
func RunVIIRSStub(
	ctx context.Context,
	alerts <-chan AlertRef,
	corroborate CorroborateFunc,
	tickInterval time.Duration,
) {
	if tickInterval <= 0 {
		tickInterval = 60 * time.Second
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
			// ponytail: stub — fire on first pending alert each tick to show the interface
			for id := range pending {
				evidence := CorroborationEvidence{
					Source:              "viirs",
					TileID:              "VNP46A1.STUB.2026-07-11",
					DetectionConfidence: 0.71,
					Stub:                true, // IMPORTANT: always set; do not remove
					DetectedAt:          time.Now().UTC(),
				}
				log.Printf("[viirs-stub] upgrading alert %s (stub corroboration)", id)
				corroborate(id, "viirs", evidenceToMap(evidence))
				delete(pending, id)
				break // one per tick is enough for the demo
			}
		}
	}
}
