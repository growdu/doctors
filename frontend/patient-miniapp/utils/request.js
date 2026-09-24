// utils/request.js
// 全局 HTTP 客户端 + 拦截器（参考 spec §5.2 + plan Task 5）。
//
// 关键约束：
//   - 不引 axios；走 uni.request + uni.addInterceptor。
//   - 请求拦截：自动加 X-Trace-Id + Authorization。
//   - 响应拦截：统一错误码处理（业务码 11001 = 未登录 → 清 token + reLaunch login）。
//   - 业务码 0 / HTTP 2xx 视为成功；其它一律抛 ApiError。
//
// 后续 plan (Task 5+) 会扩展：
//   - 401 自动 refresh。
//   - 13001..13009 业务码 → 用户友好 toast。
//   - OpenAPI codegen 注入 types。

import { getToken, clearToken } from './auth.js';

// 默认 baseURL；后续可改读 env / manifest.json。
const DEFAULT_BASE_URL = 'https://api.dev.doctors.example.com/api/v1';

// 业务码：未登录 / token 失效
const CODE_NO_TOKEN = 11001;
// 业务码：服务端正常但业务失败（统一 reject 的兜底码）
const CODE_OK = 0;

/**
 * 生成 trace_id，格式 mp-<timestamp13>-<random6>，与后端 / 日志约定一致。
 */
export function newTraceId(now) {
  const ts = typeof now === 'number' ? now : Date.now();
  const rand = Math.random().toString(36).slice(2, 8).padEnd(6, '0');
  return `mp-${ts}-${rand}`;
}

/**
 * 构造请求头：X-Trace-Id + Authorization（按需）。
 */
function buildHeader(opts) {
  const header = Object.assign({}, opts.header || {});
  if (!header['X-Trace-Id']) {
    header['X-Trace-Id'] = newTraceId();
  }
  if (opts.auth !== false) {
    if (!header['Authorization']) {
      const token = getToken();
      if (token) {
        header['Authorization'] = `Bearer ${token}`;
      }
    }
  }
  if (!header['Content-Type']) {
    header['Content-Type'] = 'application/json';
  }
  return header;
}

/**
 * 把 query 对象拼到 url 上（不破坏已有的 ? / #）。
 */
function appendQuery(url, query) {
  if (!query || typeof query !== 'object') return url;
  const parts = [];
  Object.keys(query).forEach((k) => {
    const v = query[k];
    if (v === undefined || v === null) return;
    parts.push(`${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
  });
  if (parts.length === 0) return url;
  return url + (url.indexOf('?') >= 0 ? '&' : '?') + parts.join('&');
}

/**
 * 统一 API 错误对象。保留 code + statusCode，方便上层按 code 分支。
 */
export class ApiError extends Error {
  constructor(code, message, statusCode) {
    super(message || 'request failed');
    this.name = 'ApiError';
    this.code = code;
    this.statusCode = statusCode;
  }
}

/**
 * 单次请求封装。返回后端 `data` 字段（已剥壳）。
 *
 * @param {Object} opts
 * @param {string} opts.url             - 相对路径或绝对路径（带 baseURL 自动拼接）
 * @param {string} [opts.method]        - GET / POST / PUT / DELETE，默认 GET
 * @param {Object} [opts.data]          - body
 * @param {Object} [opts.query]         - query string
 * @param {Object} [opts.header]        - 额外 header
 * @param {boolean} [opts.auth=true]    - 是否带 Bearer
 * @param {string} [opts.baseURL]       - 覆盖默认 baseURL
 */
export function request(opts) {
  const baseURL = opts.baseURL || DEFAULT_BASE_URL;
  const fullUrl = appendQuery(
    /^(https?:)?\/\//i.test(opts.url) ? opts.url : baseURL + opts.url,
    opts.query,
  );

  return new Promise(function (resolve, reject) {
    if (typeof uni === 'undefined' || typeof uni.request !== 'function') {
      reject(new ApiError(-1, 'uni.request not available', -1));
      return;
    }

    uni.request({
      url: fullUrl,
      method: opts.method || 'GET',
      data: opts.data,
      header: buildHeader(opts),
      success: function (res) {
        const statusCode = res.statusCode || 0;
        const body = res.data;
        const code = body && typeof body.code === 'number' ? body.code : statusCode;
        if (statusCode >= 400 || (typeof body === 'object' && body && body.code !== undefined && body.code !== CODE_OK)) {
          reject(new ApiError(code, (body && body.message) || 'request failed', statusCode));
          return;
        }
        resolve(body && body.data !== undefined ? body.data : body);
      },
      fail: function (err) {
        const statusCode = (err && err.statusCode) || 0;
        reject(new ApiError(statusCode || -1, (err && err.errMsg) || 'network error', statusCode));
      },
    });
  });
}

/**
 * 安装全局拦截器。建议在 App 入口（main.js）调用一次。
 *
 *   1. request 拦截：兜底加 X-Trace-Id + Authorization（绕过 request() 的裸 uni.request 也覆盖）。
 *   2. responseError 拦截：业务码 11001 → 清 token + reLaunch /pages/auth/login。
 */
export function installInterceptors() {
  if (typeof uni === 'undefined' || typeof uni.addInterceptor !== 'function') {
    return;
  }

  // ---- request 拦截：兜底 header
  uni.addInterceptor('request', {
    request: function requestInterceptor(options) {
      options.header = options.header || {};
      if (!options.header['X-Trace-Id']) {
        options.header['X-Trace-Id'] = newTraceId();
      }
      if (!options.header['Authorization']) {
        const token = getToken();
        if (token) {
          options.header['Authorization'] = 'Bearer ' + token;
        }
      }
      return options;
    },
  });

  // ---- responseError 拦截：业务码 11001 / HTTP 401 → 清 token + reLaunch login
  uni.addInterceptor('responseError', {
    fail: function responseErrorInterceptor(ctx) {
      const statusCode = (ctx && ctx.statusCode) || 0;
      const data = ctx && ctx.data;
      const code = data && typeof data.code === 'number' ? data.code : statusCode;

      // 401 (HTTP) 或业务码 11001 → 视为未登录
      if (statusCode === 401 || code === CODE_NO_TOKEN) {
        clearToken();
        try {
          uni.reLaunch({ url: '/pages/auth/login' });
        } catch (e) {
          // 静默：H5 / 测试场景 reLaunch 可能未注册
        }
      }

      const message = (data && data.message) || (statusCode === 0 ? 'network error' : 'request error');
      return Promise.reject(new ApiError(code, message, statusCode));
    },
  });
}

/**
 * 业务错误码统一提示（占位）。后续 plan 会按 l2-api-gap-design.md §3.2 细化。
 *
 * @param {ApiError} err
 */
export function describeApiError(err) {
  if (!err) return '';
  switch (err.code) {
    case CODE_NO_TOKEN:
      return '登录已过期，请重新登录';
    case 13001:
      return '资源不存在';
    case 13003:
      return '余额不足';
    case 13004:
      return '优惠券不可用';
    case 13005:
      return '请先完成实名认证';
    default:
      return err.message || `请求失败（${err.code}）`;
  }
}

export default {
  request,
  installInterceptors,
  newTraceId,
  ApiError,
  describeApiError,
};