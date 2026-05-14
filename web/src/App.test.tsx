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

  it('renders main app templates when not processing', () => {
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
    expect(screen.getByTestId('auth')).toBeInTheDocument();
    expect(screen.getByTestId('unauth')).toBeInTheDocument();
  });
});
