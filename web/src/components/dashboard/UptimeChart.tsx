import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../hooks/useAuth';
import { setAuthToken } from '../../lib/axios';
import { healthService } from '../../services/healthService';

interface UptimeChartProps {
  target: string;
}

const UptimeChart: React.FC<UptimeChartProps> = ({ target }) => {
  const { getAccessToken, isAuthenticated } = useAuth();

  const { data: history, isLoading, isError } = useQuery({
    queryKey: ['healthHistory', target],
    queryFn: async () => {
      const token = await getAccessToken();
      setAuthToken(token);
      return healthService.getTargetHistory(target, 30);
    },
    enabled: isAuthenticated && !!target,
    refetchInterval: 30000, // Refresh every 30s
  });

  if (isLoading) {
    return (
      <div className="h-10 flex items-center justify-center">
        <div className="w-4 h-4 border-2 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin" />
      </div>
    );
  }

  if (isError || !history) {
    return <div className="text-[10px] text-red-400">Failed to load history</div>;
  }

  // Find max latency for visual scaling (min 100ms to avoid divide by zero/flat lines)
  const maxLatency = Math.max(...history.map(h => h.latency_ms), 100);

  // Build the SVG path for a sparkline
  const width = 240;
  const height = 32;
  const points = history.map((item, index) => {
    const x = history.length > 1 ? (index / (history.length - 1)) * width : width / 2;
    // Invert Y because SVG 0 is at the top, max height is at the bottom
    // Scale latency to fit the height
    const y = height - (item.status === 'down' ? 0 : (item.latency_ms / maxLatency) * (height - 8) + 4);
    return `${x},${y}`;
  });

  const pathD = points.length > 0 ? `M ${points.join(' L ')}` : '';
  const safeId = target.replace(/[^a-zA-Z0-9]/g, '');

  return (
    <div className="space-y-2 mt-4 pt-4 border-t border-slate-800/50">
      <div className="flex items-center justify-between text-[10px] text-slate-500 font-medium">
        <span>Latency History (last 30 pings)</span>
        <span>Max: {maxLatency}ms</span>
      </div>

      <div className="flex items-center gap-4">
        {/* Latency Sparkline */}
        <div className="relative bg-slate-950/40 rounded-lg p-1.5 border border-slate-800/40 flex-1 h-10 flex items-center">
          {history.length > 0 ? (
            <svg width="100%" height="100%" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="overflow-visible">
              <defs>
                <linearGradient id={`gradient-${safeId}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#818cf8" stopOpacity="0.4" />
                  <stop offset="100%" stopColor="#818cf8" stopOpacity="0.0" />
                </linearGradient>
              </defs>
              {/* Fill Area */}
              <path
                d={`${pathD} L ${width},${height} L 0,${height} Z`}
                fill={`url(#gradient-${safeId})`}
              />
              {/* Line */}
              <path
                d={pathD}
                fill="none"
                stroke="#6366f1"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          ) : (
            <span className="text-[10px] text-slate-600 mx-auto">No history data</span>
          )}
        </div>

        {/* State Bar Row (Ticks) */}
        <div className="flex gap-[2px] h-6 items-end">
          {history.slice(-15).map((item, idx) => (
            <div
              key={idx}
              className={`w-[4px] rounded-full transition-all duration-300 ${
                item.status === 'up'
                  ? item.latency_ms > 500
                    ? 'bg-amber-500 h-4'
                    : 'bg-emerald-500 h-5'
                  : 'bg-red-500 h-6 animate-pulse'
              }`}
              title={`${new Date(item.checked_at).toLocaleTimeString()}: ${item.status === 'up' ? `${item.latency_ms}ms` : 'DOWN'}`}
            />
          ))}
          {history.length === 0 && (
            <div className="flex gap-[2px] h-6 items-end">
              {Array.from({ length: 15 }).map((_, idx) => (
                <div key={idx} className="w-[4px] h-2 bg-slate-800 rounded-full" />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default UptimeChart;
