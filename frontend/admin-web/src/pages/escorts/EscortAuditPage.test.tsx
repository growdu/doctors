/**
 * EscortAuditPage 测试：
 *   - 列表渲染：列 + 状态 Badge；
 *   - viewer 角色不显示审核按钮（RBAC）；
 *   - 通过 → 调 approveEscort 并 message.success；
 *   - 拒绝 → 弹 Modal 输入原因后调 rejectEscort；
 *   - 拒绝原因缺失时校验未通过。
 *
 * 策略：vi.mock @/api/admin/escorts 注入固定响应；vi.mock antd message。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchPendingAudit: vi.fn(),
  approveEscort: vi.fn(),
  rejectEscort: vi.fn(),
  message: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('@/api/admin/escorts', () => ({
  fetchPendingAudit: mocks.fetchPendingAudit,
  approveEscort: mocks.approveEscort,
  rejectEscort: mocks.rejectEscort,
  escortQueryKeys: {
    pendingAudit: ['escorts-pending-audit'] as const,
    detail: (id: number | string) => ['escort-detail', id] as const,
    auditHistory: (id: number | string) => ['escort-audit-history', id] as const,
  },
}));

// antd 静态 message 直接替换为 mock（mock 必须早于 import EscortAuditPage 生效）。
vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import EscortAuditPage from './EscortAuditPage';

const FIXTURES = [
  {
    id: 1001,
    name: '王陪诊',
    phone: '138****1001',
    city: '北京',
    rating: 0,
    audit_status: 'pending' as const,
    audit_note: null,
    created_at: '2026-09-23T08:00:00Z',
  },
  {
    id: 1002,
    name: '李陪诊',
    phone: '138****1002',
    city: '上海',
    rating: 0,
    audit_status: 'pending' as const,
    audit_note: null,
    created_at: '2026-09-23T09:00:00Z',
  },
];

function setupAuth(role: 'super_admin' | 'audit_admin' | 'viewer' = 'audit_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'audit', display_name: '审核员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(initialRole: 'super_admin' | 'audit_admin' | 'viewer' = 'audit_admin') {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/escorts/audit']}>
        <Routes>
          <Route path="/escorts/audit" element={<EscortAuditPage />} />
          <Route
            path="/escorts/:id"
            element={<div data-testid="escort-detail-stub">detail-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('EscortAuditPage', () => {
  beforeEach(() => {
    mocks.fetchPendingAudit.mockReset();
    mocks.approveEscort.mockReset();
    mocks.rejectEscort.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('audit_admin 渲染列表 + 审批按钮', async () => {
    mocks.fetchPendingAudit.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('audit_admin');
    await waitFor(() =>
      expect(screen.getByTestId('audit-table')).toBeInTheDocument(),
    );
    // 行按钮
    expect(screen.getByTestId('btn-approve-1001')).toBeInTheDocument();
    expect(screen.getByTestId('btn-reject-1001')).toBeInTheDocument();
    // 状态徽章（audit_status 不在 OrderStatus 枚举，StatusBadge 兜底渲染原 status 文本）
    expect(screen.getByTestId('row-status-1001')).toHaveTextContent('pending');
  });

  it('viewer 角色不显示审批按钮', async () => {
    mocks.fetchPendingAudit.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('audit-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-approve-1001')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-reject-1001')).not.toBeInTheDocument();
    // 但详情按钮仍可见
    expect(screen.getByTestId('btn-detail-1001')).toBeInTheDocument();
  });

  it('点击通过 → 调 approveEscort + message.success', async () => {
    const user = userEvent.setup();
    mocks.fetchPendingAudit.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    mocks.approveEscort.mockResolvedValue({ ...FIXTURES[0], audit_status: 'approved' });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-approve-1001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-approve-1001'));
    await waitFor(() => {
      expect(mocks.approveEscort).toHaveBeenCalledWith(1001, expect.any(Object));
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('通过陪诊师 #1001'),
      );
    });
  });

  it('点击拒绝 → 弹 Modal → 输入原因 → 调 rejectEscort', async () => {
    const user = userEvent.setup();
    mocks.fetchPendingAudit.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    mocks.rejectEscort.mockResolvedValue({ ...FIXTURES[0], audit_status: 'rejected' });
    renderPage('audit_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject-1001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject-1001'));
    // Modal 出现 + 输入原因
    await waitFor(() =>
      expect(screen.getByTestId('reject-reason-input')).toBeInTheDocument(),
    );
    await user.type(
      screen.getByTestId('reject-reason-input'),
      '资质过期',
    );
    await user.click(screen.getByTestId('btn-reject-confirm'));
    await waitFor(() => {
      expect(mocks.rejectEscort).toHaveBeenCalledWith(
        1001,
        expect.objectContaining({ reason: '资质过期' }),
      );
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已拒绝陪诊师 #1001'),
      );
    });
  });

  it('拒绝时未输入原因 → message 不调 reject', async () => {
    const user = userEvent.setup();
    mocks.fetchPendingAudit.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('audit_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject-1001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject-1001'));
    // Modal 出现后直接点确认
    await waitFor(() =>
      expect(screen.getByTestId('btn-reject-confirm')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-reject-confirm'));
    expect(mocks.rejectEscort).not.toHaveBeenCalled();
  });

  it('点击查看详情 → 路由跳 /escorts/:id', async () => {
    const user = userEvent.setup();
    mocks.fetchPendingAudit.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('btn-detail-1001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-detail-1001'));
    expect(await screen.findByTestId('escort-detail-stub')).toBeInTheDocument();
  });
});
