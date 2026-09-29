/**
 * admin/packages/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listHospitals 拉医院列表
 *   - 渲染医院选择区
 *   - 点击医院 → 调 listPackagesByHospital + 渲染套餐列表
 *   - 套餐为空 / API 错误显示对应提示
 *   - 医院为空 / 加载中显示对应提示
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AdminPackages from './index.vue';

vi.mock('@/api/user', () => ({
  listHospitals: vi.fn(),
  listPackagesByHospital: vi.fn(),
}));

import * as apiUser from '@/api/user';
import type { Hospital, Package } from '@/api/user';

const makeHospital = (id: number, overrides: Partial<Hospital> = {}): Hospital => ({
  id,
  name: `医院${id}`,
  city: '北京',
  address: '北京市朝阳区',
  phone: `010-${10000000 + id}`,
  level: '三级甲等',
  ...overrides,
});

const makePackage = (id: number, overrides: Partial<Package> = {}): Package => ({
  id,
  hospital_id: 101,
  title: `套餐 ${id}`,
  description: '全程陪诊',
  price: 29900,
  duration_minutes: 120,
  cover_url: null,
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('admin/packages/index · 加载医院', () => {
  it('mount 时调 listHospitals', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({ items: [], total: 0 });
    mount(AdminPackages);
    await flushPromises();
    expect(apiUser.listHospitals).toHaveBeenCalledTimes(1);
  });

  it('渲染医院选择列表', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({
      items: [makeHospital(101), makeHospital(102, { name: '协和医院', city: '上海' })],
      total: 2,
    });
    const w = mount(AdminPackages);
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-hospital-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-packages-hospital-102"]').exists()).toBe(true);
    expect(w.text()).toContain('医院101');
    expect(w.text()).toContain('协和医院');
  });

  it('医院列表加载中显示 UiLoading', async () => {
    vi.mocked(apiUser.listHospitals).mockReturnValue(new Promise(() => {}));
    const w = mount(AdminPackages);
    await flushPromises();
    expect(w.find('[data-testid="admin-packages-hospitals-loading"]').exists()).toBe(true);
  });

  it('医院列表为空显示 UiEmpty', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({ items: [], total: 0 });
    const w = mount(AdminPackages);
    await flushPromises();
    expect(w.find('[data-testid="admin-packages-hospitals-empty"]').exists()).toBe(true);
  });
});

describe('admin/packages/index · 选择医院 + 套餐列表', () => {
  beforeEach(() => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({
      items: [makeHospital(101)],
      total: 1,
    });
  });

  it('点击医院 → 调 listPackagesByHospital + 渲染套餐列表', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockResolvedValue({
      items: [makePackage(1), makePackage(2)],
    });
    const w = mount(AdminPackages);
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-packages-section"]').exists()).toBe(false);

    await w.find('[data-testid="admin-packages-hospital-101"]').trigger('click');
    await flushPromises();

    expect(apiUser.listPackagesByHospital).toHaveBeenCalledWith(101);
    expect(w.find('[data-testid="admin-packages-packages-section"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-packages-package-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-packages-package-2"]').exists()).toBe(true);
    expect(w.text()).toContain('套餐 1');
    expect(w.text()).toContain('全程陪诊');
    expect(w.text()).toContain('⏱ 120 分钟');
    expect(w.text()).toContain('¥299.00');
  });

  it('选中医院标记 active class', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockResolvedValue({ items: [] });
    const w = mount(AdminPackages);
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-hospital-101"]').classes()).not.toContain(
      'admin-packages__hospital--active',
    );

    await w.find('[data-testid="admin-packages-hospital-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-hospital-101"]').classes()).toContain(
      'admin-packages__hospital--active',
    );
  });

  it('套餐列表加载中显示 UiLoading', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockReturnValue(new Promise(() => {}));
    const w = mount(AdminPackages);
    await flushPromises();
    await w.find('[data-testid="admin-packages-hospital-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-packages-loading"]').exists()).toBe(true);
  });

  it('套餐列表为空显示 UiEmpty', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockResolvedValue({ items: [] });
    const w = mount(AdminPackages);
    await flushPromises();
    await w.find('[data-testid="admin-packages-hospital-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-packages-empty"]').exists()).toBe(true);
  });

  it('套餐 API 错误显示 UiEmpty「加载失败」', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockRejectedValue(new Error('网络超时'));
    const w = mount(AdminPackages);
    await flushPromises();
    await w.find('[data-testid="admin-packages-hospital-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="admin-packages-packages-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络超时');
  });
});