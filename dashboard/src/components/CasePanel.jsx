import { useState, useEffect } from 'react';
import '../styles/case.css';

export default function CasePanel({ alerts, selectedAlertId, onSelectAlert, sarModeEnabled, toggleSarMode }) {
  const [trustScores, setTrustScores] = useState({});

  // Fetch trust scores periodically
  useEffect(() => {
    const fetchTrust = async () => {
      try {
        const res = await fetch('http://localhost:8080/api/trust/');
        if (res.ok) {
          const data = await res.json();
          setTrustScores(data || {});
        }
      } catch (err) {
        console.error("Failed to fetch trust scores", err);
      }
    };
    fetchTrust();
    const interval = setInterval(fetchTrust, 2000);
    return () => clearInterval(interval);
  }, []);

  const formatTime = (isoString) => {
    return new Date(isoString).toLocaleTimeString([], { hour12: false });
  };

  const getFormatClass = (type) => `text-${type}`;

  const formatType = (type) => {
    return type.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
  };

  const getRecommendation = (alert, trust) => {
    if (alert.type === 'suspected_sts_transfer') {
      return { text: "DISPATCH PATROL VESSEL", color: "#FFB347" };
    }
    if (alert.type === 'geofence_breach' && trust > 0.8) {
      return { text: "IMMEDIATE INTERCEPTION", color: "#E74C3C" };
    }
    if (trust < 0.4) {
      return { text: "VISUAL VERIFICATION REQUIRED", color: "#FFB347" };
    }
    return { text: "MONITOR & LOG", color: "#95A5A6" };
  };

  const getTrustColor = (score) => {
    if (score >= 0.8) return '#2ecc71'; // Green
    if (score >= 0.4) return '#f1c40f'; // Yellow
    return '#e74c3c'; // Red
  };

  return (
    <div className="glass-panel case-panel-container">
      <div className="case-header">
        <h2>Active Cases</h2>
        <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{alerts.length} OPEN</span>
      </div>
      
      <div className="case-list">
        {alerts.length === 0 ? (
          <div style={{ padding: '24px', textAlign: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>
            No active cases detected.
          </div>
        ) : (
          alerts.map(alert => {
            const trustObj = trustScores[alert.vessel_id];
            const trustScore = trustObj ? trustObj.score : 1.0;
            const rec = getRecommendation(alert, trustScore);
            const isSelected = selectedAlertId === alert.alert_id;

            return (
              <div 
                key={alert.alert_id}
                className={`case-card ${isSelected ? 'selected' : ''}`}
                onClick={() => onSelectAlert(isSelected ? null : alert.alert_id)}
              >
                <div className="case-card-header">
                  <span className={`case-type ${getFormatClass(alert.type)}`}>
                    {formatType(alert.type)}
                  </span>
                  <span className="case-time">{formatTime(alert.timestamp)}</span>
                </div>

                <div className="case-vessel">
                  TARGET: {alert.vessel_id}
                </div>

                <div className="trust-meter-container">
                  <div className="trust-meter-label">
                    <span>Trust Score</span>
                    <span style={{ color: getTrustColor(trustScore) }}>
                      {(trustScore * 100).toFixed(0)}%
                    </span>
                  </div>
                  <div className="trust-meter-bar">
                    <div 
                      className="trust-meter-fill" 
                      style={{ 
                        width: `${trustScore * 100}%`,
                        backgroundColor: getTrustColor(trustScore)
                      }} 
                    />
                  </div>
                </div>

                {isSelected && trustObj && trustObj.deductions?.length > 0 && (
                  <div style={{ marginBottom: '12px', fontSize: '11px', color: '#e74c3c' }}>
                    <strong>DEDUCTIONS:</strong>
                    <ul style={{ paddingLeft: '16px', marginTop: '4px' }}>
                      {trustObj.deductions.map((d, i) => (
                        <li key={i}>{d.reason} (-{d.amount.toFixed(1)})</li>
                      ))}
                    </ul>
                  </div>
                )}

                <div className="recommendation-box" style={{ borderLeftColor: rec.color }}>
                  <div className="recommendation-label">System Recommendation</div>
                  <div className="recommendation-text" style={{ color: rec.color }}>
                    {rec.text}
                  </div>
                </div>

                {isSelected && (
                  <button 
                    className={`action-btn ${sarModeEnabled ? 'active' : ''}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleSarMode(alert.vessel_id);
                    }}
                  >
                    {sarModeEnabled ? 'DISABLE SAR MODE' : 'ENABLE SAR MODE'}
                  </button>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
