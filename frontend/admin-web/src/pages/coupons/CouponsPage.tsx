/**
 * CouponsPage：优惠券管理页骨架（v1 Task 25 占位）。
 *
 * 范围：
 *   - 优惠券模板列表（满减 / 折扣 / 限时）；
 *   - 发放批次 + 核销统计；
 *   - 创建 / 编辑模板。
 *
 * TODO: 实现页面（来自 plan v1 Task 25）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function CouponsPage() {
  return (
    <div data-testid="coupons-page">
      <PageHeader
        title="优惠券管理"
        subtitle="模板列表 + 发放批次 + 核销统计"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/coupons + 创建/编辑/停用
        </p>
      </Card>
    </div>
  );
}