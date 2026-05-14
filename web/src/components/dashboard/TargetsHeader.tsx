import React from 'react';
import { Server } from 'lucide-react';

interface TargetsHeaderProps {
  count: number;
}

const TargetsHeader: React.FC<TargetsHeaderProps> = ({ count }) => {
  return (
    <div className="flex items-center justify-between">
      <h2 className="text-lg font-semibold flex items-center gap-2">
        <Server className="w-5 h-5 text-indigo-400" />
        Active Targets
        <span className="ml-2 px-2 py-0.5 bg-slate-800 rounded text-xs text-slate-400">{count}</span>
      </h2>
    </div>
  );
};

export default TargetsHeader;
