/**
 * EscortsPage：陪诊师列表页骨架（v1 Task 12 占位）。
 *
 * 范围：
 *   - 列出所有陪诊师（含审核状态筛选）；
 *   - 行操作：详情 / 审核。
 *
 * TODO: 实现页面（来自 plan v1 Task 12）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function EscortsPage() {
  return (
    <div data-testid="escorts-page">
      <PageHeader
        title="陪诊师管理"
        subtitle="陪诊师列表 · 审核队列 · 详情"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/escorts + ProTable 渲染 + 状态筛选 + 跳详情
        </p>
      </Card>
    </div>
  );
}