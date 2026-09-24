/**
 * OrderListPage v2 测试：
 *   - 列渲染：selected_escort_id + escort_pending_expire_at；
 *   - 状态筛选 selecting_escort 仅显示对应订单；
 *   - 点击订单号跳详情页。
 *
 * 策略：vi.mock @/api/admin/orders 返回固定 fixture，避免依赖真实网络。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 1
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import type { OrderListItem } from '@/types/generated';

// ── Mock API ────────────────────────────────────────────────────────
const mocks = vi.hoisted(() => ({
  fetchOrders: vi.fn(),
}));

vi.mock('@/api/admin/orders', () => ({
  fetchOrders: mocks.fetchOrders,
  orderQueryKeys: {
    list: (params?: { status?: string }) => ['orders', params?.status ?? 'all'] as const,
    detail: (id: number | string) => ['order', id] as const,
  },
}));

import OrderListPage from './OrderListPage';

const FIXTURES: OrderListItem[] = [
  {
    id: 9001,
    status: 'selecting_escort',
    selected_escort_id: null,
    escort_pending_expire_at: null,
    hospital_name: '协和医院',
    final_amount: 500,
    created_at: '2026-09-24T10:00:00Z',
  },
  {
    id: 9002,
    status: 'escort_pending_acceptance',
    selected_escort_id: 42,
    escort_pending_expire_at: '2026-09-24T10:30:30Z',
    hospital_name: '同仁医院',
    final_amount: 800,
    created_at: '2026-09-24T10:01:00Z',
  },
];

function renderList(initialEntry = '/orders') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/orders" element={<OrderListPage />} />
          <Route
            path="/orders/:id"
            element={<div data-testid="order-detail-stub">detail-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('OrderListPage v2 selecting_escort 列与筛选', () => {
  it('renders selected_escort_id + escort_pending_expire_at 2 列', async () => {
    mocks.fetchOrders.mockResolvedValue({ data: FIXTURES, total: FIXTURES.length });
    renderList();
    await waitFor(() =>
      expect(screen.getByTestId('order-list-page')).toBeInTheDocument(),
    );
    // 列名
    expect(screen.getByText('已选陪诊师')).toBeInTheDocument();
    expect(screen.getByText('确认截止')).toBeInTheDocument();
    // 行内容：9001 未选 / 9002 #42
    expect(screen.getByTestId('row-status-9001')).toHaveTextContent('待患者选人');
    expect(screen.getByTestId('row-status-9002')).toHaveTextContent('待陪诊师确认');
    expect(screen.getByText('#42')).toBeInTheDocument();
    expect(screen.getAllByText('未选').length).toBeGreaterThanOrEqual(1);
  });

  it('URL ?status=selecting_escort 自动过滤，仅显示 id=9001', async () => {
    mocks.fetchOrders.mockImplementation(
      async (params?: { status?: string }) => {
        const list = params?.status
          ? FIXTURES.filter((o) => o.status === params.status)
          : FIXTURES;
        return { data: list, total: list.length };
      },
    );
    renderList('/orders?status=selecting_escort');
    await waitFor(() =>
      expect(screen.getByTestId('order-list-page')).toBeInTheDocument(),
    );
    // 等查询返回后断言：9001 在，9002 不在
    await waitFor(() => {
      expect(screen.getByTestId('order-link-9001')).toBeInTheDocument();
      expect(screen.queryByTestId('order-link-9002')).not.toBeInTheDocument();
    });
    // 确认 fetchOrders 收到 status=selecting_escort
    expect(mocks.fetchOrders).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'selecting_escort' }),
    );
  });

  it('点击订单号跳详情页 /orders/9002', async () => {
    const user = userEvent.setup();
    mocks.fetchOrders.mockResolvedValue({ data: FIXTURES, total: FIXTURES.length });
    renderList();
    await waitFor(() =>
      expect(screen.getByTestId('order-link-9002')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('order-link-9002'));
    // 路由跳转到 /orders/9002（detail stub 渲染）
    expect(await screen.findByTestId('order-detail-stub')).toBeInTheDocument();
  });

  it('列头部顺序：订单号 / 医院 / 金额 / 状态 / 已选陪诊师 / 确认截止 / 下单时间 / 操作', async () => {
    mocks.fetchOrders.mockResolvedValue({ data: FIXTURES, total: FIXTURES.length });
    renderList();
    await waitFor(() =>
      expect(screen.getByTestId('order-list-page')).toBeInTheDocument(),
    );
    // 用 queryAllByRole('columnheader') 验证 header 顺序
    const headers = screen
      .getAllByRole('columnheader')
      .map((h) => within(h).queryByText(/./)?.textContent ?? '');
    // 关键列名按顺序出现（顺序断言）
    const idxOf = (kw: string) => headers.findIndex((h) => h.includes(kw));
    expect(idxOf('订单号')).toBeGreaterThanOrEqual(0);
    expect(idxOf('已选陪诊师')).toBeGreaterThan(idxOf('状态'));
    expect(idxOf('确认截止')).toBeGreaterThan(idxOf('已选陪诊师'));
    expect(idxOf('下单时间')).toBeGreaterThan(idxOf('确认截止'));
    expect(idxOf('操作')).toBeGreaterThanOrEqual(0);
  });
});