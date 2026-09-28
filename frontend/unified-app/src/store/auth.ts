/**
 * auth store（Pinia）：v2 多角色状态 + 角色切换。
 *
 * 状态：
 *   - token：当前 JWT
 *   - user：当前用户画像（id / phone / active_role / roles[]）
 *   - activeRole：前端 unified 按此决定显示哪个域路由
 *
 * 行为：
 *   - bootstrap()：onLaunch 时从 localStorage 恢复
 *   - login()：SMS 登录后写入 token + user
 *   - switchRole()：调 /auth/switch-role 拿新 token + 更新 active
 *   - onUnauthorized()：401 回调清状态
 *
 * 对应 spec：2026-09-28-unified-app-v2.md §2.4
 */
import { defineStore } from 'pinia';
import type { MeResponse, Role, SwitchRoleResponse } from '@/types/auth';
import { fetchMe, loginByPhone, sendSmsCode, switchActiveRole } from '@/api/auth';

const STORAGE_KEY = 'unified.auth';

interface PersistedAuth {
  token: string;
  user: MeResponse;
}

interface AuthState {
  token: string;
  user: MeResponse | null;
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ token: '', user: null }),

  getters: {
    isAuthed: (s) => !!s.token && !!s.user,
    activeRole: (s): Role | '' => (s.user?.active_role as Role) || '',
    roles: (s): Role[] => (s.user?.roles as Role[]) || [],
    hasRole(role: Role): boolean {
      return this.roles.includes(role);
    },
  },

  actions: {
    /** 启动时从 localStorage 恢复（uni-app h5 用 uni.setStorageSync；mock 用 localStorage） */
    bootstrap() {
      try {
        const raw = typeof uni !== 'undefined'
          ? uni.getStorageSync(STORAGE_KEY)
          : localStorage.getItem(STORAGE_KEY);
        if (typeof raw === 'string' && raw) {
          const data = JSON.parse(raw) as PersistedAuth;
          this.token = data.token;
          this.user = data.user;
        }
      } catch {
        // 静默：localStorage 损坏等同未登录
      }
    },

    /** SMS 登录：下发验证码 + 拿 token */
    async login(phone: string, code: string) {
      await sendSmsCode(phone);
      const r = await loginByPhone(phone, code);
      this.token = r.token;
      // 立即拉 /me 拿到 roles + active_role（首次登录时 active_role = role）
      await this.refreshMe();
      this.persist();
    },

    /** 调 /auth/me 刷新 user 状态 */
    async refreshMe() {
      this.user = await fetchMe();
    },

    /** 切换当前激活角色（v2 核心：unified-app HomeShell 调用入口）*/
    async switchRole(active: Role) {
      const r: SwitchRoleResponse = await switchActiveRole(active);
      this.token = r.token;
      await this.refreshMe();
      this.persist();
    },

    /** 401 回调：清状态 + 跳登录 */
    onUnauthorized() {
      this.token = '';
      this.user = null;
      try {
        if (typeof uni !== 'undefined') uni.removeStorageSync(STORAGE_KEY);
        else localStorage.removeItem(STORAGE_KEY);
      } catch {
        /* noop */
      }
      // 跳 home shell（unified-app 入口）
      if (typeof uni !== 'undefined') uni.reLaunch({ url: '/pages/home/index' });
    },

    /** 持久化到 localStorage / uni storage */
    persist() {
      const data: PersistedAuth = { token: this.token, user: this.user! };
      const raw = JSON.stringify(data);
      try {
        if (typeof uni !== 'undefined') uni.setStorageSync(STORAGE_KEY, raw);
        else localStorage.setItem(STORAGE_KEY, raw);
      } catch {
        /* noop */
      }
    },
  },
});