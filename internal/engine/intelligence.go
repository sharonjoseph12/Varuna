package engine

import (
	"fmt"
	"time"
)

func (e *Engine) EvaluateVessel(v *VesselState, ingestTime time.Time) {
	// 1. MODEL 1: Check if the ship is lying (Spoofing) - Trust Model
	if v.TrustBufferIdx == 10 {
		if e.TrustModel != nil {
			// Flatten buffer for Model 1
			points := make([]float32, 0, 30)
			for i := 0; i < 10; i++ {
				points = append(points, v.TrustBuffer[i][0], v.TrustBuffer[i][1], v.TrustBuffer[i][2])
			}
			
			// Run Inference
			out, err := e.TrustModel.RunInference(points, []int64{1, 30})
			if err == nil && len(out) == 30 {
				// Calculate MSE
				var mse float32
				for i := 0; i < len(points); i++ {
					diff := points[i] - out[i]
					mse += diff * diff
				}
				mse /= float32(len(points))

				// Map MSE to Trust Impact
				impact := float32(0.05)
				if mse > 0.05 {
					impact = -0.2
				}
				v.TrustScore += impact
				if v.TrustScore > 1.0 { v.TrustScore = 1.0 }
				if v.TrustScore < 0.0 { v.TrustScore = 0.0 }
			}
		}

		// Shift sliding window
		for i := 1; i < 10; i++ {
			v.TrustBuffer[i-1] = v.TrustBuffer[i]
		}
		v.TrustBufferIdx = 9
	}

	// 2. MODEL 2: Check if the ship is trying to hide (Dark Transit) - Intent Model
	// We check this in the absence logic, but we can also evaluate it here if IsDark is true.
	// Actually, the prompt says "if time.Since(v.LastSeen) > 30*time.Second" but we don't 
	// always have a continuous loop running EvaluateVessel for all ships. We evaluate when 
	// absence is detected. Since EvaluateVessel is called, let's assume IsDark is set by absence.go.
	
	// Mocking Intent Score execution if IsDark is true
	if v.IsDark {
		if e.IntentModel != nil {
			// Calculate inputs: [Speed, Dist_to_Protected, Dist_to_Port, Type]
			// Use the last known speed from the buffer or a default
			var speed float32 = 12.0
			if v.PosIdx > 0 {
				speed = float32(v.Positions[(v.PosIdx-1)%RingBufferSize].SpeedKnots)
			}
			inputs := []float32{speed, v.DistToMPA, v.DistToPort, v.Type}
			out, err := e.IntentModel.RunInference(inputs, []int64{1, 4})
			if err == nil && len(out) == 1 {
				v.IntentScore = out[0]
			}
		} else {
			// MOCK inference if model isn't loaded
			// If near MPA, high intent score
			if v.DistToMPA < 10 {
				v.IntentScore = 0.85
			} else {
				v.IntentScore = 0.2
			}
		}
	}

	// Default state
	v.Priority = "NORMAL"
	v.CaseLabel = ""
	v.Action = ""

	// 3. THE WINNING MOVE: Intelligence Fusion
	if v.TrustScore < 0.4 && v.IsDark {
		// SCENARIO: Lied about position, then turned off AIS.
		v.Priority = "CRITICAL"
		v.CaseLabel = "CONFIRMED EVASION"
		v.Action = "Dispatch Interceptor to last known coordinates immediately."
		
		e.emitFusionAlert(v, "IDENTITY_FRAUD", ingestTime)
	} else if v.IsDark && v.IntentScore > 0.8 {
		// SCENARIO: Normal trust, but disappeared near a protected reef.
		v.Priority = "HIGH"
		v.CaseLabel = "SUSPECTED POACHING"
		v.Action = "Notify Fisheries Dept for visual drone survey."
		
		e.emitFusionAlert(v, "SUSPECTED_ILLEGAL_FISHING", ingestTime)
	} else if v.TrustScore > 0.9 && v.IsDark && v.DistToMPA > 50 {
		// 4. SEARCH & RESCUE
		v.Priority = "EMERGENCY"
		v.CaseLabel = "VESSEL IN DISTRESS"
		v.Action = "Initiate Search & Rescue protocol. AIS lost in open sea."
		
		e.emitFusionAlert(v, "DISTRESS", ingestTime)
	}
}

func (e *Engine) emitFusionAlert(v *VesselState, alertType string, ingestTime time.Time) {
	alert := Alert{
		AlertID:    fmt.Sprintf("fusion-%s-%d", v.VesselID, time.Now().UnixMilli()),
		Type:       alertType,
		VesselID:   v.VesselID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Position:   LatLon{Lat: v.LastLat, Lon: v.LastLon},
		Zone:       "System",
		Confidence: 0.95,
		Evidence: map[string]interface{}{
			"trust_score": v.TrustScore,
			"intent_score": v.IntentScore,
			"is_dark": v.IsDark,
		},
		ReasoningTrace: ReasoningTrace{
			InputsEvaluated: []string{"TrustScore", "IntentScore", "IsDark", "DistToMPA"},
			ThresholdsUsed:  map[string]float64{"Trust_Min": 0.4, "Intent_Min": 0.8},
			ModalitiesAvailable: []string{"Trajectory Autoencoder", "Intent Classifier"},
			EngineVersion: "v2.0.0-fusion",
		},
		Corroboration: Corroboration{Status: "none"},
		RecommendedAction: v.Action,
	}
	e.emitAlert(alert, ingestTime)
}
