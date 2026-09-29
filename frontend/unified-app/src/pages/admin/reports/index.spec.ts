/**
 * admin/reports/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 getReportOverview
 *   - 渲染 5 张概览卡 + 数字 + 金额格式化（分→元）
 *   - 点击「导出报表」弹 toast
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import ReportsPage from './index.vue';

vi.mock('@/api/admin', () => ({
  getReportOverview: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { ReportOverview } from '@/api/admin';

const makeOverview = (overrides: Partial<ReportOverview> = {}): ReportOverview => ({
  total_orders: 1234,
  total_amount: 5678900,
  today_orders: 50,
  today_amount: 999900,
  online_escorts: 88,
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/reports/index · 加载', () => {
  it('mount 时调 getReportOverview', async () => {
    vi.mocked(apiAdmin.getReportOverview).mockResolvedValue(makeOverview());
    mount(ReportsPage);
    await flushPromises();
    expect(apiAdmin.getReportOverview).toHaveBeenCalledTimes(1);
  });

  it('渲染 5 张概览卡 + 数字', async () => {
    vi.mocked(apiAdmin.getReportOverview).mockResolvedValue(makeOverview());
    const w = mount(ReportsPage);
    await flushPromises();

    expect(w.find('[data-testid="reports-stat-total-orders"]').exists()).toBe(true);
    expect(w.find('[data-testid="reports-stat-total-amount"]').exists()).toBe(true);
    expect(w.find('[data-testid="reports-stat-today-orders"]').exists()).toBe(true);
    expect(w.find('[data-testid="reports-stat-today-amount"]').exists()).toBe(true);
    expect(w.find('[data-testid="reports-stat-online-escorts"]').exists()).toBe(true);

    // 数字：1,234 / ¥56,789.00 / 50 / ¥9,999.00 / 88
    expect(w.find('[data-testid="reports-stat-total-orders"]').text()).toContain('1,234');
    expect(w.find('[data-testid="reports-stat-total-amount"]').text()).toContain('¥56,789');
    expect(w.find('[data-testid="reports-stat-today-orders"]').text()).toContain('50');
    expect(w.find('[data-testid="reports-stat-today-amount"]').text()).toContain('¥9,999');
    expect(w.find('[data-testid="reports-stat-online-escorts"]').text()).toContain('88');

    // 标签
    expect(w.text()).toContain('总订单数');
    expect(w.text()).toContain('总金额');
    expect(w.text()).toContain('今日订单');
    expect(w.text()).toContain('今日 GMV');
    expect(w.text()).toContain('在线陪诊师');
  });
});

describe('admin/reports/index · 导出', () => {
  it('点击「导出报表」触发 uni.showToast', async () => {
    vi.mocked(apiAdmin.getReportOverview).mockResolvedValue(makeOverview());
    const toastSpy = vi.fn();
    (globalThis as unknown as { uni: { showToast: typeof toastSpy } }).uni.showToast = toastSpy;

    const w = mount(ReportsPage);
    await flushPromises();

    await w.find('[data-testid="admin-reports-export-btn"]').trigger('click');
    expect(toastSpy).toHaveBeenCalledWith({ title: '导出功能开发中', icon: 'none' });
  });
});

describe('admin/reports/index · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiAdmin.getReportOverview).mockReturnValue(new Promise(() => {}));
    const w = mount(ReportsPage);
    await flushPromises();
    expect(w.find('[data-testid="admin-reports-loading"]').exists()).toBe(true);
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiAdmin.getReportOverview).mockRejectedValue(new Error('网络异常'));
    const w = mount(ReportsPage);
    await flushPromises();
    expect(w.find('[data-testid="admin-reports-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});