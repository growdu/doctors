// src/api/order.js
//
// 订单 API client —— 详情 / 选陪诊师（v1.1 增量）。
// （spec §4.2 patient API + plan Task P3 + spec `2026-09-24-order-matching-redesign.md` §4.1）
//
// 端点：
//   GET  /api/v1/orders/:id
//   POST /api/v1/orders/:id/select-escort      body={ escort_id }
//   GET  /api/v1/orders                         query={ status, page, page_size }
//   POST /api/v1/orders/:id/cancel             body={ reason }
//
// 数据模型（camelCase —— 前端 UI 直接消费；raw 响应字段由 backend Go 命名 snake_case，
// 本模块做最小透传，store 端按 backend 字段名读取 —— 见 stores/order.js 测试）：
//   Order = {
//     id, status,
//     selectedEscortId, escortPendingExpireAt, escortRejectReason,
//     ... 其它 backend 字段（amount / hospitalId / ...）原样透传
//   }
//
// 历史：本文件 v1 计划含 listOrders / cancelOrder。v1.1 选陪诊师增量阶段，
// 已落地 stores/order.js（commit 946cb72）通过 `await import('@/api/order.js')`
// 取 getOrder / listOrders / getCandidates / selectEscort / cancelOrder 五个函数，
// 故此处同步保留 listOrders / cancelOrder 实现 + 从 candidates.js re-export getCandidates，
// 不破坏既有 store 单测（"已完成（不要动）"约束）。

import { request } from '../../utils/request.js';
import { getCandidates } from './candidates.js';

// 让 store 的 `import('@/api/order.js').getCandidates(...)` 兼容（已落地 stores/order.js）。
export { getCandidates };

/**
 * @typedef {Object} Order
 * @property {number|string} id
 * @property {string} status                  - 状态机取值：pendingPayment / selectingEscort / escortPendingAcceptance / ...
 * @property {number|string} [selectedEscortId] - v1.1 选人模式：患者已选 escort
 * @property {string}        [escortPendingExpireAt] - v1.1：escort 30s 倒计时到期时刻（ISO8601）
 * @property {string}        [escortRejectReason]    - v1.1：escort 拒接时填充的原因
 */

/**
 * @typedef {Object} SelectEscortResp
 * @property {number|string} orderId
 * @property {string}        status              - 期望 'escortPendingAcceptance'
 * @property {string}        escortPendingExpireAt - ISO8601；UI 30s 倒计时的截止时刻
 */

/**
 * 拉订单详情（详情页 / 状态轮询时调）。
 *
 * @param {number|string} orderId
 * @returns {Promise<Order>}
 */
export function getOrder(orderId) {
  if (!orderId) {
    return Promise.reject(new Error('getOrder: orderId is required'));
  }
  return request({
    url: `/orders/${orderId}`,
    method: 'GET',
  });
}

/**
 * 患者选定某 escort —— 后端把订单状态切到 escortPendingAcceptance，
 * 返回 escort 30s 确认窗口的截止时刻（escortPendingExpireAt）。
 *
 * 入参兼容两种形态（按调用方方便）：
 *   - `selectEscort(orderId, 11)`                       （brief 描述形式：裸 escort_id）
 *   - `selectEscort(orderId, { escort_id: 11 })`        （OpenAPI / 已落地 store 调用形式）
 *
 * @param {number|string} orderId
 * @param {number|string|{escort_id: number|string}} escortIdOrPayload
 * @returns {Promise<SelectEscortResp>}
 */
export function selectEscort(orderId, escortIdOrPayload) {
  if (!orderId) {
    return Promise.reject(new Error('selectEscort: orderId is required'));
  }
  if (!escortIdOrPayload) {
    return Promise.reject(new Error('selectEscort: escortId is required'));
  }
  // 兼容：裸 escort_id 或 { escort_id } 对象 body
  const body =
    typeof escortIdOrPayload === 'object'
      ? escortIdOrPayload
      : { escort_id: escortIdOrPayload };

  return request({
    url: `/orders/${orderId}/select-escort`,
    method: 'POST',
    data: body,
  });
}

/**
 * 拉订单列表（订单列表页用；store 兼容字段）。
 *
 * @param {object} [query]
 * @param {string} [query.status]    - 状态过滤 e.g. 'selectingEscort'
 * @param {number} [query.page]
 * @param {number} [query.page_size]
 * @returns {Promise<{ items: Order[] } | Order[]>} 兼容直返数组 / {items} 两种形态
 */
export function listOrders(query) {
  return request({
    url: '/orders',
    method: 'GET',
    query: query || {},
  });
}

/**
 * 取消订单（用户主动）。
 *
 * @param {number|string} orderId
 * @param {string}        reason
 * @returns {Promise<unknown>}
 */
export function cancelOrder(orderId, reason) {
  if (!orderId) {
    return Promise.reject(new Error('cancelOrder: orderId is required'));
  }
  return request({
    url: `/orders/${orderId}/cancel`,
    method: 'POST',
    data: { reason: reason || '' },
  });
}

export default {
  getOrder,
  selectEscort,
  listOrders,
  cancelOrder,
  getCandidates,
};
