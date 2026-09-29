// src/api/wallet.js
//
// 钱包 API client —— 余额 / 冻结金额 / 流水。
//（spec §4.6 + plan Task M1）
//
// 端点：
//   GET /api/v1/wallet                       → 余额 + 冻结
//   GET /api/v1/wallet/transactions          → 流水（query={ page, page_size, type }）
//
// 响应契约（与 backend Go JSON 1:1）：
//   Wallet = {
//     user_id,
//     balance,           // 可用余额（元）
//     frozen,            // 冻结金额（元；提现中 / 退款处理中）
//     currency,          // 'CNY'
//     updated_at,
//   }
//   Transaction = {
//     id, user_id, type,           // recharge / payment / refund / withdraw
//     amount,                       // 正数；负向操作（payment）的 amount 为负
//     balance_after,                // 操作后余额（冗余字段，便于前端少一次拉取）
//     order_id,                     // 关联订单（type=payment/refund 时有）
//     description,                 // 备注
//     created_at,
//   }
//   TransactionListResp = { items: Transaction[], total: number, page: number, page_size: number }
//
// 约定：
//   - 后端保证同一 user 的流水按 created_at DESC 返回

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Wallet
 * @property {number|string} user_id
 * @property {number} balance
 * @property {number} frozen
 * @property {string} currency
 * @property {string} updated_at
 */

/**
 * @typedef {Object} Transaction
 * @property {number|string} id
 * @property {number|string} user_id
 * @property {string} type          - recharge / payment / refund / withdraw
 * @property {number} amount
 * @property {number} balance_after
 * @property {number|string} [order_id]
 * @property {string} description
 * @property {string} created_at
 */

/**
 * 拉当前用户钱包（余额 + 冻结）。
 *
 * @returns {Promise<Wallet>}
 */
export function getWallet() {
  return request({
    url: '/wallet',
    method: 'GET',
  });
}

/**
 * 拉钱包流水。
 *
 * @param {{ page?: number, page_size?: number, type?: string }} [query]
 * @returns {Promise<{ items: Transaction[], total: number, page: number, page_size: number }>}
 */
export function listTransactions(query) {
  return request({
    url: '/wallet/transactions',
    method: 'GET',
    query: query || {},
  });
}

export default {
  getWallet,
  listTransactions,
};