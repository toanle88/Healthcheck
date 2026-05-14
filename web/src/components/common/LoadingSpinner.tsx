import React from 'react';

const LoadingSpinner: React.FC = () => {
  return (
    <div className="flex flex-col items-center justify-center py-32 gap-4">
      <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin" />
      <p className="text-slate-400 animate-pulse font-medium">Authenticating and loading data...</p>
    </div>
  );
};

export default LoadingSpinner;
