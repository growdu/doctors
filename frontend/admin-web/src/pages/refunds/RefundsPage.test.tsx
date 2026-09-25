/**
 * RefundsPage 测试：
 *   - 加载列表 + 行渲染；
 *   - viewer 不显示审批按钮（RBAC）；
 *   - 点击通过 → 弹 Modal → 输入 → 调 approveRefund；
 *   - 点击驳回 → 弹 Modal → 输入原因 → 调 rejectRefund；
 *   - 点击详情 → 跳 /refunds/:id；
 *   - 驳回原因缺失时校验未通过。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchRefunds: vi.fn(),
  approveRefund: vi.fn(),
  rejectRefund: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/refunds', () => ({
  fetchRefunds: mocks.fetchRefunds,
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

import RefundsPage from './RefundsPage';

const FIXTURES = [
  {
    id: 2001,
    order_id: 1001,
    patient_name: '甲',
    amount: 300,
    reason: '患者取消',
    status: 'pending' as const,
    refund_note: null,
    created_at: '2026-09-24T09:00:00Z',
  },
  {
    id: 2003,
    order_id: 1003,
    patient_name: '丙',
    amount: 800,
    reason: '服务不达标',
    status: 'approved' as const,
    refund_note: '已全额退款',
    created_at: '2026-09-22T11:00:00Z',
  },
];

function setupAuth(role: 'super_admin' | 'refund_admin' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(
  initialRole: 'super_admin' | 'refund_admin' | 'viewer' = 'super_admin',
) {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/refunds']}>
        <Routes>
          <Route path="/refunds" element={<RefundsPage />} />
          <Route
            path="/refunds/:id"
            element={<div data-testid="refund-detail-stub">detail-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('RefundsPage', () => {
  beforeEach(() => {
    mocks.fetchRefunds.mockReset();
    mocks.approveRefund.mockReset();
    mocks.rejectRefund.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchRefunds.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('refunds-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('患者取消')).toBeInTheDocument();
    expect(screen.getByText('服务不达标')).toBeInTheDocument();
    expect(screen.getByText('已全额退款')).toBeInTheDocument();
  });

  it('viewer 不显示审批按钮（仅详情）', async () => {
    mocks.fetchRefunds.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('refunds-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-approve-2001')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-reject-2001')).not.toBeInTheDocument();
    expect(screen.getByTestId('btn-detail-2001')).toBeInTheDocument();
  });

  it('点击通过 → 弹 Modal → 输入 → 调 approveRefund', async () => {
    const user = userEvent.setup();
    mocks.fetchRefunds.mockResolvedValue({ data: FIXTURES, total: 2 });
    mocks.approveRefund.mockResolvedValue({
      ...FIXTURES[0],
      status: 'approved',
      refund_note: '已通过',
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-approve-2001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-approve-2001'));
    await waitFor(() =>
      expect(screen.getByTestId('approve-note-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('approve-note-input'), '已通过审批');
    await user.click(screen.getByTestId('btn-approve-confirm'));
    await waitFor(() => {
      expect(mocks.approveRefund).toHaveBeenCalledWith(2001, '已通过审批');
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已通过退款 #2001'),
      );
    });
  });

  it('点击驳回 → 弹 Modal → 输入原因 → 调 rejectRefund', async () => {
    const user = userEvent.setup();
    mocks.fetchRefunds.mockResolvedValue({ data: FIXTURES, total: 2 });
    mocks.rejectRefund.mockResolvedValue({
      ...FIXTURES[0],
      status: 'rejected',
      refund_note: '重复申请',
    });
    renderPage('refund_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject-2001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject-2001'));
    await waitFor(() =>
      expect(screen.getByTestId('reject-reason-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('reject-reason-input'), '重复申请');
    await user.click(screen.getByTestId('btn-reject-confirm'));
    await waitFor(() => {
      expect(mocks.rejectRefund).toHaveBeenCalledWith(2001, '重复申请');
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已驳回退款 #2001'),
      );
    });
  });

  it('驳回时未输入原因 → 不调 rejectRefund', async () => {
    const user = userEvent.setup();
    mocks.fetchRefunds.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject-2001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject-2001'));
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject-confirm')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject-confirm'));
    expect(mocks.rejectRefund).not.toHaveBeenCalled();
  });

  it('点击详情 → 跳 /refunds/:id', async () => {
    const user = userEvent.setup();
    mocks.fetchRefunds.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('btn-detail-2001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-detail-2001'));
    expect(await screen.findByTestId('refund-detail-stub')).toBeInTheDocument();
  });
});