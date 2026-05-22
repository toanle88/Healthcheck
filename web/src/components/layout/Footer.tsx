import React from 'react';
import { getEnv } from '../../config/env';

interface FooterProps {
  count: number;
}

const Footer: React.FC<FooterProps> = ({ count }) => {
  return (
    <footer className="mt-auto border-t border-border-primary py-8 px-6">
      <div className="max-w-6xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        <p className="text-sm text-text-secondary font-medium">
          Monitoring {count} endpoints across global infrastructure
        </p>
        <div className="flex items-center gap-3">
          <span className="text-[10px] font-bold uppercase tracking-widest text-text-secondary/70">Version</span>
          <span className="px-2 py-0.5 bg-bg-card text-text-secondary rounded text-[10px] font-mono border border-border-primary">
            {getEnv('VITE_APP_VERSION') || 'local-dev'}
          </span>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
