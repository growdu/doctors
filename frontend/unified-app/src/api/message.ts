/**
 * message-service API 客户端（v2 unified-app · :8084）。
 *
 * 端点：
 *   - POST /api/v1/messages           发送站内信
 *   - GET  /api/v1/messages           消息列表（按 type / read 过滤）
 *   - GET  /api/v1/messages/{id}      消息详情
 *   - POST /api/v1/messages/broadcast 系统广播（admin-only）
 *
 * 对应：services/message/internal/handler/handler.go
 */
import { request } from './client';

const MESSAGE_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_MESSAGE_BASE_URL) || 'http://127.0.0.1:8084';

// ── 类型定义 ───────────────────────────────────────────────────────

export type MessageType = 'system' | 'order' | 'payment' | 'sos' | 'review';

export interface Message {
  id: number;
  /** 收件人 user_id */
  to_user_id: number;
  type: MessageType;
  title: string;
  content: string;
  /** 关联资源（订单 / 退款 / SOS 等） */
  ref_id: number | null;
  ref_type: string | null;
  read: boolean;
  created_at: string;
}

export interface SendMessageRequest {
  to_user_id: number;
  type: MessageType;
  title: string;
  content: string;
  ref_id?: number;
  ref_type?: string;
}

export interface BroadcastRequest {
  /** 广播范围：all / patient / escort / admin */
  audience: 'all' | 'patient' | 'escort' | 'admin';
  title: string;
  content: string;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/** 发送站内信。 */
export async function sendMessage(req: SendMessageRequest): Promise<Message> {
  return request({
    url: '/api/v1/messages',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: MESSAGE_BASE_URL,
  });
}

/** 消息列表（按 type / read 过滤）。 */
export async function listMessages(query: { type?: MessageType; read?: boolean; page?: number; page_size?: number } = {}): Promise<{ items: Message[]; total: number; unread: number }> {
  const qs = new URLSearchParams();
  if (query.type) qs.set('type', query.type);
  if (query.read !== undefined) qs.set('read', String(query.read));
  if (query.page) qs.set('page', String(query.page));
  if (query.page_size) qs.set('page_size', String(query.page_size));
  const url = qs.toString() ? `/api/v1/messages?${qs}` : '/api/v1/messages';
  return request({ url, baseURL: MESSAGE_BASE_URL });
}

/** 消息详情。 */
export async function getMessage(id: number): Promise<Message> {
  return request({ url: `/api/v1/messages/${id}`, baseURL: MESSAGE_BASE_URL });
}

/** 系统广播（admin-only）。 */
export async function broadcast(req: BroadcastRequest): Promise<{ ok: true; sent_count: number }> {
  return request({
    url: '/api/v1/messages/broadcast',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: MESSAGE_BASE_URL,
  });
}