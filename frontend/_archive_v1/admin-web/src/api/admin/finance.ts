/**
 * admin-finance API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW handlers 对齐）：
 *   - GET /api/v1/admin/finance/overview?range=today|week|month
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 23
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type FinanceRange = 'today' | 'week' | 'month';

export interface FinanceChannel {
  /** 渠道名（wechat / alipay / unionpay / ...） */
  name: string;
  amount: number;
  count: number;
}

export interface FinanceOverview {
  range: FinanceRange;
  gmv: number;
  refund_amount: number;
  net_revenue: number;
  withdraw_amount: number;
  order_count: number;
  /** 每日 GMV 序列（长度 = range 天数 + 1） */
  daily_gmv: Array<{ date: string; amount: number }>;
  channels: FinanceChannel[];
}

export interface FinanceOverviewResponse {
  code: number;
  data: FinanceOverview;
  trace_id?: string;
}

/**
 * 拉取财务概览。
 *
 * mock 阶段：直接由 fetchFinanceOverviewMock 在前端组装响应（避免新增 MSW handler）。
 * 真实接入时改为 fetch GET /api/v1/admin/finance/overview。
 */
export async function fetchFinanceOverview(
  range: FinanceRange = 'today',
): Promise<FinanceOverview> {
  return fetchFinanceOverviewMock(range);
}

/** 前端 mock：根据 range 拼装一个稳定的 FinanceOverview。 */
function fetchFinanceOverviewMock(range: FinanceRange): FinanceOverview {
  const days = range === 'today' ? 1 : range === 'week' ? 7 : 30;
  const today = new Date();
  const daily_gmv = Array.from({ length: days }, (_, i) => {
    const d = new Date(today);
    d.setDate(today.getDate() - (days - 1 - i));
    const base = range === 'today' ? 8600 : 6000 + i * 350;
    return {
      date: d.toISOString().slice(0, 10),
      amount: base,
    };
  });
  const gmv = daily_gmv.reduce((acc, d) => acc + d.amount, 0);
  const order_count = days * 12 + Math.floor(gmv / 600);
  const refund_amount = Math.round(gmv * 0.08);
  const net_revenue = gmv - refund_amount;
  const withdraw_amount = Math.round(net_revenue * 0.3);
  return {
    range,
    gmv,
    refund_amount,
    net_revenue,
    withdraw_amount,
    order_count,
    daily_gmv,
    channels: [
      { name: '微信支付', amount: Math.round(gmv * 0.6), count: Math.round(order_count * 0.6) },
      { name: '支付宝', amount: Math.round(gmv * 0.3), count: Math.round(order_count * 0.3) },
      { name: '银联', amount: Math.round(gmv * 0.1), count: Math.round(order_count * 0.1) },
    ],
  };
}

export const financeQueryKeys = {
  overview: (range: FinanceRange = 'today') => ['finance-overview', range] as const,
};

/** 保留 fetch 工具以备真实后端接入时使用（当前未使用） */
export async function _fetchFinanceOverviewRaw(
  range: FinanceRange,
): Promise<FinanceOverview> {
  const resp = await fetch(`${BASE}/finance/overview?range=${range}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchFinanceOverview failed: ${resp.status}`);
  const json = (await resp.json()) as FinanceOverviewResponse;
  if (json.code !== 0) throw new Error(`fetchFinanceOverview: code=${json.code}`);
  return json.data;
}