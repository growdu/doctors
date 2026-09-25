/**
 * PackagesPage 测试：
 *   - 加载列表 + 行渲染；
 *   - super_admin 显示创建 / 编辑 / 上下架按钮；
 *   - viewer 不显示操作按钮；
 *   - 点击创建 → 弹 Modal；
 *   - 点击编辑 → 弹 Modal；
 *   - 点击上下架 → 调 updatePackage 切换 status。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchPackages: vi.fn(),
  createPackage: vi.fn(),
  updatePackage: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/packages', () => ({
  fetchPackages: mocks.fetchPackages,
  createPackage: mocks.createPackage,
  updatePackage: mocks.updatePackage,
  packageQueryKeys: {
    list: (params?: { status?: string }) => ['packages', params?.status ?? 'all'] as const,
    detail: (id: number | string) => ['package-detail', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import PackagesPage from './PackagesPage';

const FIXTURES = [
  {
    id: 14001,
    name: '半日陪诊',
    price: 300,
    duration: 'half_day' as const,
    status: 'on' as const,
  },
  {
    id: 14003,
    name: '专项陪诊',
    price: 1500,
    duration: 'full_day' as const,
    status: 'off' as const,
  },
];

function setupAuth(role: 'super_admin' | 'order_admin' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(
  initialRole: 'super_admin' | 'order_admin' | 'viewer' = 'super_admin',
) {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/packages']}>
        <Routes>
          <Route path="/packages" element={<PackagesPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('PackagesPage', () => {
  beforeEach(() => {
    mocks.fetchPackages.mockReset();
    mocks.createPackage.mockReset();
    mocks.updatePackage.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchPackages.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('packages-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('半日陪诊')).toBeInTheDocument();
    expect(screen.getByText('专项陪诊')).toBeInTheDocument();
    expect(screen.getByText('上架')).toBeInTheDocument();
    expect(screen.getByText('下架')).toBeInTheDocument();
  });

  it('super_admin 显示创建按钮', async () => {
    mocks.fetchPackages.mockResolvedValue({ data: [], total: 0 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-create')).toBeInTheDocument(),
    );
  });

  it('order_admin 也可操作', async () => {
    mocks.fetchPackages.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('order_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-create')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('btn-edit-14001')).toBeInTheDocument();
    expect(screen.getByTestId('btn-toggle-14001')).toBeInTheDocument();
  });

  it('viewer 不显示创建 / 编辑 / 上下架按钮', async () => {
    mocks.fetchPackages.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('packages-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-create')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-edit-14001')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-toggle-14001')).not.toBeInTheDocument();
  });

  it('点击创建 → 弹 Modal → 输入', async () => {
    const user = userEvent.setup();
    mocks.fetchPackages.mockResolvedValue({ data: [], total: 0 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-create')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-create'));
    await waitFor(() =>
      expect(screen.getByTestId('create-name')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('create-name'), '专项陪诊');
    expect(screen.getByTestId('create-name')).toHaveValue('专项陪诊');
  });

  it('点击上下架 → 调 updatePackage 切换 status', async () => {
    const user = userEvent.setup();
    mocks.fetchPackages.mockResolvedValue({ data: FIXTURES, total: 2 });
    mocks.updatePackage.mockResolvedValue({ ...FIXTURES[0], status: 'off' });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-toggle-14001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-toggle-14001'));
    await waitFor(() => {
      expect(mocks.updatePackage).toHaveBeenCalledWith(
        14001,
        expect.objectContaining({ status: 'off' }),
      );
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已更新套餐 #14001'),
      );
    });
  });

  it('点击编辑 → 弹 Modal → name 已填充', async () => {
    const user = userEvent.setup();
    mocks.fetchPackages.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-edit-14001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-edit-14001'));
    await waitFor(() =>
      expect(screen.getByTestId('edit-name')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('edit-name')).toHaveValue('半日陪诊');
  });
});