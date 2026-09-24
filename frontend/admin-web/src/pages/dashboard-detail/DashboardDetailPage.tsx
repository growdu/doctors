/**
 * DashboardDetailPage：数据看板明细页骨架（v1 Task 28 占位）。
 *
 * 范围：
 *   - 看板卡片点击下钻（订单详情列表 / 退款工单列表 / 陪诊师审核队列）；
 *   - 路由参数：type=orders_pending | refunds_pending | escorts_pending
 *
 * TODO: 实现页面（来自 plan v1 Task 28）
 */
import { Card } from 'antd';
import { useSearchParams } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';

const TYPE_LABEL: Record<string, string> = {
  orders_pending: '待处理订单',
  refunds_pending: '待审退款',
  escorts_pending: '待审核陪诊师',
};

export default function DashboardDetailPage() {
  const [searchParams] = useSearchParams();
  const type = searchParams.get('type') ?? '';
  const title = TYPE_LABEL[type] ?? '看板明细';
  return (
    <div data-testid="dashboard-detail-page">
      <PageHeader title={title} subtitle={`下钻类型：${type || '—'}`} />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入对应 MSW 列表接口 + 下钻筛选条件 + ProTable
        </p>
      </Card>
    </div>
  );
}