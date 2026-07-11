import json
import asyncio
import websockets
import httpx
import time

API_KEY = "<YOUR_AISSTREAM_API_KEY>" # Get it free from aisstream.io
INGEST_URL = "http://localhost:8080/api/ingest"

# Bounding box around Indian Ocean (to match our hardcoded zones)
# format: [[lat_min, lon_min], [lat_max, lon_max]]
BOUNDING_BOX = [[-10.0, 50.0], [25.0, 100.0]]

async def connect_ais_stream():
    if API_KEY == "<YOUR_AISSTREAM_API_KEY>":
        print("❌ Please set your API_KEY in aisstream_client.py")
        return

    subscribe_message = {
        "APIKey": API_KEY,
        "BoundingBoxes": [BOUNDING_BOX],
        "FilterMessageTypes": ["PositionReport"]
    }
    
    print(f"Connecting to AISStream... (Bounding Box: {BOUNDING_BOX})")
    
    async with httpx.AsyncClient() as client:
        try:
            async with websockets.connect("wss://stream.aisstream.io/v0/stream") as websocket:
                await websocket.send(json.dumps(subscribe_message))
                print("✅ Connected and subscribed to AISStream.")
                
                async for message_json in websocket:
                    message = json.loads(message_json)
                    if message["MessageType"] == "PositionReport":
                        ais_msg = message["Message"]["PositionReport"]
                        meta = message["MetaData"]
                        
                        # Convert to Varuna's expected format
                        payload = {
                            "vessel_id": str(meta["MMSI"]),
                            "mmsi": str(meta["MMSI"]),
                            "lat": meta["latitude"],
                            "lon": meta["longitude"],
                            "heading": ais_msg["TrueHeading"] if ais_msg["TrueHeading"] != 511 else 0.0,
                            "speed_knots": ais_msg["Sog"],
                            "timestamp_ms": int(time.time() * 1000)
                        }
                        
                        try:
                            # Push to Go Engine
                            r = await client.post(INGEST_URL, json=payload)
                            if r.status_code == 202:
                                print(f"📍 Ingested: MMSI {payload['mmsi']} at {payload['lat']}, {payload['lon']}")
                            else:
                                print(f"⚠️ Engine backpressure: {r.status_code}")
                        except Exception as e:
                            print(f"Error posting to engine: {e}")
                            
        except Exception as e:
            print(f"AISStream Connection Error: {e}")

if __name__ == "__main__":
    try:
        asyncio.run(connect_ais_stream())
    except KeyboardInterrupt:
        print("\nDisconnected.")
