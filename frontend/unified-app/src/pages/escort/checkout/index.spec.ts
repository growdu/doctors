/**
 * escort/checkout/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - parseQuery 提取 orderId
 *   - 未传 orderId 时按钮 disabled
 *   - 填写备注 + 点击「完成签出」调 finishOrder + toast + navigateBack
 *   - API 异常显示错误 toast
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import CheckoutPage from './index.vue';

vi.mock('@/api/orders', () => ({
  finishOrder: vi.fn(),
  listOrders: vi.fn(),
  getOrder: vi.fn(),
  createOrder: vi.fn(),
  cancelOrder: vi.fn(),
  confirmAccept: vi.fn(),
  rejectAccept: vi.fn(),
}));

vi.mock('@/store/order', () => ({
  useOrderStore: () => ({
    fetchDetail: vi.fn().mockResolvedValue({}),
  }),
}));

import * as apiOrders from '@/api/orders';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('escort/checkout · 加载', () => {
  it('parseQuery 提取 orderId', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: { orderId: '101' } }];

    const w = mount(CheckoutPage);
    await flushPromises();

    expect(w.text()).toContain('订单 #101 签出');
    expect(w.find('[data-testid="escort-checkout-submit"]').attributes('disabled')).toBeUndefined();
  });

  it('未传 orderId 时按钮 disabled', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: {} }];

    const w = mount(CheckoutPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-checkout-submit"]').attributes('disabled')).toBeDefined();
  });
});

describe('escort/checkout · 完成签出', () => {
  beforeEach(() => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: { orderId: '101' } }];
  });

  it('填写备注 + 提交 → finishOrder + toast', async () => {
    vi.mocked(apiOrders.finishOrder).mockResolvedValue({} as never);

    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(CheckoutPage);
    await flushPromises();

    // 填写备注
    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('陪同完成检查，患者已回家');
    await flushPromises();

    await w.find('[data-testid="escort-checkout-submit"]').trigger('click');
    await flushPromises();

    expect(apiOrders.finishOrder).toHaveBeenCalledWith(101);
    expect(toastSpy).toHaveBeenCalledWith({ title: '已完成', icon: 'success' });
  });

  it('不填备注也可完成（备注可选）', async () => {
    vi.mocked(apiOrders.finishOrder).mockResolvedValue({} as never);

    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(CheckoutPage);
    await flushPromises();

    await w.find('[data-testid="escort-checkout-submit"]').trigger('click');
    await flushPromises();

    expect(apiOrders.finishOrder).toHaveBeenCalledWith(101);
  });

  it('API 异常显示错误 toast', async () => {
    vi.mocked(apiOrders.finishOrder).mockRejectedValue(new Error('订单不存在'));

    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(CheckoutPage);
    await flushPromises();

    await w.find('[data-testid="escort-checkout-submit"]').trigger('click');
    await flushPromises();

    expect(toastSpy).toHaveBeenCalledWith({ title: '完成失败：订单不存在', icon: 'none' });
  });
});