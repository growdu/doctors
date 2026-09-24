/**
 * FinancePage：财务概览页骨架（v1 Task 23 占位）。
 *
 * 范围：
 *   - GMV / 退款额 / 净收入 / 提现汇总；
 *   - 每日曲线 + 渠道拆分。
 *
 * TODO: 实现页面（来自 plan v1 Task 23）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function FinancePage() {
  return (
    <div data-testid="finance-page">
      <PageHeader
        title="财务概览"
        subtitle="GMV / 退款 / 净收入 / 提现汇总 + 渠道拆分"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/finance/overview + @ant-design/charts 曲线
        </p>
      </Card>
    </div>
  );
}