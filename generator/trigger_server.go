package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// TriggerServer handles HTTP trigger endpoints for judge-controllable demo events.
// All endpoints are POST except /status (GET). Events are repeatable without restart.
type TriggerServer struct {
	gen       *Generator
	zoneLookup func(name string) (lat, lon float64, ok bool)
}

// NewTriggerServer creates a TriggerServer. zoneLookup maps zone name → centroid.
func NewTriggerServer(g *Generator, zoneLookup func(string) (float64, float64, bool)) *TriggerServer {
	return &TriggerServer{gen: g, zoneLookup: zoneLookup}
}

// ListenAndServe starts the HTTP server on the configured port and blocks until ctx is done.
func (ts *TriggerServer) ListenAndServe(ctx context.Context, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/trigger/dark-transit", ts.handleDarkTransit)
	mux.HandleFunc("/trigger/loitering", ts.handleLoitering)
	mux.HandleFunc("/trigger/duplicate-mmsi", ts.handleDuplicateMMSI)
	mux.HandleFunc("/trigger/geofence-crossing", ts.handleGeofenceCrossing)
	mux.HandleFunc("/trigger/reset", ts.handleReset)
	mux.HandleFunc("/status", ts.handleStatus)

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	log.Printf("[trigger] HTTP trigger server listening on :%d", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// POST /trigger/dark-transit?vessel=ID&silence_s=45&variant=plausible|jump
func (ts *TriggerServer) handleDarkTransit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("vessel")
	silenceS, _ := strconv.Atoi(r.URL.Query().Get("silence_s"))
	variant := DarkVariant(r.URL.Query().Get("variant"))
	if variant != VariantJump {
		variant = VariantPlausible
	}
	if !ts.gen.ApplyEvent(id, func(v *VesselState) {
		applyDarkTransit(v, silenceS, variant)
	}) {
		http.Error(w, "vessel not found", http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]interface{}{
		"ok": true, "vessel_id": id, "event": "dark_transit",
		"params": map[string]interface{}{"silence_s": silenceS, "variant": variant},
	})
}

// POST /trigger/loitering?vessel=ID&duration_s=120
func (ts *TriggerServer) handleLoitering(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("vessel")
	durationS, _ := strconv.Atoi(r.URL.Query().Get("duration_s"))
	if !ts.gen.ApplyEvent(id, func(v *VesselState) {
		applyLoitering(v, durationS)
	}) {
		http.Error(w, "vessel not found", http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]interface{}{
		"ok": true, "vessel_id": id, "event": "loitering",
		"params": map[string]interface{}{"duration_s": durationS},
	})
}

// POST /trigger/duplicate-mmsi?vessel=ID&target_mmsi=MMSI
func (ts *TriggerServer) handleDuplicateMMSI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("vessel")
	targetMMSI := r.URL.Query().Get("target_mmsi")
	if targetMMSI == "" {
		http.Error(w, "target_mmsi required", http.StatusBadRequest)
		return
	}
	if !ts.gen.ApplyEvent(id, func(v *VesselState) {
		applyDuplicateMMSI(v, targetMMSI)
	}) {
		http.Error(w, "vessel not found", http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]interface{}{
		"ok": true, "vessel_id": id, "event": "duplicate_mmsi",
		"params": map[string]interface{}{"target_mmsi": targetMMSI},
	})
}

// POST /trigger/geofence-crossing?vessel=ID&zone=NAME
func (ts *TriggerServer) handleGeofenceCrossing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("vessel")
	zoneName := r.URL.Query().Get("zone")
	if zoneName == "" {
		http.Error(w, "zone required", http.StatusBadRequest)
		return
	}
	zoneLat, zoneLon, ok := ts.zoneLookup(zoneName)
	if !ok {
		http.Error(w, "zone not found", http.StatusNotFound)
		return
	}
	if !ts.gen.ApplyEvent(id, func(v *VesselState) {
		applyGeofenceCrossing(v, zoneName, zoneLat, zoneLon)
	}) {
		http.Error(w, "vessel not found", http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]interface{}{
		"ok": true, "vessel_id": id, "event": "geofence_crossing",
		"params": map[string]interface{}{"zone": zoneName},
	})
}

// POST /trigger/reset?vessel=ID
func (ts *TriggerServer) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("vessel")
	if !ts.gen.ApplyEvent(id, func(v *VesselState) { resetEvent(v) }) {
		http.Error(w, "vessel not found", http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]interface{}{"ok": true, "vessel_id": id, "event": "reset"})
}

// GET /status — returns JSON list of all vessels with active events
func (ts *TriggerServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	ids := ts.gen.AllVesselIDs()
	type entry struct {
		VesselID string `json:"vessel_id"`
		Event    string `json:"event"`
	}
	var active []entry
	ts.gen.mu.RLock()
	for _, v := range ts.gen.vessels {
		if v.ActiveEvent != EventNone {
			active = append(active, entry{VesselID: v.ID, Event: eventName(v.ActiveEvent)})
		}
	}
	ts.gen.mu.RUnlock()
	_ = ids
	jsonOK(w, map[string]interface{}{"ok": true, "active_events": active})
}

func eventName(e ScriptedEvent) string {
	switch e {
	case EventDarkTransit:
		return "dark_transit"
	case EventLoitering:
		return "loitering"
	case EventDuplicateMMSI:
		return "duplicate_mmsi"
	case EventGeofenceCrossing:
		return "geofence_crossing"
	default:
		return "none"
	}
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
