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
    if (status === 'CONNECTED') {
      positionsMap.current.clear();
      setGeoJson({ type: 'FeatureCollection', features: [] });
    }
  }, [status]);

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
      const colors = ['red', 'green', 'blue', 'purple', 'yellow', 'orange', 'cyan'];
      function getColorForId(id) {
        let hash = 0;
        const str = String(id);
        for (let i = 0; i < str.length; i++) {
          hash = str.charCodeAt(i) + ((hash << 5) - hash);
        }
        return colors[Math.abs(hash) % colors.length];
      }

      const features = [];
      for (const pos of positionsMap.current.values()) {
        features.push({
          type: 'Feature',
          geometry: { type: 'Point', coordinates: [pos.lon, pos.lat] },
          properties: {
            vessel_id: pos.vessel_id,
            heading: pos.heading || 0,
            speed_knots: pos.speed_knots,
            ship_name: pos.ship_name || '',
            nav_status: pos.nav_status ?? -1,
            cog: pos.cog || 0,
            trust_score: pos.trust_score ?? 1.0,
            color: getColorForId(pos.vessel_id)
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
