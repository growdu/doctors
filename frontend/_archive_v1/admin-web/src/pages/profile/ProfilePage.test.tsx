/**
 * ProfilePage 测试：
 *   - 渲染当前用户信息（avatar / display_name / role）；
 *   - 修改密码表单 + 提交（mock）；
 *   - 两次密码不一致时校验失败；
 *   - 点击退出登录 → 清状态 + 跳 /login。
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

import ProfilePage from './ProfilePage';

function setupAuth(role: 'super_admin' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: {
      id: 1001,
      username: 'admin',
      display_name: '管理员甲',
      role,
      avatar_url: null,
    },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(role: 'super_admin' | 'viewer' = 'super_admin') {
  setupAuth(role);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/profile']}>
        <Routes>
          <Route path="/profile" element={<ProfilePage />} />
          <Route
            path="/login"
            element={<div data-testid="login-stub">login-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('ProfilePage', () => {
  beforeEach(() => {
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('渲染当前用户基本信息', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('profile-page')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('profile-name')).toHaveTextContent('管理员甲');
    expect(screen.getByTestId('profile-role')).toHaveTextContent('超级管理员');
    expect(screen.getByTestId('profile-avatar')).toBeInTheDocument();
  });

  it('viewer 角色显示「只读观察」标签', async () => {
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('profile-role')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('profile-role')).toHaveTextContent('只读观察');
  });

  it('修改密码：填表 + 提交（mock）', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('pw-old')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('pw-old'), 'old123456');
    await user.type(screen.getByTestId('pw-new'), 'new123456');
    await user.type(screen.getByTestId('pw-confirm'), 'new123456');
    await user.click(screen.getByTestId('btn-change-password'));
    // 表单 resetFields 是同步的，但这里仅校验不报错（不调真实接口）
    await waitFor(() => {
      expect(screen.getByTestId('pw-old')).toHaveValue('');
    });
  });

  it('两次新密码不一致 → 校验提示', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('pw-old')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('pw-old'), 'old123456');
    await user.type(screen.getByTestId('pw-new'), 'new123456');
    await user.type(screen.getByTestId('pw-confirm'), 'diff999');
    await user.click(screen.getByTestId('btn-change-password'));
    await waitFor(() => {
      expect(screen.getByText('两次输入的密码不一致')).toBeInTheDocument();
    });
  });

  it('点击退出登录 → 跳 /login + 清状态', async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('btn-logout')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-logout'));
    expect(await screen.findByTestId('login-stub')).toBeInTheDocument();
    const state = useAuthStore.getState();
    expect(state.isAuthed).toBe(false);
  });
});