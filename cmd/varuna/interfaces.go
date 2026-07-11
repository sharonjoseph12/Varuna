// Package main (cmd/varuna) defines the EngineInterface so main.go compiles
// and can be tested standalone before teammate 1's engine package is ready.
package main

import (
	"log"
	"sync/atomic"

	"varuna/generator"
)

// EngineInterface is the surface Varuna's generator and corroboration jobs
// depend on. Teammate 1 must implement both methods with these exact signatures.
type EngineInterface interface {
	// Ingest reads AISMessages from in until it is closed.
	// Run as a goroutine: go engine.Ingest(ch)
	Ingest(in <-chan generator.AISMessage)

	// Corroborate upgrades an existing alert with corroboration evidence.
	// source is "sar" or "viirs". evidence is the CorroborationEvidence as a map.
	Corroborate(alertID string, source string, evidence map[string]interface{})
}

// stubEngine is a no-op EngineInterface used when running the generator
// standalone (e.g. for benchmark or demo without teammate 1's package).
type stubEngine struct {
	ingested    int64
	corroborated int64
}

func (e *stubEngine) Ingest(in <-chan generator.AISMessage) {
	for msg := range in {
		_ = msg
		n := atomic.AddInt64(&e.ingested, 1)
		if n%100000 == 0 {
			log.Printf("[stub-engine] ingested %d messages", n)
		}
	}
}

func (e *stubEngine) Corroborate(alertID, source string, evidence map[string]interface{}) {
	atomic.AddInt64(&e.corroborated, 1)
	log.Printf("[stub-engine] corroborate alert=%s source=%s confidence=%v",
		alertID, source, evidence["detection_confidence"])
}
