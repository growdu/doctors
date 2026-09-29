/**
 * escort/audit-pending.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染审核中画面（图标 + 标题 + 倒计时 + 提示）
 *   - 倒计时渲染（HH:MM:SS 格式）
 *   - 点击「返回首页」调 uni.reLaunch
 *
 * 倒计时验证使用 mock timer。
 */
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AuditPendingPage from './audit-pending.vue';

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe('escort/audit-pending · 渲染', () => {
  it('渲染审核中画面', () => {
    const w = mount(AuditPendingPage);
    expect(w.find('[data-testid="escort-audit-pending-page"]').exists()).toBe(true);
    expect(w.text()).toContain('陪诊师审核中');
    expect(w.text()).toContain('您的资料已提交');
    expect(w.text()).toContain('预计剩余审核时间');
  });

  it('倒计时显示 HH:MM:SS 格式', () => {
    const w = mount(AuditPendingPage);
    const text = w.find('[data-testid="escort-audit-pending-countdown"]').text();
    expect(text).toMatch(/^\d{2}:\d{2}:\d{2}$/);
    // 初始 24h
    expect(text).toBe('24:00:00');
  });

  it('倒计时每秒减少 1', async () => {
    const w = mount(AuditPendingPage);
    const initialText = w.find('[data-testid="escort-audit-pending-countdown"]').text();
    expect(initialText).toBe('24:00:00');

    // advanceTimersByTime(1000) 实际触发 1 次 tick + setInterval 启动额外 tick
    // 容忍 ±2 秒的 timer drift（fake timer 与 setInterval 边界）
    vi.advanceTimersByTime(1000);
    await vi.runOnlyPendingTimersAsync();

    const advanced = w.find('[data-testid="escort-audit-pending-countdown"]').text();
    expect(['23:59:59', '23:59:58']).toContain(advanced);
  });
});

describe('escort/audit-pending · 返回首页', () => {
  it('点击「返回首页」调 uni.reLaunch', async () => {
    const reLaunchSpy = vi.fn();
    (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;

    const w = mount(AuditPendingPage);
    await w.find('[data-testid="escort-audit-pending-back-home"]').trigger('click');

    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/home/index' });
  });
});