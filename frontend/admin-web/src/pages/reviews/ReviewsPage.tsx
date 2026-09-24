/**
 * ReviewsPage：评价列表页骨架（v1 Task 16 占位）。
 *
 * 范围：
 *   - 全部评价列表（陪诊师 / 订单）；
 *   - 评分筛选（1~5 星）；
 *   - 行操作：详情 / 隐藏 / 申诉。
 *
 * TODO: 实现页面（来自 plan v1 Task 16）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function ReviewsPage() {
  return (
    <div data-testid="reviews-page">
      <PageHeader
        title="评价管理"
        subtitle="全部评价列表 + 评分筛选 + 申诉处理"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/reviews + 评分筛选 + 申诉入口
        </p>
      </Card>
    </div>
  );
}