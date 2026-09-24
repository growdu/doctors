/**
 * PageHeader：标题 + 副标题 + extra 区（操作按钮 slot）。
 *
 * 设计要点：
 *   - 风格对齐 antd ProComponents 的 PageHeader；
 *   - 顶部一行：左侧 title（h2）+ subtitle（muted）；
 *   - 顶部一行：右侧 extra slot（按钮组）；
 *   - 底部 divider 分隔线。
 *
 * 对应 spec：2026-09-24-admin-web-design.md §5.2
 */
import type { ReactNode } from 'react';
import { Typography, Divider, Space } from 'antd';

const { Title, Text } = Typography;

export interface PageHeaderProps {
  title: ReactNode;
  subtitle?: ReactNode;
  /** 右侧操作区（按钮组） */
  extra?: ReactNode;
  /** 面包屑（占位，本期未实现） */
  breadcrumb?: ReactNode;
  /** 顶层 data-testid */
  testId?: string;
}

export function PageHeader({ title, subtitle, extra, breadcrumb, testId }: PageHeaderProps) {
  return (
    <div data-testid={testId ?? 'page-header'} style={{ marginBottom: 16 }}>
      {breadcrumb ? (
        <div style={{ marginBottom: 8 }}>
          <Space size={4}>{breadcrumb}</Space>
        </div>
      ) : null}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 16,
          flexWrap: 'wrap',
        }}
      >
        <div>
          <Title level={3} style={{ margin: 0 }}>
            {title}
          </Title>
          {subtitle ? (
            <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
              {subtitle}
            </Text>
          ) : null}
        </div>
        {extra ? <Space>{extra}</Space> : null}
      </div>
      <Divider style={{ marginTop: 16, marginBottom: 0 }} />
    </div>
  );
}

export default PageHeader;