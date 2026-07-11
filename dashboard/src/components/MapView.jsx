import { useEffect, useRef } from 'react';
import maplibregl from 'maplibre-gl';
import { zones } from '../utils/zones';

export default function MapView({ positionsGeoJson, alerts, selectedAlertId, sarVesselId }) {
  const mapContainer = useRef(null);
  const map = useRef(null);
  
  useEffect(() => {
    if (map.current) return; // initialize map only once
    
    map.current = new maplibregl.Map({
      container: mapContainer.current,
      style: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
      center: [-72.5, 40.0],
      zoom: 6,
      interactive: true
    });
    
    map.current.on('load', () => {
      // Add Zones
      map.current.addSource('zones', {
        type: 'geojson',
        data: zones
      });
      
      // Zones rendering removed per user request
      
      // Add Vessels
      map.current.addSource('vessels', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });
      
      // Changed from symbol to circle to ensure it renders without needing an external sprite
      map.current.addLayer({
        id: 'vessels-layer',
        type: 'circle',
        source: 'vessels',
        paint: {
          'circle-radius': 4,
          'circle-color': '#ffffff',
          'circle-stroke-width': 1,
          'circle-stroke-color': '#000000'
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
      
      // Risk Heatmap Layer (MapLibre native)
      map.current.addSource('risk-heatmap-src', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });

      map.current.addLayer({
        id: 'risk-heatmap-layer',
        type: 'heatmap',
        source: 'risk-heatmap-src',
        paint: {
          'heatmap-weight': ['get', 'weight'],
          'heatmap-intensity': 1,
          'heatmap-color': [
            'interpolate', ['linear'], ['heatmap-density'],
            0, 'rgba(33,102,172,0)',
            0.2, 'rgb(103,169,207)',
            0.4, 'rgb(209,229,240)',
            0.6, 'rgb(253,219,199)',
            0.8, 'rgb(239,138,98)',
            1, 'rgb(178,24,43)'
          ],
          'heatmap-radius': 30,
          'heatmap-opacity': 0.6
        }
      });

      // Dark Ship Probability Heatmap (Polygons)
      map.current.addSource('dark-heatmap-src', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });

      map.current.addLayer({
        id: 'dark-heatmap-layer',
        type: 'fill',
        source: 'dark-heatmap-src',
        paint: {
          'fill-color': '#FFB347',
          'fill-opacity': ['*', 0.8, ['get', 'weight']]
        }
      });

      // SAR Area Layer
      map.current.addSource('sar-area', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });

      map.current.addLayer({
        id: 'sar-area-fill',
        type: 'fill',
        source: 'sar-area',
        paint: {
          'fill-color': '#e74c3c',
          'fill-opacity': 0.15
        }
      });
      
      map.current.addLayer({
        id: 'sar-area-line',
        type: 'line',
        source: 'sar-area',
        paint: {
          'line-color': '#e74c3c',
          'line-width': 2,
          'line-dasharray': [2, 4]
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
  
  // Fetch risk heatmap data
  useEffect(() => {
    const fetchRiskMap = async () => {
      try {
        const res = await fetch('http://localhost:8080/api/risk-heatmap');
        if (res.ok) {
          const geojson = await res.json();
          if (map.current && map.current.isStyleLoaded()) {
            const src = map.current.getSource('risk-heatmap-src');
            if (src) src.setData(geojson);
          }
        }
      } catch (err) {
        console.error("Failed to fetch risk heatmap", err);
      }
    };
    fetchRiskMap();
    const int = setInterval(fetchRiskMap, 5000);
    return () => clearInterval(int);
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
    
    // Draw dotted path or dark ship heatmap for suspected_dark_transit
    if (alert.type === 'suspected_dark_transit') {
      const { lat, lon } = alert.position;
      
      // Fetch ML heatmap
      fetch(`http://localhost:8080/api/heatmap/${alert.vessel_id}`)
        .then(r => r.json())
        .then(geoJson => {
          const src = map.current.getSource('dark-heatmap-src');
          if (src && !geoJson.error) src.setData(geoJson);
        })
        .catch(console.error);

      // Simple projection line for demo
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
      
      const darkSrc = map.current.getSource('dark-heatmap-src');
      if (darkSrc) darkSrc.setData({ type: 'FeatureCollection', features: [] });
    }
  }, [selectedAlertId, alerts]);

  // SAR Mode Effect
  useEffect(() => {
    if (!map.current || !map.current.isStyleLoaded()) return;
    const sarSrc = map.current.getSource('sar-area');
    if (!sarSrc) return;

    if (!sarVesselId) {
      sarSrc.setData({ type: 'FeatureCollection', features: [] });
      return;
    }

    const fetchSar = async () => {
      try {
        const res = await fetch(`http://localhost:8080/api/sar/${sarVesselId}`);
        if (res.ok) {
          const sar = await res.json();
          // Generate a rough circle polygon for GeoJSON (MapLibre doesn't have a native circle geometry for fills)
          const points = 64;
          const coords = [];
          // Rough approximation: 1 deg lat ~= 111km, 1 deg lon ~= 111km * cos(lat)
          const latRadius = sar.radius_km / 111.0;
          const lonRadius = sar.radius_km / (111.0 * Math.cos(sar.drift_lat * Math.PI / 180));
          
          for (let i = 0; i <= points; i++) {
            const angle = (i * 360 / points) * Math.PI / 180;
            coords.push([
              sar.drift_lon + lonRadius * Math.cos(angle),
              sar.drift_lat + latRadius * Math.sin(angle)
            ]);
          }
          
          sarSrc.setData({
            type: 'FeatureCollection',
            features: [{
              type: 'Feature',
              geometry: { type: 'Polygon', coordinates: [coords] },
              properties: { type: 'sar_area' }
            }]
          });
        }
      } catch (err) {
        console.error("Failed to fetch SAR data", err);
      }
    };
    
    fetchSar();
    const int = setInterval(fetchSar, 2000);
    return () => clearInterval(int);
  }, [sarVesselId]);

  return (
    <div ref={mapContainer} style={{ width: '100%', height: '100%' }} />
  );
}
