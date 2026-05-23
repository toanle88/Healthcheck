import { renderHook, act } from '@testing-library/react';
import { useAuth } from './useAuth';
import { useMsal } from "@azure/msal-react";
import type { IPublicClientApplication } from "@azure/msal-browser";
import { Logger } from "@azure/msal-browser";
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock MSAL
vi.mock('@azure/msal-react', () => ({
  useMsal: vi.fn(),
}));

const makeMsalMock = (overrides: Partial<{
  accounts: object[];
  inProgress: string;
  acquireTokenSilent: ReturnType<typeof vi.fn>;
  acquireTokenPopup: ReturnType<typeof vi.fn>;
  loginRedirect: ReturnType<typeof vi.fn>;
  logoutRedirect: ReturnType<typeof vi.fn>;
}> = {}) => {
  const {
    accounts = [{ name: 'Test User', username: 'test@example.com', homeAccountId: '', environment: '', tenantId: '', localAccountId: '' }],
    inProgress = 'none',
    acquireTokenSilent = vi.fn().mockResolvedValue({ accessToken: 'silent-token' }),
    acquireTokenPopup = vi.fn().mockResolvedValue({ accessToken: 'popup-token' }),
    loginRedirect = vi.fn(),
    logoutRedirect = vi.fn(),
  } = overrides;

  vi.mocked(useMsal).mockReturnValue({
    instance: {
      logoutRedirect,
      loginRedirect,
      acquireTokenSilent,
      acquireTokenPopup,
      initialize: vi.fn().mockResolvedValue(undefined),
    } as unknown as IPublicClientApplication,
    accounts: accounts as never[],
    inProgress: inProgress as never,
    logger: new Logger({}),
  });
};

