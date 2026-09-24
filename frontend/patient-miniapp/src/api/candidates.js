// src/api/candidates.js
//
// 候选陪诊师列表 API client —— 订单支付成功后患者从此列表中选定 1 位陪诊师。
// （spec §4.1 patient API + plan Task P3 + spec `2026-09-24-order-matching-redesign.md` §1.2）
//
// 端点：
//   GET /api/v1/orders/:id/candidates
//
// 响应契约（与 backend OpenAPI 1:1 对齐 —— 后端约定返回带 generated_at 的包装对象）：
//   {
//     items: Candidate[],
//     generated_at: string   // ISO8601；前端展示"X 秒前刷新"用
//   }
//
// 数据模型（camelCase —— 前端 UI 直接消费）：
//   Candidate = { escortId, nickname, rating, completedOrders, distanceKm, tags[] }
//
// 实现要点：
//   - 复用根目录 utils/request.js 的 request 函数（拦截器统一加 X-Trace-Id + Authorization / 11001 reLaunch login）
//   - 不再做二次 data 解包 —— request.js 已把 `{ code, data, message }` 剥壳为 data 直接 return
//   - 由 src/api/order.js 同步 re-export，供已落地 stores/order.js 的
//     `await import('@/api/order.js').getCandidates(...)` 兼容（store 不要动）

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Candidate
 * @property {number|string} escortId         - 陪诊师 ID（用于 selectEscort 的入参）
 * @property {string}        nickname         - 展示名
 * @property {number}        rating           - 0.0 ~ 5.0
 * @property {number}        [completedOrders] - 累计完成订单数（可选，后端可能不返）
 * @property {number}        [distanceKm]     - 距离（公里，可选）
 * @property {string[]}      [tags]           - 标签 e.g. ['耐心','三甲熟悉','陪同手术']
 */

/**
 * @typedef {Object} CandidatesResp
 * @property {Candidate[]} items
 * @property {string}      generated_at
 */

/**
 * 拉订单的候选陪诊师列表（订单处于 selectingEscort 时调）。
 *
 * @param {number|string} orderId
 * @returns {Promise<CandidatesResp>}
 */
export function getCandidates(orderId) {
  if (!orderId) {
    return Promise.reject(new Error('getCandidates: orderId is required'));
  }
  return request({
    url: `/orders/${orderId}/candidates`,
    method: 'GET',
  });
}

export default {
  getCandidates,
};
