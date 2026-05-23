import { screen, waitFor, fireEvent } from '@testing-library/react';
import IncidentLogModal from './IncidentLogModal';
import { renderWithProviders } from '../../test/testUtils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../../test/setup';
import { getEnv } from '../../config/env';

const API_BASE_URL = getEnv('VITE_API_URL');

// Mock useAuth to avoid MSAL complexity
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    getAccessToken: vi.fn().mockResolvedValue('fake-token'),
    user: { name: 'Test User' },
    isAdmin: true,
    isProcessing: false,
  }),
}));

const defaultProps = {
  target: 'https://example.com',
  name: 'Example Service',
  onClose: vi.fn(),
};

const makeCheck = (
  status: 'up' | 'down' | 'pending',
  latency_ms: number,
  checked_at: string,
  uptime_sla = 99.9
) => ({
  status,
  latency_ms,
  checked_at,
  uptime_sla,
  target: 'https://example.com',
  name: 'Example Service',
});

describe('IncidentLogModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, async () => {
        await new Promise((r) => setTimeout(r, 200));
        return HttpResponse.json([]);
      })
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);
    expect(screen.getByText(/Fetching SRE telemetry/i)).toBeInTheDocument();
  });

  it('shows error state when history fetch fails', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => {
        return new HttpResponse(null, { status: 500 });
      })
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText(/Telemetry Load Failed/i)).toBeInTheDocument();
    });
    expect(screen.getByText(/Could not fetch target status log history/i)).toBeInTheDocument();
  });

  it('renders modal header with target name and URL', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json([]))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    expect(screen.getByText('Example Service')).toBeInTheDocument();
    expect(screen.getByText('https://example.com')).toBeInTheDocument();
  });

  it('calls onClose when the close button is clicked', async () => {
    const onClose = vi.fn();
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json([]))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} onClose={onClose} />);
    fireEvent.click(screen.getByTitle('Close'));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('shows telemetry quick cards with computed uptime SLA, latency, and outages', async () => {
    const checks = [
      makeCheck('up', 45, '2024-01-01T10:00:00Z', 99.5),
      makeCheck('up', 120, '2024-01-01T10:01:00Z', 99.5),
      makeCheck('down', 0, '2024-01-01T10:02:00Z', 99.5),
      makeCheck('up', 80, '2024-01-01T10:03:00Z', 99.5),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('Uptime SLA', { exact: false })).toBeInTheDocument();
    });

    // Avg latency: (45+120+0+80)/4 = 61ms (down checks ARE included; only 'pending' excluded)
    expect(screen.getByText('61ms')).toBeInTheDocument();
    // 1 incident (down→up sequence)
    expect(screen.getByText('1')).toBeInTheDocument();
  });

  it('shows "No incidents detected" when all checks are up', async () => {
    const checks = [
      makeCheck('up', 45, '2024-01-01T10:00:00Z'),
      makeCheck('up', 100, '2024-01-01T10:01:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText(/All green. No incidents detected/i)).toBeInTheDocument();
    });
  });

  it('shows ongoing incident when last check is down', async () => {
    const checks = [
      makeCheck('up', 50, '2024-01-01T10:00:00Z'),
      makeCheck('down', 0, '2024-01-01T10:01:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('Ongoing')).toBeInTheDocument();
    });
  });

  it('renders raw check log table with status badges', async () => {
    const checks = [
      makeCheck('up', 45, '2024-01-01T10:00:00Z'),
      makeCheck('down', 0, '2024-01-01T10:01:00Z'),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('Raw Check Logs')).toBeInTheDocument();
    });

    expect(screen.getByText('up')).toBeInTheDocument();
    expect(screen.getByText('down')).toBeInTheDocument();
  });

  it('renders latency histogram with all buckets', async () => {
    const checks = [
      makeCheck('up', 30, '2024-01-01T10:00:00Z'),   // 0-50ms
      makeCheck('up', 75, '2024-01-01T10:01:00Z'),   // 50-100ms
      makeCheck('up', 150, '2024-01-01T10:02:00Z'),  // 100-250ms
      makeCheck('up', 300, '2024-01-01T10:03:00Z'),  // 250-500ms
      makeCheck('up', 750, '2024-01-01T10:04:00Z'),  // 500-1000ms
      makeCheck('up', 1500, '2024-01-01T10:05:00Z'), // 1000ms+
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('Max Latency Histogram (Last 100 pings)')).toBeInTheDocument();
    });

    expect(screen.getByText('0-50ms')).toBeInTheDocument();
    expect(screen.getByText('50-100ms')).toBeInTheDocument();
    expect(screen.getByText('100-250ms')).toBeInTheDocument();
    expect(screen.getByText('250-500ms')).toBeInTheDocument();
    expect(screen.getByText('500-1000ms')).toBeInTheDocument();
    expect(screen.getByText('1000ms+')).toBeInTheDocument();
  });

  it('displays 100.00% uptime when no checks exist', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json([]))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('100.00%')).toBeInTheDocument();
    });
  });

  it('shows resolved incident with duration in seconds', async () => {
    const checks = [
      makeCheck('up', 50, '2024-01-01T10:00:00Z'),
      makeCheck('down', 0, '2024-01-01T10:00:30Z'),
      makeCheck('up', 50, '2024-01-01T10:01:00Z'), // 30 seconds outage
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('30s')).toBeInTheDocument();
    });
  });

  it('shows formatted duration in minutes and seconds', async () => {
    const start = new Date('2024-01-01T10:00:00Z');
    const end = new Date(start.getTime() + 90_000); // 1m 30s
    const checks = [
      makeCheck('up', 50, '2024-01-01T09:59:00Z'),
      makeCheck('down', 0, start.toISOString()),
      makeCheck('up', 50, end.toISOString()),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('1m 30s')).toBeInTheDocument();
    });
  });

  it('shows formatted duration in hours and minutes', async () => {
    const start = new Date('2024-01-01T10:00:00Z');
    const end = new Date(start.getTime() + 3_900_000); // 1h 5m
    const checks = [
      makeCheck('up', 50, '2024-01-01T09:59:00Z'),
      makeCheck('down', 0, start.toISOString()),
      makeCheck('up', 50, end.toISOString()),
    ];

    server.use(
      http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json(checks))
    );

    renderWithProviders(<IncidentLogModal {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('1h 5m')).toBeInTheDocument();
    });
  });
});
