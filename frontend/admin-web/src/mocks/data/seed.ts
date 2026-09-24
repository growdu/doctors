/**
 * MSW 种子数据（v2 增量适配）。
 *
 * v2 新增 fixture：
 *   - id=9001 status=selecting_escort         （未选陪诊师）
 *   - id=9002 status=escort_pending_acceptance（已选 #42，30s 截止）
 *   - id=9003 status=selecting_escort         （拒接回退，含 escort_reject_reason）
 *
 * 注：v1 的 mockOrders 基线数据由 setup 骨架（Task 1~3）外层补；
 * 本文件只声明 v2 增量 fixture 的 schema，避免覆盖 v1 数据。
 * 真实组装点在 `handlers/admin/orders.ts`，会用 spread 合并 v1+ v2。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4
 *           2026-09-24-admin-web-setup.md §Task 5
 */
import type { OrderDetail } from '@/types/generated';

/** v1 基线 fixture（占位 5 条覆盖主流状态机） */
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

/** v2 新增 fixture：覆盖 selecting_escort / escort_pending_acceptance + 拒接回退 */
const V2_FIXTURES: OrderDetail[] = [
  {
    id: 9001,
    status: 'selecting_escort',           // 待患者选人
    selected_escort_id: null,             // 尚未选人
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
    status: 'escort_pending_acceptance',  // 待陪诊师确认
    selected_escort_id: 42,               // 已选陪诊师
    // 30s 截止（mock 默认过期时间，详情页倒计时组件会基于 now 重新计算）
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
    status: 'selecting_escort',           // 已回退到 selecting_escort
    selected_escort_id: 42,
    escort_pending_expire_at: null,
    escort_reject_reason: 'escort_declined',  // 陪诊师主动拒接
    hospital_name: '301医院',
    final_amount: 600,
    created_at: '2026-09-24T10:02:00Z',
    patient_name: '王五', patient_phone: '138****0003',
    package_name: '半日陪诊',
  },
];

/** 暴露给 handler 的完整订单数组（v1 + v2 合并） */
export const mockOrders: OrderDetail[] = [...V1_BASELINE, ...V2_FIXTURES];

/** 看板聚合指标 mock（v2 含 2 新指标） */
export const mockOverview = {
  today_orders: 12,
  today_gmv: 8600,
  pending_escorts: 3,
  pending_refunds: 2,
  // v2 新增 ↓
  pending_selecting_escort: 5,
  pending_escort_acceptance: 2,
  // v2 新增 ↑
};