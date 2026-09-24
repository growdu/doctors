/**
 * WalletsPage：钱包流水列表页骨架（v1 Task 14 占位）。
 *
 * 范围：
 *   - 患者 / 陪诊师钱包流水列表；
 *   - 类型筛选（充值 / 扣款 / 退款 / 提现）；
 *   - 行操作：详情。
 *
 * TODO: 实现页面（来自 plan v1 Task 14）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function WalletsPage() {
  return (
    <div data-testid="wallets-page">
      <PageHeader
        title="钱包流水"
        subtitle="患者 / 陪诊师钱包流水 + 类型筛选"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/wallets + 类型/主体筛选 + ProTable
        </p>
      </Card>
    </div>
  );
}