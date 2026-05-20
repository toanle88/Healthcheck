import { screen } from '@testing-library/react';
import HealthCard from './HealthCard';
import type { Check } from '../../types';
import { describe, it, expect } from 'vitest';
import { renderWithProviders } from '../../test/testUtils';

const mockCheck: Check = {
  target: 'https://example.com',
  status: 'up',
  latency_ms: 150,
  checked_at: new Date().toISOString(),
  uptime_sla: 100.0,
};

describe('HealthCard', () => {
  it('renders endpoint target and status', () => {
    renderWithProviders(<HealthCard check={mockCheck} />);
    
    expect(screen.getByText('example.com')).toBeInTheDocument();
    expect(screen.getByText('up')).toBeInTheDocument();
    expect(screen.getByText('150ms')).toBeInTheDocument();
  });

  it('renders down status with correct colors', () => {
    const downCheck = { ...mockCheck, status: 'down' };
    renderWithProviders(<HealthCard check={downCheck} />);
    
    const statusBadge = screen.getByText('down');
    expect(statusBadge).toHaveClass('text-red-400');
  });
});
