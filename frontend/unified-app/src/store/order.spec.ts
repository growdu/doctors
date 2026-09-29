/**
 * orderStore 单测（vitest + Pinia）。
 *
 * 验证目标：
 *   - 初始 state 默认值
 *   - fetchList 加载 + 写入 + 错误捕获
 *   - fetchDetail 写入 currentOrder
 *   - create / cancel / confirmAccept / rejectAccept / finish 都 upsert 到列表
 *   - clearError 重置 error
 *   - filterByStatusLocal 客户端过滤
 *
 * 策略：mock api/orders 模块提供可控的 resolved/rejected value；setActivePinia 初始化。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useOrderStore } from './order';
import type { Order } from '@/api/orders';

vi.mock('@/api/orders', () => ({
  createOrder: vi.fn(),
  listOrders: vi.fn(),
  getOrder: vi.fn(),
  cancelOrder: vi.fn(),
  confirmAccept: vi.fn(),
  rejectAccept: vi.fn(),
  finishOrder: vi.fn(),
}));

import * as apiOrders from '@/api/orders';

const fakeOrder: Order = {
  id: 1,
  patient_id: 100,
  escort_id: null,
  hospital_id: 5,
  package_id: 7,
  status: 'pending_escort',
  appointment_time: '2026-10-01T09:00:00Z',
  address: '北京协和医院',
  total_amount: 29900,
  created_at: '2026-09-29T10:00:00Z',
  updated_at: '2026-09-29T10:00:00Z',
};

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('orderStore · initial state', () => {
  it('starts with empty orders and currentOrder null', () => {
    const s = useOrderStore();
    expect(s.orders).toEqual([]);
    expect(s.currentOrder).toBeNull();
    expect(s.loading).toBe(false);
    expect(s.error).toBeNull();
    expect(s.total).toBe(0);
    expect(s.hasOrders).toBe(false);
  });
});

describe('orderStore · fetchList', () => {
  it('loads orders and sets total', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [fakeOrder], total: 5 });
    const s = useOrderStore();
    await s.fetchList();
    expect(s.orders).toEqual([fakeOrder]);
    expect(s.total).toBe(5);
    expect(s.loading).toBe(false);
  });

  it('captures error and clears loading', async () => {
    vi.mocked(apiOrders.listOrders).mockRejectedValue(new Error('网络异常'));
    const s = useOrderStore();
    await s.fetchList();
    expect(s.error).toBe('网络异常');
    expect(s.loading).toBe(false);
    expect(s.orders).toEqual([]);
  });

  it('clears error on success', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const s = useOrderStore();
    s.error = '旧错误';
    await s.fetchList();
    expect(s.error).toBeNull();
  });
});

describe('orderStore · fetchDetail', () => {
  it('sets currentOrder', async () => {
    vi.mocked(apiOrders.getOrder).mockResolvedValue(fakeOrder);
    const s = useOrderStore();
    const o = await s.fetchDetail(1);
    expect(o).toEqual(fakeOrder);
    expect(s.currentOrder).toEqual(fakeOrder);
  });

  it('rethrows on error', async () => {
    vi.mocked(apiOrders.getOrder).mockRejectedValue(new Error('订单不存在'));
    const s = useOrderStore();
    await expect(s.fetchDetail(999)).rejects.toThrow('订单不存在');
    expect(s.error).toBe('订单不存在');
    expect(s.currentOrder).toBeNull();
  });
});

describe('orderStore · create', () => {
  it('prepends created order to list', async () => {
    vi.mocked(apiOrders.createOrder).mockResolvedValue(fakeOrder);
    const s = useOrderStore();
    const o = await s.create({ hospital_id: 5, appointment_time: '2026-10-01T09:00:00Z', address: '北京协和医院' });
    expect(o).toEqual(fakeOrder);
    expect(s.orders[0]).toEqual(fakeOrder);
  });

  it('rethrows on error', async () => {
    vi.mocked(apiOrders.createOrder).mockRejectedValue(new Error('余额不足'));
    const s = useOrderStore();
    await expect(s.create({ hospital_id: 5, appointment_time: 'x', address: 'y' })).rejects.toThrow('余额不足');
    expect(s.error).toBe('余额不足');
  });
});

describe('orderStore · state transitions (upsert)', () => {
  beforeEach(() => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [fakeOrder], total: 1 });
  });

  it('cancel updates existing order in list and currentOrder', async () => {
    const cancelled: Order = { ...fakeOrder, status: 'cancelled' };
    vi.mocked(apiOrders.cancelOrder).mockResolvedValue(cancelled);
    const s = useOrderStore();
    await s.fetchList();
    s.currentOrder = fakeOrder;
    const o = await s.cancel(1, '改时间');
    expect(o.status).toBe('cancelled');
    expect(s.orders[0]?.status).toBe('cancelled');
    expect(s.currentOrder?.status).toBe('cancelled');
  });

  it('confirmAccept updates to escort_confirmed', async () => {
    const confirmed: Order = { ...fakeOrder, status: 'escort_confirmed', escort_id: 200 };
    vi.mocked(apiOrders.confirmAccept).mockResolvedValue(confirmed);
    const s = useOrderStore();
    await s.fetchList();
    await s.confirmAccept(1);
    expect(s.orders[0]?.status).toBe('escort_confirmed');
  });

  it('rejectAccept updates status', async () => {
    const rejected: Order = { ...fakeOrder, status: 'cancelled' };
    vi.mocked(apiOrders.rejectAccept).mockResolvedValue(rejected);
    const s = useOrderStore();
    await s.fetchList();
    await s.rejectAccept(1, '已接其他单');
    expect(s.orders[0]?.status).toBe('cancelled');
  });

  it('finish updates to completed', async () => {
    const finished: Order = { ...fakeOrder, status: 'completed' };
    vi.mocked(apiOrders.finishOrder).mockResolvedValue(finished);
    const s = useOrderStore();
    await s.fetchList();
    await s.finish(1);
    expect(s.orders[0]?.status).toBe('completed');
    expect(s.hasOrders).toBe(true);
  });
});

describe('orderStore · helpers', () => {
  it('clearError resets error', () => {
    const s = useOrderStore();
    s.error = 'xxx';
    s.clearError();
    expect(s.error).toBeNull();
  });

  it('filterByStatusLocal returns matching orders', async () => {
    const o1: Order = { ...fakeOrder, id: 1, status: 'pending_escort' };
    const o2: Order = { ...fakeOrder, id: 2, status: 'completed' };
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [o1, o2], total: 2 });
    const s = useOrderStore();
    await s.fetchList();
    expect(s.filterByStatusLocal('pending_escort')).toEqual([o1]);
    expect(s.filterByStatusLocal('completed')).toEqual([o2]);
    expect(s.filterByStatusLocal('cancelled')).toEqual([]);
  });
});