// src/stores/review.test.js
//
// review store 单测 —— submit / loadList / loadDetail + submitting 状态机。

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('review store - 基础', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('初始：mine / current 都为空，submitting=false', async () => {
    const { useReviewStore } = await import('@/stores/review.js');
    const s = useReviewStore();
    expect(s.mine).toEqual([]);
    expect(s.current).toBeNull();
    expect(s.submitting).toBe(false);
    expect(s.loading).toBe(false);
    expect(s.error).toBeNull();
  });
});

describe('review store - submit', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('submit 成功 → 追加到 mine + 写 lastSubmittedId', async () => {
    const submitReview = jest.fn(async () => ({
      id: 999,
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: '专业',
    }));
    jest.doMock('@/api/review.js', () => ({ submitReview }));

    const { useReviewStore } = await import('@/stores/review.js');
    const s = useReviewStore();
    const r = await s.submit({
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: '专业',
    });

    expect(submitReview).toHaveBeenCalled();
    expect(r.id).toBe(999);
    expect(s.mine).toHaveLength(1);
    expect(s.mine[0].rating).toBe(5);
    expect(s.lastSubmittedId).toBe(999);
    expect(s.submitting).toBe(false);
  });

  it('submit 失败 → 抛错 + error 字段写入 + submitting 复位', async () => {
    const submitReview = jest.fn(async () => {
      throw new Error('rating invalid');
    });
    jest.doMock('@/api/review.js', () => ({ submitReview }));

    const { useReviewStore } = await import('@/stores/review.js');
    const s = useReviewStore();
    await expect(s.submit({ order_id: 7, escort_id: 11, rating: 5 }))
      .rejects.toThrow(/rating invalid/);
    expect(s.error.message).toBe('rating invalid');
    expect(s.submitting).toBe(false);
    expect(s.mine).toEqual([]);
  });
});

describe('review store - loadList / loadDetail', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('loadList 兼容 reviews / items / data 三种返回', async () => {
    const listReviews = jest.fn(async () => ({
      reviews: [{ id: 1, rating: 5 }],
      page: 1,
      page_size: 20,
    }));
    jest.doMock('@/api/review.js', () => ({ listReviews }));

    const { useReviewStore } = await import('@/stores/review.js');
    const s = useReviewStore();
    await s.loadList();
    expect(s.mine).toHaveLength(1);
  });

  it('loadList 兼容 {items: [...]}', async () => {
    const listReviews = jest.fn(async () => ({ items: [{ id: 1 }] }));
    jest.doMock('@/api/review.js', () => ({ listReviews }));

    const { useReviewStore } = await import('@/stores/review.js');
    const s = useReviewStore();
    await s.loadList({ escort_id: 11 });
    expect(s.mine).toEqual([{ id: 1 }]);
    expect(listReviews).toHaveBeenCalledWith({ escort_id: 11 });
  });

  it('loadDetail → 写入 current', async () => {
    const getReviewDetail = jest.fn(async () => ({
      id: 1,
      order_id: 7,
      escort_id: 11,
      rating: 5,
      comment: 'good',
    }));
    jest.doMock('@/api/review.js', () => ({ getReviewDetail }));

    const { useReviewStore } = await import('@/stores/review.js');
    const s = useReviewStore();
    await s.loadDetail(1);
    expect(s.current.id).toBe(1);
    expect(getReviewDetail).toHaveBeenCalledWith(1);
  });
});