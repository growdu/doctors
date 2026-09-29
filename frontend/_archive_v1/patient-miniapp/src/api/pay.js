// src/api/pay.js
//
// 支付 API client —— 微信支付沙箱（v1 mock）。
//（spec §4.2 patient API + plan Task 5/6）
//
// 端点：
//   POST /api/v1/orders/:id/pay             body={ channel }    → 触发支付（v1 mock：返回「沙箱中」的 payUrl）
//   GET  /api/v1/orders/:id/pay-status                            → 轮询支付状态
//
// 响应契约（与 backend Go JSON 1:1）：
//   PayResp = {
//     order_id, channel,
//     pay_url,            // 沙箱 URL；H5/mp-weixin/app-plus 不同
//     prepay_id,          // 微信 prepay_id（v1 mock 假值）
//     nonce_str,
//     timestamp,
//     sign,               // 签名（v1 mock 假值）
//   }
//   PayStatusResp = {
//     order_id, status,   // pending / paid / failed / refunded
//     paid_at,            // ISO8601（paid 时填充）
//     transaction_id,     // 微信 transaction_id（paid 时填充）
//   }
//
// 实现要点：
//   - v1 后端默认返回 sandbox pay_url；前端直接打开（uni.navigateTo 或 uni.redirectTo URL scheme）
//   - payOrder 默认 channel='wechat'；后续可扩 'alipay'
//   - getPayStatus：详情页支付态轮询用（30s 内）

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} PayResp
 * @property {number|string} order_id
 * @property {string}        channel         - wechat / alipay
 * @property {string}        pay_url         - 沙箱 URL
 * @property {string}        [prepay_id]
 * @property {string}        [nonce_str]
 * @property {string}        [timestamp]
 * @property {string}        [sign]
 */

/**
 * @typedef {Object} PayStatusResp
 * @property {number|string} order_id
 * @property {string}        status          - pending / paid / failed / refunded
 * @property {string}        [paid_at]
 * @property {string}        [transaction_id]
 */

/**
 * 触发订单支付（v1 走 mock 沙箱）。
 *
 * @param {number|string} orderId
 * @param {{ channel?: string }} [opts]
 * @returns {Promise<PayResp>}
 */
export function payOrder(orderId, opts) {
  if (!orderId) {
    return Promise.reject(new Error('payOrder: orderId is required'));
  }
  return request({
    url: `/orders/${orderId}/pay`,
    method: 'POST',
    data: { channel: (opts && opts.channel) || 'wechat' },
  });
}

/**
 * 轮询订单支付状态（pay 页「我已支付」/ 详情页轮询）。
 *
 * @param {number|string} orderId
 * @returns {Promise<PayStatusResp>}
 */
export function getPayStatus(orderId) {
  if (!orderId) {
    return Promise.reject(new Error('getPayStatus: orderId is required'));
  }
  return request({
    url: `/orders/${orderId}/pay-status`,
    method: 'GET',
  });
}

export default {
  payOrder,
  getPayStatus,
};