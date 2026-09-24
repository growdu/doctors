/**
 * ErrorBoundary 测试契约（待 vitest 启用后跑）：
 *   - 正常子组件 → render children；
 *   - 子组件抛错 → 渲染 Result 500 + reset 按钮；
 *   - 点击 reset → 重新尝试 render children。
 *
 * 注：当前骨架未装 vitest / @testing-library/react，本文件为契约样。
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorBoundary } from './ErrorBoundary';

// 强制子组件抛错的 helper
function Bomb({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) throw new Error('boom');
  return <div data-testid="ok">ok</div>;
}

describe('ErrorBoundary', () => {
  it('正常子组件 → render children', () => {
    render(
      <ErrorBoundary>
        <Bomb shouldThrow={false} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId('ok')).toBeInTheDocument();
  });

  it('子组件抛错 → 渲染 Result 500 + reset 按钮', () => {
    render(
      <ErrorBoundary>
        <Bomb shouldThrow />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId('error-reset')).toBeInTheDocument();
    expect(screen.queryByTestId('ok')).toBeNull();
  });

  it('点击 reset → 重新 render children', () => {
    const onError = vi.fn();
    const { rerender } = render(
      <ErrorBoundary onError={onError}>
        <Bomb shouldThrow />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId('error-reset')).toBeInTheDocument();
    // 模拟 reset：手动把 child 的 shouldThrow 改成 false + 重渲
    rerender(
      <ErrorBoundary onError={onError}>
        <Bomb shouldThrow={false} />
      </ErrorBoundary>,
    );
    fireEvent.click(screen.getByTestId('error-reset'));
    expect(screen.getByTestId('ok')).toBeInTheDocument();
    expect(onError).toHaveBeenCalled();
  });
});