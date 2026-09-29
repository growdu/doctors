/**
 * sos-service API 客户端（v2 unified-app · :8087）。
 *
 * 端点：
 *   - POST /api/v1/sos                   一键紧急联系（patient + escort）
 *   - GET  /api/v1/sos                   SOS 工单列表（按 status / role 过滤）
 *   - GET  /api/v1/sos/{id}              SOS 工单详情
 *   - POST /api/v1/sos/{id}/resolve      处置完成（admin / 操作员）
 *
 * 对应：services/sos/internal/handler/handler.go
 */
import { request } from './client';

const SOS_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_SOS_BASE_URL) || 'http://127.0.0.1:8087';

// ── 类型定义 ───────────────────────────────────────────────────────

export type SosStatus = 'pending' | 'in_progress' | 'resolved' | 'closed';

export interface Sos {
  id: number;
  /** 触发人 user_id（patient 或 escort） */
  trigger_user_id: number;
  /** 关联订单 ID（如有） */
  order_id: number | null;
  /** 触发原因描述 */
  reason: string;
  /** 当前位置 */
  location: {
    lat: number;
    lng: number;
    address: string;
  } | null;
  /** 联系手机号（脱敏） */
  contact_phone_masked: string;
  status: SosStatus;
  resolved_at: string | null;
  created_at: string;
}

export interface RaiseSosRequest {
  order_id?: number;
  reason: string;
  location?: {
    lat: number;
    lng: number;
    address: string;
  };
}

// ── 端点函数 ───────────────────────────────────────────────────────

/** 触发 SOS。 */
export async function raiseSos(req: RaiseSosRequest): Promise<Sos> {
  return request({
    url: '/api/v1/sos',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: SOS_BASE_URL,
  });
}

/** SOS 列表。 */
export async function listSos(query: { status?: SosStatus; page?: number; page_size?: number } = {}): Promise<{ items: Sos[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.status) qs.set('status', query.status);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/sos?${qs}` : '/api/v1/sos';
  return request({ url, baseURL: SOS_BASE_URL });
}

/** SOS 详情。 */
export async function getSos(id: number): Promise<Sos> {
  return request({ url: `/api/v1/sos/${id}`, baseURL: SOS_BASE_URL });
}

/** 处置完成。 */
export async function resolveSos(id: number, remark: string): Promise<Sos> {
  return request({
    url: `/api/v1/sos/${id}/resolve`,
    method: 'POST',
    data: { remark } as unknown as Record<string, unknown>,
    baseURL: SOS_BASE_URL,
  });
}