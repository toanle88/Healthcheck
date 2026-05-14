import { http, HttpResponse } from 'msw';

const API_BASE_URL = import.meta.env.VITE_API_URL || '';

export const handlers = [
  http.get(`${API_BASE_URL}/api/status`, () => {
    return HttpResponse.json({
      checks: [
        {
          target: 'https://google.com',
          status: 'up',
          latency_ms: 45,
          checked_at: new Date().toISOString(),
        },
        {
          target: 'https://github.com',
          status: 'up',
          latency_ms: 120,
          checked_at: new Date().toISOString(),
        },
      ],
      count: 2,
    });
  }),
];
