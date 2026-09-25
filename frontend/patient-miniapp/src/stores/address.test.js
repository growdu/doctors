// src/stores/address.test.js
//
// address store 单测 —— loadList / add / update / remove / setDefault + defaultId 计算。

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('address store - 基础', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('初始：list 为空，defaultId=null，loading=false', async () => {
    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    expect(s.list).toEqual([]);
    expect(s.defaultId).toBeNull();
    expect(s.loading).toBe(false);
    expect(s.error).toBeNull();
  });
});

describe('address store - loadList', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('拉列表 → 写入 list + 自动计算 defaultId', async () => {
    const getAddressList = jest.fn(async () => ({
      items: [
        { id: 1, recipient: 'a', is_default: true },
        { id: 2, recipient: 'b', is_default: false },
      ],
    }));
    jest.doMock('@/api/address.js', () => ({ getAddressList }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.loadList();

    expect(s.list).toHaveLength(2);
    expect(s.defaultId).toBe(1);
    expect(getAddressList).toHaveBeenCalledTimes(1);
  });

  it('loadList 无 is_default=true → 取首条', async () => {
    const getAddressList = jest.fn(async () => ({
      items: [
        { id: 1, recipient: 'a', is_default: false },
        { id: 2, recipient: 'b', is_default: false },
      ],
    }));
    jest.doMock('@/api/address.js', () => ({ getAddressList }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.loadList();
    expect(s.defaultId).toBe(1);
  });

  it('loadList 失败 → 抛错 + error 字段写入', async () => {
    const getAddressList = jest.fn(async () => {
      throw new Error('boom');
    });
    jest.doMock('@/api/address.js', () => ({ getAddressList }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await expect(s.loadList()).rejects.toThrow(/boom/);
    expect(s.error.message).toBe('boom');
    expect(s.loading).toBe(false);
  });
});

describe('address store - add / update / remove', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('add 成功 → 追加到列表头 + 写 lastAddedId', async () => {
    const addAddress = jest.fn(async () => ({
      id: 100,
      recipient: 'x',
      is_default: false,
    }));
    const getAddressList = jest.fn(async () => ({ items: [] }));
    jest.doMock('@/api/address.js', () => ({ addAddress, getAddressList }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    const r = await s.add({ recipient: 'x', phone: '13800138000', detail: 'addr' });

    expect(addAddress).toHaveBeenCalled();
    expect(s.list).toHaveLength(1);
    expect(s.list[0].id).toBe(100);
    expect(s.lastAddedId).toBe(100);
    expect(r.id).toBe(100);
  });

  it('add is_default=true → 自动 reload list 以同步 defaultId', async () => {
    const addAddress = jest.fn(async () => ({
      id: 100,
      recipient: 'x',
      is_default: true,
    }));
    const getAddressList = jest.fn(async () => ({
      items: [{ id: 100, recipient: 'x', is_default: true }],
    }));
    jest.doMock('@/api/address.js', () => ({ addAddress, getAddressList }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.add({ recipient: 'x', phone: '13800138000', detail: 'addr', is_default: true });
    expect(getAddressList).toHaveBeenCalled();
    expect(s.defaultId).toBe(100);
  });

  it('update → 列表里对应项合并新数据', async () => {
    const updateAddress = jest.fn(async () => ({
      id: 1,
      recipient: 'new',
    }));
    jest.doMock('@/api/address.js', () => ({
      updateAddress,
      getAddressList: jest.fn(async () => ({
        items: [{ id: 1, recipient: 'old', is_default: true }],
      })),
    }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.loadList();
    await s.update(1, { recipient: 'new', phone: '13800138000', detail: 'addr' });

    expect(s.list[0].recipient).toBe('new');
  });

  it('remove → 从列表移除；删的是默认 → defaultId 回退到首条', async () => {
    const deleteAddress = jest.fn(async () => ({}));
    jest.doMock('@/api/address.js', () => ({
      deleteAddress,
      getAddressList: jest.fn(async () => ({
        items: [
          { id: 1, recipient: 'a', is_default: true },
          { id: 2, recipient: 'b', is_default: false },
        ],
      })),
    }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.loadList();
    expect(s.defaultId).toBe(1);
    await s.remove(1);
    expect(s.list).toHaveLength(1);
    expect(s.list[0].id).toBe(2);
    expect(s.defaultId).toBe(2);
  });
});

describe('address store - setDefault / defaultAddress', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    jest.resetModules();
  });

  it('setDefault → 调 api + 本地同步 is_default', async () => {
    const setDefaultAddress = jest.fn(async () => ({}));
    jest.doMock('@/api/address.js', () => ({
      setDefaultAddress,
      getAddressList: jest.fn(async () => ({
        items: [
          { id: 1, recipient: 'a', is_default: true },
          { id: 2, recipient: 'b', is_default: false },
        ],
      })),
    }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.loadList();
    await s.setDefault(2);
    expect(setDefaultAddress).toHaveBeenCalledWith(2);
    expect(s.defaultId).toBe(2);
    expect(s.list[0].is_default).toBe(false);
    expect(s.list[1].is_default).toBe(true);
  });

  it('defaultAddress helper → 返回默认项', async () => {
    const getAddressList = jest.fn(async () => ({
      items: [
        { id: 1, recipient: 'a', is_default: false },
        { id: 2, recipient: 'b', is_default: true },
      ],
    }));
    jest.doMock('@/api/address.js', () => ({ getAddressList }));

    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    await s.loadList();
    const d = s.defaultAddress();
    expect(d.id).toBe(2);
  });

  it('defaultAddress 在 defaultId=null 时返回 null', async () => {
    const { useAddressStore } = await import('@/stores/address.js');
    const s = useAddressStore();
    expect(s.defaultAddress()).toBeNull();
  });
});