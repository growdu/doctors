/**
 * WalletDetailPage：钱包主体详情页骨架（v1 Task 14 占位）。
 *
 * 范围：
 *   - 主体（患者 / 陪诊师）基本信息 + 余额；
 *   - 流水明细列表。
 *
 * TODO: 实现页面（来自 plan v1 Task 14）
 */
import { Card } from 'antd';
import { useParams } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';

export default function WalletDetailPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <div data-testid="wallet-detail-page">
      <PageHeader
        title={`钱包详情 #${id ?? '—'}`}
        subtitle="主体信息 + 余额 + 流水明细"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/wallets/:id + 余额卡片 + 流水子表
        </p>
      </Card>
    </div>
  );
}