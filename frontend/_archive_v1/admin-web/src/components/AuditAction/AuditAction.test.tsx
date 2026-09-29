/**
 * AuditAction 测试契约（待 vitest 启用后跑）：
 *   - items 空 → 渲染 Empty「暂无操作记录」；
 *   - items 非空 → 渲染 Timeline + 每行 action/actor/timestamp；
 *   - max 截断：超过 max 不渲染，超出条数显示「还有 N 条」。
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AuditAction, type AuditActionItem } from './AuditAction';

const items: AuditActionItem[] = [
  { timestamp: '2026-09-24T10:00:00Z', actor: 'super', action: '强制取消订单 9001' },
  { timestamp: '2026-09-24T10:05:00Z', actor: 'order', action: '审批通过', color: 'green' },
];

describe('AuditAction', () => {
  it('items 空 → 渲染 Empty', () => {
    render(<AuditAction items={[]} />);
    expect(screen.getByText('暂无操作记录')).toBeInTheDocument();
  });

  it('items 非空 → 渲染 action + actor + 时间', () => {
    render(<AuditAction items={items} />);
    expect(screen.getByText('强制取消订单 9001')).toBeInTheDocument();
    expect(screen.getByText('审批通过')).toBeInTheDocument();
    expect(screen.getByText('super')).toBeInTheDocument();
  });

  it('max 截断超出条数', () => {
    const many: AuditActionItem[] = Array.from({ length: 5 }, (_, i) => ({
      timestamp: '2026-09-24T10:00:00Z',
      actor: 'u',
      action: `act${i}`,
    }));
    render(<AuditAction items={many} max={3} />);
    expect(screen.getByText(/还有 2 条/)).toBeInTheDocument();
    // 仅前 3 个 item 渲染
    expect(screen.getByTestId('audit-item-0')).toBeInTheDocument();
    expect(screen.queryByTestId('audit-item-3')).toBeNull();
  });
});