import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { setAuthToken, clearAuthToken } from './axios';

// We need to mock the imports that execute on module load
vi.mock('../config/env', () => ({
  getEnv: () => 'http://localhost:8080',
}));

vi.mock('../authConfig', () => ({
  msalInstance: {
    getActiveAccount: vi.fn(),
    acquireTokenSilent: vi.fn(),
  },
  tokenRequest: { scopes: ['api://test/.default'] },
  loginRequest: { scopes: ['openid'] },
}));

describe('axios lib', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearAuthToken();
  });

  afterEach(() => {
    clearAuthToken();
  });

  describe('setAuthToken', () => {
    it('sets the Authorization header on the api instance', async () => {
      setAuthToken('my-bearer-token');
      // Import api lazily so the mock above takes effect
      const { api } = await import('./axios');
      expect(api.defaults.headers.common['Authorization']).toBe('Bearer my-bearer-token');
    });
  });

  describe('clearAuthToken', () => {
    it('removes the Authorization header from api defaults', async () => {
      setAuthToken('my-bearer-token');
      clearAuthToken();
      const { api } = await import('./axios');
      expect(api.defaults.headers.common['Authorization']).toBeUndefined();
    });
  });
});
