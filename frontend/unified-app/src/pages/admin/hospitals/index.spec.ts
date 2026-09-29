/**
 * admin/hospitals/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listHospitals
 *   - 渲染医院列表卡（名称 / 城市 / 等级 / 地址 / 电话）
 *   - 关键词搜索（按名称 / 城市过滤）
 *   - 点击医院卡展开套餐列表（listPackagesByHospital）
 *   - 再次点击收起
 *   - 套餐列表渲染（标题 / 描述 / 时长 / 价格 分→元）
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AdminHospitals from './index.vue';

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
  address: '北京市朝阳区某街道',
  phone: `010-${10000000 + id}`,
  level: '三级甲等',
  ...overrides,
});

const makePackage = (id: number, hospitalId: number, overrides: Partial<Package> = {}): Package => ({
  id,
  hospital_id: hospitalId,
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

describe('admin/hospitals/index · 加载', () => {
  it('mount 时调 listHospitals', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({ items: [], total: 0 });
    mount(AdminHospitals);
    await flushPromises();
    expect(apiUser.listHospitals).toHaveBeenCalledTimes(1);
  });
});

describe('admin/hospitals/index · 列表渲染', () => {
  it('渲染医院卡（名称 / 城市 / 等级 / 地址 / 电话）', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({
      items: [
        makeHospital(101),
        makeHospital(102, { name: '协和医院', city: '上海', level: '三级特等', phone: null }),
      ],
      total: 2,
    });
    const w = mount(AdminHospitals);
    await flushPromises();

    expect(w.find('[data-testid="admin-hospitals-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-hospitals-card-102"]').exists()).toBe(true);
    expect(w.text()).toContain('医院101');
    expect(w.text()).toContain('协和医院');
    expect(w.text()).toContain('北京 · 三级甲等');
    expect(w.text()).toContain('上海 · 三级特等');
    expect(w.text()).toContain('010-10000101');
  });
});

describe('admin/hospitals/index · 关键词搜索', () => {
  it('输入关键词过滤列表', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({
      items: [
        makeHospital(101, { name: '协和医院', city: '北京' }),
        makeHospital(102, { name: '同仁医院', city: '上海' }),
      ],
      total: 2,
    });
    const w = mount(AdminHospitals);
    await flushPromises();

    expect(w.find('[data-testid="admin-hospitals-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-hospitals-card-102"]').exists()).toBe(true);

    // 设置关键词
    const input = w.find('[data-testid="ui-input-inner"]');
    await input.setValue('上海');
    await flushPromises();

    expect(w.find('[data-testid="admin-hospitals-card-101"]').exists()).toBe(false);
    expect(w.find('[data-testid="admin-hospitals-card-102"]').exists()).toBe(true);
  });

  it('无匹配显示空态', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({
      items: [makeHospital(101, { name: '协和医院' })],
      total: 1,
    });
    const w = mount(AdminHospitals);
    await flushPromises();

    const input = w.find('[data-testid="ui-input-inner"]');
    await input.setValue('不存在的医院');
    await flushPromises();

    expect(w.find('[data-testid="admin-hospitals-empty"]').exists()).toBe(true);
  });
});

describe('admin/hospitals/index · 套餐展开', () => {
  beforeEach(() => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({
      items: [makeHospital(101)],
      total: 1,
    });
  });

  it('点击医院卡展开套餐列表', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockResolvedValue({
      items: [makePackage(1, 101), makePackage(2, 101)],
    });
    const w = mount(AdminHospitals);
    await flushPromises();

    expect(w.find('[data-testid="admin-hospitals-packages-101"]').exists()).toBe(false);

    await w.find('[data-testid="admin-hospitals-toggle-101"]').trigger('click');
    await flushPromises();

    expect(apiUser.listPackagesByHospital).toHaveBeenCalledWith(101);
    expect(w.find('[data-testid="admin-hospitals-packages-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-hospitals-package-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-hospitals-package-2"]').exists()).toBe(true);
    expect(w.text()).toContain('套餐 1');
    expect(w.text()).toContain('全程陪诊');
    expect(w.text()).toContain('⏱ 120 分钟');
    expect(w.text()).toContain('¥299.00');
  });

  it('再次点击收起套餐列表', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockResolvedValue({ items: [] });
    const w = mount(AdminHospitals);
    await flushPromises();

    await w.find('[data-testid="admin-hospitals-toggle-101"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-hospitals-packages-101"]').exists()).toBe(true);

    await w.find('[data-testid="admin-hospitals-toggle-101"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="admin-hospitals-packages-101"]').exists()).toBe(false);
  });

  it('套餐为空显示 UiEmpty', async () => {
    vi.mocked(apiUser.listPackagesByHospital).mockResolvedValue({ items: [] });
    const w = mount(AdminHospitals);
    await flushPromises();
    await w.find('[data-testid="admin-hospitals-toggle-101"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="admin-hospitals-packages-empty"]').exists()).toBe(true);
  });
});

describe('admin/hospitals/index · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiUser.listHospitals).mockReturnValue(new Promise(() => {}));
    const w = mount(AdminHospitals);
    await flushPromises();
    expect(w.find('[data-testid="admin-hospitals-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiUser.listHospitals).mockResolvedValue({ items: [], total: 0 });
    const w = mount(AdminHospitals);
    await flushPromises();
    expect(w.find('[data-testid="admin-hospitals-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂无医院');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiUser.listHospitals).mockRejectedValue(new Error('网络异常'));
    const w = mount(AdminHospitals);
    await flushPromises();
    expect(w.find('[data-testid="admin-hospitals-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});