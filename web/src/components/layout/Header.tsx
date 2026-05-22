import React from 'react';
import { Activity, Clock, RefreshCw, LogOut, Sun, Moon } from 'lucide-react';
import { useAuth } from '../../hooks/useAuth';

interface HeaderProps {
  error: boolean;
  lastUpdated: Date;
  isRefreshing: boolean;
  onRefresh: () => void;
  theme?: 'light' | 'dark';
  onToggleTheme?: () => void;
}

const Header: React.FC<HeaderProps> = ({ 
  error, 
  lastUpdated, 
  isRefreshing, 
  onRefresh,
  theme,
  onToggleTheme
}) => {
  const { user, logout } = useAuth();

  return (
    <header className="border-b border-border-primary bg-bg-card/50 backdrop-blur-xl sticky top-0 z-50 transition-colors duration-300 text-text-primary">
      <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-500/10 rounded-lg">
            <Activity className="w-6 h-6 text-indigo-400 animate-pulse" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight">Healthcheck <span className="text-indigo-400">Dashboard</span></h1>
            <div className="text-xs text-text-secondary font-medium flex items-center gap-1.5">
              <div className={`w-1.5 h-1.5 rounded-full ${error ? 'bg-red-500' : 'bg-emerald-500 animate-pulse'}`} />
              System {error ? 'Degraded' : 'Operational'}
            </div>
          </div>
        </div>
        
        <div className="flex items-center gap-4 text-sm font-medium">
          <div className="hidden md:flex items-center gap-2 text-text-secondary bg-bg-base px-3 py-1.5 rounded-full border border-border-primary transition-colors duration-300">
            <span className="w-2 h-2 bg-emerald-500 rounded-full" />
            <span className="truncate max-w-[150px]">{user?.name}</span>
          </div>
          <div className="flex items-center gap-2 text-text-secondary bg-bg-base px-3 py-1.5 rounded-full border border-border-primary transition-colors duration-300">
            <Clock className="w-4 h-4" />
            <span>{lastUpdated.toLocaleTimeString()}</span>
          </div>
          {onToggleTheme && (
            <button
              onClick={onToggleTheme}
              className="p-2 hover:bg-indigo-500/10 rounded-lg transition-colors group active:scale-95 text-text-secondary hover:text-indigo-400"
              title={`Switch to ${theme === 'dark' ? 'Light' : 'Dark'} Mode`}
            >
              {theme === 'dark' ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
            </button>
          )}
          <button 
            onClick={onRefresh}
            className="p-2 hover:bg-indigo-500/10 rounded-lg transition-colors group active:scale-95"
            title="Refresh Data"
          >
            <RefreshCw className={`w-5 h-5 text-text-secondary group-hover:text-indigo-400 transition-colors ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>
          <button 
            onClick={logout}
            className="p-2 hover:bg-red-500/10 rounded-lg transition-colors group active:scale-95"
            title="Sign Out"
          >
            <LogOut className="w-5 h-5 text-text-secondary group-hover:text-red-400 transition-colors" />
          </button>
        </div>
      </div>
    </header>
  );
};

export default Header;
