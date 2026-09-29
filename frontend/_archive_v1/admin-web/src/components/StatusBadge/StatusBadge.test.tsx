/**
 * StatusBadge v2 测试：验证 selecting_escort (橙色) + escort_pending_acceptance (蓝色) 渲染。
 *
 * 测试策略：
 *   - AntD Badge.status="warning" / "processing" 会在 .ant-badge 内嵌
 *     .ant-badge-status-{color} class 用于 CSS 着色；
 *   - 我们直接断言该 className 存在 + 文本内容正确。
 *
 * 注：当前骨架未装 vitest / @testing-library/react，本测试文件为
 * 「契约 + 验收脚本」式样，写好待用户在本地 `npm install` 后跑 vitest。
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StatusBadge } from './StatusBadge';

describe('StatusBadge v2 新状态色', () => {
  it('selecting_escort 渲染橙色 Badge + 文本"待患者选人"', () => {
    render(<StatusBadge status="selecting_escort" />);
    const text = screen.getByText('待患者选人');
    expect(text).toBeInTheDocument();
    // AntD Badge.status="warning" → 父节点 .ant-badge-status-warning
    const badge = text.closest('.ant-badge-status-warning');
    expect(badge).not.toBeNull();
    // data 属性校验：便于 e2e / RTL 双重定位
    expect(text.closest('[data-status]')?.getAttribute('data-status')).toBe('selecting_escort');
  });

  it('escort_pending_acceptance 渲染蓝色 Badge + 文本"待陪诊师确认"', () => {
    render(<StatusBadge status="escort_pending_acceptance" />);
    const text = screen.getByText('待陪诊师确认');
    expect(text).toBeInTheDocument();
    const badge = text.closest('.ant-badge-status-processing');
    expect(badge).not.toBeNull();
    expect(text.closest('[data-status]')?.getAttribute('data-status')).toBe(
      'escort_pending_acceptance',
    );
  });

  it('未知 status 兜底渲染灰 Badge + 原文本（不抛错）', () => {
    render(<StatusBadge status="some_unknown_status" />);
    const text = screen.getByText('some_unknown_status');
    expect(text).toBeInTheDocument();
    const badge = text.closest('.ant-badge-status-default');
    expect(badge).not.toBeNull();
  });
});