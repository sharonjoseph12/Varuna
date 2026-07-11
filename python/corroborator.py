import json
import httpx
from detectron2.config import get_cfg
from detectron2.engine import DefaultPredictor
import cv2

# Configuration
CORE_ENGINE_URL = "http://localhost:8080"
SSE_URL = f"{CORE_ENGINE_URL}/ws/alerts"
CORROBORATE_URL = f"{CORE_ENGINE_URL}/api/corroborate"

# Initialize SAR Model (runs once on startup)
print("Loading HRSID SAR model...")
cfg = get_cfg()
cfg.merge_from_file("config.yaml")
cfg.MODEL.WEIGHTS = "model_final.pth"
cfg.MODEL.ROI_HEADS.SCORE_THRESH_TEST = 0.5
cfg.MODEL.DEVICE = "cpu"  # Change to "cuda" if GPU is available
predictor = DefaultPredictor(cfg)
print("Model loaded successfully.")

def run_inference(image_path: str):
    image = cv2.imread(image_path)
    if image is None:
        return []
    
    outputs = predictor(image)
    instances = outputs["instances"].to("cpu")
    
    ships = []
    for box, score in zip(instances.pred_boxes, instances.scores):
        ships.append({
            "bbox": box.tolist(),
            "confidence": float(score)
        })
    return ships

def corroborate_alert(alert_id: str, ships: list):
    payload = {
        "alert_id": alert_id,
        "source": "hrsid-sar-model",
        "evidence": {
            "ships_detected": len(ships),
            "detections": ships,
            "model_version": "PUSHPENDAR/hrsid-ship-detection"
        }
    }
    try:
        response = httpx.post(CORROBORATE_URL, json=payload)
        response.raise_for_status()
        print(f"✅ Alert {alert_id} successfully corroborated.")
    except Exception as e:
        print(f"❌ Failed to corroborate alert {alert_id}: {e}")

def listen_for_alerts():
    print(f"Listening for alerts on {SSE_URL}...")
    try:
        with httpx.stream("GET", SSE_URL) as r:
            for line in r.iter_lines():
                if line.startswith("data: "):
                    data_str = line[6:]
                    try:
                        alert = json.loads(data_str)
                        if alert.get("type") == "suspected_dark_transit" and alert.get("corroboration", {}).get("status") == "none":
                            print(f"\n🚨 Detected uncorroborated dark transit: {alert['alert_id']}")
                            
                            # In a real system, you would fetch the corresponding SAR tile based on the alert's position/time.
                            # For the hackathon demo, we use a static test image provided in the repository.
                            test_image = "test_sar.jpg" 
                            
                            print(f"Running SAR inference on {test_image}...")
                            ships = run_inference(test_image)
                            
                            if len(ships) > 0:
                                print(f"Found {len(ships)} ships. Corroborating alert...")
                                corroborate_alert(alert["alert_id"], ships)
                            else:
                                print("No ships found. Alert remains uncorroborated.")
                                
                    except json.JSONDecodeError:
                        continue
    except KeyboardInterrupt:
        print("Shutting down...")
    except Exception as e:
        print(f"Stream disconnected: {e}")

if __name__ == "__main__":
    # Ensure httpx is installed, otherwise suggest it
    try:
        import httpx
    except ImportError:
        print("Please install httpx: pip install httpx")
        exit(1)
        
    listen_for_alerts()
