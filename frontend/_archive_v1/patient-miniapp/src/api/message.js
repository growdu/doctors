// src/api/message.js
//
// 站内信 API client —— 系统通知 / 营销消息列表。
//（spec §4.8 + plan Task M1）
//
// 端点：
//   GET /api/v1/messages                 query={ page, page_size, category, unread }
//   GET /api/v1/messages/:id
//   POST /api/v1/messages/:id/read      → 标记已读
//   POST /api/v1/messages/read-all      → 全部已读
//
// 响应契约（与 backend Go JSON 1:1）：
//   Message = {
//     id, user_id,
//     category,                  // system / promo / order / escort
//     title, content,
//     read_at,                   // null = 未读
//     created_at,
//     link,                      // 可选；点击跳转 URL（如 /pages/order/detail?orderId=7）
//   }
//   MessageListResp = { items: Message[], total: number, unread: number, page: number, page_size: number }
//
// 约定：
//   - 列表按 created_at DESC 返回
//   - unread 字段是用户级未读总数（与分页无关）

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Message
 * @property {number|string} id
 * @property {number|string} user_id
 * @property {string} category
 * @property {string} title
 * @property {string} content
 * @property {string|null} read_at
 * @property {string} created_at
 * @property {string} [link]
 */

/**
 * @typedef {Object} MessageListResp
 * @property {Message[]} items
 * @property {number} total
 * @property {number} unread
 * @property {number} page
 * @property {number} page_size
 */

/**
 * 拉站内信列表（分页 + 分类过滤 + 未读过滤）。
 *
 * @param {{ page?: number, page_size?: number, category?: string, unread?: boolean }} [query]
 * @returns {Promise<MessageListResp>}
 */
export function listMessages(query) {
  return request({
    url: '/messages',
    method: 'GET',
    query: query || {},
  });
}

/**
 * 拉单条站内信详情。
 *
 * @param {number|string} id
 * @returns {Promise<Message>}
 */
export function getMessageDetail(id) {
  if (!id) {
    return Promise.reject(new Error('getMessageDetail: id is required'));
  }
  return request({
    url: `/messages/${id}`,
    method: 'GET',
  });
}

/**
 * 标记单条已读。
 *
 * @param {number|string} id
 * @returns {Promise<{ id: number|string, read_at: string }>}
 */
export function markRead(id) {
  if (!id) {
    return Promise.reject(new Error('markRead: id is required'));
  }
  return request({
    url: `/messages/${id}/read`,
    method: 'POST',
  });
}

/**
 * 全部标记已读。
 *
 * @returns {Promise<{ updated: number }>}
 */
export function markAllRead() {
  return request({
    url: '/messages/read-all',
    method: 'POST',
  });
}

export default {
  listMessages,
  getMessageDetail,
  markRead,
  markAllRead,
};