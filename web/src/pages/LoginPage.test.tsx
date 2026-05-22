import { render, screen, fireEvent } from '@testing-library/react';
import LoginPage from './LoginPage';
import { useAuth } from '../hooks/useAuth';
import { describe, it, expect, vi } from 'vitest';

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

describe('LoginPage', () => {
  it('renders login button and calls login function', () => {
    const mockLogin = vi.fn();
    vi.mocked(useAuth).mockReturnValue({
      login: mockLogin,
      isAuthenticated: false,
      user: null,
      logout: vi.fn(),
      getAccessToken: vi.fn(),
      isProcessing: false,
      isAdmin: false,
    });

    render(<LoginPage />);
    
    expect(screen.getByText(/Welcome Back/i)).toBeInTheDocument();
    
    const loginBtn = screen.getByText(/Sign In with Entra ID/i);
    fireEvent.click(loginBtn);
    
    expect(mockLogin).toHaveBeenCalled();
  });
});
