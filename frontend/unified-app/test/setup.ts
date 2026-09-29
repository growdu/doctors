/**
 * Vitest 测试 setup：注入 uni-app 全局 mock + 浏览器 API。
 *
 * 关键决策：手写一个最小 StorageMock，不依赖 jsdom / happy-dom 的 localStorage 实现。
 * 原因：jsdom 25 在 vitest 2.1.9 默认 url=about:blank 时不暴露 localStorage.setItem；
 *      happy-dom 20 在 vitest 2.1.9 集成也不完整。两者都不可靠。
 *      手写 mock 保证 CI 一致，且能精确测试边界（异常、超 quota、序列化）。
 *
 * 这里提供最小 mock 集：
 *   - uni.navigateTo / redirectTo / reLaunch / switchTab / showToast / getStorageSync / setStorageSync / removeStorageSync
 *   - globalThis.uni 的 typeof 探测（spike 已有 `typeof uni !== 'undefined'` 兼容写法）
 *
 * 业务测试如需更细 mock，可在各 .spec.ts 顶层用 vi.mock('@dcloudio/uni-app', ...) 覆盖。
 */

/**
 * 最小 Storage polyfill：满足 Map 语义的 getItem/setItem/removeItem/clear/length/key。
 * 与浏览器 Storage 接口 100% 兼容，业务代码无感。
 */
class StorageMock implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }

  key(index: number): string | null {
    const arr = Array.from(this.store.keys());
    return arr[index] ?? null;
  }

  getItem(key: string): string | null {
    return this.store.has(key) ? (this.store.get(key) as string) : null;
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value));
  }

  removeItem(key: string): void {
    this.store.delete(key);
  }

  clear(): void {
    this.store.clear();
  }
}

const storage = new StorageMock();

// 挂到 globalThis 与 window 双向（vitest 2 jsdom 默认只暴露 window，spike 业务代码两路都会用）
// @ts-expect-error - 测试环境 mock，未声明 global 类型
globalThis.localStorage = storage;
try {
  // jsdom 提供 window；happy-dom 也提供；无 window 时跳过
  // @ts-expect-error - window 上重定义 localStorage
  window.localStorage = storage;
} catch {
  /* window 不存在（node 环境）时跳过 */
}

// @ts-expect-error - 测试环境 mock，未声明 global 类型
globalThis.uni = {
  navigateTo: (opts: { url: string }) => {
    // eslint-disable-next-line no-console
    if (typeof console !== 'undefined') console.log('[mock uni.navigateTo]', opts.url);
    return Promise.resolve();
  },
  redirectTo: (opts: { url: string }) => {
    // eslint-disable-next-line no-console
    if (typeof console !== 'undefined') console.log('[mock uni.redirectTo]', opts.url);
    return Promise.resolve();
  },
  reLaunch: (opts: { url: string }) => {
    // eslint-disable-next-line no-console
    if (typeof console !== 'undefined') console.log('[mock uni.reLaunch]', opts.url);
    return Promise.resolve();
  },
  switchTab: (opts: { url: string }) => {
    // eslint-disable-next-line no-console
    if (typeof console !== 'undefined') console.log('[mock uni.switchTab]', opts.url);
    return Promise.resolve();
  },
  showToast: (opts: { title: string; icon?: string }) => {
    // eslint-disable-next-line no-console
    if (typeof console !== 'undefined') console.log('[mock uni.showToast]', opts.title, opts.icon ?? '');
  },
  getStorageSync: (key: string): unknown => {
    return storage.getItem(key) ?? '';
  },
  setStorageSync: (key: string, value: string) => {
    storage.setItem(key, value);
  },
  removeStorageSync: (key: string) => {
    storage.removeItem(key);
  },
};

declare const globalThis: {
  uni?: unknown;
  localStorage?: Storage;
};