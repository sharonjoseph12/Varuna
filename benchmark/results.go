// Package benchmark implements the Varuna benchmark harness.
// It drives the generator at sustained ≥50k msg/sec for 120 seconds and
// writes a benchmark-results.json at the repo root with all required metrics.
package benchmark

import (
	"encoding/json"
	"os"
	"time"
)

// BenchmarkResult holds all metrics captured during a benchmark run.
type BenchmarkResult struct {
	HardwareCPU      string    `json:"hardware_cpu"`
	HardwareCores    int       `json:"hardware_cores"`
	HardwareRAMGB    float64   `json:"hardware_ram_gb"`
	MessageSizeBytes int       `json:"message_size_bytes"`
	ZoneCount        int       `json:"zone_count"`
	WSClientCount    int       `json:"ws_client_count"`
	DurationS        int       `json:"duration_s"`
	ThroughputMsgSec float64   `json:"throughput_msg_sec"`
	P50LatencyMs     float64   `json:"p50_latency_ms"`
	P95LatencyMs     float64   `json:"p95_latency_ms"`
	P99LatencyMs     float64   `json:"p99_latency_ms"`
	AlertsExpected   int       `json:"alerts_expected"`
	AlertsFired      int       `json:"alerts_fired"`
	FalsePositives   int       `json:"false_positives"`
	DroppedMessages  int       `json:"dropped_messages"`
	RunAt            time.Time `json:"run_at"`
}

// BenchmarkConfig configures a benchmark run.
type BenchmarkConfig struct {
	VesselCount   int
	DurationS     int
	WSClientCount int
	ZoneCount     int
	EngineAddr    string // WebSocket address of the engine (e.g. "ws://localhost:8080/ws")
	TriggerAddr   string // HTTP address of generator trigger server
}

// DefaultBenchmarkConfig returns the benchmark config matching PRD §5.
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		VesselCount:   200,
		DurationS:     120,
		WSClientCount: 5,
		ZoneCount:     8,
		EngineAddr:    "ws://localhost:8080/ws",
		TriggerAddr:   "http://localhost:8081",
	}
}

// WriteResults marshals r to JSON and writes it to outputPath.
func WriteResults(r BenchmarkResult, outputPath string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
