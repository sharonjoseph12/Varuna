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
      </div>

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
