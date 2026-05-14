import { useQuery } from '@tanstack/react-query';
import { useAuth } from './useAuth';
import { setAuthToken } from '../lib/axios';
import { healthService } from '../services/healthService';
import type { ApiResponse } from '../types';

export const useHealthQuery = () => {
  const { isAuthenticated, getAccessToken } = useAuth();

  return useQuery<ApiResponse>({
    queryKey: ['healthStatus'],
    queryFn: async () => {
      // 1. Get Token (handles silent/popup logic internally)
      const token = await getAccessToken();

      // 2. Call API via Service
      setAuthToken(token);
      return healthService.getHealthStatus();
    },
    enabled: isAuthenticated,
    refetchInterval: 10000,
    retry: 2,
  });
};
