import { Line } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
);

export default function MetricsPanel({ metrics }) {
  const chartData = {
    labels: metrics.history.map((_, i) => i),
    datasets: [
      {
        label: 'p50 Latency (ms)',
        data: metrics.history.map(m => m.p50_latency_ms),
        borderColor: '#94a3b8',
        backgroundColor: '#94a3b8',
        tension: 0.4,
        pointRadius: 0
      },
      {
        label: 'p99 Latency (ms)',
        data: metrics.history.map(m => m.p99_latency_ms),
        borderColor: '#FF6B6B',
        backgroundColor: '#FF6B6B',
        tension: 0.4,
        pointRadius: 0
      }
    ]
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    plugins: {
      legend: { position: 'bottom', labels: { color: '#f8fafc' } }
    },
    scales: {
      x: { display: false },
      y: { 
        beginAtZero: true,
        grid: { color: 'rgba(255,255,255,0.1)' },
        ticks: { color: '#94a3b8' }
      }
    }
  };

  return (
    <div className="glass-panel metrics-container">
      <h2 style={{ fontSize: '14px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '8px' }}>
        System Throughput
      </h2>
      <div style={{ fontSize: '36px', fontWeight: '700', marginBottom: '24px' }}>
        {metrics.throughput_per_sec.toLocaleString()} <span style={{ fontSize: '14px', fontWeight: '400', color: 'var(--text-secondary)' }}>msgs/sec</span>
      </div>
      
      <h2 style={{ fontSize: '14px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '8px' }}>
        End-to-End Latency
      </h2>
      <div style={{ height: '150px' }}>
        <Line data={chartData} options={chartOptions} />
      </div>
    </div>
  );
}
