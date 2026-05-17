import { screen } from '@testing-library/react';
import App from './App';
import { renderWithProviders } from './test/testUtils';
import { useAuth } from './hooks/useAuth';
import { describe, it, expect, vi } from 'vitest';

// Mock useAuth
vi.mock('./hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

// Mock components to simplify
vi.mock('./pages/DashboardPage', () => ({
  default: () => <div>Dashboard Page</div>
}));
vi.mock('./pages/LoginPage', () => ({
  default: () => <div>Login Page</div>
}));

// Mock AuthenticatedTemplate and UnauthenticatedTemplate
vi.mock('@azure/msal-react', async () => {
  const actual = await vi.importActual('@azure/msal-react');
  return {
    ...actual,
    AuthenticatedTemplate: ({ children }: { children: React.ReactNode }) => <div data-testid="auth">{children}</div>,
    UnauthenticatedTemplate: ({ children }: { children: React.ReactNode }) => <div data-testid="unauth">{children}</div>,
  };
});

describe('App', () => {
  it('renders processing state', () => {
    vi.mocked(useAuth).mockReturnValue({
      isProcessing: true,
      isAuthenticated: false,
      user: null,
      login: vi.fn(),
      logout: vi.fn(),
      getAccessToken: vi.fn(),
    });

    renderWithProviders(<App />);
    expect(screen.getByText(/Processing secure login/i)).toBeInTheDocument();
  });

  it('renders DashboardPage when authenticated', () => {
    vi.mocked(useAuth).mockReturnValue({
      isProcessing: false,
      isAuthenticated: true,
      user: { 
        name: 'Test User', 
        username: 'test@example.com', 
        account: { 
          homeAccountId: '', 
          environment: '', 
          tenantId: '', 
          username: 'test@example.com', 
          localAccountId: '' 
        } 
      },
      login: vi.fn(),
      logout: vi.fn(),
      getAccessToken: vi.fn(),
    });

    renderWithProviders(<App />);
    expect(screen.getByText('Dashboard Page')).toBeInTheDocument();
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument();
  });

  it('renders LoginPage when not authenticated', () => {
    vi.mocked(useAuth).mockReturnValue({
      isProcessing: false,
      isAuthenticated: false,
      user: null,
      login: vi.fn(),
      logout: vi.fn(),
      getAccessToken: vi.fn(),
    });

    renderWithProviders(<App />);
    expect(screen.getByText('Login Page')).toBeInTheDocument();
    expect(screen.queryByText('Dashboard Page')).not.toBeInTheDocument();
  });
});
