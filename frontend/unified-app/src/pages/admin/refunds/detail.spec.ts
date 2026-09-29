/**
 * admin/refunds/detail.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - parseQuery 提取 id 并从 listRefunds 过滤找到 refund
 *   - 渲染基本信息（id / 订单 ID / 支付 ID / 原因 / 金额 / 状态徽章 / 时间）
 *   - pending 状态显示「通过」/「驳回」按钮
 *   - approved/rejected 状态仅显示「返回列表」
 *   - 通过 / 驳回 / 返回 三个流程
 *   - 未传 id / 找不到 refund / API 异常 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import RefundDetail from './detail.vue';

vi.mock('@/api/admin', () => ({
  listRefunds: vi.fn(),
  approveRefund: vi.fn(),
  rejectRefund: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { Refund } from '@/api/admin';

const makeRefund = (id: number, status: Refund['status']): Refund => ({
  id,
  payment_id: 1000 + id,
  order_id: 2000 + id,
  amount: 29900,
  reason: `退款原因 #${id}`,
  status,
  created_at: '2026-09-29T10:00:00Z',
});

beforeEach(() => {
  vi.clearAllMocks();
  (globalThis as unknown as {
    uni: {
      getCurrentPages: () => Array<{ options?: Record<string, string> }>;
      navigateBack: (opts?: { delta?: number }) => Promise<void>;
    };
  }).uni.getCurrentPages = () => [{ options: { id: '101' } }];
});

describe('admin/refunds/detail · 加载', () => {
  it('parseQuery 提取 id 并从 listRefunds 过滤找到', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(100, 'pending'), makeRefund(101, 'pending')],
      total: 2,
    });
    const w = mount(RefundDetail);
    await flushPromises();

    expect(apiAdmin.listRefunds).toHaveBeenCalledTimes(1);
    expect(w.text()).toContain('退款 #101');
    expect(w.text()).toContain('订单 ID');
    expect(w.text()).toContain('#2101');
    expect(w.text()).toContain('支付 ID');
    expect(w.text()).toContain('#1101');
    expect(w.text()).toContain('退款原因 #101');
    expect(w.text()).toContain('¥299.00');
    expect(w.text()).toContain('2026-09-29 10:00');
  });

  it('未传 id 时显示错误', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: {} }];
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-error"]').exists()).toBe(true);
    expect(w.text()).toContain('未指定退款 ID');
  });

  it('找不到 refund 显示「退款工单不存在」', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-error"]').exists()).toBe(true);
    expect(w.text()).toContain('退款工单不存在');
  });

  it('API 异常显示错误', async () => {
    vi.mocked(apiAdmin.listRefunds).mockRejectedValue(new Error('网络超时'));
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});

describe('admin/refunds/detail · 操作按钮可见性', () => {
  it('pending 显示「通过」/「驳回」/「返回列表」3 按钮', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-approve"]').exists()).toBe(true);
    expect(w.find('[data-testid="refund-detail-reject"]').exists()).toBe(true);
    expect(w.find('[data-testid="refund-detail-back"]').exists()).toBe(true);
  });

  it.each(['approved', 'rejected'] as const)('%s 状态不显示审批按钮', async (status) => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, status)],
      total: 1,
    });
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-approve"]').exists()).toBe(false);
    expect(w.find('[data-testid="refund-detail-reject"]').exists()).toBe(false);
    expect(w.find('[data-testid="refund-detail-back"]').exists()).toBe(true);
  });
});

describe('admin/refunds/detail · 通过流程', () => {
  it('点击「通过」打开 modal', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-approve-modal"]').exists()).toBe(false);
    await w.find('[data-testid="refund-detail-approve"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="refund-detail-approve-modal"]').exists()).toBe(true);
  });

  it('输入备注 + 确认 → 调 approveRefund(id, note) + navigateBack', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    vi.mocked(apiAdmin.approveRefund).mockResolvedValue(makeRefund(101, 'approved'));

    const backSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateBack: typeof backSpy } }).uni.navigateBack = backSpy;

    const w = mount(RefundDetail);
    await flushPromises();
    await w.find('[data-testid="refund-detail-approve"]').trigger('click');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('已联系财务');
    await flushPromises();

    await w.find('[data-testid="refund-detail-approve-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.approveRefund).toHaveBeenCalledWith(101, '已联系财务');
    expect(backSpy).toHaveBeenCalled();
  });
});

describe('admin/refunds/detail · 驳回流程', () => {
  beforeEach(() => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
  });

  it('点击「驳回」打开 modal', async () => {
    const w = mount(RefundDetail);
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-reject-modal"]').exists()).toBe(false);
    await w.find('[data-testid="refund-detail-reject"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="refund-detail-reject-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 rejectRefund + navigateBack', async () => {
    vi.mocked(apiAdmin.rejectRefund).mockResolvedValue(makeRefund(101, 'rejected'));
    const backSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateBack: typeof backSpy } }).uni.navigateBack = backSpy;

    const w = mount(RefundDetail);
    await flushPromises();
    await w.find('[data-testid="refund-detail-reject"]').trigger('click');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('已在其他渠道退款');
    await flushPromises();

    await w.find('[data-testid="refund-detail-reject-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.rejectRefund).toHaveBeenCalledWith(101, '已在其他渠道退款');
    expect(backSpy).toHaveBeenCalled();
  });

  it('原因为空时 confirm 按钮 disabled', async () => {
    const w = mount(RefundDetail);
    await flushPromises();
    await w.find('[data-testid="refund-detail-reject"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="refund-detail-reject-confirm"]').attributes('disabled')).toBeDefined();
  });
});

describe('admin/refunds/detail · 返回', () => {
  it('点击「返回列表」调 navigateBack', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    const backSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateBack: typeof backSpy } }).uni.navigateBack = backSpy;

    const w = mount(RefundDetail);
    await flushPromises();
    await w.find('[data-testid="refund-detail-back"]').trigger('click');
    expect(backSpy).toHaveBeenCalled();
  });
});