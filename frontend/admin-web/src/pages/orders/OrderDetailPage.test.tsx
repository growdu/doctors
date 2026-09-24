/**
 * OrderDetailPage v2 测试：
 *   - selecting_escort 状态展示「待患者选人」步骤 + 未选 tag；
 *   - escort_pending_acceptance 状态展示 30s 倒计时（剩余 < 60s 红色高亮）+ 已选 #42；
 *   - 拒接回退：escort_reject_reason='escort_declined' 时顶部 Alert 卡展示「陪诊师拒接」
 *     + 「患者可重新选」提示。
 *
 * 策略：vi.mock @/api/admin/orders 直接返回固定数据，避免依赖 msw/node。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 2
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import dayjs from 'dayjs';
import type { OrderDetail } from '@/types/generated';

// ── Mock API ────────────────────────────────────────────────────────
const mocks = vi.hoisted(() => ({
  fetchOrderDetail: vi.fn<(id: number) => Promise<OrderDetail>>(),
}));

vi.mock('@/api/admin/orders', () => ({
  fetchOrderDetail: mocks.fetchOrderDetail,
  orderQueryKeys: {
    detail: (id: number | string) => ['order', id] as const,
  },
}));

import OrderDetailPage from './OrderDetailPage';

const FIXTURES: Record<string, OrderDetail> = {
  '9001': {
    id: 9001,
    status: 'selecting_escort',
    selected_escort_id: null,
    escort_pending_expire_at: null,
    escort_reject_reason: null,
    hospital_name: '协和医院',
    final_amount: 500,
    created_at: '2026-09-24T10:00:00Z',
  },
  '9002': {
    id: 9002,
    status: 'escort_pending_acceptance',
    selected_escort_id: 42,
    // 截止时间 = now + 30s（触发 < 60s 红色高亮）
    escort_pending_expire_at: dayjs().add(30, 'second').toISOString(),
    escort_reject_reason: null,
    hospital_name: '同仁医院',
    final_amount: 800,
    created_at: '2026-09-24T10:01:00Z',
  },
  '9003': {
    id: 9003,
    status: 'selecting_escort',
    selected_escort_id: 42,
    escort_pending_expire_at: null,
    escort_reject_reason: 'escort_declined',
    hospital_name: '301医院',
    final_amount: 600,
    created_at: '2026-09-24T10:02:00Z',
  },
};

function renderDetail(orderId: string) {
  mocks.fetchOrderDetail.mockImplementation(async (id: number) => {
    const fd = FIXTURES[String(id)];
    if (!fd) throw new Error(`not found: ${id}`);
    return fd;
  });

  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/orders/${orderId}`]}>
        <Routes>
          <Route path="/orders/:id" element={<OrderDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('OrderDetailPage v2 选人模式分支', () => {
  it('selecting_escort 状态展示「待患者选人」步骤 + 未选 tag', async () => {
    renderDetail('9001');
    await waitFor(() =>
      expect(screen.getByTestId('order-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText(/订单 9001/)).toBeInTheDocument();
    // 步骤条文本
    expect(screen.getByText('待患者选人')).toBeInTheDocument();
    // 已选陪诊师单元格显示「未选」
    const cell = screen.getByTestId('selected-escort-cell');
    expect(cell).toHaveTextContent('未选');
  });

  it('escort_pending_acceptance 展示 30s 倒计时 + 已选 #42 + < 60s 红色高亮', async () => {
    renderDetail('9002');
    await waitFor(() =>
      expect(screen.getByTestId('order-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText(/订单 9002/)).toBeInTheDocument();
    // 已选陪诊师 ID
    expect(screen.getByTestId('selected-escort-tag')).toHaveTextContent('#42');
    // 倒计时组件 + 红色高亮
    const countdown = await screen.findByTestId('escort-countdown');
    expect(countdown).toBeInTheDocument();
    expect(countdown.style.color).toBe('rgb(255, 77, 79)');
    expect(countdown.getAttribute('data-critical')).toBe('true');
    // 剩余秒数 < 60s
    const m = countdown.textContent?.match(/^(\d+)s$/);
    expect(m).not.toBeNull();
    expect(Number(m![1])).toBeLessThan(60);
  });

  it('拒接回退：escort_reject_reason=escort_declined 展示拒接卡 + 可重新选', async () => {
    renderDetail('9003');
    await waitFor(() =>
      expect(screen.getByTestId('order-detail-page')).toBeInTheDocument(),
    );
    const alert = screen.getByTestId('reject-alert');
    expect(alert).toBeInTheDocument();
    expect(alert).toHaveTextContent(/陪诊师拒接/);
    expect(alert).toHaveTextContent(/拒接原因/);
    expect(alert).toHaveTextContent(/陪诊师主动拒接/);
    // 可重新选 链接
    expect(alert).toHaveTextContent(/患者可在 miniapp 端/);
    expect(alert).toHaveTextContent(/重新选择其他陪诊师/);
  });
});