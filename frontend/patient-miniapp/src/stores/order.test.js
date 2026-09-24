// src/store/order.test.js
//
// order store 单测 —— 订单状态机 + 列表 / 详情 / 轮询 / 选人模式（v1.1）
//
// 测试策略：
//   - jest.doMock + jest.resetModules 隔离每个 case 的 api mock
//   - 动态 import('@/stores/order.js') 拿到 useOrderStore
//   - 轮询 case 用 jest fake timers + advanceTimersByTimeAsync（自动 flush microtask）
//
// 覆盖：
//   1. 初始：current / list / candidates 都为空
//   2. loadOrder：写入 current
//   3. loadList：写入 list（兼容直返数组与 { items }）
//   4. loadCandidates：写入 candidates + generated_at（v1.1 选人模式入口）
//   5. selectEscortBy：调 api + 缓存 selection + 立即 loadOrder（v1.1 核心）
//   6. selectEscortBy 缺参数 → 抛错
//   7. startPolling / stopPolling：节拍 + stop 后不再触发
//   8. cancel：调 api + 刷新
//   9. 状态机常量：selectingEscort / escortPendingAcceptance 暴露
//  10. isStatus helper

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('order store - 基础 CRUD', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('初始：current / list / candidates / selection 都为空', async () => {
    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    expect(s.current).toBeNull();
    expect(s.list).toEqual([]);
    expect(s.candidates).toEqual([]);
    expect(s.candidatesGeneratedAt).toBe('');
    expect(s.selection).toBeNull();
  });

  it('loadOrder 设置 current', async () => {
    const getOrder = jest.fn(async () => ({ id: 7, status: 'paid', amount: 100 }));
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    await s.loadOrder(7);

    expect(s.current).toEqual({ id: 7, status: 'paid', amount: 100 });
    expect(getOrder).toHaveBeenCalledWith(7);
  });

  it('loadList 兼容两种返回形态（{items} / 直返数组）', async () => {
    const listOrders = jest.fn(async () => ({ items: [{ id: 1 }, { id: 2 }] }));
    jest.doMock('@/api/order.js', () => ({ listOrders }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    await s.loadList({ status: 'paid' });
    expect(s.list).toHaveLength(2);
    expect(listOrders).toHaveBeenCalledWith({ status: 'paid' });
  });

  it('loadList 直返数组也能写入', async () => {
    const listOrders = jest.fn(async () => [{ id: 9 }]);
    jest.doMock('@/api/order.js', () => ({ listOrders }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    await s.loadList();
    expect(s.list).toEqual([{ id: 9 }]);
  });
});

describe('order store - 选人模式（v1.1）', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('loadCandidates 写入 candidates + generated_at', async () => {
    const getCandidates = jest.fn(async () => ({
      items: [
        { escort_id: 11, name: '张三', rating: 4.9 },
        { escort_id: 12, name: '李四', rating: 4.7 },
      ],
      generated_at: '2026-09-24T15:30:00+08:00',
    }));
    jest.doMock('@/api/order.js', () => ({ getCandidates }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    const r = await s.loadCandidates(7);

    expect(s.candidates).toHaveLength(2);
    expect(s.candidates[0].name).toBe('张三');
    expect(s.candidatesGeneratedAt).toBe('2026-09-24T15:30:00+08:00');
    expect(r.generated_at).toBe('2026-09-24T15:30:00+08:00');
    expect(getCandidates).toHaveBeenCalledWith(7);
  });

  it('loadCandidates 兼容 data 字段（{data: [...]})', async () => {
    const getCandidates = jest.fn(async () => ({
      data: [{ escort_id: 99, name: '王五' }],
    }));
    jest.doMock('@/api/order.js', () => ({ getCandidates }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    await s.loadCandidates(7);
    expect(s.candidates).toEqual([{ escort_id: 99, name: '王五' }]);
    expect(s.candidatesGeneratedAt).toBe('');
  });

  it('selectEscortBy 调 api + 缓存 selection + 立即 loadOrder', async () => {
    const selectEscort = jest.fn(async () => ({
      order_id: 7,
      selected_escort_id: 11,
      escort_pending_expire_at: '2026-09-24T15:30:30+08:00',
      status: 'escortPendingAcceptance',
    }));
    const getOrder = jest.fn(async () => ({
      id: 7,
      status: 'escortPendingAcceptance',
      selected_escort_id: 11,
    }));
    jest.doMock('@/api/order.js', () => ({ selectEscort, getOrder }));

    const { useOrderStore, ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE } = await import('@/stores/order.js');
    const s = useOrderStore();
    const r = await s.selectEscortBy(7, 11);

    expect(selectEscort).toHaveBeenCalledWith(7, { escort_id: 11 });
    expect(getOrder).toHaveBeenCalledWith(7);
    expect(s.selection.escort_pending_expire_at).toBe('2026-09-24T15:30:30+08:00');
    expect(s.current.status).toBe(ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE);
    expect(r.selected_escort_id).toBe(11);
  });

  it('selectEscortBy 缺参数 → 抛错且不发请求', async () => {
    const selectEscort = jest.fn();
    jest.doMock('@/api/order.js', () => ({ selectEscort }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    await expect(s.selectEscortBy(0, 11)).rejects.toThrow(/orderId & escortId required/);
    await expect(s.selectEscortBy(7, 0)).rejects.toThrow(/orderId & escortId required/);
    expect(selectEscort).not.toHaveBeenCalled();
  });

  it('状态机常量：selectingEscort + escortPendingAcceptance 已暴露', async () => {
    const {
      ORDER_STATUS_SELECTING_ESCORT,
      ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE,
      ORDER_STATUS_LABEL,
      ORDER_PROGRESS_STEPS,
    } = await import('@/stores/order.js');
    expect(ORDER_STATUS_SELECTING_ESCORT).toBe('selectingEscort');
    expect(ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE).toBe('escortPendingAcceptance');
    expect(ORDER_STATUS_LABEL[ORDER_STATUS_SELECTING_ESCORT]).toBe('选择陪诊师');
    expect(ORDER_STATUS_LABEL[ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE]).toBe('陪诊师确认中');
    // 进度条节点顺序：选人模式两节点在中间
    const idx1 = ORDER_PROGRESS_STEPS.indexOf(ORDER_STATUS_SELECTING_ESCORT);
    const idx2 = ORDER_PROGRESS_STEPS.indexOf(ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE);
    expect(idx1).toBeGreaterThanOrEqual(0);
    expect(idx2).toBeGreaterThan(idx1);
  });
});

describe('order store - 轮询', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('startPolling 每 intervalMs 拉一次 loadOrder', async () => {
    jest.useFakeTimers();
    const getOrder = jest.fn(async () => ({ id: 7, status: 'inService' }));
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    s.startPolling(7, 3000);

    // 立刻跑的第一次 tick（含 microtask）
    await jest.advanceTimersByTimeAsync(0);
    expect(getOrder).toHaveBeenCalledTimes(1);

    await jest.advanceTimersByTimeAsync(3000);
    expect(getOrder).toHaveBeenCalledTimes(2);

    await jest.advanceTimersByTimeAsync(3000);
    expect(getOrder).toHaveBeenCalledTimes(3);

    s.stopPolling();
    jest.useRealTimers();
  });

  it('stopPolling 后定时器取消', async () => {
    jest.useFakeTimers();
    const getOrder = jest.fn(async () => ({ id: 7, status: 'inService' }));
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    s.startPolling(7, 1000);
    await jest.advanceTimersByTimeAsync(0);
    expect(getOrder).toHaveBeenCalledTimes(1);

    s.stopPolling();
    await jest.advanceTimersByTimeAsync(10000);
    expect(getOrder).toHaveBeenCalledTimes(1);

    jest.useRealTimers();
  });

  it('startPolling 网络抖动：单次失败不影响后续 tick', async () => {
    jest.useFakeTimers();
    let count = 0;
    const getOrder = jest.fn(async () => {
      count += 1;
      if (count === 1) throw new Error('network');
      return { id: 7, status: 'inService' };
    });
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    s.startPolling(7, 1000);

    await jest.advanceTimersByTimeAsync(0);
    expect(count).toBe(1);

    await jest.advanceTimersByTimeAsync(1000);
    expect(count).toBe(2);

    s.stopPolling();
    jest.useRealTimers();
  });

  it('startPolling 切换到新 orderId → 老定时器被抢占', async () => {
    jest.useFakeTimers();
    const getOrder = jest.fn(async (id) => ({ id, status: 'inService' }));
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    s.startPolling(7, 1000);
    // 切换到新 orderId —— stopPolling 会被 startPolling 内部先调
    s.startPolling(8, 1000);
    await jest.advanceTimersByTimeAsync(0);
    // 应该只剩 8 的一次（不是 7 + 8 = 2）
    expect(getOrder).toHaveBeenCalledTimes(1);
    expect(getOrder).toHaveBeenCalledWith(8);

    s.stopPolling();
    jest.useRealTimers();
  });

  it('startPolling(orderId=0) 立即返回（不创建定时器）', async () => {
    jest.useFakeTimers();
    const getOrder = jest.fn(async () => ({ id: 0 }));
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    s.startPolling(0);
    await jest.advanceTimersByTimeAsync(5000);
    expect(getOrder).not.toHaveBeenCalled();

    jest.useRealTimers();
  });
});

describe('order store - cancel / isStatus', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('cancel 调 api + 立即 loadOrder 刷新 current', async () => {
    const cancelOrder = jest.fn(async () => ({ ok: true }));
    const getOrder = jest.fn(async () => ({ id: 7, status: 'cancelled' }));
    jest.doMock('@/api/order.js', () => ({ cancelOrder, getOrder }));

    const { useOrderStore, ORDER_STATUS_CANCELLED } = await import('@/stores/order.js');
    const s = useOrderStore();
    await s.cancel(7, '改主意了');
    expect(cancelOrder).toHaveBeenCalledWith(7, '改主意了');
    expect(getOrder).toHaveBeenCalledWith(7);
    expect(s.current.status).toBe(ORDER_STATUS_CANCELLED);
  });

  it('isStatus helper 正确判断当前订单状态', async () => {
    const getOrder = jest.fn(async () => ({ id: 7, status: 'escortPendingAcceptance' }));
    jest.doMock('@/api/order.js', () => ({ getOrder }));

    const { useOrderStore, ORDER_STATUS_SELECTING_ESCORT, ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE } = await import('@/stores/order.js');
    const s = useOrderStore();
    await s.loadOrder(7);
    expect(s.isStatus(ORDER_STATUS_ESCORT_PENDING_ACCEPTANCE)).toBe(true);
    expect(s.isStatus(ORDER_STATUS_SELECTING_ESCORT)).toBe(false);
  });

  it('isStatus 在 current=null 时返回 false', async () => {
    const { useOrderStore } = await import('@/stores/order.js');
    const s = useOrderStore();
    expect(s.isStatus('any')).toBe(false);
  });
});