import { screen, fireEvent } from '@testing-library/react';
import Header from './Header';
import { renderWithProviders } from '../../test/testUtils';
import { describe, it, expect, vi } from 'vitest';

// Mock useAuth
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { name: 'Test User' },
    logout: vi.fn(),
  }),
}));

describe('Header', () => {
  const defaultProps = {
    error: false,
    lastUpdated: new Date(),
    isRefreshing: false,
    onRefresh: vi.fn(),
  };

  it('renders title and system status', () => {
    renderWithProviders(<Header {...defaultProps} />);
    
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('System Operational')).toBeInTheDocument();
  });

  it('shows user name', () => {
    renderWithProviders(<Header {...defaultProps} />);
    expect(screen.getByText('Test User')).toBeInTheDocument();
  });

  it('calls onRefresh when refresh button is clicked', () => {
    renderWithProviders(<Header {...defaultProps} />);
    
    const refreshBtn = screen.getByTitle('Refresh Data');
    fireEvent.click(refreshBtn);
    expect(defaultProps.onRefresh).toHaveBeenCalled();
  });
});
