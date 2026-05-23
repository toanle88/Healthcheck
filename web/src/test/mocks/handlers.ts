import { http, HttpResponse } from 'msw';
import { getEnv } from '../../config/env';

const API_BASE_URL = getEnv('VITE_API_URL');

export const handlers = [
  http.get(`${API_BASE_URL}/api/status`, () => {
    return HttpResponse.json({
      checks: [
        {
          name: 'Google',
          target: 'https://google.com',
          status: 'up',
          latency_ms: 45,
          checked_at: new Date().toISOString(),
          uptime_sla: 100.0,
        },
        {
          name: 'GitHub',
          target: 'https://github.com',
          status: 'up',
          latency_ms: 120,
          checked_at: new Date().toISOString(),
          uptime_sla: 100.0,
        },
      ],
      count: 2,
    });
  }),

  http.get(`${API_BASE_URL}/api/targets`, () => {
    return HttpResponse.json([
      { id: 1, name: 'Google', url: 'https://google.com', method: 'GET', expected_status: 200, failure_threshold: 3 },
      { id: 2, name: 'GitHub', url: 'https://github.com', method: 'GET', expected_status: 200, failure_threshold: 3 },
    ]);
  }),

  http.get(`${API_BASE_URL}/api/history`, () => {
    return HttpResponse.json([]);
  }),
];
