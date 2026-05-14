import { render, screen } from '@testing-library/react';
import LoadingSpinner from './LoadingSpinner';
import { describe, it, expect } from 'vitest';

describe('LoadingSpinner', () => {
  it('renders loading text', () => {
    render(<LoadingSpinner />);
    expect(screen.getByText(/Authenticating and loading data/i)).toBeInTheDocument();
  });
});
