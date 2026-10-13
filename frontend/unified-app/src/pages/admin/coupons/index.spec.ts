/**
 * admin/coupons/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染标题 + 4 张统计卡
 *   - 加载成功：渲染可领取券列表 + 用户已领取列表
 *   - 加载失败：渲染空态 + 重试按钮
 *   - 点击重试重新调 onLoad
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import * as userApi from '@/api/user';
import type { Coupon } from '@/api/user';
import AdminCoupons from './index.vue';

const coupon = (id: number, status: Coupon['status']): Coupon => ({
  id,
  title: `优惠券 #${id}`,
  amount: 5000,
  min_amount: 10000,
  valid_from: '2026-10-01T00:00:00Z',
  valid_to: '2026-12-31T23:59:59Z',
  status,
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/coupons/index · 渲染', () => {
  it('渲染标题', () => {
    vi.spyOn(userApi, 'listAvailableCoupons').mockResolvedValue({ items: [] });
    vi.spyOn(userApi, 'listMyCoupons').mockResolvedValue({ items: [] });

    const w = mount(AdminCoupons);
    expect(w.text()).toContain('优惠券管理');
  });

  it('加载成功渲染 4 张统计卡', async () => {
    vi.spyOn(userApi, 'listAvailableCoupons').mockResolvedValue({
      items: [coupon(1, 'available'), coupon(2, 'available')],
    });
    vi.spyOn(userApi, 'listMyCoupons').mockResolvedValue({
      items: [coupon(3, 'claimed'), coupon(4, 'used')],
    });

    const w = mount(AdminCoupons);
    await flushPromises();

    expect(w.find('[data-testid="admin-coupons-stat-total"]').text()).toContain('4');
    expect(w.find('[data-testid="admin-coupons-stat-available"]').text()).toContain('2');
    expect(w.find('[data-testid="admin-coupons-stat-claimed"]').text()).toContain('1');
    expect(w.find('[data-testid="admin-coupons-stat-used"]').text()).toContain('1');
  });

  it('加载成功渲染可领取 + 已领取 2 个 section', async () => {
    vi.spyOn(userApi, 'listAvailableCoupons').mockResolvedValue({
      items: [coupon(1, 'available')],
    });
    vi.spyOn(userApi, 'listMyCoupons').mockResolvedValue({
      items: [coupon(2, 'claimed')],
    });

    const w = mount(AdminCoupons);
    await flushPromises();

    expect(w.find('[data-testid="admin-coupon-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-my-coupon-2"]').exists()).toBe(true);
    expect(w.text()).toContain('可领取券（1）');
    expect(w.text()).toContain('用户已领取（1）');
  });
});

describe('admin/coupons/index · 失败回退', () => {
  it('两个 API 都失败渲染空态 + 重试按钮', async () => {
    vi.spyOn(userApi, 'listAvailableCoupons').mockRejectedValue(new Error('network'));
    vi.spyOn(userApi, 'listMyCoupons').mockRejectedValue(new Error('network'));

    const w = mount(AdminCoupons);
    await flushPromises();

    expect(w.find('[data-testid="admin-coupons-error"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-coupons-retry"]').exists()).toBe(true);
  });

  it('点击重试重新调 onLoad', async () => {
    const availableSpy = vi.spyOn(userApi, 'listAvailableCoupons')
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({ items: [] });
    vi.spyOn(userApi, 'listMyCoupons').mockRejectedValue(new Error('network'));

    const w = mount(AdminCoupons);
    await flushPromises();

    await w.find('[data-testid="admin-coupons-retry"]').trigger('click');
    await flushPromises();

    expect(availableSpy).toHaveBeenCalledTimes(2);
  });

  it('单 API 失败不阻塞渲染（部分降级）', async () => {
    vi.spyOn(userApi, 'listAvailableCoupons').mockRejectedValue(new Error('network'));
    vi.spyOn(userApi, 'listMyCoupons').mockResolvedValue({ items: [coupon(99, 'claimed')] });

    const w = mount(AdminCoupons);
    await flushPromises();

    // 不进入 error 态，因为另一个 API 成功
    expect(w.find('[data-testid="admin-coupons-error"]').exists()).toBe(false);
    // 但渲染「已领取」section（降级为可领取 0 + 已领取 1）
    expect(w.find('[data-testid="admin-my-coupon-99"]').exists()).toBe(true);
  });
});