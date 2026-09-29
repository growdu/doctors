/**
 * AuditPage：审计日志页骨架（v1 Task 22 占位）。
 *
 * 范围：
 *   - 全局操作日志（按操作者 / 模块 / 时间筛选）；
 *   - 导出 CSV。
 *
 * TODO: 实现页面（来自 plan v1 Task 22）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';
import { AuditAction } from '@/components/AuditAction';

export default function AuditPage() {
  return (
    <div data-testid="audit-page">
      <PageHeader
        title="审计日志"
        subtitle="全局操作日志（按操作者 / 模块 / 时间筛选）"
      />
      <Card>
        <p style={{ color: '#999' }}>TODO: 接入 MSW GET /api/v1/admin/audit-logs + 筛选 + 导出</p>
        <AuditAction />
      </Card>
    </div>
  );
}