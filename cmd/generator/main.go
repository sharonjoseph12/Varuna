// cmd/generator runs the Member 3 standalone generator + corroboration jobs.
// Use this when you want to run the generator independently of the core engine,
// e.g. to test throughput or trigger events before teammate 1 is integrated.
//
// Usage:
//
//	go run ./cmd/generator [-vessels 200] [-trigger-port 8081] [-data ./data] [-stub]
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sharonjoseph12/Varuna/corroboration"
	"github.com/sharonjoseph12/Varuna/generator"
	"github.com/sharonjoseph12/Varuna/zones"
)

func main() {
	vessels := flag.Int("vessels", 200, "number of simulated vessels")
	triggerPort := flag.Int("trigger-port", 8081, "HTTP trigger server port")
	dataDir := flag.String("data", "data", "path to data directory")
	sarTick := flag.Duration("sar-tick", 30*time.Second, "SAR job poll interval")
	viirsTick := flag.Duration("viirs-tick", 60*time.Second, "VIIRS stub poll interval")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load zones
	zoneList, err := zones.LoadZones("zones/zones.geojson")
	if err != nil {
		log.Fatalf("load zones: %v", err)
	}
	log.Printf("[generator] loaded %d zones", len(zoneList))
	zoneLookup := zones.ZoneLookup(zoneList)

	// Start generator
	genCfg := generator.Config{
		VesselCount:  *vessels,
		OutputBuffer: 50000,
		TriggerPort:  *triggerPort,
		DataDir:      *dataDir,
	}
	gen := generator.NewGenerator(genCfg)
	aisCh := make(chan generator.AISMessage, genCfg.OutputBuffer)

	// Stub engine — drains channel and logs
	eng := &stubEngine{}

	// Corroboration alert channels (stubbed for standalone mode)
	sarAlerts := make(chan corroboration.AlertRef, 64)
	viirsAlerts := make(chan corroboration.AlertRef, 64)

	corroborateFn := corroboration.CorroborateFunc(
		func(alertID, source string, evidence map[string]interface{}) {
			eng.Corroborate(alertID, source, evidence)
		},
	)

	sarTile := corroboration.SARTile{
		TileID:   "S1A_IW_GRDH_1SDV_20260710T123456",
		FilePath: *dataDir + "/sar/S1A_IW_GRDH_1SDV_20260710T123456.tiff",
		MinLat:   1.0, MaxLat: 6.5,
		MinLon:   99.0, MaxLon: 104.5,
	}
	sarModelPath := *dataDir + "/models/sar_infer.py"

	go gen.Run(ctx, aisCh)
	go eng.Ingest(aisCh)
	go corroboration.RunSARJob(ctx, sarAlerts, corroborateFn, sarTile, sarModelPath, *sarTick)
	go corroboration.RunVIIRSStub(ctx, viirsAlerts, corroborateFn, *viirsTick)

	ts := generator.NewTriggerServer(gen, zoneLookup)
	go func() {
		if err := ts.ListenAndServe(ctx, *triggerPort); err != nil {
			log.Printf("[generator] trigger server error: %v", err)
		}
	}()

	log.Printf("[generator] running — %d vessels, trigger port :%d", *vessels, *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/dark-transit?vessel=vessel-0000&silence_s=45&variant=plausible'", *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/duplicate-mmsi?vessel=vessel-0001&target_mmsi=200000000'", *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/loitering?vessel=vessel-0002&duration_s=120'", *triggerPort)
	log.Printf("  curl -X POST 'http://localhost:%d/trigger/geofence-crossing?vessel=vessel-0003&zone=Strait+of+Malacca'", *triggerPort)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("[generator] shutting down")
	cancel()
}
