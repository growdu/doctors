/**
 * api/sos 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { raiseSos, listSos, resolveSos } from './sos';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({});
});

describe('api/sos', () => {
  it('raiseSos POSTs reason + location', async () => {
    await raiseSos({
      order_id: 42,
      reason: '陪诊现场出现意外',
      location: { lat: 39.9, lng: 116.4, address: '北京协和医院' },
    });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/sos',
      method: 'POST',
      data: {
        order_id: 42,
        reason: '陪诊现场出现意外',
        location: { lat: 39.9, lng: 116.4, address: '北京协和医院' },
      },
      baseURL: 'http://127.0.0.1:8087',
    });
  });

  it('raiseSos without location', async () => {
    await raiseSos({ reason: '突发' });
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ reason: '突发' });
  });

  it('listSos builds query with status', async () => {
    await listSos({ status: 'pending' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('status=pending');
  });

  it('resolveSos POSTs remark', async () => {
    await resolveSos(99, '已联系家属');
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/sos/99/resolve',
      method: 'POST',
      data: { remark: '已联系家属' },
      baseURL: 'http://127.0.0.1:8087',
    });
  });
});