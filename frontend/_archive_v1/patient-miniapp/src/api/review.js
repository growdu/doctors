// src/api/review.js
//
// 评价 API client —— 患者对已完成订单创建评价 + 拉评价列表/详情。
// （spec §4.6 + plan Task M1 + 后端 handler：`services/review/internal/handler/review.go`）
//
// 端点：
//   POST /api/v1/reviews           body=CreateReviewReq
//   GET  /api/v1/reviews           query={ escort_id, order_id, min_rating, page, page_size }
//   GET  /api/v1/reviews/:id
//   POST /api/v1/reviews/:id/reply （admin）
//
// 响应契约（与 backend Go JSON 1:1）：
//   Review = {
//     id, order_id, escort_id, reviewer_id, rating, comment,
//     reply, replied_by, created_at,
//   }
//
// 数据模型：
//   - rating 1..5
//   - comment 可选（最长 500 字由后端校验）
//   - reply / replied_by 仅 admin 已回复时填充

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Review
 * @property {number|string} id
 * @property {number|string} order_id
 * @property {number|string} escort_id
 * @property {number|string} reviewer_id
 * @property {number}        rating      - 1..5
 * @property {string}        [comment]   - 评价内容
 * @property {string}        [reply]     - admin 回复
 * @property {number|string} [replied_by]
 * @property {string}        [created_at]
 */

/**
 * @typedef {Object} CreateReviewReq
 * @property {number|string} order_id
 * @property {number|string} escort_id
 * @property {number}        rating      - 1..5
 * @property {string}        [comment]
 */

/**
 * @typedef {Object} ReviewListQuery
 * @property {number} [escort_id]
 * @property {number} [order_id]
 * @property {number} [min_rating]
 * @property {number} [page]       - 默认 1
 * @property {number} [page_size]  - 默认 20
 */

/**
 * 提交评价（评价创建页用）。
 *
 * @param {CreateReviewReq} payload
 * @returns {Promise<Review>}
 */
export function submitReview(payload) {
  if (!payload || !payload.order_id || !payload.escort_id || !payload.rating) {
    return Promise.reject(new Error(
      'submitReview: order_id / escort_id / rating are required',
    ));
  }
  const r = Number(payload.rating);
  if (!Number.isFinite(r) || r < 1 || r > 5) {
    return Promise.reject(new Error('submitReview: rating must be 1..5'));
  }
  return request({
    url: '/reviews',
    method: 'POST',
    data: {
      order_id: payload.order_id,
      escort_id: payload.escort_id,
      rating: r,
      comment: payload.comment || '',
    },
  });
}

/**
 * 拉评价列表（按 escort / order / min_rating 过滤）。
 *
 * @param {ReviewListQuery} [query]
 * @returns {Promise<{reviews: Review[], page: number, page_size: number}>}
 */
export function listReviews(query) {
  return request({
    url: '/reviews',
    method: 'GET',
    query: query || {},
  });
}

/**
 * 拉评价详情。
 *
 * @param {number|string} id
 * @returns {Promise<Review>}
 */
export function getReviewDetail(id) {
  if (!id) {
    return Promise.reject(new Error('getReviewDetail: id is required'));
  }
  return request({
    url: `/reviews/${id}`,
    method: 'GET',
  });
}

export default {
  submitReview,
  listReviews,
  getReviewDetail,
};