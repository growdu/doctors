/**
 * admin/finance/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时并发拉 3 个类型 total + listBillings
 *   - 3 概览卡 + 4 tab 渲染
 *   - 渲染账单卡（id / 类型徽章 / 订单 ID / 金额 / 时间）
 *   - 金额格式化（分→元，正负号）
 *   - 点击 tab 重新拉数据（带 ?type=）
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import FinancePage from './index.vue';

vi.mock('@/api/admin', () => ({
  listBillings: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { Billing } from '@/api/admin';

const makeBilling = (id: number, type: Billing['type'], amount: number): Billing => ({
  id,
  order_id: 2000 + id,
  amount,
  type,
  created_at: '2026-09-29T10:00:00Z',
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/finance/index · 加载', () => {
  it('mount 时并发拉 3 个类型 total + listBillings', async () => {
    vi.mocked(apiAdmin.listBillings)
      // overview 3 calls
      .mockResolvedValueOnce({ items: [], total: 10 }) // income
      .mockResolvedValueOnce({ items: [], total: 2 }) // refund
      .mockResolvedValueOnce({ items: [], total: 1 }) // withdraw
      // main list call
      .mockResolvedValueOnce({ items: [], total: 0 });

    mount(FinancePage);
    await flushPromises();

    expect(apiAdmin.listBillings).toHaveBeenCalledTimes(4);
    expect(apiAdmin.listBillings).toHaveBeenCalledWith({ type: 'income', page: 1, page_size: 1 });
    expect(apiAdmin.listBillings).toHaveBeenCalledWith({ type: 'refund', page: 1, page_size: 1 });
    expect(apiAdmin.listBillings).toHaveBeenCalledWith({ type: 'withdraw', page: 1, page_size: 1 });
    expect(apiAdmin.listBillings).toHaveBeenCalledWith({}); // main list
  });

  it('渲染 3 概览卡 + 4 tab', async () => {
    vi.mocked(apiAdmin.listBillings).mockResolvedValue({ items: [], total: 0 });
    const w = mount(FinancePage);
    await flushPromises();

    expect(w.find('[data-testid="admin-finance-overview-income"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-finance-overview-refund"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-finance-overview-withdraw"]').exists()).toBe(true);

    for (const t of ['all', 'income', 'refund', 'withdraw']) {
      expect(w.find(`[data-testid="admin-finance-tab-${t}"]`).exists()).toBe(true);
    }
  });

  it('概览数字来自 API total', async () => {
    vi.mocked(apiAdmin.listBillings)
      .mockResolvedValueOnce({ items: [], total: 5 })
      .mockResolvedValueOnce({ items: [], total: 3 })
      .mockResolvedValueOnce({ items: [], total: 1 })
      .mockResolvedValueOnce({ items: [], total: 0 });

    const w = mount(FinancePage);
    await flushPromises();

    expect(w.text()).toContain('收入');
    expect(w.text()).toContain('退款');
    expect(w.text()).toContain('提现');
    // 三个数字 5/3/1 应展示
    expect(w.find('[data-testid="admin-finance-overview-income"]').text()).toContain('5');
    expect(w.find('[data-testid="admin-finance-overview-refund"]').text()).toContain('3');
    expect(w.find('[data-testid="admin-finance-overview-withdraw"]').text()).toContain('1');
  });
});

describe('admin/finance/index · tab 切换', () => {
  it('点击 income tab 重新拉数据并带 ?type=income', async () => {
    vi.mocked(apiAdmin.listBillings).mockResolvedValue({ items: [], total: 0 });
    const w = mount(FinancePage);
    await flushPromises();

    await w.find('[data-testid="admin-finance-tab-income"]').trigger('click');
    await flushPromises();

    // 至少一次 type=income 调用（最后一次）
    const calls = vi.mocked(apiAdmin.listBillings).mock.calls;
    const last = calls[calls.length - 1]![0] as { type?: string };
    expect(last.type).toBe('income');
  });
});

describe('admin/finance/index · 列表渲染', () => {
  it('渲染账单卡（id / 类型 / 订单 ID / 金额 / 时间）', async () => {
    vi.mocked(apiAdmin.listBillings)
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({
        items: [
          makeBilling(101, 'income', 29900),
          makeBilling(102, 'refund', -10000),
        ],
        total: 2,
      });

    const w = mount(FinancePage);
    await flushPromises();

    expect(w.find('[data-testid="admin-finance-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-finance-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('账单 #101');
    expect(w.text()).toContain('订单 #2101');
    expect(w.text()).toContain('¥299.00');
    expect(w.text()).toContain('-¥100.00');
    expect(w.text()).toContain('收入');
    expect(w.text()).toContain('退款');
  });

  it('类型徽章显示对应文案', async () => {
    vi.mocked(apiAdmin.listBillings)
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({
        items: [
          makeBilling(1, 'income', 100),
          makeBilling(2, 'refund', -100),
          makeBilling(3, 'withdraw', -100),
        ],
        total: 3,
      });

    const w = mount(FinancePage);
    await flushPromises();

    expect(w.find('[data-testid="admin-finance-type-1"]').text()).toContain('收入');
    expect(w.find('[data-testid="admin-finance-type-2"]').text()).toContain('退款');
    expect(w.find('[data-testid="admin-finance-type-3"]').text()).toContain('提现');
  });
});

describe('admin/finance/index · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiAdmin.listBillings).mockReturnValue(new Promise(() => {}));
    const w = mount(FinancePage);
    await flushPromises();
    expect(w.find('[data-testid="admin-finance-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiAdmin.listBillings).mockResolvedValue({ items: [], total: 0 });
    const w = mount(FinancePage);
    await flushPromises();
    expect(w.find('[data-testid="admin-finance-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无账单');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiAdmin.listBillings).mockRejectedValue(new Error('网络超时'));
    const w = mount(FinancePage);
    await flushPromises();
    expect(w.find('[data-testid="admin-finance-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});