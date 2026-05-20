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
];
