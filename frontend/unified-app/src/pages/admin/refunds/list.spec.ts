/**
 * admin/refunds/list.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listRefunds
 *   - 4 个 status tab + 默认 active 是 all
 *   - 点击 tab 重新拉数据（带 ?status=xxx）
 *   - 渲染工单卡（id / 原因 / 金额 / 时间 / 状态徽章）
 *   - pending 工单显示「详情 / 通过 / 驳回」3 按钮
 *   - approved/rejected 工单仅显示「详情」按钮
 *   - 「详情」跳 detail?id=
 *   - 「通过」打开 approve modal → 确认调 approveRefund + 移除
 *   - 「驳回」打开 reject modal → 输入原因 → 确认调 rejectRefund + 移除
 *   - 驳回原因为空时 confirm 按钮 disabled
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import RefundsList from './list.vue';

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
});

describe('admin/refunds/list · 加载与 tab', () => {
  it('mount 时调 listRefunds 不带 status（默认 all）', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    mount(RefundsList);
    await flushPromises();
    expect(apiAdmin.listRefunds).toHaveBeenCalledWith({});
  });

  it('渲染 4 个 tab（all / pending / approved / rejected）', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    const w = mount(RefundsList);
    await flushPromises();

    for (const s of ['all', 'pending', 'approved', 'rejected']) {
      expect(w.find(`[data-testid="refunds-list-tab-${s}"]`).exists()).toBe(true);
    }
  });

  it('默认 active tab 是 all', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    const w = mount(RefundsList);
    await flushPromises();
    expect(w.find('[data-testid="refunds-list-tab-all"]').classes()).toContain('admin-refunds-list__tab--active');
  });

  it('点击 pending tab 重新拉数据并带 ?status=pending', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    const w = mount(RefundsList);
    await flushPromises();
    expect(apiAdmin.listRefunds).toHaveBeenCalledTimes(1);

    await w.find('[data-testid="refunds-list-tab-pending"]').trigger('click');
    await flushPromises();
    expect(apiAdmin.listRefunds).toHaveBeenCalledTimes(2);
    expect(apiAdmin.listRefunds).toHaveBeenLastCalledWith({ status: 'pending' });
  });
});

describe('admin/refunds/list · 渲染', () => {
  it('渲染工单卡（id / 原因 / 金额 / 时间）', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending'), makeRefund(102, 'approved'), makeRefund(103, 'rejected')],
      total: 3,
    });
    const w = mount(RefundsList);
    await flushPromises();

    expect(w.find('[data-testid="refunds-list-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-list-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('退款 #101');
    expect(w.text()).toContain('订单 #2101');
    expect(w.text()).toContain('退款原因 #101');
    expect(w.text()).toContain('¥299.00');
    expect(w.text()).toContain('2026-09-29 10:00');
  });

  it('pending 显示「详情/通过/驳回」3 按钮', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    const w = mount(RefundsList);
    await flushPromises();

    expect(w.find('[data-testid="refunds-list-detail-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-list-approve-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-list-reject-101"]').exists()).toBe(true);
  });

  it('approved/rejected 仅显示「详情」按钮', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(102, 'approved'), makeRefund(103, 'rejected')],
      total: 2,
    });
    const w = mount(RefundsList);
    await flushPromises();

    expect(w.find('[data-testid="refunds-list-detail-102"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-list-approve-102"]').exists()).toBe(false);
    expect(w.find('[data-testid="refunds-list-reject-102"]').exists()).toBe(false);

    expect(w.find('[data-testid="refunds-list-detail-103"]').exists()).toBe(true);
    expect(w.find('[data-testid="refunds-list-approve-103"]').exists()).toBe(false);
  });
});

describe('admin/refunds/list · 跳转', () => {
  it('点击「详情」跳 detail?id=', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    const w = mount(RefundsList);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="refunds-list-detail-101"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/refunds/detail?id=101' });
  });
});

describe('admin/refunds/list · 通过流程', () => {
  it('点击「通过」打开 approve modal', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    const w = mount(RefundsList);
    await flushPromises();

    expect(w.find('[data-testid="refunds-approve-modal"]').exists()).toBe(false);
    await w.find('[data-testid="refunds-list-approve-101"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="refunds-approve-modal"]').exists()).toBe(true);
  });

  it('输入备注 + 确认 → 调 approveRefund(id, note) 并移除', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    vi.mocked(apiAdmin.approveRefund).mockResolvedValue(makeRefund(101, 'approved'));

    const w = mount(RefundsList);
    await flushPromises();
    await w.find('[data-testid="refunds-list-approve-101"]').trigger('click');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('已联系财务');
    await flushPromises();

    await w.find('[data-testid="refunds-approve-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.approveRefund).toHaveBeenCalledWith(101, '已联系财务');
    expect(w.find('[data-testid="refunds-list-card-101"]').exists()).toBe(false);
  });

  it('不填备注也可通过（note 为 undefined）', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
    vi.mocked(apiAdmin.approveRefund).mockResolvedValue(makeRefund(101, 'approved'));

    const w = mount(RefundsList);
    await flushPromises();
    await w.find('[data-testid="refunds-list-approve-101"]').trigger('click');
    await flushPromises();
    await w.find('[data-testid="refunds-approve-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.approveRefund).toHaveBeenCalledWith(101, undefined);
  });
});

describe('admin/refunds/list · 驳回流程', () => {
  beforeEach(() => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({
      items: [makeRefund(101, 'pending')],
      total: 1,
    });
  });

  it('点击「驳回」打开 reject modal', async () => {
    const w = mount(RefundsList);
    await flushPromises();

    expect(w.find('[data-testid="refunds-reject-modal"]').exists()).toBe(false);
    await w.find('[data-testid="refunds-list-reject-101"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="refunds-reject-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 rejectRefund 并移除', async () => {
    vi.mocked(apiAdmin.rejectRefund).mockResolvedValue(makeRefund(101, 'rejected'));
    const w = mount(RefundsList);
    await flushPromises();

    await w.find('[data-testid="refunds-list-reject-101"]').trigger('click');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('不在退款范围');
    await flushPromises();

    await w.find('[data-testid="refunds-reject-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.rejectRefund).toHaveBeenCalledWith(101, '不在退款范围');
    expect(w.find('[data-testid="refunds-list-card-101"]').exists()).toBe(false);
  });

  it('原因为空时 confirm 按钮 disabled', async () => {
    const w = mount(RefundsList);
    await flushPromises();
    await w.find('[data-testid="refunds-list-reject-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="refunds-reject-confirm"]').attributes('disabled')).toBeDefined();
  });

  it('点击「返回」关闭 modal，不调 rejectRefund', async () => {
    const w = mount(RefundsList);
    await flushPromises();
    await w.find('[data-testid="refunds-list-reject-101"]').trigger('click');
    await flushPromises();

    await w.find('[data-testid="refunds-reject-cancel"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="refunds-reject-modal"]').exists()).toBe(false);
    expect(apiAdmin.rejectRefund).not.toHaveBeenCalled();
  });
});

describe('admin/refunds/list · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiAdmin.listRefunds).mockReturnValue(new Promise(() => {}));
    const w = mount(RefundsList);
    await flushPromises();
    expect(w.find('[data-testid="refunds-list-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiAdmin.listRefunds).mockResolvedValue({ items: [], total: 0 });
    const w = mount(RefundsList);
    await flushPromises();
    expect(w.find('[data-testid="refunds-list-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无退款工单');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiAdmin.listRefunds).mockRejectedValue(new Error('网络超时'));
    const w = mount(RefundsList);
    await flushPromises();
    expect(w.find('[data-testid="refunds-list-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});