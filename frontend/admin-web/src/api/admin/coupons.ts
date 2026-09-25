/**
 * admin-coupons API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW coupons handler 对齐）：
 *   - GET    /api/v1/admin/coupons?type=
 *   - GET    /api/v1/admin/coupons/:id
 *   - POST   /api/v1/admin/coupons
 *   - POST   /api/v1/admin/coupons/:id/disable
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 25
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type CouponType = 'amount_off' | 'discount';

export interface CouponItem {
  id: number;
  name: string;
  type: CouponType;
  value: number;
  valid_until: string;
  disabled?: boolean;
  disabled_at?: string;
}

export interface CouponListResponse {
  code: number;
  data: CouponItem[];
  total: number;
  trace_id?: string;
}

export interface CouponDetailResponse {
  code: number;
  data: CouponItem;
  trace_id?: string;
}

export async function fetchCoupons(
  params?: { type?: string },
): Promise<{ data: CouponItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.type) qs.set('type', params.type);
  const url = `${BASE}/coupons${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchCoupons failed: ${resp.status}`);
  const json = (await resp.json()) as CouponListResponse;
  if (json.code !== 0) throw new Error(`fetchCoupons: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchCouponDetail(id: number): Promise<CouponItem> {
  const resp = await fetch(`${BASE}/coupons/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchCouponDetail failed: ${resp.status}`);
  const json = (await resp.json()) as CouponDetailResponse;
  if (json.code !== 0) throw new Error(`fetchCouponDetail: code=${json.code}`);
  return json.data;
}

export async function createCoupon(body: {
  name: string;
  type: CouponType;
  value: number;
  valid_until: string;
}): Promise<CouponItem> {
  const resp = await fetch(`${BASE}/coupons`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as CouponDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`createCoupon failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`createCoupon: code=${json.code}`);
  return json.data;
}

export async function disableCoupon(id: number): Promise<CouponItem> {
  const resp = await fetch(`${BASE}/coupons/${id}/disable`, {
    method: 'POST',
    credentials: 'include',
    headers: authHeader(),
  });
  const json = (await resp.json().catch(() => null)) as CouponDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`disableCoupon failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`disableCoupon: code=${json.code}`);
  return json.data;
}

export const couponQueryKeys = {
  list: (params?: { type?: string }) => ['coupons', params?.type ?? 'all'] as const,
  detail: (id: number | string) => ['coupon-detail', id] as const,
};