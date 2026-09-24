/**
 * HospitalsPage：医院字典页骨架（v1 Task 26 占位）。
 *
 * 范围：
 *   - 医院列表（名称 / 城市 / 等级 / 状态）；
 *   - 创建 / 编辑 / 停用医院；
 *   - 关联科室字典。
 *
 * TODO: 实现页面（来自 plan v1 Task 26）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function HospitalsPage() {
  return (
    <div data-testid="hospitals-page">
      <PageHeader
        title="医院字典"
        subtitle="医院列表 + 科室字典 + 状态管理"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/hospitals + CRUD + 科室子表
        </p>
      </Card>
    </div>
  );
}