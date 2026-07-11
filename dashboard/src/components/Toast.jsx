import { useEffect, useState } from 'react';

export default function Toast({ droppedCount }) {
  const [visible, setVisible] = useState(false);
  
  useEffect(() => {
    if (droppedCount > 0) {
      setVisible(true);
      const timer = setTimeout(() => setVisible(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [droppedCount]);

  if (!visible) return null;

  return (
    <div style={{
      background: '#E74C3C',
      color: 'white',
      padding: '8px 16px',
      borderRadius: '20px',
      fontSize: '13px',
      fontWeight: '600',
      boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
      animation: 'slideUp 0.3s ease-out'
    }}>
      ⚠️ High volume: Dropped {droppedCount} alerts. Resynced.
      <style>{`
        @keyframes slideUp {
          from { transform: translateY(20px); opacity: 0; }
          to { transform: translateY(0); opacity: 1; }
        }
      `}</style>
    </div>
  );
}
