import React from 'react';
import { ShieldCheck, ShieldAlert, Clock, HelpCircle } from 'lucide-react';
import type { Check } from '../../types';
import UptimeChart from './UptimeChart';

interface HealthCardProps {
  check: Check;
  onClick?: () => void;
}

const HealthCard: React.FC<HealthCardProps> = ({ check, onClick }) => {
  const isDown = check.status === 'down';
  const isUp = check.status === 'up';
  const isTransient = isDown && check.consecutive_failures < (check.failure_threshold || 3);

  // Icon container class
  let iconContainerBg = 'bg-amber-500/10';
  if (isUp) {
    iconContainerBg = 'bg-emerald-500/10';
  } else if (isTransient) {
    iconContainerBg = 'bg-amber-500/10';
  } else if (isDown) {
    iconContainerBg = 'bg-red-500/10';
  }
  const iconContainerClass = `p-2.5 rounded-xl ${iconContainerBg}`;

  // Status badge class
  let badgeColorClass = 'text-amber-400 bg-amber-400/10';
  if (isUp) {
    badgeColorClass = 'text-emerald-400 bg-emerald-400/10';
  } else if (isTransient) {
    badgeColorClass = 'text-amber-400 bg-amber-400/10';
  } else if (isDown) {
    badgeColorClass = 'text-red-400 bg-red-400/10';
  }
  const badgeClass = `text-xs font-bold uppercase tracking-wider px-2 py-1 rounded-md ${badgeColorClass}`;

  // Badge label text
  let badgeText = check.status;
  if (isTransient) {
    badgeText = `FAILING (${check.consecutive_failures}/${check.failure_threshold || 3})`;
  } else if (isDown) {
    badgeText = `DOWN (${check.consecutive_failures})`;
  }

  // Latency text color class
  let latencyColor = 'text-red-400';
  if (check.status === 'pending') {
    latencyColor = 'text-text-secondary/60';
  } else if (check.latency_ms < 200) {
    latencyColor = 'text-emerald-400';
  } else if (check.latency_ms < 500) {
    latencyColor = 'text-amber-400';
  }
  const latencyClass = `text-sm font-bold font-mono ${latencyColor}`;

  // Uptime SLA text color class
  let uptimeSLAColor = 'text-red-400 bg-red-400/10';
  if (check.status === 'pending') {
    uptimeSLAColor = 'text-text-secondary/60 bg-text-secondary/10';
  } else if (check.uptime_sla >= 99.9) {
    uptimeSLAColor = 'text-emerald-400 bg-emerald-400/10';
  } else if (check.uptime_sla >= 99) {
    uptimeSLAColor = 'text-amber-400 bg-amber-400/10';
  }
  const uptimeSLAClass = `text-[10px] font-bold font-mono px-2 py-0.5 rounded ${uptimeSLAColor}`;

  // Status icon component
  let StatusIcon = <HelpCircle className="w-6 h-6 text-amber-400 animate-pulse" />;
  if (isUp) {
    StatusIcon = <ShieldCheck className="w-6 h-6 text-emerald-400" />;
  } else if (isTransient) {
    StatusIcon = <ShieldAlert className="w-6 h-6 text-amber-400" />;
  } else if (isDown) {
    StatusIcon = <ShieldAlert className="w-6 h-6 text-red-400" />;
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (onClick && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      onClick();
    }
  };

  return (
    <div 
      onClick={onClick}
      onKeyDown={handleKeyDown}
      role="button"
      tabIndex={0}
      className="group bg-bg-card/50 border border-border-primary rounded-2xl p-6 hover:border-indigo-500/50 hover:bg-bg-card transition-all duration-300 hover:shadow-2xl hover:shadow-indigo-500/5 cursor-pointer text-text-primary"
    >
      <div className="flex items-start justify-between mb-4">
        <div className={iconContainerClass}>
          {StatusIcon}
        </div>
        <div className="flex flex-col items-end">
          <span className={badgeClass}>
            {badgeText}
          </span>
          <span className="text-[10px] text-text-secondary/60 mt-2 font-mono">{new Date(check.checked_at).toLocaleTimeString()}</span>
        </div>
      </div>
      
      <div className="space-y-1 mb-6">
        <h3 className="font-bold text-text-primary truncate group-hover:text-indigo-500 dark:group-hover:text-indigo-400 transition-colors" title={check.name || check.target}>
          {check.name || check.target.replace(/^https?:\/\//, '')}
        </h3>
        <p className="text-xs text-text-secondary/70 font-mono truncate">{check.target}</p>
      </div>

      <div className="flex items-center justify-between pt-4 border-t border-border-primary/50">
        <div className="flex items-center gap-2 text-text-secondary">
          <Clock className="w-3.5 h-3.5" />
          <span className="text-xs font-medium">Latency</span>
        </div>
        <span className={latencyClass}>
          {check.status === 'pending' ? '-' : `${check.latency_ms}ms`}
        </span>
      </div>

      <div className="flex items-center justify-between pt-2">
        <span className="text-xs text-text-secondary font-medium">24h Uptime SLA</span>
        <span className={uptimeSLAClass}>
          {check.status === 'pending' ? '100.00%' : `${check.uptime_sla.toFixed(2)}%`}
        </span>
      </div>

      {check.status !== 'pending' && <UptimeChart target={check.target} />}
    </div>
  );
};

export default HealthCard;
