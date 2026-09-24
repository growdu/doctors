/**
 * MSW 种子数据（v1 + v2 合并）。
 *
 * v2 增量：
 *   - orders 5+3=8 条（详见 V1_BASELINE / V2_FIXTURES）；
 *   - overview 看板指标。
 *
 * v1 增量（本批次 Task A10）：
 *   - 9 类目 fixture（users / escorts / refunds / wallets / patients / work_orders
 *     / reviews / messages / sos / hospitals / packages / coupons / audit_logs），
 *     共 30+ 条数据，覆盖每个模块的列表 + 详情 fixture。
 *
 * 注：业务类型由本文件就地定义；generated.ts 故意不写（spec 仅 v2 增量落 generated）。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4
 *           2026-09-24-admin-web-setup.md §Task 5
 */
import type { OrderDetail } from '@/types/generated';

// ── v2 订单 fixture（保留原状） ──────────────────────────────────────
const V1_BASELINE: OrderDetail[] = [
  {
    id: 1001, status: 'created',
    hospital_name: '协和医院', final_amount: 300, created_at: '2026-09-20T09:00:00Z',
    selected_escort_id: null, escort_pending_expire_at: null, escort_reject_reason: null,
    patient_name: '甲', patient_phone: '138****0000', package_name: '半日陪诊',
  },
  {
    id: 1002, status: 'paid',
    hospital_name: '同济医院', final_amount: 500, created_at: '2026-09-21T10:00:00Z',
    selected_escort_id: null, escort_pending_expire_at: null, escort_reject_reason: null,
    patient_name: '乙', patient_phone: '138****0010', package_name: '全日陪诊',
  },
  {
    id: 1003, status: 'accepted',
    hospital_name: '301医院', final_amount: 800, created_at: '2026-09-22T11:00:00Z',
    selected_escort_id: 42, escort_pending_expire_at: null, escort_reject_reason: null,
    patient_name: '丙', patient_phone: '138****0020', package_name: '半日陪诊',
  },
  {
    id: 1004, status: 'in_service',
    hospital_name: '同仁医院', final_amount: 1000, created_at: '2026-09-23T12:00:00Z',
    selected_escort_id: 42, escort_pending_expire_at: null, escort_reject_reason: null,
    patient_name: '丁', patient_phone: '138****0030', package_name: '全日陪诊',
  },
  {
    id: 1005, status: 'completed',
    hospital_name: '安贞医院', final_amount: 600, created_at: '2026-09-19T13:00:00Z',
    selected_escort_id: 42, escort_pending_expire_at: null, escort_reject_reason: null,
    patient_name: '戊', patient_phone: '138****0040', package_name: '半日陪诊',
  },
];

const V2_FIXTURES: OrderDetail[] = [
  {
    id: 9001,
    status: 'selecting_escort',
    selected_escort_id: null,
    escort_pending_expire_at: null,
    escort_reject_reason: null,
    hospital_name: '协和医院',
    final_amount: 500,
    created_at: '2026-09-24T10:00:00Z',
    patient_name: '张三', patient_phone: '138****0001',
    package_name: '半日陪诊',
  },
  {
    id: 9002,
    status: 'escort_pending_acceptance',
    selected_escort_id: 42,
    escort_pending_expire_at: '2026-09-24T10:30:30Z',
    escort_reject_reason: null,
    hospital_name: '同仁医院',
    final_amount: 800,
    created_at: '2026-09-24T10:01:00Z',
    patient_name: '李四', patient_phone: '138****0002',
    package_name: '全日陪诊',
  },
  {
    id: 9003,
    status: 'selecting_escort',
    selected_escort_id: 42,
    escort_pending_expire_at: null,
    escort_reject_reason: 'escort_declined',
    hospital_name: '301医院',
    final_amount: 600,
    created_at: '2026-09-24T10:02:00Z',
    patient_name: '王五', patient_phone: '138****0003',
    package_name: '半日陪诊',
  },
];

export const mockOrders: OrderDetail[] = [...V1_BASELINE, ...V2_FIXTURES];

export const mockOverview = {
  today_orders: 12,
  today_gmv: 8600,
  pending_escorts: 3,
  pending_refunds: 2,
  pending_selecting_escort: 5,
  pending_escort_acceptance: 2,
};

