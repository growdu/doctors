/**
 * DashboardDetailPage 测试：
 *   - 合法 id → 渲染明细表；
 *   - 非法 id → 渲染 Alert；
 *   - 点击返回 → 跳 /dashboard。
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

import DashboardDetailPage from './DashboardDetailPage';

function renderPage(id: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/dashboard-detail/${id}`]}>
        <Routes>
          <Route
            path="/dashboard"
            element={<div data-testid="dashboard-stub">dashboard-stub</div>}
          />
          <Route
            path="/orders/:id"
            element={<div data-testid="order-detail-stub">order-stub</div>}
          />
          <Route path="/dashboard-detail/:id" element={<DashboardDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('DashboardDetailPage', () => {
  it('合法 id (orders_pending) → 渲染明细表', async () => {
    renderPage('orders_pending');
    await waitFor(() =>
      expect(screen.getByTestId('dashboard-detail-page')).toBeInTheDocument(),
    );
    // seed.ts V1 + V2 含 selecting_escort / escort_pending_acceptance / created / paid
    expect(screen.getByTestId('dashboard-detail-table')).toBeInTheDocument();
  });

  it('合法 id (refunds_pending) → 渲染明细', async () => {
    renderPage('refunds_pending');
    await waitFor(() =>
      expect(screen.getByTestId('dashboard-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('dashboard-detail-table')).toBeInTheDocument();
  });

  it('非法 id → 渲染 Alert', async () => {
    renderPage('unknown_type');
    await waitFor(() =>
      expect(screen.getByTestId('dashboard-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('未知下钻类型')).toBeInTheDocument();
  });

  it('点击返回 → 跳 /dashboard', async () => {
    const user = userEvent.setup();
    renderPage('orders_pending');
    await waitFor(() =>
      expect(screen.getByTestId('btn-back')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-back'));
    expect(await screen.findByTestId('dashboard-stub')).toBeInTheDocument();
  });
});