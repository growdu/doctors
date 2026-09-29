/**
 * escort-service API 客户端（v2 unified-app · :8089）。
 *
 * 端点（escort-only）：
 *   - POST   /escorts                          注册陪诊师
 *   - GET    /escorts/{id}                     陪诊师资料
 *   - POST   /escorts/{id}/availability        设置可接单时段（服务端协调）
 *   - PATCH  /escorts/me/location              更新当前位置
 *   - PATCH  /escorts/me/city                  更新常驻城市
 *   - /escorts/qualifications                 资质 CRUD
 *   - /escorts/trainings                      培训记录
 *   - /escorts/me/availabilities              个人可接单时段
 *
 * 对应：services/escort/internal/{handler,availability}/*.go
 */
import { request } from './client';

const ESCORT_BASE_URL =
  (typeof process !== 'undefined' && process.env?.UNI_ESCORT_BASE_URL) || 'http://127.0.0.1:8089';

// ── 类型定义 ───────────────────────────────────────────────────────

export interface Escort {
  id: number;
  user_id: number;
  nickname: string;
  avatar_url: string | null;
  city: string;
  /** 资质认证状态：none / pending / approved / rejected */
  audit_status: 'none' | 'pending' | 'approved' | 'rejected';
  /** 服务评分 */
  rating: number;
  /** 接单数 */
  total_orders: number;
}

export interface Qualification {
  id: number;
  escort_id: number;
  /** 资质类型：medical / nurse / care / other */
  type: string;
  title: string;
  /** 证明材料 URL */
  cert_url: string;
  /** 认证状态 */
  verified: boolean;
}

export interface Training {
  id: number;
  escort_id: number;
  title: string;
  /** 完成时间 */
  completed_at: string;
  /** 证书 URL */
  cert_url: string | null;
}

export interface Availability {
  id: number;
  escort_id: number;
  /** 可接单开始时间 */
  start_at: string;
  /** 可接单结束时间 */
  end_at: string;
  /** 时段备注 */
  remark: string;
}

export interface Location {
  city: string;
  lat: number;
  lng: number;
  address: string;
}

// ── 端点函数 ───────────────────────────────────────────────────────

/** 注册陪诊师。 */
export async function registerEscort(req: { city: string; nickname?: string }): Promise<Escort> {
  return request({
    url: '/api/v1/escorts',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

/** 陪诊师资料。 */
export async function getEscort(id: number): Promise<Escort> {
  return request({ url: `/api/v1/escorts/${id}`, baseURL: ESCORT_BASE_URL });
}

/** 设置可接单时段（覆盖式）。 */
export async function setAvailability(escortId: number, slots: Omit<Availability, 'id' | 'escort_id'>[]): Promise<{ items: Availability[] }> {
  return request({
    url: `/api/v1/escorts/${escortId}/availability`,
    method: 'POST',
    data: { slots } as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

/** 更新当前位置。 */
export async function updateLocation(location: Location): Promise<{ ok: true }> {
  return request({
    url: '/api/v1/escorts/me/location',
    method: 'PATCH',
    data: location as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

/** 更新常驻城市。 */
export async function updateCity(city: string): Promise<{ ok: true }> {
  return request({
    url: '/api/v1/escorts/me/city',
    method: 'PATCH',
    data: { city } as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

// ── 资质 ───────────────────────────────────────────────────────────

export async function createQualification(req: Omit<Qualification, 'id' | 'escort_id' | 'verified'>): Promise<Qualification> {
  return request({
    url: '/api/v1/escorts/qualifications',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

export async function listQualifications(): Promise<{ items: Qualification[] }> {
  return request({ url: '/api/v1/escorts/qualifications', baseURL: ESCORT_BASE_URL });
}

export async function updateQualification(id: number, req: Partial<Omit<Qualification, 'id' | 'escort_id' | 'verified'>>): Promise<Qualification> {
  return request({
    url: `/api/v1/escorts/qualifications/${id}`,
    method: 'PATCH',
    data: req as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

export async function deleteQualification(id: number): Promise<{ ok: true }> {
  return request({
    url: `/api/v1/escorts/qualifications/${id}`,
    method: 'DELETE',
    baseURL: ESCORT_BASE_URL,
  });
}

// ── 培训 ───────────────────────────────────────────────────────────

export async function createTraining(req: Omit<Training, 'id' | 'escort_id'>): Promise<Training> {
  return request({
    url: '/api/v1/escorts/trainings',
    method: 'POST',
    data: req as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

export async function listTrainings(): Promise<{ items: Training[] }> {
  return request({ url: '/api/v1/escorts/trainings', baseURL: ESCORT_BASE_URL });
}

// ── 个人可接单时段 ──────────────────────────────────────────────────

export async function addAvailability(slot: Omit<Availability, 'id' | 'escort_id'>): Promise<Availability> {
  return request({
    url: '/api/v1/escorts/me/availabilities',
    method: 'PUT',
    data: slot as unknown as Record<string, unknown>,
    baseURL: ESCORT_BASE_URL,
  });
}

export async function listMyAvailabilities(): Promise<{ items: Availability[] }> {
  return request({ url: '/api/v1/escorts/me/availabilities', baseURL: ESCORT_BASE_URL });
}

export async function removeAvailability(id: number): Promise<{ ok: true }> {
  return request({
    url: `/api/v1/escorts/me/availabilities/${id}`,
    method: 'DELETE',
    baseURL: ESCORT_BASE_URL,
  });
}