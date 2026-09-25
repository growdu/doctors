// src/stores/hospital.test.js
//
// hospital store 单测 —— loadList / loadDetail / clearDetail + loading/error 状态机。
//
// 测试策略：
//   - jest.doMock('@/api/hospital.js') 注入 fake api
//   - 动态 import('@/stores/hospital.js') 拿到 useHospitalStore
//   - setActivePinia(createPinia()) 每个 case 隔离 store 实例

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('hospital store - 基础', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('初始：list / detail / pagination 都为空，loading=false', async () => {
    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    expect(s.list).toEqual([]);
    expect(s.detail).toBeNull();
    expect(s.pagination).toEqual({ total: 0, page: 1, limit: 20 });
    expect(s.loading).toBe(false);
    expect(s.loadingDetail).toBe(false);
    expect(s.error).toBeNull();
  });
});

describe('hospital store - loadList', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('拉列表 → 写入 list + pagination + lastQuery', async () => {
    const getHospitals = jest.fn(async () => ({
      items: [
        { id: 101, name: '北京协和医院', city_id: 1, level: '三甲' },
        { id: 102, name: '华西医院', city_id: 2, level: '三甲' },
      ],
      total: 5,
      page: 1,
      limit: 20,
    }));
    jest.doMock('@/api/hospital.js', () => ({ getHospitals }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    const r = await s.loadList({ city_id: 1 });

    expect(s.list).toHaveLength(2);
    expect(s.list[0].name).toBe('北京协和医院');
    expect(s.pagination.total).toBe(5);
    expect(s.pagination.page).toBe(1);
    expect(s.pagination.limit).toBe(20);
    expect(s.lastQuery).toEqual({ city_id: 1 });
    expect(getHospitals).toHaveBeenCalledWith({ city_id: 1 });
    expect(r.items).toHaveLength(2);
  });

  it('loadList 直返数组也能写入（兼容）', async () => {
    const getHospitals = jest.fn(async () => [{ id: 1, name: 'x' }]);
    jest.doMock('@/api/hospital.js', () => ({ getHospitals }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    await s.loadList();
    expect(s.list).toEqual([{ id: 1, name: 'x' }]);
    // 直返数组：pagination.total = list.length
    expect(s.pagination.total).toBe(1);
  });

  it('loadList 失败 → 抛错 + error 字段写入 + loading 复位', async () => {
    const getHospitals = jest.fn(async () => {
      throw new Error('network down');
    });
    jest.doMock('@/api/hospital.js', () => ({ getHospitals }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    await expect(s.loadList()).rejects.toThrow(/network down/);
    expect(s.error).toBeTruthy();
    expect(s.error.message).toBe('network down');
    expect(s.loading).toBe(false);
  });

  it('loadList 兼容 data 字段（{data: [...]})', async () => {
    const getHospitals = jest.fn(async () => ({
      data: [{ id: 1, name: 'x' }],
      total: 1,
      page: 1,
      limit: 20,
    }));
    jest.doMock('@/api/hospital.js', () => ({ getHospitals }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    await s.loadList();
    expect(s.list).toEqual([{ id: 1, name: 'x' }]);
  });
});

describe('hospital store - loadDetail / clearDetail', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('loadDetail 写入 detail', async () => {
    const detail = {
      id: 101,
      name: '北京协和医院',
      departments: ['内科', '外科'],
    };
    const getHospitalDetail = jest.fn(async () => detail);
    jest.doMock('@/api/hospital.js', () => ({ getHospitalDetail }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    const r = await s.loadDetail(101);

    expect(s.detail).toBe(detail);
    expect(getHospitalDetail).toHaveBeenCalledWith(101);
    expect(r.departments).toHaveLength(2);
    expect(s.loadingDetail).toBe(false);
  });

  it('loadDetail 失败 → 抛错 + error 字段写入', async () => {
    const getHospitalDetail = jest.fn(async () => {
      throw new Error('not found');
    });
    jest.doMock('@/api/hospital.js', () => ({ getHospitalDetail }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    await expect(s.loadDetail(999)).rejects.toThrow(/not found/);
    expect(s.error.message).toBe('not found');
  });

  it('clearDetail 清空当前详情', async () => {
    const getHospitalDetail = jest.fn(async () => ({ id: 1, name: 'x' }));
    jest.doMock('@/api/hospital.js', () => ({ getHospitalDetail }));

    const { useHospitalStore } = await import('@/stores/hospital.js');
    const s = useHospitalStore();
    await s.loadDetail(1);
    expect(s.detail).not.toBeNull();
    s.clearDetail();
    expect(s.detail).toBeNull();
  });
});