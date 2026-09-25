/**
 * PatientsPage 测试：
 *   - 加载列表 + 行渲染；
 *   - viewer 不显示封禁 / 解封按钮；
 *   - 点击封禁 → Modal 输入 reason → 调 banPatient；
 *   - 点击解封 → 直接调 unbanPatient；
 *   - 关键词搜索触发新查询；
 *   - 跳详情。
 *
 * 策略：vi.mock @/api/admin/patients + vi.mock antd message。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchPatients: vi.fn(),
  banPatient: vi.fn(),
  unbanPatient: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/patients', () => ({
  fetchPatients: mocks.fetchPatients,
  banPatient: mocks.banPatient,
  unbanPatient: mocks.unbanPatient,
  patientQueryKeys: {
    list: (params?: { keyword?: string }) =>
      ['patients', params?.keyword ?? 'all'] as const,
    detail: (id: number | string) => ['patient-detail', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import PatientsPage from './PatientsPage';

const FIXTURES = [
  {
    id: 7001,
    name: '张三',
    phone: '138****0001',
    registered_at: '2026-01-15T08:00:00Z',
    order_count: 5,
    refund_count: 0,
    verify_status: 'verified' as const,
    status: 'active' as const,
    ban_reason: null,
  },
  {
    id: 7002,
    name: '李四',
    phone: '138****0002',
    registered_at: '2026-03-20T09:00:00Z',
    order_count: 3,
    refund_count: 1,
    verify_status: 'pending' as const,
    status: 'banned' as const,
    ban_reason: '恶意下单',
  },
];

function setupAuth(role: 'super_admin' | 'order_admin' | 'cs' | 'viewer' = 'super_admin') {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role, avatar_url: null },
    role,
    isAuthed: true,
  } as never);
}

function renderPage(
  initialRole: 'super_admin' | 'order_admin' | 'cs' | 'viewer' = 'super_admin',
) {
  setupAuth(initialRole);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/patients']}>
        <Routes>
          <Route path="/patients" element={<PatientsPage />} />
          <Route
            path="/patients/:id"
            element={<div data-testid="patient-detail-stub">detail-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('PatientsPage', () => {
  beforeEach(() => {
    mocks.fetchPatients.mockReset();
    mocks.banPatient.mockReset();
    mocks.unbanPatient.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载列表 + 行渲染', async () => {
    mocks.fetchPatients.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('patient-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('张三')).toBeInTheDocument();
    expect(screen.getByText('李四')).toBeInTheDocument();
    expect(screen.getByTestId('row-ban-7002')).toHaveTextContent('封禁');
    expect(screen.getByTestId('row-ban-7001')).toHaveTextContent('正常');
  });

  it('viewer 不显示封禁 / 解封按钮', async () => {
    mocks.fetchPatients.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('patient-table')).toBeInTheDocument(),
    );
    expect(screen.queryByTestId('btn-ban-7001')).not.toBeInTheDocument();
    expect(screen.queryByTestId('btn-unban-7002')).not.toBeInTheDocument();
    // 详情按钮始终可见
    expect(screen.getByTestId('btn-detail-7001')).toBeInTheDocument();
  });

  it('点击封禁 → Modal → 输入 reason → 调 banPatient', async () => {
    const user = userEvent.setup();
    mocks.fetchPatients.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    mocks.banPatient.mockResolvedValue({
      ...FIXTURES[0],
      status: 'banned',
      ban_reason: '恶意下单',
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-ban-7001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-ban-7001'));
    await waitFor(() =>
      expect(screen.getByTestId('ban-reason-input')).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId('ban-reason-input'), '恶意下单');
    await user.click(screen.getByTestId('btn-ban-confirm'));
    await waitFor(() => {
      expect(mocks.banPatient).toHaveBeenCalledWith(7001, '恶意下单');
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已封禁患者 #7001'),
      );
    });
  });

  it('点击解封 → 直接调 unbanPatient', async () => {
    const user = userEvent.setup();
    mocks.fetchPatients.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    mocks.unbanPatient.mockResolvedValue({
      ...FIXTURES[1],
      status: 'active',
    });
    renderPage('super_admin');
    await waitFor(() =>
      expect(screen.getByTestId('btn-unban-7002')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-unban-7002'));
    await waitFor(() => {
      expect(mocks.unbanPatient).toHaveBeenCalledWith(7002);
      expect(mocks.message.success).toHaveBeenCalledWith(
        expect.stringContaining('已解封患者 #7002'),
      );
    });
  });

  it('关键词搜索触发 fetchPatients 带 keyword', async () => {
    const user = userEvent.setup({ delay: null });
    mocks.fetchPatients.mockResolvedValue({ data: [], total: 0 });
    renderPage();
    await waitFor(() => expect(mocks.fetchPatients).toHaveBeenCalled());
    const input = screen.getByTestId('patient-keyword-input');
    await user.type(input, '张');
    // debounce 500ms + rerender → wait a bit longer
    await waitFor(
      () => {
        expect(mocks.fetchPatients).toHaveBeenCalledWith(
          expect.objectContaining({ keyword: '张' }),
        );
      },
      { timeout: 1500 },
    );
  });

  it('点击详情 → 跳 /patients/:id', async () => {
    const user = userEvent.setup();
    mocks.fetchPatients.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage('viewer');
    await waitFor(() =>
      expect(screen.getByTestId('btn-detail-7001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-detail-7001'));
    expect(await screen.findByTestId('patient-detail-stub')).toBeInTheDocument();
  });
});
