/**
 * admin-packages API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW packages handler 对齐）：
 *   - GET    /api/v1/admin/packages?status=
 *   - GET    /api/v1/admin/packages/:id
 *   - POST   /api/v1/admin/packages
 *   - PATCH  /api/v1/admin/packages/:id   → 含上下架 status 切换
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 27
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type PackageDuration = 'half_day' | 'full_day';
export type PackageStatus = 'on' | 'off';

export interface PackageItem {
  id: number;
  name: string;
  price: number;
  duration: PackageDuration;
  status: PackageStatus;
}

export interface PackageListResponse {
  code: number;
  data: PackageItem[];
  total: number;
  trace_id?: string;
}

export interface PackageDetailResponse {
  code: number;
  data: PackageItem;
  trace_id?: string;
}

export async function fetchPackages(
  params?: { status?: string },
): Promise<{ data: PackageItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set('status', params.status);
  const url = `${BASE}/packages${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchPackages failed: ${resp.status}`);
  const json = (await resp.json()) as PackageListResponse;
  if (json.code !== 0) throw new Error(`fetchPackages: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchPackageDetail(id: number): Promise<PackageItem> {
  const resp = await fetch(`${BASE}/packages/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchPackageDetail failed: ${resp.status}`);
  const json = (await resp.json()) as PackageDetailResponse;
  if (json.code !== 0) throw new Error(`fetchPackageDetail: code=${json.code}`);
  return json.data;
}

export async function createPackage(body: {
  name: string;
  price: number;
  duration: PackageDuration;
}): Promise<PackageItem> {
  const resp = await fetch(`${BASE}/packages`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as PackageDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`createPackage failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`createPackage: code=${json.code}`);
  return json.data;
}

export async function updatePackage(
  id: number,
  body: { name?: string; price?: number; duration?: PackageDuration; status?: PackageStatus },
): Promise<PackageItem> {
  const resp = await fetch(`${BASE}/packages/${id}`, {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as PackageDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`updatePackage failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`updatePackage: code=${json.code}`);
  return json.data;
}

export const packageQueryKeys = {
  list: (params?: { status?: string }) => ['packages', params?.status ?? 'all'] as const,
  detail: (id: number | string) => ['package-detail', id] as const,
};