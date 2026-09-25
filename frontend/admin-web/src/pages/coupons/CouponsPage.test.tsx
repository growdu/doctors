/**
 * CouponsPage 测试：
 *   - 加载列表 + 行渲染（金额 / 折扣 + 状态）；
 *   - super_admin 显示创建 / 停用按钮；
 *   - viewer 不显示任何操作按钮；
 *   - 点击创建 → 弹 Modal → 输入 → 调 createCoupon；
 *   - 点击停用 → 弹 confirm → 调 disableCoupon。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchCoupons: vi.fn(),
  createCoupon: vi.fn(),
  disableCoupon: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/coupons', () => ({
  fetchCoupons: mocks.fetchCoupons,
  createCoupon: mocks.createCoupon,
  disableCoupon: mocks.disableCoupon,
  couponQueryKeys: {
    list: (params?: { type?: string }) => ['coupons', params?.type ?? 'all'] as const,
    detail: (id: number | string) => ['coupon-detail', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import CouponsPage from './CouponsPage';

const FIXTURES = [
  {
    id: 15001,
    name: '新人 50 元券',
    type: 'amount_off' as const,
    value: 50,
    valid_until: '2026-12-31T23:59:59Z',
    disabled: false,
  },
  {
    id: 15002,
    name: '8 折优惠券',
    type: 'discount' as const,
    value: 0.8,
    valid_until: '2026-12-31T23:59:59Z',
    disabled: false,
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
      <MemoryRouter initialEntries={['/coupons']}>
        <Routes>
          <Route path="/coupons" element={<CouponsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('CouponsPage', () => {
  beforeEach(() => {
    mocks.fetchCoupons.mockReset();
    mocks.createCoupon.mockReset();
    mocks.disableCoupon.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchCoupons.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('coupons-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('新人 50 元券')).toBeInTheDocument();
    expect(screen.getByText('8 折优惠券')).toBeInTheDocument();
    expect(screen.getByText('¥50')).toBeInTheDocument();
    expect(screen.getByText('80 折')).toBeInTheDocument();
  });

  it('super_admin 显示创建按钮', async () => {
    mocks.fetchCoupons.mockResolvedValue({ data: [], total: 0 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-create')).toBeInTheDocument(),
    );
  });

  it('order_admin 也可操作', async () => {
    mocks.fetchCoupons.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('order_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-create')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('btn-disable-15001')).toBeInTheDocument();
  });

  it('viewer 不显示创建 / 停用按钮', async () => {
    mocks.fetchCoupons.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('coupons-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-create')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-disable-15001')).not.toBeInTheDocument();
  });

  it('点击创建 → 弹 Modal → 输入 name', async () => {
    const user = userEvent.setup();
    mocks.fetchCoupons.mockResolvedValue({ data: [], total: 0 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-create')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-create'));
    await waitFor(() =>
      expect(screen.getByTestId('create-name')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('create-name'), '测试卡券');
    expect(screen.getByTestId('create-name')).toHaveValue('测试卡券');
  });

  it('点击停用 → 弹 confirm → 调 disableCoupon', async () => {
    const user = userEvent.setup();
    mocks.fetchCoupons.mockResolvedValue({ data: FIXTURES, total: 2 });
    mocks.disableCoupon.mockResolvedValue({
      ...FIXTURES[0],
      disabled: true,
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-disable-15001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-disable-15001'));
    await waitFor(() =>
      expect(screen.getByTestId('btn-disable-confirm')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-disable-confirm'));
    await waitFor(() => {
      expect(mocks.disableCoupon).toHaveBeenCalledWith(15001);
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已停用优惠券 #15001'),
      );
    });
  });
});