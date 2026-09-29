/**
 * match-service API 客户端（v2 unified-app · :8083）。
 *
 * 端点（v2 后端契约）：
 *   - escort-only：
 *     GET  /api/v1/match/feed?order_id=...        候选订单 feed（抢单池）
 *   - 任意角色：
 *     POST /api/v1/match/candidates              给订单推荐陪诊师候选
 *     POST /api/v1/match/dispatch                派单（运营 / admin 触发）
 *
 * 关键设计：
 *   - feed 是 escort 域核心：进入 escort 域后第一个页面就是抢单池
 *   - candidates / dispatch 是 patient / admin 域辅助
 *
 * 对应：dev.md §41（match-service 端到端跑通）
 *      services/match/internal/handler/match.go
 */
import { request } from './client';

const MATCH_BASE_PATH = '/api/v1/match';
const MATCH_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_MATCH_BASE_URL) || 'http://127.0.0.1:8083';

// ── 类型定义 ───────────────────────────────────────────────────────

export interface MatchCandidate {
  /** 陪诊师 ID */
  escort_id: number;
  /** 候选打分（0-100，越高越匹配） */
  score: number;
  /** 距离订单的距离（米） */
  distance_m: number;
  /** ETA（秒） */
  eta_s: number;
  /** 当前是否可接单 */
  available: boolean;
}

export interface MatchFeedItem {
  order_id: number;
  patient_id: number;
  hospital_id: number;
  appointment_time: string;
  /** 抢单池中的剩余席位 */
  remaining_slots: number;
  /** 该订单已收到的接单申请数 */
  accepted_count: number;
  /** 总陪诊师候选池 */
  total_candidates: number;
  /** 抢单截止时间（ISO 8601） */
  deadline: string;
}

export interface CandidatesRequest {
  order_id: number;
  /** 限定候选数（默认 10） */
  limit?: number;
}

export interface DispatchRequest {
  order_id: number;
  escort_id: number;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/**
 * 抢单池 feed（escort-only）。
 * 返回该 escort 可接的候选订单列表（已按 score + distance 排序）。
 */
export async function fetchFeed(orderId?: number): Promise<{ items: MatchFeedItem[] }> {
  const url = orderId !== undefined ? `${MATCH_BASE_PATH}/feed?order_id=${orderId}` : `${MATCH_BASE_PATH}/feed`;
  return request({ url, baseURL: MATCH_BASE_URL });
}

/**
 * 候选陪诊师推荐（任意角色）。
 * patient 域：下单时显示候选
 * admin 域：审批时显示候选
 */
export async function listCandidates(req: CandidatesRequest): Promise<{ items: MatchCandidate[] }> {
  return request({
    url: `${MATCH_BASE_PATH}/candidates`,
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: MATCH_BASE_URL,
  });
}

/**
 * 派单（任意角色，admin 触发）。
 * 通常 admin 域「人工派单」按钮调用。
 */
export async function dispatchOrder(req: DispatchRequest): Promise<{ ok: true; order_id: number }> {
  return request({
    url: `${MATCH_BASE_PATH}/dispatch`,
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: MATCH_BASE_URL,
  });
}