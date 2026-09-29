/**
 * addressStore 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useAddressStore } from './address';
import type { Address } from '@/api/user';

vi.mock('@/api/user', () => ({
  listAddresses: vi.fn(),
  createAddress: vi.fn(),
  updateAddress: vi.fn(),
  deleteAddress: vi.fn(),
  setDefaultAddress: vi.fn(),
}));

import * as apiUser from '@/api/user';

const fakeAddr: Address = {
  id: 1,
  user_id: 100,
  name: '张三',
  phone: '13800138000',
  province: '北京市',
  city: '北京',
  district: '东城区',
  detail: 'xx 路 1 号',
  is_default: true,
};

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('addressStore · initial state', () => {
  it('starts empty', () => {
    const s = useAddressStore();
    expect(s.list).toEqual([]);
    expect(s.count).toBe(0);
    expect(s.atLimit).toBe(false);
    expect(s.defaultAddress).toBeNull();
  });
});

describe('addressStore · fetchList', () => {
  it('loads addresses', async () => {
    vi.mocked(apiUser.listAddresses).mockResolvedValue({ items: [fakeAddr] });
    const s = useAddressStore();
    const r = await s.fetchList();
    expect(r).toEqual([fakeAddr]);
    expect(s.list).toEqual([fakeAddr]);
    expect(s.count).toBe(1);
    expect(s.defaultAddress).toEqual(fakeAddr);
  });

  it('returns empty on error', async () => {
    vi.mocked(apiUser.listAddresses).mockRejectedValue(new Error('网络异常'));
    const s = useAddressStore();
    const r = await s.fetchList();
    expect(r).toEqual([]);
    expect(s.error).toBe('网络异常');
  });
});

describe('addressStore · create', () => {
  it('prepends new address to list', async () => {
    vi.mocked(apiUser.createAddress).mockResolvedValue({ ...fakeAddr, id: 2, is_default: false });
    const s = useAddressStore();
    await s.fetchList();
    s.list = [{ ...fakeAddr, id: 1, is_default: true }];
    const r = await s.create({ name: '李四', phone: '13900139000', province: '上海市', city: '上海', district: '黄浦区', detail: 'yy 路 2 号' });
    expect(r.id).toBe(2);
    expect(s.list[0]?.id).toBe(2);
  });

  it('clears other defaults when new is default', async () => {
    vi.mocked(apiUser.createAddress).mockResolvedValue({ ...fakeAddr, id: 2, is_default: true });
    const s = useAddressStore();
    s.list = [{ ...fakeAddr, id: 1, is_default: true }];
    await s.create({ name: '李四', phone: '13900139000', province: '上海市', city: '上海', district: '黄浦区', detail: 'yy', is_default: true });
    expect(s.list.find((a) => a.id === 1)?.is_default).toBe(false);
    expect(s.list.find((a) => a.id === 2)?.is_default).toBe(true);
  });
});

describe('addressStore · update / remove / setDefault', () => {
  it('update replaces item', async () => {
    vi.mocked(apiUser.updateAddress).mockResolvedValue({ ...fakeAddr, detail: '新地址' });
    const s = useAddressStore();
    s.list = [{ ...fakeAddr }];
    await s.update(1, { detail: '新地址' });
    expect(s.list[0]?.detail).toBe('新地址');
  });

  it('remove filters item out', async () => {
    vi.mocked(apiUser.deleteAddress).mockResolvedValue({ ok: true });
    const s = useAddressStore();
    s.list = [{ ...fakeAddr, id: 1 }, { ...fakeAddr, id: 2 }];
    await s.remove(1);
    expect(s.list).toHaveLength(1);
    expect(s.list[0]?.id).toBe(2);
  });

  it('setDefault marks target and clears others', async () => {
    vi.mocked(apiUser.setDefaultAddress).mockResolvedValue({ ...fakeAddr, id: 2, is_default: true });
    const s = useAddressStore();
    s.list = [
      { ...fakeAddr, id: 1, is_default: true },
      { ...fakeAddr, id: 2, is_default: false },
    ];
    await s.setDefault(2);
    expect(s.list.find((a) => a.id === 1)?.is_default).toBe(false);
    expect(s.list.find((a) => a.id === 2)?.is_default).toBe(true);
  });
});

describe('addressStore · helpers', () => {
  it('atLimit becomes true when 5+ addresses', () => {
    const s = useAddressStore();
    s.list = Array.from({ length: 5 }, (_, i) => ({ ...fakeAddr, id: i + 1 }));
    expect(s.atLimit).toBe(true);
  });

  it('findById returns matching address', () => {
    const s = useAddressStore();
    s.list = [{ ...fakeAddr, id: 5 }];
    expect(s.findById(5)?.id).toBe(5);
    expect(s.findById(999)).toBeUndefined();
  });

  it('clearError resets error', () => {
    const s = useAddressStore();
    s.error = 'something';
    s.clearError();
    expect(s.error).toBeNull();
  });
});