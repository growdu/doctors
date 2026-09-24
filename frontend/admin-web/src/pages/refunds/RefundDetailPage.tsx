/**
 * RefundDetailPage：退款详情页骨架（v1 Task 13 占位）。
 *
 * 范围：
 *   - 退款工单基本信息；
 *   - 关联订单 + 患者；
 *   - 审批历史时间线；
 *   - 审批操作区（通过 / 驳回）。
 *
 * TODO: 实现页面（来自 plan v1 Task 13）
 */
import { Card } from 'antd';
import { useParams } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';

export default function RefundDetailPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <div data-testid="refund-detail-page">
      <PageHeader
        title={`退款详情 #${id ?? '—'}`}
        subtitle="工单信息 + 关联订单 + 审批操作"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/refunds/:id + Descriptions + AuditAction + 审批按钮组
        </p>
      </Card>
    </div>
  );
}