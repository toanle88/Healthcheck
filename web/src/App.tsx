import { AuthenticatedTemplate, UnauthenticatedTemplate } from "@azure/msal-react";
import DashboardPage from './pages/DashboardPage';
import LoginPage from './pages/LoginPage';
import { useAuth } from './hooks/useAuth';

function App() {
  const { isProcessing } = useAuth();

  // If we are in the middle of a login/redirect process, show a clean loading screen
  if (isProcessing) {
    return (
      <div className="min-h-screen bg-[#0f172a] flex items-center justify-center p-6">
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin mx-auto mb-4" />
          <p className="text-slate-400 animate-pulse font-medium">Processing secure login...</p>
        </div>
      </div>
    );
  }

  return (
    <>
      <AuthenticatedTemplate>
        <DashboardPage />
      </AuthenticatedTemplate>
      <UnauthenticatedTemplate>
        <LoginPage />
      </UnauthenticatedTemplate>
    </>
  );
}

export default App;
