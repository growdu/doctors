/**
 * admin-hospitals API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW hospitals handler 对齐）：
 *   - GET    /api/v1/admin/hospitals?city=&status=
 *   - GET    /api/v1/admin/hospitals/:id
 *   - POST   /api/v1/admin/hospitals
 *   - PATCH  /api/v1/admin/hospitals/:id
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 26
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type HospitalLevel = '三甲' | '三乙' | '二甲';
export type HospitalStatus = 'active' | 'inactive';

export interface HospitalItem {
  id: number;
  name: string;
  city: string;
  level: HospitalLevel;
  status: HospitalStatus;
}

export interface HospitalListResponse {
  code: number;
  data: HospitalItem[];
  total: number;
  trace_id?: string;
}

export interface HospitalDetailResponse {
  code: number;
  data: HospitalItem;
  trace_id?: string;
}

export async function fetchHospitals(
  params?: { city?: string; status?: string },
): Promise<{ data: HospitalItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.city) qs.set('city', params.city);
  if (params?.status) qs.set('status', params.status);
  const url = `${BASE}/hospitals${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchHospitals failed: ${resp.status}`);
  const json = (await resp.json()) as HospitalListResponse;
  if (json.code !== 0) throw new Error(`fetchHospitals: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchHospitalDetail(id: number): Promise<HospitalItem> {
  const resp = await fetch(`${BASE}/hospitals/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchHospitalDetail failed: ${resp.status}`);
  const json = (await resp.json()) as HospitalDetailResponse;
  if (json.code !== 0) throw new Error(`fetchHospitalDetail: code=${json.code}`);
  return json.data;
}

export async function createHospital(body: {
  name: string;
  city: string;
  level: HospitalLevel;
}): Promise<HospitalItem> {
  const resp = await fetch(`${BASE}/hospitals`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as HospitalDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`createHospital failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`createHospital: code=${json.code}`);
  return json.data;
}

export async function updateHospital(
  id: number,
  body: { name?: string; city?: string; level?: HospitalLevel; status?: HospitalStatus },
): Promise<HospitalItem> {
  const resp = await fetch(`${BASE}/hospitals/${id}`, {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as HospitalDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`updateHospital failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`updateHospital: code=${json.code}`);
  return json.data;
}

export const hospitalQueryKeys = {
  list: (params?: { city?: string; status?: string }) =>
    ['hospitals', params?.city ?? 'all', params?.status ?? 'all'] as const,
  detail: (id: number | string) => ['hospital-detail', id] as const,
};