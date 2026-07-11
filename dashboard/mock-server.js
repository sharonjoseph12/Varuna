import { WebSocketServer, WebSocket } from 'ws';
import http from 'http';

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

// Generate some synthetic ones just to be safe
Array.from({ length: 10 }, (_, i) => {
  vessels.set(`vessel_${i}`, {
    id: `vessel_${i}`,
    lat: 40.0 + (Math.random() - 0.5) * 5,
    lon: -72.5 + (Math.random() - 0.5) * 5,
    heading: Math.random() * 360,
    speed: 5 + Math.random() * 15
  });
});

const AISSTREAM_API_KEY = "b475125d12173e57f2a82eb974ae821ba39b952f";
const aisSocket = new WebSocket("wss://stream.aisstream.io/v0/stream");

aisSocket.on('open', () => {
  console.log("Connected to aisstream.io");
  const subscriptionMessage = {
      APIKey: AISSTREAM_API_KEY,
      BoundingBoxes: [[[-90, -180], [90, 180]]],
      FilterMessageTypes: ["PositionReport"]
  };
  aisSocket.send(JSON.stringify(subscriptionMessage));
});

aisSocket.on('error', (err) => {
  console.error("AisStream error", err);
});

let aisMsgCount = 0;

aisSocket.on('message', (data) => {
  try {
    const msg = JSON.parse(data.toString());
    
    // Log occasionally to confirm receipt
    aisMsgCount++;
    if (aisMsgCount % 50 === 0) {
      console.log(`Received 50 more AIS messages... Total: ${aisMsgCount}, Current Map Size: ${vessels.size}`);
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
        speed: report.Sog || 0
      });
      
      // Cleanup old vessels if we have too many
      if (vessels.size > 2000) {
        const firstKey = vessels.keys().next().value;
        vessels.delete(firstKey);
      }
    } else {
      console.log("Received other message type:", msg.MessageType);
    }
  } catch (err) {
    console.error("Parse error:", err);
  }
});

let lastAlertTime = 0;

// Broadcast positions
setInterval(() => {
  vessels.forEach(v => {
    // Move synthetic vessels a bit
    if (v.id && v.id.startsWith('vessel_')) {
      v.lat += (Math.cos(v.heading * Math.PI / 180) * 0.0001 * v.speed);
      v.lon += (Math.sin(v.heading * Math.PI / 180) * 0.0001 * v.speed);
      
      // Bounce off edges roughly
      if (v.lat > 41 || v.lat < 39) v.heading = 180 - v.heading;
      if (v.lon > -71 || v.lon < -74) v.heading = 360 - v.heading;
    }

    const msg = JSON.stringify({
      vessel_id: v.id,
      lat: v.lat,
      lon: v.lon,
      heading: v.heading,
      speed_knots: v.speed,
      timestamp_ms: Date.now()
    });

    wssPositions.clients.forEach(client => {
      if (client.readyState === 1) client.send(msg);
    });
  });
}, 500); // ~2fps update rate

// Simulate random alerts occasionally
setInterval(() => {
  if (Date.now() - lastAlertTime < 5000) return;
  if (vessels.size === 0) return;
  
  const types = ["geofence_breach", "suspected_dark_transit", "suspected_illegal_fishing", "identity_conflict"];
  const type = types[Math.floor(Math.random() * types.length)];
  
  const vesselArray = Array.from(vessels.values());
  const v = vesselArray[Math.floor(Math.random() * vesselArray.length)];
  
  const alert = {
    alert_id: crypto.randomUUID(),
    type,
    vessel_id: v.id,
    timestamp: new Date().toISOString(),
    position: { lat: v.lat, lon: v.lon },
    zone: "Coastal Protected Zone",
    confidence: 0.6 + Math.random() * 0.3,
    evidence: {
      silence_duration_s: 340,
      boundary_proximity_km: 0.8,
      conflicting_position: type === "identity_conflict" ? { lat: v.lat + 1, lon: v.lon + 1 } : null
    },
    reasoning_trace: {
      inputs_evaluated: ["silence_ratio", "boundary_proximity", "historical_gap_pattern"],
      thresholds_used: { zone_tolerance_s: 106, boundary_buffer_km: 2.0 },
      modalities_available: ["ais"],
      engine_version: "absence-engine-v1"
    },
    corroboration: { status: Math.random() > 0.8 ? "corroborated" : "none", source: "Sentinel-1" }
  };

  const msg = JSON.stringify(alert);
  wssAlerts.clients.forEach(client => {
    if (client.readyState === 1) client.send(msg);
  });
  lastAlertTime = Date.now();
}, 2000);

server.listen(8080, () => {
  console.log('Mock server running on port 8080');
});
