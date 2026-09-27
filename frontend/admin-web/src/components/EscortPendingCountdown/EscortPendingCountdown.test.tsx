/**
 * EscortPendingCountdown 单元测试：
 *   - 未来 90s：蓝色「剩余 Xs」，data-critical=false；
 *   - 未来 30s：红色（< 60s 阈值），data-critical=true；
 *   - 已过期（diff <= 0）：文案「已超时」，data-expired=true；
 *   - 卸载时清理 setInterval（不写泄漏断言，靠 fake timers + cleanup 间接测）。
 *
 * 策略：vi.useFakeTimers() + dayjs 真实计算避开时区问题（expireAt 用未来/过去的固定秒数）。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import dayjs from 'dayjs';
import { EscortPendingCountdown } from './EscortPendingCountdown';

describe('EscortPendingCountdown v1 收官', () => {
  beforeEach(() => {
    // 把系统时间钉在固定点，避免跨时区 / ms 抖动导致 remaining 偏 1s
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-09-24T10:00:00Z'));
  });
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it('未来 90s（> 阈值）：蓝色「剩余 90s」，data-critical=false', () => {
    const expireAt = dayjs().add(90, 'second').toISOString();
    render(<EscortPendingCountdown expireAt={expireAt} />);
    const el = screen.getByTestId('escort-countdown');
    expect(el).toHaveTextContent('剩余 90s');
    expect(el).toHaveAttribute('data-critical', 'false');
  });

  it('未来 30s（< 阈值）：红色「剩余 30s」，data-critical=true', () => {
    const expireAt = dayjs().add(30, 'second').toISOString();
    render(<EscortPendingCountdown expireAt={expireAt} />);
    const el = screen.getByTestId('escort-countdown');
    expect(el).toHaveTextContent('剩余 30s');
    expect(el).toHaveAttribute('data-critical', 'true');
  });

  it('过去时间（diff <= 0）：文案「已超时」，data-expired=true', () => {
    const expireAt = dayjs().subtract(5, 'second').toISOString();
    render(<EscortPendingCountdown expireAt={expireAt} />);
    const el = screen.getByTestId('escort-countdown');
    expect(el).toHaveTextContent('已超时');
    expect(el).toHaveAttribute('data-expired', 'true');
  });
});
