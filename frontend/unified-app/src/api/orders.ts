/**
 * order-service API 客户端（v2 unified-app · :8082）。
 *
 * 端点（v2 后端契约，按角色分组）：
 *   - patient-only：
 *     POST /api/v1/orders                          创建订单
 *     POST /api/v1/orders/{id}/select-escort       选择陪诊师
 *     POST /api/v1/orders/{id}/cancel              取消订单
 *   - escort-only：
 *     POST /api/v1/orders/{id}/confirm-accept      确认接单
 *     POST /api/v1/orders/{id}/reject-accept       拒绝接单
 *     POST /api/v1/orders/{id}/finish              完成订单
 *   - 任意角色：
 *     GET  /api/v1/orders                          列表（按 query 过滤）
 *     GET  /api/v1/orders/{id}                     详情
 *
 * 对应：dev.md §42（v2 multi-role + RoleAuthWithKey 已挂 patient/escort gate）
 *      services/order/internal/handler/order.go（端点 source of truth）
 */
import { request } from './client';

const ORDER_BASE_PATH = '/api/v1/orders';
const ORDER_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_ORDER_BASE_URL) || 'http://127.0.0.1:8082';

// ── 类型定义 ───────────────────────────────────────────────────────

export type OrderStatus =
  | 'pending_escort'
  | 'escort_confirmed'
  | 'in_service'
  | 'completed'
  | 'cancelled';

export interface Order {
  id: number;
  patient_id: number;
  escort_id: number | null;
  hospital_id: number;
  package_id: number | null;
  status: OrderStatus;
  appointment_time: string;
  address: string;
  total_amount: number;
  created_at: string;
  updated_at: string;
}

export interface CreateOrderRequest {
  hospital_id: number;
  package_id?: number;
  appointment_time: string;
  address: string;
  /** 备注 */
  remark?: string;
}

export interface SelectEscortRequest {
  escort_id: number;
}

export interface ListOrdersQuery {
  status?: OrderStatus;
  role?: 'patient' | 'escort';
  page?: number;
  page_size?: number;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/**
 * 创建订单（patient-only）。
 * 后端会发 Kafka OrderCreated 事件 → match-service 候选打分 + 抢单池。
 */
export async function createOrder(req: CreateOrderRequest): Promise<Order> {
  return request({
    url: ORDER_BASE_PATH,
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: ORDER_BASE_URL,
  });
}

/**
 * 选择陪诊师（patient-only）。
 * 通常用于 patient 模式"指定陪诊师"创建后调用。
 */
export async function selectEscort(orderId: number, escortId: number): Promise<Order> {
  return request({
    url: `${ORDER_BASE_PATH}/${orderId}/select-escort`,
    method: 'POST',
    data: { escort_id: escortId } as unknown as Record<string, unknown>,
    baseURL: ORDER_BASE_URL,
  });
}

/**
 * 取消订单（patient-only）。
 * 状态机：pending_escort / escort_confirmed → cancelled
 */
export async function cancelOrder(orderId: number, reason?: string): Promise<Order> {
  return request({
    url: `${ORDER_BASE_PATH}/${orderId}/cancel`,
    method: 'POST',
    data: reason ? { reason } : {},
    baseURL: ORDER_BASE_URL,
  });
}

/**
 * 确认接单（escort-only）。
 * 状态机：pending_escort → escort_confirmed
 */
export async function confirmAccept(orderId: number): Promise<Order> {
  return request({
    url: `${ORDER_BASE_PATH}/${orderId}/confirm-accept`,
    method: 'POST',
    baseURL: ORDER_BASE_URL,
  });
}

/**
 * 拒绝接单（escort-only）。
 */
export async function rejectAccept(orderId: number, reason?: string): Promise<Order> {
  return request({
    url: `${ORDER_BASE_PATH}/${orderId}/reject-accept`,
    method: 'POST',
    data: reason ? { reason } : {},
    baseURL: ORDER_BASE_URL,
  });
}

/**
 * 完成订单（escort-only）。
 * 状态机：in_service → completed
 * 完成会触发 OrderCompletedEvent → wallet T+7 结算 + review 待评价（+24h）
 */
export async function finishOrder(orderId: number): Promise<Order> {
  return request({
    url: `${ORDER_BASE_PATH}/${orderId}/finish`,
    method: 'POST',
    baseURL: ORDER_BASE_URL,
  });
}

/** 列表（按 status / role 过滤）。 */
export async function listOrders(query: ListOrdersQuery = {}): Promise<{ items: Order[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.status) qs.set('status', query.status);
  if (query.role) qs.set('role', query.role);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `${ORDER_BASE_PATH}?${qs}` : ORDER_BASE_PATH;
  return request({ url, baseURL: ORDER_BASE_URL });
}

/** 详情。 */
export async function getOrder(orderId: number): Promise<Order> {
  return request({ url: `${ORDER_BASE_PATH}/${orderId}`, baseURL: ORDER_BASE_URL });
}