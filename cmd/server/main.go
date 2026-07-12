package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sharonjoseph12/Varuna/engine"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the hackathon
	},
}

// Global active client lists for broadcasting
var (
	posClients = make(map[*websocket.Conn]bool)
	posMutex   sync.Mutex

	alertClients = make(map[*websocket.Conn]bool)
	alertMutex   sync.Mutex
)

// GeoJSON Types
type FeatureCollection struct {
	Features []Feature `json:"features"`
}
type Feature struct {
	Properties map[string]interface{} `json:"properties"`
	Geometry   Geometry               `json:"geometry"`
}
type Geometry struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"` // For Polygon
}

// AISStream Types
type AISStreamMessage struct {
	MessageType string `json:"MessageType"`
	Message     struct {
		PositionReport *struct {
			Latitude    float64 `json:"Latitude"`
			Longitude   float64 `json:"Longitude"`
			TrueHeading float64 `json:"TrueHeading"`
			Sog         float64 `json:"Sog"`
		} `json:"PositionReport"`
	} `json:"Message"`
	MetaData struct {
		MMSI      int     `json:"MMSI"`
		ShipName  string  `json:"ShipName"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Time_utc  string  `json:"time_utc"`
	} `json:"MetaData"`
}

func loadZones(filePath string) []engine.Zone {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read zones file: %v", err)
	}

	var fc FeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		log.Fatalf("Failed to parse zones file: %v", err)
	}

	var zones []engine.Zone
	for i, feature := range fc.Features {
		if feature.Geometry.Type != "Polygon" || len(feature.Geometry.Coordinates) == 0 {
			continue
		}

		name := "Unknown Zone"
		if n, ok := feature.Properties["name"].(string); ok {
			name = n
		}

		zType := "offshore"
		if t, ok := feature.Properties["zone_type"].(string); ok {
			zType = t
		}

		var poly [][2]float64
		// GeoJSON is [lon, lat], Engine wants [lat, lon]
		for _, coord := range feature.Geometry.Coordinates[0] {
			if len(coord) >= 2 {
				poly = append(poly, [2]float64{coord[1], coord[0]})
			}
		}

		zones = append(zones, engine.Zone{
			ID:                  fmt.Sprintf("zone-%d", i),
			Name:                name,
			Type:                zType,
			Polygon:             poly,
			HysteresisMarginDeg: 0.005, // Generous margin
			SilenceToleranceSec: 600,
			BoundaryBufferKm:    10.0,
		})
	}
	log.Printf("Loaded %d zones from %s", len(zones), filePath)
	return zones
}

func connectAIS(ingestCh chan<- engine.AISMessage) {
	apiKey := "e33255b8c258a8b550fe039f87028b5a86959737"
	url := "wss://stream.aisstream.io/v0/stream"

	for {
		time.Sleep(2 * time.Second) // Prevent aggressive reconnect loops
		log.Println("Connecting to aisstream.io...")
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Printf("Dial error: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		subMsg := map[string]interface{}{
			"APIKey": apiKey,
			"BoundingBoxes": [][][]float64{
				{{20.0, -100.0}, {60.0, 40.0}}, // Europe & North Atlantic
			},
			"FilterMessageTypes": []string{"PositionReport"},
		}

		if err := c.WriteJSON(subMsg); err != nil {
			log.Printf("Write error: %v", err)
			c.Close()
			continue
		}
		log.Println("Subscribed to aisstream.io successfully")

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				break
			}

			// Parse asynchronously to avoid blocking the websocket read loop
			go func(rawMsg []byte) {
				var streamMsg AISStreamMessage
				err := json.Unmarshal(rawMsg, &streamMsg)
				if err != nil {
					// Silent fail on unmarshal errors to avoid log spam
					return
				}

				if streamMsg.MessageType == "PositionReport" && streamMsg.Message.PositionReport != nil {
					mmsiStr := fmt.Sprintf("%d", streamMsg.MetaData.MMSI)

					ingestCh <- engine.AISMessage{
						VesselID:   "MMSI-" + mmsiStr,
						MMSI:       mmsiStr,
						Lat:        streamMsg.MetaData.Latitude,
						Lon:        streamMsg.MetaData.Longitude,
						HeadingDeg: streamMsg.Message.PositionReport.TrueHeading,
						SpeedKnots: streamMsg.Message.PositionReport.Sog,
						TimestampMs: time.Now().UnixMilli(),
					}
				}
			}(message)
		}
		c.Close()
		log.Println("Disconnected from aisstream.io, reconnecting...")
	}
}

func main() {
	log.Println("Starting Varuna Go Engine Server...")

	cfg := engine.DefaultConfig()
	// Tweak config for hackathon performance limits
	cfg.TickIntervalMs = 50 
	
	zones := loadZones("engine/zones.json")
	varunaEngine := engine.NewEngine(cfg, zones)

	ingestCh := make(chan engine.AISMessage, 50000)

	// Determine mode: Normal or Benchmark
	if len(os.Args) > 1 && os.Args[1] == "--benchmark" {
		log.Println("🚀 RUNNING IN BENCHMARK MODE (Max Throughput Test)")
		go func() {
			// Pre-generate a batch of messages to avoid math/rand overhead in the hot loop
			batchSize := 10000
			batch := make([]engine.AISMessage, batchSize)
			for i := 0; i < batchSize; i++ {
				batch[i] = engine.AISMessage{
					VesselID:   fmt.Sprintf("MMSI-BENCH-%d", rand.Intn(20000)),
					MMSI:       fmt.Sprintf("BENCH-%d", rand.Intn(20000)),
					Lat:        -90 + rand.Float64()*180,
					Lon:        -180 + rand.Float64()*360,
					HeadingDeg: rand.Float64() * 360,
					SpeedKnots: rand.Float64() * 20,
				}
			}

			// Blast them!
			idx := 0
			for {
				msg := batch[idx]
				msg.TimestampMs = time.Now().UnixMilli()
				ingestCh <- msg
				idx = (idx + 1) % batchSize
			}
		}()
	} else {
		// Live Mode
		go connectAIS(ingestCh)
	}

	// Start Engine Processing
	go varunaEngine.Ingest(ingestCh)

	// Metrics Logger
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics := varunaEngine.Metrics()
			log.Printf("METRICS: Throughput: %.0f msgs/sec | Latency (p99): %.2f ms | Alerts: %d", 
				metrics.ThroughputMsgsSec, metrics.LatencyP99Ms, metrics.TotalAlerts)
		}
	}()

	// Broadcast Positions from Engine -> WebSockets (include trust score)
	go func() {
		for pos := range varunaEngine.Positions() {
			payload := map[string]interface{}{
				"vessel_id":    pos.VesselID,
				"lat":          pos.Lat,
				"lon":          pos.Lon,
				"heading":      pos.HeadingDeg,
				"speed_knots":  pos.SpeedKnots,
				"timestamp_ms": pos.TimestampMs,
			}

			// Attach trust score if available
			if ts, ok := varunaEngine.TrustScoreFor(pos.VesselID); ok {
				payload["trust_score"] = ts.Score
			}

			msg, _ := json.Marshal(payload)

			posMutex.Lock()
			for client := range posClients {
				_ = client.WriteMessage(websocket.TextMessage, msg)
			}
			posMutex.Unlock()
		}
	}()

	// Broadcast Alerts from Engine -> WebSockets
	go func() {
		for alert := range varunaEngine.Alerts() {
			// Convert to map to inject trust score
			alertMap := map[string]interface{}{
				"alert_id": alert.AlertID,
				"type": alert.Type,
				"vessel_id": alert.VesselID,
				"timestamp": alert.Timestamp,
				"position": alert.Position,
				"zone": alert.Zone,
				"confidence": alert.Confidence,
				"evidence": alert.Evidence,
				"reasoning_trace": alert.ReasoningTrace,
				"corroboration": alert.Corroboration,
			}

			// Attach trust score if available
			if ts, ok := varunaEngine.TrustScoreFor(alert.VesselID); ok {
				alertMap["trust_score"] = ts.Score
			} else {
				alertMap["trust_score"] = 1.0
			}

			msg, _ := json.Marshal(alertMap)
			
			alertMutex.Lock()
			for client := range alertClients {
				_ = client.WriteMessage(websocket.TextMessage, msg)
			}
			alertMutex.Unlock()
		}
	}()

	// CORS middleware helper
	corsHandler := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(204)
				return
			}
			next(w, r)
		}
	}

	// Metrics endpoint — polled by dashboard MetricsPanel
	http.HandleFunc("/metrics", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		m := varunaEngine.Metrics()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"throughput_per_sec": m.ThroughputMsgsSec,
			"p50_latency_ms":    m.LatencyP50Ms,
			"p95_latency_ms":    m.LatencyP95Ms,
			"p99_latency_ms":    m.LatencyP99Ms,
			"total_processed":   m.TotalProcessed,
			"total_alerts":      m.TotalAlerts,
		})
	}))

	// Vessel trust score endpoint — fetched by VesselDetailsModal
	http.HandleFunc("/api/vessel/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		vesselID := r.URL.Path[len("/api/vessel/"):]
		if vesselID == "" {
			http.Error(w, "vessel id required", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		ts, ok := varunaEngine.TrustScoreFor("MMSI-" + vesselID)
		if !ok {
			// Return a default trust score for vessels not yet evaluated
			json.NewEncoder(w).Encode(map[string]interface{}{
				"vessel_id":   vesselID,
				"trust_score": 1.0,
				"risk_level":  "LOW",
				"deductions":  []interface{}{},
				"evaluated":   false,
			})
			return
		}
		riskLevel := "LOW"
		if ts.Score < 0.5 {
			riskLevel = "HIGH"
		} else if ts.Score < 0.8 {
			riskLevel = "MEDIUM"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"vessel_id":        ts.VesselID,
			"trust_score":      ts.Score,
			"risk_level":       riskLevel,
			"deductions":       ts.Deductions,
			"hash_valid":       ts.HashValid,
			"last_evaluated_ms": ts.LastEvaluatedMs,
			"evaluated":        true,
		})
	}))

	// HTTP Handlers
	http.HandleFunc("/ws/positions", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		posMutex.Lock()
		posClients[conn] = true
		posMutex.Unlock()

		go func() {
			defer func() {
				posMutex.Lock()
				delete(posClients, conn)
				posMutex.Unlock()
				conn.Close()
			}()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})

	http.HandleFunc("/ws/alerts", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		alertMutex.Lock()
		alertClients[conn] = true
		alertMutex.Unlock()

		go func() {
			defer func() {
				alertMutex.Lock()
				delete(alertClients, conn)
				alertMutex.Unlock()
				conn.Close()
			}()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})

	// Serve the static dashboard UI
	fs := http.FileServer(http.Dir("dashboard/dist"))
	http.Handle("/", fs)

	port := "8080"
	log.Printf("Server running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
