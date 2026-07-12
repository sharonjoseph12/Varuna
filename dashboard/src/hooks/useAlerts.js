import { useState, useEffect } from 'react';
import { useWebSocket } from './useWebSocket';

export function useAlerts(url) {
  const { status, messageHandlerRef } = useWebSocket(url);
  const [alerts, setAlerts] = useState([]);
  const [droppedCount, setDroppedCount] = useState(0);

  useEffect(() => {
    if (status === 'CONNECTED') {
      setAlerts([]);
    }
  }, [status]);

  useEffect(() => {
    messageHandlerRef.current = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'alerts_dropped') {
        setDroppedCount(prev => prev + data.count);
        // Toast component will watch droppedCount
        return;
      }
      
      setAlerts(prev => {
        // Prepend new alert, cap at 100 for UI performance
        const next = [data, ...prev];
        if (next.length > 100) next.length = 100;
        return next;
      });
    };
  }, [messageHandlerRef]);

  return { status, alerts, droppedCount };
}
