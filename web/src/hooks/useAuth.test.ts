import { renderHook } from '@testing-library/react';
import { useAuth } from './useAuth';
import { useMsal } from "@azure/msal-react";
import type { IPublicClientApplication } from "@azure/msal-browser";
import { Logger } from "@azure/msal-browser";
import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock MSAL
vi.mock('@azure/msal-react', () => ({
  useMsal: vi.fn(),
}));

describe('useAuth', () => {
  const mockLogout = vi.fn();
  const mockLogin = vi.fn();
  const mockAcquireTokenSilent = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useMsal).mockReturnValue({
      instance: {
        logoutRedirect: mockLogout,
        loginRedirect: mockLogin,
        acquireTokenSilent: mockAcquireTokenSilent,
        initialize: vi.fn().mockResolvedValue(undefined),
      } as unknown as IPublicClientApplication,
      accounts: [{ 
        name: 'Test User', 
        username: 'test@example.com',
        homeAccountId: '',
        environment: '',
        tenantId: '',
        localAccountId: '',
      }],
      inProgress: 'none',
      logger: new Logger({}),
    });
  });

  it('returns user information when logged in', () => {
    const { result } = renderHook(() => useAuth());
    expect(result.current.user?.name).toBe('Test User');
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('calls loginRedirect when login is called', () => {
    const { result } = renderHook(() => useAuth());
    result.current.login();
    expect(mockLogin).toHaveBeenCalled();
  });

  it('calls logoutRedirect when logout is called', () => {
    const { result } = renderHook(() => useAuth());
    result.current.logout();
    expect(mockLogout).toHaveBeenCalledWith({
      postLogoutRedirectUri: window.location.origin,
    });
  });

  it('gets access token silently', async () => {
    mockAcquireTokenSilent.mockResolvedValue({ accessToken: 'new-token' });
    const { result } = renderHook(() => useAuth());
    const token = await result.current.getAccessToken();
    expect(token).toBe('new-token');
    expect(mockAcquireTokenSilent).toHaveBeenCalled();
  });
});
