// src/stores/coupon.test.js
//
// coupon store 单测 —— loadTemplates / loadMine / claim / useCoupon + filterByStatus。

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('coupon store - 基础', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('初始：templates / mine 都为空', async () => {
    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    expect(s.templates).toEqual([]);
    expect(s.mine).toEqual([]);
    expect(s.loading).toBe(false);
    expect(s.loadingMine).toBe(false);
  });

  it('状态常量：COUPON_STATUS_* / COUPON_STATUS_LABEL 暴露', async () => {
    const {
      COUPON_STATUS_UNUSED,
      COUPON_STATUS_USED,
      COUPON_STATUS_EXPIRED,
      COUPON_STATUS_LABEL,
    } = await import('@/stores/coupon.js');
    expect(COUPON_STATUS_UNUSED).toBe('unused');
    expect(COUPON_STATUS_USED).toBe('used');
    expect(COUPON_STATUS_EXPIRED).toBe('expired');
    expect(COUPON_STATUS_LABEL[COUPON_STATUS_UNUSED]).toBe('未使用');
    expect(COUPON_STATUS_LABEL[COUPON_STATUS_USED]).toBe('已使用');
    expect(COUPON_STATUS_LABEL[COUPON_STATUS_EXPIRED]).toBe('已过期');
  });
});

describe('coupon store - loadTemplates', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('拉模板列表 → 写入 templates', async () => {
    const getCoupons = jest.fn(async () => ({
      items: [
        { id: 1, name: '新人券', value: 30, threshold: 100, stock: 100 },
        { id: 2, name: '满减券', value: 50, threshold: 200, stock: 50 },
      ],
    }));
    jest.doMock('@/api/coupon.js', () => ({ getCoupons }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    const r = await s.loadTemplates({ page: 1 });

    expect(s.templates).toHaveLength(2);
    expect(s.templates[0].name).toBe('新人券');
    expect(getCoupons).toHaveBeenCalledWith({ page: 1 });
    expect(r).toHaveLength(2);
  });

  it('loadTemplates 失败 → 抛错 + error 字段写入', async () => {
    const getCoupons = jest.fn(async () => {
      throw new Error('boom');
    });
    jest.doMock('@/api/coupon.js', () => ({ getCoupons }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    await expect(s.loadTemplates()).rejects.toThrow(/boom/);
    expect(s.error.message).toBe('boom');
  });
});

describe('coupon store - loadMine / claim', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('loadMine → 写入 mine', async () => {
    const getMyCoupons = jest.fn(async () => ({
      items: [
        { id: 100, status: 'unused', coupon: { id: 1, name: 'x' } },
        { id: 101, status: 'used', coupon: { id: 1, name: 'x' } },
      ],
    }));
    jest.doMock('@/api/coupon.js', () => ({ getMyCoupons }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    await s.loadMine();
    expect(s.mine).toHaveLength(2);
    expect(s.mine[0].status).toBe('unused');
  });

  it('claim → 调 api + 重新拉 my', async () => {
    const claimCoupon = jest.fn(async () => ({
      id: 100,
      user_id: 7,
      coupon_id: 1,
      status: 'unused',
      coupon: { id: 1, name: '新人券' },
    }));
    const getMyCoupons = jest.fn(async () => ({
      items: [
        { id: 100, status: 'unused', coupon: { id: 1, name: '新人券' } },
      ],
    }));
    jest.doMock('@/api/coupon.js', () => ({ claimCoupon, getMyCoupons }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    const r = await s.claim(1);
    expect(claimCoupon).toHaveBeenCalledWith(1);
    expect(r.id).toBe(100);
    expect(getMyCoupons).toHaveBeenCalled();
    expect(s.mine).toHaveLength(1);
    expect(s.lastClaimedId).toBe(100);
  });
});

describe('coupon store - useCoupon / filterByStatus / statusCount', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('useCoupon → 调 api + 本地同步 status=used', async () => {
    const useMyCoupon = jest.fn(async () => ({}));
    jest.doMock('@/api/coupon.js', () => ({
      useMyCoupon,
      getMyCoupons: jest.fn(async () => ({
        items: [{ id: 100, status: 'unused' }, { id: 101, status: 'unused' }],
      })),
    }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    await s.loadMine();
    await s.useCoupon(100);
    expect(useMyCoupon).toHaveBeenCalledWith(100);
    expect(s.mine[0].status).toBe('used');
    expect(s.mine[0].used_at).toBeTruthy();
    expect(s.mine[1].status).toBe('unused');
  });

  it('filterByStatus 按 status 过滤', async () => {
    jest.doMock('@/api/coupon.js', () => ({
      getMyCoupons: jest.fn(async () => ({
        items: [
          { id: 100, status: 'unused' },
          { id: 101, status: 'used' },
          { id: 102, status: 'unused' },
        ],
      })),
    }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    await s.loadMine();
    expect(s.filterByStatus('unused')).toHaveLength(2);
    expect(s.filterByStatus('used')).toHaveLength(1);
    expect(s.filterByStatus('expired')).toHaveLength(0);
    expect(s.filterByStatus()).toHaveLength(3);
  });

  it('statusCount 正确分类计数', async () => {
    jest.doMock('@/api/coupon.js', () => ({
      getMyCoupons: jest.fn(async () => ({
        items: [
          { id: 1, status: 'unused' },
          { id: 2, status: 'unused' },
          { id: 3, status: 'used' },
          { id: 4, status: 'expired' },
        ],
      })),
    }));

    const { useCouponStore } = await import('@/stores/coupon.js');
    const s = useCouponStore();
    await s.loadMine();
    const c = s.statusCount();
    expect(c).toEqual({ unused: 2, used: 1, expired: 1, total: 4 });
  });
});