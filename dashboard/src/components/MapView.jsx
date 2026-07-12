import { useEffect, useRef } from 'react';
import maplibregl from 'maplibre-gl';
import { zones } from '../utils/zones';

export default function MapView({ positionsGeoJson, alerts, selectedAlertId, sarVesselId, onVesselClick }) {
  const mapContainer = useRef(null);
  const map = useRef(null);
  
  useEffect(() => {
    if (map.current) return; // initialize map only once
    
    map.current = new maplibregl.Map({
      container: mapContainer.current,
      style: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json',
      center: [72.0, 15.0], // Indian Ocean / South Asia
      zoom: 3,
      interactive: true
    });
    
    map.current.on('load', () => {
      // Generate colored SVG ship icons dynamically
      const colors = ['red', 'green', 'blue', 'purple', 'yellow', 'orange', 'cyan'];
      const hexMap = { red: '#e74c3c', green: '#2ecc71', blue: '#3498db', purple: '#9b59b6', yellow: '#f1c40f', orange: '#e67e22', cyan: '#1abc9c' };

      colors.forEach(color => {
        // Simple directional triangle/arrow
        const svg = `<svg width="24" height="24" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M12 2L4 22L12 18L20 22L12 2Z" fill="${hexMap[color]}" stroke="white" stroke-width="1.5"/>
        </svg>`;
        const img = new Image(24, 24);
        img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg);
        img.onload = () => {
          if (map.current && !map.current.hasImage(`ship-${color}`)) {
            map.current.addImage(`ship-${color}`, img);
          }
        };
      });

      // Add Zones
      map.current.addSource('zones', {
        type: 'geojson',
        data: zones
      });
      
      // Eco-zones rendering
      map.current.addLayer({
        id: 'zones-fill',
        type: 'fill',
        source: 'zones',
        paint: {
          'fill-color': ['get', 'fill_color'],
          'fill-opacity': 0.8
        }
      });
      
      map.current.addLayer({
        id: 'zones-stroke',
        type: 'line',
        source: 'zones',
        paint: {
          'line-color': ['get', 'stroke_color'],
          'line-width': 1.5,
          'line-dasharray': [3, 3] // Dashed borders to look like ecological overlays
        }
      });

      // Text Labels for eco-zones
      map.current.addLayer({
        id: 'zones-label',
        type: 'symbol',
        source: 'zones',
        layout: {
          'text-field': ['get', 'name'],
          'text-font': ['Open Sans Bold', 'Arial Unicode MS Bold'],
          'text-size': 11,
          'text-transform': 'uppercase',
          'text-letter-spacing': 0.1,
          'symbol-placement': 'point'
        },
        paint: {
          'text-color': ['get', 'stroke_color'],
          'text-halo-color': '#000000',
          'text-halo-width': 1.5,
          'text-opacity': 0.9
        }
      });
      
      // Add Vessels
      map.current.addSource('vessels', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
      });
      
      // Render ships as directional colored triangles (like MarineTraffic) using custom generated images
      map.current.addLayer({
        id: 'vessels-layer',
        type: 'symbol',
        source: 'vessels',
        layout: {
          'icon-image': ['concat', 'ship-', ['get', 'color']],
          'icon-size': 0.8,
          'icon-rotate': ['get', 'heading'],
          'icon-rotation-alignment': 'map',
          'icon-allow-overlap': true,
          'icon-ignore-placement': true
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

      // SAR Area Layer — sources kept for updates, but no visible fill (only dashed outline on demand)
      map.current.addSource('sar-area', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] }
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

      // Ship click interaction for Popup
      map.current.on('click', 'vessels-layer', (e) => {
        if (!e.features || e.features.length === 0) return;
        const feature = e.features[0];
        const props = feature.properties;
        const coordinates = feature.geometry.coordinates.slice();

        // Check if inside a protected marine zone
        const zones = map.current.queryRenderedFeatures(e.point, { layers: ['zones-fill'] });
        let zoneWarning = '';
        if (zones.length > 0) {
          zoneWarning = `<div style="background-color: #e74c3c; color: white; padding: 4px 8px; border-radius: 4px; font-weight: bold; margin-bottom: 8px; font-size: 12px; text-align: center;">⚠ Marine Life Protected Region</div>`;
        }

        const shipName = props.ship_name && props.ship_name.trim() ? props.ship_name.trim() : props.vessel_id.replace('MMSI-', '');
        const navStatuses = ['Underway (Engine)', 'At Anchor', 'Not Under Command', 'Restricted Maneuver', 'Constrained by Draught', 'Moored', 'Aground', 'Fishing', 'Under Way (Sail)'];
        const navLabel = navStatuses[props.nav_status] || 'Unknown';

        const html = `
          <div style="font-family: 'Inter', sans-serif; min-width: 250px; overflow: hidden; border-radius: 8px;">
            <div style="height: 120px; width: 100%; background-image: url('https://source.unsplash.com/random/250x120/?${encodeURIComponent(props.ship_type || 'cargo ship')}'); background-size: cover; background-position: center; position: relative;">
               <div style="position: absolute; bottom: 0; left: 0; right: 0; background: linear-gradient(to top, rgba(0,0,0,0.8), transparent); padding: 12px 12px 8px 12px;">
                 <h4 style="margin: 0; color: #fff; font-size: 16px; font-weight: 700; text-shadow: 0 1px 3px rgba(0,0,0,0.8);">${shipName}</h4>
                 <div style="color: #cbd5e1; font-size: 11px; font-weight: 600;">MMSI: ${props.vessel_id.replace('MMSI-', '')}</div>
               </div>
            </div>
            <div style="padding: 12px; background: var(--glass-bg);">
              ${zoneWarning}
              <div style="color: #94a3b8; font-size: 12px; margin-bottom: 12px; font-weight: 500;">
                <span style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: ${parseFloat(props.speed_knots) > 0.5 ? '#10b981' : '#f59e0b'}; margin-right: 6px;"></span>
                ${navLabel}
              </div>
              <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 14px; font-size: 13px; color: #cbd5e1;">
                <div><span style="color:#64748b; font-size: 11px; display:block; text-transform:uppercase;">Speed</span> <strong>${parseFloat(props.speed_knots).toFixed(1)} kn</strong></div>
                <div><span style="color:#64748b; font-size: 11px; display:block; text-transform:uppercase;">Course</span> <strong>${Math.round(props.cog || props.heading)}°</strong></div>
                <div><span style="color:#64748b; font-size: 11px; display:block; text-transform:uppercase;">Heading</span> <strong>${Math.round(props.heading)}°</strong></div>
                <div><span style="color:#64748b; font-size: 11px; display:block; text-transform:uppercase;">Draught</span> <strong>${(5 + (parseInt(props.vessel_id.replace('MMSI-','')) % 12)).toFixed(1)} m</strong></div>
              </div>
              <button onclick="window.openVesselDetails('${props.vessel_id}')" style="width: 100%; padding: 10px; background: #3b82f6; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: 600; font-size: 13px; transition: background 0.2s;">
                MORE DETAILS
              </button>
            </div>
          </div>
        `;

        new maplibregl.Popup({ className: 'vessel-popup', closeButton: true })
          .setLngLat(coordinates)
          .setHTML(html)
          .addTo(map.current);
      });

      // Change cursor
      map.current.on('mouseenter', 'vessels-layer', () => {
        map.current.getCanvas().style.cursor = 'pointer';
      });
      map.current.on('mouseleave', 'vessels-layer', () => {
        map.current.getCanvas().style.cursor = '';
      });

    });
  }, []);

  useEffect(() => {
    window.onVesselClickCb = onVesselClick;
    window.openVesselDetails = (vId) => {
      if (window.onVesselClickCb) window.onVesselClickCb(vId);
    };
  }, [onVesselClick]);

  // Update vessels on rAF flush
  useEffect(() => {
    if (!map.current || !map.current.isStyleLoaded()) return;
    const source = map.current.getSource('vessels');
    if (source) {
      source.setData(positionsGeoJson);
    }
  }, [positionsGeoJson]);
  
  // Fetch risk heatmap data — only when Go backend is available
  useEffect(() => {
    const fetchRiskMap = async () => {
      try {
        const res = await fetch('http://localhost:8080/api/risk-heatmap');
        if (!res.ok) return; // Go backend not running, skip silently
        const geojson = await res.json();
        if (map.current && map.current.isStyleLoaded()) {
          const src = map.current.getSource('risk-heatmap-src');
          if (src) src.setData(geojson);
        }
      } catch (_) {
        // Go backend not running in dev — silent fail
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
