/**
 * admin-reports API 客户端（v2 增量适配 + v3 报表页扩展）。
 *
 * 端点：
 *   - GET /api/v1/admin/reports/overview → dashboard 看板聚合指标（v2）
 *   - GET /api/v1/admin/reports/business?from=&to=  → 业务报表（v3 mock）
 *
 * 设计：
 *   - overview 走真 MSW handler；
 *   - business 报告暂时由前端 mock 拼装（避免新增 MSW handler）。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 3 + §Task 24
 */
import type { OverviewReport, OverviewReportResponse } from '@/types/generated';
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

/** GET /api/v1/admin/reports/overview */
export async function fetchOverview(): Promise<OverviewReport> {
  const resp = await fetch(`${BASE}/reports/overview`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchOverview failed: ${resp.status}`);
  const json = (await resp.json()) as OverviewReportResponse;
  return json.data;
}

export const reportsQueryKeys = {
  overview: ['dashboard-overview'] as const,
  business: (params?: { from?: string; to?: string }) =>
    ['reports-business', params?.from ?? '', params?.to ?? ''] as const,
};

// ── v3 增量：业务报表 ─────────────────────────────────────────────────
export interface BusinessReportRow {
  /** 维度键：hospital_id / package_id / escort_id */
  dimension: 'hospital' | 'package' | 'escort';
  key: number;
  label: string;
  order_count: number;
  gmv: number;
  refund_count: number;
  refund_amount: number;
}

export interface BusinessReport {
  from: string;
  to: string;
  totals: {
    order_count: number;
    gmv: number;
    refund_count: number;
    refund_amount: number;
    net_revenue: number;
    avg_rating: number;
  };
  rows: BusinessReportRow[];
}

/**
 * 拉取业务报表。
 * mock 阶段由前端组装；真实接入时改为真 fetch。
 */
export async function fetchBusinessReport(
  params: { from: string; to: string; dimension?: BusinessReportRow['dimension'] },
): Promise<BusinessReport> {
  return buildBusinessReportMock(params);
}

function buildBusinessReportMock(params: {
  from: string;
  to: string;
  dimension?: BusinessReportRow['dimension'];
}): BusinessReport {
  const dimension = params.dimension ?? 'hospital';
  const labels =
    dimension === 'hospital'
      ? [
          { key: 13001, label: '北京协和医院' },
          { key: 13002, label: '上海同济医院' },
          { key: 13003, label: '广州安贞医院' },
        ]
      : dimension === 'package'
      ? [
          { key: 14001, label: '半日陪诊' },
          { key: 14002, label: '全日陪诊' },
          { key: 14003, label: '专项陪诊' },
        ]
      : [
          { key: 1003, label: '赵陪诊' },
          { key: 1004, label: '孙陪诊' },
          { key: 1001, label: '王陪诊' },
        ];
  const rows: BusinessReportRow[] = labels.map((it, idx) => {
    const order_count = 80 - idx * 18 + 5;
    const gmv = order_count * (dimension === 'package' ? 800 : 600);
    const refund_count = Math.max(1, Math.round(order_count * 0.05));
    const refund_amount = refund_count * 500;
    return {
      dimension,
      key: it.key,
      label: it.label,
      order_count,
      gmv,
      refund_count,
      refund_amount,
    };
  });
  const totals = rows.reduce(
    (acc, r) => ({
      order_count: acc.order_count + r.order_count,
      gmv: acc.gmv + r.gmv,
      refund_count: acc.refund_count + r.refund_count,
      refund_amount: acc.refund_amount + r.refund_amount,
      net_revenue: acc.net_revenue + (r.gmv - r.refund_amount),
      avg_rating: acc.avg_rating + 4.5,
    }),
    { order_count: 0, gmv: 0, refund_count: 0, refund_amount: 0, net_revenue: 0, avg_rating: 0 },
  );
  return { from: params.from, to: params.to, totals, rows };
}