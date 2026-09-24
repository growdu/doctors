/**
 * MessagesPage：消息中心页骨架（v1 Task 17 占位）。
 *
 * 范围：
 *   - 系统通知 / 公告 / 工单消息；
 *   - 已读 / 未读筛选；
 *   - 行操作：标记已读 / 全部已读。
 *
 * TODO: 实现页面（来自 plan v1 Task 17）
 */
import { Card } from 'antd';
import { PageHeader } from '@/components/PageHeader';

export default function MessagesPage() {
  return (
    <div data-testid="messages-page">
      <PageHeader
        title="消息中心"
        subtitle="系统通知 / 公告 / 工单消息"
      />
      <Card>
        <p style={{ color: '#999' }}>
          TODO: 接入 MSW GET /api/v1/admin/messages + 已读/未读筛选 + 标记已读
        </p>
      </Card>
    </div>
  );
}