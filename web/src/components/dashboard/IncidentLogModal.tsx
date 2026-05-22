import React, { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { X, Clock, Activity, CheckCircle2, XCircle, AlertCircle, ExternalLink, Shield } from 'lucide-react';
import { healthService } from '../../services/healthService';
import { useAuth } from '../../hooks/useAuth';
import { getEnv } from '../../config/env';

interface IncidentLogModalProps {
  target: string;
  name: string;
  onClose: () => void;
}

const IncidentLogModal: React.FC<IncidentLogModalProps> = ({ target, name, onClose }) => {
  const { getAccessToken } = useAuth();
  const [rawLogsUrl, setRawLogsUrl] = useState('');

  // Fetch history for the target
  const { data: history, isLoading, error } = useQuery({
    queryKey: ['targetHistory', target, 100],
    queryFn: () => healthService.getTargetHistory(target, 100),
    enabled: !!target,
  });

  useEffect(() => {
    const fetchTokenAndSetUrl = async () => {
      try {
        const token = await getAccessToken();
        const url = `${getEnv('VITE_API_URL')}/api/history?target=${encodeURIComponent(target)}&limit=100&token=${encodeURIComponent(token)}`;
        setRawLogsUrl(url);
      } catch (err) {
        console.error('Failed to get access token for raw logs url:', err);
      }
    };
    fetchTokenAndSetUrl();
  }, [target, getAccessToken]);

  // Calculations
  const sortedChecks = history
    ? [...history].sort((a, b) => new Date(a.checked_at).getTime() - new Date(b.checked_at).getTime())
    : [];

  // Compute average latency
  const validLatencyChecks = sortedChecks.filter(c => c.status !== 'pending' && c.latency_ms !== undefined);
  const avgLatency = validLatencyChecks.length > 0
    ? Math.round(validLatencyChecks.reduce((sum, c) => sum + c.latency_ms, 0) / validLatencyChecks.length)
    : 0;

  // Compute Incident Duration Timeline
  const incidents: Array<{
    start: string;
    end: string | null;
    durationMs: number | null;
  }> = [];

  let currentOutageStart: Date | null = null;
  for (const check of sortedChecks) {
    if (check.status === 'down') {
      if (!currentOutageStart) {
        currentOutageStart = new Date(check.checked_at);
      }
    } else if (check.status === 'up') {
      if (currentOutageStart) {
        const end = new Date(check.checked_at);
        incidents.push({
          start: currentOutageStart.toISOString(),
          end: end.toISOString(),
          durationMs: end.getTime() - currentOutageStart.getTime(),
        });
        currentOutageStart = null;
      }
    }
  }
  if (currentOutageStart) {
    incidents.push({
      start: currentOutageStart.toISOString(),
      end: null,
      durationMs: null, // ongoing
    });
  }
  incidents.reverse(); // Newest first

  // Compute Latency Histogram
  const buckets = [
    { label: '0-50ms', min: 0, max: 50, count: 0, color: 'bg-emerald-500' },
    { label: '50-100ms', min: 50, max: 100, count: 0, color: 'bg-emerald-400' },
    { label: '100-250ms', min: 100, max: 250, count: 0, color: 'bg-amber-400' },
    { label: '250-500ms', min: 250, max: 500, count: 0, color: 'bg-amber-500' },
    { label: '500-1000ms', min: 500, max: 1000, count: 0, color: 'bg-orange-500' },
    { label: '1000ms+', min: 1000, max: Infinity, count: 0, color: 'bg-red-500' },
  ];

  validLatencyChecks.forEach(c => {
    const lat = c.latency_ms;
    for (const b of buckets) {
      if (lat >= b.min && lat < b.max) {
        b.count++;
        break;
      }
    }
  });

  const totalValidChecks = validLatencyChecks.length;
  const maxBucketCount = Math.max(...buckets.map(b => b.count), 1);

  // Format Duration helper
  const formatDuration = (ms: number | null) => {
    if (ms === null) return 'Ongoing';
    const seconds = Math.floor(ms / 1000);
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 md:p-6 backdrop-blur-md bg-slate-950/60 transition-all duration-300">
      <div className="relative w-full max-w-5xl bg-bg-card border border-border-primary rounded-3xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh] text-text-primary animate-in fade-in zoom-in-95 duration-300">
        
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border-primary/80">
          <div className="flex items-center gap-3">
            <div className="p-2.5 bg-indigo-500/10 rounded-xl">
              <Activity className="w-6 h-6 text-indigo-500 dark:text-indigo-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold tracking-tight">{name || 'Service Details'}</h2>
              <p className="text-xs text-text-secondary/70 font-mono truncate max-w-md md:max-w-xl">{target}</p>
            </div>
          </div>
          <button 
            onClick={onClose}
            className="p-2 hover:bg-indigo-500/10 rounded-xl transition-colors text-text-secondary hover:text-indigo-500 active:scale-95"
            title="Close"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Scrollable Container */}
        <div className="p-6 overflow-y-auto space-y-8 flex-1">
          {isLoading ? (
            <div className="py-20 flex flex-col items-center justify-center">
              <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin mb-4" />
              <p className="text-text-secondary font-medium">Fetching SRE telemetry...</p>
            </div>
          ) : error ? (
            <div className="py-12 text-center space-y-4">
              <div className="w-12 h-12 bg-red-500/10 text-red-500 rounded-full flex items-center justify-center mx-auto">
                <AlertCircle className="w-6 h-6" />
              </div>
              <div>
                <h3 className="font-bold text-lg">Telemetry Load Failed</h3>
                <p className="text-sm text-text-secondary">Could not fetch target status log history.</p>
              </div>
            </div>
          ) : (
            <>
              {/* Telemetry Quick Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div className="bg-bg-base border border-border-primary/60 rounded-2xl p-5 flex items-center gap-4">
                  <div className="p-3 bg-emerald-500/10 text-emerald-500 rounded-xl">
                    <Shield className="w-6 h-6" />
                  </div>
                  <div>
                    <p className="text-xs font-semibold text-text-secondary/70 uppercase tracking-wider">Uptime SLA</p>
                    <p className="text-2xl font-bold font-mono text-emerald-500">
                      {sortedChecks.length > 0 ? `${sortedChecks[sortedChecks.length - 1].uptime_sla.toFixed(2)}%` : '100.00%'}
                    </p>
                  </div>
                </div>

                <div className="bg-bg-base border border-border-primary/60 rounded-2xl p-5 flex items-center gap-4">
                  <div className="p-3 bg-indigo-500/10 text-indigo-500 rounded-xl">
                    <Clock className="w-6 h-6" />
                  </div>
                  <div>
                    <p className="text-xs font-semibold text-text-secondary/70 uppercase tracking-wider">Avg Latency</p>
                    <p className="text-2xl font-bold font-mono text-indigo-500 dark:text-indigo-400">
                      {avgLatency}ms
                    </p>
                  </div>
                </div>

                <div className="bg-bg-base border border-border-primary/60 rounded-2xl p-5 flex items-center gap-4">
                  <div className="p-3 bg-red-500/10 text-red-500 rounded-xl">
                    <AlertCircle className="w-6 h-6" />
                  </div>
                  <div>
                    <p className="text-xs font-semibold text-text-secondary/70 uppercase tracking-wider">Total Outages</p>
                    <p className="text-2xl font-bold font-mono text-red-500">
                      {incidents.length}
                    </p>
                  </div>
                </div>
              </div>

              {/* Histogram and Outage list */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                {/* Latency Histogram */}
                <div className="bg-bg-base border border-border-primary/60 rounded-2xl p-6 space-y-6">
                  <h3 className="text-sm font-bold uppercase tracking-wider text-text-secondary border-b border-border-primary/50 pb-3">
                    Max Latency Histogram (Last 100 pings)
                  </h3>
                  
                  <div className="space-y-4">
                    {buckets.map((bucket, index) => {
                      const percent = totalValidChecks > 0 ? (bucket.count / totalValidChecks) * 100 : 0;
                      const fillPercent = (bucket.count / maxBucketCount) * 100;
                      return (
                        <div key={index} className="flex items-center gap-4 text-xs">
                          <span className="w-20 font-medium text-text-secondary/80 font-mono text-right">{bucket.label}</span>
                          <div className="flex-1 bg-border-primary/30 h-5 rounded-md overflow-hidden relative border border-border-primary/50">
                            <div 
                              className={`h-full ${bucket.color} opacity-85 transition-all duration-500`} 
                              style={{ width: `${fillPercent}%` }}
                            />
                          </div>
                          <span className="w-16 font-semibold text-right font-mono">
                            {bucket.count} ({percent.toFixed(0)}%)
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Incident Duration Timeline */}
                <div className="bg-bg-base border border-border-primary/60 rounded-2xl p-6 flex flex-col">
                  <h3 className="text-sm font-bold uppercase tracking-wider text-text-secondary border-b border-border-primary/50 pb-3 mb-4">
                    Incident Duration Timeline
                  </h3>
                  <div className="flex-1 overflow-y-auto max-h-[250px] space-y-3 pr-2">
                    {incidents.length > 0 ? (
                      incidents.map((incident, index) => (
                        <div 
                          key={index}
                          className="flex items-start justify-between p-3 rounded-xl border border-red-500/10 bg-red-500/5 text-xs"
                        >
                          <div className="space-y-1">
                            <div className="flex items-center gap-1.5 font-bold text-red-500">
                              <AlertCircle className="w-3.5 h-3.5" />
                              <span>Outage Triggered</span>
                            </div>
                            <p className="text-[10px] text-text-secondary font-mono">
                              Start: {new Date(incident.start).toLocaleString()}
                            </p>
                            {incident.end && (
                              <p className="text-[10px] text-text-secondary font-mono">
                                Resolved: {new Date(incident.end).toLocaleString()}
                              </p>
                            )}
                          </div>
                          <span className="px-2 py-1 font-mono font-bold bg-red-500/10 text-red-500 rounded border border-red-500/20">
                            {formatDuration(incident.durationMs)}
                          </span>
                        </div>
                      ))
                    ) : (
                      <div className="flex-1 flex flex-col items-center justify-center py-10 text-center space-y-2">
                        <CheckCircle2 className="w-8 h-8 text-emerald-500" />
                        <p className="text-xs font-semibold text-emerald-500">All green. No incidents detected.</p>
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Raw checks table */}
              <div className="space-y-4">
                <div className="flex items-center justify-between border-b border-border-primary/50 pb-3">
                  <h3 className="text-sm font-bold uppercase tracking-wider text-text-secondary">
                    Raw Check Logs
                  </h3>
                  {rawLogsUrl && (
                    <a 
                      href={rawLogsUrl}
                      target="_blank" 
                      rel="noopener noreferrer"
                      className="text-xs font-bold text-indigo-500 dark:text-indigo-400 hover:underline flex items-center gap-1 cursor-pointer"
                    >
                      <span>Open Raw JSON</span>
                      <ExternalLink className="w-3.5 h-3.5" />
                    </a>
                  )}
                </div>

                <div className="border border-border-primary rounded-2xl overflow-hidden bg-bg-base/30 max-h-[300px] overflow-y-auto">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr className="bg-bg-base border-b border-border-primary text-text-secondary font-bold uppercase tracking-wider">
                        <th className="py-3 px-4">Time</th>
                        <th className="py-3 px-4">Status</th>
                        <th className="py-3 px-4 text-right">Latency</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border-primary/50">
                      {history?.map((check, index) => {
                        const isCheckUp = check.status === 'up';
                        return (
                          <tr key={index} className="hover:bg-bg-base/50 transition-colors">
                            <td className="py-3 px-4 font-mono text-text-secondary">{new Date(check.checked_at).toLocaleString()}</td>
                            <td className="py-3 px-4">
                              <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded font-bold uppercase tracking-wider text-[10px] ${
                                isCheckUp 
                                  ? 'text-emerald-500 bg-emerald-500/10 border border-emerald-500/20' 
                                  : 'text-red-500 bg-red-500/10 border border-red-500/20'
                              }`}>
                                {isCheckUp ? <CheckCircle2 className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                                <span>{check.status}</span>
                              </span>
                            </td>
                            <td className="py-3 px-4 text-right font-mono font-bold text-text-secondary">
                              {check.latency_ms}ms
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default IncidentLogModal;
