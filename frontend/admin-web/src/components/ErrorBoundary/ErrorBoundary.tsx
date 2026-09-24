/**
 * ErrorBoundary：class component + getDerivedStateFromError + reset 按钮。
 *
 * 设计要点：
 *   - 捕获子组件树 render 阶段抛出的错误；
 *   - 触发 componentDidCatch 上报（mock console.error 占位）；
 *   - 提供 reset 按钮让用户重置 error state，重新尝试渲染；
 *   - 自定义 fallback：未传则默认 Result 500 + reset 按钮。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §5.2
 */
import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Button, Result } from 'antd';

export interface ErrorBoundaryProps {
  children: ReactNode;
  /** 自定义 fallback；不传则用内置 Result 500 */
  fallback?: ReactNode | ((error: Error, reset: () => void) => ReactNode);
  /** 错误回调（上报埋点可挂这里） */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  /** 顶层 data-testid */
  testId?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // 默认 console.error 上报，业务可在 onError 注入埋点
    // eslint-disable-next-line no-console
    console.error('[ErrorBoundary]', error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  private readonly reset = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    const { hasError, error } = this.state;
    const { children, fallback, testId } = this.props;

    if (!hasError) return <>{children}</>;

    if (typeof fallback === 'function') {
      return <>{fallback(error ?? new Error('unknown'), this.reset)}</>;
    }
    if (fallback) return <>{fallback}</>;

    return (
      <div data-testid={testId ?? 'error-boundary'}>
        <Result
          status="500"
          title="500"
          subTitle={error?.message ?? '页面渲染出错'}
          extra={
            <Button type="primary" onClick={this.reset} data-testid="error-reset">
              重试
            </Button>
          }
        />
      </div>
    );
  }
}

export default ErrorBoundary;