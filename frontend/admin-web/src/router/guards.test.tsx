/**
 * Router guards 测试契约（待 vitest 启用后跑）：
 *   - AuthGuard 未登录跳 /login（Navigate replace）；
 *   - AuthGuard 已登录 render children / Outlet；
 *   - RequireRole role 不在列表 → 渲染 403；
 *   - RequireRole role 在列表 → render children；
 *   - RequireRole 未登录（即便外层 AuthGuard 漏挂）→ 跳 /login。
 *
 * 注：测试用 vi.mock('@/stores/authStore') 替换 store。
 * 当前骨架未装 vitest / @testing-library/react，本文件为契约样。
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

// mock authStore
const mockState: {
  isAuthed: boolean;
  role: string | null;
} = { isAuthed: false, role: null };

vi.mock('@/stores/authStore', () => ({
  useAuthStore: (selector: (s: typeof mockState) => unknown) => selector(mockState),
  ADMIN_ROLES: ['super_admin', 'order_admin', 'refund_admin', 'audit_admin', 'cs', 'viewer'],
}));

import { AuthGuard, RequireRole } from './guards';

describe('AuthGuard', () => {
  it('未登录 → 渲染 /login（Navigate）', () => {
    mockState.isAuthed = false;
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route path="/login" element={<div data-testid="login">login</div>} />
          <Route element={<AuthGuard />}>
            <Route path="/dashboard" element={<div data-testid="dash">dash</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByTestId('login')).toBeInTheDocument();
    expect(screen.queryByTestId('dash')).toBeNull();
  });

  it('已登录 → render children', () => {
    mockState.isAuthed = true;
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route element={<AuthGuard />}>
            <Route path="/dashboard" element={<div data-testid="dash">dash</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByTestId('dash')).toBeInTheDocument();
  });
});

describe('RequireRole', () => {
  it('role 在 roles 列表 → render children', () => {
    mockState.isAuthed = true;
    mockState.role = 'super_admin';
    render(
      <MemoryRouter initialEntries={['/admin/order']}>
        <Routes>
          <Route element={<RequireRole roles={['super_admin', 'order_admin']} />}>
            <Route path="/admin/order" element={<div data-testid="ok">ok</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByTestId('ok')).toBeInTheDocument();
  });

  it('role 不在列表 → 渲染 403', () => {
    mockState.isAuthed = true;
    mockState.role = 'viewer';
    render(
      <MemoryRouter initialEntries={['/admin/order']}>
        <Routes>
          <Route element={<RequireRole roles={['super_admin']} />}>
            <Route path="/admin/order" element={<div data-testid="ok">ok</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByTestId('rbac-403')).toBeInTheDocument();
    expect(screen.queryByTestId('ok')).toBeNull();
  });

  it('未登录 → 跳 /login（兜底）', () => {
    mockState.isAuthed = false;
    mockState.role = null;
    render(
      <MemoryRouter initialEntries={['/admin/order']}>
        <Routes>
          <Route path="/login" element={<div data-testid="login">login</div>} />
          <Route element={<RequireRole roles={['super_admin']} />}>
            <Route path="/admin/order" element={<div data-testid="ok">ok</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByTestId('login')).toBeInTheDocument();
  });
});