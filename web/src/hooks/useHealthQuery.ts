import { useQuery } from '@tanstack/react-query';
import { useAuth } from './useAuth';
import { healthService } from '../services/healthService';
import type { ApiResponse } from '../types';

export const useHealthQuery = () => {
  const { isAuthenticated } = useAuth();

  return useQuery<ApiResponse>({
    queryKey: ['healthStatus'],
    queryFn: () => healthService.getHealthStatus(),
    enabled: isAuthenticated,
    refetchInterval: 10000,
    retry: 2,
  });
};

