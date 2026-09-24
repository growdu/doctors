/**
 * AuditAction：列出操作记录（时间 + 操作者 + 动作）。
 *
 * 设计要点：
 *   - Timeline 风格：左侧时间戳 + 右侧操作者 / 动作；
 *   - 支持自定义 items 数组，缺省渲染占位「暂无记录」；
 *   - 单 item 字段：timestamp(ISO string) + actor + action + optional note；
 *   - data-testid：顶层 + 每行加 item-{idx}。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §5.2 (Audit)
 */
import { Empty, Tag, Timeline, Typography } from 'antd';
import dayjs from 'dayjs';

const { Text } = Typography;

export interface AuditActionItem {
  /** ISO 时间字符串 */
  timestamp: string;
  /** 操作者（用户名 / 角色） */
  actor: string;
  /** 动作描述（如「强制取消订单」「审批通过退款」） */
  action: string;
  /** 备注（可选） */
  note?: string;
  /** 自定义颜色（default / processing / success / warning / error） */
  color?: 'blue' | 'green' | 'orange' | 'red' | 'gray';
}

export interface AuditActionProps {
  items?: AuditActionItem[];
  testId?: string;
  /** 最大展示条数；超出折叠（默认 50） */
  max?: number;
}

const COLOR_MAP: Record<NonNullable<AuditActionItem['color']>, string> = {
  blue: 'blue',
  green: 'green',
  orange: 'orange',
  red: 'red',
  gray: 'gray',
};

export function AuditAction({ items = [], testId, max = 50 }: AuditActionProps) {
  if (items.length === 0) {
    return (
      <div data-testid={testId ?? 'audit-action'}>
        <Empty description="暂无操作记录" />
      </div>
    );
  }
  const visible = items.slice(0, max);
  return (
    <div data-testid={testId ?? 'audit-action'}>
      <Timeline
        items={visible.map((it, idx) => ({
          color: COLOR_MAP[it.color ?? 'blue'],
          children: (
            <div data-testid={`audit-item-${idx}`}>
              <div>
                <Text strong>{it.action}</Text>
                {it.note ? (
                  <Text type="secondary" style={{ marginLeft: 8 }}>
                    {it.note}
                  </Text>
                ) : null}
              </div>
              <div style={{ marginTop: 4 }}>
                <Tag>{dayjs(it.timestamp).format('YYYY-MM-DD HH:mm:ss')}</Tag>
                <Tag color="geekblue">{it.actor}</Tag>
              </div>
            </div>
          ),
        }))}
      />
      {items.length > max ? (
        <Text type="secondary">… 还有 {items.length - max} 条</Text>
      ) : null}
    </div>
  );
}

export default AuditAction;