/**
 * api/orders 单测：mock client 模块验证端点函数调用正确。
 *
 * 验证目标：
 *   - 每个端点调用 request() 时传正确的 url / method / data / baseURL
 *   - listOrders query string 正确构造
 *   - 返回值透传（不破坏 data）
 *
 * 策略：vi.mock('./client') 替换 request 为 spy，避免触发 auth store + 真实 fetch。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  createOrder,
  selectEscort,
  cancelOrder,
  confirmAccept,
  rejectAccept,
  finishOrder,
  listOrders,
  getOrder,
} from './orders';
import type { Order } from './orders';

// ── mock client.request ─────────────────────────────────────────────
const requestSpy = vi.fn();

vi.mock('./client', () => ({
  request: (...args: unknown[]) => requestSpy(...args),
}));

const fakeOrder: Order = {
  id: 1,
  patient_id: 100,
  escort_id: 200,
  hospital_id: 5,
  package_id: 7,
  status: 'escort_confirmed',
  appointment_time: '2026-10-01T09:00:00Z',
  address: '北京协和医院',
  total_amount: 29900,
  created_at: '2026-09-29T10:00:00Z',
  updated_at: '2026-09-29T10:00:00Z',
};

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue(fakeOrder);
});

describe('api/orders · patient-only endpoints', () => {
  it('createOrder POSTs to /orders', async () => {
    await createOrder({
      hospital_id: 5,
      appointment_time: '2026-10-01T09:00:00Z',
      address: '北京协和医院',
    });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/orders',
      method: 'POST',
      data: {
        hospital_id: 5,
        appointment_time: '2026-10-01T09:00:00Z',
        address: '北京协和医院',
      },
      baseURL: 'http://127.0.0.1:8082',
    });
  });

  it('selectEscort POSTs to /orders/:id/select-escort', async () => {
    await selectEscort(1, 200);
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/orders/1/select-escort',
      method: 'POST',
      data: { escort_id: 200 },
      baseURL: 'http://127.0.0.1:8082',
    });
  });

  it('cancelOrder POSTs with optional reason', async () => {
    await cancelOrder(1, '改时间');
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/orders/1/cancel',
      method: 'POST',
      data: { reason: '改时间' },
      baseURL: 'http://127.0.0.1:8082',
    });
  });

  it('cancelOrder omits reason when empty', async () => {
    await cancelOrder(1);
    expect(requestSpy.mock.calls[0]![0].data).toEqual({});
  });
});

describe('api/orders · escort-only endpoints', () => {
  it('confirmAccept POSTs to /orders/:id/confirm-accept', async () => {
    await confirmAccept(1);
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/orders/1/confirm-accept',
      method: 'POST',
      data: undefined,
      baseURL: 'http://127.0.0.1:8082',
    });
  });

  it('rejectAccept POSTs with optional reason', async () => {
    await rejectAccept(1, '已接其他单');
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ reason: '已接其他单' });
  });

  it('finishOrder POSTs to /orders/:id/finish', async () => {
    await finishOrder(1);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/orders/1/finish');
    expect(requestSpy.mock.calls[0]![0].method).toBe('POST');
  });
});

describe('api/orders · common endpoints', () => {
  it('getOrder GETs /orders/:id', async () => {
    await getOrder(1);
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/orders/1',
      baseURL: 'http://127.0.0.1:8082',
    });
  });

  it('listOrders with empty query returns no query string', async () => {
    await listOrders();
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/orders');
  });

  it('listOrders builds query string from filters', async () => {
    await listOrders({ status: 'pending_escort', role: 'patient', page: 2, page_size: 20 });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('/api/v1/orders?');
    expect(url).toContain('status=pending_escort');
    expect(url).toContain('role=patient');
    expect(url).toContain('page=2');
    expect(url).toContain('page_size=20');
  });

  it('listOrders omits undefined filters', async () => {
    await listOrders({ status: 'completed' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('status=completed');
    expect(url).not.toContain('role=');
    expect(url).not.toContain('page=');
  });
});

describe('api/orders · return value', () => {
  it('passes through request response', async () => {
    const result = await getOrder(1);
    expect(result).toEqual(fakeOrder);
    expect(result.status).toBe('escort_confirmed');
  });
});