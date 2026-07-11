// cmd/varuna is the main Varuna binary.
// It wires the synthetic AIS generator, the engine (teammate 1), and the
// offline corroboration jobs (SAR + VIIRS stub) into a single process.
//
// Usage:
//
//	go run ./cmd/varuna [-vessels 200] [-trigger-port 8081] [-data ./data] [-stub]
//
// -stub runs with a no-op engine (useful for generator standalone testing).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"varuna/corroboration"
	"varuna/generator"
	"varuna/zones"
)

func main() {
	vessels := flag.Int("vessels", 200, "number of simulated vessels")
	triggerPort := flag.Int("trigger-port", 8081, "HTTP trigger server port")
	dataDir := flag.String("data", "data", "path to data directory")
	stub := flag.Bool("stub", false, "use stub engine (no teammate 1 dependency)")
	sarTick := flag.Duration("sar-tick", 30*time.Second, "SAR job poll interval")
	viirsick := flag.Duration("viirs-tick", 60*time.Second, "VIIRS stub poll interval")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Load zones ---
	zoneList, err := zones.LoadZones("zones/zones.geojson")
	if err != nil {
		log.Fatalf("load zones: %v", err)
	}
	log.Printf("[varuna] loaded %d zones", len(zoneList))
	zoneLookup := zones.ZoneLookup(zoneList)

	// --- Generator ---
	genCfg := generator.Config{
		VesselCount:  *vessels,
		OutputBuffer: 50000,
		TriggerPort:  *triggerPort,
		DataDir:      *dataDir,
	}
	gen := generator.NewGenerator(genCfg)
	aisCh := make(chan generator.AISMessage, genCfg.OutputBuffer)

	// --- Engine ---
	var eng EngineInterface
	if *stub {
		log.Println("[varuna] running with stub engine")
		eng = &stubEngine{}
	} else {
		// TODO: replace with real engine when teammate 1's package is ready:
		//   import "varuna/engine"
		//   eng = engine.New(engineCfg)
		log.Println("[varuna] WARNING: no real engine configured; using stub")
		eng = &stubEngine{}
	}

	// --- Corroboration alert channels ---
	// Each corroboration job receives AlertRefs from the engine via these channels.
	// The engine pushes AlertRef onto these channels whenever a suspected_dark_transit fires.
	// For standalone testing we use stub channels that never receive.
	sarAlerts := make(chan corroboration.AlertRef, 64)
	viirsAlerts := make(chan corroboration.AlertRef, 64)

	corroborateFn := corroboration.CorroborateFunc(
		func(alertID, source string, evidence map[string]interface{}) {
			eng.Corroborate(alertID, source, evidence)
		},
	)

	// --- SAR tile config (pre-downloaded Sentinel-1 GRD tile) ---
	sarTile := corroboration.SARTile{
		TileID:   "S1A_IW_GRDH_1SDV_20260710T123456",
		FilePath: *dataDir + "/sar/S1A_IW_GRDH_1SDV_20260710T123456.tiff",
		MinLat:   1.0, MaxLat: 6.5,  // covers Strait of Malacca scripted vessel area
		MinLon:   99.0, MaxLon: 104.5,
	}
	sarModelPath := *dataDir + "/models/sar_infer.py"

	// --- Start goroutines ---
	go gen.Run(ctx, aisCh)
	go eng.Ingest(aisCh)
	go corroboration.RunSARJob(ctx, sarAlerts, corroborateFn, sarTile, sarModelPath, *sarTick)
	go corroboration.RunVIIRSStub(ctx, viirsAlerts, corroborateFn, *viirsick)

	// --- Trigger server ---
	ts := generator.NewTriggerServer(gen, zoneLookup)
	go func() {
		if err := ts.ListenAndServe(ctx, *triggerPort); err != nil {
			log.Printf("[varuna] trigger server error: %v", err)
		}
	}()

	log.Printf("[varuna] running — %d vessels, trigger port :%d", *vessels, *triggerPort)
	log.Printf("[varuna] trigger examples:")
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/dark-transit?vessel=vessel-0000&silence_s=45&variant=plausible'", *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/duplicate-mmsi?vessel=vessel-0001&target_mmsi=200000000'", *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/loitering?vessel=vessel-0002&duration_s=120'", *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/geofence-crossing?vessel=vessel-0003&zone=Strait+of+Malacca'", *triggerPort)

	// Graceful shutdown on SIGINT / SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("[varuna] shutting down")
	cancel()
}
