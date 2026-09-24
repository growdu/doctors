/**
 * admin-reports MSW handlers（v2 增量适配）。
 *
 * 路由：GET /api/v1/admin/reports/overview
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 3
 */
import { http, HttpResponse } from 'msw';
import { mockOverview } from '../data/seed';

export const reportHandlers = [
  http.get('/api/v1/admin/reports/overview', () => {
    return HttpResponse.json({ data: mockOverview });
  }),
];