// src/api/hospital.test.js
//
// hospital api 单测 —— 验证 getHospitals / getHospitalDetail 的 URL + method + query + 缺参兜底。
//
// 测试策略：与 order.test.js / candidates.test.js 同款 ——
//   jest.doMock('../../utils/request.js') 替换 fetch，避免依赖 uni runtime。
//
// 注意：mock 路径必须与 import 路径一致 —— api/hospital.js 用的是根 utils/request.js，
// 而 jest.config.js 的 '@/' 映射是 src/utils/，故 mock 走 '../../utils/request.js'。

import { jest } from '@jest/globals';

describe('api/hospital - getHospitals', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 request GET /hospitals 并透传 query', async () => {
    const resp = {
      items: [
        { id: 101, name: '北京协和医院', city_id: 1, level: '三甲' },
        { id: 102, name: '华西医院', city_id: 2, level: '三甲' },
      ],
      total: 2,
      page: 1,
      limit: 20,
    };
    const requestMock = jest.fn(async () => resp);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getHospitals } = await import('@/api/hospital.js');
    const r = await getHospitals({ city_id: 1, level: '三甲', keyword: '协和' });

    expect(requestMock).toHaveBeenCalledTimes(1);
    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/hospitals',
        method: 'GET',
        query: { city_id: 1, level: '三甲', keyword: '协和' },
      }),
    );
    expect(r.items).toHaveLength(2);
    expect(r.total).toBe(2);
  });

  it('不传 query → query 默认 {}', async () => {
    const requestMock = jest.fn(async () => ({ items: [], total: 0 }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getHospitals } = await import('@/api/hospital.js');
    await getHospitals();

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/hospitals', method: 'GET', query: {} }),
    );
  });
});

describe('api/hospital - getHospitalDetail', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 request GET /hospitals/:id 并透传响应', async () => {
    const hospital = {
      id: 101,
      name: '北京协和医院',
      city_id: 1,
      level: '三甲',
      status: 'active',
      address: '东城区帅府园 1 号',
      lat: 39.91,
      lng: 116.42,
      phone: '010-69155555',
      departments: ['内科', '外科', '妇产科'],
      description: '三甲综合医院',
    };
    const requestMock = jest.fn(async () => hospital);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getHospitalDetail } = await import('@/api/hospital.js');
    const r = await getHospitalDetail(101);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/hospitals/101', method: 'GET' }),
    );
    expect(r).toBe(hospital);
    expect(r.departments).toHaveLength(3);
  });

  it('getHospitalDetail 缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getHospitalDetail } = await import('@/api/hospital.js');
    await expect(getHospitalDetail(undefined)).rejects.toThrow(/id is required/);
    await expect(getHospitalDetail(null)).rejects.toThrow(/id is required/);
    await expect(getHospitalDetail('')).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('getHospitalDetail 失败时透传 ApiError', async () => {
    const requestMock = jest.fn(async () => {
      throw new Error('hospital not found');
    });
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getHospitalDetail } = await import('@/api/hospital.js');
    await expect(getHospitalDetail(999)).rejects.toThrow(/hospital not found/);
  });
});