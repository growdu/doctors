/**
 * escort/availability/index.vue 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - mount 时调 listMyAvailabilities
 *   - 渲染时段卡（开始 / 结束时间 / 备注）
 *   - 点击「新增时段」打开 modal → 输入合法时间 + 确认 → insert 新项
 *   - 校验：end_at 必须晚于 start_at
 *   - 点击「删除」调 removeAvailability + 从列表移除
 *   - 空 / 加载 / 错误 三态
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AvailabilityPage from './index.vue';

vi.mock('@/api/escort', () => ({
  listMyAvailabilities: vi.fn(),
  addAvailability: vi.fn(),
  removeAvailability: vi.fn(),
}));

import * as apiEscort from '@/api/escort';
import type { Availability } from '@/api/escort';

const makeAvail = (id: number, overrides: Partial<Availability> = {}): Availability => ({
  id,
  escort_id: 2001,
  start_at: '2026-10-01T09:00:00Z',
  end_at: '2026-10-01T17:00:00Z',
  remark: '周末优先',
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('escort/availability · 加载', () => {
  it('mount 时调 listMyAvailabilities', async () => {
    vi.mocked(apiEscort.listMyAvailabilities).mockResolvedValue({ items: [] });
    mount(AvailabilityPage);
    await flushPromises();
    expect(apiEscort.listMyAvailabilities).toHaveBeenCalledTimes(1);
  });

  it('渲染时段卡（开始 / 结束 / 备注）', async () => {
    vi.mocked(apiEscort.listMyAvailabilities).mockResolvedValue({
      items: [makeAvail(101), makeAvail(102, { remark: '' })],
    });
    const w = mount(AvailabilityPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-availability-card-101"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-availability-card-102"]').exists()).toBe(true);

    expect(w.find('[data-testid="escort-availability-time-101"]').text()).toContain('2026-10-01 09:00');
    expect(w.find('[data-testid="escort-availability-time-101"]').text()).toContain('2026-10-01 17:00');
    expect(w.text()).toContain('周末优先');
  });
});

describe('escort/availability · 新增时段', () => {
  beforeEach(() => {
    vi.mocked(apiEscort.listMyAvailabilities).mockResolvedValue({ items: [] });
  });

  it('点击「新增时段」打开 modal', async () => {
    const w = mount(AvailabilityPage);
    await flushPromises();

    expect(w.find('[data-testid="escort-availability-create-modal"]').exists()).toBe(false);
    await w.find('[data-testid="escort-availability-create-btn"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-availability-create-modal"]').exists()).toBe(true);
  });

  it('end_at ≤ start_at 时确认后显示错误', async () => {
    const w = mount(AvailabilityPage);
    await flushPromises();
    await w.find('[data-testid="escort-availability-create-btn"]').trigger('click');
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('2026-10-01T17:00:00Z');
    await inputs[1]!.setValue('2026-10-01T09:00:00Z');
    await flushPromises();

    await w.find('[data-testid="escort-availability-create-confirm"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-availability-create-error"]').exists()).toBe(true);
    expect(w.text()).toContain('结束时间必须晚于开始时间');
    expect(apiEscort.addAvailability).not.toHaveBeenCalled();
  });

  it('合法时间 → 调 addAvailability + 列表追加', async () => {
    vi.mocked(apiEscort.addAvailability).mockResolvedValue(
      makeAvail(201, { start_at: '2026-10-02T09:00:00Z', end_at: '2026-10-02T17:00:00Z' }),
    );

    const w = mount(AvailabilityPage);
    await flushPromises();
    await w.find('[data-testid="escort-availability-create-btn"]').trigger('click');
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('2026-10-02T09:00:00Z');
    await inputs[1]!.setValue('2026-10-02T17:00:00Z');
    await inputs[2]!.setValue('国庆假期');
    await flushPromises();

    await w.find('[data-testid="escort-availability-create-confirm"]').trigger('click');
    await flushPromises();

    expect(apiEscort.addAvailability).toHaveBeenCalledWith({
      start_at: '2026-10-02T09:00:00Z',
      end_at: '2026-10-02T17:00:00Z',
      remark: '国庆假期',
    });
    expect(w.find('[data-testid="escort-availability-card-201"]').exists()).toBe(true);
  });
});

describe('escort/availability · 删除时段', () => {
  it('点击「删除」调 removeAvailability + 从列表移除', async () => {
    vi.mocked(apiEscort.listMyAvailabilities).mockResolvedValue({
      items: [makeAvail(101), makeAvail(102)],
    });
    vi.mocked(apiEscort.removeAvailability).mockResolvedValue({ ok: true });

    const w = mount(AvailabilityPage);
    await flushPromises();

    await w.find('[data-testid="escort-availability-remove-101"]').trigger('click');
    await flushPromises();

    expect(apiEscort.removeAvailability).toHaveBeenCalledWith(101);
    expect(w.find('[data-testid="escort-availability-card-101"]').exists()).toBe(false);
    expect(w.find('[data-testid="escort-availability-card-102"]').exists()).toBe(true);
  });
});

describe('escort/availability · 三态', () => {
  it('加载中显示 UiLoading', async () => {
    vi.mocked(apiEscort.listMyAvailabilities).mockReturnValue(new Promise(() => {}));
    const w = mount(AvailabilityPage);
    await flushPromises();
    expect(w.find('[data-testid="escort-availability-loading"]').exists()).toBe(true);
  });

  it('空数据显示 UiEmpty', async () => {
    vi.mocked(apiEscort.listMyAvailabilities).mockResolvedValue({ items: [] });
    const w = mount(AvailabilityPage);
    await flushPromises();
    expect(w.find('[data-testid="escort-availability-empty"]').exists()).toBe(true);
    expect(w.text()).toContain('暂未设置时段');
  });

  it('API 错误显示 UiEmpty「加载失败」+ 重试', async () => {
    vi.mocked(apiEscort.listMyAvailabilities).mockRejectedValue(new Error('网络异常'));
    const w = mount(AvailabilityPage);
    await flushPromises();
    expect(w.find('[data-testid="escort-availability-error"]').exists()).toBe(true);
    expect(w.text()).toContain('网络异常');
  });
});