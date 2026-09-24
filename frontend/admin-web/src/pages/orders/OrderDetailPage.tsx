/**
 * OrderDetailPage：admin-web 订单详情页（v2 选人模式适配）。
 *
 * v2 增量（基于原 v1 详情页雏形）：
 *   - 状态机进度条加 2 状态分支（selecting_escort / escort_pending_acceptance）；
 *   - 展示 selected_escort_id（已选陪诊师）+ 30s 倒计时（仅 escort_pending_acceptance 时）；
 *   - escort_reject_reason 非空时顶部 Alert 卡「陪诊师拒接，订单已回退」+ 提示「可重新选」。
 *
 * 设计要点：
 *   - 用 useParams 取 :id，TanStack Query 5.x 拉详情；
 *   - 进度条用 AntD Steps 组件单链渲染（STEPS 6 节点）；
 *   - 倒计时独立 EscortPendingCountdown 组件（详情页外可复用）。
 *
 * 对应 spec：2026-09-24-order-matching-redesign.md §4 + §5
 *           2026-09-24-admin-web-setup.md §Task 2
 */
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Steps, Descriptions, Card as AntCard, Alert, Tag, Spin, Typography } from 'antd';
import dayjs from 'dayjs';
import { StatusBadge } from '@/components/StatusBadge';
import { EscortPendingCountdown } from './EscortPendingCountdown';
import { fetchOrderDetail, orderQueryKeys } from '@/api/admin/orders';
import type { OrderDetail } from '@/types/generated';

const { Title } = Typography;

// ── 进度条分支步骤定义 ──────────────────────────────────────────────
// 单链 6 节点：paid → selecting_escort → escort_pending_acceptance → accepted → in_service → completed
const STEPS = [
  'paid',
  'selecting_escort',
  'escort_pending_acceptance',
  'accepted',
  'in_service',
  'completed',
] as const;

const STEP_LABEL: Record<(typeof STEPS)[number] | 'selecting_escort', string> = {
  paid: '已支付',
  selecting_escort: '待患者选人',
  escort_pending_acceptance: '待陪诊师确认',
  accepted: '已接单',
  in_service: '服务中',
  completed: '已完成',
};

function currentStepIndex(status: string): number {
  // selecting_escort 时（含拒接回退）跳回 index 1
  if (status === 'selecting_escort') return 1;
  const idx = STEPS.indexOf(status as (typeof STEPS)[number]);
  return idx >= 0 ? idx : 0;
}

// 拒接原因文案映射
function rejectReasonText(reason: NonNullable<OrderDetail['escort_reject_reason']>): string {
  if (reason === 'escort_declined') return '陪诊师主动拒接';
  if (reason === 'lock_expired') return '陪诊师超时未确认';
  return String(reason);
}

export default function OrderDetailPage() {
  const { id = '' } = useParams<{ id: string }>();
  const numericId = Number(id);
  const { data, isLoading, isError, error } = useQuery({
    queryKey: orderQueryKeys.detail(numericId),
    queryFn: () => fetchOrderDetail(numericId),
    enabled: Number.isFinite(numericId) && numericId > 0,
  });

  if (isLoading) {
    return (
      <div data-testid="order-detail-loading">
        <Spin />
      </div>
    );
  }
  if (isError) {
    return (
      <Alert
        type="error"
        showIcon
        message="订单加载失败"
        description={(error as Error)?.message ?? '未知错误'}
      />
    );
  }
  if (!data) {
    return <Alert type="error" showIcon message="订单不存在" />;
  }

  const order = data;
  const stepIdx = currentStepIndex(order.status);
  const showCountdown =
    order.status === 'escort_pending_acceptance' &&
    typeof order.escort_pending_expire_at === 'string';

  return (
    <div data-testid="order-detail-page">
      <Title level={3}>
        订单 {order.id}
      </Title>

      {/* 状态机进度条：v2 加 2 状态分支 */}
      <div data-testid="order-progress" style={{ marginBottom: 16 }}>
        <Steps
          current={stepIdx}
          size="small"
          items={STEPS.map((s) => ({ title: STEP_LABEL[s] ?? s }))}
        />
      </div>

      {/* 拒接回退卡：escort_reject_reason 非空时展示 */}
      {order.escort_reject_reason && (
        <Alert
          style={{ marginBottom: 16 }}
          type="warning"
          showIcon
          message="陪诊师拒接，订单已回退"
          data-testid="reject-alert"
          description={
            <>
              <div>
                <strong>拒接原因：</strong>
                {rejectReasonText(order.escort_reject_reason)}
              </div>
              <div style={{ marginTop: 8 }}>
                患者可在 miniapp 端
                <Link to="/orders?status=selecting_escort"> 重新选择其他陪诊师</Link>
                。
              </div>
            </>
          }
        />
      )}

      {/* 基础信息 */}
      <AntCard title="基础信息">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="订单号">{order.id}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <StatusBadge status={order.status} testId="order-status-badge" />
          </Descriptions.Item>
          <Descriptions.Item label="医院">{order.hospital_name}</Descriptions.Item>
          <Descriptions.Item label="金额">¥{order.final_amount}</Descriptions.Item>
          <Descriptions.Item label="下单时间">
            {dayjs(order.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>

          {/* v2 新增字段：已选陪诊师 */}
          <Descriptions.Item label="已选陪诊师" data-testid="selected-escort-cell">
            {order.selected_escort_id ? (
              <Tag color="blue" data-testid="selected-escort-tag">
                #{order.selected_escort_id}
              </Tag>
            ) : (
              <Tag data-testid="selected-escort-empty">未选</Tag>
            )}
          </Descriptions.Item>

          {/* v2 新增字段：30s 倒计时（仅 escort_pending_acceptance 时展示） */}
          <Descriptions.Item label="确认截止" data-testid="expire-cell">
            {showCountdown && order.escort_pending_expire_at ? (
              <>
                <EscortPendingCountdown expireAt={order.escort_pending_expire_at} />
                <span style={{ marginLeft: 8, color: '#999' }}>
                  ({dayjs(order.escort_pending_expire_at).format('HH:mm:ss')})
                </span>
              </>
            ) : (
              '—'
            )}
          </Descriptions.Item>
        </Descriptions>
      </AntCard>

      {/* 患者信息（可选，仅 detail 才有） */}
      {order.patient_name && (
        <AntCard style={{ marginTop: 16 }} title="患者信息">
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="姓名">{order.patient_name}</Descriptions.Item>
            <Descriptions.Item label="电话">{order.patient_phone ?? '—'}</Descriptions.Item>
            <Descriptions.Item label="套餐" span={2}>
              {order.package_name ?? '—'}
            </Descriptions.Item>
          </Descriptions>
        </AntCard>
      )}
    </div>
  );
}