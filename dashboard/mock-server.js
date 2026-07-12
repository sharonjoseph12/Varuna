import { WebSocketServer, WebSocket } from 'ws';
import http from 'http';
import { zones } from './src/utils/zones.js';

const server = http.createServer((req, res) => {
  if (req.url === '/metrics' && req.method === 'GET') {
    res.writeHead(200, { 
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*' 
    });
    res.end(JSON.stringify({
      throughput_per_sec: 50000 + Math.floor(Math.random() * 2000),
      p50_latency_ms: 10 + Math.floor(Math.random() * 5),
      p99_latency_ms: 35 + Math.floor(Math.random() * 10)
    }));
  } else {
    res.writeHead(404);
    res.end();
  }
});

const wssPositions = new WebSocketServer({ noServer: true });
const wssAlerts = new WebSocketServer({ noServer: true });

server.on('upgrade', (request, socket, head) => {
  if (request.url === '/ws/positions') {
    wssPositions.handleUpgrade(request, socket, head, (ws) => {
      wssPositions.emit('connection', ws, request);
    });
  } else if (request.url === '/ws/alerts') {
    wssAlerts.handleUpgrade(request, socket, head, (ws) => {
      wssAlerts.emit('connection', ws, request);
    });
  } else {
    socket.destroy();
  }
});

// State for synthetic vessels (still keep a few for synthetic alerts if needed)
const vessels = new Map();

const AISSTREAM_API_KEY = "68528e0932a0a089229d3e5bc4089619e568fc03";
let aisMsgCount = 0;

function connectAIS() {
  console.log("Connecting to aisstream.io...");
  const aisSocket = new WebSocket("wss://stream.aisstream.io/v0/stream");

  aisSocket.on('open', () => {
    console.log("✓ Connected to aisstream.io");
    const subscriptionMessage = {
      APIKey: AISSTREAM_API_KEY,
      BoundingBoxes: [[[-90, -180], [90, 180]]],
      FilterMessageTypes: ["PositionReport"]
    };
    aisSocket.send(JSON.stringify(subscriptionMessage));
  });

  aisSocket.on('error', (err) => {
    console.error("AisStream error:", err.message || err);
  });

  aisSocket.on('close', (code, reason) => {
    console.log(`AisStream closed: code=${code} reason=${reason}. Reconnecting in 3s...`);
    setTimeout(connectAIS, 3000);
  });

  aisSocket.on('message', (data) => {
    try {
      const msg = JSON.parse(data.toString());

      aisMsgCount++;
      if (aisMsgCount % 100 === 0) {
        console.log(`AIS messages: ${aisMsgCount}, Vessels tracked: ${vessels.size}`);
      }

      if (msg.MessageType === "PositionReport") {
        const metadata = msg.MetaData;
        const report = msg.Message.PositionReport;

        const vId = `MMSI-${metadata.MMSI}`;
        vessels.set(vId, {
          id: vId,
          lat: metadata.latitude,
          lon: metadata.longitude,
          heading: report.TrueHeading || 0,
          speed: report.Sog || 0,
          ship_name: metadata.ShipName || '',
          nav_status: report.NavigationalStatus ?? -1,
          cog: report.Cog || 0
        });

        // Cap at 15k unique vessels (FIFO eviction)
        if (vessels.size > 15000) {
          const firstKey = vessels.keys().next().value;
          vessels.delete(firstKey);
        }
      }
    } catch (err) {
      console.error("Parse error:", err);
    }
  });
}

connectAIS();

let lastAlertTime = 0;

// Broadcast positions
setInterval(() => {
  vessels.forEach(v => {
    // Broadcast positions of live ships

    const msg = JSON.stringify({
      vessel_id: v.id,
      lat: v.lat,
      lon: v.lon,
      heading: v.heading,
      speed_knots: v.speed,
      ship_name: v.ship_name || '',
      nav_status: v.nav_status ?? -1,
      cog: v.cog || 0,
      timestamp_ms: Date.now()
    });

    wssPositions.clients.forEach(client => {
      if (client.readyState === 1) client.send(msg);
    });
  });
}, 500); // ~2fps update rate

// Ray-casting point in polygon algorithm
function pointInPolygon(point, vs) {
  let x = point[0], y = point[1];
  let inside = false;
  for (let i = 0, j = vs.length - 1; i < vs.length; j = i++) {
    let xi = vs[i][0], yi = vs[i][1];
    let xj = vs[j][0], yj = vs[j][1];
    let intersect = ((yi > y) !== (yj > y)) && (x < (xj - xi) * (y - yi) / (yj - yi) + xi);
    if (intersect) inside = !inside;
  }
  return inside;
}

// Keep track of active alerts to avoid spamming for the same vessel
const activeAlerts = new Set();

// Evaluate vessels against protected zones periodically
setInterval(() => {
  if (vessels.size === 0) return;
  
  const vesselArray = Array.from(vessels.values());
  let alertsEmittedThisCycle = 0;
  
  for (const v of vesselArray) {
    if (activeAlerts.has(v.id)) continue; // Already alerted
    
    for (const feature of zones.features) {
      if (feature.geometry.type === "Polygon") {
        const poly = feature.geometry.coordinates[0];
        // Note: GeoJSON is [lon, lat]
        if (pointInPolygon([v.lon, v.lat], poly)) {
          // Intersection found! Generate alert.
          const riskLevel = feature.properties.risk_level === 'critical' ? 'CRITICAL' : 'HIGH';
          const alertType = feature.properties.zone_type === 'marine_protected' ? 'Marine Sanctuary Violation' :
                            feature.properties.zone_type === 'rare_species' ? 'Rare Species Zone Incursion' : 
                            'High Risk Area Entry';

          const alert = {
            alert_id: crypto.randomUUID(),
            type: alertType,
            vessel_id: v.id,
            timestamp: new Date().toISOString(),
            position: { lat: v.lat, lon: v.lon },
            zone: feature.properties.name,
            confidence: 0.95 + (Math.random() * 0.04), // 0.95 - 0.99
            evidence: {
              boundary_proximity_km: 0.0,
              intersection_confirmed: true,
              speed_recorded: v.speed,
              nav_status: v.nav_status
            },
            reasoning_trace: {
              inputs_evaluated: ["ais_position", "geo_fence_intersection"],
              thresholds_used: { risk_level: riskLevel, polygon_check: "ray-casting" },
              modalities_available: ["ais"],
              engine_version: "varuna-eval-v1"
            },
            corroboration: { status: "corroborated", source: "Varuna Spatial Engine" }
          };

          const msg = JSON.stringify(alert);
          wssAlerts.clients.forEach(client => {
            if (client.readyState === 1) client.send(msg);
          });
          
          activeAlerts.add(v.id);
          alertsEmittedThisCycle++;
          
          // Limit to 2 new alerts per cycle to prevent overwhelming UI
          if (alertsEmittedThisCycle >= 2) return;
          break; // Move to next vessel
        }
      }
    }
  }
  
  // Cleanup active alerts after a while so they can fire again if ship re-enters (e.g. 5 minutes)
  if (activeAlerts.size > 200) {
     const iterator = activeAlerts.values();
     activeAlerts.delete(iterator.next().value);
  }
}, 3000);

server.listen(8080, () => {
  console.log('Mock server running on port 8080');
});
