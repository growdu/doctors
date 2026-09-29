// src/api/address.js
//
// 地址簿 API client —— 患者端地址 CRUD + 默认地址设置。
// （spec §4.4 + plan Task M1 + 后端 handler：`services/user/internal/address/address.go`）
//
// 端点：
//   GET    /api/v1/addresses
//   POST   /api/v1/addresses                 body=CreateAddressReq
//   PUT    /api/v1/addresses/:id            body=UpdateAddressReq
//   PUT    /api/v1/addresses/:id/default    设置默认地址（无 body）
//   DELETE /api/v1/addresses/:id
//
// 响应契约（与 backend Go JSON 1:1）：
//   Address = {
//     id, user_id, recipient, phone, detail,
//     lat, lng, is_default,
//     created_at, updated_at,
//   }
//   AddressListResp = { items: Address[] }
//
// 约定：
//   - 单一上限 5 条由后端校验（CodeConflict 13002 等）；前端可依据错误码提示用户
//   - recipient / phone / detail 必填，长度校验由后端兜底

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Address
 * @property {number|string} id
 * @property {number|string} user_id
 * @property {string}        recipient   - 收件人
 * @property {string}        phone       - 手机号
 * @property {string}        detail      - 详细地址
 * @property {number}        [lat]       - 纬度（可选）
 * @property {number}        [lng]       - 经度（可选）
 * @property {boolean}       [is_default]
 * @property {string}        [created_at]
 * @property {string}        [updated_at]
 */

/**
 * @typedef {Object} CreateAddressReq
 * @property {string}  recipient
 * @property {string}  phone
 * @property {string}  detail
 * @property {number}  [lat]
 * @property {number}  [lng]
 * @property {boolean} [is_default]
 */

/**
 * @typedef {Object} UpdateAddressReq
 * @property {string}  recipient
 * @property {string}  phone
 * @property {string}  detail
 * @property {number}  [lat]
 * @property {number}  [lng]
 */

/**
 * 拉地址列表（地址管理页用）。
 *
 * @returns {Promise<{items: Address[]} | Address[]>} 兼容 {items} 与直返数组
 */
export function getAddressList() {
  return request({
    url: '/addresses',
    method: 'GET',
  });
}

/**
 * 新增地址。
 *
 * @param {CreateAddressReq} payload
 * @returns {Promise<Address>}
 */
export function addAddress(payload) {
  if (!payload || !payload.recipient || !payload.phone || !payload.detail) {
    return Promise.reject(new Error(
      'addAddress: recipient / phone / detail are required',
    ));
  }
  return request({
    url: '/addresses',
    method: 'POST',
    data: payload,
  });
}

/**
 * 修改地址。
 *
 * @param {number|string} id
 * @param {UpdateAddressReq} payload
 * @returns {Promise<Address>}
 */
export function updateAddress(id, payload) {
  if (!id) {
    return Promise.reject(new Error('updateAddress: id is required'));
  }
  return request({
    url: `/addresses/${id}`,
    method: 'PUT',
    data: payload || {},
  });
}

/**
 * 删除地址。
 *
 * @param {number|string} id
 * @returns {Promise<unknown>}
 */
export function deleteAddress(id) {
  if (!id) {
    return Promise.reject(new Error('deleteAddress: id is required'));
  }
  return request({
    url: `/addresses/${id}`,
    method: 'DELETE',
  });
}

/**
 * 把 id 设为默认地址。
 *
 * @param {number|string} id
 * @returns {Promise<unknown>}
 */
export function setDefaultAddress(id) {
  if (!id) {
    return Promise.reject(new Error('setDefaultAddress: id is required'));
  }
  return request({
    url: `/addresses/${id}/default`,
    method: 'PUT',
  });
}

export default {
  getAddressList,
  addAddress,
  updateAddress,
  deleteAddress,
  setDefaultAddress,
};