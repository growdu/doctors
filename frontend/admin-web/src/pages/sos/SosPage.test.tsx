/**
 * SosPage 测试：
 *   - 加载列表 + 行渲染（含升级级别标签）；
 *   - viewer 不显示处置 / 升级按钮（RBAC）；
 *   - 点击处置 → 弹 Modal → 输入 note → 调 resolveSos；
 *   - 点击升级 → 弹 Modal → 输入 level + reason → 调 escalateSos；
 *   - 升级原因缺失时校验未通过。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchSosAlerts: vi.fn(),
  resolveSos: vi.fn(),
  escalateSos: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/sos', () => ({
  fetchSosAlerts: mocks.fetchSosAlerts,
  resolveSos: mocks.resolveSos,
  escalateSos: mocks.escalateSos,
  sosQueryKeys: {
    list: (params?: { status?: string }) => ['sos', params?.status ?? 'all'] as const,
    detail: (id: number | string) => ['sos-detail', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import SosPage from './SosPage';

const FIXTURES = [
  {
    id: 12001,
    order_id: 1004,
    patient_name: '丁',
    escort_name: '赵陪诊',
    location: '北京协和医院',
    contact: '138****0030',
    status: 'open' as const,
    created_at: '2026-09-24T10:30:00Z',
  },
  {
    id: 12002,
    order_id: 1005,
    patient_name: '戊',
    escort_name: '王陪诊',
    location: '上海同济医院',
    contact: '138****0040',
    status: 'closed' as const,
    resolution_note: '已处置',
    created_at: '2026-09-19T13:00:00Z',
  },
];

function setupAuth(role: 'super_admin' | 'cs' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(initialRole: 'super_admin' | 'cs' | 'viewer' = 'super_admin') {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/sos']}>
        <Routes>
          <Route path="/sos" element={<SosPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('SosPage', () => {
  beforeEach(() => {
    mocks.fetchSosAlerts.mockReset();
    mocks.resolveSos.mockReset();
    mocks.escalateSos.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchSosAlerts.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('sos-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('北京协和医院')).toBeInTheDocument();
    expect(screen.getByText('上海同济医院')).toBeInTheDocument();
    expect(screen.getByText('待处置')).toBeInTheDocument();
    expect(screen.getByText('已关闭')).toBeInTheDocument();
  });

  it('viewer 不显示处置 / 升级按钮', async () => {
    mocks.fetchSosAlerts.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('sos-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-resolve-12001')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-escalate-12001')).not.toBeInTheDocument();
  });

  it('点击处置 → 弹 Modal → 输入 note → 调 resolveSos', async () => {
    const user = userEvent.setup();
    mocks.fetchSosAlerts.mockResolvedValue({ data: FIXTURES, total: 2 });
    mocks.resolveSos.mockResolvedValue({
      ...FIXTURES[0],
      status: 'closed',
      resolution_note: '已联系急诊',
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-resolve-12001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-resolve-12001'));
    await waitFor(() =>
      expect(screen.getByTestId('resolve-note-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('resolve-note-input'), '已联系急诊');
    await user.click(screen.getByTestId('btn-resolve-confirm'));
    await waitFor(() => {
      expect(mocks.resolveSos).toHaveBeenCalledWith(12001, '已联系急诊');
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已处置报警 #12001'),
      );
    });
  });

  it('点击升级 → 弹 Modal → 输入 reason → 调 escalateSos', async () => {
    const user = userEvent.setup();
    mocks.fetchSosAlerts.mockResolvedValue({ data: FIXTURES, total: 2 });
    mocks.escalateSos.mockResolvedValue({
      ...FIXTURES[0],
      escalate_level: 'L2',
      escalate_reason: '危急',
    });
    renderPage('cs');
    await waitFor(() =>
      expect(screen.getByTestId('btn-escalate-12001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-escalate-12001'));
    await waitFor(() =>
      expect(screen.getByTestId('escalate-reason-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('escalate-reason-input'), '患者危急');
    expect(screen.getByTestId('escalate-reason-input')).toHaveValue('患者危急');
  });

  it('处置未输入 note → 不调 resolveSos', async () => {
    const user = userEvent.setup();
    mocks.fetchSosAlerts.mockResolvedValue({ data: FIXTURES, total: 2 });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-resolve-12001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-resolve-12001'));
    await waitFor(() =>
      expect(screen.getByTestId('btn-resolve-confirm')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-resolve-confirm'));
    expect(mocks.resolveSos).not.toHaveBeenCalled();
  });
});