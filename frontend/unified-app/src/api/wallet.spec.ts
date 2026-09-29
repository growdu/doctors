/**
 * api/wallet 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  getUserWallet,
  getEscortWallet,
  listTransactions,
  requestWithdrawal,
  approveWithdrawal,
  markWithdrawalPaid,
  rejectWithdrawal,
} from './wallet';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({});
});

describe('api/wallet · patient / escort wallets', () => {
  it('getUserWallet GETs /users/me/wallet', async () => {
    await getUserWallet();
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/users/me/wallet',
      baseURL: 'http://127.0.0.1:8090',
    });
  });

  it('getEscortWallet GETs /escorts/me/wallet', async () => {
    await getEscortWallet();
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/escorts/me/wallet');
  });
});

describe('api/wallet · transactions', () => {
  it('listTransactions with empty query', async () => {
    await listTransactions();
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/wallet/transactions');
  });

  it('listTransactions builds query string', async () => {
    await listTransactions({ type: 'income', page: 1, page_size: 20 });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('type=income');
    expect(url).toContain('page=1');
    expect(url).toContain('page_size=20');
  });
});

describe('api/wallet · withdrawal request (escort-only)', () => {
  it('requestWithdrawal POSTs body', async () => {
    await requestWithdrawal({ amount: 5000, account: '6222021234567', account_name: '张三' });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/escorts/me/wallet/withdraw',
      method: 'POST',
      data: { amount: 5000, account: '6222021234567', account_name: '张三' },
      baseURL: 'http://127.0.0.1:8090',
    });
  });
});

describe('api/wallet · admin withdrawal approval', () => {
  it('approveWithdrawal POSTs to approve', async () => {
    await approveWithdrawal(99);
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/admin/wallet/withdrawals/99/approve',
      method: 'POST',
      data: undefined,
      baseURL: 'http://127.0.0.1:8090',
    });
  });

  it('markWithdrawalPaid with channel_tx_id', async () => {
    await markWithdrawalPaid(99, 'WX20261001');
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ channel_tx_id: 'WX20261001' });
  });

  it('markWithdrawalPaid empty body when no tx id', async () => {
    await markWithdrawalPaid(99);
    expect(requestSpy.mock.calls[0]![0].data).toEqual({});
  });

  it('rejectWithdrawal requires reason', async () => {
    await rejectWithdrawal(99, '账户信息有误');
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/admin/wallet/withdrawals/99/reject',
      method: 'POST',
      data: { reason: '账户信息有误' },
      baseURL: 'http://127.0.0.1:8090',
    });
  });
});