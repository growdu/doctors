/**
 * EscortAuditPage：陪诊师审核队列页骨架（v1 Task 12 占位）。
 *
 * 范围：
 *   - 待审核陪诊师列表（status=pending）；
 *   - 行操作：通过 / 拒绝 + 备注。
 *
 * TODO: 实现页面（来自 plan v1 Task 12）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function EscortAuditPage() {
  return (
    <div data-testid="escort-audit-page">
      <PageHeader
        title="陪诊师审核"
        subtitle="待审核陪诊师队列（status=pending）"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/escorts?status=pending + 通过/拒绝操作
        </p>
      </Card>
    </div>
  );
}