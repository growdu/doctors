/**
 * storage 抽象层（v2 unified-app · 移动端兼容）。
 *
 * 设计动机：
 *   - 移动端（mp-weixin / app-plus）无 `localStorage`，需走 `uni.setStorageSync` 等 uni API
 *   - web (h5) 有 `localStorage`，uni 也会 polyfill 成 localStorage 但行为有差异（如 sync 模式）
 *   - 4 端 API 统一：getItem / setItem / removeItem / clear
 *
 * 端兼容矩阵：
 *   - h5         : localStorage（fallback）
 *   - mp-weixin  : uni.setStorageSync（同步 API）
 *   - app-plus   : uni.setStorageSync（同步 API）
 *
 * 使用示例：
 *   import { storage } from '@/utils/storage';
 *   storage.setItem('key', JSON.stringify(value));
 *   const v = storage.getItem('key');
 *
 * 配套测试：src/utils/storage.spec.ts
 */
declare const uni:
  | {
      getStorageSync: (key: string) => unknown;
      setStorageSync: (key: string, value: string) => void;
      removeStorageSync: (key: string) => void;
    }
  | undefined;

declare const localStorage:
  | {
      getItem: (key: string) => string | null;
      setItem: (key: string, value: string) => void;
      removeItem: (key: string) => void;
      clear: () => void;
    }
  | undefined;

function hasUniStorage(): boolean {
  return typeof uni !== 'undefined' && typeof uni.setStorageSync === 'function';
}

export const storage = {
  /**
   * 取值。返回 string | null（不存在时 null）。
   * 自动 try-catch（quota / JSON / 安全异常），出错返回 null。
   */
  getItem(key: string): string | null {
    try {
      if (hasUniStorage()) {
        const v = uni!.getStorageSync(key);
        return typeof v === 'string' ? v : (v == null ? null : String(v));
      }
      if (typeof localStorage !== 'undefined') {
        return localStorage.getItem(key);
      }
      return null;
    } catch {
      return null;
    }
  },

  /**
   * 写值。值会被 stringify（接收 string 或对象）。
   * 自动 try-catch；quota 超限 / 安全异常静默。
   */
  setItem(key: string, value: string | object | number | boolean): void {
    const raw = typeof value === 'string' ? value : JSON.stringify(value);
    try {
      if (hasUniStorage()) {
        uni!.setStorageSync(key, raw);
      } else if (typeof localStorage !== 'undefined') {
        localStorage.setItem(key, raw);
      }
    } catch {
      /* noop */
    }
  },

  /**
   * 删值。
   */
  removeItem(key: string): void {
    try {
      if (hasUniStorage()) {
        uni!.removeStorageSync(key);
      } else if (typeof localStorage !== 'undefined') {
        localStorage.removeItem(key);
      }
    } catch {
      /* noop */
    }
  },

  /**
   * 取 JSON 对象（解析失败返回 null）。
   */
  getJSON<T>(key: string): T | null {
    const raw = this.getItem(key);
    if (raw == null || raw === '') return null;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return null;
    }
  },

  /**
   * 清空所有 storage（慎用）。
   */
  clear(): void {
    try {
      if (hasUniStorage()) {
        // uni.clearStorageSync 清空当前应用全部 storage
        if (typeof (uni as unknown as { clearStorageSync?: () => void }).clearStorageSync === 'function') {
          (uni as unknown as { clearStorageSync: () => void }).clearStorageSync();
        }
      } else if (typeof localStorage !== 'undefined') {
        localStorage.clear();
      }
    } catch {
      /* noop */
    }
  },
};

/** 便利 export */
export const storageGet = storage.getItem.bind(storage);
export const storageSet = storage.setItem.bind(storage);
export const storageRemove = storage.removeItem.bind(storage);
export const storageGetJSON = storage.getJSON.bind(storage);