// ── v1 增量：users（6 条，覆盖 6 角色） ─────────────────────────────
export interface AdminUserFixture {
  id: number;
  username: string;
  display_name: string;
  role: string;
  status: 'active' | 'disabled';
  last_login_at: string;
  avatar_url: string | null;
}

export const mockUsers: AdminUserFixture[] = [
  { id: 1, username: 'super', display_name: '超级管理员', role: 'super_admin', status: 'active', last_login_at: '2026-09-24T08:00:00Z', avatar_url: null },
  { id: 2, username: 'order', display_name: '订单管理员', role: 'order_admin', status: 'active', last_login_at: '2026-09-23T18:00:00Z', avatar_url: null },
  { id: 3, username: 'refund', display_name: '退款管理员', role: 'refund_admin', status: 'active', last_login_at: '2026-09-24T09:00:00Z', avatar_url: null },
  { id: 4, username: 'audit', display_name: '审核管理员', role: 'audit_admin', status: 'active', last_login_at: '2026-09-22T15:00:00Z', avatar_url: null },
  { id: 5, username: 'cs', display_name: '客服小张', role: 'cs', status: 'active', last_login_at: '2026-09-24T10:00:00Z', avatar_url: null },
  { id: 6, username: 'viewer', display_name: '只读观察', role: 'viewer', status: 'active', last_login_at: '2026-09-21T11:00:00Z', avatar_url: null },
];

// ── v1 增量：escorts（4 条，覆盖 pending/approved/rejected） ────────
export interface EscortFixture {
  id: number;
  name: string;
  phone: string;
  city: string;
  rating: number;
  audit_status: 'pending' | 'approved' | 'rejected';
  audit_note: string | null;
  created_at: string;
}

export const mockEscorts: EscortFixture[] = [
  { id: 1001, name: '王陪诊', phone: '138****1001', city: '北京', rating: 0, audit_status: 'pending',  audit_note: null,           created_at: '2026-09-23T08:00:00Z' },
  { id: 1002, name: '李陪诊', phone: '138****1002', city: '上海', rating: 0, audit_status: 'pending',  audit_note: null,           created_at: '2026-09-23T09:00:00Z' },
  { id: 1003, name: '赵陪诊', phone: '138****1003', city: '北京', rating: 4.9, audit_status: 'approved', audit_note: '资料完整',     created_at: '2026-09-20T10:00:00Z' },
  { id: 1004, name: '孙陪诊', phone: '138****1004', city: '广州', rating: 4.2, audit_status: 'rejected', audit_note: '资质过期',     created_at: '2026-09-19T11:00:00Z' },
];

// ── v1 增量：refunds（4 条） ────────────────────────────────────────
export interface RefundFixture {
  id: number;
  order_id: number;
  patient_name: string;
  amount: number;
  reason: string;
  status: 'pending' | 'approved' | 'rejected';
  refund_note: string | null;
  created_at: string;
}

export const mockRefunds: RefundFixture[] = [
  { id: 2001, order_id: 1001, patient_name: '甲', amount: 300, reason: '患者取消',         status: 'pending',  refund_note: null,           created_at: '2026-09-24T09:00:00Z' },
  { id: 2002, order_id: 1002, patient_name: '乙', amount: 500, reason: '医生停诊',         status: 'pending',  refund_note: null,           created_at: '2026-09-24T09:30:00Z' },
  { id: 2003, order_id: 1003, patient_name: '丙', amount: 800, reason: '服务不达标',       status: 'approved', refund_note: '已全额退款',     created_at: '2026-09-22T11:00:00Z' },
  { id: 2004, order_id: 1004, patient_name: '丁', amount: 1000, reason: '重复申请',        status: 'rejected', refund_note: '已在其他渠道退', created_at: '2026-09-21T12:00:00Z' },
];

// ── v1 增量：wallets（2 主体 + 8 流水） ─────────────────────────────
export interface WalletSubjectFixture {
  id: number;
  subject_type: 'patient' | 'escort';
  subject_name: string;
  balance: number;
  frozen: number;
}

export interface WalletTxFixture {
  id: number;
  subject_id: number;
  subject_type: 'patient' | 'escort';
  tx_type: 'recharge' | 'payment' | 'refund' | 'withdraw';
  amount: number;
  balance_after: number;
  created_at: string;
}

