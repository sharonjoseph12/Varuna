package corroboration

import (
	"context"
	"testing"
	"time"
)

// TestVIIRSStub: confirms corroborate is called with stub:true evidence.
func TestVIIRSStub(t *testing.T) {
	alerts := make(chan AlertRef, 1)
	alerts <- AlertRef{AlertID: "alert-viirs-001", Lat: 12.0, Lon: 76.0}

	var called bool
	var gotStub bool
	corroborate := func(id, source string, evidence map[string]interface{}) {
		called = true
		if stub, ok := evidence["stub"].(bool); ok && stub {
			gotStub = true
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go RunVIIRSStub(ctx, alerts, corroborate, 50*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for !called && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if !called {
		t.Fatal("expected corroborate to be called by VIIRS stub")
	}
	if !gotStub {
		t.Error("expected evidence to contain stub:true")
	}
}
