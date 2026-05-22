import { useEffect, useState } from 'react';
import DashboardPage from './pages/DashboardPage';
import LoginPage from './pages/LoginPage';
import { useAuth } from './hooks/useAuth';

function App() {
  const { isAuthenticated, isProcessing } = useAuth();

  const [theme, setTheme] = useState<'light' | 'dark'>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('theme');
      if (saved === 'light' || saved === 'dark') return saved;
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return 'dark';
  });

  useEffect(() => {
    const root = window.document.documentElement;
    if (theme === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    localStorage.setItem('theme', theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme(prev => (prev === 'light' ? 'dark' : 'light'));
  };

  // If we are in the middle of a login/redirect process, show a clean loading screen
  if (isProcessing) {
    return (
      <div className="min-h-screen bg-bg-base flex items-center justify-center p-6 transition-colors duration-300">
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin mx-auto mb-4" />
          <p className="text-text-secondary animate-pulse font-medium">Processing secure login...</p>
        </div>
      </div>
    );
  }

  return isAuthenticated ? (
    <DashboardPage theme={theme} toggleTheme={toggleTheme} />
  ) : (
    <LoginPage />
  );
}

export default App;
