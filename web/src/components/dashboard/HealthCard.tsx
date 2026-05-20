import React from 'react';
import { ShieldCheck, ShieldAlert, Clock, HelpCircle } from 'lucide-react';
import type { Check } from '../../types';
import UptimeChart from './UptimeChart';

interface HealthCardProps {
  check: Check;
}

const HealthCard: React.FC<HealthCardProps> = ({ check }) => {
  return (
    <div className="group bg-slate-900/50 border border-slate-800 rounded-2xl p-6 hover:border-slate-700 hover:bg-slate-800/50 transition-all duration-300 hover:shadow-2xl hover:shadow-indigo-500/5">
      <div className="flex items-start justify-between mb-4">
        <div className={`p-2.5 rounded-xl ${
          check.status === 'up' ? 'bg-emerald-500/10' : 
          check.status === 'down' ? 'bg-red-500/10' : 'bg-amber-500/10'
        }`}>
          {check.status === 'up' ? (
            <ShieldCheck className="w-6 h-6 text-emerald-400" />
          ) : check.status === 'down' ? (
            <ShieldAlert className="w-6 h-6 text-red-400" />
          ) : (
            <HelpCircle className="w-6 h-6 text-amber-400 animate-pulse" />
          )}
        </div>
        <div className="flex flex-col items-end">
          <span className={`text-xs font-bold uppercase tracking-wider px-2 py-1 rounded-md ${
            check.status === 'up' ? 'text-emerald-400 bg-emerald-400/10' : 
            check.status === 'down' ? 'text-red-400 bg-red-400/10' : 'text-amber-400 bg-amber-400/10'
          }`}>
            {check.status}
          </span>
          <span className="text-[10px] text-slate-500 mt-2 font-mono">{new Date(check.checked_at).toLocaleTimeString()}</span>
        </div>
      </div>
      
      <div className="space-y-1 mb-6">
        <h3 className="font-bold text-slate-200 truncate group-hover:text-white transition-colors" title={check.name || check.target}>
          {check.name || check.target.replace(/^https?:\/\//, '')}
        </h3>
        <p className="text-xs text-slate-500 font-mono truncate">{check.target}</p>
      </div>

      <div className="flex items-center justify-between pt-4 border-t border-slate-800/50">
        <div className="flex items-center gap-2 text-slate-400">
          <Clock className="w-3.5 h-3.5" />
          <span className="text-xs font-medium">Latency</span>
        </div>
        <span className={`text-sm font-bold font-mono ${
          check.status === 'pending' ? 'text-slate-500' :
          check.latency_ms < 200 ? 'text-emerald-400' : check.latency_ms < 500 ? 'text-amber-400' : 'text-red-400'
        }`}>
          {check.status === 'pending' ? '-' : `${check.latency_ms}ms`}
        </span>
      </div>

      <div className="flex items-center justify-between pt-2">
        <span className="text-xs text-slate-400 font-medium">24h Uptime SLA</span>
        <span className={`text-[10px] font-bold font-mono px-2 py-0.5 rounded ${
          check.status === 'pending' ? 'text-slate-500 bg-slate-500/10' :
          check.uptime_sla >= 99.9 ? 'text-emerald-400 bg-emerald-400/10' :
          check.uptime_sla >= 99.0 ? 'text-amber-400 bg-amber-400/10' : 'text-red-400 bg-red-400/10'
        }`}>
          {check.status === 'pending' ? '100.00%' : `${check.uptime_sla.toFixed(2)}%`}
        </span>
      </div>

      {check.status !== 'pending' && <UptimeChart target={check.target} />}
    </div>
  );
};

export default HealthCard;
