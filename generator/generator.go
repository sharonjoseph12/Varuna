package generator

import (
	"context"
	"sync"
	"time"
)

// Generator drives N vessel goroutines and writes AISMessages to out.
type Generator struct {
	cfg     Config
	vessels []*VesselState
	mu      sync.RWMutex // protects vessels slice and individual VesselState fields
}

// NewGenerator initialises a generator with the given config.
func NewGenerator(cfg Config) *Generator {
	zoneTypes := []string{"coastal", "coastal", "offshore", "open_ocean"}
	vessels := make([]*VesselState, cfg.VesselCount)
	for i := range vessels {
		vessels[i] = newVessel(i, zoneTypes[i%len(zoneTypes)])
	}
	return &Generator{cfg: cfg, vessels: vessels}
}

// Run starts the generator and writes AISMessages to out until ctx is cancelled.
// Call as: go g.Run(ctx, out)
// The out channel should be buffered (cfg.OutputBuffer) to avoid blocking.
func (g *Generator) Run(ctx context.Context, out chan<- AISMessage) {
	var wg sync.WaitGroup
	for _, v := range g.vessels {
		wg.Add(1)
		v := v // capture
		go func() {
			defer wg.Done()
			g.runVessel(ctx, v, out)
		}()
	}
	wg.Wait()
}

// runVessel is the per-vessel goroutine. It advances state and emits messages
// at the vessel's cadence, skipping emission when the vessel is intentionally dark.
func (g *Generator) runVessel(ctx context.Context, v *VesselState, out chan<- AISMessage) {
	ticker := time.NewTicker(time.Duration(v.CadenceMs) * time.Millisecond)
	defer ticker.Stop()
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			elapsed := now.Sub(last)
			last = now

			g.mu.Lock()
			tickEvent(v)
			isDark := v.ActiveEvent == EventDarkTransit && !v.DarkUntil.IsZero() && time.Now().Before(v.DarkUntil)
			if !isDark {
				v.advance(elapsed)
			}

			// Build the message under lock so we snapshot consistent state
			mmsi := v.MMSI
			if v.ActiveEvent == EventDuplicateMMSI && v.DuplicateMMSI != "" {
				mmsi = v.DuplicateMMSI
			}
			msg := AISMessage{
				VesselID:    v.ID,
				MMSI:        mmsi,
				Lat:         v.Lat,
				Lon:         v.Lon,
				HeadingDeg:  v.HeadingDeg,
				SpeedKnots:  v.SpeedKnots,
				TimestampMs: now.UnixMilli(),
			}
			g.mu.Unlock()

			if isDark {
				continue // vessel is silent — do not emit
			}

			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		}
	}
}

// GetVessel returns a pointer to the named vessel for use by the trigger server.
// Returns nil if not found.
func (g *Generator) GetVessel(id string) *VesselState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, v := range g.vessels {
		if v.ID == id {
			return v
		}
	}
	return nil
}

// ApplyEvent applies a scripted event to the named vessel under the generator lock.
func (g *Generator) ApplyEvent(id string, fn func(*VesselState)) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, v := range g.vessels {
		if v.ID == id {
			fn(v)
			return true
		}
	}
	return false
}

// AllVesselIDs returns a snapshot of all vessel IDs.
func (g *Generator) AllVesselIDs() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := make([]string, len(g.vessels))
	for i, v := range g.vessels {
		ids[i] = v.ID
	}
	return ids
}
