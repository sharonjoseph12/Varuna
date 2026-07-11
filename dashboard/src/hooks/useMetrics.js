import { useState, useEffect } from 'react';

export function useMetrics(url) {
  const [metrics, setMetrics] = useState({
    throughput_per_sec: 0,
    p50_latency_ms: 0,
    p99_latency_ms: 0,
    history: []
  });

  useEffect(() => {
    let intervalId;
    
    async function fetchMetrics() {
      try {
        const res = await fetch(url);
        const data = await res.json();
        
        setMetrics(prev => {
          const newHistory = [...prev.history, data];
          if (newHistory.length > 60) newHistory.shift(); // Keep last 60 points
          
          return {
            ...data,
            history: newHistory
          };
        });
      } catch (err) {
        // Silent fail on polling errors to avoid console spam
      }
    }
    
    fetchMetrics();
    intervalId = setInterval(fetchMetrics, 1500);
    
    return () => clearInterval(intervalId);
  }, [url]);

  return metrics;
}
