/**
 * PatientDetailPage：患者详情页骨架（v1 Task 11 占位）。
 *
 * 范围：
 *   - 患者基本信息（姓名 / 手机 / 注册时间）；
 *   - 关联订单历史；
 *   - 关联退款历史。
 *
 * TODO: 实现页面（来自 plan v1 Task 11）
 */
import { Card } from 'antd';
import { useParams } from 'react-router-dom';
import { PageHeader } from '@/components/PageHeader';

export default function PatientDetailPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <div data-testid="patient-detail-page">
      <PageHeader
        title={`患者详情 #${id ?? '—'}`}
        subtitle="基本信息 + 订单历史 + 退款历史"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/patients/:id + Descriptions + 订单/退款子表
        </p>
      </Card>
    </div>
  );
}