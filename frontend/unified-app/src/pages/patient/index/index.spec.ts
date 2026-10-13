/**
 * patient/index/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 渲染欢迎卡 + 快捷入口 4 + 推荐医院
 *   - 「进行中订单」widget：登录 + 有 active 订单时显示
 *   - 「进行中订单」widget：未登录时不显示
 *   - 「进行中订单」widget：加载失败时不抛错
 *   - 点击订单跳详情；点击医院跳详情
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import * as userApi from '@/api/user';
import type { Hospital, Package } from '@/api/user';
import * as ordersApi from '@/api/orders';
import PatientHome from './index.vue';

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

const HOSPITALS: { items: Hospital[]; total: number } = {
  items: [
    { id: 1, name: '协和医院', city: '北京', level: '三级甲等', address: '东城区帅府园1号', phone: '010-12345' },
    { id: 2, name: '301 医院', city: '北京', level: '三级甲等', address: '海淀区复兴路28号', phone: '010-54321' },
  ],
  total: 2,
};

const PACKAGES: { items: Package[] } = {
  items: [
    { id: 100, hospital_id: 1, title: '半日陪诊', description: '陪诊半天', price: 29900, duration_minutes: 240, cover_url: null },
  ],
};

describe('patient/index/index · 渲染', () => {
  it('渲染欢迎卡 + 4 快捷入口', async () => {
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({ items: [], total: 0 });

    const w = mount(PatientHome);
    await flushPromises();

    expect(w.find('[data-testid="patient-home-welcome"]').exists()).toBe(true);
    expect(w.text()).toContain('您好');
    expect(w.find('[data-testid="home-quick-选择医院"]').exists()).toBe(true);
    expect(w.find('[data-testid="home-quick-我的订单"]').exists()).toBe(true);
    expect(w.find('[data-testid="home-quick-优惠券"]').exists()).toBe(true);
    expect(w.find('[data-testid="home-quick-消息"]').exists()).toBe(true);
  });

  it('渲染推荐医院列表', async () => {
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({ items: [], total: 0 });

    const w = mount(PatientHome);
    await flushPromises();

    expect(w.find('[data-testid="home-hospital-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="home-hospital-2"]').exists()).toBe(true);
    expect(w.text()).toContain('协和医院');
    expect(w.text()).toContain('北京');
  });
});

/** 已登录态：isAuthed 需要 token && user 都非空 */
function loginAsPatient(auth: ReturnType<typeof useAuthStore>): void {
  auth.token = 'test-token';
  auth.user = {
    id: 1,
    phone: '13800138000',
    role: 'patient',
    active_role: 'patient',
    roles: ['patient'],
    real_name_verified: true,
  };
}

describe('patient/index/index · 进行中订单 widget', () => {
  beforeEach(() => {
    loginAsPatient(useAuthStore());
  });

  it('有 active 订单时显示 widget + 订单卡', async () => {
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({
      items: [
        {
          id: 100,
          order_no: 'P202610130001',
          patient_id: 1,
          escort_id: null,
          hospital_id: 1,
          package_id: 100,
          service_start_at: '2026-10-15T10:00:00Z',
          status: 'pending_escort',
          amount: 29900,
          final_amount: 29900,
          selected_escort_id: null,
          version: 1,
        },
      ],
      total: 1,
    });

    const w = mount(PatientHome);
    await flushPromises();

    expect(w.find('[data-testid="home-active-orders"]').exists()).toBe(true);
    expect(w.find('[data-testid="home-active-order-100"]').exists()).toBe(true);
    expect(w.text()).toContain('进行中订单');
    expect(w.text()).toContain('待接单');
  });

  it('completed/cancelled 订单不在 widget 显示', async () => {
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({
      items: [
        {
          id: 200,
          order_no: 'P202610130002',
          patient_id: 1,
          escort_id: 1,
          hospital_id: 1,
          package_id: 100,
          service_start_at: '2026-09-01T10:00:00Z',
          status: 'completed',
          amount: 29900,
          final_amount: 29900,
          selected_escort_id: 1,
          version: 1,
        },
      ],
      total: 1,
    });

    const w = mount(PatientHome);
    await flushPromises();

    // 完成订单不算 active，整个 widget 不应显示
    expect(w.find('[data-testid="home-active-orders"]').exists()).toBe(false);
  });

  it('订单加载失败不阻塞页面渲染', async () => {
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockRejectedValue(new Error('network'));

    const w = mount(PatientHome);
    await flushPromises();

    // 推荐医院区块仍渲染
    expect(w.find('[data-testid="home-hospital-1"]').exists()).toBe(true);
    // widget 不显示（无 active 订单）
    expect(w.find('[data-testid="home-active-orders"]').exists()).toBe(false);
  });
});

describe('patient/index/index · 未登录态', () => {
  it('未登录时不显示「进行中订单」widget（不调用 listOrders）', async () => {
    const listOrdersSpy = vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({ items: [], total: 0 });
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);

    const w = mount(PatientHome);
    await flushPromises();

    expect(w.find('[data-testid="home-active-orders"]').exists()).toBe(false);
    expect(listOrdersSpy).not.toHaveBeenCalled();
  });
});

describe('patient/index/index · 跳转', () => {
  beforeEach(() => {
    loginAsPatient(useAuthStore());
  });

  it('点击医院跳详情', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({ items: [], total: 0 });

    const w = mount(PatientHome);
    await flushPromises();

    await w.find('[data-testid="home-hospital-1"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/hospitals/detail?id=1' });
  });

  it('点击 active 订单跳详情', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({
      items: [
        {
          id: 100,
          order_no: 'P202610130001',
          patient_id: 1,
          escort_id: null,
          hospital_id: 1,
          package_id: 100,
          service_start_at: '2026-10-15T10:00:00Z',
          status: 'escort_confirmed',
          amount: 29900,
          final_amount: 29900,
          selected_escort_id: null,
          version: 1,
        },
      ],
      total: 1,
    });

    const w = mount(PatientHome);
    await flushPromises();

    await w.find('[data-testid="home-active-order-100"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/order/detail?id=100' });
  });

  it('点击「查看全部」跳订单列表', async () => {
    const navSpy = vi.fn();
    (globalThis as unknown as { uni: { navigateTo: typeof navSpy } }).uni.navigateTo = navSpy;
    vi.spyOn(userApi, 'listHospitals').mockResolvedValue(HOSPITALS);
    vi.spyOn(userApi, 'listPackagesByHospital').mockResolvedValue(PACKAGES);
    vi.spyOn(ordersApi, 'listOrders').mockResolvedValue({
      items: [
        {
          id: 100,
          order_no: 'P202610130001',
          patient_id: 1,
          escort_id: null,
          hospital_id: 1,
          package_id: 100,
          service_start_at: '2026-10-15T10:00:00Z',
          status: 'pending_escort',
          amount: 29900,
          final_amount: 29900,
          selected_escort_id: null,
          version: 1,
        },
      ],
      total: 1,
    });

    const w = mount(PatientHome);
    await flushPromises();

    await w.find('[data-testid="home-view-all-orders"]').trigger('click');
    expect(navSpy).toHaveBeenCalledWith({ url: '/pages/patient/order/list' });
  });
});