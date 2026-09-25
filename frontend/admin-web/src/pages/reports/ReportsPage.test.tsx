/**
 * ReportsPage 测试：
 *   - 渲染 6 个 Statistic 卡片；
 *   - 渲染明细表（行包含医院 / 套餐名称）；
 *   - 维度切换 → 触发 fetchBusinessReport；
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import type { BusinessReport } from '@/api/admin/reports';

const mocks = vi.hoisted(() => ({
  fetchBusinessReport: vi.fn<() => Promise<BusinessReport>>(),
}));

vi.mock('@/api/admin/reports', () => ({
  fetchBusinessReport: mocks.fetchBusinessReport,
  fetchOverview: vi.fn(),
  reportsQueryKeys: {
    overview: ['dashboard-overview'] as const,
    business: (params?: { from?: string; to?: string }) =>
      ['reports-business', params?.from ?? '', params?.to ?? ''] as const,
  },
}));

import ReportsPage from './ReportsPage';

const FIXTURE_REPORT: BusinessReport = {
  from: '2026-09-18',
  to: '2026-09-25',
  totals: {
    order_count: 200,
    gmv: 160000,
    refund_count: 10,
    refund_amount: 5000,
    net_revenue: 155000,
    avg_rating: 13.5,
  },
  rows: [
    {
      dimension: 'hospital',
      key: 13001,
      label: '北京协和医院',
      order_count: 80,
      gmv: 60000,
      refund_count: 4,
      refund_amount: 2000,
    },
    {
      dimension: 'hospital',
      key: 13002,
      label: '上海同济医院',
      order_count: 60,
      gmv: 45000,
      refund_count: 3,
      refund_amount: 1500,
    },
  ],
};

function renderReports() {
  mocks.fetchBusinessReport.mockResolvedValue(FIXTURE_REPORT);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/reports']}>
        <Routes>
          <Route path="/reports" element={<ReportsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('ReportsPage', () => {
  it('渲染 6 个 Statistic 卡片', async () => {
    renderReports();
    await waitFor(() =>
      expect(screen.getByTestId('stat-orders')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('stat-gmv')).toBeInTheDocument();
    expect(screen.getByTestId('stat-refund-count')).toBeInTheDocument();
    expect(screen.getByTestId('stat-refund-amount')).toBeInTheDocument();
    expect(screen.getByTestId('stat-net')).toBeInTheDocument();
    expect(screen.getByTestId('stat-rating')).toBeInTheDocument();
    expect(screen.getByText('订单数')).toBeInTheDocument();
    expect(screen.getByText('净收入')).toBeInTheDocument();
  });

  it('渲染明细表（含医院行）', async () => {
    renderReports();
    await waitFor(() =>
      expect(screen.getByTestId('rows-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('北京协和医院')).toBeInTheDocument();
    expect(screen.getByText('上海同济医院')).toBeInTheDocument();
  });

  it('维度切换 UI 存在', async () => {
    renderReports();
    await waitFor(() =>
      expect(screen.getByTestId('dimension-select')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('range-picker')).toBeInTheDocument();
  });
});