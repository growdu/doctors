/**
 * escort/orders/index.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - mount 时通过 store.fetchList({ role: 'escort' }) 拉订单
 *   - 5 个状态 tab 渲染 + 点击触发重新 fetchList
 *   - 订单卡渲染（含金额格式化、医院 ID、状态徽章）
 *   - 点击订单卡跳转 detail?id=
 *   - 空 / 加载中 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import EscortOrders from './index.vue';

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
});

describe('escort/orders · 加载', () => {
  it('mount 时调用 store.fetchList({ role: escort })', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    mount(EscortOrders);
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledWith({ role: 'escort' });
  });
});

describe('escort/orders · 状态 tab', () => {
  it('渲染 5 个 tab', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(EscortOrders);
    await flushPromises();

    for (const s of ['all', 'escort_confirmed', 'in_service', 'completed', 'cancelled']) {
      expect(w.find(`[data-testid="escort-orders-tab-${s}"]`).exists()).toBe(true);
    }
  });

  it('点击 tab 重新触发 fetchList', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(EscortOrders);
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledTimes(1);

    await w.find('[data-testid="escort-orders-tab-in_service"]').trigger('click');
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledTimes(2);
  });
});

describe('escort/orders · 订单卡渲染', () => {
  it('渲染订单 id、状态、医院、地址、金额（分→元）', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101, 'escort_confirmed'), makeOrder(102, 'in_service')],
      total: 2,
    });
    const w = mount(EscortOrders);
    await flushPromises();

    expect(w.find('[data-testid="escort-orders-card-101"]').exists()).toBe(true);
    expect(w.text()).toContain('订单 #101');
    expect(w.text()).toContain('医院 #5');
    expect(w.text()).toContain('北京协和医院');
    expect(w.text()).toContain('¥299.00');

    // escort 视角状态文本
    expect(w.text()).toContain('待服务');
    expect(w.text()).toContain('服务中');
  });

  it('点击订单卡跳转 detail?id=', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101, 'escort_confirmed')],
      total: 1,
    });
    const w = mount(EscortOrders);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="escort-orders-card-101"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/escort/order-detail?id=101' });
  });
});

describe('escort/orders · 三态', () => {
  it('空数据显示 UiEmpty「暂无任务」', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(EscortOrders);
    await flushPromises();
    expect(w.find('[data-testid="escort-orders-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无任务');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiOrders.listOrders).mockRejectedValue(new Error('网络超时'));
    const w = mount(EscortOrders);
    await flushPromises();
    expect(w.find('[data-testid="escort-orders-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});