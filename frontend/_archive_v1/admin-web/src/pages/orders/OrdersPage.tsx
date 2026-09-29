import { Typography } from 'antd';

const { Title } = Typography;

/**
 * Orders 占位页（Task 3 雏形）：
 * 后续 Task 会按 spec §4.1 接入 ProTable + StatusBadge + 状态机操作
 * （订单号 / 患者 / 医院 / 金额 / 状态 / 操作 + 5s 轮询）。
 */
export default function OrdersPage() {
  return (
    <div>
      <Title level={3}>订单管理</Title>
      <p>ProTable + 状态机操作占位（v1 后续实现）。</p>
    </div>
  );
}