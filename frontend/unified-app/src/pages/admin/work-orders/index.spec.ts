/**
 * admin/work-orders/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listWorkOrders
 *   - 4 tab + 默认 active 是 all
 *   - 渲染工单卡（id / 类型 / 标题 / 内容 / 时间 / 状态徽章）
 *   - 点击「创建工单」打开 modal
 *   - modal 内 4 类型 chip 切换
 *   - 标题 + 内容必填；提交后调 createWorkOrder + 列表插入新项
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import WorkOrdersPage from './index.vue';

vi.mock('@/api/admin', () => ({
  listWorkOrders: vi.fn(),
  createWorkOrder: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { WorkOrder } from '@/api/admin';

const makeWorkOrder = (id: number, overrides: Partial<WorkOrder> = {}): WorkOrder => ({
  id,
  type: 'complaint',
  ref_id: null,
  ref_type: null,
  title: `工单标题 ${id}`,
  content: `工单内容 ${id}`,
  status: 'open',
  assignee: null,
  created_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/work-orders/index · 加载与 tab', () => {
  it('mount 时调 listWorkOrders 不带 status（默认 all）', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    mount(WorkOrdersPage);
    await flushPromises();
    expect(apiAdmin.listWorkOrders).toHaveBeenCalledWith({});
  });

  it('渲染 4 个 tab（all / open / in_progress / closed）', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(WorkOrdersPage);
    await flushPromises();
    for (const t of ['all', 'open', 'in_progress', 'closed']) {
      expect(w.find(`[data-testid="admin-work-orders-tab-${t}"]`).exists()).toBe(true);
    }
  });

  it('点击 open tab 重新拉数据并带 ?status=open', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(WorkOrdersPage);
    await flushPromises();

    await w.find('[data-testid="admin-work-orders-tab-open"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.listWorkOrders).toHaveBeenLastCalledWith({ status: 'open' });
  });
});

describe('admin/work-orders/index · 渲染', () => {
  it('渲染工单卡（id / 类型 / 标题 / 内容 / 状态徽章 / 时间）', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({
      items: [
        makeWorkOrder(101, { type: 'complaint' }),
        makeWorkOrder(102, { type: 'refund', status: 'in_progress', assignee: 999 }),
      ],
      total: 2,
    });
    const w = mount(WorkOrdersPage);
    await flushPromises();

    expect(w.find('[data-testid="admin-work-orders-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-work-orders-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('工单 #101');
    expect(w.text()).toContain('投诉');
    expect(w.text()).toContain('工单标题 101');
    expect(w.text()).toContain('工单内容 101');
    expect(w.text()).toContain('退款');
    expect(w.text()).toContain('处理中');
    expect(w.text()).toContain('负责人 #999');
    expect(w.text()).toContain('2026-09-29 10:00');
  });

  it('三种状态徽章正确渲染（open / in_progress / closed）', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({
      items: [
        makeWorkOrder(1, { status: 'open' }),
        makeWorkOrder(2, { status: 'in_progress' }),
        makeWorkOrder(3, { status: 'closed' }),
      ],
      total: 3,
    });
    const w = mount(WorkOrdersPage);
    await flushPromises();

    expect(w.find('[data-testid="admin-work-orders-status-1"]').text()).toContain('待处理');
    expect(w.find('[data-testid="admin-work-orders-status-2"]').text()).toContain('处理中');
    expect(w.find('[data-testid="admin-work-orders-status-3"]').text()).toContain('已关闭');
  });
});

describe('admin/work-orders/index · 创建工单', () => {
  it('点击「创建工单」打开 modal', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(WorkOrdersPage);
    await flushPromises();

    expect(w.find('[data-testid="admin-work-orders-create-modal"]').exists()).toBe(false);
    await w.find('[data-testid="admin-work-orders-create-btn"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-create-modal"]').exists()).toBe(true);
  });

  it('modal 内 4 类型 chip 切换默认 complaint + 可切换', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(WorkOrdersPage);
    await flushPromises();
    await w.find('[data-testid="admin-work-orders-create-btn"]').trigger('click');
    await flushPromises();

    // 默认 complaint 是 active
    expect(w.find('[data-testid="admin-work-orders-create-type-complaint"]').classes()).toContain(
      'admin-work-orders__type-chip--active',
    );

    // 切到 refund
    await w.find('[data-testid="admin-work-orders-create-type-refund"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-create-type-refund"]').classes()).toContain(
      'admin-work-orders__type-chip--active',
    );
    expect(w.find('[data-testid="admin-work-orders-create-type-complaint"]').classes()).not.toContain(
      'admin-work-orders__type-chip--active',
    );
  });

  it('标题 + 内容必填，缺一则 confirm 按钮 disabled', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(WorkOrdersPage);
    await flushPromises();
    await w.find('[data-testid="admin-work-orders-create-btn"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="admin-work-orders-create-confirm"]').attributes('disabled')).toBeDefined();

    // 仅填标题
    const titleInputs = w.findAll('[data-testid="ui-input-inner"]');
    await titleInputs[0]!.setValue('测试标题');
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-create-confirm"]').attributes('disabled')).toBeDefined();

    // 填内容
    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('测试内容');
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-create-confirm"]').attributes('disabled')).toBeUndefined();
  });

  it('提交 → 调 createWorkOrder + 列表插入新项', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({
      items: [makeWorkOrder(101)],
      total: 1,
    });
    vi.mocked(apiAdmin.createWorkOrder).mockResolvedValue(makeWorkOrder(102, { title: '新工单', content: '新内容' }));

    const w = mount(WorkOrdersPage);
    await flushPromises();
    await w.find('[data-testid="admin-work-orders-create-btn"]').trigger('click');
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('新工单');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('新内容');
    await flushPromises();

    await w.find('[data-testid="admin-work-orders-create-confirm"]').trigger('click');
    await flushPromises();

    expect(apiAdmin.createWorkOrder).toHaveBeenCalledWith({
      type: 'complaint',
      title: '新工单',
      content: '新内容',
    });
    expect(w.find('[data-testid="admin-work-orders-card-102"]').exists()).toBe(true);
    // 新项插入到首部
    expect(w.text()).toContain('工单 #102');
  });
});

describe('admin/work-orders/index · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockReturnValue(new Promise(() => {}));
    const w = mount(WorkOrdersPage);
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(WorkOrdersPage);
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无工单');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiAdmin.listWorkOrders).mockRejectedValue(new Error('网络异常'));
    const w = mount(WorkOrdersPage);
    await flushPromises();
    expect(w.find('[data-testid="admin-work-orders-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});