import { screen, waitFor, fireEvent } from '@testing-library/react';
import DashboardPage from './DashboardPage';
import { renderWithProviders } from '../test/testUtils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '../test/setup';
import { getEnv } from '../config/env';

const API_BASE_URL = getEnv('VITE_API_URL');

// Default mock: authenticated admin user
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    getAccessToken: vi.fn().mockResolvedValue('fake-token'),
    user: { name: 'Test User' },
    isProcessing: false,
    isAdmin: true,
  }),
}));

// Mock EventSource globally (arrow functions can't be constructors)
const mockClose = vi.fn();
class MockEventSource {
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  close = mockClose;
  constructor() {}
}
Object.defineProperty(window, 'EventSource', { value: MockEventSource, writable: true });

const defaultStatusResponse = {
  checks: [
    { name: 'Google', target: 'https://google.com', status: 'up', latency_ms: 45, checked_at: new Date().toISOString(), uptime_sla: 100.0 },
    { name: 'GitHub', target: 'https://github.com', status: 'up', latency_ms: 120, checked_at: new Date().toISOString(), uptime_sla: 100.0 },
  ],
  count: 2,
};

const defaultTargetsResponse = [
  { id: 1, name: 'Google', url: 'https://google.com', method: 'GET', expected_status: 200, failure_threshold: 3 },
  { id: 2, name: 'GitHub', url: 'https://github.com', method: 'GET', expected_status: 200, failure_threshold: 3 },
];

beforeEach(() => {
  vi.clearAllMocks();
  server.use(
    http.get(`${API_BASE_URL}/api/status`, () => HttpResponse.json(defaultStatusResponse)),
    http.get(`${API_BASE_URL}/api/targets`, () => HttpResponse.json(defaultTargetsResponse)),
    http.get(`${API_BASE_URL}/api/history`, () => HttpResponse.json([]))
  );
});

describe('DashboardPage', () => {
  it('renders loading state then displays health cards', async () => {
    renderWithProviders(<DashboardPage />);

    expect(screen.getByText(/Authenticating and loading data/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Google')).toBeInTheDocument();
    });

    expect(screen.getByText('GitHub')).toBeInTheDocument();
    expect(screen.getByText(/Monitoring 2 endpoints/i)).toBeInTheDocument();
  });

  it('renders error display when health query fails', async () => {
    // The QueryClient in testUtils has retry: false, so error shows immediately
    server.use(
      http.get(`${API_BASE_URL}/api/status`, () => new HttpResponse(null, { status: 500 }))
    );

    renderWithProviders(<DashboardPage />);

    // ErrorDisplay renders "Retry Connection" button
    await waitFor(() => {
      expect(screen.getByText('Retry Connection')).toBeInTheDocument();
    }, { timeout: 5000 });
  });

  it('shows "Manage Targets" button for admin users', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: /Manage Targets/i })).toBeInTheDocument();
  });

  it('toggles the manage panel open on "Manage Targets" click', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));

    await waitFor(() => {
      expect(screen.getByText('Configure Targets')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /Hide Settings/i })).toBeInTheDocument();
  });

  it('hides manage panel when "Hide Settings" is clicked', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));
    await waitFor(() => expect(screen.getByText('Configure Targets')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Hide Settings/i }));
    await waitFor(() => {
      expect(screen.queryByText('Configure Targets')).not.toBeInTheDocument();
    });
  });

  it('shows current targets in the manage panel', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));
    await waitFor(() => expect(screen.getByText('Current Targets')).toBeInTheDocument());
  });

  it('shows "No targets configured" when targets list is empty', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/targets`, () => HttpResponse.json([]))
    );
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));
    await waitFor(() => {
      expect(screen.getByText(/No targets configured/i)).toBeInTheDocument();
    });
  });

  it('successfully adds a target via the form', async () => {
    server.use(
      http.post(`${API_BASE_URL}/api/targets`, () =>
        HttpResponse.json({ id: 99, name: 'New Service', url: 'https://new.com', method: 'GET', expected_status: 200, failure_threshold: 3 })
      )
    );
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));
    await waitFor(() => expect(screen.getByText('Configure Targets')).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText('e.g. Google Search'), {
      target: { value: 'New Service' },
    });
    fireEvent.change(screen.getByPlaceholderText('https://example.com'), {
      target: { value: 'https://new.com' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Add Target/i }));

    await waitFor(() => {
      expect(screen.getByText('Target added successfully')).toBeInTheDocument();
    });
  });

  it('shows JSON validation error for invalid custom headers', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));
    await waitFor(() => expect(screen.getByText('Configure Targets')).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText('e.g. Google Search'), {
      target: { value: 'Test Service' },
    });
    fireEvent.change(screen.getByPlaceholderText('https://example.com'), {
      target: { value: 'https://test.com' },
    });
    // Use the textarea by its ID (defined in DashboardPage)
    fireEvent.change(document.getElementById('target-headers')!, {
      target: { value: 'not-valid-json' },
    });

    fireEvent.click(screen.getByRole('button', { name: /Add Target/i }));

    await waitFor(() => {
      // The error appears in both the form inline error and the toast
      const msgs = screen.getAllByText(/Custom Headers must be valid JSON/i);
      expect(msgs.length).toBeGreaterThan(0);
    });
  });

  it('shows error toast when add target API call fails', async () => {
    server.use(
      http.post(`${API_BASE_URL}/api/targets`, () =>
        HttpResponse.json({ error: 'Duplicate URL' }, { status: 409 })
      )
    );
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));
    await waitFor(() => expect(screen.getByText('Configure Targets')).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText('e.g. Google Search'), {
      target: { value: 'Test' },
    });
    fireEvent.change(screen.getByPlaceholderText('https://example.com'), {
      target: { value: 'https://test.com' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Add Target/i }));

    // The error appears in both the inline form error and the toast
    await waitFor(() => {
      const msgs = screen.getAllByText('Duplicate URL');
      expect(msgs.length).toBeGreaterThan(0);
    }, { timeout: 5000 });
  });

  it('opens IncidentLogModal when a health card is clicked', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByText('Google'));

    await waitFor(() => {
      expect(screen.getByTitle('Close')).toBeInTheDocument();
    });
  });

  it('closes IncidentLogModal when the close button is clicked', async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByText('Google'));
    await waitFor(() => expect(screen.getByTitle('Close')).toBeInTheDocument());

    fireEvent.click(screen.getByTitle('Close'));
    await waitFor(() => {
      expect(screen.queryByTitle('Close')).not.toBeInTheDocument();
    });
  });

  it('shows empty state message when checks array is empty', async () => {
    server.use(
      http.get(`${API_BASE_URL}/api/status`, () => HttpResponse.json({ checks: [], count: 0 }))
    );
    renderWithProviders(<DashboardPage />);
    await waitFor(() => {
      expect(screen.getByText(/No data received yet/i)).toBeInTheDocument();
    });
  });

  it('deletes a target from the manage panel on button click', async () => {
    server.use(
      http.delete(`${API_BASE_URL}/api/targets/:id`, () => new HttpResponse(null, { status: 204 }))
    );
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByText('Google')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /Manage Targets/i }));

    // The targets query is enabled once the panel is open; wait for delete buttons
    await waitFor(() => {
      const deleteBtns = screen.queryAllByTitle('Delete Target');
      expect(deleteBtns.length).toBeGreaterThan(0);
    }, { timeout: 5000 });

    fireEvent.click(screen.getAllByTitle('Delete Target')[0]);

    await waitFor(() => {
      expect(screen.getByText('Target deleted successfully')).toBeInTheDocument();
    });
  });
});
