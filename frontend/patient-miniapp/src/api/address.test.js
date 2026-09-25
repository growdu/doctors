// src/api/address.test.js
//
// address api 单测 —— 验证 5 个端点 URL + method + body + 缺参兜底。
//
// 测试策略：与 hospital.test.js 同款 jest.doMock('../../utils/request.js')。

import { jest } from '@jest/globals';

describe('api/address - getAddressList', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 request GET /addresses 并透传响应', async () => {
    const resp = {
      items: [
        {
          id: 1,
          user_id: 7,
          recipient: '张三',
          phone: '13800138000',
          detail: '北京市东城区某街道 1 号',
          is_default: true,
          created_at: '2026-09-24T10:00:00Z',
        },
      ],
    };
    const requestMock = jest.fn(async () => resp);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getAddressList } = await import('@/api/address.js');
    const r = await getAddressList();

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/addresses', method: 'GET' }),
    );
    expect(r.items[0].recipient).toBe('张三');
  });

  it('getAddressList 兼容直返数组形态', async () => {
    const requestMock = jest.fn(async () => []);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { getAddressList } = await import('@/api/address.js');
    const r = await getAddressList();
    expect(r).toEqual([]);
  });
});

describe('api/address - addAddress', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('调 request POST /addresses + 透传完整 body', async () => {
    const created = {
      id: 2,
      user_id: 7,
      recipient: '李四',
      phone: '13900139000',
      detail: '北京市朝阳区某街道 2 号',
      lat: 39.92,
      lng: 116.43,
      is_default: false,
    };
    const requestMock = jest.fn(async () => created);
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { addAddress } = await import('@/api/address.js');
    const r = await addAddress({
      recipient: '李四',
      phone: '13900139000',
      detail: '北京市朝阳区某街道 2 号',
      lat: 39.92,
      lng: 116.43,
      is_default: false,
    });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/addresses',
        method: 'POST',
        data: expect.objectContaining({
          recipient: '李四',
          phone: '13900139000',
          detail: '北京市朝阳区某街道 2 号',
          lat: 39.92,
          lng: 116.43,
          is_default: false,
        }),
      }),
    );
    expect(r.id).toBe(2);
  });

  it('addAddress 缺 recipient / phone / detail → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { addAddress } = await import('@/api/address.js');
    await expect(addAddress(null)).rejects.toThrow(/required/);
    await expect(addAddress({})).rejects.toThrow(/required/);
    await expect(addAddress({ recipient: 'x' })).rejects.toThrow(/required/);
    await expect(addAddress({ recipient: 'x', phone: 'y' })).rejects.toThrow(/required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});

describe('api/address - updateAddress / deleteAddress / setDefaultAddress', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  it('updateAddress 调 PUT /addresses/:id + 透传 body', async () => {
    const requestMock = jest.fn(async () => ({ id: 1, recipient: '新名' }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { updateAddress } = await import('@/api/address.js');
    await updateAddress(1, { recipient: '新名', phone: '13800138000', detail: 'addr' });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/addresses/1',
        method: 'PUT',
        data: expect.objectContaining({ recipient: '新名' }),
      }),
    );
  });

  it('updateAddress 缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { updateAddress } = await import('@/api/address.js');
    await expect(updateAddress(undefined, {})).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('deleteAddress 调 DELETE /addresses/:id', async () => {
    const requestMock = jest.fn(async () => ({ id: 1, deleted: true }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { deleteAddress } = await import('@/api/address.js');
    await deleteAddress(1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/addresses/1', method: 'DELETE' }),
    );
  });

  it('deleteAddress 缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { deleteAddress } = await import('@/api/address.js');
    await expect(deleteAddress(undefined)).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });

  it('setDefaultAddress 调 PUT /addresses/:id/default', async () => {
    const requestMock = jest.fn(async () => ({ id: 1, is_default: true }));
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { setDefaultAddress } = await import('@/api/address.js');
    await setDefaultAddress(1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({ url: '/addresses/1/default', method: 'PUT' }),
    );
  });

  it('setDefaultAddress 缺 id → 同步 reject', async () => {
    const requestMock = jest.fn();
    jest.doMock('../../utils/request.js', () => ({ request: requestMock }));

    const { setDefaultAddress } = await import('@/api/address.js');
    await expect(setDefaultAddress(undefined)).rejects.toThrow(/id is required/);
    expect(requestMock).not.toHaveBeenCalled();
  });
});