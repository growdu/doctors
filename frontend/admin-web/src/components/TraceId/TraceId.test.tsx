/**
 * TraceId 测试契约（待 vitest 启用后跑）：
 *   - 空 traceId → 渲染 placeholder；
 *   - 有效 traceId → 渲染 <code> + 复制按钮；
 *   - 点击复制按钮 → 调 navigator.clipboard.writeText。
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TraceId } from './TraceId';

describe('TraceId', () => {
  it('空 traceId → 渲染 placeholder', () => {
    render(<TraceId traceId={null} />);
    expect(screen.getByText('— 无 trace_id —')).toBeInTheDocument();
  });

  it('有效 traceId → 渲染 <code> + 复制按钮', () => {
    render(<TraceId traceId="abc-123" />);
    expect(screen.getByText('abc-123')).toBeInTheDocument();
    expect(screen.getByTestId('trace-id-copy')).toBeInTheDocument();
  });

  it('点击复制按钮 → 调 navigator.clipboard.writeText', () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<TraceId traceId="abc-123" />);
    fireEvent.click(screen.getByTestId('trace-id-copy'));
    expect(writeText).toHaveBeenCalledWith('abc-123');
  });
});