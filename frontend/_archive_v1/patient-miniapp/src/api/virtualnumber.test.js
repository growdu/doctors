// src/api/virtualnumber.test.js
//
// virtualnumber api 单测 —— allocateVirtualNumber / getVirtualNumber。
//
// 测试策略：与 hospital.test.js 同款 jest.doMock('../../utils/request.js')。

import { jest } from '@jest/globals';

describe('api/virtualnumber - allocateVirtualNumber', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 POST /virtual-numbers/allocate + 透传完整 body', async () => {
    const created = {
      id: 1,
      order_id: 7,
      patient_id: 7,
      escort_id: 11,
      phone: '17000000001',
      status: 'active',
      expire_at: '2026-09-25T12:00:00Z',
      released_at: null,
      created_at: '2026-09-24T12:00:00Z',
    };
    const requestMock = jest.fn(async () => created);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { allocateVirtualNumber } = await import('@/api/virtualnumber.js');
    const r = await allocateVirtualNumber({
      order_id: 7,
      patient_id: 7,
      escort_id: 11,
      expire_at: '2026-09-25T12:00:00Z',
    });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/virtual-numbers/allocate',
        method: 'POST',
        data: {
          order_id: 7,
          patient_id: 7,
          escort_id: 11,
          expire_at: '2026-09-25T12:00:00Z',
        },
      }),
    );
    expect(r.phone).toBe('17000000001');
    expect(r.status).toBe('active');
  });

  it('缺 order_id / patient_id / escort_id / expire_at → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { allocateVirtualNumber } = await import('@/api/virtualnumber.js');
    await expect(allocateVirtualNumber(null)).rejects.toThrow(/required/);
    await expect(allocateVirtualNumber({})).rejects.toThrow(/required/);
    await expect(allocateVirtualNumber({ order_id: 7 })).rejects.toThrow(/required/);
    await expect(allocateVirtualNumber({ order_id: 7, patient_id: 7 })).rejects.toThrow(/required/);
    await expect(
      allocateVirtualNumber({ order_id: 7, patient_id: 7, escort_id: 11 }),
    ).rejects.toThrow(/required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});

describe('api/virtualnumber - getVirtualNumber', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 GET /virtual-numbers/:id 并透传响应', async () => {
    const vn = {
      id: 1,
      order_id: 7,
      patient_id: 7,
      escort_id: 11,
      phone: '17000000001',
      status: 'active',
      expire_at: '2026-09-25T12:00:00Z',
    };
    const requestMock = jest.fn(async () => vn);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getVirtualNumber } = await import('@/api/virtualnumber.js');
    const r = await getVirtualNumber(1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/virtual-numbers/1', method: 'GET' }),
    );
    expect(r.phone).toBe('17000000001');
  });

  it('缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getVirtualNumber } = await import('@/api/virtualnumber.js');
    await expect(getVirtualNumber(undefined)).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});

describe('api/index.js - 扩展后命名空间聚合', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('默认导出 api 命名空间含 7 个子模块', async () => {
    jest.doMock('../../utils/request.js', () => ({
      request: jest.fn(async () => ({})),
    }));

    const mod = await import('@/api/index.js');
    const api = mod.default;

    expect(api).toBeDefined();
    expect(typeof api.candidates.getCandidates).toBe('function');
    expect(typeof api.order.getOrder).toBe('function');
    expect(typeof api.hospital.getHospitals).toBe('function');
    expect(typeof api.hospital.getHospitalDetail).toBe('function');
    expect(typeof api.address.getAddressList).toBe('function');
    expect(typeof api.address.addAddress).toBe('function');
    expect(typeof api.coupon.getCoupons).toBe('function');
    expect(typeof api.coupon.claimCoupon).toBe('function');
    expect(typeof api.review.submitReview).toBe('function');
    expect(typeof api.virtualnumber.getVirtualNumber).toBe('function');
  });

  it('named exports 也都可用（5 个新模块的代表函数）', async () => {
    jest.doMock('../../utils/request.js', () => ({
      request: jest.fn(async () => ({})),
    }));

    const mod = await import('@/api/index.js');
    expect(typeof mod.getHospitals).toBe('function');
    expect(typeof mod.getHospitalDetail).toBe('function');
    expect(typeof mod.getAddressList).toBe('function');
    expect(typeof mod.addAddress).toBe('function');
    expect(typeof mod.getCoupons).toBe('function');
    expect(typeof mod.claimCoupon).toBe('function');
    expect(typeof mod.submitReview).toBe('function');
    expect(typeof mod.getVirtualNumber).toBe('function');
  });
});