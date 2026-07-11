import json
import httpx
import numpy as np
from sklearn.ensemble import IsolationForest

# Configuration
CORE_ENGINE_URL = "http://localhost:8080"
SSE_URL = f"{CORE_ENGINE_URL}/ws/alerts"
CORROBORATE_URL = f"{CORE_ENGINE_URL}/api/corroborate"

print("Initializing AI Behavior Model (Isolation Forest)...")
# In a real system, this would be trained on months of historical AIS data.
# For the hackathon demo, we use a basic Isolation Forest that scores anomalies
# based on speed and time of day (simulated).
model = IsolationForest(contamination=0.1, random_state=42)

# Dummy training data just to fit the model for the demo
# Features: [Speed, HourOfDay, IsNearReserve]
dummy_history = [
    [10.0, 12, 0], [12.0, 14, 0], [11.5, 10, 0], [9.0, 8, 0], # Normal cargo
    [2.0, 15, 1], [1.5, 16, 1], [2.5, 14, 1],                # Normal fishing
    [25.0, 2, 1], [0.0, 3, 1]                                # Highly anomalous
]
model.fit(dummy_history)
print("Model initialized and ready to score live alerts.")

def extract_features(alert):
    # Try to extract speed from evidence or position
    speed = 0.0
    if "evidence" in alert and "speed_knots" in alert["evidence"]:
        speed = float(alert["evidence"]["speed_knots"])
    elif "position" in alert and "speed_knots" in alert["position"]:
         speed = float(alert["position"]["speed_knots"])
    
    # Simulate an hour of day (just mock it based on alert type for the demo)
    hour = 2 if alert.get("type") == "suspected_dark_transit" else 14
    is_reserve = 1 if "marine_protected_area" in alert.get("zone", "") else 0
    
    return [speed, hour, is_reserve]

def calculate_anomaly_score(features):
    # Predict returns 1 for normal, -1 for anomaly
    # decision_function returns negative for anomalies, positive for normal
    raw_score = model.decision_function([features])[0]
    
    # Map raw_score to 1-99 Risk Score (Lower raw_score = Higher Risk)
    risk = int(np.clip(50 - (raw_score * 50), 1, 99))
    return risk

def corroborate_alert(alert_id: str, risk_score: int):
    payload = {
        "alert_id": alert_id,
        "source": "behavioral-ai-model",
        "evidence": {
            "ai_risk_score": risk_score,
            "ai_confidence": "HIGH" if risk_score > 75 else "MEDIUM",
            "model_used": "IsolationForest"
        }
    }
    try:
        response = httpx.post(CORROBORATE_URL, json=payload)
        response.raise_for_status()
        print(f"✅ AI Risk Score [{risk_score}/99] attached to alert {alert_id}.")
    except Exception as e:
        print(f"❌ Failed to attach AI score to alert {alert_id}: {e}")

def listen_for_alerts():
    print(f"Listening for alerts on {SSE_URL}...")
    try:
        with httpx.stream("GET", SSE_URL) as r:
            for line in r.iter_lines():
                if line.startswith("data: "):
                    data_str = line[6:]
                    try:
                        alert = json.loads(data_str)
                        
                        # Only score alerts that aren't already corroborated
                        if alert.get("corroboration", {}).get("status") == "none":
                            features = extract_features(alert)
                            risk_score = calculate_anomaly_score(features)
                            
                            # Give a massive penalty for identity conflict and dark transit for demo purposes
                            if alert.get("type") in ["identity_conflict", "suspected_dark_transit"]:
                                risk_score = max(risk_score, 85)
                                
                            print(f"\n🧠 Scoring new alert: {alert['type']} (Vessel: {alert['vessel_id']})")
                            corroborate_alert(alert["alert_id"], risk_score)
                                
                    except json.JSONDecodeError:
                        continue
    except KeyboardInterrupt:
        print("Shutting down...")
    except Exception as e:
        print(f"Stream disconnected: {e}")

if __name__ == "__main__":
    try:
        import sklearn
    except ImportError:
        print("Please install required packages: pip install httpx scikit-learn numpy")
        exit(1)
        
    listen_for_alerts()
