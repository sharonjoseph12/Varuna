package generator

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestGeneratorThroughput confirms ≥ 50,000 msg/sec for 10 seconds.
func TestGeneratorThroughput(t *testing.T) {
	cfg := Config{VesselCount: 200, OutputBuffer: 50000, TriggerPort: 0}
	g := NewGenerator(cfg)
	out := make(chan AISMessage, cfg.OutputBuffer)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	go g.Run(ctx, out)

	// Drain for 10 seconds
	start := time.Now()
	deadline := start.Add(10 * time.Second)
	var count int64
	for time.Now().Before(deadline) {
		select {
		case <-out:
			count++
		case <-ctx.Done():
			t.Fatal("context cancelled before measurement window")
		}
	}

	elapsed := time.Since(start).Seconds()
	rate := float64(count) / elapsed
	t.Logf("throughput: %.0f msg/sec over %.1fs (%d messages)", rate, elapsed, count)
	if rate < 50000 {
		t.Errorf("throughput %.0f msg/sec is below 50,000 msg/sec target", rate)
	}
}

// TestDarkTransitEvent confirms no messages are emitted during the silence window.
func TestDarkTransitEvent(t *testing.T) {
	cfg := Config{VesselCount: 5, OutputBuffer: 1000, TriggerPort: 18081}
	g := NewGenerator(cfg)
	out := make(chan AISMessage, cfg.OutputBuffer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx, out)

	// Start trigger server
	ts := NewTriggerServer(g, func(string) (float64, float64, bool) { return 0, 0, false })
	go ts.ListenAndServe(ctx, cfg.TriggerPort)
	time.Sleep(100 * time.Millisecond) // let server start

	// Pick first vessel
	ids := g.AllVesselIDs()
	targetID := ids[0]

	// Trigger dark-transit with 2-second silence
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:%d/trigger/dark-transit?vessel=%s&silence_s=2&variant=plausible", cfg.TriggerPort, targetID),
		"application/json", nil)
	if err != nil {
		t.Fatalf("trigger request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}

	// Drain messages for 1.5 seconds; target vessel should not appear
	deadline := time.Now().Add(1500 * time.Millisecond)
	var darkCount int
	for time.Now().Before(deadline) {
		select {
		case msg := <-out:
			if msg.VesselID == targetID {
				darkCount++
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if darkCount > 0 {
		t.Errorf("expected 0 messages from %s during dark window, got %d", targetID, darkCount)
	}

	// After silence expires (2s), vessel should reappear
	time.Sleep(700 * time.Millisecond)
	deadline = time.Now().Add(500 * time.Millisecond)
	var reappearedCount int
	for time.Now().Before(deadline) {
		select {
		case msg := <-out:
			if msg.VesselID == targetID {
				reappearedCount++
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if reappearedCount == 0 {
		t.Errorf("vessel %s did not reappear after dark-transit window", targetID)
	}
}

// TestDuplicateMMSI confirms two vessels emit the same MMSI simultaneously.
func TestDuplicateMMSI(t *testing.T) {
	cfg := Config{VesselCount: 5, OutputBuffer: 1000, TriggerPort: 18082}
	g := NewGenerator(cfg)
	out := make(chan AISMessage, cfg.OutputBuffer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx, out)

	ts := NewTriggerServer(g, func(string) (float64, float64, bool) { return 0, 0, false })
	go ts.ListenAndServe(ctx, cfg.TriggerPort)
	time.Sleep(100 * time.Millisecond)

	ids := g.AllVesselIDs()
	spoofVessel := ids[1]
	targetMMSI := "123456789"

	resp, err := http.Post(
		fmt.Sprintf("http://localhost:%d/trigger/duplicate-mmsi?vessel=%s&target_mmsi=%s", cfg.TriggerPort, spoofVessel, targetMMSI),
		"application/json", nil)
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	resp.Body.Close()

	// Collect messages for 500ms; spoof vessel should broadcast targetMMSI
	deadline := time.Now().Add(500 * time.Millisecond)
	var spoofed bool
	for time.Now().Before(deadline) {
		select {
		case msg := <-out:
			if msg.VesselID == spoofVessel && msg.MMSI == targetMMSI {
				spoofed = true
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if !spoofed {
		t.Errorf("vessel %s did not broadcast duplicate MMSI %s", spoofVessel, targetMMSI)
	}
}
