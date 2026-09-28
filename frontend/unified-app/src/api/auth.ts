/**
 * auth 域 API 客户端（v2 unified-app）。
 *
 * 端点：
 *   - POST /api/v1/auth/sms/send { phone }
 *   - POST /api/v1/auth/login { type:'sms', phone, code } → LoginResponse
 *   - POST /api/v1/auth/switch-role { active } → SwitchRoleResponse
 *   - GET  /api/v1/users/me → MeResponse
 */
import { request } from './client';
import type {
  LoginResponse,
  MeResponse,
  SwitchRoleResponse,
} from '@/types/auth';

const AUTH_BASE = '/api/v1/auth';

export async function sendSmsCode(phone: string): Promise<{ sent: boolean; ttl: number }> {
  return request({ url: `${AUTH_BASE}/sms/send`, method: 'POST', data: { phone } });
}

export async function loginByPhone(phone: string, code: string): Promise<LoginResponse> {
  return request({
    url: `${AUTH_BASE}/login`,
    method: 'POST',
    data: { type: 'sms', phone, code },
  });
}

export async function switchActiveRole(active: string): Promise<SwitchRoleResponse> {
  return request({
    url: `${AUTH_BASE}/switch-role`,
    method: 'POST',
    data: { active } as unknown as Record<string, unknown>,
  });
}

export async function fetchMe(): Promise<MeResponse> {
  // /api/v1/users/me 在 auth-service 内
  return request({ url: '/api/v1/users/me' });
}