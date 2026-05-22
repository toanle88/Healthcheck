import { screen } from '@testing-library/react';
import HealthCard from './HealthCard';
import type { Check } from '../../types';
import { describe, it, expect } from 'vitest';
import { renderWithProviders } from '../../test/testUtils';

const mockCheck: Check = {
  name: 'Example Service',
  target: 'https://example.com',
  status: 'up',
  latency_ms: 150,
  checked_at: new Date().toISOString(),
  uptime_sla: 100.0,
  failure_threshold: 3,
  consecutive_failures: 0,
  last_alert_status: 'up',
};

describe('HealthCard', () => {
  it('renders endpoint target and status', () => {
    renderWithProviders(<HealthCard check={mockCheck} />);
    
    expect(screen.getByText('Example Service')).toBeInTheDocument();
    expect(screen.getByText('https://example.com')).toBeInTheDocument();
    expect(screen.getByText('up')).toBeInTheDocument();
    expect(screen.getByText('150ms')).toBeInTheDocument();
  });

  it('renders transient failure warning with correct colors', () => {
    const transientCheck: Check = {
      ...mockCheck,
      status: 'down',
      consecutive_failures: 1,
      failure_threshold: 3,
    };
    renderWithProviders(<HealthCard check={transientCheck} />);
    
    const statusBadge = screen.getByText('FAILING (1/3)');
    expect(statusBadge).toHaveClass('text-amber-400');
  });

  it('renders confirmed down alert with correct colors', () => {
    const downCheck: Check = {
      ...mockCheck,
      status: 'down',
      consecutive_failures: 3,
      failure_threshold: 3,
    };
    renderWithProviders(<HealthCard check={downCheck} />);
    
    const statusBadge = screen.getByText('DOWN (3)');
    expect(statusBadge).toHaveClass('text-red-400');
  });
});
