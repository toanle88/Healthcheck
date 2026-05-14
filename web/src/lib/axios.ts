import axios from 'axios';
import { getEnv } from '../config/env';

const API_BASE_URL = getEnv('VITE_API_URL');

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Helper to set authorization header
export const setAuthToken = (token: string) => {
  api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
};

// Helper to clear authorization header
export const clearAuthToken = () => {
  delete api.defaults.headers.common['Authorization'];
};
