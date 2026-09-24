/**
 * ProfilePage：当前用户个人资料页骨架（v1 Task 20 占位）。
 *
 * 范围：
 *   - 当前登录管理员信息（头像 / 角色 / 上次登录）；
 *   - 修改密码表单；
 *   - 操作日志入口（最近 20 条）。
 *
 * TODO: 实现页面（来自 plan v1 Task 20）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function ProfilePage() {
  return (
    <div data-testid="profile-page">
      <PageHeader
        title="个人资料"
        subtitle="当前管理员信息 + 修改密码 + 操作日志"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/me + 修改密码 + AuditAction
        </p>
      </Card>
    </div>
  );
}