/**
 * admin/reviews/index.vue 组件单测（vitest + Vue Test Utils）。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import * as reviewApi from '@/api/review';
import type { Review } from '@/api/review';
import AdminReviews from './index.vue';

const review = (id: number, rating: number, content = 'good'): Review => ({
  id,
  order_id: 100 + id,
  patient_id: 1,
  escort_id: 10 + id,
  rating,
  tags: ['专业', '准时'],
  content,
  reply: null,
  reply_at: null,
  created_at: '2026-10-01T10:00:00Z',
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/reviews/index · 渲染', () => {
  it('渲染标题', () => {
    vi.spyOn(reviewApi, 'listReviews').mockResolvedValue({ items: [], total: 0 });

    const w = mount(AdminReviews);
    expect(w.text()).toContain('评价管理');
  });

  it('加载成功渲染 4 张统计卡', async () => {
    vi.spyOn(reviewApi, 'listReviews').mockResolvedValue({
      items: [review(1, 5), review(2, 5), review(3, 4), review(4, 2)],
      total: 4,
    });

    const w = mount(AdminReviews);
    await flushPromises();

    expect(w.find('[data-testid="admin-reviews-stat-total"]').text()).toContain('4');
    expect(w.find('[data-testid="admin-reviews-stat-avg"]').text()).toContain('4.0');
    expect(w.find('[data-testid="admin-reviews-stat-five"]').text()).toContain('2');
    expect(w.find('[data-testid="admin-reviews-stat-low"]').text()).toContain('1');
  });

  it('渲染评价列表（截前 20 条）', async () => {
    vi.spyOn(reviewApi, 'listReviews').mockResolvedValue({
      items: [review(1, 5, '非常好')],
      total: 1,
    });

    const w = mount(AdminReviews);
    await flushPromises();

    expect(w.find('[data-testid="admin-review-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-review-rating-1"]').text()).toContain('★');
    expect(w.text()).toContain('非常好');
    expect(w.text()).toContain('#专业');
  });
});

describe('admin/reviews/index · 失败回退', () => {
  it('加载失败渲染空态 + 重试按钮', async () => {
    vi.spyOn(reviewApi, 'listReviews').mockRejectedValue(new Error('network'));

    const w = mount(AdminReviews);
    await flushPromises();

    expect(w.find('[data-testid="admin-reviews-error"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-reviews-retry"]').exists()).toBe(true);
  });

  it('点击重试重新调 onLoad', async () => {
    const listSpy = vi.spyOn(reviewApi, 'listReviews')
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({ items: [], total: 0 });

    const w = mount(AdminReviews);
    await flushPromises();

    await w.find('[data-testid="admin-reviews-retry"]').trigger('click');
    await flushPromises();

    expect(listSpy).toHaveBeenCalledTimes(2);
  });
});