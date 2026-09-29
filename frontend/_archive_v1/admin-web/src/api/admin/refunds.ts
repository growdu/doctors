/**
 * admin-refunds API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW refunds handler 对齐）：
 *   - GET    /api/v1/admin/refunds?status=
 *   - GET    /api/v1/admin/refunds/:id
 *   - POST   /api/v1/admin/refunds/:id/approve
 *   - POST   /api/v1/admin/refunds/:id/reject
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 13
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type RefundStatus = 'pending' | 'approved' | 'rejected';

export interface RefundItem {
  id: number;
  order_id: number;
  patient_name: string;
  amount: number;
  reason: string;
  status: RefundStatus;
  refund_note: string | null;
  created_at: string;
}

export interface RefundListResponse {
  code: number;
  data: RefundItem[];
  total: number;
  trace_id?: string;
}

export interface RefundDetailResponse {
  code: number;
  data: RefundItem;
  trace_id?: string;
}

export async function fetchRefunds(
  params?: { status?: string },
): Promise<{ data: RefundItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set('status', params.status);
  const url = `${BASE}/refunds${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchRefunds failed: ${resp.status}`);
  const json = (await resp.json()) as RefundListResponse;
  if (json.code !== 0) throw new Error(`fetchRefunds: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchRefundDetail(id: number): Promise<RefundItem> {
  const resp = await fetch(`${BASE}/refunds/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchRefundDetail failed: ${resp.status}`);
  const json = (await resp.json()) as RefundDetailResponse;
  if (json.code !== 0) throw new Error(`fetchRefundDetail: code=${json.code}`);
  return json.data;
}

export async function approveRefund(
  id: number,
  note?: string,
): Promise<RefundItem> {
  const resp = await fetch(`${BASE}/refunds/${id}/approve`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ note }),
  });
  const json = (await resp.json().catch(() => null)) as RefundDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`approveRefund failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`approveRefund: code=${json.code}`);
  return json.data;
}

export async function rejectRefund(
  id: number,
  reason?: string,
): Promise<RefundItem> {
  const resp = await fetch(`${BASE}/refunds/${id}/reject`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ reason }),
  });
  const json = (await resp.json().catch(() => null)) as RefundDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`rejectRefund failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`rejectRefund: code=${json.code}`);
  return json.data;
}

export const refundQueryKeys = {
  list: (params?: { status?: string }) => ['refunds', params?.status ?? 'all'] as const,
  detail: (id: number | string) => ['refund-detail', id] as const,
};