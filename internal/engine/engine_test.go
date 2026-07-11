package engine

import (
	"testing"
	"time"
)

// --- Geofence Tests ---

func TestGeofenceSingleAlertPerCrossing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StorePath = "" // no persistence in tests
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 1000)
	go eng.Ingest(in)

	// Start draining positions
	go func() { for range eng.Positions() {} }()

	ts := time.Now().UnixMilli()

	// Vessel starts outside Gulf of Mannar (south of 8.5 lat)
	// then crosses well inside (past hysteresis margin)
	for i := 0; i < 5; i++ {
		in <- AISMessage{
			VesselID: "V-TEST-GEO", MMSI: "100000001",
			Lat: 8.3 + float64(i)*0.05, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 12,
			TimestampMs: ts + int64(i)*1000,
		}
	}
	// Now inside: 8.3 + 4*0.05 = 8.5 (boundary), need to go further past hysteresis
	for i := 5; i < 10; i++ {
		in <- AISMessage{
			VesselID: "V-TEST-GEO", MMSI: "100000001",
			Lat: 8.5 + float64(i-5)*0.1, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 12,
			TimestampMs: ts + int64(i)*1000,
		}
	}

	close(in)
	time.Sleep(200 * time.Millisecond) // let processing finish

	// Count geofence_breach alerts
	alertCount := 0
	done := false
	for !done {
		select {
		case alert := <-eng.Alerts():
			if alert.Type == "geofence_breach" && alert.VesselID == "V-TEST-GEO" {
				alertCount++
				// Verify reasoning trace
				if len(alert.ReasoningTrace.InputsEvaluated) == 0 {
					t.Error("reasoning trace inputs_evaluated is empty")
				}
				if alert.ReasoningTrace.EngineVersion == "" {
					t.Error("reasoning trace engine_version is empty")
				}
				if len(alert.ReasoningTrace.ModalitiesAvailable) == 0 {
					t.Error("reasoning trace modalities_available is empty")
				}
			}
		default:
			done = true
		}
	}

	if alertCount != 1 {
		t.Errorf("expected exactly 1 geofence_breach alert, got %d", alertCount)
	}
}

func TestGeofenceHysteresisNoAlertOnJitter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StorePath = ""
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 1000)
	go eng.Ingest(in)
	go func() { for range eng.Positions() {} }()

	ts := time.Now().UnixMilli()
	boundary := 8.5 // southern edge of Gulf of Mannar

	// Jitter back and forth within hysteresis margin (0.001 deg ≈ 100m)
	for i := 0; i < 20; i++ {
		offset := 0.0003
		if i%2 == 0 {
			offset = -0.0003
		}
		in <- AISMessage{
			VesselID: "V-TEST-JITTER", MMSI: "100000002",
			Lat: boundary + offset, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 2,
			TimestampMs: ts + int64(i)*1000,
		}
	}

	close(in)
	time.Sleep(200 * time.Millisecond)

	// Should be zero geofence alerts
	alertCount := 0
	done := false
	for !done {
		select {
		case alert := <-eng.Alerts():
			if alert.Type == "geofence_breach" && alert.VesselID == "V-TEST-JITTER" {
				alertCount++
			}
		default:
			done = true
		}
	}

	if alertCount != 0 {
		t.Errorf("expected 0 geofence_breach alerts from jitter, got %d", alertCount)
	}
}

// --- Identity Conflict Tests ---

func TestIdentityConflictFiresOnImpossiblePositions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StorePath = ""
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 1000)
	go eng.Ingest(in)
	go func() { for range eng.Positions() {} }()

	ts := time.Now().UnixMilli()
	spoofedMMSI := "SPOOF-TEST-001"

	// Vessel A at position 1
	in <- AISMessage{
		VesselID: "V-SPOOF-A", MMSI: spoofedMMSI,
		Lat: 10.0, Lon: 72.0,
		HeadingDeg: 90, SpeedKnots: 10,
		TimestampMs: ts,
	}

	time.Sleep(50 * time.Millisecond) // ensure first message processes

	// Vessel B with same MMSI, 300km away, 30 seconds later — impossible
	in <- AISMessage{
		VesselID: "V-SPOOF-B", MMSI: spoofedMMSI,
		Lat: 12.0, Lon: 74.0,
		HeadingDeg: 270, SpeedKnots: 10,
		TimestampMs: ts + 30000,
	}

	close(in)
	time.Sleep(200 * time.Millisecond)

	found := false
	done := false
	for !done {
		select {
		case alert := <-eng.Alerts():
			if alert.Type == "identity_conflict" {
				found = true
				// Verify evidence
				if alert.Evidence["mmsi"] != spoofedMMSI {
					t.Error("identity_conflict alert missing mmsi in evidence")
				}
				if len(alert.ReasoningTrace.InputsEvaluated) == 0 {
					t.Error("reasoning trace inputs_evaluated is empty")
				}
			}
		default:
			done = true
		}
	}

	if !found {
		t.Error("expected identity_conflict alert, got none")
	}
}

