import { api } from '../lib/axios';
import type { ApiResponse, Target, Check } from '../types';

export const healthService = {
  /**
   * Fetches the current health status of all monitored endpoints.
   */
  getHealthStatus: async (): Promise<ApiResponse> => {
    const { data } = await api.get<ApiResponse>('/api/status');
    return data;
  },

  getTargets: async (): Promise<Target[]> => {
    const { data } = await api.get<Target[]>('/api/targets');
    return data;
  },

  createTarget: async (
    name: string,
    url: string,
    method?: string,
    headers?: string,
    expectedStatus?: number,
    responseContains?: string,
    failureThreshold?: number
  ): Promise<Target> => {
    const { data } = await api.post<Target>('/api/targets', {
      name,
      url,
      method,
      headers,
      expected_status: expectedStatus,
      response_contains: responseContains,
      failure_threshold: failureThreshold,
    });
    return data;
  },

  deleteTarget: async (id: number): Promise<void> => {
    await api.delete(`/api/targets/${id}`);
  },

  getTargetHistory: async (target: string, limit = 30): Promise<Check[]> => {
    const { data } = await api.get<Check[]>('/api/history', {
      params: { target, limit },
    });
    return data;
  },
};
