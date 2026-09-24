/**
 * authStore：admin-web 全局鉴权状态（Zustand 4.5 + persist 中间件）。
 *
 * 设计要点：
 *   - state 形如 `{ token, user, role, isAuthed }`；
 *   - 6 个角色常量（与后端 admin_user.role 对齐）：
 *       super_admin / order_admin / refund_admin / audit_admin / cs / viewer；
 *   - actions：
 *       · login(username, password)：mock 登录（实际生产应调 /api/v1/admin/login）；
 *       · bootstrap()：从 localStorage 恢复 + 校验 role 与 user.role 一致性；
 *       · logout()：清空状态 + 清 persist；
 *       · onUnauthorized()：401 触发时调用，跳 /login。
 *   - role 与 user.role 不一致 → `11003 admin_forbidden`。
 *   - localStorage key：`doctors-admin-auth`。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §3.3 (RBAC) + §5.1 (Auth)
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

// ── 角色常量 ────────────────────────────────────────────────────────
export const ADMIN_ROLES = [
  'super_admin',
  'order_admin',
  'refund_admin',
  'audit_admin',
  'cs',
  'viewer',
] as const;

export type AdminRole = (typeof ADMIN_ROLES)[number];

export function isAdminRole(value: unknown): value is AdminRole {
  return typeof value === 'string' && (ADMIN_ROLES as readonly string[]).includes(value);
}

// ── 用户模型 ────────────────────────────────────────────────────────
export interface AdminUser {
  id: number;
  username: string;
  display_name: string;
  /** 角色字符串，必须是 6 个 ADMIN_ROLES 之一；type 收紧为 AdminRole 但仍允许 unknown 兜底 */
  role: AdminRole | string;
  avatar_url?: string | null;
}

// ── 错误码常量 ──────────────────────────────────────────────────────
export const ERR_ADMIN_FORBIDDEN = 11003; // role 与 user.role 不一致
export const ERR_NOT_AUTHED = 11001; // 未登录

// ── Store 类型 ──────────────────────────────────────────────────────
export interface AuthState {
  token: string | null;
  user: AdminUser | null;
  role: AdminRole | null;
  isAuthed: boolean;

  /** 设置 token + user + role（已通过一致性校验）。 */
  _setSession: (input: { token: string; user: AdminUser; role: AdminRole }) => void;

  /** Mock 登录：根据 username 决定 role，写入 session。 */
  login: (username: string, password: string) => Promise<{ token: string; user: AdminUser }>;

  /** 从 localStorage 恢复 + 校验 role 与 user.role 一致性。 */
  bootstrap: () => void;

  /** 清空状态 + 清 persist（删除 localStorage key）。 */
  logout: () => void;

  /** 401 触发时调用：清状态 + 跳 /login（由调用方提供 navigateFn 注入以避免 store 耦合 router）。 */
  onUnauthorized: () => void;
}

// ── 工具函数 ────────────────────────────────────────────────────────
function buildDemoUser(username: string): AdminUser {
  // 简单 mock：username 前缀匹配 → role 映射，方便 dev 阶段切换不同角色
  const map: Record<string, AdminRole> = {
    super: 'super_admin',
    order: 'order_admin',
    refund: 'refund_admin',
    audit: 'audit_admin',
    cs: 'cs',
    viewer: 'viewer',
  };
  const role: AdminRole = map[username] ?? 'viewer';
  return {
    id: 1000 + Math.floor(Math.random() * 9000),
    username,
    display_name: username,
    role,
    avatar_url: null,
  };
}

// ── Store 实现 ──────────────────────────────────────────────────────
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      role: null,
      isAuthed: false,

      _setSession: ({ token, user, role }) => {
        // role 与 user.role 一致性硬校验：业务代码绝不允许绕过
        if (user.role !== role) {
          const err = new Error(
            `[authStore] role 与 user.role 不一致 (code=${ERR_ADMIN_FORBIDDEN})`,
          );
          (err as Error & { code?: number }).code = ERR_ADMIN_FORBIDDEN;
          throw err;
        }
        set({ token, user, role, isAuthed: true });
      },

      login: async (username: string, _password: string) => {
        // 真实实现：POST /api/v1/admin/login，返回 { token, user }。
        // 这里 mock：直接构造 user + fake token。
        const user = buildDemoUser(username);
        if (!isAdminRole(user.role)) {
          const err = new Error(`[authStore] unknown role: ${user.role}`);
          (err as Error & { code?: number }).code = ERR_ADMIN_FORBIDDEN;
          throw err;
        }
        const token = `mock.${user.id}.${Date.now()}`;
        get()._setSession({ token, user, role: user.role as AdminRole });
        return { token, user };
      },

      bootstrap: () => {
        const { token, user, role } = get();
        if (!token || !user || !role) {
          // 没有持久化 session → 视为未登录
          set({ token: null, user: null, role: null, isAuthed: false });
          return;
        }
        if (!isAdminRole(role) || user.role !== role) {
          // role 与 user.role 不一致 → 清状态并抛 11003
          set({ token: null, user: null, role: null, isAuthed: false });
          const err = new Error(`[authStore] bootstrap forbidden (code=${ERR_ADMIN_FORBIDDEN})`);
          (err as Error & { code?: number }).code = ERR_ADMIN_FORBIDDEN;
          throw err;
        }
        set({ isAuthed: true });
      },

      logout: () => {
        set({ token: null, user: null, role: null, isAuthed: false });
      },

      onUnauthorized: () => {
        get().logout();
        // 路由跳转由调用方（axios 拦截器 / 业务组件）通过注入的 navigateFn 完成；
        // 这里只清状态。简化方案：直接 window.location 兜底。
        if (typeof window !== 'undefined') {
          window.location.assign('/login');
        }
      },
    }),
    {
      name: 'doctors-admin-auth',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        role: state.role,
        isAuthed: state.isAuthed,
      }),
    },
  ),
);

// ── Selector 工具（推荐用法） ───────────────────────────────────────
export const selectIsAuthed = (s: AuthState) => s.isAuthed;
export const selectRole = (s: AuthState) => s.role;
export const selectUser = (s: AuthState) => s.user;

export default useAuthStore;