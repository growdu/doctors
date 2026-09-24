import { Typography } from 'antd';

const { Title } = Typography;

/**
 * Escorts 占位页（Task 3 雏形）：
 * 后续 Task 会按 spec §3.1 拆分为：escort_list / escort_detail / escort_audit 三个子页。
 */
export default function EscortsPage() {
  return (
    <div>
      <Title level={3}>陪诊师管理</Title>
      <p>陪诊师列表 / 详情 / 审核队列占位（v1 后续实现）。</p>
    </div>
  );
}