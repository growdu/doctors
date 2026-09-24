/**
 * RefundsPage：退款列表页骨架（v1 Task 13 占位）。
 *
 * 范围：
 *   - 待审 / 已批准 / 已驳回 三状态列表；
 *   - 行操作：审批通过 / 驳回。
 *
 * TODO: 实现页面（来自 plan v1 Task 13）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function RefundsPage() {
  return (
    <div data-testid="refunds-page">
      <PageHeader
        title="退款管理"
        subtitle="待审 / 已批准 / 已驳回 退款工单"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/refunds + 状态筛选 + 审批/驳回操作
        </p>
      </Card>
    </div>
  );
}