import { render, screen, fireEvent } from '@testing-library/react';
import ErrorDisplay from './ErrorDisplay';
import { describe, it, expect, vi } from 'vitest';

describe('ErrorDisplay', () => {
  it('renders error message and retry button', () => {
    const mockRetry = vi.fn();
    render(<ErrorDisplay error="Failed to fetch" onRetry={mockRetry} />);
    
    expect(screen.getByText('Failed to fetch')).toBeInTheDocument();
    
    const retryBtn = screen.getByText('Retry Connection');
    fireEvent.click(retryBtn);
    expect(mockRetry).toHaveBeenCalled();
  });
});
