import { useState, useEffect, useRef } from 'react';

export function useSSE(url) {
  const [status, setStatus] = useState('CONNECTING');
  const eventSourceRef = useRef(null);
  const backoffRef = useRef(1000);
  const maxBackoff = 10000;
  
  const messageHandlerRef = useRef(null);

  useEffect(() => {
    let timeoutId;
    let isActive = true;

    function connect() {
      if (!isActive) return;
      
      setStatus('CONNECTING');
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.onopen = () => {
        if (!isActive) return;
        setStatus('CONNECTED');
        backoffRef.current = 1000;
      };

      es.onmessage = (event) => {
        if (messageHandlerRef.current) {
          messageHandlerRef.current(event);
        }
      };

      es.onerror = (err) => {
        if (!isActive) return;
        setStatus('DISCONNECTED');
        es.close();
        timeoutId = setTimeout(connect, backoffRef.current);
        backoffRef.current = Math.min(backoffRef.current * 1.5, maxBackoff);
      };
    }

    connect();

    return () => {
      isActive = false;
      clearTimeout(timeoutId);
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, [url]);

  return { status, eventSourceRef, messageHandlerRef };
}