export const mockWalletSubjects: WalletSubjectFixture[] = [
  { id: 5001, subject_type: 'patient', subject_name: '张三', balance:  500, frozen:   0 },
  { id: 5002, subject_type: 'patient', subject_name: '李四', balance: 1200, frozen: 200 },
  { id: 5003, subject_type: 'escort',  subject_name: '王陪诊', balance: 3500, frozen:   0 },
  { id: 5004, subject_type: 'escort',  subject_name: '赵陪诊', balance: 8200, frozen: 500 },
];

export const mockWalletTransactions: WalletTxFixture[] = [
  { id: 6001, subject_id: 5001, subject_type: 'patient', tx_type: 'recharge', amount:  500, balance_after:  500, created_at: '2026-09-20T10:00:00Z' },
  { id: 6002, subject_id: 5001, subject_type: 'patient', tx_type: 'payment',  amount: -500, balance_after:    0, created_at: '2026-09-24T09:00:00Z' },
  { id: 6003, subject_id: 5002, subject_type: 'patient', tx_type: 'recharge', amount: 1500, balance_after: 1500, created_at: '2026-09-22T11:00:00Z' },
  { id: 6004, subject_id: 5002, subject_type: 'patient', tx_type: 'payment',  amount: -300, balance_after: 1200, created_at: '2026-09-23T12:00:00Z' },
  { id: 6005, subject_id: 5003, subject_type: 'escort',  tx_type: 'payment',  amount:  800, balance_after:  800, created_at: '2026-09-22T13:00:00Z' },
  { id: 6006, subject_id: 5003, subject_type: 'escort',  tx_type: 'payment',  amount:  700, balance_after: 1500, created_at: '2026-09-23T14:00:00Z' },
  { id: 6007, subject_id: 5004, subject_type: 'escort',  tx_type: 'withdraw', amount: -500, balance_after: 8200, created_at: '2026-09-24T08:00:00Z' },
  { id: 6008, subject_id: 5002, subject_type: 'patient', tx_type: 'refund',   amount:  300, balance_after: 1500, created_at: '2026-09-22T11:30:00Z' },
];

// ── v1 增量：patients（3 条） ───────────────────────────────────────
export interface PatientFixture {
  id: number;
  name: string;
  phone: string;
  registered_at: string;
  order_count: number;
  refund_count: number;
}

export const mockPatients: PatientFixture[] = [
  { id: 7001, name: '张三', phone: '138****0001', registered_at: '2026-01-15T08:00:00Z', order_count: 5, refund_count: 0 },
  { id: 7002, name: '李四', phone: '138****0002', registered_at: '2026-03-20T09:00:00Z', order_count: 3, refund_count: 1 },
  { id: 7003, name: '王五', phone: '138****0003', registered_at: '2026-05-10T10:00:00Z', order_count: 8, refund_count: 2 },
];

// ── v1 增量：work_orders（3 条） ────────────────────────────────────
export interface WorkOrderFixture {
  id: number;
  category: 'complaint' | 'appeal' | 'inquiry';
  subject: string;
  priority: 'low' | 'medium' | 'high';
  status: 'open' | 'in_progress' | 'closed';
  created_at: string;
}

export const mockWorkOrders: WorkOrderFixture[] = [
  { id: 8001, category: 'complaint', subject: '陪诊师迟到',     priority: 'high',   status: 'open',        created_at: '2026-09-24T08:00:00Z' },
  { id: 8002, category: 'appeal',    subject: '退款被驳回申诉', priority: 'medium', status: 'in_progress', created_at: '2026-09-23T15:00:00Z' },
  { id: 8003, category: 'inquiry',   subject: '套餐咨询',       priority: 'low',    status: 'closed',      created_at: '2026-09-22T11:00:00Z' },
];

// ── v1 增量：reviews（3 条） ────────────────────────────────────────
export interface ReviewFixture {
  id: number;
  order_id: number;
  escort_id: number;
  rating: number; // 1~5
  content: string;
  created_at: string;
}

