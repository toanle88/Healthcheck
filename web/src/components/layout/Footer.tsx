import React from 'react';
import { getEnv } from '../../config/env';

interface FooterProps {
  count: number;
}

const Footer: React.FC<FooterProps> = ({ count }) => {
  return (
    <footer className="mt-auto border-t border-slate-800 py-8 px-6">
      <div className="max-w-6xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        <p className="text-sm text-slate-500 font-medium">
          Monitoring {count} endpoints across global infrastructure
        </p>
        <div className="flex items-center gap-3">
          <span className="text-[10px] font-bold uppercase tracking-widest text-slate-600">Version</span>
          <span className="px-2 py-0.5 bg-slate-800 text-slate-400 rounded text-[10px] font-mono border border-slate-700/50">
            {getEnv('VITE_APP_VERSION') || 'local-dev'}
          </span>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
