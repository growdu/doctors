/**
 * escort/wallet/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时并发拉钱包 + 流水
 *   - 渲染余额卡（4 指标）
 *   - 4 tab + 流水列表渲染
 *   - 提现按钮跳 /pages/escort/wallet/withdraw
 *   - 金额正负渲染（收入 + / 提现 - / 退款 -）
 *   - 钱包加载 / 流水加载 / 错误三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import WalletPage from './index.vue';

vi.mock('@/api/wallet', () => ({
  getEscortWallet: vi.fn(),
  listTransactions: vi.fn(),
  getUserWallet: vi.fn(),
  requestWithdrawal: vi.fn(),
  approveWithdrawal: vi.fn(),
  markWithdrawalPaid: vi.fn(),
  rejectWithdrawal: vi.fn(),
}));

import * as apiWallet from '@/api/wallet';
import type { Wallet, Transaction, TransactionType } from '@/api/wallet';

const makeWallet = (overrides: Partial<Wallet> = {}): Wallet => ({
  user_id: 2001,
  balance: 125000,
  frozen: 5000,
  total_earned: 200000,
  total_withdrawn: 80000,
  updated_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

const makeTx = (id: number, type: TransactionType, amount: number): Transaction => ({
  id,
  wallet_user_id: 2001,
  type,
  amount,
  status: 'success',
  order_id: type === 'income' || type === 'refund' ? 100 + id : null,
  remark: '',
  created_at: '2026-09-29T10:00:00Z',
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('escort/wallet · 加载', () => {
  it('mount 时并发调 getEscortWallet + listTransactions', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    mount(WalletPage);
    await flushPromises();

    expect(apiWallet.getEscortWallet).toHaveBeenCalledTimes(1);
    expect(apiWallet.listTransactions).toHaveBeenCalledTimes(1);
  });

  it('渲染余额卡（4 指标）', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    const w = mount(WalletPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-wallet-card"]').exists()).toBe(true);
    expect(w.text()).toContain('可用余额');
    expect(w.text()).toContain('冻结金额');
    expect(w.text()).toContain('累计收入');
    expect(w.text()).toContain('累计提现');

    expect(w.find('[data-testid="escort-wallet-balance"]').text()).toContain('¥1,250.00');
    expect(w.find('[data-testid="escort-wallet-frozen"]').text()).toContain('¥50.00');
    expect(w.find('[data-testid="escort-wallet-earned"]').text()).toContain('¥2,000.00');
    expect(w.find('[data-testid="escort-wallet-withdrawn"]').text()).toContain('¥800.00');
  });
});

describe('escort/wallet · 流水 tab', () => {
  it('渲染 4 tab', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    const w = mount(WalletPage);
    await flushPromises();

    for (const t of ['all', 'income', 'withdraw', 'refund']) {
      expect(w.find(`[data-testid="escort-wallet-tab-${t}"]`).exists()).toBe(true);
    }
  });

  it('点击 income tab 重新拉数据并带 ?type=income', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    const w = mount(WalletPage);
    await flushPromises();

    await w.find('[data-testid="escort-wallet-tab-income"]').trigger('click');
    await flushPromises();

    expect(apiWallet.listTransactions).toHaveBeenLastCalledWith({ type: 'income' });
  });
});

describe('escort/wallet · 流水渲染', () => {
  it('渲染流水卡 + 收入金额前加 +', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({
      items: [makeTx(101, 'income', 10000), makeTx(102, 'withdraw', 5000)],
      total: 2,
    });

    const w = mount(WalletPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-wallet-tx-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-wallet-tx-102"]').exists()).toBe(true);
    expect(w.text()).toContain('收入');
    expect(w.text()).toContain('提现');
    expect(w.text()).toContain('订单 #201');
    expect(w.find('[data-testid="escort-wallet-tx-amount-101"]').text()).toContain('+¥100.00');
    expect(w.find('[data-testid="escort-wallet-tx-amount-102"]').text()).toContain('-¥50.00');
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    const w = mount(WalletPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-wallet-tx-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无流水');
  });
});

describe('escort/wallet · 提现跳转', () => {
  it('点击「提现」按钮跳 /pages/escort/wallet/withdraw', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    const w = mount(WalletPage);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="escort-wallet-withdraw-btn"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/wallet/withdraw' });
  });
});

describe('escort/wallet · 错误态', () => {
  it('钱包 API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockRejectedValue(new Error('网络异常'));
    vi.mocked(apiWallet.listTransactions).mockResolvedValue({ items: [], total: 0 });

    const w = mount(WalletPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-wallet-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });

  it('流水 API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    vi.mocked(apiWallet.listTransactions).mockRejectedValue(new Error('流水加载失败'));

    const w = mount(WalletPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-wallet-tx-error"]').exists()).toBe(true);
    expect(w.text()).toContain('流水加载失败');
  });
});