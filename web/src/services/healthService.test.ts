import { healthService } from './healthService';
import { api } from '../lib/axios';
import { describe, it, expect, vi } from 'vitest';

vi.mock('../lib/axios', () => ({
  api: {
    get: vi.fn(),
  },
}));

describe('healthService', () => {
  it('calls the correct endpoint and returns data', async () => {
    const mockData = { checks: [], count: 0 };
    vi.mocked(api.get).mockResolvedValue({ data: mockData });

    const result = await healthService.getHealthStatus();

    expect(api.get).toHaveBeenCalledWith('/api/status');
    expect(result).toEqual(mockData);
  });
});
