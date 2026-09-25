// src/stores/virtualnumber.test.js
//
// virtualnumber store 单测 —— allocate / loadDetail + byOrderId 缓存 + maskedPhone。

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('virtualnumber store - 基础', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('初始：current=null + byOrderId={} + loading=false', async () => {
    const { useVirtualNumberStore } = await import('@/stores/virtualnumber.js');
    const s = useVirtualNumberStore();
    expect(s.current).toBeNull();
    expect(s.byOrderId).toEqual({});
    expect(s.loading).toBe(false);
    expect(s.error).toBeNull();
  });

  it('状态常量 + 状态文案暴露', async () => {
    const {
      VIRTUAL_NUMBER_STATUS_ACTIVE,
      VIRTUAL_NUMBER_STATUS_RELEASED,
      VIRTUAL_NUMBER_STATUS_EXPIRED,
      VIRTUAL_NUMBER_STATUS_LABEL,
    } = await import('@/stores/virtualnumber.js');
    expect(VIRTUAL_NUMBER_STATUS_ACTIVE).toBe('active');
    expect(VIRTUAL_NUMBER_STATUS_RELEASED).toBe('released');
    expect(VIRTUAL_NUMBER_STATUS_EXPIRED).toBe('expired');
    expect(VIRTUAL_NUMBER_STATUS_LABEL[VIRTUAL_NUMBER_STATUS_ACTIVE]).toBe('使用中');
    expect(VIRTUAL_NUMBER_STATUS_LABEL[VIRTUAL_NUMBER_STATUS_RELEASED]).toBe('已释放');
    expect(VIRTUAL_NUMBER_STATUS_LABEL[VIRTUAL_NUMBER_STATUS_EXPIRED]).toBe('已过期');
  });
});

describe('virtualnumber store - allocate', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('allocate 成功 → 写 current + byOrderId', async () => {
    const vn = {
      id: 1,
      order_id: 7,
      patient_id: 7,
      escort_id: 11,
      phone: '17000000001',
      status: 'active',
      expire_at: '2026-09-25T12:00:00Z',
    };
    const allocateVirtualNumber = jest.fn(async () => vn);
    jest.doMock('@/api/virtualnumber.js', () => ({ allocateVirtualNumber }));

    const { useVirtualNumberStore } = await import('@/stores/virtualnumber.js');
    const s = useVirtualNumberStore();
    const r = await s.allocate({
      order_id: 7,
      patient_id: 7,
      escort_id: 11,
      expire_at: '2026-09-25T12:00:00Z',
    });

    expect(r.id).toBe(1);
    expect(s.current).toBe(vn);
    expect(s.byOrderId[7]).toBe(vn);
    expect(s.loading).toBe(false);
  });

  it('allocate 失败 → 抛错 + error 字段写入', async () => {
    const allocateVirtualNumber = jest.fn(async () => {
      throw new Error('order already has active virtual number');
    });
    jest.doMock('@/api/virtualnumber.js', () => ({ allocateVirtualNumber }));

    const { useVirtualNumberStore } = await import('@/stores/virtualnumber.js');
    const s = useVirtualNumberStore();
    await expect(s.allocate({
      order_id: 7,
      patient_id: 7,
      escort_id: 11,
      expire_at: '2026-09-25T12:00:00Z',
    })).rejects.toThrow(/already has active/);
    expect(s.error.message).toBe('order already has active virtual number');
    expect(s.loading).toBe(false);
  });
});

describe('virtualnumber store - loadDetail / getByOrderId / maskedPhone', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('loadDetail → 写 current + byOrderId', async () => {
    const vn = { id: 1, order_id: 7, phone: '17000000001', status: 'active' };
    const getVirtualNumber = jest.fn(async () => vn);
    jest.doMock('@/api/virtualnumber.js', () => ({ getVirtualNumber }));

    const { useVirtualNumberStore } = await import('@/stores/virtualnumber.js');
    const s = useVirtualNumberStore();
    await s.loadDetail(1);
    expect(s.current).toBe(vn);
    expect(s.byOrderId[7]).toBe(vn);
    expect(getVirtualNumber).toHaveBeenCalledWith(1);
  });

  it('getByOrderId helper → 拿缓存中的虚拟号', async () => {
    const { useVirtualNumberStore } = await import('@/stores/virtualnumber.js');
    const s = useVirtualNumberStore();
    expect(s.getByOrderId(undefined)).toBeNull();
    expect(s.getByOrderId(7)).toBeNull();
    // 手动写 byOrderId
    s.byOrderId = { 7: { id: 1, order_id: 7, phone: '17000000001' } };
    const v = s.getByOrderId(7);
    expect(v.id).toBe(1);
  });

  it('maskedPhone → 11 位手机号中间四位掩码；非 11 位原样返回', async () => {
    const { useVirtualNumberStore } = await import('@/stores/virtualnumber.js');
    const s = useVirtualNumberStore();
    expect(s.maskedPhone('17000000001')).toBe('170****0001');
    expect(s.maskedPhone('12345')).toBe('12345');
    expect(s.maskedPhone('')).toBe('');
    expect(s.maskedPhone(undefined)).toBe('');
  });
});