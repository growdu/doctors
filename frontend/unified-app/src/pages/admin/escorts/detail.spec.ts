/**
 * admin/escorts/detail.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - parseQuery 提取 id 并从 listPendingEscorts 过滤找到 escort
 *   - 渲染基本信息（昵称 / 手机 / 用户 ID / 提交时间）
 *   - 资质材料列表渲染（有 / 无 两态）
 *   - 点击「通过审核」调 approveEscort + navigateBack
 *   - 点击「拒绝审核」打开 UiModal → 输入原因 → 调 rejectEscort + navigateBack
 *   - 点击「返回」调 navigateBack
 *   - 未传 id / 找不到 escort / API 异常 三态
 *
 * 策略：mock api/admin；uni.getCurrentPages 返回带 options 的栈底；uni.navigateBack 用 spy。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import EscortDetail from './detail.vue';

vi.mock('@/api/admin', () => ({
  listPendingEscorts: vi.fn(),
  approveEscort: vi.fn(),
  rejectEscort: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { PendingEscort } from '@/api/admin';

const makeEscort = (id: number, overrides: Partial<PendingEscort> = {}): PendingEscort => ({
  id,
  user_id: 100 + id,
  nickname: `陪诊师${id}`,
  phone: `13900${String(100000 + id)}`,
  qualification_urls: [`https://example.com/q1.jpg`, `https://example.com/q2.jpg`],
  submitted_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
  (globalThis as unknown as {
    uni: {
      getCurrentPages: () => Array<{ options?: Record<string, string> }>;
      navigateBack: (opts?: { delta?: number }) => Promise<void>;
    };
  }).uni.getCurrentPages = () => [{ options: { id: '101' } }];
});

describe('admin/escorts/detail · 加载', () => {
  it('parseQuery 提取 id 并从 listPendingEscorts 中查找', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makeEscort(100), makeEscort(101), makeEscort(102)],
    });
    const w = mount(EscortDetail);
    await flushPromises();

    expect(apiAdmin.listPendingEscorts).toHaveBeenCalledTimes(1);
    expect(w.text()).toContain('陪诊师101');
    expect(w.text()).toContain('13900100101');
    expect(w.text()).toContain('#201');
  });

  it('未传 id 时显示错误', async () => {
    (globalThis as unknown as {
      uni: { getCurrentPages: () => Array<{ options?: Record<string, string> }> };
    }).uni.getCurrentPages = () => [{ options: {} }];
    const w = mount(EscortDetail);
    await flushPromises();

    expect(w.find('[data-testid="escort-detail-error"]').exists()).toBe(true);
    expect(w.text()).toContain('未指定陪诊师 ID');
  });

  it('找不到 escort（已审核）显示「陪诊师不存在或已审核」', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({ items: [] });
    const w = mount(EscortDetail);
    await flushPromises();

    expect(w.find('[data-testid="escort-detail-error"]').exists()).toBe(true);
    expect(w.text()).toContain('陪诊师不存在或已审核');
  });

  it('API 异常显示错误', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockRejectedValue(new Error('网络超时'));
    const w = mount(EscortDetail);
    await flushPromises();

    expect(w.find('[data-testid="escort-detail-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});

describe('admin/escorts/detail · 资质材料', () => {
  it('无资质材料时显示「暂无」', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makeEscort(101, { qualification_urls: [] })],
    });
    const w = mount(EscortDetail);
    await flushPromises();
    expect(w.find('[data-testid="escort-detail-no-quals"]').exists()).toBe(true);
  });

  it('有资质材料时逐条渲染', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makeEscort(101)],
    });
    const w = mount(EscortDetail);
    await flushPromises();
    expect(w.find('[data-testid="escort-detail-quals"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-detail-qual-0"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-detail-qual-1"]').exists()).toBe(true);
  });
});

describe('admin/escorts/detail · 通过', () => {
  it('点击「通过审核」调 approveEscort + navigateBack', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makeEscort(101)],
    });
    vi.mocked(apiAdmin.approveEscort).mockResolvedValue(makeEscort(101));

    const backSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateBack: typeof backSpy } }).uni.navigateBack = backSpy;

    const w = mount(EscortDetail);
    await flushPromises();

    await w.find('[data-testid="escort-detail-approve"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.approveEscort).toHaveBeenCalledWith(101);
    expect(backSpy).toHaveBeenCalled();
  });
});

describe('admin/escorts/detail · 拒绝', () => {
  beforeEach(() => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makeEscort(101)],
    });
    vi.mocked(apiAdmin.rejectEscort).mockResolvedValue(makeEscort(101, { id: 101 }));
  });

  it('点击「拒绝审核」打开 modal', async () => {
    const w = mount(EscortDetail);
    await flushPromises();
    expect(w.find('[data-testid="escort-detail-reject-modal"]').exists()).toBe(false);

    await w.find('[data-testid="escort-detail-reject"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="escort-detail-reject-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 rejectEscort + navigateBack', async () => {
    const backSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateBack: typeof backSpy } }).uni.navigateBack = backSpy;

    const w = mount(EscortDetail);
    await flushPromises();

    await w.find('[data-testid="escort-detail-reject"]').trigger('click');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('资质不完整');
    await flushPromises();

    await w.find('[data-testid="escort-detail-reject-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.rejectEscort).toHaveBeenCalledWith(101, '资质不完整');
    expect(backSpy).toHaveBeenCalled();
  });

  it('原因为空时确认按钮 disabled', async () => {
    const w = mount(EscortDetail);
    await flushPromises();
    await w.find('[data-testid="escort-detail-reject"]').trigger('click');
    await flushPromises();

    const confirmBtn = w.find('[data-testid="escort-detail-reject-confirm"]');
    expect(confirmBtn.attributes('disabled')).toBeDefined();
  });

  it('点击「返回」关闭 modal，不调 rejectEscort', async () => {
    const w = mount(EscortDetail);
    await flushPromises();
    await w.find('[data-testid="escort-detail-reject"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="escort-detail-reject-modal"]').exists()).toBe(true);

    await w.find('[data-testid="escort-detail-reject-cancel"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="escort-detail-reject-modal"]').exists()).toBe(false);
    expect(apiAdmin.rejectEscort).not.toHaveBeenCalled();
  });
});

describe('admin/escorts/detail · 返回', () => {
  it('点击「返回」按钮调 navigateBack', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makeEscort(101)],
    });
    const backSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateBack: typeof backSpy } }).uni.navigateBack = backSpy;

    const w = mount(EscortDetail);
    await flushPromises();
    await w.find('[data-testid="escort-detail-back"]').trigger('click');
    expect(backSpy).toHaveBeenCalled();
  });
});