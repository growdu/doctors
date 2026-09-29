/**
 * api/admin 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  listUsers,
  forceCancelOrder,
  listPendingEscorts,
  approveEscort,
  rejectEscort,
  listRefunds,
  approveRefund,
  rejectRefund,
  listWorkOrders,
  createWorkOrder,
  listBillings,
  getReportOverview,
} from './admin';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({ items: [] });
});

describe('api/admin · users', () => {
  it('listUsers with role filter', async () => {
    await listUsers({ role: 'escort' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('role=escort');
  });
});

describe('api/admin · orders', () => {
  it('forceCancelOrder POSTs reason', async () => {
    await forceCancelOrder(42, '违规');
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/admin/orders/42/force-cancel',
      method: 'POST',
      data: { reason: '违规' },
      baseURL: 'http://127.0.0.1:8091',
    });
  });
});

describe('api/admin · escort audit', () => {
  it('listPendingEscorts GETs pending-audit', async () => {
    await listPendingEscorts();
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/admin/escorts/pending-audit');
  });

  it('approveEscort POSTs approve', async () => {
    await approveEscort(99);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/admin/escorts/99/approve');
  });

  it('rejectEscort POSTs reason', async () => {
    await rejectEscort(99, '材料不全');
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ reason: '材料不全' });
  });
});

describe('api/admin · refunds', () => {
  it('listRefunds with pending filter', async () => {
    await listRefunds({ status: 'pending' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('status=pending');
  });

  it('approveRefund POSTs approve', async () => {
    await approveRefund(99);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/admin/refunds/99/approve');
  });

  it('rejectRefund POSTs reason', async () => {
    await rejectRefund(99, '不在退款范围');
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ reason: '不在退款范围' });
  });
});

describe('api/admin · work orders', () => {
  it('createWorkOrder POSTs body', async () => {
    await createWorkOrder({ type: 'complaint', title: '客户投诉', content: '详情' });
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/admin/work-orders');
    expect(requestSpy.mock.calls[0]![0].method).toBe('POST');
  });

  it('listWorkOrders with status filter', async () => {
    await listWorkOrders({ status: 'open' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('status=open');
  });
});

describe('api/admin · billings + reports', () => {
  it('listBillings with type filter', async () => {
    await listBillings({ type: 'income' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('type=income');
  });

  it('getReportOverview GETs reports/overview', async () => {
    await getReportOverview();
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/admin/reports/overview',
      baseURL: 'http://127.0.0.1:8091',
    });
  });
});