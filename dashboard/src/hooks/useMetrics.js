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
        
        const mappedData = {
          throughput_per_sec: data.throughput_msgs_sec || 0,
          p50_latency_ms: data.latency_p50_ms || 0,
          p99_latency_ms: data.latency_p99_ms || 0
        };
        
        setMetrics(prev => {
          const newHistory = [...prev.history, mappedData];
          if (newHistory.length > 60) newHistory.shift(); // Keep last 60 points
          
          return {
            ...mappedData,
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
