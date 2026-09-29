/**
 * escort/order-detail.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - parseQuery 提取 id 并调用 fetchDetail
 *   - 渲染订单详情
 *   - escort_confirmed 显示「开始服务」按钮 → confirmAccept
 *   - in_service 显示「完成任务」按钮 → finishOrder
 *   - 其他状态不显示操作按钮
 *   - 操作完成后重新 fetchDetail 拉最新状态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import OrderDetail from './order-detail.vue';

vi.mock('@/api/orders', () => ({
  listOrders: vi.fn(),
  getOrder: vi.fn(),
  createOrder: vi.fn(),
  cancelOrder: vi.fn(),
  confirmAccept: vi.fn(),
  rejectAccept: vi.fn(),
  finishOrder: vi.fn(),
}));

import * as apiOrders from '@/api/orders';
import { useAuthStore } from '@/store/auth';
import type { Order, OrderStatus } from '@/api/orders';

const makeOrder = (id: number, status: OrderStatus): Order => ({
  id,
  patient_id: 100,
  escort_id: 200,
  hospital_id: 5,
  package_id: 7,
  status,
  appointment_time: '2026-10-01T09:00:00Z',
  address: '北京协和医院',
  total_amount: 29900,
  created_at: '2026-09-29T10:00:00Z',
  updated_at: '2026-09-29T10:00:00Z',
});

beforeEach(() => {
  setActivePinia(createPinia());
  const auth = useAuthStore();
  auth.user = {
    id: 1,
    phone: '13800138000',
    role: 'escort',
    active_role: 'escort',
    roles: ['escort'],
    real_name_verified: true,
  };
  vi.clearAllMocks();
  (globalThis as unknown as {
    uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
  }).uni.getCurrentPages = () => [{ options: { id: '101' } }];
});

describe('escort/order-detail · 加载', () => {
  it('parseQuery 提取 id 并调用 fetchDetail(101)', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'escort_confirmed'));
    mount(OrderDetail);
    await flushPromises();
    expect(apiOrders.getOrder).toHaveBeenCalledWith(101);
  });
});

describe('escort/order-detail · 渲染', () => {
  it('显示订单 id、状态、医院、地址、金额', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'escort_confirmed'));
    const w = mount(OrderDetail);
    await flushPromises();

    expect(w.find('[data-testid="escort-order-detail-header"]').exists()).toBe(true);
    expect(w.text()).toContain('订单 #101');
    expect(w.text()).toContain('待服务');
    expect(w.text()).toContain('北京协和医院');
    expect(w.text()).toContain('¥299.00');
  });
});

describe('escort/order-detail · 操作按钮可见性', () => {
  it('escort_confirmed 显示「开始服务」按钮', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'escort_confirmed'));
    const w = mount(OrderDetail);
    await flushPromises();
    expect(w.find('[data-testid="escort-order-detail-start"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-order-detail-finish"]').exists()).toBe(false);
  });

  it('in_service 显示「完成任务」按钮', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, 'in_service'));
    const w = mount(OrderDetail);
    await flushPromises();
    expect(w.find('[data-testid="escort-order-detail-finish"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-order-detail-start"]').exists()).toBe(false);
  });

  it.each(['completed', 'cancelled', 'pending_escort'] as OrderStatus[])(
    '%s 状态不显示任何操作按钮',
    async (status) => {
      vi.mocked(apiOrders.getOrder).mockResolvedValue(makeOrder(101, status));
      const w = mount(OrderDetail);
      await flushPromises();
      expect(w.find('[data-testid="escort-order-detail-actions"]').exists()).toBe(false);
    },
  );
});

describe('escort/order-detail · 操作流程', () => {
  it('点击「开始服务」调 confirmAccept(101) + 重新 fetchDetail', async () => {
    vi.mocked(apiOrders.getOrder)
      .mockResolvedValueOnce(makeOrder(101, 'escort_confirmed'))
      .mockResolvedValueOnce(makeOrder(101, 'in_service'));
    vi.mocked(apiOrders.confirmAccept).mockResolvedValue(makeOrder(101, 'in_service'));

    const w = mount(OrderDetail);
    await flushPromises();
    vi.clearAllMocks();

    await w.find('[data-testid="escort-order-detail-start"]').trigger('click');
    await flushPromises();

    expect(apiOrders.confirmAccept).toHaveBeenCalledWith(101);
    expect(apiOrders.getOrder).toHaveBeenCalledWith(101);
  });

  it('点击「完成任务」调 finishOrder(101) + 重新 fetchDetail', async () => {
    vi.mocked(apiOrders.getOrder)
      .mockResolvedValueOnce(makeOrder(101, 'in_service'))
      .mockResolvedValueOnce(makeOrder(101, 'completed'));
    vi.mocked(apiOrders.finishOrder).mockResolvedValue(makeOrder(101, 'completed'));

    const w = mount(OrderDetail);
    await flushPromises();
    vi.clearAllMocks();

    await w.find('[data-testid="escort-order-detail-finish"]').trigger('click');
    await flushPromises();

    expect(apiOrders.finishOrder).toHaveBeenCalledWith(101);
    expect(apiOrders.getOrder).toHaveBeenCalledWith(101);
  });
});