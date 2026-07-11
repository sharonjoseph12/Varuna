package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sharonjoseph12/Varuna/engine"
)

func main() {
	cfg := engine.DefaultConfig()
	zones := engine.DefaultZones()
	eng := engine.NewEngine(cfg, zones)
	defer eng.Close()

	// Start mini-generator
	msgCh := make(chan engine.AISMessage, 100000)
	go miniGenerator(msgCh, zones)

	// Start ingestion
	go eng.Ingest(msgCh)

	// Start alert/position consumers (log to console for standalone mode)
	go consumeAlerts(eng)
	go consumePositions(eng)

	// HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/metrics", metricsHandler(eng))
	mux.HandleFunc("/api/trigger/", triggerHandler(msgCh, zones))
	mux.HandleFunc("/api/corroborate", corroborateHandler(eng))
	mux.HandleFunc("/ws/alerts", wsAlertsHandler(eng))
	mux.HandleFunc("/ws/positions", wsPositionsHandler(eng))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "varuna-core"})
	})

	log.Println("Varuna Core Engine starting on :8080")
	log.Println("Endpoints: /api/metrics, /api/trigger/{event}, /ws/alerts, /ws/positions")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func metricsHandler(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(eng.Metrics())
	}
}

func triggerHandler(msgCh chan<- engine.AISMessage, zones []engine.Zone) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		event := strings.TrimPrefix(r.URL.Path, "/api/trigger/")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		switch event {
		case "dark_transit":
			go scriptDarkTransit(msgCh, zones)
			json.NewEncoder(w).Encode(map[string]string{"triggered": "dark_transit"})
		case "boundary_jitter":
			go scriptBoundaryJitter(msgCh, zones)
			json.NewEncoder(w).Encode(map[string]string{"triggered": "boundary_jitter"})
		case "mmsi_conflict":
			go scriptMMSIConflict(msgCh)
			json.NewEncoder(w).Encode(map[string]string{"triggered": "mmsi_conflict"})
		case "loiter":
			go scriptLoiter(msgCh, zones)
			json.NewEncoder(w).Encode(map[string]string{"triggered": "loiter"})
		default:
			http.Error(w, "unknown event: "+event, http.StatusBadRequest)
		}
	}
}

type CorroborateRequest struct {
	AlertID  string                 `json:"alert_id"`
	Source   string                 `json:"source"`
	Evidence map[string]interface{} `json:"evidence"`
}

func corroborateHandler(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req CorroborateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		eng.Corroborate(req.AlertID, req.Source, req.Evidence)
		
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]string{"status": "corroborated", "alert_id": req.AlertID})
	}
}

