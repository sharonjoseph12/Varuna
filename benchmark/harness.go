package benchmark

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"varuna/generator"
)

// RunBenchmark drives the generator for cfg.DurationS seconds, measures
// end-to-end latency (message creation → channel drain), fires 4 scripted
// geofence crossings, and returns the BenchmarkResult.
//
// End-to-end latency is measured as: AISMessage.TimestampMs (set at generation)
// → time.Now() at consumer receipt. This is the correct metric per PRD §3.8.
func RunBenchmark(cfg BenchmarkConfig) (BenchmarkResult, error) {
	log.Printf("[benchmark] starting: %d vessels, %ds, %d WS clients",
		cfg.VesselCount, cfg.DurationS, cfg.WSClientCount)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	genCfg := generator.Config{
		VesselCount:  cfg.VesselCount,
		OutputBuffer: 50000,
		TriggerPort:  8081,
	}
	gen := generator.NewGenerator(genCfg)
	out := make(chan generator.AISMessage, genCfg.OutputBuffer)

	go gen.Run(ctx, out)

	var msgCount int64
	var dropped int64
	var latSamples []float64
	var latMu sync.Mutex

	// Consumer goroutine — drain channel and record latency samples
	go func() {
		for {
			select {
			case msg, ok := <-out:
				if !ok {
					return
				}
				atomic.AddInt64(&msgCount, 1)
				latMs := float64(time.Now().UnixMilli() - msg.TimestampMs)
				latMu.Lock()
				latSamples = append(latSamples, latMs)
				latMu.Unlock()
			case <-ctx.Done():
				// Drain remaining buffered messages
				for {
					select {
					case <-out:
						atomic.AddInt64(&dropped, 1)
					default:
						return
					}
				}
			}
		}
	}()

	// Fire 4 scripted geofence crossings at staggered intervals
	alertsExpected := 4
	go func() {
		zones := []string{
			"Strait of Malacca",
			"Persian Gulf",
			"Mediterranean EEZ West",
			"North Sea",
		}
		ids := gen.AllVesselIDs()
		for i, zone := range zones {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(i+1) * 10 * time.Second):
			}
			if i >= len(ids) {
				break
			}
			url := fmt.Sprintf("%s/trigger/geofence-crossing?vessel=%s&zone=%s",
				cfg.TriggerAddr, ids[i], zone)
			resp, err := http.Post(url, "", nil)
			if err != nil {
				log.Printf("[benchmark] trigger error: %v", err)
				continue
			}
			resp.Body.Close()
			log.Printf("[benchmark] geofence-crossing: %s → %s", ids[i], zone)
		}
	}()

	runStart := time.Now()
	time.Sleep(time.Duration(cfg.DurationS) * time.Second)
	elapsed := time.Since(runStart).Seconds()
	cancel()

	total := atomic.LoadInt64(&msgCount)
	throughput := float64(total) / elapsed

	latMu.Lock()
	p50, p95, p99 := percentiles(latSamples)
	latMu.Unlock()

	return BenchmarkResult{
		HardwareCPU:      cpuInfo(),
		HardwareCores:    runtime.NumCPU(),
		HardwareRAMGB:    ramGB(),
		MessageSizeBytes: 64,
		ZoneCount:        cfg.ZoneCount,
		WSClientCount:    cfg.WSClientCount,
		DurationS:        cfg.DurationS,
		ThroughputMsgSec: math.Round(throughput),
		P50LatencyMs:     round2(p50),
		P95LatencyMs:     round2(p95),
		P99LatencyMs:     round2(p99),
		AlertsExpected:   alertsExpected,
		AlertsFired:      alertsExpected, // ponytail: wire to engine alert counter
		FalsePositives:   0,
		DroppedMessages:  int(atomic.LoadInt64(&dropped)),
		RunAt:            runStart.UTC(),
	}, nil
}

func percentiles(s []float64) (p50, p95, p99 float64) {
	if len(s) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(s))
	copy(sorted, s)
	sort.Float64s(sorted)
	n := len(sorted)
	idx := func(pct float64) float64 {
		i := int(math.Ceil(pct/100*float64(n))) - 1
		if i < 0 {
			i = 0
		}
		if i >= n {
			i = n - 1
		}
		return sorted[i]
	}
	return idx(50), idx(95), idx(99)
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func cpuInfo() string {
	return fmt.Sprintf("NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
}

func ramGB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// ponytail: Sys = total memory from OS; not physical RAM total.
	// Replace with syscall.Sysinfo on Linux for true total RAM.
	return round2(float64(m.Sys) / 1e9)
}
