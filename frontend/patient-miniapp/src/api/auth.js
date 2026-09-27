// src/api/auth.js
//
// 鉴权 API client —— 患者端短信验证码登录 / 微信登录 / 当前用户信息 / 实名认证。
//（spec §5.1 + plan Task 5/6 + plan v1 增量）
//
// 端点：
//   POST /api/v1/auth/sms/send                  body={ phone }          → 触发短信下发（v1 mock 沙箱）
//   POST /api/v1/auth/login-by-phone            body={ phone, code }    → 登录
//   POST /api/v1/auth/login-by-wechat           body={ code, ... }      → 微信登录（v1 mock）
//   GET  /api/v1/me                                                   → 拉当前用户
//   POST /api/v1/auth/real-name                 body={ name, id_card } → 提交实名信息（v1 不接 OCR）
//
// 响应契约（与 backend Go JSON 1:1，request.js 已剥壳 data）：
//   LoginResp = {
//     access_token, token_type, expires_in,
//     user: { id, phone, real_name_verified, ... },
//   }
//
// 实现要点：
//   - 复用 utils/request.js 的 request（拦截器统一加 X-Trace-Id + Authorization / 11001 reLaunch login）
//   - sendSmsCode：v1 阶段后端沙箱总是 200；前端做 phone 长度校验（11 位中国大陆手机号）
//   - loginByWechat：v1 阶段后端 mock「WechatAuth」入口，前端只调一次拿 access_token
//   - submitRealName：v1 不接 OCR，客户端直接传 name + id_card 字符串；后端做格式校验

import { request } from '../../utils/request.js';

/**
 * @typedef {Object} LoginResp
 * @property {string} access_token
 * @property {string} [token_type]
 * @property {number} [expires_in]
 * @property {object} [user]
 */

/**
 * @typedef {Object} MeResp
 * @property {number|string} id
 * @property {string}        phone
 * @property {string}        [real_name]
 * @property {boolean}       [real_name_verified]
 * @property {boolean}       [is_minor]
 * @property {string}        [created_at]
 */

/**
 * 触发短信验证码下发（v1 后端沙箱总是 200，仅校验 phone 格式）。
 *
 * @param {{ phone: string }} payload
 * @returns {Promise<{ sent: boolean, ttl: number }>}
 */
export function sendSmsCode(payload) {
  if (!payload || !payload.phone) {
    return Promise.reject(new Error('sendSmsCode: phone is required'));
  }
  if (!/^1\d{10}$/.test(String(payload.phone))) {
    return Promise.reject(new Error('sendSmsCode: phone must be 11-digit CN mobile'));
  }
  return request({
    url: '/auth/sms/send',
    method: 'POST',
    data: { phone: payload.phone },
  });
}

/**
 * 手机号 + 短信验证码登录。
 *
 * @param {{ phone: string, code: string }} payload
 * @returns {Promise<LoginResp>}
 */
export function loginByPhone(payload) {
  if (!payload || !payload.phone || !payload.code) {
    return Promise.reject(new Error('loginByPhone: phone + code are required'));
  }
  return request({
    url: '/auth/login-by-phone',
    method: 'POST',
    data: { phone: payload.phone, code: payload.code },
  });
}

/**
 * 微信登录（v1 mock：后端只认 code 字符串 + 返回固定 mock user）。
 *
 * @param {{ code: string, nickname?: string, avatar?: string }} payload
 * @returns {Promise<LoginResp>}
 */
export function loginByWechat(payload) {
  if (!payload || !payload.code) {
    return Promise.reject(new Error('loginByWechat: code is required'));
  }
  return request({
    url: '/auth/login-by-wechat',
    method: 'POST',
    data: {
      code: payload.code,
      nickname: payload.nickname || '',
      avatar: payload.avatar || '',
    },
  });
}

/**
 * 拉当前登录用户信息（个人中心 / 实名认证跳转前用）。
 *
 * @returns {Promise<MeResp>}
 */
export function fetchMe() {
  return request({
    url: '/me',
    method: 'GET',
  });
}

/**
 * 提交实名认证（v1 不接 OCR，客户端传身份证号 + 姓名）。
 *
 * @param {{ name: string, id_card: string }} payload
 * @returns {Promise<MeResp>}
 */
export function submitRealName(payload) {
  if (!payload || !payload.name || !payload.id_card) {
    return Promise.reject(new Error('submitRealName: name + id_card are required'));
  }
  return request({
    url: '/auth/real-name',
    method: 'POST',
    data: { name: payload.name, id_card: payload.id_card },
  });
}

export default {
  sendSmsCode,
  loginByPhone,
  loginByWechat,
  fetchMe,
  submitRealName,
};