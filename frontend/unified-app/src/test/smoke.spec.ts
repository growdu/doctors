/**
 * 测试基础设施 smoke：验证 vitest + happy-dom + uni mock 链路跑通。
 *
 * 这是 Pre-Phase 3.0 的第一个 commit 自带验证：
 *   - happy-dom 环境（window.localStorage 可用）
 *   - globalThis.uni mock 生效
 *   - 一个真业务模块导入可用（types/auth）
 *
 * 跑通后所有后续 .spec.ts 都能享受这套基础设施。
 */
import { describe, expect, it } from 'vitest';
import type { Role, MeResponse } from '@/types/auth';

// setup.ts 注入的 mock 类型（uni 是 TS 内置名，直接用 any 避免命名冲突）
interface UniMock {
  navigateTo: (opts: { url: string }) => Promise<void>;
  redirectTo: (opts: { url: string }) => Promise<void>;
  reLaunch: (opts: { url: string }) => Promise<void>;
  switchTab: (opts: { url: string }) => Promise<void>;
  showToast: (opts: { title: string; icon?: string }) => void;
  getStorageSync: (key: string) => unknown;
  setStorageSync: (key: string, value: string) => void;
  removeStorageSync: (key: string) => void;
}

describe('test infrastructure smoke', () => {
  it('happy-dom provides localStorage via window', () => {
    window.localStorage.setItem('foo', 'bar');
    expect(window.localStorage.getItem('foo')).toBe('bar');
    window.localStorage.removeItem('foo');
  });

  it('globalThis.uni is mocked', () => {
    const uni = (globalThis as unknown as { uni: UniMock }).uni;
    expect(typeof uni).toBe('object');
    expect(typeof uni.navigateTo).toBe('function');
  });

  it('uni.navigateTo accepts url param', async () => {
    const uni = (globalThis as unknown as { uni: UniMock }).uni;
    const result = await uni.navigateTo({ url: '/pages/home/index' });
    expect(result).toBeUndefined();
  });

  it('Role type union compiles', () => {
    const r: Role = 'patient';
    expect(r).toBe('patient');
  });

  it('MeResponse interface shape', () => {
    const me: MeResponse = {
      id: 1,
      phone: '13800138000',
      role: 'patient',
      active_role: 'patient',
      roles: ['patient', 'escort'],
      real_name_verified: false,
    };
    expect(me.roles).toContain('escort');
  });
});