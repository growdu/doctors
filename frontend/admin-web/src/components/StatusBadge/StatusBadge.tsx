/**
 * StatusBadge：统一渲染订单 / 退款 / 申诉 13 个状态的彩色 Badge。
 *
 * v2 增量：新增 2 状态色：
 *   - selecting_escort          → 橙色（warning）= 等待患者选陪诊师
 *   - escort_pending_acceptance → 蓝色（processing）= 等待陪诊师 30s 确认
 *
 * 配色规范：
 *   - default    → 灰   （已创建 / 已退款 / 已关闭 / 已取消 / 已评价）
 *   - processing → 蓝   （进行中：已支付 / 待陪诊师确认 / 结算中 / 匹配中）
 *   - success    → 绿   （正向终态：已接单 / 服务中 / 已完成）
 *   - warning    → 橙   （需关注：待患者选人 / 退款中）
 *   - error      → 红   （需介入：申诉中）
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4
 */
import { Badge } from 'antd';
import type { OrderStatus } from '@/types/generated';

const STATUS_MAP: Record<
  OrderStatus,
  { text: string; color: 'default' | 'processing' | 'success' | 'warning' | 'error' }
> = {
  created:                       { text: '已创建',       color: 'default' },
  paid:                          { text: '已支付',       color: 'processing' },
  matching:                      { text: '匹配中',       color: 'processing' },
  // v2 新增 ↓
  selecting_escort:              { text: '待患者选人',   color: 'warning' },
  escort_pending_acceptance:     { text: '待陪诊师确认', color: 'processing' },
  // v2 新增 ↑
  accepted:                      { text: '已接单',       color: 'success' },
  in_service:                    { text: '服务中',       color: 'success' },
  completed:                     { text: '已完成',       color: 'success' },
  reviewed:                      { text: '已评价',       color: 'default' },
  refunding:                     { text: '退款中',       color: 'warning' },
  refunded:                      { text: '已退款',       color: 'default' },
  settling:                      { text: '结算中',       color: 'processing' },
  disputed:                      { text: '申诉中',       color: 'error' },
  closed:                        { text: '已关闭',       color: 'default' },
  canceled:                      { text: '已取消',       color: 'default' },
};

export interface StatusBadgeProps {
  status: string;
  /** 测试钩子：允许外部传入 data-testid */
  testId?: string;
}

/**
 * 渲染订单状态徽章。
 *
 * @example
 *   <StatusBadge status="selecting_escort" />
 *   <StatusBadge status="escort_pending_acceptance" testId="hdr-status" />
 */
export function StatusBadge({ status, testId }: StatusBadgeProps) {
  const meta = STATUS_MAP[status as OrderStatus] ?? { text: status, color: 'default' as const };
  return (
    <Badge
      status={meta.color}
      text={meta.text}
      data-testid={testId}
      data-status={status}
    />
  );
}

export default StatusBadge;