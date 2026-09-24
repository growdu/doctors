/**
 * SettingsPage：系统设置页骨架（v1 Task 19 占位）。
 *
 * 范围：
 *   - 通用设置（站点名 / Logo / 联系信息）；
 *   - 业务配置（套餐模板 / 医院字典）；
 *   - 权限管理（角色矩阵）。
 *
 * TODO: 实现页面（来自 plan v1 Task 19）
 */
import { Card, Tabs } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function SettingsPage() {
  return (
    <div data-testid="settings-page">
      <PageHeader
        title="系统设置"
        subtitle="通用 / 业务 / 权限 三段式"
      />
      <Card>
        <Tabs
          items={[
            { key: 'general', label: '通用', children: <p>TODO: 站点名 / Logo / 联系信息</p> },
            { key: 'biz', label: '业务配置', children: <p>TODO: 套餐模板 / 医院字典</p> },
            { key: 'rbac', label: '权限管理', children: <p>TODO: 角色矩阵</p> },
          ]}
        />
      </Card>
    </div>
  );
}