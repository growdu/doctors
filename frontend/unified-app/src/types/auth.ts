/**
 * v2（unified-app）多角色类型定义。
 *
 * 后端契约（auth-service v2）：
 *   - POST /api/v1/auth/sms/send { phone } → 200
 *   - POST /api/v1/auth/login { type, phone, code } → LoginResponse（含 v1 token）
 *   - POST /api/v1/auth/switch-role { active } → SwitchRoleResponse（新 token + active）
 *   - GET  /api/v1/users/me → MeResponse（含 active_role + roles[]）
 *
 * 对应：docs/superpowers/plans/2026-09-28-unified-app-v2.md §2.2
 */

/** v2 单角色枚举（与后端 Role 字符串对齐）*/
export type Role =
  | 'patient'
  | 'escort'
  | 'super_admin'
  | 'order_admin'
  | 'refund_admin'
  | 'cs'
  | 'audit_admin'
  | 'viewer';

/** /me 返回的用户画像 */
export interface MeResponse {
  id: number;
  phone: string;
  role: string;             // v1 兼容：取 active_role
  active_role: string;
  roles: string[];
  real_name_verified: boolean;
}

/** LoginResponse：v1 单 token 兼容 */
export interface LoginResponse {
  token: string;
  user_id: number;
}

/** SwitchRoleResponse：v2 切角色 */
export interface SwitchRoleResponse {
  token: string;
  user_id: number;
  active: string;
  roles: string[];
}

/** 通用请求：active 切换 */
export interface SwitchRoleRequest {
  active: string;
}