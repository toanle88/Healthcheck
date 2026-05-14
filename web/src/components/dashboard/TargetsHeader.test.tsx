import { render, screen } from '@testing-library/react';
import TargetsHeader from './TargetsHeader';
import { describe, it, expect } from 'vitest';

describe('TargetsHeader', () => {
  it('renders count correctly', () => {
    render(<TargetsHeader count={10} />);
    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('Active Targets')).toBeInTheDocument();
  });
});
