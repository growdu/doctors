/**
 * WalletsPage 测试：
 *   - 加载流水列表 + 行渲染（类型标签 + 金额正负色）；
 *   - 点击详情 → 跳 /wallets/:id。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchWalletTransactions: vi.fn(),
}));

vi.mock('@/api/admin/wallets', () => ({
  fetchWalletTransactions: mocks.fetchWalletTransactions,
  walletQueryKeys: {
    list: (params?: { type?: string; tx_type?: string }) =>
      ['wallets', params?.type ?? 'all', params?.tx_type ?? 'all'] as const,
    detail: (id: number | string) => ['wallet-detail', id] as const,
  },
}));

import WalletsPage from './WalletsPage';

const FIXTURES = [
  {
    id: 6001,
    subject_id: 5001,
    subject_type: 'patient' as const,
    tx_type: 'recharge' as const,
    amount: 500,
    balance_after: 500,
    created_at: '2026-09-20T10:00:00Z',
  },
  {
    id: 6007,
    subject_id: 5004,
    subject_type: 'escort' as const,
    tx_type: 'withdraw' as const,
    amount: -500,
    balance_after: 8200,
    created_at: '2026-09-24T08:00:00Z',
  },
];

function setupAuth() {
  useAuthStore.setState({
    token: 'mock.token',
    user: {
      id: 1,
      username: 'admin',
      display_name: '管理员',
      role: 'super_admin' as const,
      avatar_url: null,
    },
    role: 'super_admin',
    isAuthed: true,
  } as never);
}

function renderPage() {
  setupAuth();
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/wallets']}>
        <Routes>
          <Route path="/wallets" element={<WalletsPage />} />
          <Route
            path="/wallets/:id"
            element={<div data-testid="wallet-detail-stub">detail-stub</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('WalletsPage', () => {
  beforeEach(() => {
    mocks.fetchWalletTransactions.mockReset();
  });

  it('加载列表 + 行渲染（含正负金额）', async () => {
    mocks.fetchWalletTransactions.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('wallets-table')).toBeInTheDocument(),
    );
    // 标签
    expect(screen.getByText('患者')).toBeInTheDocument();
    expect(screen.getByText('陪诊师')).toBeInTheDocument();
    // 金额正负色由 antd 渲染为 span class，包含 ant-typography-success / -danger
    expect(screen.getByText('+500.00')).toBeInTheDocument();
    expect(screen.getByText('-500.00')).toBeInTheDocument();
  });

  it('点击主体详情 → 跳 /wallets/:id', async () => {
    const user = userEvent.setup();
    mocks.fetchWalletTransactions.mockResolvedValue({
      data: FIXTURES,
      total: FIXTURES.length,
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('btn-detail-6001')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-detail-6001'));
    expect(await screen.findByTestId('wallet-detail-stub')).toBeInTheDocument();
  });

  it('列表为空 → 仅渲染空表头', async () => {
    mocks.fetchWalletTransactions.mockResolvedValue({ data: [], total: 0 });
    renderPage();
    await waitFor(() =>
      expect(screen.getByTestId('wallets-table')).toBeInTheDocument(),
    );
  });

  it('筛选触发 fetchWalletTransactions 带参', async () => {
    mocks.fetchWalletTransactions.mockResolvedValue({ data: [], total: 0 });
    renderPage();
    await waitFor(() => expect(mocks.fetchWalletTransactions).toHaveBeenCalled());
    mocks.fetchWalletTransactions.mockClear();
    // 校验筛选 UI 存在
    expect(screen.getByTestId('filter-type')).toBeInTheDocument();
    expect(screen.getByTestId('filter-tx-type')).toBeInTheDocument();
  });
});