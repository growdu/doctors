/**
 * utils/storage 单元测试（vitest）。
 *
 * 验证矩阵：
 *   - h5 模式（带 localStorage）：读写删 + JSON
 *   - uni 模式（无 localStorage）：用 mock uni
 *   - 无 uni + 无 localStorage：no-op（不抛异常）
 *   - 异常场景：JSON parse 失败、quota 异常静默
 */
import { describe, expect, it, beforeEach, vi } from 'vitest';
import { storage, storageGet, storageSet, storageGetJSON } from './storage';

describe('utils/storage · h5 模式（localStorage）', () => {
  beforeEach(() => {
    // 禁用 test/setup.ts 注入的 uni 全局 mock，让 h5 路径生效
    vi.stubGlobal('uni', undefined);
    localStorage.clear();
  });

  it('setItem + getItem 往返', () => {
    storageSet('key1', 'hello');
    expect(storageGet('key1')).toBe('hello');
  });

  it('setItem 接受对象并 JSON.stringify', () => {
    storageSet('obj', { a: 1, b: [2, 3] });
    expect(storageGet('obj')).toBe('{"a":1,"b":[2,3]}');
  });

  it('getJSON 反序列化', () => {
    storageSet('obj', { name: 'Alice', age: 30 });
    expect(storageGetJSON<{ name: string; age: number }>('obj')).toEqual({ name: 'Alice', age: 30 });
  });

  it('getJSON 解析失败返回 null', () => {
    localStorage.setItem('bad', '{not json');
    expect(storageGetJSON('bad')).toBeNull();
  });

  it('removeItem 后 getItem 返回 null', () => {
    storageSet('key', 'value');
    storage.removeItem('key');
    expect(storageGet('key')).toBeNull();
  });

  it('不存在的 key 返回 null', () => {
    expect(storageGet('never')).toBeNull();
  });
});

describe('utils/storage · uni 模式（无 localStorage）', () => {
  beforeEach(() => {
    // 模拟移动端：localStorage 不存在 + 提供 uni mock
    vi.stubGlobal('localStorage', undefined);
    const uniMock = {
      getStorageSync: vi.fn((k: string) => (uniMock as unknown as { _s: Record<string, unknown> })._s[k] ?? null),
      setStorageSync: vi.fn((k: string, v: string) => {
        (uniMock as unknown as { _s: Record<string, unknown> })._s[k] = v;
      }),
      removeStorageSync: vi.fn((k: string) => {
        delete (uniMock as unknown as { _s: Record<string, unknown> })._s[k];
      }),
      clearStorageSync: vi.fn(),
      _s: {} as Record<string, unknown>,
    };
    vi.stubGlobal('uni', uniMock);
  });

  it('走 uni.setStorageSync/getStorageSync', () => {
    storageSet('key', 'value');
    expect(uni.setStorageSync).toHaveBeenCalledWith('key', 'value');

    storageGet('key');
    expect(uni.getStorageSync).toHaveBeenCalledWith('key');
  });

  it('removeItem 走 uni.removeStorageSync', () => {
    storage.removeItem('key');
    expect(uni.removeStorageSync).toHaveBeenCalledWith('key');
  });
});

describe('utils/storage · 异常处理', () => {
  it('无 uni + 无 localStorage 时不抛异常', () => {
    vi.stubGlobal('localStorage', undefined);
    vi.stubGlobal('uni', undefined);

    expect(() => {
      storageSet('k', 'v');
      storageGet('k');
      storage.removeItem('k');
      storage.clear();
    }).not.toThrow();
  });
});