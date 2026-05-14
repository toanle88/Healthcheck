import { renderHook, waitFor } from '@testing-library/react';
import { useHealthQuery } from './useHealthQuery';
import { createWrapper } from '../test/testUtils';
import { describe, it, expect, vi } from 'vitest';

// Mock our custom useAuth hook
vi.mock('./useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    getAccessToken: vi.fn().mockResolvedValue('fake-token'),
    user: { name: 'Test User' },
  }),
}));

describe('useHealthQuery', () => {
  it('fetches health data successfully', async () => {
    const { result } = renderHook(() => useHealthQuery(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.count).toBe(2);
    expect(result.current.data?.checks[0].target).toBe('https://google.com');
  });
});
