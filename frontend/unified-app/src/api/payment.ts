/**
 * payment-service API 客户端（v2 unified-app · :8085）。
 *
 * 端点：
 *   - POST /api/v1/payments                  创建支付单（patient-only）
 *   - GET  /api/v1/payments/{id}             支付单详情
 *   - POST /api/v1/payments/{id}/complete    标记完成（mock channel webhook 同步）
 *   - POST /api/v1/payments/{id}/refund     申请退款
 *
 * 完成支付会触发 Kafka PaymentCompletedEvent → wallet T+7 结算 + review 待评价（+24h）。
 *
 * 对应：services/payment/internal/handler/payment.go
 */
import { request } from './client';

const PAYMENT_BASE_PATH = '/api/v1/payments';
const PAYMENT_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_PAYMENT_BASE_URL) || 'http://127.0.0.1:8085';

// ── 类型定义 ───────────────────────────────────────────────────────

export type PaymentStatus = 'pending' | 'paid' | 'refunded' | 'failed';

export interface Payment {
  id: number;
  order_id: number;
  patient_id: number;
  /** 支付金额（分） */
  amount: number;
  status: PaymentStatus;
  /** 支付渠道：wechat / alipay / mock */
  channel: 'wechat' | 'alipay' | 'mock';
  /** 渠道交易号（mock 时为空） */
  channel_tx_id: string | null;
  paid_at: string | null;
  refunded_at: string | null;
  created_at: string;
}

export interface CreatePaymentRequest {
  order_id: number;
  channel?: 'wechat' | 'alipay' | 'mock';
}

export interface RefundRequest {
  /** 退款金额（分），不传则全额 */
  amount?: number;
  reason?: string;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/** 创建支付单（patient-only）。 */
export async function createPayment(req: CreatePaymentRequest): Promise<Payment> {
  return request({
    url: PAYMENT_BASE_PATH,
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: PAYMENT_BASE_URL,
  });
}

/** 支付单详情。 */
export async function getPayment(paymentId: number): Promise<Payment> {
  return request({ url: `${PAYMENT_BASE_PATH}/${paymentId}`, baseURL: PAYMENT_BASE_URL });
}

/**
 * 标记完成（dev mock 渠道使用）。
 * 真支付渠道通过 webhook 异步调用，此端点用于 dev / 测试。
 */
export async function completePayment(paymentId: number): Promise<Payment> {
  return request({
    url: `${PAYMENT_BASE_PATH}/${paymentId}/complete`,
    method: 'POST',
    baseURL: PAYMENT_BASE_URL,
  });
}

/** 申请退款。 */
export async function refundPayment(paymentId: number, req: RefundRequest = {}): Promise<Payment> {
  return request({
    url: `${PAYMENT_BASE_PATH}/${paymentId}/refund`,
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: PAYMENT_BASE_URL,
  });
}