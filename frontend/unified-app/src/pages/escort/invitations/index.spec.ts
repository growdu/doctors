/**
 * escort/invitations/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 fetchFeed
 *   - 渲染候选订单卡（订单 ID / 医院 / 时间 / 剩余席位 / 接单数 / 截止）
 *   - 点击「接单」调 confirmAccept + 从列表移除
 *   - 点击「拒单」打开 modal → 输入原因 → 调 rejectAccept + 移除
 *   - 原因为空时 confirm 按钮 disabled
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import EscortInvitations from './index.vue';

vi.mock('@/api/match', () => ({
  fetchFeed: vi.fn(),
}));

vi.mock('@/api/orders', () => ({
  confirmAccept: vi.fn(),
  rejectAccept: vi.fn(),
  listOrders: vi.fn(),
  getOrder: vi.fn(),
  createOrder: vi.fn(),
  cancelOrder: vi.fn(),
  finishOrder: vi.fn(),
}));

import * as apiMatch from '@/api/match';
import * as apiOrders from '@/api/orders';
import type { MatchFeedItem } from '@/api/match';

const makeFeed = (id: number, overrides: Partial<MatchFeedItem> = {}): MatchFeedItem => ({
  order_id: id,
  patient_id: 100 + id,
  hospital_id: 5,
  appointment_time: '2026-10-01T09:00:00Z',
  remaining_slots: 3,
  accepted_count: 1,
  total_candidates: 5,
  deadline: '2026-10-01T08:30:00Z',
  ...overrides,
});

beforeEach(() => {
  setActivePinia(createPinia());
  // RoleGuard 需要 active_role='escort' 才放行
  const auth = useAuthStore();
  auth.user = {
    id: 1,
    phone: '13800138000',
    role: 'escort',
    active_role: 'escort',
    roles: ['escort'],
    real_name_verified: true,
  };
  vi.clearAllMocks();
});

describe('escort/invitations · 加载', () => {
  it('mount 时调 fetchFeed', async () => {
    vi.mocked(apiMatch.fetchFeed).mockResolvedValue({ items: [] });
    mount(EscortInvitations);
    await flushPromises();
    expect(apiMatch.fetchFeed).toHaveBeenCalledTimes(1);
  });
});

describe('escort/invitations · 列表渲染', () => {
  it('渲染订单卡（订单 ID / 医院 / 时间 / 剩余席位 / 截止）', async () => {
    vi.mocked(apiMatch.fetchFeed).mockResolvedValue({
      items: [makeFeed(101), makeFeed(102, { remaining_slots: 1 })],
    });
    const w = mount(EscortInvitations);
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-invitations-card-102"]').exists()).toBe(true);

    expect(w.text()).toContain('订单 #101');
    expect(w.text()).toContain('医院 #5');
    expect(w.text()).toContain('2026-10-01 09:00');
    expect(w.text()).toContain('剩余 3 / 5');
    expect(w.text()).toContain('剩余 1 / 5');
    expect(w.text()).toContain('已收到 1 单');
    expect(w.text()).toContain('截止 2026-10-01 08:30');
  });
});

describe('escort/invitations · 接单', () => {
  it('点击「接单」调 confirmAccept + 从列表移除', async () => {
    vi.mocked(apiMatch.fetchFeed).mockResolvedValue({
      items: [makeFeed(101), makeFeed(102)],
    });
    vi.mocked(apiOrders.confirmAccept).mockResolvedValue({} as never);

    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-accept-101"]').trigger('click');
    await flushPromises();

    expect(apiOrders.confirmAccept).toHaveBeenCalledWith(101);
    expect(w.find('[data-testid="escort-invitations-card-101"]').exists()).toBe(false);
    expect(w.find('[data-testid="escort-invitations-card-102"]').exists()).toBe(true);
  });
});

describe('escort/invitations · 拒单', () => {
  beforeEach(() => {
    vi.mocked(apiMatch.fetchFeed).mockResolvedValue({
      items: [makeFeed(101)],
    });
  });

  it('点击「拒单」打开 modal', async () => {
    const w = mount(EscortInvitations);
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-reject-modal"]').exists()).toBe(false);
    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-reject-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 rejectAccept + 移除', async () => {
    vi.mocked(apiOrders.rejectAccept).mockResolvedValue({} as never);
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    const textarea = w.find('[data-testid="ui-input-textarea"]');
    await textarea.setValue('已接其他单');
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-confirm"]').trigger('click');
    await flushPromises();

    expect(apiOrders.rejectAccept).toHaveBeenCalledWith(101, '已接其他单');
    expect(w.find('[data-testid="escort-invitations-card-101"]').exists()).toBe(false);
  });

  it('原因为空时 confirm 按钮 disabled', async () => {
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-reject-confirm"]').attributes('disabled')).toBeDefined();
  });

  it('点击「返回」关闭 modal，不调 rejectAccept', async () => {
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-cancel"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-reject-modal"]').exists()).toBe(false);
    expect(apiOrders.rejectAccept).not.toHaveBeenCalled();
  });
});

describe('escort/invitations · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiMatch.fetchFeed).mockReturnValue(new Promise(() => {}));
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiMatch.fetchFeed).mockResolvedValue({ items: [] });
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无可接订单');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiMatch.fetchFeed).mockRejectedValue(new Error('网络超时'));
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});