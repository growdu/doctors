/**
 * WalletDetailPage 测试：
 *   - 加载成功 → 渲染主体信息 + 余额 / 冻结卡片 + 最近流水表；
 *   - 加载失败 → 渲染 Alert；
 *   - 点击返回 → 跳 /wallets。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

const mocks = vi.hoisted(() => ({
  fetchWalletDetail: vi.fn(),
}));

vi.mock('@/api/admin/wallets', () => ({
  fetchWalletDetail: mocks.fetchWalletDetail,
  walletQueryKeys: {
    list: (params?: { type?: string; tx_type?: string }) =>
      ['wallets', params?.type ?? 'all', params?.tx_type ?? 'all'] as const,
    detail: (id: number | string) => ['wallet-detail', id] as const,
  },
}));

import WalletDetailPage from './WalletDetailPage';

const SUBJECT_FIXTURE = {
  id: 5001,
  subject_type: 'patient' as const,
  subject_name: '张三',
  balance: 1200,
  frozen: 200,
  recent_transactions: [
    {
      id: 6001,
      subject_id: 5001,
      subject_type: 'patient' as const,
      tx_type: 'recharge' as const,
      amount: 500,
      balance_after: 500,
      created_at: '2026-09-20T10:00:00Z',
    },
  ],
};

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

function renderDetail(id: string) {
  setupAuth();
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/wallets/${id}`]}>
        <Routes>
          <Route path="/wallets" element={<div data-testid="wallets-list-stub" />} />
          <Route path="/wallets/:id" element={<WalletDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('WalletDetailPage', () => {
  beforeEach(() => {
    mocks.fetchWalletDetail.mockReset();
  });

  it('加载成功 → 渲染余额 / 冻结 + 最近流水', async () => {
    mocks.fetchWalletDetail.mockResolvedValue(SUBJECT_FIXTURE);
    renderDetail('5001');
    await waitFor(() =>
      expect(screen.getByTestId('wallet-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('card-balance')).toBeInTheDocument();
    expect(screen.getByTestId('card-frozen')).toBeInTheDocument();
    expect(screen.getByText('张三')).toBeInTheDocument();
    expect(screen.getByTestId('recent-tx-table')).toBeInTheDocument();
  });

  it('加载失败 → 渲染 Alert', async () => {
    mocks.fetchWalletDetail.mockRejectedValue(new Error('not found'));
    renderDetail('9999');
    await waitFor(() =>
      expect(screen.getByTestId('wallet-detail-page')).toBeInTheDocument(),
    );
    expect(screen.getByText('加载失败')).toBeInTheDocument();
  });

  it('点击返回 → 跳 /wallets', async () => {
    const user = userEvent.setup();
    mocks.fetchWalletDetail.mockResolvedValue(SUBJECT_FIXTURE);
    renderDetail('5001');
    await waitFor(() =>
      expect(screen.getByTestId('btn-back')).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId('btn-back'));
    expect(await screen.findByTestId('wallets-list-stub')).toBeInTheDocument();
  });
});