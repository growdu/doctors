/**
 * 手写 stub 的 OpenAPI 客户端类型（v2 增量适配）。
 *
 * 说明：
 * - 真正的 `generated.ts` 应由 `pnpm run generate:client` 从
 *   `openapi/contracts.yaml` 调用 openapi-typescript 生成。
 * - 本机当前未跑生成脚本（任务契约禁止 npm install / 跑工具链），
 *   故手写最小可用 stub，保证 v2 新字段在 TS 编译期可见。
 * - 待工具链补齐后，`pnpm run generate:client` 会覆盖本文件。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4 (订单状态机)
 *           2026-09-24-admin-web-setup.md §Task 6
 */

// ── OrderStatus 枚举 ────────────────────────────────────────────────
// 状态机全集（含 v2 新增 selecting_escort / escort_pending_acceptance）
export type OrderStatus =
  | 'created'
  | 'paid'
  | 'matching'
  | 'selecting_escort'           // v2 新增：候选已生成，待患者选
  | 'escort_pending_acceptance'  // v2 新增：患者选了某人，待陪诊师 30s 确认
  | 'accepted'
  | 'in_service'
  | 'completed'
  | 'reviewed'
  | 'refunding'
  | 'refunded'
  | 'settling'
  | 'disputed'
  | 'closed'
  | 'canceled';

// 拒接原因枚举（用于 escort_reject_reason 字段）
export type EscortRejectReason = 'escort_declined' | 'lock_expired' | null;

// ── OrderListItem（列表项） ─────────────────────────────────────────
export interface OrderListItem {
  id: number;
  status: OrderStatus;
  hospital_name: string;
  final_amount: number;
  created_at: string;
  // v2 新增字段
  selected_escort_id: number | null;
  escort_pending_expire_at: string | null;
}

// ── OrderDetail（详情） ─────────────────────────────────────────────
export interface OrderDetail extends OrderListItem {
  patient_name?: string;
  patient_phone?: string;
  package_name?: string;
  // v2 新增字段
  escort_reject_reason: EscortRejectReason;
}

// ── OrderListResponse（列表 API 响应包装） ──────────────────────────
export interface OrderListResponse {
  data: OrderListItem[];
  total: number;
}

export interface OrderDetailResponse {
  data: OrderDetail;
}

// ── OverviewReport（看板聚合指标） ─────────────────────────────────
export interface OverviewReport {
  today_orders: number;
  today_gmv: number;
  pending_escorts: number;
  pending_refunds: number;
  // v2 新增指标
  pending_selecting_escort: number;
  pending_escort_acceptance: number;
}

export interface OverviewReportResponse {
  data: OverviewReport;
}