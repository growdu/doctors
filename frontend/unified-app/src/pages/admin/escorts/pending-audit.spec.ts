/**
 * admin/escorts/pending-audit.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listPendingEscorts
 *   - 渲染列表（昵称 / 手机 / 提交时间）+ 「待审核」徽章
 *   - 点击「通过」调 approveEscort 并从列表移除该项
 *   - 点击「拒绝」打开 UiModal → 输入原因 → 确认调 rejectEscort 并移除
 *   - 点击「查看详情」跳 detail?id=
 *   - 空 / 加载 / 错误 三态
 *
 * 策略：mock api/admin 模块。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import EscortsAudit from './pending-audit.vue';

vi.mock('@/api/admin', () => ({
  listPendingEscorts: vi.fn(),
  approveEscort: vi.fn(),
  rejectEscort: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { PendingEscort } from '@/api/admin';

const makePending = (id: number, nickname = `小陪${id}`): PendingEscort => ({
  id,
  user_id: 100 + id,
  nickname,
  phone: `13900${String(100000 + id)}`,
  qualification_urls: id === 1 ? [] : [`https://example.com/q${id}.jpg`],
  submitted_at: '2026-09-29T10:00:00Z',
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/escorts/pending-audit · 加载', () => {
  it('mount 时调 listPendingEscorts', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({ items: [] });
    mount(EscortsAudit);
    await flushPromises();
    expect(apiAdmin.listPendingEscorts).toHaveBeenCalledTimes(1);
  });
});

describe('admin/escorts/pending-audit · 列表渲染', () => {
  it('渲染昵称、手机、提交时间、待审核徽章', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makePending(101, '张三'), makePending(102, '李四')],
    });
    const w = mount(EscortsAudit);
    await flushPromises();

    expect(w.find('[data-testid="audit-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="audit-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('#101 张三');
    expect(w.text()).toContain('#102 李四');
    expect(w.text()).toContain('13900100101');
    expect(w.text()).toContain('13900100102');
    expect(w.text()).toContain('2026-09-29 10:00');

    expect(w.find('[data-testid="audit-status-101"]').exists()).toBe(true);
    expect(w.text()).toContain('待审核');
  });
});

describe('admin/escorts/pending-audit · 通过', () => {
  it('点击「通过」调 approveEscort 并从列表移除', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makePending(101), makePending(102)],
    });
    vi.mocked(apiAdmin.approveEscort).mockResolvedValue(makePending(101));

    const w = mount(EscortsAudit);
    await flushPromises();

    await w.find('[data-testid="audit-approve-101"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.approveEscort).toHaveBeenCalledWith(101);
    expect(w.find('[data-testid="audit-card-101"]').exists()).toBe(false);
    expect(w.find('[data-testid="audit-card-102"]').exists()).toBe(true);
  });
});

describe('admin/escorts/pending-audit · 拒绝', () => {
  beforeEach(() => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makePending(101), makePending(102)],
    });
  });

  it('点击「拒绝」打开 modal', async () => {
    const w = mount(EscortsAudit);
    await flushPromises();
    expect(w.find('[data-testid="audit-reject-modal"]').exists()).toBe(false);

    await w.find('[data-testid="audit-reject-101"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="audit-reject-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 rejectEscort 并移除', async () => {
    vi.mocked(apiAdmin.rejectEscort).mockResolvedValue(makePending(101));
    const w = mount(EscortsAudit);
    await flushPromises();

    await w.find('[data-testid="audit-reject-101"]').trigger('click');
    await flushPromises();

    // 设置原因
    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('健康证过期');
    await flushPromises();

    await w.find('[data-testid="audit-reject-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.rejectEscort).toHaveBeenCalledWith(101, '健康证过期');
    expect(w.find('[data-testid="audit-card-101"]').exists()).toBe(false);
  });

  it('原因为空时确认按钮 disabled', async () => {
    const w = mount(EscortsAudit);
    await flushPromises();
    await w.find('[data-testid="audit-reject-101"]').trigger('click');
    await flushPromises();

    const confirmBtn = w.find('[data-testid="audit-reject-confirm"]');
    expect(confirmBtn.attributes('disabled')).toBeDefined();
  });

  it('点击「返回」关闭 modal，不调 rejectEscort', async () => {
    const w = mount(EscortsAudit);
    await flushPromises();

    await w.find('[data-testid="audit-reject-101"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="audit-reject-modal"]').exists()).toBe(true);

    await w.find('[data-testid="audit-reject-cancel"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="audit-reject-modal"]').exists()).toBe(false);
    expect(apiAdmin.rejectEscort).not.toHaveBeenCalled();
  });
});

describe('admin/escorts/pending-audit · 跳转', () => {
  it('点击「查看详情」跳 detail?id=', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({
      items: [makePending(101)],
    });
    const w = mount(EscortsAudit);
    await flushPromises();

    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;

    await w.find('[data-testid="audit-detail-101"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/admin/escorts/detail?id=101' });
  });
});

describe('admin/escorts/pending-audit · 空 / 加载 / 错误 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockReturnValue(new Promise(() => {}));
    const w = mount(EscortsAudit);
    await flushPromises();
    expect(w.find('[data-testid="audit-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockResolvedValue({ items: [] });
    const w = mount(EscortsAudit);
    await flushPromises();
    expect(w.find('[data-testid="audit-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无待审核陪诊师');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiAdmin.listPendingEscorts).mockRejectedValue(new Error('网络异常'));
    const w = mount(EscortsAudit);
    await flushPromises();
    expect(w.find('[data-testid="audit-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
    expect(w.text()).toContain('点击重试');
  });
});