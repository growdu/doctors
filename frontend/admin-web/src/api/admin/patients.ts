/**
 * admin-patients API 客户端（5 个真实业务页依赖）。
 *
 * 端点（与 MSW patients handler 对齐）：
 *   - GET    /api/v1/admin/patients?keyword=...
 *   - GET    /api/v1/admin/patients/:id
 *   - POST   /api/v1/admin/patients/:id/ban    → body: { reason }
 *   - POST   /api/v1/admin/patients/:id/unban
 *
 * 设计：
 *   - 走原生 fetch + Bearer token；
 *   - list 端点支持 keyword 模糊搜索（昵称 / 手机号）。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.2
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type PatientVerifyStatus = 'verified' | 'unverified' | 'pending';
export type PatientStatus = 'active' | 'banned';

export interface PatientListItem {
  id: number;
  name: string;
  phone: string;
  registered_at: string;
  order_count: number;
  refund_count: number;
  /** v3 增强字段（详情 / 列表共用；列表只校验存在） */
  verify_status?: PatientVerifyStatus;
  status?: PatientStatus;
  ban_reason?: string | null;
  avatar_url?: string | null;
  // 钱包余额（仅详情）
  wallet_balance?: number;
  wallet_frozen?: number;
}

export interface PatientListResponse {
  code: number;
  data: PatientListItem[];
  total: number;
  trace_id?: string;
}

export interface PatientDetailResponse {
  code: number;
  data: PatientListItem;
  trace_id?: string;
}

/** 关键词搜索患者列表。 */
export async function fetchPatients(
  params?: { keyword?: string; page?: number; page_size?: number },
): Promise<{ data: PatientListItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.keyword) qs.set('keyword', params.keyword);
  if (params?.page) qs.set('page', String(params.page));
  if (params?.page_size) qs.set('page_size', String(params.page_size));
  const url = `${BASE}/patients${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchPatients failed: ${resp.status}`);
  const json = (await resp.json()) as PatientListResponse;
  if (json.code !== 0) throw new Error(`fetchPatients: code=${json.code}`);
  return { data: json.data, total: json.total };
}

/** 患者详情。 */
export async function fetchPatientDetail(id: number): Promise<PatientListItem> {
  const resp = await fetch(`${BASE}/patients/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchPatientDetail failed: ${resp.status}`);
  const json = (await resp.json()) as PatientDetailResponse;
  if (json.code !== 0) throw new Error(`fetchPatientDetail: code=${json.code}`);
  return json.data;
}

/** 封禁患者。reason 必填，否则后端会返回 code=10001。 */
export async function banPatient(
  id: number,
  reason: string,
): Promise<PatientListItem> {
  const resp = await fetch(`${BASE}/patients/${id}/ban`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ reason }),
  });
  const json = (await resp.json().catch(() => null)) as PatientDetailResponse | null;
  if (!resp.ok || !json) {
    throw new Error(`banPatient failed: ${resp.status}`);
  }
  if (json.code !== 0) {
    const err = new Error(`banPatient: code=${json.code}`);
    (err as Error & { code?: number }).code = json.code;
    throw err;
  }
  return json.data;
}

/** 解封患者。 */
export async function unbanPatient(id: number): Promise<PatientListItem> {
  const resp = await fetch(`${BASE}/patients/${id}/unban`, {
    method: 'POST',
    credentials: 'include',
    headers: authHeader(),
  });
  const json = (await resp.json().catch(() => null)) as PatientDetailResponse | null;
  if (!resp.ok || !json) {
    throw new Error(`unbanPatient failed: ${resp.status}`);
  }
  if (json.code !== 0) {
    throw new Error(`unbanPatient: code=${json.code}`);
  }
  return json.data;
}

/** TanStack Query key 工厂 */
export const patientQueryKeys = {
  list: (params?: { keyword?: string }) =>
    ['patients', params?.keyword ?? 'all'] as const,
  detail: (id: number | string) => ['patient-detail', id] as const,
};
