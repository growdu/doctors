/**
 * user-service API 客户端（v2 unified-app · :8088）。
 *
 * 模块：
 *   - 用户画像：me / nickname / avatar
 *   - 地址管理：list / create / update / setDefault / delete
 *   - 优惠券：list / claim / mine / use
 *   - 医院：list / detail / byHospital（套餐）
 *   - 套餐：detail
 *   - 虚拟号码：allocate / get
 *
 * 对应：services/user/internal/{handler,address,coupon,hospital,pkg,virtualnumber}/*.go
 */
import { request } from './client';

const USER_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_USER_BASE_URL) || 'http://127.0.0.1:8088';

// ── 类型定义 ───────────────────────────────────────────────────────

export interface UserProfile {
  id: number;
  phone: string;
  nickname: string;
  avatar_url: string | null;
  real_name_verified: boolean;
  active_role: string;
  roles: string[];
}

export interface Address {
  id: number;
  user_id: number;
  name: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  detail: string;
  is_default: boolean;
}

export interface CreateAddressRequest {
  name: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  detail: string;
  is_default?: boolean;
}

export interface Coupon {
  id: number;
  title: string;
  /** 优惠金额（分） */
  amount: number;
  /** 最低消费（分） */
  min_amount: number;
  /** 有效期 */
  valid_from: string;
  valid_to: string;
  status: 'available' | 'claimed' | 'used' | 'expired';
}

export interface Hospital {
  id: number;
  name: string;
  city: string;
  address: string;
  /** 联系电话 */
  phone: string | null;
  /** 等级：三级甲等 / 二级甲等 等 */
  level: string;
}

export interface Package {
  id: number;
  hospital_id: number;
  title: string;
  description: string;
  /** 价格（分） */
  price: number;
  /** 服务时长（分钟） */
  duration_minutes: number;
  cover_url: string | null;
}

export interface VirtualNumber {
  id: number;
  number: string;
  expire_at: string;
}

// ── 用户画像 ───────────────────────────────────────────────────────

export async function updateNickname(userId: number, nickname: string): Promise<UserProfile> {
  return request({
    url: `/api/v1/users/${userId}/nickname`,
    method: 'PATCH',
    data: { nickname } as unknown as Record<string, unknown>,
    baseURL: USER_BASE_URL,
  });
}

export async function updateAvatar(userId: number, avatarUrl: string): Promise<UserProfile> {
  return request({
    url: `/api/v1/users/${userId}/avatar`,
    method: 'PATCH',
    data: { avatar_url: avatarUrl } as unknown as Record<string, unknown>,
    baseURL: USER_BASE_URL,
  });
}

// ── 地址 ───────────────────────────────────────────────────────────

export async function listAddresses(): Promise<{ items: Address[] }> {
  return request({ url: '/api/v1/address', baseURL: USER_BASE_URL });
}

export async function createAddress(req: CreateAddressRequest): Promise<Address> {
  return request({
    url: '/api/v1/address',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: USER_BASE_URL,
  });
}

export async function updateAddress(id: number, req: Partial<CreateAddressRequest>): Promise<Address> {
  return request({
    url: `/api/v1/address/${id}`,
    method: 'PUT',
    data: req as unknown as Record<string, unknown>,
    baseURL: USER_BASE_URL,
  });
}

export async function setDefaultAddress(id: number): Promise<Address> {
  return request({
    url: `/api/v1/address/${id}/default`,
    method: 'PUT',
    baseURL: USER_BASE_URL,
  });
}

export async function deleteAddress(id: number): Promise<{ ok: true }> {
  return request({
    url: `/api/v1/address/${id}`,
    method: 'DELETE',
    baseURL: USER_BASE_URL,
  });
}

// ── 优惠券 ─────────────────────────────────────────────────────────

export async function listAvailableCoupons(): Promise<{ items: Coupon[] }> {
  return request({ url: '/api/v1/coupons', baseURL: USER_BASE_URL });
}

export async function getCoupon(id: number): Promise<Coupon> {
  return request({ url: `/api/v1/coupons/${id}`, baseURL: USER_BASE_URL });
}

export async function claimCoupon(id: number): Promise<Coupon> {
  return request({
    url: `/api/v1/coupons/${id}/claim`,
    method: 'POST',
    baseURL: USER_BASE_URL,
  });
}

export async function listMyCoupons(): Promise<{ items: Coupon[] }> {
  return request({ url: '/api/v1/me/coupons', baseURL: USER_BASE_URL });
}

export async function useMyCoupon(id: number, orderId: number): Promise<{ ok: true }> {
  return request({
    url: `/api/v1/me/coupons/${id}/use`,
    method: 'POST',
    data: { order_id: orderId } as unknown as Record<string, unknown>,
    baseURL: USER_BASE_URL,
  });
}

// ── 医院 + 套餐 ────────────────────────────────────────────────────

export async function listHospitals(query: { city?: string; keyword?: string; page?: number; page_size?: number } = {}): Promise<{ items: Hospital[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.city) qs.set('city', query.city);
  if (query.keyword) qs.set('keyword', query.keyword);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/hospitals?${qs}` : '/api/v1/hospitals';
  return request({ url, baseURL: USER_BASE_URL });
}

export async function getHospital(id: number): Promise<Hospital> {
  return request({ url: `/api/v1/hospitals/${id}`, baseURL: USER_BASE_URL });
}

export async function listPackagesByHospital(hospitalId: number): Promise<{ items: Package[] }> {
  return request({ url: `/api/v1/hospitals/${hospitalId}/packages`, baseURL: USER_BASE_URL });
}

export async function getPackage(id: number): Promise<Package> {
  return request({ url: `/api/v1/packages/${id}`, baseURL: USER_BASE_URL });
}

// ── 虚拟号码 ───────────────────────────────────────────────────────

export async function allocateVirtualNumber(orderId: number): Promise<VirtualNumber> {
  return request({
    url: '/api/v1/virtual-numbers/allocate',
    method: 'POST',
    data: { order_id: orderId } as unknown as Record<string, unknown>,
    baseURL: USER_BASE_URL,
  });
}

export async function getVirtualNumber(id: number): Promise<VirtualNumber> {
  return request({ url: `/api/v1/virtual-numbers/${id}`, baseURL: USER_BASE_URL });
}