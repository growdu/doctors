/**
 * admin-reviews API 客户端（5 个真实业务页依赖）。
 *
 * 端点（与 MSW reviews handler 对齐）：
 *   - GET    /api/v1/admin/reviews?escort_id=&min_rating=
 *   - GET    /api/v1/admin/reviews/:id
 *   - POST   /api/v1/admin/reviews/:id/audit   → body: { result: 'pass' | 'hide', reason? }
 *   - POST   /api/v1/admin/reviews/:id/reply   → body: { reply }
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.2
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type ReviewAuditResult = 'pass' | 'hide';

export interface ReviewListItem {
  id: number;
  order_id: number;
  escort_id: number;
  rating: number; // 1~5
  content: string;
  created_at: string;
  /** v2+ 字段 */
  audit_result?: ReviewAuditResult | null;
  audit_reason?: string | null;
  admin_reply?: string | null;
  /** UI 增强字段（mock 时由种子数据补齐） */
  patient_name?: string;
  escort_name?: string;
  tags?: string[];
}

export interface ReviewListResponse {
  code: number;
  data: ReviewListItem[];
  total: number;
  trace_id?: string;
}

export interface ReviewDetailResponse {
  code: number;
  data: ReviewListItem;
  trace_id?: string;
}

/** 评价列表（按 escort_id / min_rating 过滤）。 */
export async function fetchReviews(
  params?: {
    escort_id?: number;
    min_rating?: number;
    keyword?: string;
    page?: number;
    page_size?: number;
  },
): Promise<{ data: ReviewListItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.escort_id != null) qs.set('escort_id', String(params.escort_id));
  if (params?.min_rating != null) qs.set('min_rating', String(params.min_rating));
  if (params?.keyword) qs.set('keyword', params.keyword);
  if (params?.page) qs.set('page', String(params.page));
  if (params?.page_size) qs.set('page_size', String(params.page_size));
  const url = `${BASE}/reviews${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchReviews failed: ${resp.status}`);
  const json = (await resp.json()) as ReviewListResponse;
  if (json.code !== 0) throw new Error(`fetchReviews: code=${json.code}`);
  return { data: json.data, total: json.total };
}

/** 审核评价：通过 / 隐藏。 */
export async function auditReview(
  id: number,
  body: { result: ReviewAuditResult; reason?: string },
): Promise<ReviewListItem> {
  const resp = await fetch(`${BASE}/reviews/${id}/audit`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as ReviewDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`auditReview failed: ${resp.status}`);
  if (json.code !== 0) {
    const err = new Error(`auditReview: code=${json.code}`);
    (err as Error & { code?: number }).code = json.code;
    throw err;
  }
  return json.data;
}

/** 客服回复。 */
export async function replyReview(
  id: number,
  reply: string,
): Promise<ReviewListItem> {
  const resp = await fetch(`${BASE}/reviews/${id}/reply`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ reply }),
  });
  const json = (await resp.json().catch(() => null)) as ReviewDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`replyReview failed: ${resp.status}`);
  if (json.code !== 0) {
    const err = new Error(`replyReview: code=${json.code}`);
    (err as Error & { code?: number }).code = json.code;
    throw err;
  }
  return json.data;
}

/** TanStack Query key 工厂 */
export const reviewQueryKeys = {
  list: (params?: { escort_id?: number; min_rating?: number }) =>
    ['reviews', params?.escort_id ?? 'all', params?.min_rating ?? 0] as const,
};
