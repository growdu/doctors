/**
 * PatientDetailPage 测试：
 *   - 加载成功 → 渲染实名 + 钱包 + 订单历史；
 *   - 加载失败 → Alert；
 *   - 返回按钮跳列表。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchPatientDetail: vi.fn(),
  fetchOrders: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@/api/admin/patients', () => ({
  fetchPatientDetail: mocks.fetchPatientDetail,
  banPatient: vi.fn(),
  unbanPatient: vi.fn(),
  patientQueryKeys: {
    list: (params?: { keyword?: string }) =>
      ['patients', params?.keyword ?? 'all'] as const,
    detail: (id: number | string) => ['patient-detail', id] as const,
  },
}));

vi.mock('@/api/admin/orders', () => ({
  fetchOrders: mocks.fetchOrders,
  fetchOrderDetail: vi.fn(),
  orderQueryKeys: {
    list: (params?: { status?: string }) => ['orders', params?.status ?? 'all'] as const,
    detail: (id: number | string) => ['order', id] as const,
  },
}));

vi.mock('antd', async (orig) => {
  const actual = await orig<typeof import('antd')>();
  return {
    ...actual,
    message: mocks.message,
  };
});

import PatientDetailPage from './PatientDetailPage';

const PATIENT_FIXTURE = {
  id: 7001,
  name: '张三',
  phone: '138****0001',
  registered_at: '2026-01-15T08:00:00Z',
  order_count: 5,
  refund_count: 0,
  verify_status: 'verified' as const,
  status: 'active' as const,
  ban_reason: null,
  wallet_balance: 800,
  wallet_frozen: 0,
};

const ORDERS_FIXTURE = [
  {
    id: 1001,
    status: 'completed' as const,
    hospital_name: '协和医院',
    final_amount: 300,
    created_at: '2026-09-15T09:00:00Z',
    selected_escort_id: null,
    escort_pending_expire_at: null,
  },
];

function setupAuth() {
  useAuthStore.setState({
    token: 'mock.token',
    user: { id: 1, username: 'admin', display_name: '管理员', role: 'super_admin', avatar_url: null },
    role: 'super_admin',
    isAuthed: true,
  } as never);
}

function renderDetail(id: string) {
  setupAuth();
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/patients/${id}`]}>
        <Routes>
          <Route path="/patients" element={<div data-testid="patient-list-stub" />} />
          <Route path="/patients/:id" element={<PatientDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('PatientDetailPage', () => {
  beforeEach(() => {
    mocks.fetchPatientDetail.mockReset();
    mocks.fetchOrders.mockReset();
    mocks.message.success.mockReset();
    mocks.message.error.mockReset();
  });

  it('加载成功 → 实名 + 钱包 + 订单历史', async () => {
    mocks.fetchPatientDetail.mockResolvedValue(PATIENT_FIXTURE);
    mocks.fetchOrders.mockResolvedValue({ data: ORDERS_FIXTURE, total: 1 });
    renderDetail('7001');
    await waitFor(() =>
      expect(screen.getByTestId('patient-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('张三')).toBeInTheDocument();
    expect(screen.getByText('138****0001')).toBeInTheDocument();
    expect(screen.getByTestId('patient-verify')).toHaveTextContent('已实名');
    expect(screen.getByText('800.00')).toBeInTheDocument();
    // 等 orders 子查询解析
    await waitFor(() =>
      expect(screen.getByTestId('orders-table')).toBeInTheDocument(),
    );
    expect(screen.getByText('协和医院')).toBeInTheDocument();
  });

  it('加载失败 → Alert', async () => {
    mocks.fetchPatientDetail.mockRejectedValue(new Error('not found'));
    mocks.fetchOrders.mockResolvedValue({ data: [], total: 0 });
    renderDetail('9999');
    await waitFor(() =>
      expect(screen.getByTestId('patient-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('加载失败')).toBeInTheDocument();
  });

  it('点击返回跳列表', async () => {
    const user = userEvent.setup();
    mocks.fetchPatientDetail.mockResolvedValue(PATIENT_FIXTURE);
    mocks.fetchOrders.mockResolvedValue({ data: [], total: 0 });
    renderDetail('7001');
    await waitFor(() =>
      expect(screen.getByTestId('btn-back')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-back'));
    expect(await screen.findByTestId('patient-list-stub')).toBeInTheDocument();
  });

  it('无订单记录 → Empty', async () => {
    mocks.fetchPatientDetail.mockResolvedValue(PATIENT_FIXTURE);
    mocks.fetchOrders.mockResolvedValue({ data: [], total: 0 });
    renderDetail('7001');
    await waitFor(() =>
      expect(screen.getByTestId('orders-empty')).toBeInTheDocument(),
    );
  });
});
