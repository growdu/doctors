import { ConfigProvider, App as AntdApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { AppRouter } from './router';

// 单一 QueryClient 实例：refetchOnWindowFocus 默认 true（订单/退款队列后台刷新友好）
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5_000,
    },
  },
});

/**
 * App 根组件：按 spec §6 主题（主色 #1677ff）+ §3 RBAC 守卫 + §5 TanStack Query。
 * 负责装配：ConfigProvider(zhCN) → QueryClientProvider → AntdApp → BrowserRouter → AppRouter
 */
export default function App() {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#1677ff',
          colorWarning: '#faad14',
          colorSuccess: '#52c41a',
          colorError: '#ff4d4f',
          borderRadius: 6,
        },
      }}
    >
      <QueryClientProvider client={queryClient}>
        <AntdApp>
          <BrowserRouter>
            <AppRouter />
          </BrowserRouter>
        </AntdApp>
      </QueryClientProvider>
    </ConfigProvider>
  );
}