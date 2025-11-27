/**
 * API Client for OneDrive Cloud Storage
 * 
 * This module provides a clean interface for communicating with the middleware API.
 * The cloud drive application is completely separated from the middleware.
 */

import axios from 'axios';
import type { AxiosInstance, AxiosResponse } from 'axios';

// Default API base URL - use relative path to leverage Vite proxy
// This works in Codespaces and local development
const DEFAULT_API_URL = '/api/v1';

// Get API URL from localStorage or use default
const getApiBaseUrl = (): string => {
  const savedUrl = localStorage.getItem('api_base_url');
  return savedUrl || DEFAULT_API_URL;
};

// Create axios instance
const createApiClient = (): AxiosInstance => {
  const client = axios.create({
    baseURL: getApiBaseUrl(),
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Response interceptor for error handling
  client.interceptors.response.use(
    (response: AxiosResponse) => response,
    (error) => {
      if (error.response) {
        // Server responded with error
        const message = error.response.data?.error?.message || error.response.data?.message || '请求失败';
        return Promise.reject(new Error(message));
      } else if (error.request) {
        // No response received
        return Promise.reject(new Error('无法连接到服务器，请检查网络或 API 地址配置'));
      }
      return Promise.reject(error);
    }
  );

  return client;
};

let apiClient = createApiClient();

// Reset client (useful when API URL changes)
export const resetApiClient = (): void => {
  apiClient = createApiClient();
};

// Set API base URL
export const setApiBaseUrl = (url: string): void => {
  localStorage.setItem('api_base_url', url.replace(/\/+$/, ''));
  resetApiClient();
};

// Get current API base URL
export const getCurrentApiUrl = (): string => {
  return getApiBaseUrl();
};

export default apiClient;
