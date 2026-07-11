import { useState } from 'react';

export default function JitterControl() {
  const [position, setPosition] = useState(50);
  
  // In a real implementation this would send a WS message to the backend
  // to force a vessel position update. For this demo we just show the UI control.
  const handleChange = (e) => {
    setPosition(e.target.value);
  };

  return (
    <div className="glass-panel" style={{ padding: '12px', width: '250px' }}>
      <div style={{ fontSize: '12px', fontWeight: '600', marginBottom: '8px', color: 'var(--text-secondary)' }}>
        DEMO: Boundary Jitter
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <span style={{ fontSize: '11px' }}>Out</span>
        <input 
          type="range" 
          min="0" 
          max="100" 
          value={position} 
          onChange={handleChange}
          style={{ flexGrow: 1 }}
        />
        <span style={{ fontSize: '11px' }}>In</span>
      </div>
      <div style={{ fontSize: '10px', color: 'var(--text-secondary)', marginTop: '8px', textAlign: 'center' }}>
        Slide to move vessel across zone edge
      </div>
    </div>
  );
}
