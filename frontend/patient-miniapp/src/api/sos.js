// src/api/sos.js
//
// SOS 紧急呼救 API client —— 长按按钮触发位置上报 + 通知陪诊师/客服。
//（spec §4.5 + plan Task M1）
//
// 端点：
//   POST /api/v1/sos/trigger                body=TriggerSosReq
//   GET  /api/v1/sos/:id                    → 拉 SOS 单条详情（最新状态）
//   POST /api/v1/sos/:id/cancel             body={ reason }  → 误触取消（30s 内）
//
// 响应契约（与 backend Go JSON 1:1）：
//   SosEvent = {
//     id, user_id, order_id,                // order_id 可选：仅订单陪诊中的 SOS
//     lat, lng, address,                    // 上报位置（mock 用经纬度）
//     status,                               // pending / acknowledged / dispatched / cancelled / closed
//     contact_phone,                        // 紧急联系人（用户 profile 字段）
//     created_at,
//     acknowledged_at,
//     cancelled_at,
//   }
//   TriggerSosReq = {
//     order_id?, lat, lng, address?, contact_phone?,
//   }
//
// 约束：
//   - 后端兜底：order_id 在 5 分钟内的陪诊中订单自动关联；SOS 不带 order_id 也会建独立事件
//   - 同一患者 30s 内重复 trigger → 后端合并到同一条

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} TriggerSosReq
 * @property {number|string} [order_id]
 * @property {number} lat
 * @property {number} lng
 * @property {string} [address]
 * @property {string} [contact_phone]
 */

/**
 * @typedef {Object} SosEvent
 * @property {number|string} id
 * @property {number|string} user_id
 * @property {number|string} [order_id]
 * @property {number} lat
 * @property {number} lng
 * @property {string} [address]
 * @property {string} status
 * @property {string} [contact_phone]
 * @property {string} created_at
 * @property {string} [acknowledged_at]
 * @property {string} [cancelled_at]
 */

/**
 * 触发 SOS 事件。
 *
 * @param {TriggerSosReq} payload
 * @returns {Promise<SosEvent>}
 */
export function triggerSos(payload) {
  if (!payload || payload.lat == null || payload.lng == null) {
    return Promise.reject(new Error('triggerSos: lat + lng are required'));
  }
  return request({
    url: '/sos/trigger',
    method: 'POST',
    data: payload,
  });
}

/**
 * 拉 SOS 详情（确认状态用）。
 *
 * @param {number|string} id
 * @returns {Promise<SosEvent>}
 */
export function getSosDetail(id) {
  if (!id) {
    return Promise.reject(new Error('getSosDetail: id is required'));
  }
  return request({
    url: `/sos/${id}`,
    method: 'GET',
  });
}

/**
 * 取消 SOS（误触场景；30s 窗口内）。
 *
 * @param {number|string} id
 * @param {string} [reason]
 * @returns {Promise<SosEvent>}
 */
export function cancelSos(id, reason) {
  if (!id) {
    return Promise.reject(new Error('cancelSos: id is required'));
  }
  return request({
    url: `/sos/${id}/cancel`,
    method: 'POST',
    data: { reason: reason || 'user_cancelled' },
  });
}

export default {
  triggerSos,
  getSosDetail,
  cancelSos,
};