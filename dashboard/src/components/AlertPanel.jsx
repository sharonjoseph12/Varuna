import AlertDetail from './AlertDetail';

export default function AlertPanel({ alerts, selectedAlertId, onSelectAlert }) {
  const formatTime = (isoString) => {
    return new Date(isoString).toLocaleTimeString([], { hour12: false });
  };

  const getFormatClass = (type) => {
    return `text-${type}`;
  };

  const getBgClass = (type) => {
    return `bg-${type}`;
  };

  const formatType = (type) => {
    return type.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
  };

  return (
    <div className="glass-panel alerts-container">
      <div style={{ padding: '16px', borderBottom: 'var(--glass-border)' }}>
        <h2 style={{ fontSize: '16px', fontWeight: '600' }}>Active Cases</h2>
      </div>
      
      <div style={{ flexGrow: 1, overflowY: 'auto', padding: '8px' }}>
        {alerts.length === 0 ? (
          <div style={{ padding: '24px', textAlign: 'center', color: 'var(--text-secondary)' }}>
            No alerts yet. Waiting for events...
          </div>
        ) : (
          alerts.map((alert, index) => (
            <div key={alert.alert_id + '-' + index}>
              <div 
                onClick={() => onSelectAlert(selectedAlertId === alert.alert_id ? null : alert.alert_id)}
                style={{ 
                  padding: '12px', 
                  borderRadius: '8px',
                  marginBottom: '8px',
                  cursor: 'pointer',
                  backgroundColor: selectedAlertId === alert.alert_id ? 'rgba(255,255,255,0.1)' : 'transparent',
                  border: '1px solid transparent',
                  borderColor: selectedAlertId === alert.alert_id ? 'var(--border-color)' : 'transparent',
                  transition: 'all 0.2s'
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
                  <span className={getFormatClass(alert.type)} style={{ fontWeight: '600', fontSize: '13px' }}>
                    {formatType(alert.type)}
                  </span>
                  <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                    {formatTime(alert.timestamp)}
                  </span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px' }}>
                  <span>Vessel: {alert.vessel_id.split('_')[1]}</span>
                  <span>Conf: {(alert.confidence * 100).toFixed(0)}%</span>
                </div>
              </div>
              
              {selectedAlertId === alert.alert_id && (
                <AlertDetail alert={alert} getFormatClass={getFormatClass} formatType={formatType} />
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
