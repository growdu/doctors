/**
 * EscortDetailPage：陪诊师详情页骨架（v1 Task 12 占位）。
 *
 * 范围：
 *   - 基本信息（姓名 / 手机 / 资质 / 评分）；
 *   - 审核历史时间线；
 *   - 接单记录列表。
 *
 * TODO: 实现页面（来自 plan v1 Task 12）
 */
import { Card } from 'antd';
import { useParams } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';

export default function EscortDetailPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <div data-testid="escort-detail-page">
      <PageHeader
        title={`陪诊师详情 #${id ?? '—'}`}
        subtitle="基本信息 + 资质 + 审核历史 + 接单记录"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/escorts/:id + Descriptions + AuditAction
        </p>
      </Card>
    </div>
  );
}