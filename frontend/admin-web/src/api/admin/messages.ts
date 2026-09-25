/**
 * admin-messages API 客户端（14 P0 页依赖）。
 *
 * 端点（与 MSW messages handler 对齐）：
 *   - GET    /api/v1/admin/messages?category=&read=
 *   - GET    /api/v1/admin/messages/:id
 *   - POST   /api/v1/admin/messages                → 发送站内信
 *   - POST   /api/v1/admin/messages/broadcast      → 全员广播
 *
 * 对应 spec：2026-09-24-admin-web-design.md §Task 17
 */
import { useAuthStore } from '@/stores/authStore';

const BASE = '/api/v1/admin';

function authHeader(): HeadersInit {
  const token = useAuthStore.getState().token;
  return token ? { authorization: `Bearer ${token}` } : {};
}

export type MessageCategory = 'system' | 'announcement' | 'work_order';

export interface MessageItem {
  id: number;
  category: MessageCategory;
  title: string;
  read: boolean;
  created_at: string;
  /** 站内信专用字段（mock） */
  target_user_id?: number;
  /** 广播专用字段（mock） */
  audience?: 'all' | string;
}

export interface MessageListResponse {
  code: number;
  data: MessageItem[];
  total: number;
  trace_id?: string;
}

export interface MessageDetailResponse {
  code: number;
  data: MessageItem;
  trace_id?: string;
}

export async function fetchMessages(
  params?: { category?: string; read?: boolean },
): Promise<{ data: MessageItem[]; total: number }> {
  const qs = new URLSearchParams();
  if (params?.category) qs.set('category', params.category);
  if (params?.read !== undefined) qs.set('read', String(params.read));
  const url = `${BASE}/messages${qs.toString() ? `?${qs}` : ''}`;
  const resp = await fetch(url, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchMessages failed: ${resp.status}`);
  const json = (await resp.json()) as MessageListResponse;
  if (json.code !== 0) throw new Error(`fetchMessages: code=${json.code}`);
  return { data: json.data, total: json.total };
}

export async function fetchMessageDetail(id: number): Promise<MessageItem> {
  const resp = await fetch(`${BASE}/messages/${id}`, {
    credentials: 'include',
    headers: authHeader(),
  });
  if (!resp.ok) throw new Error(`fetchMessageDetail failed: ${resp.status}`);
  const json = (await resp.json()) as MessageDetailResponse;
  if (json.code !== 0) throw new Error(`fetchMessageDetail: code=${json.code}`);
  return json.data;
}

/** 发送站内信（指定用户） */
export async function sendMessage(body: {
  target_user_id: number;
  title: string;
  category: MessageCategory;
}): Promise<MessageItem> {
  const resp = await fetch(`${BASE}/messages`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as MessageDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`sendMessage failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`sendMessage: code=${json.code}`);
  return json.data;
}

/** 全员广播 */
export async function broadcastMessage(body: {
  title: string;
  category: MessageCategory;
}): Promise<MessageItem> {
  const resp = await fetch(`${BASE}/messages/broadcast`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  });
  const json = (await resp.json().catch(() => null)) as MessageDetailResponse | null;
  if (!resp.ok || !json) throw new Error(`broadcastMessage failed: ${resp.status}`);
  if (json.code !== 0) throw new Error(`broadcastMessage: code=${json.code}`);
  return json.data;
}

export const messageQueryKeys = {
  list: (params?: { category?: string; read?: boolean }) =>
    ['messages', params?.category ?? 'all', params?.read ?? 'any'] as const,
};