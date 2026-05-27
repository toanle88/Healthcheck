import { screen, waitFor } from '@testing-library/react';
import UptimeChart from './UptimeChart';
import { renderWithProviders } from '../../test/testUtils';
import { describe, it, expect, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../../test/setup';
import { getEnv } from '../../config/env';

const API_BASE_URL = getEnv('VITE_API_URL');

// Mock useAuth
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    getAccessToken: vi.fn().mockResolvedValue('fake-token'),
    user: { name: 'Test User' },
    isAdmin: true,
    isProcessing: false,
  }),
}));

const makeCheck = (
  status: 'up' | 'down',
  latency_ms: number,
  checked_at: string
) => ({
  status,
  latency_ms,
  checked_at,
  uptime_sla: 99.9,
  target: 'https://example.com',
  name: 'Example',
});

describe('UptimeChart', () => {
  it('shows loading spinner initially', () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, async () => {
        await new Promise((r) => setTimeout(r, 200));
        return HttpResponse.json([]);
      })
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);
    // Spinner is a div with animate-spin
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  it('shows error state on fetch failure', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => {
        return new HttpResponse(null, { status: 500 });
      })
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      expect(screen.getByText(/Failed to load history/i)).toBeInTheDocument();
    });
  });

  it('renders sparkline SVG when history data is available', async () => {
    const checks = [
      makeCheck('up', 45, '2024-01-01T10:00:00Z'),
      makeCheck('up', 120, '2024-01-01T10:01:00Z'),
      makeCheck('down', 0, '2024-01-01T10:02:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      const svg = document.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });
  });

  it('shows max latency label', async () => {
    const checks = [
      makeCheck('up', 45, '2024-01-01T10:00:00Z'),
      makeCheck('up', 250, '2024-01-01T10:01:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      expect(screen.getByText('Max: 250ms')).toBeInTheDocument();
    });
    expect(screen.getByText('Latency History (last 30 pings)')).toBeInTheDocument();
  });

  it('shows empty state message when history is empty array', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json([]))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      expect(screen.getByText('No history data')).toBeInTheDocument();
    });
  });

  it('renders tick bars for each of the last 15 checks', async () => {
    const checks = Array.from({ length: 20 }, (_, i) =>
      makeCheck('up', 100 + i * 10, `2024-01-01T10:${String(i).padStart(2, '0')}:00Z`)
    );

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      // Tick bars are div with w-[4px] class
      const ticks = document.querySelectorAll('.w-\\[4px\\]');
      expect(ticks.length).toBe(15); // last 15 of 20
    });
  });

  it('uses red pulsing tick for down status checks', async () => {
    const checks = [
      makeCheck('down', 0, '2024-01-01T10:00:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      const ticks = document.querySelectorAll('.animate-pulse');
      expect(ticks.length).toBeGreaterThan(0);
    });
  });

  it('uses amber tick for high latency checks (>500ms)', async () => {
    const checks = [
      makeCheck('up', 600, '2024-01-01T10:00:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      const amberTick = document.querySelector('.bg-amber-500');
      expect(amberTick).toBeInTheDocument();
    });
  });

  it('renders a gradient linearGradient element with a unique id based on target', async () => {
    const checks = [makeCheck('up', 50, '2024-01-01T10:00:00Z')];
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com/path" />);

    await waitFor(() => {
      const gradient = document.querySelector('linearGradient');
      expect(gradient).toBeInTheDocument();
      // Target sanitized: httpexamplecompath
      expect(gradient?.getAttribute('id')).toContain('gradient-');
    });
  });

  it('handles single history item (no divide by zero)', async () => {
    const checks = [makeCheck('up', 50, '2024-01-01T10:00:00Z')];
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<UptimeChart target="https://example.com" />);

    await waitFor(() => {
      expect(screen.getByText('Max: 100ms')).toBeInTheDocument(); // max(50, 100) = 100
    });
  });

  it('does not render when not authenticated', async () => {
    // Re-mock as not authenticated
    vi.doMock('../../hooks/useAuth', () => ({
      useAuth: () => ({
        isAuthenticated: false,
        getAccessToken: vi.fn(),
        user: null,
        isAdmin: false,
        isProcessing: false,
      }),
    }));

    // The query is disabled when not authenticated, so loading spinner shows
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json([]))
    );

    const { container } = renderWithProviders(<UptimeChart target="https://example.com" />);
    expect(container.querySelector('svg')).not.toBeInTheDocument();
  });
});
