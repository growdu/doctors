// src/api/order.test.js
//
// order api 单测 —— 验证 getOrder / selectEscort 的 URL + method + body + 响应透传 + 缺参兜底。
//
// 测试策略：同 candidates.test.js —— jest.doMock('../../utils/request.js') 替换 fetch。
// 备注：brief 说 `jest.doMock('@/utils/request.js')`，但 `@/` 在 jest.config.js 映射到
// `src/utils/`（≠ 根 utils/），而 api 模块实际 import 的是根 utils/request.js，
// 故 mock 路径必须与 import 路径一致 —— 这里走 `'../../utils/request.js'`。

import { jest } from '@jest/globals';

describe('api/order - getOrder', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 request GET /orders/:id 并透传响应', async () => {
    const order = {
      id: 7,
      status: 'selectingEscort',
      amount: 38800,
      hospitalId: 101,
      selectedEscortId: null,
      escortPendingExpireAt: '',
      escortRejectReason: '',
    };
    const requestMock = jest.fn(async () => order);
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getOrder } = await import('@/api/order.js');
    const r = await getOrder(7);

    expect(requestMock).toHaveBeenCalledTimes(1);
    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/7',
        method: 'GET',
      }),
    );
    expect(r).toBe(order);
    expect(r.status).toBe('selectingEscort');
  });

  it('Order 字段包含 v1.1 选人模式相关字段', async () => {
    const order = {
      id: 7,
      status: 'escortPendingAcceptance',
      selectedEscortId: 11,
      escortPendingExpireAt: '2026-09-24T15:30:30+08:00',
      escortRejectReason: '',
    };
    const requestMock = jest.fn(async () => order);
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getOrder } = await import('@/api/order.js');
    const r = await getOrder(7);

    expect(r.selectedEscortId).toBe(11);
    expect(r.escortPendingExpireAt).toBe('2026-09-24T15:30:30+08:00');
    expect(r.escortRejectReason).toBe('');
  });

  it('Order 在 escortReject 场景下携带 escortRejectReason', async () => {
    const order = {
      id: 7,
      status: 'selectingEscort',
      selectedEscortId: 11,
      escortPendingExpireAt: '',
      escortRejectReason: '当前满单，请重新选择',
    };
    const requestMock = jest.fn(async () => order);
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getOrder } = await import('@/api/order.js');
    const r = await getOrder(7);

    expect(r.escortRejectReason).toBe('当前满单，请重新选择');
  });

  it('getOrder 支持字符串 orderId', async () => {
    const requestMock = jest.fn(async () => ({ id: 'uuid', status: 'paid' }));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getOrder } = await import('@/api/order.js');
    await getOrder('order-uuid');

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/orders/order-uuid' }),
    );
  });

  it('getOrder 缺 orderId → 同步 reject 且不发请求', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getOrder } = await import('@/api/order.js');
    await expect(getOrder(undefined)).rejects.toThrow(/orderId is required/);
    await expect(getOrder(null)).rejects.toThrow(/orderId is required/);
    await expect(getOrder('')).rejects.toThrow(/orderId is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('getOrder 失败时透传 request 抛出的 ApiError', async () => {
    const requestMock = jest.fn(async () => {
      throw new Error('escort busy');
    });
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getOrder } = await import('@/api/order.js');
    await expect(getOrder(7)).rejects.toThrow(/escort busy/);
  });
});