func TestIdentityConflictSilentOnPlausiblePositions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StorePath = ""
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 1000)
	go eng.Ingest(in)
	go func() { for range eng.Positions() {} }()

	ts := time.Now().UnixMilli()
	mmsi := "PLAUS-TEST-001"

	// Two vessels with same MMSI but plausible positions
	in <- AISMessage{
		VesselID: "V-PLAUS-A", MMSI: mmsi,
		Lat: 10.0, Lon: 72.0,
		HeadingDeg: 90, SpeedKnots: 10,
		TimestampMs: ts,
	}

	time.Sleep(50 * time.Millisecond)

	// Very close, 60s later — plausible at any speed
	in <- AISMessage{
		VesselID: "V-PLAUS-B", MMSI: mmsi,
		Lat: 10.001, Lon: 72.001,
		HeadingDeg: 90, SpeedKnots: 10,
		TimestampMs: ts + 60000,
	}

	close(in)
	time.Sleep(200 * time.Millisecond)

	done := false
	for !done {
		select {
		case alert := <-eng.Alerts():
			if alert.Type == "identity_conflict" {
				t.Error("unexpected identity_conflict alert on plausible positions")
			}
		default:
			done = true
		}
	}
}

// --- Reasoning Trace Tests ---

func TestEveryAlertHasReasoningTrace(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StorePath = ""
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 1000)
	go eng.Ingest(in)
	go func() { for range eng.Positions() {} }()

	ts := time.Now().UnixMilli()

	// Generate a geofence crossing
	for i := 0; i < 10; i++ {
		in <- AISMessage{
			VesselID: "V-TRACE-TEST", MMSI: "TRACE001",
			Lat: 8.3 + float64(i)*0.05, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 12,
			TimestampMs: ts + int64(i)*1000,
		}
	}
	for i := 10; i < 15; i++ {
		in <- AISMessage{
			VesselID: "V-TRACE-TEST", MMSI: "TRACE001",
			Lat: 8.8 + float64(i-10)*0.1, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 12,
			TimestampMs: ts + int64(i)*1000,
		}
	}

	close(in)
	time.Sleep(200 * time.Millisecond)

	done := false
	for !done {
		select {
		case alert := <-eng.Alerts():
			if len(alert.ReasoningTrace.InputsEvaluated) == 0 {
				t.Errorf("alert %s (type=%s) has empty inputs_evaluated", alert.AlertID, alert.Type)
			}
			if alert.ReasoningTrace.EngineVersion == "" {
				t.Errorf("alert %s (type=%s) has empty engine_version", alert.AlertID, alert.Type)
			}
			if len(alert.ReasoningTrace.ModalitiesAvailable) == 0 {
				t.Errorf("alert %s (type=%s) has empty modalities_available", alert.AlertID, alert.Type)
			}
		default:
			done = true
		}
	}
}

// --- Corroboration Test ---

func TestCorroborateUpgradesAlertStatus(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StorePath = ""
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 1000)
	go eng.Ingest(in)
	go func() { for range eng.Positions() {} }()

	ts := time.Now().UnixMilli()

	// Generate a geofence crossing to produce an alert
	for i := 0; i < 15; i++ {
		in <- AISMessage{
			VesselID: "V-CORR-TEST", MMSI: "CORR001",
			Lat: 8.3 + float64(i)*0.05, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 12,
			TimestampMs: ts + int64(i)*1000,
		}
	}

	close(in)
	time.Sleep(200 * time.Millisecond)

	// Find the alert
	var alertID string
	done := false
	for !done {
		select {
		case alert := <-eng.Alerts():
			if alert.Type == "geofence_breach" {
				alertID = alert.AlertID
			}
		default:
			done = true
		}
	}

	if alertID == "" {
		t.Fatal("no geofence_breach alert produced for corroboration test")
	}

	// Corroborate it
	eng.Corroborate(alertID, "SAR", map[string]interface{}{"tile": "S1A_GRD_test"})

	// Verify
	alert, ok := eng.GetAlert(alertID)
	if !ok {
		t.Fatal("alert not found after corroboration")
	}
	if alert.Corroboration.Status != "corroborated" {
		t.Errorf("expected status 'corroborated', got '%s'", alert.Corroboration.Status)
	}
	if alert.Corroboration.Source == nil || *alert.Corroboration.Source != "SAR" {
		t.Error("expected source 'SAR'")
	}
}

// --- PIP Tests ---

func TestPointInPolygon(t *testing.T) {
	// Simple square: (0,0), (0,10), (10,10), (10,0)
	poly := [][2]float64{{0, 0}, {0, 10}, {10, 10}, {10, 0}}

	tests := []struct {
		lat, lon float64
		want     bool
	}{
		{5, 5, true},    // center
		{0.1, 0.1, true}, // just inside corner
		{-1, 5, false},  // outside
		{11, 5, false},  // outside
		{5, 11, false},  // outside
	}

	for _, tt := range tests {
		got := PointInPolygon(tt.lat, tt.lon, poly)
		if got != tt.want {
			t.Errorf("PointInPolygon(%f, %f) = %v, want %v", tt.lat, tt.lon, got, tt.want)
		}
	}
}

// --- Throughput Benchmark ---

func BenchmarkThroughput(b *testing.B) {
	cfg := DefaultConfig()
	cfg.StorePath = "" // no persistence during benchmark
	cfg.AlertChannelSize = 100000
	cfg.PositionChannelSize = 100000
	zones := DefaultZones()
	eng := NewEngine(cfg, zones)
	defer eng.Close()

	in := make(chan AISMessage, 200000)

	// Drain outputs
	go func() { for range eng.Alerts() {} }()
	go func() { for range eng.Positions() {} }()
	go eng.Ingest(in)

	ts := int64(0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ts++
		in <- AISMessage{
			VesselID:    "V-BENCH",
			MMSI:        "BENCH001",
			Lat:         9.0 + float64(i%100)*0.001,
			Lon:         78.5 + float64(i%100)*0.001,
			HeadingDeg:  45,
			SpeedKnots:  12,
			TimestampMs: ts,
		}
	}

	close(in)
	time.Sleep(100 * time.Millisecond)
}
