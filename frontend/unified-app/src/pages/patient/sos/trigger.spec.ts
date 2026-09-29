/**
 * patient/sos/trigger.vue 组件单测（mobile batch 4d）。
 *
 * 验证目标（仅 mobile batch 4d 新增行为）：
 *   - 渲染紧急热线区块（hotline 号码 + 拨打按钮）
 *   - 点击拨打按钮 → callPhone('400-123-4567') 被调用
 *   - callPhone 失败且非用户取消 → toast 提示
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';

vi.mock('@/utils/callPhone', () => ({
  callPhone: vi.fn().mockResolvedValue({ ok: true, channel: 'uni-call' }),
}));

vi.mock('@/api/sos', () => ({
  raiseSos: vi.fn().mockResolvedValue({ id: 9001, status: 'received', created_at: '2026-09-29T18:00:00Z' }),
}));

import SosPage from './trigger.vue';
import { callPhone } from '@/utils/callPhone';

beforeEach(() => {
  vi.clearAllMocks();
});

describe('patient/sos/trigger · 紧急热线（mobile batch 4d）', () => {
  it('渲染 hotline 区块：号码 + 拨打按钮', () => {
    const w = mount(SosPage);
    expect(w.find('[data-testid="sos-hotline-block"]').exists()).toBe(true);
    expect(w.text()).toContain('400-123-4567');
    expect(w.find('[data-testid="sos-call-hotline-btn"]').exists()).toBe(true);
  });

  it('点击拨打按钮 → callPhone(\'400-123-4567\') 被调用', async () => {
    const w = mount(SosPage);
    await w.find('[data-testid="sos-call-hotline-btn"]').trigger('click');
    await new Promise((r) => setTimeout(r, 0));
    expect(callPhone).toHaveBeenCalledTimes(1);
    expect(callPhone).toHaveBeenCalledWith('400-123-4567');
  });

  it('callPhone 失败且非用户取消 → 触发 toast', async () => {
    (callPhone as unknown as { mockResolvedValueOnce: (v: unknown) => void }).mockResolvedValueOnce({
      ok: false,
      errMsg: 'h5-tel-failed',
    });
    const showToast = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof showToast; showModal?: unknown } }).uni.showToast = showToast;

    const w = mount(SosPage);
    await w.find('[data-testid="sos-call-hotline-btn"]').trigger('click');
    await new Promise((r) => setTimeout(r, 0));
    expect(showToast).toHaveBeenCalledWith(expect.objectContaining({ title: expect.stringContaining('拨号失败') }));
  });

  it('callPhone 返回 user-cancelled → 不弹失败 toast（静默处理）', async () => {
    (callPhone as unknown as { mockResolvedValueOnce: (v: unknown) => void }).mockResolvedValueOnce({
      ok: false,
      errMsg: 'user-cancelled',
    });
    const showToast = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof showToast; showModal?: unknown } }).uni.showToast = showToast;

    const w = mount(SosPage);
    await w.find('[data-testid="sos-call-hotline-btn"]').trigger('click');
    await new Promise((r) => setTimeout(r, 0));
    expect(showToast).not.toHaveBeenCalled();
  });

  it('原有 SOS 表单仍然存在（保持向后兼容）', () => {
    const w = mount(SosPage);
    expect(w.find('[data-testid="sos-warning-banner"]').exists()).toBe(true);
    expect(w.find('[data-testid="sos-reason"]').exists()).toBe(true);
    expect(w.find('[data-testid="sos-submit-btn"]').exists()).toBe(true);
  });
});