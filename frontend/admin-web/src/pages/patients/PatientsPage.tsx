/**
 * PatientsPage：患者列表页骨架（v1 Task 11 占位）。
 *
 * 范围：
 *   - 列出所有患者（按手机号 / 姓名搜索）；
 *   - 行操作：详情（订单历史 + 退款历史）。
 *
 * TODO: 实现页面（来自 plan v1 Task 11）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function PatientsPage() {
  return (
    <div data-testid="patients-page">
      <PageHeader
        title="患者管理"
        subtitle="患者列表 · 详情 · 订单/退款历史"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/patients + ProTable 渲染 + 手机号搜索 + 跳详情
        </p>
      </Card>
    </div>
  );
}