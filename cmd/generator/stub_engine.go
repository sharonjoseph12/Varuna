package main

import (
	"log"
	"sync/atomic"

	"github.com/sharonjoseph12/Varuna/generator"
)

// stubEngine is a no-op engine used when running the generator standalone.
type stubEngine struct {
	ingested     int64
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
