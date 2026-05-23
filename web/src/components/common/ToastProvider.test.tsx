import React from 'react';
import { render, screen, act, waitFor, fireEvent } from '@testing-library/react';
import { ToastProvider } from './ToastProvider';
import { useToast } from '../../hooks/useToast';
import { describe, it, expect, vi } from 'vitest';

// Helper component that triggers toasts
const ToastTrigger: React.FC = () => {
  const toast = useToast();
  return (
    <div>
      <button onClick={() => toast.success('Success message')} data-testid="success-btn">Success</button>
      <button onClick={() => toast.error('Error message')} data-testid="error-btn">Error</button>
      <button onClick={() => toast.info('Info message')} data-testid="info-btn">Info</button>
    </div>
  );
};

describe('ToastProvider', () => {
  it('renders children without toasts initially', () => {
    render(
      <ToastProvider>
        <div data-testid="child">Hello</div>
      </ToastProvider>
    );
    expect(screen.getByTestId('child')).toBeInTheDocument();
  });

  it('shows a success toast when toast.success() is called', () => {
    render(
      <ToastProvider>
        <ToastTrigger />
      </ToastProvider>
    );
    fireEvent.click(screen.getByTestId('success-btn'));
    expect(screen.getByText('Success message')).toBeInTheDocument();
  });

  it('shows an error toast when toast.error() is called', () => {
    render(
      <ToastProvider>
        <ToastTrigger />
      </ToastProvider>
    );
    fireEvent.click(screen.getByTestId('error-btn'));
    expect(screen.getByText('Error message')).toBeInTheDocument();
  });

  it('shows an info toast when toast.info() is called', () => {
    render(
      <ToastProvider>
        <ToastTrigger />
      </ToastProvider>
    );
    fireEvent.click(screen.getByTestId('info-btn'));
    expect(screen.getByText('Info message')).toBeInTheDocument();
  });

  it('automatically removes a toast after 4 seconds', async () => {
    vi.useFakeTimers();
    try {
      render(
        <ToastProvider>
          <ToastTrigger />
        </ToastProvider>
      );
      fireEvent.click(screen.getByTestId('success-btn'));
      expect(screen.getByText('Success message')).toBeInTheDocument();

      await act(async () => {
        vi.advanceTimersByTime(4001);
      });

      expect(screen.queryByText('Success message')).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('manually removes a toast when the close button is clicked', () => {
    render(
      <ToastProvider>
        <ToastTrigger />
      </ToastProvider>
    );

    fireEvent.click(screen.getByTestId('success-btn'));
    expect(screen.getByText('Success message')).toBeInTheDocument();

    // First 3 buttons are trigger buttons; last one is the toast X close button
    const allButtons = screen.getAllByRole('button');
    fireEvent.click(allButtons[allButtons.length - 1]);

    expect(screen.queryByText('Success message')).not.toBeInTheDocument();
  });

  it('can show multiple toasts simultaneously', () => {
    render(
      <ToastProvider>
        <ToastTrigger />
      </ToastProvider>
    );
    fireEvent.click(screen.getByTestId('success-btn'));
    fireEvent.click(screen.getByTestId('error-btn'));
    fireEvent.click(screen.getByTestId('info-btn'));

    expect(screen.getByText('Success message')).toBeInTheDocument();
    expect(screen.getByText('Error message')).toBeInTheDocument();
    expect(screen.getByText('Info message')).toBeInTheDocument();
  });
});

describe('useToast', () => {
  it('throws when used outside ToastProvider', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const BrokenComponent = () => {
      useToast();
      return null;
    };

    expect(() => render(<BrokenComponent />)).toThrow(
      'useToast must be used within a ToastProvider'
    );

    consoleSpy.mockRestore();
  });
});
