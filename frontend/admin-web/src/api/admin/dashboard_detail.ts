/**
 * admin-dashboard-detail API 客户端（14 P0 页依赖）。
 *
 * 用于看板卡片下钻的明细数据：
 *   - 根据 type（orders_pending / refunds_pending / escorts_pending / sos_open）
 *   - 返回对应的明细行。
 *
 * 当前为 mock：直接由前端组装。
 * 真实接入时改为 fetch GET /api/v1/admin/dashboard-detail?type=...。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 28
 */
import {
  mockOrders,
  mockRefunds,
  mockEscorts,
  mockSosAlerts,
} from '@/mocks/data/seed';

export type DashboardDetailType =
  | 'orders_pending'
  | 'refunds_pending'
  | 'escorts_pending'
  | 'sos_open';

export interface DashboardDetailRow {
  id: number;
  title: string;
  subtitle: string;
  status: string;
  created_at: string;
  href: string;
}

export interface DashboardDetailPayload {
  type: DashboardDetailType;
  label: string;
  rows: DashboardDetailRow[];
}

/** 拉取下钻明细（mock）。 */
export function fetchDashboardDetail(
  type: DashboardDetailType,
): DashboardDetailPayload {
  switch (type) {
    case 'orders_pending':
      return {
        type,
        label: '待处理订单',
        rows: mockOrders
          .filter((o) =>
            ['created', 'paid', 'selecting_escort', 'escort_pending_acceptance'].includes(
              o.status,
            ),
          )
          .map((o) => ({
            id: o.id,
            title: `${o.hospital_name} · ${o.package_name}`,
            subtitle: `${o.patient_name} · ${o.patient_phone}`,
            status: o.status,
            created_at: o.created_at,
            href: `/orders/${o.id}`,
          })),
      };
    case 'refunds_pending':
      return {
        type,
        label: '待审退款',
        rows: mockRefunds
          .filter((r) => r.status === 'pending')
          .map((r) => ({
            id: r.id,
            title: `${r.patient_name} · ¥${r.amount}`,
            subtitle: r.reason,
            status: r.status,
            created_at: r.created_at,
            href: `/refunds/${r.id}`,
          })),
      };
    case 'escorts_pending':
      return {
        type,
        label: '待审核陪诊师',
        rows: mockEscorts
          .filter((e) => e.audit_status === 'pending')
          .map((e) => ({
            id: e.id,
            title: `${e.name} · ${e.city}`,
            subtitle: e.phone,
            status: e.audit_status,
            created_at: e.created_at,
            href: `/escorts/${e.id}`,
          })),
      };
    case 'sos_open':
      return {
        type,
        label: '未关闭 SOS',
        rows: mockSosAlerts
          .filter((s) => s.status === 'open')
          .map((s) => ({
            id: s.id,
            title: `${s.patient_name} @ ${s.location}`,
            subtitle: `陪诊 ${s.escort_name} · 联系电话 ${s.contact}`,
            status: s.status,
            created_at: s.created_at,
            href: '/sos',
          })),
      };
    default:
      return { type, label: '未知', rows: [] };
  }
}

export const dashboardDetailQueryKeys = {
  byType: (type: DashboardDetailType) => ['dashboard-detail', type] as const,
};