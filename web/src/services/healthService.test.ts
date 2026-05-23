import { healthService } from './healthService';
import { api } from '../lib/axios';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../lib/axios', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('healthService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getHealthStatus', () => {
    it('calls the correct endpoint and returns data', async () => {
      const mockData = { checks: [], count: 0 };
      vi.mocked(api.get).mockResolvedValue({ data: mockData });

      const result = await healthService.getHealthStatus();

      expect(api.get).toHaveBeenCalledWith('/api/status');
      expect(result).toEqual(mockData);
    });
  });

  describe('getTargets', () => {
    it('calls /api/targets and returns targets list', async () => {
      const mockTargets = [
        { id: 1, name: 'Google', url: 'https://google.com', method: 'GET' },
        { id: 2, name: 'GitHub', url: 'https://github.com', method: 'GET' },
      ];
      vi.mocked(api.get).mockResolvedValue({ data: mockTargets });

      const result = await healthService.getTargets();

      expect(api.get).toHaveBeenCalledWith('/api/targets');
      expect(result).toEqual(mockTargets);
    });
  });

  describe('createTarget', () => {
    it('calls /api/targets with POST and all required fields', async () => {
      const mockTarget = { id: 3, name: 'New Target', url: 'https://new.com', method: 'GET' };
      vi.mocked(api.post).mockResolvedValue({ data: mockTarget });

      const result = await healthService.createTarget('New Target', 'https://new.com');

      expect(api.post).toHaveBeenCalledWith('/api/targets', {
        name: 'New Target',
        url: 'https://new.com',
        method: undefined,
        headers: undefined,
        expected_status: undefined,
        response_contains: undefined,
        failure_threshold: undefined,
      });
      expect(result).toEqual(mockTarget);
    });

    it('sends all optional fields when provided', async () => {
      const mockTarget = { id: 4, name: 'Advanced', url: 'https://adv.com', method: 'POST' };
      vi.mocked(api.post).mockResolvedValue({ data: mockTarget });

      const result = await healthService.createTarget(
        'Advanced',
        'https://adv.com',
        'POST',
        '{"X-Key":"value"}',
        201,
        'ok',
        5
      );

      expect(api.post).toHaveBeenCalledWith('/api/targets', {
        name: 'Advanced',
        url: 'https://adv.com',
        method: 'POST',
        headers: '{"X-Key":"value"}',
        expected_status: 201,
        response_contains: 'ok',
        failure_threshold: 5,
      });
      expect(result).toEqual(mockTarget);
    });
  });

  describe('deleteTarget', () => {
    it('calls DELETE /api/targets/:id', async () => {
      vi.mocked(api.delete).mockResolvedValue({ data: undefined });

      await healthService.deleteTarget(42);

      expect(api.delete).toHaveBeenCalledWith('/api/targets/42');
    });
  });

  describe('getTargetHistory', () => {
    it('calls /api/history with target and default limit', async () => {
      const mockChecks = [
        { status: 'up', latency_ms: 50, checked_at: '2024-01-01T10:00:00Z', uptime_sla: 100 },
      ];
      vi.mocked(api.get).mockResolvedValue({ data: mockChecks });

      const result = await healthService.getTargetHistory('https://example.com');

      expect(api.get).toHaveBeenCalledWith('/api/history', {
        params: { target: 'https://example.com', limit: 30 },
      });
      expect(result).toEqual(mockChecks);
    });

    it('calls /api/history with custom limit', async () => {
      vi.mocked(api.get).mockResolvedValue({ data: [] });

      await healthService.getTargetHistory('https://example.com', 100);

      expect(api.get).toHaveBeenCalledWith('/api/history', {
        params: { target: 'https://example.com', limit: 100 },
      });
    });
  });
});
