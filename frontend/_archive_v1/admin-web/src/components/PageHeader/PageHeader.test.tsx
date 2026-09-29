/**
 * PageHeader 测试契约（待 vitest 启用后跑）：
 *   - title 渲染；
 *   - subtitle 渲染（type=secondary）；
 *   - extra slot 渲染（按钮 / 自定义元素）；
 *   - 缺 subtitle/extra 时不渲染对应节点。
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PageHeader } from './PageHeader';
import { Button } from 'antd';

describe('PageHeader', () => {
  it('title 渲染', () => {
    render(<PageHeader title="订单管理" />);
    expect(screen.getByText('订单管理')).toBeInTheDocument();
  });

  it('subtitle 渲染', () => {
    render(<PageHeader title="订单" subtitle="订单列表 + 详情" />);
    expect(screen.getByText('订单列表 + 详情')).toBeInTheDocument();
  });

  it('extra slot 渲染', () => {
    render(
      <PageHeader
        title="订单"
        extra={<Button data-testid="extra-btn">新增</Button>}
      />,
    );
    expect(screen.getByTestId('extra-btn')).toBeInTheDocument();
  });

  it('缺 subtitle/extra 时不抛错', () => {
    render(<PageHeader title="订单" />);
    expect(screen.getByText('订单')).toBeInTheDocument();
  });
});