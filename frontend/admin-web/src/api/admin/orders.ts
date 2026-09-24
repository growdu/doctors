/**
 * admin-orders API 客户端（v2 增量适配）。
 *
 * 真实实现应使用 v1 引入的 fetch 包装（参见 docs/superpowers/specs/
 * 2026-09-24-admin-web-design.md §3.2）。当前骨架未建 fetch client，
 * 这里给最小可用 stub（直接调 fetch + 路径硬编码）。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 5 (后端 contracts.yaml)
 */
import type {
  OrderDetail,
  OrderDetailResponse,
  OrderListItem,
  OrderListResponse,
} from '@/types/generated';

const BASE = '/api/v1/admin';

/** GET /api/v1/admin/orders?status=... */
export async function fetchOrders(params?: {
  status?: string;
  page?: number;
  page_size?: number;
}): Promise<{ data: OrderListItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.status) qs.set('status', params.status);
  if (params?.page) qs.set('page', String(params.page));
  if (params?.page_size) qs.set('page_size', String(params.page_size));
  const url = `${BASE}/orders${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, { credentials: 'include' });
  if (!resp.ok) throw new Error(`fetchOrders failed: ${resp.status}`);
  const json = (await resp.json()) as OrderListResponse;
  return { data: json.data, total: json.total };
}

/** GET /api/v1/admin/orders/:id */
export async function fetchOrderDetail(id: number): Promise<OrderDetail> {
  const resp = await fetch(`${BASE}/orders/${id}`, { credentials: 'include' });
  if (!resp.ok) throw new Error(`fetchOrderDetail failed: ${resp.status}`);
  const json = (await resp.json()) as OrderDetailResponse;
  return json.data;
}

/** TanStack Query key 工厂 */
export const orderQueryKeys = {
  list: (params?: { status?: string }) => ['orders', params?.status ?? 'all'] as const,
  detail: (id: number | string) => ['order', id] as const,
};