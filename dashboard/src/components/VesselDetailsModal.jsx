import { useState, useEffect } from 'react';

// Deterministic mock data derived from MMSI so the same ship always shows the same info
const SHIP_TYPES = ['Bulk Carrier', 'Crude Oil Tanker', 'Container Ship', 'General Cargo', 'Chemical Tanker', 'LNG Tanker', 'Fishing Vessel', 'Passenger Ship', 'Vehicle Carrier', 'Tug'];
const FLAGS = [
  { name: 'PANAMA', code: 'PA', emoji: '🇵🇦' },
  { name: 'LIBERIA', code: 'LR', emoji: '🇱🇷' },
  { name: 'MARSHALL ISLANDS', code: 'MH', emoji: '🇲🇭' },
  { name: 'INDIA', code: 'IN', emoji: '🇮🇳' },
  { name: 'SINGAPORE', code: 'SG', emoji: '🇸🇬' },
  { name: 'CHINA', code: 'CN', emoji: '🇨🇳' },
  { name: 'HONG KONG', code: 'HK', emoji: '🇭🇰' },
  { name: 'BAHAMAS', code: 'BS', emoji: '🇧🇸' },
  { name: 'CAMEROON', code: 'CM', emoji: '🇨🇲' },
  { name: 'GREECE', code: 'GR', emoji: '🇬🇷' },
];
const SHIP_NAMES = ['NOSU', 'STELLAR WIND', 'OCEAN GRACE', 'PACIFIC VOYAGER', 'ARABIAN PEARL', 'BLUE HORIZON', 'GOLDEN EAGLE', 'MONSOON STAR', 'CORAL QUEEN', 'MUMBAI EXPRESS', 'JADE FORTUNE', 'SEA BREEZE', 'TITAN GLORY', 'NEPTUNE WAVE', 'HARBOR SPIRIT'];
const PORTS = ['Mumbai, India', 'Colombo, Sri Lanka', 'Dubai, UAE', 'Singapore', 'Port Said, Egypt', 'Jeddah, Saudi Arabia', 'Karachi, Pakistan', 'Mombasa, Kenya', 'Chennai, India', 'Salalah, Oman'];

function hashMMSI(mmsi) {
  let h = 0;
  const s = String(mmsi);
  for (let i = 0; i < s.length; i++) {
    h = s.charCodeAt(i) + ((h << 5) - h);
  }
  return Math.abs(h);
}

function getVesselMeta(vesselId) {
  const mmsi = vesselId.replace('MMSI-', '');
  const h = hashMMSI(mmsi);
  const shipType = SHIP_TYPES[h % SHIP_TYPES.length];
  const flag = FLAGS[h % FLAGS.length];
  const name = SHIP_NAMES[h % SHIP_NAMES.length];
  const imo = 9000000 + (h % 999999);
  const callSign = String.fromCharCode(65 + (h % 26)) + String.fromCharCode(65 + ((h >> 3) % 26)) + String.fromCharCode(65 + ((h >> 6) % 26)) + String(h % 999).padStart(3, '0');
  const loa = 100 + (h % 200);
  const beam = 15 + (h % 35);
  const draught = 5 + (h % 12);
  const departure = PORTS[h % PORTS.length];
  const destination = PORTS[(h + 3) % PORTS.length];
  return { mmsi, name, shipType, flag, imo, callSign, loa, beam, draught, departure, destination };
}

