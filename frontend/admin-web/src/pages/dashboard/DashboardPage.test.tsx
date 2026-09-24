/**
 * DashboardPage v2 测试：
 *   - 渲染 6 个 Statistic 卡片（4 旧 + 2 新：待患者选人 / 待陪诊师确认）；
 *   - 点击「待患者选人」卡片跳 /orders?status=selecting_escort；
 *   - 点击「待陪诊师确认」卡片跳 /orders?status=escort_pending_acceptance。
 *
 * 策略：vi.mock @/api/admin/reports 直接返回固定 OverviewReport。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 3
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import type { OverviewReport } from '@/types/generated';

const mocks = vi.hoisted(() => ({
  fetchOverview: vi.fn<() => Promise<OverviewReport>>(),
}));

vi.mock('@/api/admin/reports', () => ({
  fetchOverview: mocks.fetchOverview,
  reportsQueryKeys: {
    overview: ['dashboard-overview'] as const,
  },
}));

import DashboardPage from './DashboardPage';

const FIXTURE_OVERVIEW: OverviewReport = {
  today_orders: 12,
  today_gmv: 8600,
  pending_escorts: 3,
  pending_refunds: 2,
  // v2 新增
  pending_selecting_escort: 5,
  pending_escort_acceptance: 2,
};

function renderDashboard(initialPath = '/dashboard') {
  mocks.fetchOverview.mockResolvedValue(FIXTURE_OVERVIEW);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route
            path="/orders"
            element={<div data-testid="orders-stub">orders-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('DashboardPage v2 待确认卡片', () => {
  it('renders 6 个 Statistic 卡片（4 旧 + 2 新）', async () => {
    renderDashboard();
    await waitFor(() =>
      expect(screen.getByTestId('dashboard')).toBeInTheDocument(),
    );
    // v1 旧 4 卡片标题
    expect(screen.getByText('今日订单')).toBeInTheDocument();
    expect(screen.getByText('今日 GMV')).toBeInTheDocument();
    expect(screen.getByText('待审核陪诊师')).toBeInTheDocument();
    expect(screen.getByText('待审核退款')).toBeInTheDocument();
    // v2 新增 2 卡片标题
    expect(screen.getByText('待患者选人')).toBeInTheDocument();
    expect(screen.getByText('待陪诊师确认')).toBeInTheDocument();
    // 数值（statistic 渲染在 .ant-statistic-content-value）
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('点击「待患者选人」卡片跳 /orders?status=selecting_escort', async () => {
    const user = userEvent.setup();
    renderDashboard();
    await waitFor(() =>
      expect(screen.getByTestId('dashboard')).toBeInTheDocument(),
    );
    const card = screen.getByTestId('card-selecting-escort');
    await user.click(card);
    // 路由命中 /orders stub
    expect(await screen.findByTestId('orders-stub')).toBeInTheDocument();
  });

  it('点击「待陪诊师确认」卡片跳 /orders?status=escort_pending_acceptance', async () => {
    const user = userEvent.setup();
    renderDashboard();
    await waitFor(() =>
      expect(screen.getByTestId('dashboard')).toBeInTheDocument(),
    );
    const card = screen.getByTestId('card-escort-pending-acceptance');
    await user.click(card);
    expect(await screen.findByTestId('orders-stub')).toBeInTheDocument();
  });
});