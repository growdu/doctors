// src/api/review.test.js
//
// review api 单测 —— submitReview / listReviews / getReviewDetail。
//
// 测试策略：与 hospital.test.js 同款 jest.doMock('../../utils/request.js')。

import { jest } from '@jest/globals';

describe('api/review - submitReview', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 POST /reviews + 透传 {order_id, escort_id, rating, comment}', async () => {
    const created = {
      id: 999,
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: '非常专业',
    };
    const requestMock = jest.fn(async () => created);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { submitReview } = await import('@/api/review.js');
    const r = await submitReview({
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: '非常专业',
    });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/reviews',
        method: 'POST',
        data: { order_id: 7, escort_id: 11, rating: 5, comment: '非常专业' },
      }),
    );
    expect(r.id).toBe(999);
  });

  it('comment 缺省 → body.comment 默认空串', async () => {
    const requestMock = jest.fn(async () => ({}));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { submitReview } = await import('@/api/review.js');
    await submitReview({ order_id: 7, escort_id: 11, rating: 4 });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        data: { order_id: 7, escort_id: 11, rating: 4, comment: '' },
      }),
    );
  });

  it('rating 越界 (0 / 6) → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { submitReview } = await import('@/api/review.js');
    await expect(submitReview({ order_id: 7, escort_id: 11, rating: 0 })).rejects.toThrow(/rating/);
    await expect(submitReview({ order_id: 7, escort_id: 11, rating: 6 })).rejects.toThrow(/rating/);
    await expect(submitReview({ order_id: 7, escort_id: 11, rating: 'abc' })).rejects.toThrow(/rating/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('缺 order_id / escort_id / rating → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { submitReview } = await import('@/api/review.js');
    await expect(submitReview(null)).rejects.toThrow(/required/);
    await expect(submitReview({})).rejects.toThrow(/required/);
    await expect(submitReview({ order_id: 7 })).rejects.toThrow(/required/);
    await expect(submitReview({ order_id: 7, escort_id: 11 })).rejects.toThrow(/required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});

describe('api/review - listReviews', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 GET /reviews 并透传 query', async () => {
    const resp = {
      reviews: [
        { id: 1, order_id: 7, escort_id: 11, rating: 5, comment: '专业' },
      ],
      page: 1,
      page_size: 20,
    };
    const requestMock = jest.fn(async () => resp);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { listReviews } = await import('@/api/review.js');
    const r = await listReviews({ escort_id: 11, min_rating: 4, page: 1 });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/reviews',
        method: 'GET',
        query: { escort_id: 11, min_rating: 4, page: 1 },
      }),
    );
    expect(r.reviews).toHaveLength(1);
  });

  it('不传 query → query 默认 {}', async () => {
    const requestMock = jest.fn(async () => ({ reviews: [], page: 1, page_size: 20 }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { listReviews } = await import('@/api/review.js');
    await listReviews();

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/reviews', method: 'GET', query: {} }),
    );
  });
});

describe('api/review - getReviewDetail', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 GET /reviews/:id', async () => {
    const review = {
      id: 1,
      order_id: 7,
      escort_id: 11,
      reviewer_id: 7,
      rating: 5,
      comment: '专业',
      reply: '感谢支持',
    };
    const requestMock = jest.fn(async () => review);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getReviewDetail } = await import('@/api/review.js');
    const r = await getReviewDetail(1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/reviews/1', method: 'GET' }),
    );
    expect(r.reply).toBe('感谢支持');
  });

  it('缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getReviewDetail } = await import('@/api/review.js');
    await expect(getReviewDetail(undefined)).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});