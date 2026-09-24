// src/stores/auth.test.js
//
// auth store 单测 —— token / user 状态机 + login/logout/onUnauthorized + handler 注册
//
// 测试策略：
//   - jest.doMock + jest.resetModules 隔离每个 case 的 api mock
//   - 动态 import('@/stores/auth.js') 拿到 useAuthStore（配合 doMock 强制重读模块）
//   - 全局 uni 由 jest.setup.ts 注入（installUniMock）；此处只关心 storage 行为

import { jest } from '@jest/globals';
import { createPinia, setActivePinia } from 'pinia';

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    // 清 storage（jest.setup.ts 已注入 globalThis.uni）
    try {
      globalThis.uni.removeStorageSync('patient.token');
      delete globalThis.__patient_token;
    } catch (_e) {
      // ignore
    }
    jest.resetModules();
  });

  it('初始：token / user / isLoggedIn 都为空', async () => {
    const { useAuthStore } = await import('@/stores/auth.js');
    const s = useAuthStore();
    expect(s.token).toBe('');
    expect(s.user).toBeNull();
    expect(s.isLoggedIn).toBe(false);
  });

  it('loginByPhone 成功后 token + user 被设置，持久化到 storage', async () => {
    const loginByPhoneMock = jest.fn(async () => ({
      access_token: 't-abc-123',
      user: { id: 1, phone: '13800138000' },
    }));
    jest.doMock('@/api/auth.js', () => ({ loginByPhone: loginByPhoneMock }));

    const { useAuthStore } = await import('@/stores/auth.js');
    const s = useAuthStore();
    const resp = await s.loginByPhone('13800138000', '1234');

    expect(loginByPhoneMock).toHaveBeenCalledWith({ phone: '13800138000', code: '1234' });
    expect(s.token).toBe('t-abc-123');
    expect(s.user).toEqual({ id: 1, phone: '13800138000' });
    expect(s.isLoggedIn).toBe(true);
    expect(resp.access_token).toBe('t-abc-123');
    // 持久化校验（utils/auth.js fallback 到 globalThis.__patient_token）
    expect(globalThis.__patient_token).toBe('t-abc-123');
  });

  it('loginByPhone 兼容 token / profile 字段名', async () => {
    const loginByPhoneMock = jest.fn(async () => ({
      token: 't-other',
      profile: { id: 2, name: '张三' },
    }));
    jest.doMock('@/api/auth.js', () => ({ loginByPhone: loginByPhoneMock }));

    const { useAuthStore } = await import('@/stores/auth.js');
    const s = useAuthStore();
    await s.loginByPhone('13900139000', '9999');
    expect(s.token).toBe('t-other');
    expect(s.user).toEqual({ id: 2, name: '张三' });
  });

  it('loginByPhone 响应缺 token → 抛错', async () => {
    const loginByPhoneMock = jest.fn(async () => ({ user: { id: 1 } }));
    jest.doMock('@/api/auth.js', () => ({ loginByPhone: loginByPhoneMock }));

    const { useAuthStore } = await import('@/stores/auth.js');
    const s = useAuthStore();
    await expect(s.loginByPhone('13800138000', '1234')).rejects.toThrow(
      /missing access_token/,
    );
    expect(s.token).toBe('');
    expect(s.isLoggedIn).toBe(false);
  });

  it('logout 清状态 + 触发已注册 onUnauthorized handler + 不再 reLaunch（jest 无 uni）', async () => {
    const { useAuthStore, registerUnauthorizedHandler } = await import('@/stores/auth.js');
    const s = useAuthStore();
    // 模拟已登录
    s.token = 't-xyz';
    s.user = { id: 1 };
    s.isLoggedIn = true;

    const handlerMock = jest.fn();
    const unregister = registerUnauthorizedHandler(handlerMock);

    await s.logout();

    expect(s.token).toBe('');
    expect(s.user).toBeNull();
    expect(s.isLoggedIn).toBe(false);
    expect(handlerMock).toHaveBeenCalledTimes(1);
    expect(handlerMock).toHaveBeenCalledWith({ reason: 'logout' });

    unregister();
  });

  it('logout 在未登录态不触发 handler', async () => {
    const { useAuthStore, registerUnauthorizedHandler } = await import('@/stores/auth.js');
    const s = useAuthStore();
    const handlerMock = jest.fn();
    const unregister = registerUnauthorizedHandler(handlerMock);

    await s.logout();
    expect(handlerMock).not.toHaveBeenCalled();

    unregister();
  });

  it('onUnauthorized 清状态 + 通知 handler，handler 异常不影响后续', async () => {
    const { useAuthStore, registerUnauthorizedHandler } = await import('@/stores/auth.js');
    const s = useAuthStore();
    s.token = 't-old';
    s.user = { id: 1 };
    s.isLoggedIn = true;

    const calls = [];
    const unregister1 = registerUnauthorizedHandler(() => { calls.push('h1'); });
    const unregister2 = registerUnauthorizedHandler(() => { calls.push('h2-throw'); throw new Error('boom'); });
    const unregister3 = registerUnauthorizedHandler(() => { calls.push('h3'); });

    s.onUnauthorized({ code: 11001, statusCode: 401 });

    expect(s.token).toBe('');
    expect(s.user).toBeNull();
    expect(s.isLoggedIn).toBe(false);
    expect(calls).toEqual(['h1', 'h2-throw', 'h3']);

    unregister1(); unregister2(); unregister3();
  });

  it('registerUnauthorizedHandler 注册空函数 → 返回 unregister 仍可调用', async () => {
    const { registerUnauthorizedHandler } = await import('@/stores/auth.js');
    const unregister = registerUnauthorizedHandler(null);
    expect(typeof unregister).toBe('function');
    // 多次调用 unregister 不报错
    unregister();
    unregister();
  });

  it('onUnauthorized 在已为空的态不触发 handler（避免重复通知）', async () => {
    const { useAuthStore, registerUnauthorizedHandler } = await import('@/stores/auth.js');
    const s = useAuthStore();
    const handlerMock = jest.fn();
    const unregister = registerUnauthorizedHandler(handlerMock);

    s.onUnauthorized({ code: 11001 });
    expect(handlerMock).not.toHaveBeenCalled();

    unregister();
  });
});