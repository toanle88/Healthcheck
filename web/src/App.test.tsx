import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { MockInstance } from 'vitest';
import App from './App';

// 1. Mock MSAL
vi.mock("@azure/msal-react", () => ({
    useMsal: () => ({
        instance: {
            acquireTokenSilent: vi.fn().mockResolvedValue({ accessToken: "mock-token" }),
            loginPopup: vi.fn(),
            logoutPopup: vi.fn(),
        },
        accounts: [{ name: "Test User" }],
    }),
    useIsAuthenticated: () => true,
    AuthenticatedTemplate: ({ children }: { children: React.ReactNode }) => <>{children}</>,
    UnauthenticatedTemplate: () => null,
    MsalProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// Mock the fetch function globally
globalThis.fetch = vi.fn() as unknown as typeof fetch & MockInstance;

describe('App Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state initially', () => {
    // Mock fetch to return a promise that never resolves for this test
    vi.mocked(fetch).mockReturnValue(new Promise(() => {}));
    
    render(<App />);
    expect(screen.getByText(/Authenticating and loading data/i)).toBeInTheDocument();
  });

  it('renders data when fetch is successful', async () => {
    const mockData = {
      checks: [
        { target: 'http://test.com', status: 'up', latency_ms: 100, checked_at: new Date().toISOString() }
      ],
      count: 1
    };

    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockData),
    } as Response);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText('test.com')).toBeInTheDocument();
      expect(screen.getByText('100ms')).toBeInTheDocument();
      expect(screen.getByText('Test User')).toBeInTheDocument();
    });
  });

  it('renders error state when fetch fails', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
    } as Response);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText(/Connection Error/i)).toBeInTheDocument();
    });
  });
});
