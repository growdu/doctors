/**
 * HTTP client：fetch 封装 + Bearer token 自动注入 + 401 回调。
 *
 * v2（unified-app）：
 *   - token 从 auth store 读取（localStorage 持久化）
 *   - 401 时清 token + 触发 auth store.onUnauthorized 跳转 /pages/home/index
 *   - 业务错误码统一返回 { code, message, data, trace_id } 由调用方解析
 */
import { useAuthStore } from '@/store/auth';

export interface ApiResp<T> {
  code: number;
  message: string;
  data: T;
  trace_id?: string;
}

export interface ApiError extends Error {
  code: number;
  trace_id?: string;
}

/** 业务码常量（与 shared/errs 对齐；前端 mirror 便于类型推导） */
export const CodeOk = 0;
export const CodeUnauthorized = 11001;
export const CodeAdminForbidden = 11003;
export const CodeParamInvalid = 10001;

const BASE_URL = process.env.UNI_BASE_URL || 'http://127.0.0.1:8081';

/** fetch 封装：自动注入 Authorization + 解析业务错误 */
export async function request<T>(opts: {
  url: string;
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  data?: Record<string, unknown>;
  baseURL?: string;
}): Promise<T> {
  const auth = useAuthStore();
  const url = `${opts.baseURL || BASE_URL}${opts.url}`;
  const resp = await fetch(url, {
    method: opts.method || 'GET',
    headers: {
      'Content-Type': 'application/json',
      ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}),
    },
    body: opts.data ? JSON.stringify(opts.data) : undefined,
  });

  // 401：清 token + 跳登录
  if (resp.status === 401) {
    auth.onUnauthorized();
    throw makeError(CodeUnauthorized, 'unauthorized', undefined);
  }

  const json = (await resp.json()) as ApiResp<T>;
  if (json.code !== CodeOk) {
    throw makeError(json.code, json.message, json.trace_id);
  }
  return json.data;
}

function makeError(code: number, msg: string, trace_id?: string): ApiError {
  const e = new Error(msg) as ApiError;
  e.code = code;
  if (trace_id) e.trace_id = trace_id;
  return e;
}