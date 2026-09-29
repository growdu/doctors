/**
 * admin-auth API 客户端（14 P0 页依赖）。
 *
 * 端点：
 *   - POST /api/v1/admin/login
 *
 * 设计：
 *   - 直接对接 useAuthStore（不再额外封装 login 函数，避免重复）；
 *   - 这里只导出后端契约类型 + 调用方法，便于未来接入真实后端时直接替换。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §5.1
 */
import type { AdminUser } from '@/stores/authStore';
import { isAdminRole } from '@/stores/authStore';

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponseBody {
  code: number;
  data: {
    token: string;
    user: AdminUser;
  };
  trace_id?: string;
}

/**
 * v2（unified-app）多角色切换响应。
 * POST /api/v1/auth/switch-role 返回：data 含新 token + 当前 active + 用户全部 roles。
 */
export interface SwitchRoleResponseBody {
  code: number;
  data: {
    token: string;
    user_id: number;
    active: string;
    /** 用户所有角色（兼职场景如 ["patient", "escort"]）*/
    roles: string[];
  };
  trace_id?: string;
}

/**
 * v2 /me 响应额外返回 active_role + roles（v1 仅 role 单值）。
 */
export interface MeResponseBody {
  code: number;
  data: {
    id: number;
    phone: string;
    role: string;
    /** v2 当前激活角色（前端按此切域 UI）*/
    active_role: string;
    /** v2 用户所有角色*/
    roles: string[];
    real_name_verified: boolean;
  };
  trace_id?: string;
}

/**
 * 调用后端登录接口；返回 token + user。
 *
 * 当前为 mock：直接在本地构造 user + fake token。
 * 真实接入时改为 fetch(/api/v1/admin/login) + 校验 code === 0。
 */
export async function loginRequest(
  _req: LoginRequest,
): Promise<{ token: string; user: AdminUser }> {
  // mock：复用 useAuthStore 内部约定（username 前缀决定 role）
  const map: Record<string, string> = {
    super: 'super_admin',
    order: 'order_admin',
    refund: 'refund_admin',
    audit: 'audit_admin',
    cs: 'cs',
    viewer: 'viewer',
  };
  const role = map[_req.username] ?? 'viewer';
  if (!isAdminRole(role)) {
    const err = new Error(`unknown role: ${role}`);
    (err as Error & { code?: number }).code = 11003;
    throw err;
  }
  const user: AdminUser = {
    id: 1000 + Math.floor(Math.random() * 9000),
    username: _req.username,
    display_name: _req.username,
    role,
    avatar_url: null,
  };
  const token = `mock.${user.id}.${Date.now()}`;
  return { token, user };
}