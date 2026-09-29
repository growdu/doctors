/**
 * admin-service API 客户端（v2 unified-app · :8091）。
 *
 * 端点（admin-only，按模块分组）：
 *   - 用户：GET /admin/users
 *   - 订单：GET /admin/orders、POST /admin/orders/{id}/force-cancel
 *   - 陪诊师审核：GET /admin/escorts/pending-audit
 *                    POST /admin/escorts/{id}/approve|reject
 *   - 退款审批：GET /admin/refunds、POST /admin/refunds/{id}/approve|reject
 *   - 工单：GET /admin/work-orders、POST /admin/work-orders
 *   - 账单：GET /admin/billings
 *   - 报表：GET /admin/reports/overview
 *
 * 对应：services/admin/internal/handler/admin.go
 */
import { request } from './client';

const ADMIN_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_ADMIN_BASE_URL) || 'http://127.0.0.1:8091';

// ── 类型定义 ───────────────────────────────────────────────────────

export interface AdminUser {
  id: number;
  phone: string;
  nickname: string;
  active_role: string;
  roles: string[];
  real_name_verified: boolean;
  created_at: string;
}

export interface AdminOrder {
  id: number;
  patient_id: number;
  escort_id: number | null;
  hospital_id: number;
  status: string;
  total_amount: number;
  created_at: string;
}

export interface PendingEscort {
  id: number;
  user_id: number;
  nickname: string;
  phone: string;
  /** 资质材料 URL 列表 */
  qualification_urls: string[];
  /** 提交时间 */
  submitted_at: string;
}

export interface Refund {
  id: number;
  payment_id: number;
  order_id: number;
  amount: number;
  reason: string;
  status: 'pending' | 'approved' | 'rejected';
  created_at: string;
}

export interface WorkOrder {
  id: number;
  /** 工单类型 */
  type: 'complaint' | 'refund' | 'consult' | 'other';
  /** 关联 user_id / order_id */
  ref_id: number | null;
  ref_type: string | null;
  title: string;
  content: string;
  status: 'open' | 'in_progress' | 'closed';
  assignee: number | null;
  created_at: string;
}

export interface Billing {
  id: number;
  order_id: number;
  /** 金额（分） */
  amount: number;
  type: 'income' | 'refund' | 'withdraw';
  created_at: string;
}

export interface ReportOverview {
  /** 总订单数 */
  total_orders: number;
  /** 总金额（分） */
  total_amount: number;
  /** 今日订单 */
  today_orders: number;
  /** 今日 GMV */
  today_amount: number;
  /** 在线陪诊师 */
  online_escorts: number;
}

// ── 端点函数 ───────────────────────────────────────────────────────

// 用户管理
export async function listUsers(query: { role?: string; page?: number; page_size?: number } = {}): Promise<{ items: AdminUser[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.role) qs.set('role', query.role);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/admin/users?${qs}` : '/api/v1/admin/users';
  return request({ url, baseURL: ADMIN_BASE_URL });
}

// 订单管理
export async function listAdminOrders(query: { status?: string; page?: number; page_size?: number } = {}): Promise<{ items: AdminOrder[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.status) qs.set('status', query.status);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/admin/orders?${qs}` : '/api/v1/admin/orders';
  return request({ url, baseURL: ADMIN_BASE_URL });
}

export async function forceCancelOrder(orderId: number, reason: string): Promise<AdminOrder> {
  return request({
    url: `/api/v1/admin/orders/${orderId}/force-cancel`,
    method: 'POST',
    data: { reason } as unknown as Record<string, unknown>,
    baseURL: ADMIN_BASE_URL,
  });
}

// 陪诊师审核
export async function listPendingEscorts(): Promise<{ items: PendingEscort[] }> {
  return request({ url: '/api/v1/admin/escorts/pending-audit', baseURL: ADMIN_BASE_URL });
}

export async function approveEscort(escortId: number): Promise<PendingEscort> {
  return request({
    url: `/api/v1/admin/escorts/${escortId}/approve`,
    method: 'POST',
    baseURL: ADMIN_BASE_URL,
  });
}

export async function rejectEscort(escortId: number, reason: string): Promise<PendingEscort> {
  return request({
    url: `/api/v1/admin/escorts/${escortId}/reject`,
    method: 'POST',
    data: { reason } as unknown as Record<string, unknown>,
    baseURL: ADMIN_BASE_URL,
  });
}

// 退款审批
export async function listRefunds(query: { status?: 'pending' | 'approved' | 'rejected'; page?: number; page_size?: number } = {}): Promise<{ items: Refund[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.status) qs.set('status', query.status);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/admin/refunds?${qs}` : '/api/v1/admin/refunds';
  return request({ url, baseURL: ADMIN_BASE_URL });
}

export async function approveRefund(refundId: number): Promise<Refund> {
  return request({
    url: `/api/v1/admin/refunds/${refundId}/approve`,
    method: 'POST',
    baseURL: ADMIN_BASE_URL,
  });
}

export async function rejectRefund(refundId: number, reason: string): Promise<Refund> {
  return request({
    url: `/api/v1/admin/refunds/${refundId}/reject`,
    method: 'POST',
    data: { reason } as unknown as Record<string, unknown>,
    baseURL: ADMIN_BASE_URL,
  });
}

// 工单
export async function listWorkOrders(query: { status?: 'open' | 'in_progress' | 'closed'; page?: number; page_size?: number } = {}): Promise<{ items: WorkOrder[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.status) qs.set('status', query.status);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/admin/work-orders?${qs}` : '/api/v1/admin/work-orders';
  return request({ url, baseURL: ADMIN_BASE_URL });
}

export async function createWorkOrder(req: { type: WorkOrder['type']; ref_id?: number; ref_type?: string; title: string; content: string }): Promise<WorkOrder> {
  return request({
    url: '/api/v1/admin/work-orders',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: ADMIN_BASE_URL,
  });
}

// 账单 + 报表
export async function listBillings(query: { type?: 'income' | 'refund' | 'withdraw'; page?: number; page_size?: number } = {}): Promise<{ items: Billing[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.type) qs.set('type', query.type);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/admin/billings?${qs}` : '/api/v1/admin/billings';
  return request({ url, baseURL: ADMIN_BASE_URL });
}

export async function getReportOverview(): Promise<ReportOverview> {
  return request({ url: '/api/v1/admin/reports/overview', baseURL: ADMIN_BASE_URL });
}