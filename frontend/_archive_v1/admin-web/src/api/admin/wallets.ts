/**
 * admin-wallets API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW wallets handler 对齐）：
 *   - GET  /api/v1/admin/wallets?type=&tx_type=
 *   - GET  /api/v1/admin/wallets/:id    → 钱包主体 + 最近 10 条流水
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 14
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type WalletSubjectType = 'patient' | 'escort';
export type WalletTxType = 'recharge' | 'payment' | 'refund' | 'withdraw';

export interface WalletTxItem {
  id: number;
  subject_id: number;
  subject_type: WalletSubjectType;
  tx_type: WalletTxType;
  amount: number;
  balance_after: number;
  created_at: string;
}

export interface WalletSubjectDetail {
  id: number;
  subject_type: WalletSubjectType;
  subject_name: string;
  balance: number;
  frozen: number;
  recent_transactions: WalletTxItem[];
}

export interface WalletTxListResponse {
  code: number;
  data: WalletTxItem[];
  total: number;
  trace_id?: string;
}

export interface WalletDetailResponse {
  code: number;
  data: WalletSubjectDetail;
  trace_id?: string;
}

export async function fetchWalletTransactions(
  params?: { type?: string; tx_type?: string },
): Promise<{ data: WalletTxItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.type) qs.set('type', params.type);
  if (params?.tx_type) qs.set('tx_type', params.tx_type);
  const url = `${BASE}/wallets${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchWalletTransactions failed: ${resp.status}`);
  const json = (await resp.json()) as WalletTxListResponse;
  if (json.code !== 0) throw new Error(`fetchWalletTransactions: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchWalletDetail(id: number): Promise<WalletSubjectDetail> {
  const resp = await fetch(`${BASE}/wallets/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchWalletDetail failed: ${resp.status}`);
  const json = (await resp.json()) as WalletDetailResponse;
  if (json.code !== 0) throw new Error(`fetchWalletDetail: code=${json.code}`);
  return json.data;
}

export const walletQueryKeys = {
  list: (params?: { type?: string; tx_type?: string }) =>
    ['wallets', params?.type ?? 'all', params?.tx_type ?? 'all'] as const,
  detail: (id: number | string) => ['wallet-detail', id] as const,
};