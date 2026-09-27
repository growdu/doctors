// src/api/refund.js
//
// 退款 API client —— 申请退款 / 拉退款状态。
//（spec §4.2 + plan Task M1）
//
// 端点：
//   POST /api/v1/orders/:id/refund         body=RefundReq
//   GET  /api/v1/orders/:id/refund-status  → 拉当前退款状态
//
// 响应契约（与 backend Go JSON 1:1）：
//   RefundResp = {
//     id, order_id, user_id,
//     amount,                  // 元
//     reason,                  // 用户填写的退款理由
//     status,                  // pending / approved / rejected / refunded
//     refund_id,               // 微信 transaction_id（refunded 时填充）
//     created_at, updated_at,
//   }
//   RefundReq = {
//     reason,                  // 必填，10-200 字符
//     reason_code,             // 可选：提前枚举（spec §4.2 表 4-2）
//   }
//
// 约束：
//   - 后端校验：仅「未陪诊」的订单可申请退款；服务中 / 已完成订单要走客服工单
//   - reason 长度 10-200；reason_code 在 reason 为空时必填

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} RefundReq
 * @property {string} reason
 * @property {string} [reason_code]
 */

/**
 * @typedef {Object} RefundResp
 * @property {number|string} id
 * @property {number|string} order_id
 * @property {number|string} user_id
 * @property {number}        amount
 * @property {string}        reason
 * @property {string}        status
 * @property {string}        [refund_id]
 * @property {string}        created_at
 * @property {string}        updated_at
 */

/**
 * 申请退款。
 *
 * @param {number|string} orderId
 * @param {RefundReq}     payload
 * @returns {Promise<RefundResp>}
 */
export function applyRefund(orderId, payload) {
  if (!orderId) {
    return Promise.reject(new Error('applyRefund: orderId is required'));
  }
  if (!payload || !payload.reason) {
    return Promise.reject(new Error('applyRefund: reason is required'));
  }
  return request({
    url: `/orders/${orderId}/refund`,
    method: 'POST',
    data: payload,
  });
}

/**
 * 拉退款状态（轮询用；refund 详情页 / 订单详情页）。
 *
 * @param {number|string} orderId
 * @returns {Promise<RefundResp>}
 */
export function getRefundStatus(orderId) {
  if (!orderId) {
    return Promise.reject(new Error('getRefundStatus: orderId is required'));
  }
  return request({
    url: `/orders/${orderId}/refund-status`,
    method: 'GET',
  });
}

export default {
  applyRefund,
  getRefundStatus,
};