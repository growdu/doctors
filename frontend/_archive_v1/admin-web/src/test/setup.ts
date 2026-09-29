/**
 * Vitest 全局 setup：注册 @testing-library/jest-dom 的 expect 扩展。
 *
 * 作用：让测试可写 `expect(el).toBeInTheDocument()` / `toHaveTextContent()` 等。
 */
import '@testing-library/jest-dom/vitest';

// jsdom 不实现 matchMedia，AntD 部分组件内部会调用 → 静默 stub 掉
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// 兜底 localStorage：jsdom 默认提供，但部分环境（Node 25 + vitest 冷启动）
// 下 authStore.persist 拿到 localStorage 时 setItem 报 undefined。
// 用 in-memory map 兜底，确保 setItem / getItem / removeItem 一定可用。
const memoryStorage: Storage = (() => {
  const store = new Map<string, string>();
  return {
    get length() {
      return store.size;
    },
    clear() {
      store.clear();
    },
    getItem(key: string) {
      return store.has(key) ? (store.get(key) as string) : null;
    },
    key(i: number) {
      return Array.from(store.keys())[i] ?? null;
    },
    removeItem(key: string) {
      store.delete(key);
    },
    setItem(key: string, value: string) {
      store.set(key, String(value));
    },
  } as Storage;
})();
Object.defineProperty(window, 'localStorage', {
  writable: true,
  configurable: true,
  value: memoryStorage,
});
Object.defineProperty(globalThis, 'localStorage', {
  writable: true,
  configurable: true,
  value: memoryStorage,
});
