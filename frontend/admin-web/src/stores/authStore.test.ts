/**
 * authStore 测试契约（待 vitest 启用后跑）：
 *   - 6 个 ADMIN_ROLES 全覆盖；
 *   - login 写入 session + role/user.role 一致；
 *   - login 时 role 不一致抛 11003；
 *   - bootstrap 从 localStorage 恢复；
 *   - bootstrap 时 role 不一致抛 11003 并清状态；
 *   - logout 清空 token/user/role/isAuthed；
 *   - onUnauthorized 调 logout + 跳 /login。
 *
 * 注：当前骨架未装 vitest / @testing-library/react，本测试文件为
 * 「契约 + 验收脚本」式样，写好待用户在本地 `npm install` 后跑 vitest。
 */
import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore, ADMIN_ROLES, ERR_ADMIN_FORBIDDEN } from './authStore';

// 在 jsdom 环境清理 localStorage
beforeEach(() => {
  localStorage.clear();
});

describe('authStore.ADMIN_ROLES', () => {
  it('含 6 个角色常量', () => {
    expect(ADMIN_ROLES).toEqual([
      'super_admin',
      'order_admin',
      'refund_admin',
      'audit_admin',
      'cs',
      'viewer',
    ]);
  });
});

describe('authStore.login', () => {
  it('username=super 写入 super_admin 会话', async () => {
    const { token, user } = await useAuthStore.getState().login('super', 'pwd');
    expect(token).toMatch(/^mock\.\d+\.\d+$/);
    expect(user.role).toBe('super_admin');
    expect(useAuthStore.getState().isAuthed).toBe(true);
    expect(useAuthStore.getState().role).toBe('super_admin');
  });

  it('username=order 写入 order_admin', async () => {
    await useAuthStore.getState().login('order', 'pwd');
    expect(useAuthStore.getState().role).toBe('order_admin');
  });

  it('未知 username 兜底为 viewer', async () => {
    await useAuthStore.getState().login('random', 'pwd');
    expect(useAuthStore.getState().role).toBe('viewer');
  });
});

describe('authStore._setSession', () => {
  it('role 与 user.role 不一致抛 11003', () => {
    const set = useAuthStore.getState()._setSession;
    expect(() =>
      set({
        token: 't',
        user: { id: 1, username: 'x', display_name: 'x', role: 'cs' },
        role: 'super_admin', // 不一致
      }),
    ).toThrow(/11003/);
  });
});

describe('authStore.bootstrap', () => {
  it('空 localStorage → isAuthed=false', () => {
    useAuthStore.getState().bootstrap();
    expect(useAuthStore.getState().isAuthed).toBe(false);
  });

  it('持久化有效 session → isAuthed=true', async () => {
    await useAuthStore.getState().login('refund', 'pwd');
    // 重置 store state（模拟刷新页面）
    useAuthStore.setState({ token: null, user: null, role: null, isAuthed: false });
    useAuthStore.persist.rehydrate();
    useAuthStore.getState().bootstrap();
    expect(useAuthStore.getState().isAuthed).toBe(true);
    expect(useAuthStore.getState().role).toBe('refund_admin');
  });

  it('持久化 role 不在 6 角色 → 抛 11003', () => {
    useAuthStore.setState({
      token: 't',
      user: { id: 1, username: 'x', display_name: 'x', role: 'hacker' },
      role: 'hacker' as never,
      isAuthed: true,
    });
    expect(() => useAuthStore.getState().bootstrap()).toThrow(/11003/);
    expect(useAuthStore.getState().isAuthed).toBe(false);
  });

  it('持久化 user.role 与 role 不一致 → 抛 11003', () => {
    useAuthStore.setState({
      token: 't',
      user: { id: 1, username: 'x', display_name: 'x', role: 'cs' },
      role: 'super_admin',
      isAuthed: true,
    });
    expect(() => useAuthStore.getState().bootstrap()).toThrow(/11003/);
    expect(useAuthStore.getState().isAuthed).toBe(false);
  });
});

describe('authStore.logout', () => {
  it('清空 token/user/role/isAuthed', async () => {
    await useAuthStore.getState().login('audit', 'pwd');
    expect(useAuthStore.getState().isAuthed).toBe(true);
    useAuthStore.getState().logout();
    expect(useAuthStore.getState().token).toBeNull();
    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().role).toBeNull();
    expect(useAuthStore.getState().isAuthed).toBe(false);
  });
});

describe('authStore.onUnauthorized', () => {
  it('调 logout 清状态', async () => {
    await useAuthStore.getState().login('viewer', 'pwd');
    // 拦截 window.location.assign 避免 jsdom 跳转报错
    const origAssign = window.location.assign;
    let called = false;
    window.location.assign = () => {
      called = true;
    };
    useAuthStore.getState().onUnauthorized();
    expect(useAuthStore.getState().isAuthed).toBe(false);
    expect(called).toBe(true);
    window.location.assign = origAssign;
  });
});

describe('错误码常量', () => {
  it('ERR_ADMIN_FORBIDDEN=11003', () => {
    expect(ERR_ADMIN_FORBIDDEN).toBe(11003);
  });
});