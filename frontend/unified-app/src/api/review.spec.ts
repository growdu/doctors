/**
 * api/review 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { createReview, listReviews, replyReview } from './review';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({ items: [] });
});

describe('api/review', () => {
  it('createReview POSTs body', async () => {
    await createReview({ order_id: 42, rating: 5, tags: ['细心'], content: '很好' });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/reviews',
      method: 'POST',
      data: { order_id: 42, rating: 5, tags: ['细心'], content: '很好' },
      baseURL: 'http://127.0.0.1:8086',
    });
  });

  it('listReviews with escort_id filter', async () => {
    await listReviews({ escort_id: 100 });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('escort_id=100');
  });

  it('listReviews with order_id filter', async () => {
    await listReviews({ order_id: 42 });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('order_id=42');
  });

  it('replyReview POSTs reply text', async () => {
    await replyReview(7, '感谢好评');
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/reviews/7/reply',
      method: 'POST',
      data: { reply: '感谢好评' },
      baseURL: 'http://127.0.0.1:8086',
    });
  });
});