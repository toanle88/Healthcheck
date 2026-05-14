import { render, screen } from '@testing-library/react';
import Footer from './Footer';
import { describe, it, expect } from 'vitest';

describe('Footer', () => {
  it('renders count correctly', () => {
    render(<Footer count={5} />);
    expect(screen.getByText(/Monitoring 5 endpoints/i)).toBeInTheDocument();
  });

  it('renders version from env', () => {
    render(<Footer count={0} />);
    expect(screen.getByText('Version')).toBeInTheDocument();
  });
});
