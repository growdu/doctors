/**
 * api/user 单测（地址 + 优惠券 + 医院 + 套餐 + 虚拟号码 共 19 端点，抽测覆盖）。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  listAddresses,
  createAddress,
  updateAddress,
  setDefaultAddress,
  deleteAddress,
  listAvailableCoupons,
  claimCoupon,
  useMyCoupon,
  listHospitals,
  listPackagesByHospital,
  allocateVirtualNumber,
} from './user';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({ items: [] });
});

describe('api/user · address', () => {
  it('listAddresses GETs /address', async () => {
    await listAddresses();
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/address',
      baseURL: 'http://127.0.0.1:8088',
    });
  });

  it('createAddress POSTs body', async () => {
    await createAddress({ name: '张三', phone: '13800138000', province: '北京市', city: '北京', district: '东城区', detail: 'xx 路' });
    expect(requestSpy.mock.calls[0]![0].method).toBe('POST');
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/address');
  });

  it('updateAddress PUTs /address/:id', async () => {
    await updateAddress(5, { detail: 'yy 路' });
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/address/5',
      method: 'PUT',
      data: { detail: 'yy 路' },
      baseURL: 'http://127.0.0.1:8088',
    });
  });

  it('setDefaultAddress PUTs /address/:id/default', async () => {
    await setDefaultAddress(5);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/address/5/default');
    expect(requestSpy.mock.calls[0]![0].method).toBe('PUT');
  });

  it('deleteAddress DELETEs /address/:id', async () => {
    await deleteAddress(5);
    expect(requestSpy.mock.calls[0]![0].method).toBe('DELETE');
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/address/5');
  });
});

describe('api/user · coupons', () => {
  it('listAvailableCoupons GETs /coupons', async () => {
    await listAvailableCoupons();
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/coupons');
  });

  it('claimCoupon POSTs to /coupons/:id/claim', async () => {
    await claimCoupon(7);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/coupons/7/claim');
  });

  it('useMyCoupon POSTs order_id', async () => {
    await useMyCoupon(7, 42);
    expect(requestSpy.mock.calls[0]![0].data).toEqual({ order_id: 42 });
  });
});

describe('api/user · hospitals + packages', () => {
  it('listHospitals with keyword builds query', async () => {
    await listHospitals({ city: '北京', keyword: '协和' });
    const url = requestSpy.mock.calls[0]![0].url as string;
    expect(url).toContain('city=');
    expect(url).toContain('keyword=');
  });

  it('listPackagesByHospital GETs /hospitals/:id/packages', async () => {
    await listPackagesByHospital(5);
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/hospitals/5/packages');
  });
});

describe('api/user · virtual number', () => {
  it('allocateVirtualNumber POSTs order_id', async () => {
    await allocateVirtualNumber(42);
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/virtual-numbers/allocate',
      method: 'POST',
      data: { order_id: 42 },
      baseURL: 'http://127.0.0.1:8088',
    });
  });
});