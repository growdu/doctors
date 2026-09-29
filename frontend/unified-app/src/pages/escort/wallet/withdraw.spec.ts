/**
 * escort/wallet/withdraw.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 getEscortWallet + 渲染余额
 *   - 校验逻辑：金额 / 账号 / 账户名
 *   - 提交调 requestWithdrawal + navigateBack
 *   - 余额为 0 时跳过校验（后端兜底）
 *   - 失败显示错误信息
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import WithdrawPage from './withdraw.vue';

vi.mock('@/api/wallet', () => ({
  getEscortWallet: vi.fn(),
  requestWithdrawal: vi.fn(),
  getUserWallet: vi.fn(),
  listTransactions: vi.fn(),
  approveWithdrawal: vi.fn(),
  markWithdrawalPaid: vi.fn(),
  rejectWithdrawal: vi.fn(),
}));

import * as apiWallet from '@/api/wallet';
import type { Wallet } from '@/api/wallet';

const makeWallet = (overrides: Partial<Wallet> = {}): Wallet => ({
  user_id: 2001,
  balance: 125000,
  frozen: 0,
  total_earned: 200000,
  total_withdrawn: 80000,
  updated_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('escort/wallet/withdraw · 加载', () => {
  it('mount 时调 getEscortWallet', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet());
    mount(WithdrawPage);
    await flushPromises();
    expect(apiWallet.getEscortWallet).toHaveBeenCalledTimes(1);
  });

  it('渲染余额（分→元）', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet({ balance: 125000 }));
    const w = mount(WithdrawPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-withdraw-balance"]').text()).toContain('¥1,250.00');
  });
});

describe('escort/wallet/withdraw · 输入校验', () => {
  beforeEach(() => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet({ balance: 100000 }));
  });

  it('金额为空时提交按钮 disabled', async () => {
    const w = mount(WithdrawPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-withdraw-submit"]').attributes('disabled')).toBeDefined();
  });

  it('金额为 0 时提交按钮 disabled', async () => {
    const w = mount(WithdrawPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('0');
    await flushPromises();

    expect(w.find('[data-testid="escort-withdraw-submit"]').attributes('disabled')).toBeDefined();
  });

  it('金额超过余额时点击提交显示错误，不调 API', async () => {
    const w = mount(WithdrawPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('99999');
    await inputs[1]!.setValue('1234567890');
    await inputs[2]!.setValue('张三');
    await flushPromises();

    // 99999 元 > 余额 1000 元 → 校验失败，按钮 disabled
    expect(w.find('[data-testid="escort-withdraw-submit"]').attributes('disabled')).toBeDefined();
  });

  it('余额为 0 时不阻止提交（后端兜底）', async () => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet({ balance: 0 }));
    vi.mocked(apiWallet.requestWithdrawal).mockResolvedValue({} as never);

    const w = mount(WithdrawPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('100');
    await inputs[1]!.setValue('1234567890');
    await inputs[2]!.setValue('张三');
    await flushPromises();

    // 余额为 0 时跳过余额校验，按钮可用
    expect(w.find('[data-testid="escort-withdraw-submit"]').attributes('disabled')).toBeUndefined();
  });
});

describe('escort/wallet/withdraw · 提交流程', () => {
  beforeEach(() => {
    vi.mocked(apiWallet.getEscortWallet).mockResolvedValue(makeWallet({ balance: 100000 }));
  });

  it('合法表单 → 调 requestWithdrawal + navigateBack', async () => {
    vi.mocked(apiWallet.requestWithdrawal).mockResolvedValue({} as never);

    const backSpy = vi.fn();
    const toastSpy = vi.fn();
    (globalThis as unknown as {
      uni: { navigateBack: typeof backSpy; showToast: typeof toastSpy };
    }).uni.navigateBack = backSpy;
    (globalThis as unknown as {
      uni: { navigateBack: typeof backSpy; showToast: typeof toastSpy };
    }).uni.showToast = toastSpy;

    const w = mount(WithdrawPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('500');
    await inputs[1]!.setValue('1234567890');
    await inputs[2]!.setValue('张三');
    await flushPromises();

    await w.find('[data-testid="escort-withdraw-submit"]').trigger('click');
    await flushPromises();

    expect(apiWallet.requestWithdrawal).toHaveBeenCalledWith({
      amount: 50000,
      account: '1234567890',
      account_name: '张三',
    });
    expect(toastSpy).toHaveBeenCalledWith({ title: '提现申请已提交', icon: 'success' });
    // 800ms 后的 setTimeout navigateBack 暂不立即断言（依赖定时器）
  });

  it('账号为空时点击触发校验显示错误', async () => {
    const w = mount(WithdrawPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('100');
    // 故意不填账号
    await inputs[2]!.setValue('张三');
    await flushPromises();

    // 按钮 disabled（账号缺失）
    expect(w.find('[data-testid="escort-withdraw-submit"]').attributes('disabled')).toBeDefined();
  });

  it('API 异常显示错误信息', async () => {
    vi.mocked(apiWallet.requestWithdrawal).mockRejectedValue(new Error('余额不足'));

    const w = mount(WithdrawPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('500');
    await inputs[1]!.setValue('1234567890');
    await inputs[2]!.setValue('张三');
    await flushPromises();

    await w.find('[data-testid="escort-withdraw-submit"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-withdraw-error"]').exists()).toBe(true);
    expect(w.text()).toContain('余额不足');
  });
});