export default function VesselDetailsModal({ vesselId, vesselData, onClose }) {
  const [tab, setTab] = useState('summary');
  const [trustScore, setTrustScore] = useState(null);
  const meta = getVesselMeta(vesselId);

  // Fetch trust score
  useEffect(() => {
    fetch(`http://localhost:8080/api/vessel/${vesselId}`)
      .then(r => r.json())
      .then(data => {
        if (data && data.trust_score !== undefined) {
          setTrustScore(data.trust_score);
        }
      })
      .catch(err => console.error("Failed to fetch trust score", err));
  }, [vesselId]);

  const speed = vesselData?.speed_knots ?? 0;
  const heading = vesselData?.heading ?? 0;
  const lat = vesselData?.lat ?? 0;
  const lon = vesselData?.lon ?? 0;

  return (
    <div className="vessel-modal-overlay" onClick={onClose}>
      <div className="vessel-modal" onClick={e => e.stopPropagation()}>
        {/* Header with Image */}
        <div className="vessel-modal-header" style={{
          backgroundImage: `linear-gradient(to top, rgba(15,23,42,1), rgba(15,23,42,0.4)), url('https://source.unsplash.com/random/800x400/?${encodeURIComponent(meta.shipType || 'cargo ship')}')`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          padding: '24px 20px 20px',
          borderTopLeftRadius: '12px',
          borderTopRightRadius: '12px',
          position: 'relative'
        }}>
          <button className="vessel-modal-close" onClick={onClose} style={{ position: 'absolute', top: '16px', right: '16px', background: 'rgba(0,0,0,0.5)', border: 'none', color: 'white', width: '32px', height: '32px', borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '18px' }}>✕</button>
          <div className="vessel-modal-title" style={{ display: 'flex', gap: '16px', alignItems: 'flex-end', marginTop: '60px' }}>
            <span className="vessel-flag" style={{ fontSize: '32px', filter: 'drop-shadow(0 2px 4px rgba(0,0,0,0.5))' }}>{meta.flag.emoji}</span>
            <div>
              <h2 style={{ margin: '0 0 4px 0', fontSize: '24px', color: '#fff', textShadow: '0 2px 4px rgba(0,0,0,0.8)' }}>{meta.name}</h2>
              <span className="vessel-type-badge" style={{ background: 'rgba(59,130,246,0.8)', padding: '4px 10px', borderRadius: '4px', fontSize: '12px', fontWeight: '600', color: 'white', textTransform: 'uppercase', letterSpacing: '0.5px' }}>{meta.shipType}</span>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="vessel-modal-tabs">
          <button className={tab === 'summary' ? 'active' : ''} onClick={() => setTab('summary')}>Summary</button>
          <button className={tab === 'general' ? 'active' : ''} onClick={() => setTab('general')}>General</button>
          <button className={tab === 'ais' ? 'active' : ''} onClick={() => setTab('ais')}>AIS Info</button>
        </div>

        {/* Content */}
        <div className="vessel-modal-body">
          {tab === 'summary' && (
            <div className="vessel-summary">
              <p className="vessel-summary-text">
                <strong>Where is the ship?</strong><br />
                {meta.shipType} <strong>{meta.name}</strong> is currently located at {lat.toFixed(4)}°N, {lon.toFixed(4)}°E (reported just now).
              </p>
              <p className="vessel-summary-text">
                <strong>What kind of ship is this?</strong><br />
                {meta.name} (IMO: {meta.imo}) is a <strong>{meta.shipType}</strong> sailing under the flag of <strong>{meta.flag.emoji} {meta.flag.name}</strong>. Her length overall (LOA) is {meta.loa} meters and her width is {meta.beam} meters.
              </p>

              {/* Voyage bar */}
              <div className="vessel-voyage">
                <div className="voyage-endpoints">
                  <span>{meta.departure}</span>
                  <span>{meta.destination}</span>
                </div>
                <div className="voyage-bar">
                  <div className="voyage-progress" style={{ width: `${40 + (hashMMSI(meta.mmsi) % 50)}%` }}></div>
                </div>
                <div className="voyage-buttons">
                  <button className="btn-outline">⟵ Past track</button>
                  <button className="btn-outline">🔮 Route forecast</button>
                </div>
              </div>

              {/* Trust Score Box */}
              {trustScore !== null && (
                <div style={{ marginTop: '16px', background: 'rgba(15,23,42,0.8)', padding: '16px', borderRadius: '12px', border: '1px solid rgba(255,255,255,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <strong style={{ color: 'white' }}>Compliance Probability</strong>
                    <strong style={{ 
                      fontSize: '24px', 
                      color: trustScore > 0.8 ? '#10b981' : trustScore > 0.5 ? '#f59e0b' : '#ef4444' 
                    }}>
                      {(trustScore * 100).toFixed(0)}%
                    </strong>
                  </div>
                  <p style={{ margin: '8px 0 0 0', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    Based on recent historical tracks, AIS transmission consistency, and geographical constraints.
                  </p>
                </div>
              )}
            </div>
          )}

          {tab === 'general' && (
            <div className="vessel-info-grid">
              <div className="info-row"><span>Name</span><strong>{meta.name}</strong></div>
              <div className="info-row"><span>Flag</span><strong>{meta.flag.emoji} {meta.flag.name}</strong></div>
              <div className="info-row"><span>IMO</span><strong>{meta.imo}</strong></div>
              <div className="info-row"><span>MMSI</span><strong>{meta.mmsi}</strong></div>
              <div className="info-row"><span>Call Sign</span><strong>{meta.callSign}</strong></div>
              <div className="info-row"><span>AIS Transponder</span><strong>Class A</strong></div>
              <div className="info-row"><span>Vessel Type</span><strong>{meta.shipType}</strong></div>
              <div className="info-row"><span>Length (LOA)</span><strong>{meta.loa} m</strong></div>
              <div className="info-row"><span>Beam</span><strong>{meta.beam} m</strong></div>
              <div className="info-row"><span>Draught</span><strong>{meta.draught}.0 m</strong></div>
            </div>
          )}

          {tab === 'ais' && (
            <div className="vessel-info-grid">
              <div className="info-row"><span>Navigational Status</span><strong>{speed > 0.5 ? 'Underway Using Engine' : 'At Anchor'}</strong></div>
              <div className="info-row"><span>Position Received</span><strong>Just now</strong></div>
              <div className="info-row"><span>Latitude</span><strong>{lat.toFixed(5)}</strong></div>
              <div className="info-row"><span>Longitude</span><strong>{lon.toFixed(5)}</strong></div>
              <div className="info-row"><span>Speed</span><strong>{speed.toFixed(1)} kn</strong></div>
              <div className="info-row"><span>Course</span><strong>{heading}°</strong></div>
              <div className="info-row"><span>True Heading</span><strong>{heading}°</strong></div>
              <div className="info-row"><span>Rate of Turn</span><strong>0 °/min</strong></div>
              <div className="info-row"><span>Reported Destination</span><strong>{meta.destination}</strong></div>
              <div className="info-row"><span>AIS Vessel Type</span><strong>{meta.shipType}</strong></div>
              <div className="info-row"><span>AIS Source</span><strong>Terrestrial</strong></div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