export const mockReviews: ReviewFixture[] = [
  { id: 9001, order_id: 1003, escort_id: 1003, rating: 5, content: '服务专业',     created_at: '2026-09-22T13:00:00Z' },
  { id: 9002, order_id: 1004, escort_id: 1003, rating: 4, content: '不错',         created_at: '2026-09-23T14:00:00Z' },
  { id: 9003, order_id: 1005, escort_id: 1004, rating: 3, content: '还有改进空间', created_at: '2026-09-19T15:00:00Z' },
];

// ── v1 增量：messages（3 条） ───────────────────────────────────────
export interface MessageFixture {
  id: number;
  category: 'system' | 'announcement' | 'work_order';
  title: string;
  read: boolean;
  created_at: string;
}

export const mockMessages: MessageFixture[] = [
  { id: 11001, category: 'system',       title: '系统维护通知（9/25 02:00-04:00）', read: false, created_at: '2026-09-24T08:00:00Z' },
  { id: 11002, category: 'announcement', title: '新版退款流程上线',                read: false, created_at: '2026-09-23T10:00:00Z' },
  { id: 11003, category: 'work_order',   title: '工单 #8001 已分配给您',            read: true,  created_at: '2026-09-22T16:00:00Z' },
];

// ── v1 增量：sos（2 条） ────────────────────────────────────────────
export interface SosFixture {
  id: number;
  order_id: number;
  patient_name: string;
  escort_name: string;
  location: string;
  contact: string;
  status: 'open' | 'closed';
  created_at: string;
}

export const mockSosAlerts: SosFixture[] = [
  { id: 12001, order_id: 1004, patient_name: '丁', escort_name: '赵陪诊', location: '北京协和医院', contact: '138****0030', status: 'open',   created_at: '2026-09-24T10:30:00Z' },
  { id: 12002, order_id: 1005, patient_name: '戊', escort_name: '王陪诊', location: '上海同济医院', contact: '138****0040', status: 'closed', created_at: '2026-09-19T13:00:00Z' },
];

// ── v1 增量：hospitals（3 条） ──────────────────────────────────────
export interface HospitalFixture {
  id: number;
  name: string;
  city: string;
  level: '三甲' | '三乙' | '二甲';
  status: 'active' | 'inactive';
}

export const mockHospitals: HospitalFixture[] = [
  { id: 13001, name: '北京协和医院', city: '北京', level: '三甲', status: 'active'   },
  { id: 13002, name: '上海同济医院', city: '上海', level: '三甲', status: 'active'   },
  { id: 13003, name: '广州安贞医院', city: '广州', level: '三乙', status: 'inactive' },
];

// ── v1 增量：packages（3 条） ──────────────────────────────────────
export interface PackageFixture {
  id: number;
  name: string;
  price: number;
  duration: 'half_day' | 'full_day';
  status: 'on' | 'off';
}

export const mockPackages: PackageFixture[] = [
  { id: 14001, name: '半日陪诊', price: 300, duration: 'half_day', status: 'on'  },
  { id: 14002, name: '全日陪诊', price: 800, duration: 'full_day', status: 'on'  },
  { id: 14003, name: '专项陪诊', price: 1500, duration: 'full_day', status: 'off' },
];

// ── v1 增量：coupons（2 条） ───────────────────────────────────────
export interface CouponFixture {
  id: number;
  name: string;
  type: 'amount_off' | 'discount';
  value: number;
  valid_until: string;
}

export const mockCoupons: CouponFixture[] = [
  { id: 15001, name: '新人 50 元券', type: 'amount_off', value: 50,   valid_until: '2026-12-31T23:59:59Z' },
  { id: 15002, name: '8 折优惠券',   type: 'discount',   value: 0.8, valid_until: '2026-12-31T23:59:59Z' },
];

// ── v1 增量：audit_logs（3 条） ─────────────────────────────────────
export interface AuditLogFixture {
  id: number;
  actor: string;
  module: string;
  action: string;
  created_at: string;
}

export const mockAuditLogs: AuditLogFixture[] = [
  { id: 16001, actor: 'super', module: 'escorts', action: '通过陪诊师审核 #1001', created_at: '2026-09-24T09:00:00Z' },
  { id: 16002, actor: 'order', module: 'orders',  action: '强制取消订单 #1001',   created_at: '2026-09-24T10:00:00Z' },
  { id: 16003, actor: 'refund', module: 'refunds', action: '审批通过退款 #2001',   created_at: '2026-09-24T10:30:00Z' },
];