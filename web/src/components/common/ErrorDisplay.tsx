import React from 'react';
import { ShieldAlert } from 'lucide-react';

interface ErrorDisplayProps {
  error: string;
  onRetry: () => void;
}

const ErrorDisplay: React.FC<ErrorDisplayProps> = ({ error, onRetry }) => {
  return (
    <div className="bg-red-500/10 border border-red-500/20 rounded-2xl p-8 text-center max-w-lg mx-auto">
      <ShieldAlert className="w-12 h-12 text-red-400 mx-auto mb-4" />
      <h2 className="text-xl font-bold text-red-100 mb-2">Connection Error</h2>
      <p className="text-red-400/80 mb-6">{error}</p>
      <button 
        onClick={onRetry}
        className="bg-red-500 hover:bg-red-600 text-white px-6 py-2 rounded-xl font-bold transition-all active:scale-95 shadow-lg shadow-red-500/20"
      >
        Retry Connection
      </button>
    </div>
  );
};

export default ErrorDisplay;
