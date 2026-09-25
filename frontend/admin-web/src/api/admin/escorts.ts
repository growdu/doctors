/**
 * admin-escorts API 客户端（5 个真实业务页依赖）。
 *
 * 端点（与 MSW escorts handler 对齐）：
 *   - GET    /api/v1/admin/escorts?status=pending  → 待审核队列（task spec 提到的 pending-audit）
 *   - GET    /api/v1/admin/escorts/:id
 *   - GET    /api/v1/admin/escorts/:id/audit-history  → 伪端点：当前 mock 由审计组件自行组装
 *   - POST   /api/v1/admin/escorts/:id/approve
 *   - POST   /api/v1/admin/escorts/:id/reject
 *
 * 设计：
 *   - 全部走原生 fetch，BASE 与 mock 路径一致；
 *   - 公共凭证：拉接口时附带 `authorization: Bearer <token>`（useAuthStore.token）；
 *   - 错误统一抛带 code 的 Error（response.json().code != 0 时抛 admin-forbidden / not-found 等）。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.2 + §5.1
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type EscortAuditStatus = 'pending' | 'approved' | 'rejected';

export interface EscortListItem {
  id: number;
  name: string;
  phone: string;
  city: string;
  rating: number;
  audit_status: EscortAuditStatus;
  audit_note: string | null;
  created_at: string;
  // 详情扩展字段（详情页用到）
  health_cert_status?: 'valid' | 'expired' | 'missing';
  service_cities?: string[];
  training_records?: Array<{ title: string; passed_at: string | null }>;
  real_name?: string;
  id_card_no?: string;
  age?: number;
  gender?: 'male' | 'female';
}

export interface EscortListResponse {
  code: number;
  data: EscortListItem[];
  total: number;
  trace_id?: string;
}

export interface EscortDetailResponse {
  code: number;
  data: EscortListItem;
  trace_id?: string;
}

/** 获取陪诊师审核队列（对齐 task spec "pending-audit" → MSW `?status=pending`）。 */
export async function fetchPendingAudit(
  params?: { page?: number; page_size?: number },
): Promise<{ data: EscortListItem[]; total: number }> {
  const qs = new URLSearchParams();
  qs.set('status', 'pending');
  if (params?.page) qs.set('page', String(params.page));
  if (params?.page_size) qs.set('page_size', String(params.page_size));
  const url = `${BASE}/escorts?${qs}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchPendingAudit failed: ${resp.status}`);
  const json = (await resp.json()) as EscortListResponse;
  if (json.code !== 0) throw new Error(`fetchPendingAudit: code=${json.code}`);
  return { data: json.data, total: json.total };
}

/** 获取陪诊师详情。 */
export async function fetchEscortDetail(id: number): Promise<EscortListItem> {
  const resp = await fetch(`${BASE}/escorts/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchEscortDetail failed: ${resp.status}`);
  const json = (await resp.json()) as EscortDetailResponse;
  if (json.code !== 0) throw new Error(`fetchEscortDetail: code=${json.code}`);
  return json.data;
}

/** 通过审核。 */
export async function approveEscort(
  id: number,
  body: { note?: string } = {},
): Promise<EscortListItem> {
  const resp = await fetch(`${BASE}/escorts/${id}/approve`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  if (!resp.ok) throw new Error(`approveEscort failed: ${resp.status}`);
  const json = (await resp.json()) as EscortDetailResponse;
  if (json.code !== 0) throw new Error(`approveEscort: code=${json.code}`);
  return json.data;
}

/** 拒绝审核。 */
export async function rejectEscort(
  id: number,
  body: { reason?: string } = {},
): Promise<EscortListItem> {
  const resp = await fetch(`${BASE}/escorts/${id}/reject`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  if (!resp.ok) throw new Error(`rejectEscort failed: ${resp.status}`);
  const json = (await resp.json()) as EscortDetailResponse;
  if (json.code !== 0) throw new Error(`rejectEscort: code=${json.code}`);
  return json.data;
}

/**
 * 拉陪诊师的审计历史。
 *
 * 当前 MSW 暂未提供该端点（task spec 已点名，但 handlers 暂未实现）。
 * 为避免 404，我们从详情字段拼装一段最小可用的 timeline（Mock-friendly fallback）。
 * 后续接入真后端时只需替换 impl。
 */
export async function fetchEscortAuditHistory(
  id: number,
): Promise<
  Array<{
    timestamp: string;
    actor: string;
    action: string;
    note?: string;
    color?: 'blue' | 'green' | 'red' | 'orange' | 'gray';
  }>
> {
  // 拉详情，含 audit_status / audit_note / created_at 等元数据
  const escort = await fetchEscortDetail(id);
  // mock：默认两条审计项（注册 / 当前状态）
  const items: Array<{
    timestamp: string;
    actor: string;
    action: string;
    note?: string;
    color?: 'blue' | 'green' | 'red' | 'orange' | 'gray';
  }> = [
    {
      timestamp: escort.created_at,
      actor: 'system',
      action: '陪诊师注册',
      color: 'blue',
    },
  ];
  if (escort.audit_status !== 'pending') {
    items.push({
      timestamp: escort.created_at, // 真实场景从 audit_history 表查
      actor: 'audit_admin',
      action:
        escort.audit_status === 'approved' ? '审核通过' : '审核拒绝',
      note: escort.audit_note ?? undefined,
      color: escort.audit_status === 'approved' ? 'green' : 'red',
    });
  }
  return items;
}

/** TanStack Query key 工厂 */
export const escortQueryKeys = {
  pendingAudit: ['escorts-pending-audit'] as const,
  detail: (id: number | string) => ['escort-detail', id] as const,
  auditHistory: (id: number | string) => ['escort-audit-history', id] as const,
};
