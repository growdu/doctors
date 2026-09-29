/**
 * api/match 单测：mock client 验证 3 端点调用正确。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fetchFeed, listCandidates, dispatchOrder } from './match';

const requestSpy = vi.fn();

vi.mock('./client', () => ({
  request: (...args: unknown[]) => requestSpy(...args),
}));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({ items: [] });
});

describe('api/match · escort-only', () => {
  it('fetchFeed without order_id', async () => {
    await fetchFeed();
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/match/feed',
      baseURL: 'http://127.0.0.1:8083',
    });
  });

  it('fetchFeed with order_id appends query', async () => {
    await fetchFeed(42);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/match/feed?order_id=42');
  });
});

describe('api/match · common endpoints', () => {
  it('listCandidates POSTs order_id', async () => {
    await listCandidates({ order_id: 42 });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/match/candidates',
      method: 'POST',
      data: { order_id: 42 },
      baseURL: 'http://127.0.0.1:8083',
    });
  });

  it('listCandidates with limit', async () => {
    await listCandidates({ order_id: 42, limit: 5 });
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ order_id: 42, limit: 5 });
  });

  it('dispatchOrder POSTs escort_id + order_id', async () => {
    await dispatchOrder({ order_id: 42, escort_id: 100 });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/match/dispatch',
      method: 'POST',
      data: { order_id: 42, escort_id: 100 },
      baseURL: 'http://127.0.0.1:8083',
    });
  });
});