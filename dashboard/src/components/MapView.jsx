import { useEffect, useRef } from 'react';
import maplibregl from 'maplibre-gl';
import { zones } from '../utils/zones';

export default function MapView({ positionsGeoJson, alerts, selectedAlertId }) {
  const mapContainer = useRef(null);
  const map = useRef(null);
  
  useEffect(() => {
    if (map.current) return; // initialize map only once
    
    map.current = new maplibregl.Map({
      container: mapContainer.current,
      style: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
      center: [72.0, 15.0], // Indian Ocean / South Asia — live AIS traffic dense here
      zoom: 3,
      interactive: true
    });
    
    map.current.on('load', () => {
      // Add Vessels
      map.current.addSource('vessels', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });
      
      // Circle marker — trust score drives colour; red pulse for low-trust vessels
      map.current.addLayer({
        id: 'vessels-layer',
        type: 'circle',
        source: 'vessels',
        paint: {
          'circle-radius': 4,
          'circle-color': [
            'case',
            ['<', ['get', 'trust_score'], 0.5], '#ef4444',
            ['<=', ['get', 'trust_score'], 0.8], '#f59e0b',
            '#10b981'
          ],
          'circle-stroke-width': 1,
          'circle-stroke-color': 'rgba(0,0,0,0.4)',
          'circle-opacity': [
            'case',
            ['==', ['get', 'is_dark'], true], 0.4,
            1.0
          ]
        }
      });
      
      // Pulsing aura for spoofed / low-trust vessels
      map.current.addLayer({
        id: 'vessels-pulse-layer',
        type: 'circle',
        source: 'vessels',
        filter: ['<', ['get', 'trust_score'], 0.5],
        paint: {
          'circle-radius': 10,
          'circle-color': '#ef4444',
          'circle-opacity': 0.35,
          'circle-pitch-alignment': 'map'
        }
      });
      
      // Add Alert Cones / Dotted Lines
      map.current.addSource('alert-path', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });
      
      map.current.addLayer({
        id: 'alert-path-layer',
        type: 'line',
        source: 'alert-path',
        paint: {
          'line-color': '#FFB347',
          'line-width': 3,
          'line-dasharray': [2, 2] // Dotted line per spec
        }
      });
      
      // For identity conflicts, show two red dots
      map.current.addSource('identity-conflict', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });
      
      map.current.addLayer({
        id: 'identity-conflict-layer',
        type: 'circle',
        source: 'identity-conflict',
        paint: {
          'circle-radius': 8,
          'circle-color': '#E74C3C',
          'circle-stroke-width': 2,
          'circle-stroke-color': '#fff'
        }
      });
    });
  }, []);

  // Update vessels on rAF flush
  useEffect(() => {
    if (!map.current || !map.current.isStyleLoaded()) return;
    const source = map.current.getSource('vessels');
    if (source) {
      source.setData(positionsGeoJson);
    }
  }, [positionsGeoJson]);

  // Animate pulse
  useEffect(() => {
    if (!map.current) return;
    let animationId;
    const animateMarker = (timestamp) => {
      if (map.current.getLayer('vessels-pulse-layer')) {
        const radius = (Math.sin(timestamp / 200) + 1) * 10 + 10;
        const opacity = (Math.sin(timestamp / 200) + 1) * 0.25 + 0.1;
        map.current.setPaintProperty('vessels-pulse-layer', 'circle-radius', radius);
        map.current.setPaintProperty('vessels-pulse-layer', 'circle-opacity', opacity);
      }
      animationId = requestAnimationFrame(animateMarker);
    };
    animationId = requestAnimationFrame(animateMarker);
    return () => cancelAnimationFrame(animationId);
  }, []);
  
  // Update selected alert visualizations
  useEffect(() => {
    if (!map.current || !map.current.isStyleLoaded()) return;
    
    const pathSource = map.current.getSource('alert-path');
    const idSource = map.current.getSource('identity-conflict');
    
    if (!selectedAlertId) {
      if (pathSource) pathSource.setData({ type: 'FeatureCollection', features: [] });
      if (idSource) idSource.setData({ type: 'FeatureCollection', features: [] });
      return;
    }
    
    const alert = alerts.find(a => a.alert_id === selectedAlertId);
    if (!alert) return;
    
    // Draw dotted path or cone for suspected_dark_transit
    if (alert.type === 'suspected_dark_transit') {
      const { lat, lon } = alert.position;
      // Simple projection for demo purposes
      const features = [{
        type: 'Feature',
        geometry: {
          type: 'LineString',
          coordinates: [
            [lon, lat],
            [lon + 0.5, lat + 0.5] // Fake projected path
          ]
        }
      }];
      if (pathSource) pathSource.setData({ type: 'FeatureCollection', features });
      if (idSource) idSource.setData({ type: 'FeatureCollection', features: [] });
    } 
    // Show both positions for identity conflict
    else if (alert.type === 'identity_conflict') {
      const features = [{
        type: 'Feature',
        geometry: { type: 'Point', coordinates: [alert.position.lon, alert.position.lat] }
      }];
      
      if (alert.evidence?.conflicting_position) {
        features.push({
          type: 'Feature',
          geometry: { 
            type: 'Point', 
            coordinates: [alert.evidence.conflicting_position.lon, alert.evidence.conflicting_position.lat] 
          }
        });
      }
      
      if (idSource) idSource.setData({ type: 'FeatureCollection', features });
      if (pathSource) pathSource.setData({ type: 'FeatureCollection', features: [] });
    } else {
      if (pathSource) pathSource.setData({ type: 'FeatureCollection', features: [] });
      if (idSource) idSource.setData({ type: 'FeatureCollection', features: [] });
    }
  }, [selectedAlertId, alerts]);

  return (
    <div ref={mapContainer} style={{ width: '100%', height: '100%' }} />
  );
}
