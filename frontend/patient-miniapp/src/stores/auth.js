// src/stores/auth.js
//
// 鉴权 store —— 患者端的 token / user 状态机
// (spec §5.1 + plan Task 7 v1.1 增量)
//
// 范围（本文件）：
//   - token / user / isLoggedIn：响应式状态；token 初始化从 utils/auth.js 同步读（兼容 SSR）
//   - loginByPhone：调 api.loginByPhone；成功后写 token + 缓存 user
//   - fetchMe：拉一次 user 详情（实名认证后用）
//   - logout：清 token + user + 跳登录页（同时触发 onUnauthorized 回调）
//   - onUnauthorized：401 / 11001 拦截回调；清状态 + 通知 UI 层（页面订阅）
//
// 设计要点：
//   - token 持久化通过 utils/auth.js（uni.storage + globalThis fallback）
//   - onUnauthorized 注册在全局 Set（_unauthorizedHandlers）；
//     utils/request.js 拦截器或外部代码可注册/触发，不直接依赖 store 实例
//   - api 模块由 plan Task 5/6 提供；当前阶段 jest 单测用 jest.doMock 注入
//
// 不在本文件范围（plan Task 5/6）：
//   - 401 自动 refresh
//   - 多账号 / device-id 绑定
//   - user 详情缓存策略（store 不持久化 user，每次 fetchMe 重新拉）

import { defineStore } from 'pinia';
import { ref } from 'vue';
import {
  getToken,
  setToken,
  clearToken,
} from '@/utils/auth.js';

// api 模块由 plan Task 5/6 提供；本阶段 jest 单测用 jest.doMock 注入。
// 用动态 import 延迟解析（避免顶层 import 在 api 文件缺失时抛 module not found）。
async function _apiLoginByPhone(payload) {
  const mod = await import('@/api/auth.js');
  return mod.loginByPhone(payload);
}
async function _apiFetchMe() {
  const mod = await import('@/api/auth.js');
  return mod.fetchMe();
}

// utils/request.js 在 11001 / HTTP 401 拦截时会回调本 handler 列表。
// 模块级集合：store 单例 + 全进程唯一；handler 注册与 store 解耦。
const _unauthorizedHandlers = new Set();

/**
 * 注册 401 处理器（utils/request.js 拦截器 / 页面级清理逻辑调用）。
 *
 * 典型用法：
 *   const unregister = registerUnauthorizedHandler(() => {
 *     // 清空本地草稿 / 重置轮询 / 跳登录页
 *   });
 *   // 卸载时（理论上不需要，因为 handler 跟随进程生命周期）：
 *   // unregister();
 *
 * @param {() => void | Promise<void>} handler
 * @returns {() => void} 反注册函数
 */
export function registerUnauthorizedHandler(handler) {
  if (typeof handler !== 'function') return () => {};
  _unauthorizedHandlers.add(handler);
  return () => _unauthorizedHandlers.delete(handler);
}

/**
 * 触发全部已注册 401 handler（同步顺序执行；handler 自身异常被吞）。
 *
 * @param {{ code?: number, statusCode?: number, reason?: string }} [ctx]
 */
export function invokeUnauthorizedHandlers(ctx) {
  for (const h of _unauthorizedHandlers) {
    try {
      const r = h(ctx || {});
      if (r && typeof r.then === 'function') {
        r.catch(() => {});
      }
    } catch (_e) {
      // handler 自身异常不影响后续 handler；拦截器 reject 仍正常触发
    }
  }
}

export const useAuthStore = defineStore('auth', () => {
  // ---- 响应式状态
  const token = ref(getToken() || '');
  const user = ref(null);
  const isLoggedIn = ref(Boolean(token.value));

  // ---- actions

  /**
   * 手机号 + 短信验证码登录。
   * 成功后：写 token + user，持久化 token 到 storage。
   *
   * @param {string} phone
   * @param {string} code
   * @returns {Promise<{ access_token?: string, token?: string, user?: object, profile?: object }>}
   */
  async function loginByPhone(phone, code) {
    const resp = await _apiLoginByPhone({ phone, code });
    const accessToken = resp && (resp.access_token || resp.token) || '';
    const profile = resp && (resp.user || resp.profile) || null;
    if (!accessToken) {
      throw new Error('loginByPhone: response missing access_token / token');
    }
    token.value = accessToken;
    user.value = profile;
    isLoggedIn.value = true;
    setToken(accessToken);
    return resp;
  }

  /**
   * 拉一次当前用户信息（实名认证 / 个人中心打开时用）。
   * 不写 token，只更新 user。
   */
  async function fetchMe() {
    const me = await _apiFetchMe();
    user.value = me;
    return me;
  }

  /**
   * 主动登出 —— 清 token + user + 跳登录页 + 通知 handler。
   * 与 onUnauthorized 区别：主动登出也会触发 handler，让 UI 有机会清理本地草稿。
   */
  async function logout() {
    const hadToken = Boolean(token.value);
    token.value = '';
    user.value = null;
    isLoggedIn.value = false;
    clearToken();
    if (hadToken) {
      invokeUnauthorizedHandlers({ reason: 'logout' });
    }
    if (typeof uni !== 'undefined' && typeof uni.reLaunch === 'function') {
      try {
        uni.reLaunch({ url: '/pages/auth/login' });
      } catch (_e) {
        // 静默：jest / SSR / H5 test 场景 uni 可能未注册
      }
    }
  }

  /**
   * 处理 401 / 业务码 11001 —— 工具拦截器调用。
   * 行为：清状态 + 通知 UI 层；不主动 reLaunch（由 handler 决策）。
   *
   * @param {{ code?: number, statusCode?: number }} [ctx]
   */
  function onUnauthorized(ctx) {
    const hadToken = Boolean(token.value);
    token.value = '';
    user.value = null;
    isLoggedIn.value = false;
    clearToken();
    if (hadToken) {
      invokeUnauthorizedHandlers(ctx || { code: 11001, statusCode: 401 });
    }
  }

  return {
    // state
    token,
    user,
    isLoggedIn,
    // actions
    loginByPhone,
    fetchMe,
    logout,
    onUnauthorized,
  };
});

// 默认导出 register 工具函数（便于 main.js / utils/request.js 调用）
export default {
  registerUnauthorizedHandler,
  invokeUnauthorizedHandlers,
};