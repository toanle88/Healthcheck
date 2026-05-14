import React from 'react';
import { Activity, Clock, RefreshCw, LogOut } from 'lucide-react';
import { useAuth } from '../../hooks/useAuth';

interface HeaderProps {
  error: boolean;
  lastUpdated: Date;
  isRefreshing: boolean;
  onRefresh: () => void;
}

const Header: React.FC<HeaderProps> = ({ error, lastUpdated, isRefreshing, onRefresh }) => {
  const { user, logout } = useAuth();

  return (
    <header className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-xl sticky top-0 z-50">
      <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-500/10 rounded-lg">
            <Activity className="w-6 h-6 text-indigo-400 animate-pulse" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight">Healthcheck <span className="text-indigo-400">Dashboard</span></h1>
            <div className="text-xs text-slate-400 font-medium flex items-center gap-1.5">
              <div className={`w-1.5 h-1.5 rounded-full ${error ? 'bg-red-500' : 'bg-emerald-500 animate-pulse'}`} />
              System {error ? 'Degraded' : 'Operational'}
            </div>
          </div>
        </div>
        
        <div className="flex items-center gap-4 text-sm font-medium">
          <div className="hidden md:flex items-center gap-2 text-slate-400 bg-slate-800/50 px-3 py-1.5 rounded-full border border-slate-700/50">
            <span className="w-2 h-2 bg-emerald-500 rounded-full" />
            <span className="truncate max-w-[150px]">{user?.name}</span>
          </div>
          <div className="flex items-center gap-2 text-slate-400 bg-slate-800/50 px-3 py-1.5 rounded-full border border-slate-700/50">
            <Clock className="w-4 h-4" />
            <span>{lastUpdated.toLocaleTimeString()}</span>
          </div>
          <button 
            onClick={onRefresh}
            className="p-2 hover:bg-indigo-500/10 rounded-lg transition-colors group active:scale-95"
            title="Refresh Data"
          >
            <RefreshCw className={`w-5 h-5 text-slate-400 group-hover:text-indigo-400 transition-colors ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>
          <button 
            onClick={logout}
            className="p-2 hover:bg-red-500/10 rounded-lg transition-colors group active:scale-95"
            title="Sign Out"
          >
            <LogOut className="w-5 h-5 text-slate-400 group-hover:text-red-400 transition-colors" />
          </button>
        </div>
      </div>
    </header>
  );
};

export default Header;
