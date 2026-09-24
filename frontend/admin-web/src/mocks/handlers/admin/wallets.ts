/**
 * admin-wallets MSW handlers（v1 缺失骨架）。
 *
 * 路由：
 *   GET  /api/v1/admin/wallets?type=patient|escort&tx_type=recharge|payment|refund|withdraw
 *   GET  /api/v1/admin/wallets/:id                  → 钱包主体（含余额 + 最近 10 条流水）
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 14
 */
import { http, HttpResponse } from 'msw';
import { mockWalletSubjects, mockWalletTransactions } from '../data/seed';

const traceId = () => `mock-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

export const walletHandlers = [
  // ── 流水列表 ──────────────────────────────────────────────────────
  http.get('/api/v1/admin/wallets', ({ request }) => {
    const url = new URL(request.url);
    const subjectType = url.searchParams.get('type'); // patient | escort
    const txType = url.searchParams.get('tx_type'); // recharge | payment | refund | withdraw
    let data = mockWalletTransactions;
    if (subjectType) data = data.filter((t) => t.subject_type === subjectType);
    if (txType) data = data.filter((t) => t.tx_type === txType);
    return HttpResponse.json({
      code: 0,
      data,
      total: data.length,
      trace_id: traceId(),
    });
  }),

  // ── 钱包主体详情 ──────────────────────────────────────────────────
  http.get('/api/v1/admin/wallets/:id', ({ params }) => {
    const subject = mockWalletSubjects.find((s) => s.id === Number(params.id));
    if (!subject) {
      return HttpResponse.json(
        { code: 11004, error: 'not found', trace_id: traceId() },
        { status: 404 },
      );
    }
    const recent = mockWalletTransactions
      .filter((t) => t.subject_id === subject.id)
      .slice(0, 10);
    return HttpResponse.json({
      code: 0,
      data: { ...subject, recent_transactions: recent },
      trace_id: traceId(),
    });
  }),
];