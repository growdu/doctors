/**
 * escort/training/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listTrainings
 *   - 渲染培训卡（标题 / 完成时间 / 证书链接）
 *   - 点击证书「点击复制」触发 uni.setClipboardData
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import TrainingPage from './index.vue';

vi.mock('@/api/escort', () => ({
  listTrainings: vi.fn(),
}));

import * as apiEscort from '@/api/escort';
import type { Training } from '@/api/escort';

const makeTraining = (id: number, overrides: Partial<Training> = {}): Training => ({
  id,
  escort_id: 2001,
  title: `陪诊师培训 ${id}`,
  completed_at: '2026-09-15T10:00:00Z',
  cert_url: `https://example.com/cert-${id}.pdf`,
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('escort/training · 加载', () => {
  it('mount 时调 listTrainings', async () => {
    vi.mocked(apiEscort.listTrainings).mockResolvedValue({ items: [] });
    mount(TrainingPage);
    await flushPromises();
    expect(apiEscort.listTrainings).toHaveBeenCalledTimes(1);
  });

  it('渲染培训卡（标题 / 完成时间 / 证书）', async () => {
    vi.mocked(apiEscort.listTrainings).mockResolvedValue({
      items: [makeTraining(101), makeTraining(102, { cert_url: null })],
    });
    const w = mount(TrainingPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-training-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-training-card-102"]').exists()).toBe(true);

    expect(w.text()).toContain('陪诊师培训 101');
    expect(w.text()).toContain('陪诊师培训 102');
    expect(w.text()).toContain('完成于 2026-09-15');
    expect(w.find('[data-testid="escort-training-cert-101"]').exists()).toBe(true);
    // 无证书时不显示 cert link
    expect(w.find('[data-testid="escort-training-cert-102"]').exists()).toBe(false);
  });

  it('点击证书「点击复制」触发 uni.setClipboardData', async () => {
    vi.mocked(apiEscort.listTrainings).mockResolvedValue({
      items: [makeTraining(101)],
    });
    const setClipboardSpy = vi.fn();
    (globalThis as unknown as { uni: { setClipboardData: typeof setClipboardSpy } }).uni.setClipboardData = setClipboardSpy;

    const w = mount(TrainingPage);
    await flushPromises();
    await w.find('[data-testid="escort-training-cert-101"]').trigger('click');

    expect(setClipboardSpy).toHaveBeenCalledWith({ data: 'https://example.com/cert-101.pdf' });
  });
});

describe('escort/training · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiEscort.listTrainings).mockReturnValue(new Promise(() => {}));
    const w = mount(TrainingPage);
    await flushPromises();
    expect(w.find('[data-testid="escort-training-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiEscort.listTrainings).mockResolvedValue({ items: [] });
    const w = mount(TrainingPage);
    await flushPromises();
    expect(w.find('[data-testid="escort-training-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无培训记录');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiEscort.listTrainings).mockRejectedValue(new Error('网络异常'));
    const w = mount(TrainingPage);
    await flushPromises();
    expect(w.find('[data-testid="escort-training-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});