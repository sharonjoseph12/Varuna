package engine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ponytail: server-side HMAC key — in production this comes from a secret store
var trustHMACKey = []byte("varuna-trust-engine-secret-key-v1")

// TrustScore represents the trust evaluation for a single vessel.
type TrustScore struct {
	VesselID        string           `json:"vessel_id"`
	Score           float64          `json:"score"`             // 0.0–1.0
	Deductions      []TrustDeduction `json:"deductions"`
	SecurityHash    string           `json:"security_hash"`     // HMAC-SHA256 of payload
	HashValid       bool             `json:"hash_valid"`
	LastEvaluatedMs int64            `json:"last_evaluated_ms"`
}

// TrustDeduction records a single reason for lowering trust.
type TrustDeduction struct {
	Reason string  `json:"reason"` // kinematic_impossibility | identity_conflict | ml_anomaly | hash_mismatch
	Amount float64 `json:"amount"`
	Detail string  `json:"detail"`
}

// evaluateTrust computes the trust score for a vessel based on the current message.
func (e *Engine) evaluateTrust(msg AISMessage, ingestTime time.Time) {
	score := 1.0
	var deductions []TrustDeduction

	// 1. Kinematic Impossibility: speed > 50 knots for non-fast-craft
	if msg.SpeedKnots > e.cfg.MaxVesselSpeedKnots && msg.VesselType != "fast_craft" {
		d := 0.3
		score -= d
		deductions = append(deductions, TrustDeduction{
			Reason: "kinematic_impossibility",
			Amount: d,
			Detail: fmt.Sprintf("speed %.1f knots exceeds max %g for type %q",
				msg.SpeedKnots, e.cfg.MaxVesselSpeedKnots, msg.VesselType),
		})
	}

	// 2. Identity Conflict: same MMSI seen from another vessel recently
	if msg.MMSI != "" {
		e.vesselsMu.RLock()
		vs := e.mmsiIndex[msg.MMSI]
		if vs != nil && vs.VesselID != msg.VesselID && vs.LastSeen > 0 {
			elapsed := msg.TimestampMs - vs.LastSeen
			if elapsed > 0 && elapsed < 300000 { // within 5 min
				distKm := haversineKm(vs.LastLat, vs.LastLon, msg.Lat, msg.Lon)
				elapsedHrs := float64(elapsed) / 3600000.0
				maxKm := e.cfg.MaxVesselSpeedKnots * 1.852 * elapsedHrs
				if distKm > maxKm {
					d := 0.4
					score -= d
					deductions = append(deductions, TrustDeduction{
						Reason: "identity_conflict",
						Amount: d,
						Detail: fmt.Sprintf("MMSI %s also seen at vessel %s, %.1fkm away in %.0fs (max possible %.1fkm)",
							msg.MMSI, vs.VesselID, distKm, float64(elapsed)/1000, maxKm),
					})
				}
			}
		}
		e.vesselsMu.RUnlock()
	}

	// 3. ML Anomaly: simulated reconstruction error from DarkIntentModel
	// ponytail: uses the simulated ONNX model; real model swaps in at onnx.go
	if e.darkModel != nil {
		reErr := e.darkModel.ReconstructionError(msg.Lat, msg.Lon, msg.HeadingDeg, msg.SpeedKnots)
		if reErr > 0.7 { // high reconstruction error
			d := 0.3
			score -= d
			deductions = append(deductions, TrustDeduction{
				Reason: "ml_anomaly",
				Amount: d,
				Detail: fmt.Sprintf("reconstruction error %.3f exceeds threshold 0.7", reErr),
			})
		}
	}

	// 4. Security Hash check (MITM spoofing simulation)
	computedHash := computeSecurityHash(msg)
	hashValid := true
	if msg.SecurityHash != "" && msg.SecurityHash != computedHash {
		d := 0.5
		score -= d
		hashValid = false
		deductions = append(deductions, TrustDeduction{
			Reason: "hash_mismatch",
			Amount: d,
			Detail: fmt.Sprintf("expected %s, got %s — possible MITM spoofing", computedHash[:16], msg.SecurityHash[:min(16, len(msg.SecurityHash))]),
		})
	}

	// Clamp
	if score < 0 {
		score = 0
	}

	ts := &TrustScore{
		VesselID:        msg.VesselID,
		Score:           score,
		Deductions:      deductions,
		SecurityHash:    computedHash,
		HashValid:       hashValid,
		LastEvaluatedMs: msg.TimestampMs,
	}

	e.trustMu.Lock()
	e.trustScores[msg.VesselID] = ts
	e.trustMu.Unlock()
}

// TrustScoreFor returns the current trust score for a vessel. Thread-safe.
func (e *Engine) TrustScoreFor(vesselID string) (TrustScore, bool) {
	e.trustMu.RLock()
	defer e.trustMu.RUnlock()
	ts, ok := e.trustScores[vesselID]
	if !ok {
		return TrustScore{}, false
	}
	return *ts, true
}

// AllTrustScores returns a snapshot of all trust scores.
func (e *Engine) AllTrustScores() map[string]TrustScore {
	e.trustMu.RLock()
	defer e.trustMu.RUnlock()
	out := make(map[string]TrustScore, len(e.trustScores))
	for k, v := range e.trustScores {
		out[k] = *v
	}
	return out
}

// computeSecurityHash produces HMAC-SHA256 of the message payload.
func computeSecurityHash(msg AISMessage) string {
	mac := hmac.New(sha256.New, trustHMACKey)
	fmt.Fprintf(mac, "%s|%.6f|%.6f|%d", msg.MMSI, msg.Lat, msg.Lon, msg.TimestampMs)
	return hex.EncodeToString(mac.Sum(nil))
}


