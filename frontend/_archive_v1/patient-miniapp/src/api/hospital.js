// src/api/hospital.js
//
// 医院库 API client —— 列表 + 详情。
// （spec §4.3 + plan Task M1 + 后端 handler：`services/user/internal/hospital/hospital.go`）
//
// 端点：
//   GET  /api/v1/hospitals        query={ page, limit, city_id, level, keyword }
//   GET  /api/v1/hospitals/:id
//
// 响应契约（与 backend Go JSON 1:1，字段命名 snake_case；本模块最小透传，
// store 端按 backend 字段名读取 —— 见 stores/hospital.test.js）：
//   Hospital = {
//     id, name, city_id, level, status,
//     address, lat, lng, phone,
//     departments, description,
//   }
//   HospitalListResp = { items: Hospital[], total: number, page: number, limit: number }
//
// 实现要点：
//   - 复用根 utils/request.js 的 request（拦截器统一加 X-Trace-Id + Authorization / 11001 reLaunch login）
//   - request.js 已把 `{ code, data, message }` 剥壳为 data 直接 return；本模块不再做二次解包
//   - GET 列表 → query 拼字符串（city_id / level / keyword 都在 query 上）

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} Hospital
 * @property {number|string} id
 * @property {string}        name            - 医院名
 * @property {number|string} [city_id]       - 城市 id
 * @property {string}        [level]         - 三甲 / 三乙 / 二甲 等
 * @property {string}        [status]        - active / inactive
 * @property {string}        [address]       - 详细地址
 * @property {number}        [lat]           - 纬度
 * @property {number}        [lng]           - 经度
 * @property {string}        [phone]         - 联系电话
 * @property {string[]}      [departments]   - 科室标签
 * @property {string}        [description]   - 简介
 */

/**
 * @typedef {Object} HospitalListResp
 * @property {Hospital[]} items
 * @property {number}     total
 * @property {number}     page
 * @property {number}     limit
 */

/**
 * @typedef {Object} HospitalListQuery
 * @property {number} [page]       - 默认 1
 * @property {number} [limit]      - 默认 20，最大 100
 * @property {number} [city_id]    - 城市 id
 * @property {string} [level]      - 医院等级 e.g. '三甲'
 * @property {string} [keyword]    - 关键字模糊匹配
 */

/**
 * 拉医院列表（医院列表页 / 首页推荐用）。
 *
 * @param {HospitalListQuery} [query]
 * @returns {Promise<HospitalListResp>}
 */
export function getHospitals(query) {
  return request({
    url: '/hospitals',
    method: 'GET',
    query: query || {},
  });
}

/**
 * 拉医院详情（含服务包 —— detail.vue 渲染套餐卡片用）。
 *
 * @param {number|string} id
 * @returns {Promise<Hospital>}
 */
export function getHospitalDetail(id) {
  if (!id) {
    return Promise.reject(new Error('getHospitalDetail: id is required'));
  }
  return request({
    url: `/hospitals/${id}`,
    method: 'GET',
  });
}

export default {
  getHospitals,
  getHospitalDetail,
};