/**
 * admin-reports API 客户端（v2 增量适配）。
 *
 * 当前骨架未建 fetch client；这里给最小可用 stub（直接调 fetch + 路径硬编码）。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 3 (看板聚合指标)
 */
import type { OverviewReport, OverviewReportResponse } from '@/types/generated';

const BASE = '/api/v1/admin';

/** GET /api/v1/admin/reports/overview */
export async function fetchOverview(): Promise<OverviewReport> {
  const resp = await fetch(`${BASE}/reports/overview`, { credentials: 'include' });
  if (!resp.ok) throw new Error(`fetchOverview failed: ${resp.status}`);
  const json = (await resp.json()) as OverviewReportResponse;
  return json.data;
}

export const reportsQueryKeys = {
  overview: ['dashboard-overview'] as const,
};