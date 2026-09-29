/**
 * admin/orders/detail.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - mount 时 parseQuery 提取 id 并调用 store.fetchDetail
 *   - 渲染订单详情（id、状态、医院、地址、金额分→元）
 *   - 「强制取消订单」按钮在 canForceCancel 状态（pending_escort / escort_confirmed / in_service）下显示
 *   - 已完成 / 已取消订单不显示强制取消按钮
 *   - 点击「强制取消订单」打开 modal
 *   - 输入原因 + 确认 → 调 forceCancelOrder + 重新 fetchDetail + 关闭 modal
 *   - 「返回」按钮关闭 modal
 *
 * 策略：mock api/orders + api/admin；uni.getCurrentPages 返回带 options 的栈底。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import OrderDetail from './detail.vue';

vi.mock('@/api/orders', () => ({
  listOrders: vi.fn(),
  getOrder: vi.fn(),
  createOrder: vi.fn(),
  cancelOrder: vi.fn(),
  confirmAccept: vi.fn(),
  rejectAccept: vi.fn(),
  finishOrder: vi.fn(),
}));

vi.mock('@/api/admin', () => ({
  forceCancelOrder: vi.fn(),
}));

import * as apiOrders from '@/api/orders';
import * as apiAdmin from '@/api/admin';
import type { Order, OrderStatus } from '@/api/orders';

const makeOrder = (id: number, status: OrderStatus, extras: Partial<Order> = {}): Order => ({
  id,
  patient_id: 100,
  escort_id: null,
  hospital_id: 5,
  package_id: 7,
  status,
  appointment_time: '2026-10-01T09:00:00Z',
  address: '北京协和医院',
  total_amount: 29900,
  created_at: '2026-09-29T10:00:00Z',
  updated_at: '2026-09-29T10:00:00Z',
  ...extras,
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
  // 默认让 getCurrentPages 返回挂载时的 orderId=101
  (globalThis as unknown as {
    uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
  }).uni.getCurrentPages = () => [{ options: { id: '101' } }];
});

describe('admin/orders/detail · 加载', () => {
  it('parseQuery 提取 id 并调用 fetchDetail(101)', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'pending_escort'));
    mount(OrderDetail);
    await flushPromises();
    expect(apiOrders.getOrder).toHaveBeenCalledWith(101);
  });

  it('未传 id 时不调 fetchDetail', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: {} }];
    mount(OrderDetail);
    await flushPromises();
    expect(apiOrders.getOrder).not.toHaveBeenCalled();
  });
});

describe('admin/orders/detail · 渲染', () => {
  it('显示订单 id、状态、医院、地址、金额（分→元）', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'in_service'));
    const w = mount(OrderDetail);
    await flushPromises();

    expect(w.find('[data-testid="admin-order-detail-header"]').exists()).toBe(true);
    expect(w.text()).toContain('订单 #101');
    expect(w.text()).toContain('服务中');
    expect(w.text()).toContain('北京协和医院');
    expect(w.text()).toContain('¥299.00');
  });
});

describe('admin/orders/detail · 强制取消按钮可见性', () => {
  it('pending_escort 状态显示强制取消按钮', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'pending_escort'));
    const w = mount(OrderDetail);
    await flushPromises();
    expect(w.find('[data-testid="admin-order-detail-actions"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-order-detail-force-cancel-btn"]').exists()).toBe(true);
  });

  it.each(['escort_confirmed', 'in_service'] as OrderStatus[])(
    '%s 状态显示强制取消按钮',
    async (status) => {
      vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, status));
      const w = mount(OrderDetail);
      await flushPromises();
      expect(w.find('[data-testid="admin-order-detail-force-cancel-btn"]').exists()).toBe(true);
    },
  );

  it.each(['completed', 'cancelled'] as OrderStatus[])(
    '%s 状态不显示强制取消按钮',
    async (status) => {
      vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, status));
      const w = mount(OrderDetail);
      await flushPromises();
      expect(w.find('[data-testid="admin-order-detail-actions"]').exists()).toBe(false);
    },
  );
});

describe('admin/orders/detail · 强制取消流程', () => {
  beforeEach(() => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'pending_escort'));
    vi.mocked(apiAdmin.forceCancelOrder).mockResolvedValue(makeOrder(101, 'cancelled'));
  });

  it('点击「强制取消订单」打开 modal', async () => {
    const w = mount(OrderDetail);
    await flushPromises();
    expect(w.find('[data-testid="admin-order-cancel-modal"]').exists()).toBe(false);

    await w.find('[data-testid="admin-order-detail-force-cancel-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-order-cancel-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 forceCancelOrder 并重新 fetchDetail', async () => {
    const w = mount(OrderDetail);
    await flushPromises();
    vi.clearAllMocks(); // 清除初始 fetchDetail

    // 打开 modal
    await w.find('[data-testid="admin-order-detail-force-cancel-btn"]').trigger('click');
    await flushPromises();

    // 输入原因（UiInput 内部 textarea/input 才是真表单元素）
    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('测试取消原因');
    await flushPromises();

    // 确认取消
    await w.find('[data-testid="admin-order-cancel-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.forceCancelOrder).toHaveBeenCalledWith(101, '测试取消原因');
    // 取消后会重新 fetchDetail
    expect(apiOrders.getOrder).toHaveBeenCalledWith(101);
    // modal 关闭
    expect(w.find('[data-testid="admin-order-cancel-modal"]').exists()).toBe(false);
  });

  it('原因为空时确认按钮 disabled，不调 forceCancelOrder', async () => {
    const w = mount(OrderDetail);
    await flushPromises();

    await w.find('[data-testid="admin-order-detail-force-cancel-btn"]').trigger('click');
    await flushPromises();

    const confirmBtn = w.find('[data-testid="admin-order-cancel-confirm"]');
    expect(confirmBtn.attributes('disabled')).toBeDefined();

    // 直接尝试点击（disabled 状态）
    await confirmBtn.trigger('click');
    await flushPromises();
    expect(apiAdmin.forceCancelOrder).not.toHaveBeenCalled();
  });

  it('点击「返回」关闭 modal，不调 forceCancelOrder', async () => {
    const w = mount(OrderDetail);
    await flushPromises();

    await w.find('[data-testid="admin-order-detail-force-cancel-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-order-cancel-modal"]').exists()).toBe(true);

    await w.find('[data-testid="admin-order-cancel-back"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-order-cancel-modal"]').exists()).toBe(false);
    expect(apiAdmin.forceCancelOrder).not.toHaveBeenCalled();
  });
});