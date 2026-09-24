// utils/auth.js
// 轻量 token 持久化封装。后续 plan (Task 4) 会扩展：refresh / 加密 / 多账号。

const TOKEN_KEY = 'patient.token';

/**
 * 从 uni.storage 同步取 token。
 * 在非 uni runtime（如 jest 单测）下走 fallback（globalThis 内存 map），
 * 让单测可以注入而不必 mock 整个 uni 对象。
 */
export function getToken() {
  if (typeof uni !== 'undefined' && typeof uni.getStorageSync === 'function') {
    try {
      return uni.getStorageSync(TOKEN_KEY) || '';
    } catch (e) {
      return '';
    }
  }
  return globalThis.__patient_token || '';
}

/**
 * 持久化 token 到 uni.storage。
 */
export function setToken(token) {
  if (typeof uni !== 'undefined' && typeof uni.setStorageSync === 'function') {
    try {
      uni.setStorageSync(TOKEN_KEY, token);
    } catch (e) {
      // 静默失败
    }
  }
  globalThis.__patient_token = token;
}

/**
 * 清除 token（登出 / 401 业务码触发）。
 */
export function clearToken() {
  if (typeof uni !== 'undefined' && typeof uni.removeStorageSync === 'function') {
    try {
      uni.removeStorageSync(TOKEN_KEY);
    } catch (e) {
      // 静默失败
    }
  }
  delete globalThis.__patient_token;
}