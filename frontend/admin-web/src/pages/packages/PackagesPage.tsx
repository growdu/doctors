/**
 * PackagesPage：套餐管理页骨架（v1 Task 27 占位）。
 *
 * 范围：
 *   - 套餐列表（半日 / 全日 / 专项 / 加项）；
 *   - 创建 / 编辑 / 上下架；
 *   - 关联医院字典。
 *
 * TODO: 实现页面（来自 plan v1 Task 27）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function PackagesPage() {
  return (
    <div data-testid="packages-page">
      <PageHeader
        title="套餐管理"
        subtitle="套餐列表 + 上下架 + 关联医院"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/packages + CRUD + 上下架开关
        </p>
      </Card>
    </div>
  );
}