/**
 * admin/orders/list.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - mount 时通过 store.fetchList({ role: 'patient' }) 拉订单
 *   - 6 个 status tab 渲染 + 点击触发重新 fetchList
 *   - 订单卡渲染（含金额格式化、医院 ID、状态徽章）
 *   - 点击订单卡跳转 detail?id=xxx
 *   - 空 / 加载中 / 错误 三态渲染
 *
 * 策略：mock api/orders 模块，注入到真实 orderStore。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import OrdersList from './list.vue';

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
import type { Order } from '@/api/orders';

const makeOrder = (id: number, status: Order['status']): Order => ({
  id,
  patient_id: 100,
  escort_id: status === 'escort_confirmed' ? 200 : null,
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
  vi.clearAllMocks();
});

describe('admin/orders/list · 加载', () => {
  it('mount 时调用 store.fetchList({ role: patient })', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    mount(OrdersList);
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledWith({ role: 'patient' });
  });
});

describe('admin/orders/list · 状态 tab', () => {
  it('渲染 6 个 tab', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(OrdersList);
    await flushPromises();

    for (const s of ['all', 'pending_escort', 'escort_confirmed', 'in_service', 'completed', 'cancelled']) {
      expect(w.find(`[data-testid="orders-list-tab-${s}"]`).exists()).toBe(true);
    }
    expect(w.text()).toContain('全部');
    expect(w.text()).toContain('待接单');
    expect(w.text()).toContain('已确认');
    expect(w.text()).toContain('服务中');
    expect(w.text()).toContain('已完成');
    expect(w.text()).toContain('已取消');
  });

  it('点击 tab 重新触发 fetchList', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(OrdersList);
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledTimes(1);

    await w.find('[data-testid="orders-list-tab-pending_escort"]').trigger('click');
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledTimes(2);

    await w.find('[data-testid="orders-list-tab-completed"]').trigger('click');
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledTimes(3);
  });

  it('默认 active tab 是 all', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(OrdersList);
    await flushPromises();
    expect(w.find('[data-testid="orders-list-tab-all"]').classes()).toContain(
      'admin-orders-list__tab--active',
    );
  });
});

describe('admin/orders/list · 订单卡渲染', () => {
  it('渲染订单 id、状态、医院、地址、金额（分→元）', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [
        makeOrder(101, 'pending_escort'),
        makeOrder(102, 'in_service'),
        makeOrder(103, 'completed'),
      ],
      total: 3,
    });
    const w = mount(OrdersList);
    await flushPromises();

    expect(w.find('[data-testid="orders-list-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="orders-list-card-102"]').exists()).toBe(true);
    expect(w.find('[data-testid="orders-list-card-103"]').exists()).toBe(true);

    expect(w.text()).toContain('订单 #101');
    expect(w.text()).toContain('订单 #102');
    expect(w.text()).toContain('医院 #5');
    expect(w.text()).toContain('北京协和医院');
    // 29900 分 → ¥299.00
    expect(w.text()).toContain('¥299.00');

    // 状态文本
    expect(w.text()).toContain('待陪诊师接单');
    expect(w.text()).toContain('服务中');
    expect(w.text()).toContain('已完成');
  });

  it('点击订单卡跳转 detail?id=', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101, 'pending_escort')],
      total: 1,
    });
    const w = mount(OrdersList);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="orders-list-card-101"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/orders/detail?id=101' });
  });
});

describe('admin/orders/list · 空 / 加载 / 错误 三态', () => {
  it('加载中显示 UiLoading', async () => {
    // 永不 resolve 让 loading 持续
    vi.mocked(apiOrders.listOrders).mockReturnValue(new Promise(() => {}));
    const w = mount(OrdersList);
    await flushPromises();

    expect(w.find('[data-testid="orders-list-loading"]').exists()).toBe(true);
    expect(w.find('[data-testid="orders-list-empty"]').exists()).toBe(false);
  });

  it('空数据显示 UiEmpty「暂无订单」', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(OrdersList);
    await flushPromises();

    expect(w.find('[data-testid="orders-list-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无订单');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiOrders.listOrders).mockRejectedValue(new Error('网络超时'));
    const w = mount(OrdersList);
    await flushPromises();

    expect(w.find('[data-testid="orders-list-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
    expect(w.text()).toContain('点击重试');
  });
});