// wsAlertsHandler streams alerts via Server-Sent Events (SSE).
// ponytail: SSE is simpler than WebSocket for one-way streaming, and works without nhooyr.io/websocket dep
func wsAlertsHandler(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		alertCh := eng.Alerts()
		for {
			select {
			case alert, ok := <-alertCh:
				if !ok {
					return
				}
				data, _ := json.Marshal(alert)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

// wsPositionsHandler streams positions via SSE.
func wsPositionsHandler(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		posCh := eng.Positions()
		for {
			select {
			case pos, ok := <-posCh:
				if !ok {
					return
				}
				data, _ := json.Marshal(pos)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

func consumeAlerts(eng *engine.Engine) {
	for alert := range eng.Alerts() {
		log.Printf("[ALERT] type=%s vessel=%s zone=%s confidence=%.2f",
			alert.Type, alert.VesselID, alert.Zone, alert.Confidence)
	}
}

func consumePositions(eng *engine.Engine) {
	// ponytail: just drain the channel, positions are for teammate 2's map
	for range eng.Positions() {
	}
}

// --- Mini Generator ---
// ponytail: enough to self-test without teammate 3

func miniGenerator(ch chan<- engine.AISMessage, zones []engine.Zone) {
	vessels := []struct {
		id   string
		mmsi string
		lat  float64
		lon  float64
		hdg  float64
		spd  float64
	}{
		{"V-001", "111111111", 9.0, 78.5, 45, 12},   // heading into Gulf of Mannar
		{"V-002", "222222222", 10.0, 79.5, 90, 8},    // Palk Bay area
		{"V-003", "333333333", 11.0, 72.5, 180, 15},  // Lakshadweep area
		{"V-004", "444444444", 13.0, 93.0, 270, 10},  // Andaman area
		{"V-005", "555555555", 10.0, 76.1, 0, 6},     // Kochi area
		{"V-006", "666666666", 12.0, 68.0, 135, 14},  // Arabian Sea
		{"V-007", "777777777", 9.0, 80.5, 225, 11},   // Sri Lanka area
		{"V-008", "888888888", 9.5, 78.8, 90, 9},     // Near Gulf of Mannar boundary
	}

	ts := time.Now().UnixMilli()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		for i := range vessels {
			v := &vessels[i]
			// Move vessel
			v.lat += math.Cos(v.hdg*math.Pi/180) * v.spd * 0.0001
			v.lon += math.Sin(v.hdg*math.Pi/180) * v.spd * 0.0001

			// Small random heading variation
			v.hdg += (r.Float64() - 0.5) * 5

			ts += 100 // 100ms between messages per vessel

			msg := engine.AISMessage{
				VesselID:    v.id,
				MMSI:        v.mmsi,
				Lat:         v.lat,
				Lon:         v.lon,
				HeadingDeg:  v.hdg,
				SpeedKnots:  v.spd + (r.Float64()-0.5)*2,
				TimestampMs: ts,
			}

			select {
			case ch <- msg:
			default:
				// channel full, skip
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// --- Scripted Events ---

var scriptMu sync.Mutex

func scriptDarkTransit(ch chan<- engine.AISMessage, zones []engine.Zone) {
	scriptMu.Lock()
	defer scriptMu.Unlock()

	// Vessel approaches Gulf of Mannar boundary, then goes dark
	ts := time.Now().UnixMilli()
	vesselID := "V-DARK-001"
	mmsi := "999000001"

	// Send positions approaching the boundary
	for i := 0; i < 10; i++ {
		ch <- engine.AISMessage{
			VesselID: vesselID, MMSI: mmsi,
			Lat: 8.49 + float64(i)*0.01, Lon: 78.0,
			HeadingDeg: 0, SpeedKnots: 12,
			TimestampMs: ts + int64(i)*1000,
		}
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("[SCRIPT] dark_transit: vessel %s going silent near boundary", vesselID)
	// No more messages ΓÇö vessel is dark
}

func scriptBoundaryJitter(ch chan<- engine.AISMessage, zones []engine.Zone) {
	scriptMu.Lock()
	defer scriptMu.Unlock()

	// Vessel jitters back and forth across Gulf of Mannar boundary within hysteresis
	ts := time.Now().UnixMilli()
	vesselID := "V-JITTER-001"
	mmsi := "999000002"
	boundary := 8.5 // southern edge of Gulf of Mannar

	for i := 0; i < 20; i++ {
		// Alternate just above and below boundary, within ~50m (0.0005 deg)
		offset := 0.0003
		if i%2 == 0 {
			offset = -0.0003
		}
		ch <- engine.AISMessage{
			VesselID: vesselID, MMSI: mmsi,
			Lat: boundary + offset, Lon: 78.5,
			HeadingDeg: 0, SpeedKnots: 2,
			TimestampMs: ts + int64(i)*1000,
		}
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("[SCRIPT] boundary_jitter: vessel %s jittered 20 times across boundary", vesselID)
}

func scriptMMSIConflict(ch chan<- engine.AISMessage) {
	scriptMu.Lock()
	defer scriptMu.Unlock()

	ts := time.Now().UnixMilli()
	spoofedMMSI := "999000003"

	// Two "different" vessels broadcasting the same MMSI from impossible positions
	ch <- engine.AISMessage{
		VesselID: "V-SPOOF-A", MMSI: spoofedMMSI,
		Lat: 10.0, Lon: 72.0,
		HeadingDeg: 90, SpeedKnots: 10,
		TimestampMs: ts,
	}
	time.Sleep(100 * time.Millisecond)

	ch <- engine.AISMessage{
		VesselID: "V-SPOOF-B", MMSI: spoofedMMSI,
		Lat: 12.0, Lon: 74.0, // ~300km away, 100ms later ΓÇö impossible
		HeadingDeg: 270, SpeedKnots: 10,
		TimestampMs: ts + 100,
	}

	log.Printf("[SCRIPT] mmsi_conflict: MMSI %s broadcast from two impossible positions", spoofedMMSI)
}

func scriptLoiter(ch chan<- engine.AISMessage, zones []engine.Zone) {
	scriptMu.Lock()
	defer scriptMu.Unlock()

	ts := time.Now().UnixMilli()
	vesselID := "V-LOITER-001"
	mmsi := "999000004"

	// Vessel loiters inside Gulf of Mannar at low speed
	for i := 0; i < 200; i++ {
		ch <- engine.AISMessage{
			VesselID: vesselID, MMSI: mmsi,
			Lat: 9.0 + rand.Float64()*0.002, Lon: 78.5 + rand.Float64()*0.002,
			HeadingDeg: rand.Float64() * 360, SpeedKnots: 1 + rand.Float64()*1.5,
			TimestampMs: ts + int64(i)*10000, // 10s intervals, 200 msgs = ~33 min
		}
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("[SCRIPT] loiter: vessel %s loitered for ~33 minutes in Gulf of Mannar", vesselID)
}
