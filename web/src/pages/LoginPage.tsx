import React from 'react';
import { Activity, LogIn } from 'lucide-react';
import { useAuth } from '../hooks/useAuth';

const LoginPage: React.FC = () => {
  const { login } = useAuth();

  return (
    <div className="min-h-screen bg-bg-base text-text-primary flex items-center justify-center p-6 transition-colors duration-300">
      <div className="max-w-md w-full bg-bg-card/50 border border-border-primary p-10 rounded-3xl text-center backdrop-blur-xl shadow-2xl">
        <div className="w-20 h-20 bg-indigo-500/10 rounded-2xl flex items-center justify-center mx-auto mb-8">
          <Activity className="w-10 h-10 text-indigo-400" />
        </div>
        <h1 className="text-3xl font-bold text-text-primary mb-3">Welcome Back</h1>
        <p className="text-text-secondary mb-10 leading-relaxed">Please sign in with your enterprise account to access the healthcheck dashboard.</p>
        
        <button 
          onClick={login}
          className="w-full bg-indigo-600 hover:bg-indigo-500 text-white py-4 rounded-2xl font-bold transition-all active:scale-95 shadow-xl shadow-indigo-500/20 flex items-center justify-center gap-3 group cursor-pointer"
        >
          <LogIn className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
          Sign In with Entra ID
        </button>
        
        <p className="mt-8 text-xs text-text-secondary/60 font-medium uppercase tracking-widest">Enterprise Security Enabled</p>
      </div>
    </div>
  );
};

export default LoginPage;
