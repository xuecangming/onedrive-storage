/**
 * OneDrive Cloud Drive Application
 * 
 * A modern file management web application built with React + TypeScript.
 * This application is completely separated from the middleware and
 * communicates via REST API.
 */

import { ConfigProvider } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import zhCN from 'antd/locale/zh_CN';
import { FilesPage } from './pages';
import './App.css';

// Create React Query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

// Ant Design theme configuration
const theme = {
  token: {
    colorPrimary: '#1677ff',
    borderRadius: 6,
    fontFamily: '"Segoe UI", "PingFang SC", "Microsoft YaHei", system-ui, -apple-system, sans-serif',
  },
};

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider locale={zhCN} theme={theme}>
        <FilesPage />
      </ConfigProvider>
    </QueryClientProvider>
  );
}

export default App;
