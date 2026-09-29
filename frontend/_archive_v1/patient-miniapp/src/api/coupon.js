// src/api/coupon.js
//
// 优惠券 API client —— 可用券模板 / 领取 / 我的券 / 核销。
// （spec §4.5 + plan Task M1 + 后端 handler：`services/user/internal/coupon/coupon.go`）
//
// 端点：
//   GET  /api/v1/coupons                  query={ page, limit }
//   GET  /api/v1/coupons/:id
//   POST /api/v1/coupons/:id/claim        领取（无 body）
//   GET  /api/v1/me/coupons               我的券（按状态排序：unused 优先）
//   POST /api/v1/me/coupons/:id/use       核销（v1 不传 order_id）
//
// 响应契约（与 backend Go JSON 1:1）：
//   Coupon = {
//     id, name, type, value, threshold,
//     valid_from, valid_until, stock, status,
//   }
//   UserCoupon = {
//     id, user_id, coupon_id, status,        // status: unused / used / expired
//     expires_at, used_at, claimed_at,
//     coupon: Coupon,
//   }
//
// 约定：
//   - 领取 stock 0 → 后端 CodeConflict（13004 coupon unavailable）
//   - 已领取（重复 claim）→ 后端 CodeConflict
//   - v1 不接 order 关联核销（v2 接 order 后再扩）

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Coupon
 * @property {number|string} id
 * @property {string}        name         - 「新人立减券」等
 * @property {string}        type         - amount / discount / percentage 等
 * @property {number}        value        - 抵扣金额（元 / 折扣百分比数值）
 * @property {number}        threshold    - 满减门槛
 * @property {string}        valid_from   - ISO8601
 * @property {string}        valid_until  - ISO8601
 * @property {number}        stock        - 剩余库存
 * @property {string}        status       - active / inactive
 */

/**
 * @typedef {Object} UserCoupon
 * @property {number|string} id           - user_coupon 主键
 * @property {number|string} user_id
 * @property {number|string} coupon_id
 * @property {string}        status       - unused / used / expired
 * @property {string}        expires_at
 * @property {string|null}   used_at
 * @property {string}        claimed_at
 * @property {Coupon}        coupon       - 联表优惠券模板
 */

/**
 * 拉可用券模板列表（优惠券中心「领券」tab 用）。
 *
 * @param {{page?: number, limit?: number}} [query]
 * @returns {Promise<{items: Coupon[]} | Coupon[]>} 兼容 {items} 与直返数组
 */
export function getCoupons(query) {
  return request({
    url: '/coupons',
    method: 'GET',
    query: query || {},
  });
}

/**
 * 拉单个券模板详情。
 *
 * @param {number|string} id
 * @returns {Promise<Coupon>}
 */
export function getCouponDetail(id) {
  if (!id) {
    return Promise.reject(new Error('getCouponDetail: id is required'));
  }
  return request({
    url: `/coupons/${id}`,
    method: 'GET',
  });
}

/**
 * 领取优惠券。
 *
 * @param {number|string} id
 * @returns {Promise<UserCoupon>}
 */
export function claimCoupon(id) {
  if (!id) {
    return Promise.reject(new Error('claimCoupon: id is required'));
  }
  return request({
    url: `/coupons/${id}/claim`,
    method: 'POST',
  });
}

/**
 * 拉我的优惠券（按状态排序：unused 优先）。
 *
 * @returns {Promise<{items: UserCoupon[]} | UserCoupon[]>}
 */
export function getMyCoupons() {
  return request({
    url: '/me/coupons',
    method: 'GET',
  });
}

/**
 * 核销我的券（v1 不传 order_id）。
 *
 * @param {number|string} userCouponId
 * @returns {Promise<unknown>}
 */
export function useMyCoupon(userCouponId) {
  if (!userCouponId) {
    return Promise.reject(new Error('useMyCoupon: userCouponId is required'));
  }
  return request({
    url: `/me/coupons/${userCouponId}/use`,
    method: 'POST',
  });
}

export default {
  getCoupons,
  getCouponDetail,
  claimCoupon,
  getMyCoupons,
  useMyCoupon,
};