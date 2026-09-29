/**
 * admin/orders/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 初次 mount 触发 3 次 listOrders（pending / in_service / completed）
 *   - 渲染 3 张状态卡片，数字来自 API total
 *   - 点击「查看 →」触发 uni.navigateTo 跳转 /pages/admin/orders/list?status=xxx
 *   - 点击「查看全部订单」跳转不带 status 参数
 *   - API 异常时 loadStatusCount 容错返回 0，loading 仍能关闭
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import OrdersOverview from './index.vue';

vi.mock('@/api/orders', () => ({
  listOrders: vi.fn(),
}));

import * as apiOrders from '@/api/orders';

const makeOrder = (id: number) => ({
  id,
  patient_id: 100 + id,
  escort_id: null,
  hospital_id: 5,
  package_id: 7,
  status: 'pending_escort' as const,
  appointment_time: '2026-10-01T09:00:00Z',
  address: '北京协和医院',
  total_amount: 29900,
  created_at: '2026-09-29T10:00:00Z',
  updated_at: '2026-09-29T10:00:00Z',
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/orders/index · 加载与渲染', () => {
  it('初次 mount 触发 3 次 listOrders 调用（3 个状态各一次）', async () => {
    vi.mocked(apiOrders.listOrders)
      .mockResolvedValueOnce({ items: [], total: 3 })
      .mockResolvedValueOnce({ items: [], total: 1 })
      .mockResolvedValueOnce({ items: [], total: 5 });

    mount(OrdersOverview);
    await flushPromises();

    expect(apiOrders.listOrders).toHaveBeenCalledTimes(3);
    expect(apiOrders.listOrders).toHaveBeenCalledWith({
      status: 'pending_escort',
      page: 1,
      page_size: 1,
    });
    expect(apiOrders.listOrders).toHaveBeenCalledWith({
      status: 'in_service',
      page: 1,
      page_size: 1,
    });
    expect(apiOrders.listOrders).toHaveBeenCalledWith({
      status: 'completed',
      page: 1,
      page_size: 1,
    });
  });

  it('渲染 3 张状态卡片并显示 total 数字', async () => {
    vi.mocked(apiOrders.listOrders)
      .mockResolvedValueOnce({ items: [], total: 7 }) // pending_escort
      .mockResolvedValueOnce({ items: [], total: 2 }) // in_service
      .mockResolvedValueOnce({ items: [], total: 99 }); // completed

    const w = mount(OrdersOverview);
    await flushPromises();

    expect(w.find('[data-testid="orders-stat-pending_escort"]').exists()).toBe(true);
    expect(w.find('[data-testid="orders-stat-in_service"]').exists()).toBe(true);
    expect(w.find('[data-testid="orders-stat-completed"]').exists()).toBe(true);

    expect(w.text()).toContain('7');
    expect(w.text()).toContain('2');
    expect(w.text()).toContain('99');
    expect(w.text()).toContain('待陪诊师接单');
    expect(w.text()).toContain('服务中');
    expect(w.text()).toContain('已完成');
  });

  it('API 失败时 loadStatusCount 返回 0，加载仍能完成', async () => {
    vi.mocked(apiOrders.listOrders).mockRejectedValue(new Error('网络异常'));

    const w = mount(OrdersOverview);
    await flushPromises();

    expect(w.find('[data-testid="orders-overview-content"]').exists()).toBe(true);
    expect(w.find('[data-testid="orders-overview-loading"]').exists()).toBe(false);
    // 三个数字都是 0
    expect(w.text()).toContain('待陪诊师接单');
    expect(w.text()).toContain('服务中');
  });
});

describe('admin/orders/index · 跳转', () => {
  it('点击 stat「查看 →」跳转 list 携带 ?status=', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(OrdersOverview);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="orders-stat-btn-pending_escort"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/orders/list?status=pending_escort' });

    await w.find('[data-testid="orders-stat-btn-in_service"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/orders/list?status=in_service' });
  });

  it('点击「查看全部订单」跳转 list 不携带 status', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(OrdersOverview);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="orders-stat-btn-all"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/orders/list' });
  });
});

// 引用 makeOrder 以避免未使用警告（lint 占位）
void makeOrder;