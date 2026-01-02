import axios from 'axios';
import { message } from 'antd';

export const client = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

client.interceptors.response.use(
  (response) => response,
  (error) => {
    // Don't show error for 404 on audit status check as it's expected when no audit is running
    if (error.config.url.includes('/audit/status') && error.response?.status === 404) {
      return Promise.reject(error);
    }
    
    const msg = error.response?.data?.error?.message || error.message;
    // Prevent multiple alerts
    message.destroy(); 
    message.error(`API Error: ${msg}`);
    return Promise.reject(error);
  }
);
