/**
 * admin-web 入口（v1 完整装配）。
 *
 * 装配顺序：
 *   StrictMode
 *     └── ConfigProvider(zhCN, #1677ff 主题)
 *           └── QueryClientProvider
 *                 └── AntdApp
 *                       └── ErrorBoundary
 *                             └── AuthBootstrap（一次性 bootstrap）
 *                                   └── BrowserRouter
 *                                         └── AppRouter
 *
 * AuthBootstrap：
 *   - 调 useAuthStore.bootstrap() 从 localStorage 恢复 session；
 *   - 失败（role 与 user.role 不一致）抛 11003 → 自动 logout 清状态；
 *   - 失败也不会阻塞渲染（用户会被 AuthGuard 跳到 /login）。
 *
 * MSW 启动：
 *   - dev 环境（import.meta.env.DEV）启动 mock service worker；
 *   - 生产环境不启动（构建产物走真实后端）。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §6
 */
import { useEffect } from 'react';
import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider, App as AntdApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { AppRouter } from './router';
import { ErrorBoundary } from './components/ErrorBoundary';
import { useAuthStore } from './stores/authStore';

// 单一 QueryClient 实例
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5_000,
    },
  },
});

/** 应用启动时一次性 bootstrap auth session */
function AuthBootstrap({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    try {
      useAuthStore.getState().bootstrap();
    } catch (e) {
      // 11003 admin_forbidden 等：清状态后让 AuthGuard 接管跳 /login
      // eslint-disable-next-line no-console
      console.warn('[AuthBootstrap]', e);
      useAuthStore.getState().logout();
    }
  }, []);
  return <>{children}</>;
}

/** dev 环境启动 MSW */
async function maybeStartMock(): Promise<void> {
  if (!import.meta.env.DEV) return;
  try {
    const { startMockServiceWorker } = await import('./mocks/browser');
    await startMockServiceWorker();
  } catch (e) {
    // eslint-disable-next-line no-console
    console.warn('[MSW] failed to start (dev only)', e);
  }
}

function Root() {
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
          <ErrorBoundary>
            <AuthBootstrap>
              <BrowserRouter>
                <AppRouter />
              </BrowserRouter>
            </AuthBootstrap>
          </ErrorBoundary>
        </AntdApp>
      </QueryClientProvider>
    </ConfigProvider>
  );
}

void maybeStartMock();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Root />
  </React.StrictMode>,
);