/**
 * RefundDetailPage 测试：
 *   - 加载成功 → 渲染 Descriptions；
 *   - 加载失败 → 渲染 Alert；
 *   - super_admin 通过按钮调 approveRefund；
 *   - viewer 不显示审批按钮；
 *   - 驳回弹 Modal 输入原因调 rejectRefund。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchRefundDetail: vi.fn(),
  approveRefund: vi.fn(),
  rejectRefund: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/refunds', () => ({
  fetchRefundDetail: mocks.fetchRefundDetail,
  approveRefund: mocks.approveRefund,
  rejectRefund: mocks.rejectRefund,
  refundQueryKeys: {
    list: (params?: { status?: string }) => ['refunds', params?.status ?? 'all'] as const,
    detail: (id: number | string) => ['refund-detail', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import RefundDetailPage from './RefundDetailPage';

const FIXTURE = {
  id: 2001,
  order_id: 1001,
  patient_name: '甲',
  amount: 300,
  reason: '患者取消',
  status: 'pending' as const,
  refund_note: null,
  created_at: '2026-09-24T09:00:00Z',
};

function setupAuth(role: 'super_admin' | 'refund_admin' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderDetail(
  id: string,
  initialRole: 'super_admin' | 'refund_admin' | 'viewer' = 'super_admin',
) {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/refunds/${id}`]}>
        <Routes>
          <Route path="/refunds" element={<div data-testid="refunds-list-stub" />} />
          <Route path="/refunds/:id" element={<RefundDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('RefundDetailPage', () => {
  beforeEach(() => {
    mocks.fetchRefundDetail.mockReset();
    mocks.approveRefund.mockReset();
    mocks.rejectRefund.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载成功 → 渲染 Descriptions', async () => {
    mocks.fetchRefundDetail.mockResolvedValue(FIXTURE);
    renderDetail('2001');
    await waitFor(() =>
      expect(screen.getByTestId('refund-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('患者取消')).toBeInTheDocument();
    expect(screen.getByText('甲')).toBeInTheDocument();
    expect(screen.getByText('¥300')).toBeInTheDocument();
  });

  it('加载失败 → 渲染 Alert', async () => {
    mocks.fetchRefundDetail.mockRejectedValue(new Error('not found'));
    renderDetail('9999');
    await waitFor(() =>
      expect(screen.getByTestId('refund-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('加载失败')).toBeInTheDocument();
  });

  it('super_admin 通过按钮调 approveRefund + message.success', async () => {
    const user = userEvent.setup();
    mocks.fetchRefundDetail.mockResolvedValue(FIXTURE);
    mocks.approveRefund.mockResolvedValue({ ...FIXTURE, status: 'approved' });
    renderDetail('2001', 'super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-approve')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-approve'));
    await waitFor(() =>
      expect(screen.getByTestId('approve-note-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('approve-note-input'), '已通过');
    await user.click(screen.getByTestId('btn-approve-confirm'));
    await waitFor(() => {
      expect(mocks.approveRefund).toHaveBeenCalledWith(2001, '已通过');
      expect(mocks.message.success).toHaveBeenCalled();
    });
  });

  it('viewer 不显示审批按钮', async () => {
    mocks.fetchRefundDetail.mockResolvedValue(FIXTURE);
    renderDetail('2001', 'viewer');
    await waitFor(() =>
      expect(screen.getByTestId('refund-detail-page')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-approve')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-reject')).not.toBeInTheDocument();
    expect(screen.getByTestId('btn-back')).toBeInTheDocument();
  });

  it('驳回弹 Modal → 输入原因 → 调 rejectRefund', async () => {
    const user = userEvent.setup();
    mocks.fetchRefundDetail.mockResolvedValue(FIXTURE);
    mocks.rejectRefund.mockResolvedValue({ ...FIXTURE, status: 'rejected' });
    renderDetail('2001', 'refund_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject'));
    await waitFor(() =>
      expect(screen.getByTestId('reject-reason-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('reject-reason-input'), '重复申请');
    await user.click(screen.getByTestId('btn-reject-confirm'));
    await waitFor(() => {
      expect(mocks.rejectRefund).toHaveBeenCalledWith(2001, '重复申请');
    });
  });

  it('点击返回跳列表', async () => {
    const user = userEvent.setup();
    mocks.fetchRefundDetail.mockResolvedValue(FIXTURE);
    renderDetail('2001', 'viewer');
    await waitFor(() =>
      expect(screen.getByTestId('btn-back')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-back'));
    expect(await screen.findByTestId('refunds-list-stub')).toBeInTheDocument();
  });
});