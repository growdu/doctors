// src/api/coupon.test.js
//
// coupon api 单测 —— 5 个端点 URL + method + body + 缺参兜底。
//
// 测试策略：与 hospital/address.test.js 同款 jest.doMock('../../utils/request.js')。

import { jest } from '@jest/globals';

describe('api/coupon - getCoupons', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 GET /coupons 并透传 query', async () => {
    const resp = {
      items: [
        {
          id: 1,
          name: '新人立减券',
          type: 'amount',
          value: 30,
          threshold: 100,
          stock: 1000,
          status: 'active',
        },
      ],
    };
    const requestMock = jest.fn(async () => resp);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getCoupons } = await import('@/api/coupon.js');
    const r = await getCoupons({ page: 1, limit: 20 });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/coupons', method: 'GET', query: { page: 1, limit: 20 } }),
    );
    expect(r.items[0].name).toBe('新人立减券');
  });
});

describe('api/coupon - getCouponDetail', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 GET /coupons/:id', async () => {
    const requestMock = jest.fn(async () => ({ id: 1, name: '新人券', value: 30 }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getCouponDetail } = await import('@/api/coupon.js');
    await getCouponDetail(1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/coupons/1', method: 'GET' }),
    );
  });

  it('缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getCouponDetail } = await import('@/api/coupon.js');
    await expect(getCouponDetail(undefined)).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});

describe('api/coupon - claimCoupon', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 POST /coupons/:id/claim', async () => {
    const uc = {
      id: 100,
      user_id: 7,
      coupon_id: 1,
      status: 'unused',
      claimed_at: '2026-09-24T10:00:00Z',
      coupon: { id: 1, name: '新人券', value: 30 },
    };
    const requestMock = jest.fn(async () => uc);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { claimCoupon } = await import('@/api/coupon.js');
    const r = await claimCoupon(1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/coupons/1/claim', method: 'POST' }),
    );
    expect(r.status).toBe('unused');
  });

  it('缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { claimCoupon } = await import('@/api/coupon.js');
    await expect(claimCoupon(undefined)).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});

describe('api/coupon - getMyCoupons / useMyCoupon', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('getMyCoupons 调 GET /me/coupons 并透传', async () => {
    const requestMock = jest.fn(async () => ({
      items: [
        {
          id: 100,
          user_id: 7,
          coupon_id: 1,
          status: 'unused',
          coupon: { id: 1, name: '新人券', value: 30 },
        },
      ],
    }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getMyCoupons } = await import('@/api/coupon.js');
    const r = await getMyCoupons();

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/me/coupons', method: 'GET' }),
    );
    expect(r.items[0].coupon.name).toBe('新人券');
  });

  it('getMyCoupons 兼容直返数组', async () => {
    const requestMock = jest.fn(async () => []);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getMyCoupons } = await import('@/api/coupon.js');
    const r = await getMyCoupons();
    expect(r).toEqual([]);
  });

  it('useMyCoupon 调 POST /me/coupons/:id/use', async () => {
    const requestMock = jest.fn(async () => ({ id: 100, status: 'used' }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { useMyCoupon } = await import('@/api/coupon.js');
    await useMyCoupon(100);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/me/coupons/100/use', method: 'POST' }),
    );
  });

  it('useMyCoupon 缺 userCouponId → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { useMyCoupon } = await import('@/api/coupon.js');
    await expect(useMyCoupon(undefined)).rejects.toThrow(/userCouponId is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});