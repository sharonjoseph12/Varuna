import { useState } from 'react';
import './App.css';
import { usePositions } from './hooks/usePositions';
import { useAlerts } from './hooks/useAlerts';
import { useMetrics } from './hooks/useMetrics';
import MapView from './components/MapView';
import MetricsPanel from './components/MetricsPanel';
import CasePanel from './components/CasePanel';
import JitterControl from './components/JitterControl';
import Toast from './components/Toast';

const POSITIONS_URL = 'ws://localhost:8080/ws/positions';
const ALERTS_URL = 'ws://localhost:8080/ws/alerts';
const METRICS_URL = 'http://localhost:8080/metrics';

function App() {
  const { status: positionsStatus, geoJson: positionsGeoJson } = usePositions(POSITIONS_URL);
  const { status: alertsStatus, alerts, droppedCount } = useAlerts(ALERTS_URL);
  const metrics = useMetrics(METRICS_URL);
  
  const [selectedAlertId, setSelectedAlertId] = useState(null);
  const [sarVesselId, setSarVesselId] = useState(null);
  
  const isConnected = positionsStatus === 'CONNECTED' && alertsStatus === 'CONNECTED';

  return (
    <div className="app-container">
      {!isConnected && (
        <div className="connecting-overlay">
          Connecting to server...
        </div>
      )}

      <div className="map-layer">
        <MapView 
          positionsGeoJson={positionsGeoJson} 
          alerts={alerts}
          selectedAlertId={selectedAlertId}
          sarVesselId={sarVesselId}
        />
      </div>

      <div className="ui-layer">
        <div className="left-panel">
          <MetricsPanel metrics={metrics} />
        </div>
        
        <div className="right-panel">
          <CasePanel 
            alerts={alerts} 
            selectedAlertId={selectedAlertId}
            onSelectAlert={setSelectedAlertId}
            sarModeEnabled={!!sarVesselId}
            toggleSarMode={(vId) => setSarVesselId(prev => prev === vId ? null : vId)}
          />
        </div>
        
        <div className="debug-control-layer">
          <JitterControl />
        </div>
        
        <div className="toast-container">
          <Toast droppedCount={droppedCount} />
        </div>
      </div>
    </div>
  );
}

export default App;
