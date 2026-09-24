/**
 * WorkOrdersPage：工单列表页骨架（v1 Task 15 占位）。
 *
 * 范围：
 *   - 客服工单列表（投诉 / 申诉 / 咨询）；
 *   - 状态筛选（待处理 / 处理中 / 已完成）；
 *   - 行操作：详情 / 回复 / 关闭。
 *
 * TODO: 实现页面（来自 plan v1 Task 15）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function WorkOrdersPage() {
  return (
    <div data-testid="work-orders-page">
      <PageHeader
        title="客服工单"
        subtitle="投诉 / 申诉 / 咨询工单 + 处理队列"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/work-orders + 状态/优先级筛选 + 操作区
        </p>
      </Card>
    </div>
  );
}