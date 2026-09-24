/**
 * ReportsPage：业务报表页骨架（v1 Task 24 占位）。
 *
 * 范围：
 *   - 自定义时间区间报表；
 *   - 维度：渠道 / 医院 / 套餐 / 陪诊师；
 *   - 导出 Excel。
 *
 * TODO: 实现页面（来自 plan v1 Task 24）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function ReportsPage() {
  return (
    <div data-testid="reports-page">
      <PageHeader
        title="业务报表"
        subtitle="自定义时间区间 + 多维度交叉 + Excel 导出"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/reports + DatePicker + 维度筛选 + 导出
        </p>
      </Card>
    </div>
  );
}