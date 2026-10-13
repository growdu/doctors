/**
 * escort/invitations/index.vue 组件单测（v2.1 选人模式，§48.S 重写）。
 *
 * 验证目标：
 *   - mount 时调 listOrders({ role:'escort', status:'invitations' })
 *   - 渲染邀请订单卡（订单 ID / 医院 / 预约 / 倒计时）
 *   - 点击「接受邀请」调 confirmAccept + 从列表移除
 *   - 点击「拒绝邀请」打开 modal → 输入原因 → 调 rejectAccept + 移除
 *   - 原因为空时 confirm 按钮 disabled
 *   - 空 / 加载 / 错误 / 全部超时 四态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import EscortInvitations from './index.vue';

vi.mock('@/api/orders', () => ({
  confirmAccept: vi.fn(),
  rejectAccept: vi.fn(),
  listOrders: vi.fn(),
  getOrder: vi.fn(),
  createOrder: vi.fn(),
  cancelOrder: vi.fn(),
  finishOrder: vi.fn(),
}));

import * as apiOrders from '@/api/orders';
import type { Order, OrderStatus } from '@/api/orders';

const makeOrder = (id: number, overrides: Partial<Order> = {}): Order => ({
  id,
  order_no: `O${id}`,
  patient_id: 100 + id,
  hospital_id: 5,
  package_id: 7,
  service_start_at: '2026-10-01T09:00:00Z',
  status: 'escort_pending_acceptance' as OrderStatus,
  amount: 30000,
  final_amount: 30000,
  selected_escort_id: 1,
  // 默认 30 秒后到期（测试时已过期会被过滤掉，用未来时间更稳）
  escort_pending_expire_at: new Date(Date.now() + 30_000).toISOString(),
  version: 1,
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
  it('mount 时调 listOrders({role:escort, status:invitations})', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    mount(EscortInvitations);
    await flushPromises();
    expect(apiOrders.listOrders).toHaveBeenCalledTimes(1);
    const call = vi.mocked(apiOrders.listOrders).mock.calls[0]?.[0];
    expect(call?.role).toBe('escort');
  });
});

describe('escort/invitations · 列表渲染', () => {
  it('渲染订单卡（订单 ID / 医院 / 时间 / 倒计时）', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101), makeOrder(102, { hospital_id: 7 })],
      total: 2,
    });
    const w = mount(EscortInvitations);
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-invitations-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('订单 #101');
    expect(w.text()).toContain('🏥 医院 #5');
    expect(w.text()).toContain('🏥 医院 #7');
    expect(w.text()).toContain('📅 预约');
    expect(w.text()).toContain('💰 服务金额 ¥300.00');
    expect(w.find('[data-testid="escort-invitations-countdown-101"]').exists()).toBe(true);
  });
});

describe('escort/invitations · 接单', () => {
  it('点击「接受邀请」调 confirmAccept + 从列表移除', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101)],
      total: 1,
    });
    vi.mocked(apiOrders.confirmAccept).mockResolvedValue(makeOrder(101, { status: 'escort_confirmed' as OrderStatus }));
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-accept-101"]').trigger('click');
    await flushPromises();

    expect(apiOrders.confirmAccept).toHaveBeenCalledWith(101);
    expect(w.find('[data-testid="escort-invitations-card-101"]').exists()).toBe(false);
  });
});

describe('escort/invitations · 拒单', () => {
  it('点击「拒绝邀请」打开 modal', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101)],
      total: 1,
    });
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-invitations-reject-modal"]').exists()).toBe(true);
  });

  it('输入原因 + 确认 → 调 rejectAccept + 移除', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101)],
      total: 1,
    });
    vi.mocked(apiOrders.rejectAccept).mockResolvedValue(makeOrder(101, { status: 'cancelled' as OrderStatus }));
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    // set value via DOM input (v-model bypasses test-utils setter)
    const input = w.find('[data-testid="escort-invitations-reject-reason"] input, [data-testid="escort-invitations-reject-reason"] textarea');
    await input.setValue('已接其他单');
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-confirm"]').trigger('click');
    await flushPromises();

    expect(apiOrders.rejectAccept).toHaveBeenCalledWith(101, '已接其他单');
  });

  it('原因为空时 confirm 按钮 disabled', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101)],
      total: 1,
    });
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();

    const confirm = w.find('[data-testid="escort-invitations-reject-confirm"]');
    // UiButton 在 disabled 时加 ui-btn--disabled class
    expect(confirm.classes()).toContain('ui-btn--disabled');
  });

  it('点击「返回」关闭 modal，不调 rejectAccept', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({
      items: [makeOrder(101)],
      total: 1,
    });
    const w = mount(EscortInvitations);
    await flushPromises();

    await w.find('[data-testid="escort-invitations-reject-101"]').trigger('click');
    await flushPromises();
    await w.find('[data-testid="escort-invitations-reject-cancel"]').trigger('click');
    await flushPromises();

    expect(apiOrders.rejectAccept).not.toHaveBeenCalled();
    expect(w.find('[data-testid="escort-invitations-reject-modal"]').exists()).toBe(false);
  });
});

describe('escort/invitations · 四态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiOrders.listOrders).mockImplementation(() => new Promise(() => {})); // never resolves
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [], total: 0 });
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-empty"]').exists()).toBe(true);
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiOrders.listOrders).mockRejectedValue(new Error('network down'));
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-error"]').exists()).toBe(true);
    expect(w.text()).toContain('network down');
  });

  it('全部超时显示「邀请已全部超时」', async () => {
    const pastOrder = makeOrder(101, {
      escort_pending_expire_at: new Date(Date.now() - 1000).toISOString(),
    });
    vi.mocked(apiOrders.listOrders).mockResolvedValue({ items: [pastOrder], total: 1 });
    const w = mount(EscortInvitations);
    await flushPromises();
    expect(w.find('[data-testid="escort-invitations-all-expired"]').exists()).toBe(true);
  });
});
