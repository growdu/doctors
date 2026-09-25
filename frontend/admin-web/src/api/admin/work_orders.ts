/**
 * admin-work-orders API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW work_orders handler 对齐）：
 *   - GET    /api/v1/admin/work-orders?status=&category=&priority=
 *   - GET    /api/v1/admin/work-orders/:id
 *   - POST   /api/v1/admin/work-orders              → 创建工单
 *   - POST   /api/v1/admin/work-orders/:id/assign   → 分配给客服
 *   - POST   /api/v1/admin/work-orders/:id/close    → 关闭工单
 *
 * 设计：
 *   - fetch + Bearer token；
 *   - list 支持 status / category / priority 三维过滤；
 *   - mutate 后由调用方 invalidate ['work-orders', filter]。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 15
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type WorkOrderCategory = 'complaint' | 'appeal' | 'inquiry';
export type WorkOrderPriority = 'low' | 'medium' | 'high';
export type WorkOrderStatus = 'open' | 'in_progress' | 'closed';

export interface WorkOrderItem {
  id: number;
  category: WorkOrderCategory;
  subject: string;
  priority: WorkOrderPriority;
  status: WorkOrderStatus;
  created_at: string;
  assignee?: string | null;
  resolution?: string | null;
}

export interface WorkOrderListResponse {
  code: number;
  data: WorkOrderItem[];
  total: number;
  trace_id?: string;
}

export interface WorkOrderDetailResponse {
  code: number;
  data: WorkOrderItem;
  trace_id?: string;
}

export async function fetchWorkOrders(
  params?: { status?: string; category?: string; priority?: string },
): Promise<{ data: WorkOrderItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set('status', params.status);
  if (params?.category) qs.set('category', params.category);
  if (params?.priority) qs.set('priority', params.priority);
  const url = `${BASE}/work-orders${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchWorkOrders failed: ${resp.status}`);
  const json = (await resp.json()) as WorkOrderListResponse;
  if (json.code !== 0) throw new Error(`fetchWorkOrders: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchWorkOrderDetail(id: number): Promise<WorkOrderItem> {
  const resp = await fetch(`${BASE}/work-orders/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchWorkOrderDetail failed: ${resp.status}`);
  const json = (await resp.json()) as WorkOrderDetailResponse;
  if (json.code !== 0) throw new Error(`fetchWorkOrderDetail: code=${json.code}`);
  return json.data;
}

export async function createWorkOrder(body: {
  category: WorkOrderCategory;
  subject: string;
  priority: WorkOrderPriority;
}): Promise<WorkOrderItem> {
  const resp = await fetch(`${BASE}/work-orders`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as WorkOrderDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`createWorkOrder failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`createWorkOrder: code=${json.code}`);
  return json.data;
}

export async function assignWorkOrder(
  id: number,
  assignee: string,
): Promise<WorkOrderItem> {
  const resp = await fetch(`${BASE}/work-orders/${id}/assign`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ assignee }),
  });
  const json = (await resp.json().catch(() => null)) as WorkOrderDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`assignWorkOrder failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`assignWorkOrder: code=${json.code}`);
  return json.data;
}

export async function closeWorkOrder(
  id: number,
  resolution: string,
): Promise<WorkOrderItem> {
  const resp = await fetch(`${BASE}/work-orders/${id}/close`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ resolution }),
  });
  const json = (await resp.json().catch(() => null)) as WorkOrderDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`closeWorkOrder failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`closeWorkOrder: code=${json.code}`);
  return json.data;
}

export const workOrderQueryKeys = {
  list: (params?: { status?: string; category?: string; priority?: string }) =>
    [
      'work-orders',
      params?.status ?? 'all',
      params?.category ?? 'all',
      params?.priority ?? 'all',
    ] as const,
  detail: (id: number | string) => ['work-order', id] as const,
};