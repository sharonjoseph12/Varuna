// cmd/benchmark is the Varuna benchmark runner.
// Run: go run ./cmd/benchmark [-duration 120] [-vessels 200] [-clients 5]
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sharonjoseph12/Varuna/benchmark"
)

func main() {
	duration := flag.Int("duration", 120, "benchmark duration in seconds")
	vessels := flag.Int("vessels", 200, "number of simulated vessels")
	clients := flag.Int("clients", 5, "number of WebSocket clients")
	output := flag.String("output", "benchmark-results.json", "output file path")
	flag.Parse()

	cfg := benchmark.DefaultBenchmarkConfig()
	cfg.DurationS = *duration
	cfg.VesselCount = *vessels
	cfg.WSClientCount = *clients

	result, err := benchmark.RunBenchmark(cfg)
	if err != nil {
		log.Fatalf("benchmark failed: %v", err)
	}

	if err := benchmark.WriteResults(result, *output); err != nil {
		log.Fatalf("write results: %v", err)
	}

	fmt.Printf("\n=== Varuna Benchmark Results ===\n")
	fmt.Printf("Throughput:   %.0f msg/sec\n", result.ThroughputMsgSec)
	fmt.Printf("p50 latency:  %.2f ms\n", result.P50LatencyMs)
	fmt.Printf("p95 latency:  %.2f ms\n", result.P95LatencyMs)
	fmt.Printf("p99 latency:  %.2f ms\n", result.P99LatencyMs)
	fmt.Printf("Alerts (expected/fired): %d/%d\n", result.AlertsExpected, result.AlertsFired)
	fmt.Printf("False positives: %d\n", result.FalsePositives)
	fmt.Printf("Hardware: %s, cores=%d\n", result.HardwareCPU, result.HardwareCores)
	fmt.Printf("Results written to: %s\n", *output)
}
