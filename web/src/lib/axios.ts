import axios from 'axios';
import { getEnv } from '../config/env';
import { msalInstance, tokenRequest } from '../authConfig';

const API_BASE_URL = getEnv('VITE_API_URL');

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

const isE2E = typeof window !== 'undefined' && (
  window.location.search.includes("test-mode=true") || 
  localStorage.getItem("playwright-mock-auth") === "true" ||
  (window as unknown as { playwrightMockAuth?: boolean | string }).playwrightMockAuth === true ||
  (window as unknown as { playwrightMockAuth?: boolean | string }).playwrightMockAuth === "true"
);

api.interceptors.request.use(
  async (config) => {
    if (isE2E) {
      config.headers.Authorization = 'Bearer mocked-e2e-token';
      return config;
    }

    try {
      const activeAccount = msalInstance.getActiveAccount();
      if (activeAccount) {
        const response = await msalInstance.acquireTokenSilent({
          ...tokenRequest,
          account: activeAccount,
          redirectUri: window.location.origin + '/blank.html',
        });
        config.headers.Authorization = `Bearer ${response.accessToken}`;
      }
    } catch (err) {
      console.warn('Failed to acquire token silently in Axios interceptor:', err);
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Helpers to set/clear authorization header manually if required
export const setAuthToken = (token: string) => {
  api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
};

export const clearAuthToken = () => {
  delete api.defaults.headers.common['Authorization'];
};


