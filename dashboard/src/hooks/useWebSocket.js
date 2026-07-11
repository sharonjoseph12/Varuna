import { useState, useEffect, useRef } from 'react';

export function useWebSocket(url) {
  const [status, setStatus] = useState('CONNECTING');
  const wsRef = useRef(null);
  const backoffRef = useRef(1000);
  const maxBackoff = 10000;
  
  // Expose a ref to the message handler so we can update it without reconnecting
  const messageHandlerRef = useRef(null);

  useEffect(() => {
    let timeoutId;
    let isActive = true;

    function connect() {
      if (!isActive) return;
      
      setStatus('CONNECTING');
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        if (!isActive) return;
        setStatus('CONNECTED');
        backoffRef.current = 1000; // Reset backoff on success
      };

      ws.onmessage = (event) => {
        if (messageHandlerRef.current) {
          messageHandlerRef.current(event);
        }
      };

      ws.onclose = () => {
        if (!isActive) return;
        setStatus('DISCONNECTED');
        // Exponential backoff
        timeoutId = setTimeout(connect, backoffRef.current);
        backoffRef.current = Math.min(backoffRef.current * 1.5, maxBackoff);
      };

      ws.onerror = (err) => {
        // Error will immediately be followed by close
        console.error("WebSocket error:", err);
      };
    }

    connect();

    return () => {
      isActive = false;
      clearTimeout(timeoutId);
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [url]);

  return { status, wsRef, messageHandlerRef };
}
