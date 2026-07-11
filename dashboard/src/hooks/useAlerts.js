import { useState, useEffect } from 'react';
import { useSSE } from './useSSE';

export function useAlerts(url) {
  const { status, messageHandlerRef } = useSSE(url);
  const [alerts, setAlerts] = useState([]);
  const [droppedCount, setDroppedCount] = useState(0);

  useEffect(() => {
    messageHandlerRef.current = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'alerts_dropped') {
        setDroppedCount(prev => prev + data.count);
        return;
      }
      
      setAlerts(prev => {
        const next = [data, ...prev];
        if (next.length > 100) next.length = 100;
        return next;
      });
    };
  }, [messageHandlerRef]);

  return { status, alerts, droppedCount };
}