describe('useAuth (standard mode)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Ensure E2E flags are clean
    localStorage.removeItem('playwright-mock-auth');
    delete (window as unknown as Record<string, unknown>).playwrightMockAuth;
  });

  it('returns user information when logged in', () => {
    makeMsalMock();
    const { result } = renderHook(() => useAuth());
    expect(result.current.user?.name).toBe('Test User');
    expect(result.current.user?.username).toBe('test@example.com');
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('returns null user when no accounts', () => {
    makeMsalMock({ accounts: [] });
    const { result } = renderHook(() => useAuth());
    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('reflects isProcessing when inProgress !== "none"', () => {
    makeMsalMock({ inProgress: 'login' });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isProcessing).toBe(true);
  });

  it('calls loginRedirect when login() is called', async () => {
    const loginRedirect = vi.fn();
    makeMsalMock({ loginRedirect });
    const { result } = renderHook(() => useAuth());
    await act(async () => { result.current.login(); });
    expect(loginRedirect).toHaveBeenCalled();
  });

  it('calls logoutRedirect when logout() is called', () => {
    const logoutRedirect = vi.fn();
    makeMsalMock({ logoutRedirect });
    const { result } = renderHook(() => useAuth());
    result.current.logout();
    expect(logoutRedirect).toHaveBeenCalledWith({
      postLogoutRedirectUri: window.location.origin,
    });
  });

  it('acquires token silently on getAccessToken()', async () => {
    const acquireTokenSilent = vi.fn().mockResolvedValue({ accessToken: 'my-token' });
    makeMsalMock({ acquireTokenSilent });
    const { result } = renderHook(() => useAuth());
    const token = await result.current.getAccessToken();
    expect(token).toBe('my-token');
    expect(acquireTokenSilent).toHaveBeenCalled();
  });

  it('falls back to popup when silent token fails', async () => {
    const acquireTokenSilent = vi.fn().mockRejectedValue(new Error('silent failed'));
    const acquireTokenPopup = vi.fn().mockResolvedValue({ accessToken: 'popup-token' });
    makeMsalMock({ acquireTokenSilent, acquireTokenPopup });
    const { result } = renderHook(() => useAuth());
    const token = await result.current.getAccessToken();
    expect(token).toBe('popup-token');
    expect(acquireTokenPopup).toHaveBeenCalled();
  });

  it('throws when both silent and popup token acquisition fail', async () => {
    const acquireTokenSilent = vi.fn().mockRejectedValue(new Error('silent failed'));
    const acquireTokenPopup = vi.fn().mockRejectedValue(new Error('popup failed'));
    makeMsalMock({ acquireTokenSilent, acquireTokenPopup });
    const { result } = renderHook(() => useAuth());
    await expect(result.current.getAccessToken()).rejects.toThrow('popup failed');
  });

  it('throws when getAccessToken is called with no logged-in user', async () => {
    makeMsalMock({ accounts: [] });
    const { result } = renderHook(() => useAuth());
    await expect(result.current.getAccessToken()).rejects.toThrow('No active account found');
  });

  it('isAdmin returns true when roles is array containing Healthcheck.Admin', () => {
    makeMsalMock({
      accounts: [{
        name: 'Admin User',
        username: 'admin@example.com',
        homeAccountId: '',
        environment: '',
        tenantId: '',
        localAccountId: '',
        idTokenClaims: { roles: ['Healthcheck.Admin', 'Other.Role'] },
      }],
    });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(true);
  });

  it('isAdmin returns false when roles array does not contain Healthcheck.Admin', () => {
    makeMsalMock({
      accounts: [{
        name: 'Regular User',
        username: 'user@example.com',
        homeAccountId: '',
        environment: '',
        tenantId: '',
        localAccountId: '',
        idTokenClaims: { roles: ['Other.Role'] },
      }],
    });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(false);
  });

  it('isAdmin returns true when roles is a string matching Healthcheck.Admin', () => {
    makeMsalMock({
      accounts: [{
        name: 'Admin User',
        username: 'admin@example.com',
        homeAccountId: '',
        environment: '',
        tenantId: '',
        localAccountId: '',
        idTokenClaims: { roles: 'Healthcheck.Admin' },
      }],
    });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(true);
  });

  it('isAdmin returns false when roles is string not matching admin role', () => {
    makeMsalMock({
      accounts: [{
        name: 'User',
        username: 'user@example.com',
        homeAccountId: '',
        environment: '',
        tenantId: '',
        localAccountId: '',
        idTokenClaims: { roles: 'Healthcheck.Reader' },
      }],
    });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(false);
  });

  it('isAdmin returns false when no roles present', () => {
    makeMsalMock({
      accounts: [{
        name: 'User',
        username: 'user@example.com',
        homeAccountId: '',
        environment: '',
        tenantId: '',
        localAccountId: '',
        idTokenClaims: {},
      }],
    });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(false);
  });

  it('isAdmin returns false when no user is logged in', () => {
    makeMsalMock({ accounts: [] });
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(false);
  });
});

describe('useAuth (E2E / Playwright mock mode)', () => {
  afterEach(() => {
    localStorage.removeItem('playwright-mock-auth');
    localStorage.removeItem('playwright-mock-role');
    delete (window as unknown as Record<string, unknown>).playwrightMockAuth;
    delete (window as unknown as Record<string, unknown>).playwrightMockRole;
  });

  const setupE2EViaMsal = () => {
    vi.mocked(useMsal).mockReturnValue({
      instance: {
        logoutRedirect: vi.fn(),
        loginRedirect: vi.fn(),
        acquireTokenSilent: vi.fn(),
        initialize: vi.fn().mockResolvedValue(undefined),
      } as unknown as IPublicClientApplication,
      accounts: [],
      inProgress: 'none',
      logger: new Logger({}),
    });
  };

  it('returns E2E user when playwright-mock-auth is set in localStorage', () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    expect(result.current.user?.name).toBe('E2E Test User');
    expect(result.current.isAuthenticated).toBe(true);
    expect(result.current.isProcessing).toBe(false);
  });

  it('returns mocked-e2e-token from getAccessToken in E2E mode', async () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    const token = await result.current.getAccessToken();
    expect(token).toBe('mocked-e2e-token');
  });

  it('login() is a no-op in E2E mode', async () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    const loginRedirect = vi.fn();
    vi.mocked(useMsal).mockReturnValue({
      instance: { loginRedirect, logoutRedirect: vi.fn(), acquireTokenSilent: vi.fn(), initialize: vi.fn() } as unknown as IPublicClientApplication,
      accounts: [],
      inProgress: 'none',
      logger: new Logger({}),
    });
    const { result } = renderHook(() => useAuth());
    await act(async () => { await result.current.login(); });
    expect(loginRedirect).not.toHaveBeenCalled();
  });

  it('logout() is a no-op in E2E mode', () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    const logoutRedirect = vi.fn();
    vi.mocked(useMsal).mockReturnValue({
      instance: { loginRedirect: vi.fn(), logoutRedirect, acquireTokenSilent: vi.fn(), initialize: vi.fn() } as unknown as IPublicClientApplication,
      accounts: [],
      inProgress: 'none',
      logger: new Logger({}),
    });
    const { result } = renderHook(() => useAuth());
    result.current.logout();
    expect(logoutRedirect).not.toHaveBeenCalled();
  });

  it('isAdmin is true by default in E2E mode (no role set)', () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(true);
  });

  it('isAdmin reflects playwright-mock-role from localStorage in E2E mode', () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    localStorage.setItem('playwright-mock-role', 'Healthcheck.Reader');
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(false);
  });

  it('isAdmin reflects playwrightMockRole from window in E2E mode', () => {
    localStorage.setItem('playwright-mock-auth', 'true');
    (window as unknown as Record<string, unknown>).playwrightMockRole = 'Healthcheck.Admin';
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    expect(result.current.isAdmin).toBe(true);
  });

  it('detects E2E mode from window.playwrightMockAuth = true', () => {
    (window as unknown as Record<string, unknown>).playwrightMockAuth = true;
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    expect(result.current.user?.name).toBe('E2E Test User');
  });

  it('detects E2E mode from window.playwrightMockAuth = "true" (string)', () => {
    (window as unknown as Record<string, unknown>).playwrightMockAuth = 'true';
    setupE2EViaMsal();
    const { result } = renderHook(() => useAuth());
    expect(result.current.user?.name).toBe('E2E Test User');
  });
});