describe('api/order - selectEscort', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 request POST /orders/:id/select-escort + body.escort_id 并透传响应', async () => {
    // 实际 backend 响应按 Go snake_case（与 stores/order.test.js mock 对齐）
    const resp = {
      order_id: 7,
      selected_escort_id: 11,
      escort_pending_expire_at: '2026-09-24T15:30:30+08:00',
      status: 'escortPendingAcceptance',
    };
    const requestMock = jest.fn(async () => resp);
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { selectEscort } = await import('@/api/order.js');
    const r = await selectEscort(7, 11);

    expect(requestMock).toHaveBeenCalledTimes(1);
    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/7/select-escort',
        method: 'POST',
        data: { escort_id: 11 },
      }),
    );
    expect(r).toBe(resp);
  });

  it('selectEscort 兼容对象入参 { escort_id }（已落地 store 调用形式）', async () => {
    const requestMock = jest.fn(async () => ({}));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { selectEscort } = await import('@/api/order.js');
    await selectEscort(7, { escort_id: 11 });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/7/select-escort',
        method: 'POST',
        data: { escort_id: 11 },
      }),
    );
  });

  it('selectEscort 支持字符串参数', async () => {
    const requestMock = jest.fn(async () => ({}));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { selectEscort } = await import('@/api/order.js');
    await selectEscort('order-uuid', 'escort-uuid');

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/order-uuid/select-escort',
        data: { escort_id: 'escort-uuid' },
      }),
    );
  });

  it('selectEscort 缺 orderId 或 escortId → 同步 reject 且不发请求', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { selectEscort } = await import('@/api/order.js');

    await expect(selectEscort(0, 11)).rejects.toThrow(/orderId is required/);
    await expect(selectEscort(7, undefined)).rejects.toThrow(/escortId is required/);
    await expect(selectEscort(7, null)).rejects.toThrow(/escortId is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('selectEscort 失败时透传 request 抛出的 ApiError', async () => {
    const requestMock = jest.fn(async () => {
      throw new Error('escort busy');
    });
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { selectEscort } = await import('@/api/order.js');
    await expect(selectEscort(7, 11)).rejects.toThrow(/escort busy/);
  });
});

describe('api/order - 兼容 store import（listOrders / cancelOrder / getCandidates）', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('listOrders 调 GET /orders 并透传 query', async () => {
    const requestMock = jest.fn(async () => ({ items: [{ id: 1 }] }));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { listOrders } = await import('@/api/order.js');
    const r = await listOrders({ status: 'paid', page: 2 });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders',
        method: 'GET',
        query: { status: 'paid', page: 2 },
      }),
    );
    expect(r.items).toHaveLength(1);
  });

  it('listOrders 不传 query → query 默认 {}', async () => {
    const requestMock = jest.fn(async () => []);
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { listOrders } = await import('@/api/order.js');
    await listOrders();

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders',
        method: 'GET',
        query: {},
      }),
    );
  });

  it('cancelOrder 调 POST /orders/:id/cancel + body.reason', async () => {
    const requestMock = jest.fn(async () => ({ ok: true }));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { cancelOrder } = await import('@/api/order.js');
    await cancelOrder(7, '改主意了');

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/7/cancel',
        method: 'POST',
        data: { reason: '改主意了' },
      }),
    );
  });

  it('getCandidates 经 candidates.js re-export 后行为一致', async () => {
    const requestMock = jest.fn(async () => ({ items: [], generated_at: 'now' }));
    jest.doMock('../../utils/request.js', () => ({
      request: requestMock,
    }));

    const { getCandidates } = await import('@/api/order.js');
    const r = await getCandidates(7);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/orders/7/candidates',
        method: 'GET',
      }),
    );
    expect(r.generated_at).toBe('now');
  });
});

describe('api/index.js', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('默认导出 api 命名空间（含 candidates + order 子模块）', async () => {
    jest.doMock('../../utils/request.js', () => ({
      request: jest.fn(async () => ({})),
    }));

    const mod = await import('@/api/index.js');
    const api = mod.default;

    expect(api).toBeDefined();
    expect(typeof api.candidates.getCandidates).toBe('function');
    expect(typeof api.order.getOrder).toBe('function');
    expect(typeof api.order.selectEscort).toBe('function');
    expect(typeof api.order.listOrders).toBe('function');
    expect(typeof api.order.cancelOrder).toBe('function');
  });

  it('named exports 也都可用', async () => {
    jest.doMock('../../utils/request.js', () => ({
      request: jest.fn(async () => ({})),
    }));

    const mod = await import('@/api/index.js');
    expect(typeof mod.getCandidates).toBe('function');
    expect(typeof mod.getOrder).toBe('function');
    expect(typeof mod.selectEscort).toBe('function');
    expect(typeof mod.listOrders).toBe('function');
    expect(typeof mod.cancelOrder).toBe('function');
  });
});
