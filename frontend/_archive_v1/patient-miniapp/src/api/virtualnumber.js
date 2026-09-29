// src/api/virtualnumber.js
//
// 虚拟号 API client —— 订单服务中分配隐私号供患者与陪诊师互通。
// （spec §4.7 + plan Task M1 + 后端 handler：`services/user/internal/virtualnumber/virtualnumber.go`）
//
// 端点：
//   POST /api/v1/virtual-numbers/allocate   body=AllocateVirtualNumberReq
//   GET  /api/v1/virtual-numbers/:id
//
// 响应契约（与 backend Go JSON 1:1）：
//   VirtualNumber = {
//     id, order_id, patient_id, escort_id, phone,
//     status,            // active / released / expired
//     expire_at,         // ISO8601
//     released_at,       // ISO8601 或 null
//     created_at,
//   }
//
// 约束：
//   - 同一 order_id 同时刻最多 1 个 active 虚拟号（后端 partial unique 保证）
//   - expire_at 必须晚于 now（后端校验）
//   - v1 前端只读；分配由 order-service 创建订单后内部触发，详情页展示用

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} VirtualNumber
 * @property {number|string} id
 * @property {number|string} order_id
 * @property {number|string} patient_id
 * @property {number|string} escort_id
 * @property {string}        phone
 * @property {string}        status          - active / released / expired
 * @property {string}        expire_at
 * @property {string|null}   released_at
 * @property {string}        created_at
 */

/**
 * @typedef {Object} AllocateVirtualNumberReq
 * @property {number|string} order_id
 * @property {number|string} patient_id
 * @property {number|string} escort_id
 * @property {string}        expire_at       - ISO8601；必须晚于 now
 */

/**
 * 分配虚拟号（v1 主要由 order-service 触发；前端暴露供管理端/调试用）。
 *
 * @param {AllocateVirtualNumberReq} payload
 * @returns {Promise<VirtualNumber>}
 */
export function allocateVirtualNumber(payload) {
  if (!payload || !payload.order_id || !payload.patient_id ||
      !payload.escort_id || !payload.expire_at) {
    return Promise.reject(new Error(
      'allocateVirtualNumber: order_id / patient_id / escort_id / expire_at are required',
    ));
  }
  return request({
    url: '/virtual-numbers/allocate',
    method: 'POST',
    data: payload,
  });
}

/**
 * 拉虚拟号详情（订单详情页用）。
 *
 * @param {number|string} id
 * @returns {Promise<VirtualNumber>}
 */
export function getVirtualNumber(id) {
  if (!id) {
    return Promise.reject(new Error('getVirtualNumber: id is required'));
  }
  return request({
    url: `/virtual-numbers/${id}`,
    method: 'GET',
  });
}

export default {
  allocateVirtualNumber,
  getVirtualNumber,
};