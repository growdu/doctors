/**
 * admin/users/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listUsers
 *   - 4 role tab + 默认 active 是 all
 *   - 点击 tab 重新拉数据（带 ?role=）
 *   - 渲染用户卡（id / 昵称 / 手机 / 角色 / 实名状态）
 *   - 实名 vs 未实名徽章渲染
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AdminUsers from './index.vue';

vi.mock('@/api/admin', () => ({
  listUsers: vi.fn(),
}));

import * as apiAdmin from '@/api/admin';
import type { AdminUser } from '@/api/admin';

const makeUser = (id: number, overrides: Partial<AdminUser> = {}): AdminUser => ({
  id,
  phone: `13900${String(100000 + id)}`,
  nickname: `用户${id}`,
  active_role: 'patient',
  roles: ['patient'],
  real_name_verified: id === 101,
  created_at: '2026-09-29T10:00:00Z',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/users/index · 加载与 tab', () => {
  it('mount 时调 listUsers 不带 role（默认 all）', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({ items: [], total: 0 });
    mount(AdminUsers);
    await flushPromises();
    expect(apiAdmin.listUsers).toHaveBeenCalledWith({});
  });

  it('渲染 4 个 role tab（all / patient / escort / super_admin）', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({ items: [], total: 0 });
    const w = mount(AdminUsers);
    await flushPromises();

    for (const t of ['all', 'patient', 'escort', 'super_admin']) {
      expect(w.find(`[data-testid="admin-users-tab-${t}"]`).exists()).toBe(true);
    }
  });

  it('默认 active tab 是 all', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({ items: [], total: 0 });
    const w = mount(AdminUsers);
    await flushPromises();
    expect(w.find('[data-testid="admin-users-tab-all"]').classes()).toContain('admin-users__tab--active');
  });

  it('点击 patient tab 重新拉数据并带 ?role=patient', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({ items: [], total: 0 });
    const w = mount(AdminUsers);
    await flushPromises();
    expect(apiAdmin.listUsers).toHaveBeenCalledTimes(1);

    await w.find('[data-testid="admin-users-tab-patient"]').trigger('click');
    await flushPromises();
    expect(apiAdmin.listUsers).toHaveBeenCalledTimes(2);
    expect(apiAdmin.listUsers).toHaveBeenLastCalledWith({ role: 'patient' });
  });
});

describe('admin/users/index · 渲染', () => {
  it('渲染用户卡（id / 昵称 / 手机 / 角色 / 实名徽章）', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({
      items: [
        makeUser(101, { real_name_verified: true }),
        makeUser(102, { real_name_verified: false, active_role: 'escort', roles: ['escort'] }),
      ],
      total: 2,
    });
    const w = mount(AdminUsers);
    await flushPromises();

    expect(w.find('[data-testid="admin-users-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-users-card-102"]').exists()).toBe(true);

    expect(w.text()).toContain('#101');
    expect(w.text()).toContain('用户101');
    expect(w.text()).toContain('13900100101');
    expect(w.text()).toContain('当前角色：患者');
    expect(w.text()).toContain('2026-09-29 10:00');
  });

  it('real_name_verified=true 显示「已实名」徽章', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({
      items: [makeUser(101, { real_name_verified: true })],
      total: 1,
    });
    const w = mount(AdminUsers);
    await flushPromises();
    expect(w.find('[data-testid="admin-users-verified-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-users-unverified-101"]').exists()).toBe(false);
  });

  it('real_name_verified=false 显示「未实名」徽章', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({
      items: [makeUser(102, { real_name_verified: false })],
      total: 1,
    });
    const w = mount(AdminUsers);
    await flushPromises();
    expect(w.find('[data-testid="admin-users-unverified-102"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-users-verified-102"]').exists()).toBe(false);
  });
});

describe('admin/users/index · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiAdmin.listUsers).mockReturnValue(new Promise(() => {}));
    const w = mount(AdminUsers);
    await flushPromises();
    expect(w.find('[data-testid="admin-users-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiAdmin.listUsers).mockResolvedValue({ items: [], total: 0 });
    const w = mount(AdminUsers);
    await flushPromises();
    expect(w.find('[data-testid="admin-users-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无用户');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiAdmin.listUsers).mockRejectedValue(new Error('网络异常'));
    const w = mount(AdminUsers);
    await flushPromises();
    expect(w.find('[data-testid="admin-users-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});