import { translateReasoning } from '../utils/reasoningTrace';
import { exportLeadPackage } from '../utils/exportLead';

export default function AlertDetail({ alert, getFormatClass, formatType }) {
  const reasoningText = translateReasoning(alert.reasoning_trace, alert.evidence);

  return (
    <div style={{ 
      margin: '0 8px 16px 8px', 
      padding: '16px', 
      backgroundColor: 'rgba(0,0,0,0.2)', 
      borderRadius: '8px',
      borderLeft: '4px solid var(--color-' + alert.type.replace(/suspected_/, '').split('_')[0] + ')',
      fontSize: '13px'
    }}>
      <div style={{ marginBottom: '12px' }}>
        <strong>Reasoning Trace</strong>
        <p style={{ marginTop: '4px', color: 'var(--text-secondary)', lineHeight: 1.6 }}>
          {reasoningText}
        </p>
        
        {/* Intelligence Fusion Evidence Panel */}
        {alert.evidence && (alert.evidence.trust_score !== undefined || alert.evidence.intent_score !== undefined) && (
          <div style={{ 
            marginTop: '12px',
            padding: '8px 12px',
            backgroundColor: 'rgba(255, 255, 255, 0.05)',
            borderRadius: '6px',
            border: '1px solid rgba(255, 255, 255, 0.1)',
            display: 'flex',
            flexDirection: 'column',
            gap: '8px'
          }}>
            {alert.evidence.trust_score !== undefined && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Truth Score:</span>
                <strong style={{ color: alert.evidence.trust_score < 0.5 ? '#ef4444' : '#10b981' }}>
                  {Math.round(alert.evidence.trust_score * 100)}% (Kinematic Anomaly)
                </strong>
              </div>
            )}
            {alert.evidence.intent_score !== undefined && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ color: 'var(--text-secondary)' }}>Intent Risk:</span>
                <strong style={{ color: alert.evidence.intent_score > 0.8 ? '#ef4444' : '#f59e0b' }}>
                  {Math.round(alert.evidence.intent_score * 100)}% (MPA Proximity)
                </strong>
              </div>
            )}
          </div>
        )}

        {alert.type === 'IDENTITY_FRAUD' && alert.evidence?.mse && (
          <div style={{ 
            marginTop: '8px', 
            padding: '6px', 
            backgroundColor: 'rgba(239, 68, 68, 0.1)', 
            border: '1px solid rgba(239, 68, 68, 0.3)',
            borderRadius: '4px',
            color: '#ef4444',
            fontSize: '11px',
            fontWeight: '600'
          }}>
            🤖 Inference Engine: Trajectory Autoencoder (MSE: {alert.evidence.mse.toFixed(3)})
          </div>
        )}
      </div>

      {alert.recommended_action && (
        <div style={{ marginBottom: '16px', padding: '10px', backgroundColor: 'rgba(245, 158, 11, 0.1)', border: '1px solid rgba(245, 158, 11, 0.3)', borderRadius: '6px' }}>
          <strong style={{ color: '#f59e0b', display: 'block', marginBottom: '4px' }}>Recommended Action</strong>
          <span style={{ color: 'var(--text-primary)' }}>{alert.recommended_action}</span>
        </div>
      )}

      <div style={{ marginBottom: '16px', display: 'flex', gap: '16px' }}>
        <div>
          <span style={{ color: 'var(--text-secondary)' }}>Zone:</span> {alert.zone}
        </div>
        <div>
          <span style={{ color: 'var(--text-secondary)' }}>Corroboration:</span>{' '}
          <span style={{ 
            color: alert.corroboration.status === 'corroborated' ? '#4CAF50' : 'var(--text-secondary)',
            fontWeight: alert.corroboration.status === 'corroborated' ? '600' : 'normal'
          }}>
            {alert.corroboration.status}
          </span>
          {alert.corroboration.source && ` (${alert.corroboration.source})`}
        </div>
      </div>

      <button 
        onClick={(e) => { e.stopPropagation(); exportLeadPackage(alert); }}
        style={{
          width: '100%',
          padding: '8px',
          backgroundColor: 'rgba(255,255,255,0.1)',
          border: '1px solid rgba(255,255,255,0.2)',
          color: 'white',
          borderRadius: '4px',
          cursor: 'pointer',
          fontWeight: '500',
          transition: 'background 0.2s'
        }}
        onMouseOver={(e) => e.target.style.backgroundColor = 'rgba(255,255,255,0.2)'}
        onMouseOut={(e) => e.target.style.backgroundColor = 'rgba(255,255,255,0.1)'}
      >
        Export Lead Package
      </button>
    </div>
  );
}
