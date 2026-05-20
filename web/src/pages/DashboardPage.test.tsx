import { screen, waitFor } from '@testing-library/react';
import DashboardPage from './DashboardPage';
import { renderWithProviders } from '../test/testUtils';
import { describe, it, expect, vi } from 'vitest';

// Mock our custom useAuth hook
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    getAccessToken: vi.fn().mockResolvedValue('fake-token'),
    user: { name: 'Test User' },
    isProcessing: false,
  }),
}));

describe('DashboardPage', () => {
  it('renders loading state then data', async () => {
    renderWithProviders(<DashboardPage />);
    
    // Check for loading state (this will be from useHealthQuery's isLoading)
    expect(screen.getByText(/Authenticating and loading data/i)).toBeInTheDocument();

    // Wait for data to load
    await waitFor(() => {
      expect(screen.getByText('Google')).toBeInTheDocument();
    });

    expect(screen.getByText('GitHub')).toBeInTheDocument();
    expect(screen.getByText(/Monitoring 2 endpoints/i)).toBeInTheDocument();
  });
});
