/**
 * admin/sos/index.vue 组件单测（vitest + Vue Test Utils）。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import * as sosApi from '@/api/sos';
import type { Sos } from '@/api/sos';
import AdminSos from './index.vue';

const sos = (id: number, status: Sos['status']): Sos => ({
  id,
  trigger_user_id: 100,
  order_id: 200,
  reason: `紧急求助 #${id}`,
  location: { lat: 39.9, lng: 116.4, address: '北京协和医院' },
  contact_phone_masked: '138****0000',
  status,
  resolved_at: null,
  created_at: '2026-10-01T10:00:00Z',
});

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('admin/sos/index · 渲染', () => {
  it('渲染标题', () => {
    vi.spyOn(sosApi, 'listSos').mockResolvedValue({ items: [], total: 0 });

    const w = mount(AdminSos);
    expect(w.text()).toContain('SOS 管理');
  });

  it('加载成功渲染 4 状态卡 + 列表', async () => {
    vi.spyOn(sosApi, 'listSos').mockResolvedValue({
      items: [sos(1, 'pending'), sos(2, 'in_progress'), sos(3, 'resolved')],
      total: 3,
    });

    const w = mount(AdminSos);
    await flushPromises();

    expect(w.find('[data-testid="admin-sos-stat-pending"]').text()).toContain('1');
    expect(w.find('[data-testid="admin-sos-stat-in_progress"]').text()).toContain('1');
    expect(w.find('[data-testid="admin-sos-stat-resolved"]').text()).toContain('1');
    expect(w.find('[data-testid="admin-sos-stat-closed"]').text()).toContain('0');

    expect(w.find('[data-testid="admin-sos-1"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-sos-status-1"]').text()).toContain('待处理');
    expect(w.text()).toContain('紧急求助 #1');
    expect(w.text()).toContain('138****0000');
  });
});

describe('admin/sos/index · 失败回退', () => {
  it('加载失败渲染空态 + 重试按钮', async () => {
    vi.spyOn(sosApi, 'listSos').mockRejectedValue(new Error('network'));

    const w = mount(AdminSos);
    await flushPromises();

    expect(w.find('[data-testid="admin-sos-error"]').exists()).toBe(true);
    expect(w.find('[data-testid="admin-sos-retry"]').exists()).toBe(true);
  });

  it('点击重试重新调 onLoad', async () => {
    const listSpy = vi.spyOn(sosApi, 'listSos')
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({ items: [], total: 0 });

    const w = mount(AdminSos);
    await flushPromises();

    await w.find('[data-testid="admin-sos-retry"]').trigger('click');
    await flushPromises();

    expect(listSpy).toHaveBeenCalledTimes(2);
  });
});