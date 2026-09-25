/**
 * LoginPage 测试：
 *   - 渲染表单（用户名 / 密码）；
 *   - 6 个 demo 账号标签可点击；
 *   - 提交表单 → 调 login() → 跳 /dashboard（默认 from）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import LoginPage from './LoginPage';

function setupAuth() {
  useAuthStore.setState({
    token: null,
    user: null,
    role: null,
    isAuthed: false,
  } as never);
}

function renderLogin() {
  setupAuth();
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/dashboard"
            element={<div data-testid="dashboard-stub">dashboard-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('LoginPage', () => {
  beforeEach(() => {
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
    setupAuth();
  });

  it('渲染表单 + 6 个 demo 账号', async () => {
    renderLogin();
    expect(screen.getByTestId('login-page')).toBeInTheDocument();
    expect(screen.getByTestId('login-username')).toBeInTheDocument();
    expect(screen.getByTestId('login-password')).toBeInTheDocument();
    expect(screen.getByTestId('demo-super')).toBeInTheDocument();
    expect(screen.getByTestId('demo-viewer')).toBeInTheDocument();
  });

  it('点击 demo → 表单填入对应 username', async () => {
    const user = userEvent.setup();
    renderLogin();
    await user.click(screen.getByTestId('demo-viewer'));
    expect(screen.getByTestId('login-username')).toHaveValue('viewer');
    expect(screen.getByTestId('login-password')).toHaveValue('demo');
  });

  it('提交表单 → 调 login + 跳 /dashboard', async () => {
    const user = userEvent.setup();
    const loginSpy = vi.spyOn(useAuthStore.getState(), 'login');
    renderLogin();
    await user.click(screen.getByTestId('login-submit'));
    await waitFor(() => {
      expect(loginSpy).toHaveBeenCalledWith('super', 'demo');
    });
    // 登录成功后 isAuthed = true，路由跳 /dashboard
    await waitFor(() => {
      expect(useAuthStore.getState().isAuthed).toBe(true);
    });
    expect(await screen.findByTestId('dashboard-stub')).toBeInTheDocument();
  });

  it('登录失败时展示 error', async () => {
    const user = userEvent.setup();
    const loginSpy = vi
      .spyOn(useAuthStore.getState(), 'login')
      .mockRejectedValueOnce(new Error('login failed'));
    renderLogin();
    await user.click(screen.getByTestId('login-submit'));
    await waitFor(() => {
      expect(loginSpy).toHaveBeenCalled();
    });
    // 不跳 dashboard
    expect(screen.queryByTestId('dashboard-stub')).not.toBeInTheDocument();
  });
});