/**
 * wallet-service API 客户端（v2 unified-app · :8090）。
 *
 * 端点（按角色分组）：
 *   - patient：
 *     GET /api/v1/users/me/wallet                患者钱包余额
 *   - escort：
 *     GET /api/v1/escorts/me/wallet              陪诊师钱包余额
 *     POST /api/v1/escorts/me/wallet/withdraw    提现申请
 *   - 任意角色：
 *     GET /api/v1/wallet/transactions            流水列表
 *   - admin（admin-only）：
 *     POST /api/v1/admin/wallet/withdrawals/{id}/approve   审批通过
 *     POST /api/v1/admin/wallet/withdrawals/{id}/pay       标记已打款
 *     POST /api/v1/admin/wallet/withdrawals/{id}/reject    驳回
 *
 * 关键设计：
 *   - 患者端显示余额 + 充值（如有）
 *   - 陪诊师端显示余额 + 提现申请
 *   - admin 域审批提现工单
 *
 * 对应：services/wallet/internal/handler/wallet.go
 */
import { request } from './client';

const WALLET_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_WALLET_BASE_URL) || 'http://127.0.0.1:8090';

// ── 类型定义 ───────────────────────────────────────────────────────

export interface Wallet {
  user_id: number;
  /** 可用余额（分） */
  balance: number;
  /** 冻结金额（分） */
  frozen: number;
  /** 累计收入（分） */
  total_earned: number;
  /** 累计提现（分） */
  total_withdrawn: number;
  updated_at: string;
}

export type TransactionType = 'income' | 'withdraw' | 'refund' | 'freeze' | 'unfreeze';
export type TransactionStatus = 'pending' | 'success' | 'failed';

export interface Transaction {
  id: number;
  wallet_user_id: number;
  type: TransactionType;
  amount: number;
  status: TransactionStatus;
  /** 关联订单 ID（income / refund 时） */
  order_id: number | null;
  /** 备注 */
  remark: string;
  created_at: string;
}

export interface WithdrawRequest {
  /** 提现金额（分） */
  amount: number;
  /** 提现账户 */
  account: string;
  /** 账户名 */
  account_name: string;
}

export interface Withdrawal {
  id: number;
  escort_id: number;
  amount: number;
  account: string;
  account_name: string;
  status: 'pending' | 'approved' | 'paid' | 'rejected';
  reject_reason: string | null;
  created_at: string;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/** 患者钱包。 */
export async function getUserWallet(): Promise<Wallet> {
  return request({ url: '/api/v1/users/me/wallet', baseURL: WALLET_BASE_URL });
}

/** 陪诊师钱包。 */
export async function getEscortWallet(): Promise<Wallet> {
  return request({ url: '/api/v1/escorts/me/wallet', baseURL: WALLET_BASE_URL });
}

/** 流水列表（patient + escort 共用）。 */
export async function listTransactions(query: { type?: TransactionType; page?: number; page_size?: number } = {}): Promise<{ items: Transaction[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.type) qs.set('type', query.type);
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/wallet/transactions?${qs}` : '/api/v1/wallet/transactions';
  return request({ url, baseURL: WALLET_BASE_URL });
}

/** 提现申请（escort-only）。 */
export async function requestWithdrawal(req: WithdrawRequest): Promise<Withdrawal> {
  return request({
    url: '/api/v1/escorts/me/wallet/withdraw',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: WALLET_BASE_URL,
  });
}

// ── admin 域（提现工单审批） ────────────────────────────────────────

/** 审批通过。 */
export async function approveWithdrawal(withdrawalId: number): Promise<Withdrawal> {
  return request({
    url: `/api/v1/admin/wallet/withdrawals/${withdrawalId}/approve`,
    method: 'POST',
    baseURL: WALLET_BASE_URL,
  });
}

/** 标记已打款。 */
export async function markWithdrawalPaid(withdrawalId: number, channelTxId?: string): Promise<Withdrawal> {
  return request({
    url: `/api/v1/admin/wallet/withdrawals/${withdrawalId}/pay`,
    method: 'POST',
    data: channelTxId ? { channel_tx_id: channelTxId } : {},
    baseURL: WALLET_BASE_URL,
  });
}

/** 驳回。 */
export async function rejectWithdrawal(withdrawalId: number, reason: string): Promise<Withdrawal> {
  return request({
    url: `/api/v1/admin/wallet/withdrawals/${withdrawalId}/reject`,
    method: 'POST',
    data: { reason } as unknown as Record<string, unknown>,
    baseURL: WALLET_BASE_URL,
  });
}