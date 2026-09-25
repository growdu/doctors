/**
 * EscortDetailPage 测试：
 *   - 加载成功 → 渲染基础信息 + 健康证 + 审计时间线；
 *   - 加载失败 → 渲染 Alert；
 *   - super_admin 显示审批按钮 + 通过调用 approveEscort；
 *   - viewer 不显示审批按钮；
 *   - 拒绝弹 Modal 输入原因调 rejectEscort。
 *
 * 策略：vi.mock @/api/admin/escorts 注入固定响应。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchEscortDetail: vi.fn(),
  fetchEscortAuditHistory: vi.fn(),
  approveEscort: vi.fn(),
  rejectEscort: vi.fn(),
  message: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('@/api/admin/escorts', () => ({
  fetchEscortDetail: mocks.fetchEscortDetail,
  fetchEscortAuditHistory: mocks.fetchEscortAuditHistory,
  approveEscort: mocks.approveEscort,
  rejectEscort: mocks.rejectEscort,
  escortQueryKeys: {
    pendingAudit: ['escorts-pending-audit'] as const,
    detail: (id: number | string) => ['escort-detail', id] as const,
    auditHistory: (id: number | string) => ['escort-audit-history', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import EscortDetailPage from './EscortDetailPage';

const ESCORT_FIXTURE = {
  id: 1001,
  name: '王陪诊',
  phone: '138****1001',
  city: '北京',
  rating: 0,
  audit_status: 'pending' as const,
  audit_note: null,
  created_at: '2026-09-23T08:00:00Z',
  real_name: '王小明',
  id_card_no: '110101**********',
  age: 32,
  gender: 'male' as const,
  health_cert_status: 'valid' as const,
  service_cities: ['北京', '上海'],
  training_records: [
    { title: '急救基础培训', passed_at: '2026-08-01T10:00:00Z' },
    { title: '陪诊礼仪考核', passed_at: null },
  ],
};

const HISTORY_FIXTURE = [
  {
    timestamp: '2026-09-23T08:00:00Z',
    actor: 'system',
    action: '陪诊师注册',
    color: 'blue' as const,
  },
];

function setupAuth(role: 'super_admin' | 'audit_admin' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderDetail(
  id: string,
  initialRole: 'super_admin' | 'audit_admin' | 'viewer' = 'super_admin',
) {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/escorts/${id}`]}>
        <Routes>
          <Route path="/escorts/audit" element={<div data-testid="audit-list-stub" />} />
          <Route path="/escorts/:id" element={<EscortDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('EscortDetailPage', () => {
  beforeEach(() => {
    mocks.fetchEscortDetail.mockReset();
    mocks.fetchEscortAuditHistory.mockReset();
    mocks.approveEscort.mockReset();
    mocks.rejectEscort.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载成功 → 渲染基础信息 + AuditAction', async () => {
    mocks.fetchEscortDetail.mockResolvedValue(ESCORT_FIXTURE);
    mocks.fetchEscortAuditHistory.mockResolvedValue(HISTORY_FIXTURE);
    renderDetail('1001');
    await waitFor(() =>
      expect(screen.getByTestId('escort-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('王小明')).toBeInTheDocument();
    expect(screen.getByText('110101**********')).toBeInTheDocument();
    expect(screen.getByTestId('escort-status')).toBeInTheDocument();
    expect(screen.getByTestId('escort-audit-history')).toBeInTheDocument();
    expect(screen.getByText('陪诊师注册')).toBeInTheDocument();
  });

  it('加载失败 → 渲染 Alert', async () => {
    mocks.fetchEscortDetail.mockRejectedValue(new Error('not found'));
    mocks.fetchEscortAuditHistory.mockResolvedValue([]);
    renderDetail('9999');
    await waitFor(() =>
      expect(screen.getByTestId('escort-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('加载失败')).toBeInTheDocument();
  });

  it('super_admin 通过按钮调 approveEscort + message.success', async () => {
    const user = userEvent.setup();
    mocks.fetchEscortDetail.mockResolvedValue(ESCORT_FIXTURE);
    mocks.fetchEscortAuditHistory.mockResolvedValue(HISTORY_FIXTURE);
    mocks.approveEscort.mockResolvedValue({
      ...ESCORT_FIXTURE,
      audit_status: 'approved',
    });
    renderDetail('1001', 'super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-approve')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-approve'));
    await waitFor(() => {
      expect(mocks.approveEscort).toHaveBeenCalledWith(
        1001,
        expect.any(Object),
      );
      expect(mocks.message.success).toHaveBeenCalled();
    });
  });

  it('viewer 不显示审批按钮', async () => {
    mocks.fetchEscortDetail.mockResolvedValue(ESCORT_FIXTURE);
    mocks.fetchEscortAuditHistory.mockResolvedValue(HISTORY_FIXTURE);
    renderDetail('1001', 'viewer');
    await waitFor(() =>
      expect(screen.getByTestId('escort-detail-page')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-approve')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-reject')).not.toBeInTheDocument();
    expect(screen.getByTestId('btn-back')).toBeInTheDocument();
  });

  it('拒绝弹 Modal → 输入原因 → 调 rejectEscort', async () => {
    const user = userEvent.setup();
    mocks.fetchEscortDetail.mockResolvedValue(ESCORT_FIXTURE);
    mocks.fetchEscortAuditHistory.mockResolvedValue(HISTORY_FIXTURE);
    mocks.rejectEscort.mockResolvedValue({
      ...ESCORT_FIXTURE,
      audit_status: 'rejected',
    });
    renderDetail('1001', 'audit_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject'));
    await waitFor(() =>
      expect(screen.getByTestId('reject-reason-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('reject-reason-input'), '资质过期');
    await user.click(screen.getByTestId('btn-reject-confirm'));
    await waitFor(() => {
      expect(mocks.rejectEscort).toHaveBeenCalledWith(
        1001,
        expect.objectContaining({ reason: '资质过期' }),
      );
    });
  });

  it('点击返回跳审核列表', async () => {
    const user = userEvent.setup();
    mocks.fetchEscortDetail.mockResolvedValue(ESCORT_FIXTURE);
    mocks.fetchEscortAuditHistory.mockResolvedValue(HISTORY_FIXTURE);
    renderDetail('1001', 'viewer');
    await waitFor(() =>
      expect(screen.getByTestId('btn-back')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-back'));
    expect(await screen.findByTestId('audit-list-stub')).toBeInTheDocument();
  });
});
