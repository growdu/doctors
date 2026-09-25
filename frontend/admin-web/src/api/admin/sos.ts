/**
 * admin-sos API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW sos handler 对齐）：
 *   - GET    /api/v1/admin/sos?status=open|closed
 *   - GET    /api/v1/admin/sos/:id
 *   - POST   /api/v1/admin/sos/:id/resolve
 *   - POST   /api/v1/admin/sos/:id/escalate
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 18
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type SosStatus = 'open' | 'closed';

export interface SosItem {
  id: number;
  order_id: number;
  patient_name: string;
  escort_name: string;
  location: string;
  contact: string;
  status: SosStatus;
  created_at: string;
  resolution_note?: string | null;
  escalate_level?: string;
  escalate_reason?: string | null;
  escalated_at?: string;
}

export interface SosListResponse {
  code: number;
  data: SosItem[];
  total: number;
  trace_id?: string;
}

export interface SosDetailResponse {
  code: number;
  data: SosItem;
  trace_id?: string;
}

export async function fetchSosAlerts(
  params?: { status?: string },
): Promise<{ data: SosItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set('status', params.status);
  const url = `${BASE}/sos${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchSosAlerts failed: ${resp.status}`);
  const json = (await resp.json()) as SosListResponse;
  if (json.code !== 0) throw new Error(`fetchSosAlerts: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchSosDetail(id: number): Promise<SosItem> {
  const resp = await fetch(`${BASE}/sos/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchSosDetail failed: ${resp.status}`);
  const json = (await resp.json()) as SosDetailResponse;
  if (json.code !== 0) throw new Error(`fetchSosDetail: code=${json.code}`);
  return json.data;
}

export async function resolveSos(id: number, note?: string): Promise<SosItem> {
  const resp = await fetch(`${BASE}/sos/${id}/resolve`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ note }),
  });
  const json = (await resp.json().catch(() => null)) as SosDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`resolveSos failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`resolveSos: code=${json.code}`);
  return json.data;
}

export async function escalateSos(
  id: number,
  body: { level: string; reason?: string },
): Promise<SosItem> {
  const resp = await fetch(`${BASE}/sos/${id}/escalate`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as SosDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`escalateSos failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`escalateSos: code=${json.code}`);
  return json.data;
}

export const sosQueryKeys = {
  list: (params?: { status?: string }) => ['sos', params?.status ?? 'all'] as const,
  detail: (id: number | string) => ['sos-detail', id] as const,
};