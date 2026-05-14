import { api } from '../lib/axios';
import type { ApiResponse } from '../types';

export const healthService = {
  /**
   * Fetches the current health status of all monitored endpoints.
   */
  getHealthStatus: async (): Promise<ApiResponse> => {
    const { data } = await api.get<ApiResponse>('/api/status');
    return data;
  },
};
