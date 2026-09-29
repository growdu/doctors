/**
 * review-service API 客户端（v2 unified-app · :8086）。
 *
 * 端点：
 *   - POST /api/v1/reviews                创建评价（patient-only，订单完成后 +24h）
 *   - GET  /api/v1/reviews                评价列表（按 escort_id / order_id 过滤）
 *   - GET  /api/v1/reviews/{id}           评价详情
 *   - POST /api/v1/reviews/{id}/reply     商家回复（escort-only）
 *
 * 对应：services/review/internal/handler/review.go
 */
import { request } from './client';

const REVIEW_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_REVIEW_BASE_URL) || 'http://127.0.0.1:8086';

// ── 类型定义 ───────────────────────────────────────────────────────

export interface Review {
  id: number;
  order_id: number;
  patient_id: number;
  escort_id: number;
  /** 1-5 星 */
  rating: number;
  /** 评价标签 */
  tags: string[];
  content: string;
  reply: string | null;
  reply_at: string | null;
  created_at: string;
}

export interface CreateReviewRequest {
  order_id: number;
  rating: number;
  tags?: string[];
  content: string;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/** 创建评价（patient-only）。 */
export async function createReview(req: CreateReviewRequest): Promise<Review> {
  return request({
    url: '/api/v1/reviews',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: REVIEW_BASE_URL,
  });
}

/** 评价列表。 */
export async function listReviews(query: { order_id?: number; escort_id?: number; page?: number; page_size?: number } = {}): Promise<{ items: Review[]; total: number }> {
  const qs = new URLSearchParams();
  if (query.order_id) qs.set('order_id', String(query.order_id));
  if (query.escort_id) qs.set('escort_id', String(query.escort_id));
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/reviews?${qs}` : '/api/v1/reviews';
  return request({ url, baseURL: REVIEW_BASE_URL });
}

/** 评价详情。 */
export async function getReview(id: number): Promise<Review> {
  return request({ url: `/api/v1/reviews/${id}`, baseURL: REVIEW_BASE_URL });
}

/** 商家回复（escort-only）。 */
export async function replyReview(id: number, reply: string): Promise<Review> {
  return request({
    url: `/api/v1/reviews/${id}/reply`,
    method: 'POST',
    data: { reply } as unknown as Record<string, unknown>,
    baseURL: REVIEW_BASE_URL,
  });
}