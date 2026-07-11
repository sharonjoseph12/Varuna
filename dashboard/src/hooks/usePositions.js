import { useEffect, useRef, useState } from 'react';
import { useWebSocket } from './useWebSocket';

export function usePositions(url) {
  const { status, messageHandlerRef } = useWebSocket(url);
  // We use a ref for the Map to avoid React re-renders on every message
  const positionsMap = useRef(new Map());
  const [geoJson, setGeoJson] = useState({
    type: 'FeatureCollection',
    features: []
  });

  useEffect(() => {
    messageHandlerRef.current = (event) => {
      const data = JSON.parse(event.data);
      // Batch incoming positions into the map
      positionsMap.current.set(data.vessel_id, data);
    };
  }, [messageHandlerRef]);

  // Flush Map to GeoJSON once per rAF
  useEffect(() => {
    let animationFrameId;
    
    function flush() {
      const features = [];
      for (const pos of positionsMap.current.values()) {
        features.push({
          type: 'Feature',
          geometry: { type: 'Point', coordinates: [pos.lon, pos.lat] },
          properties: {
            vessel_id: pos.vessel_id,
            heading: pos.heading,
            speed_knots: pos.speed_knots
          }
        });
      }
      
      setGeoJson({
        type: 'FeatureCollection',
        features
      });
      
      animationFrameId = requestAnimationFrame(flush);
    }
    
    animationFrameId = requestAnimationFrame(flush);
    
    return () => cancelAnimationFrame(animationFrameId);
  }, []);

  return { status, geoJson };
}
