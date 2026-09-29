/**
 * api/payment 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { createPayment, getPayment, completePayment, refundPayment } from './payment';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({});
});

describe('api/payment', () => {
  it('createPayment POSTs with default channel', async () => {
    await createPayment({ order_id: 42 });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/payments',
      method: 'POST',
      data: { order_id: 42 },
      baseURL: 'http://127.0.0.1:8085',
    });
  });

  it('createPayment with custom channel', async () => {
    await createPayment({ order_id: 42, channel: 'wechat' });
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ order_id: 42, channel: 'wechat' });
  });

  it('getPayment GETs /payments/:id', async () => {
    await getPayment(99);
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/payments/99',
      baseURL: 'http://127.0.0.1:8085',
    });
  });

  it('completePayment POSTs to /payments/:id/complete', async () => {
    await completePayment(99);
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/payments/99/complete',
      method: 'POST',
      data: undefined,
      baseURL: 'http://127.0.0.1:8085',
    });
  });

  it('refundPayment with amount + reason', async () => {
    await refundPayment(99, { amount: 1000, reason: '客户取消' });
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ amount: 1000, reason: '客户取消' });
  });

  it('refundPayment with no body when empty', async () => {
    await refundPayment(99);
    expect(requestSpy.mock.calls[0]![0].data).toEqual({});
  });
});