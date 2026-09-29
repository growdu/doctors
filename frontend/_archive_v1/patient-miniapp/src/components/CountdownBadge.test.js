// src/components/CountdownBadge.test.js
//
// CountdownBadge 单文件组件单测 —— @vue/test-utils mount + jest fake timers。
//
// 测试点（覆盖 8 个 it，超出 task 最低 5 个要求）：
//   1. 初始渲染：默认 label + 初始剩余秒数（基于 Date.now()）
//   2. 自定义 label：prop 替换默认文案
//   3. prop 变化更新：expireAt 变化触发剩余秒数立刻重算（不等下一秒 tick）
//   4. 倒计时 tick：jest.advanceTimersByTime 推进 N 秒，秒数对应递减
//   5. 颜色 ≥ 60s：蓝色 #1677ff
//   6. 颜色 < 60s：红色 #ff4d4f
//   7. 减到 0 停止：tick 到 0 后颜色保持红，再推进不再变负
//   8. unmount 清理：unmount 时 clearInterval 被调用（防内存泄漏）
//
// 依赖：
//   - jest 29（含 modern fake timers）       ← 已在 devDependencies
//   - @vue/test-utils v2（提供 mount / flushPromises）  ← 待 devDependencies 增补
//   - vue-jest                                ← 待 jest.config.js 增补 .vue transform
//
// 跑测前置步骤（任务外）：
//   npm i -D @vue/test-utils@^2.4 vue-jest@^4.0
//   // jest.config.js 增加 transform:
//   transform: {
//     '^.+\\.js$': 'babel-jest',
//     '^.+\\.vue$': 'vue-jest',
//   }

import { mount, flushPromises } from '@vue/test-utils';
import CountdownBadge from './CountdownBadge.vue';

// uni-app 自定义组件 <view> / <text> 在 jsdom 测试环境下未注册，
// stub 为 div / span，避免运行时 "Failed to resolve component" 警告。
const UNI_STUBS = {
  view: { template: '<div><slot /></div>' },
  text: { template: '<span><slot /></span>' },
};

const mountBadge = (props) =>
  mount(CountdownBadge, {
    props,
    global: { stubs: UNI_STUBS },
  });

describe('CountdownBadge.vue', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    // 固定系统时间：避免 CI 上 Date.now() 漂移导致秒数差 1
    jest.setSystemTime(new Date('2026-09-24T12:00:00Z'));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('初始渲染：默认 label + 初始剩余秒数', () => {
    const expireAt = new Date(Date.now() + 90_000).toISOString();
    const w = mountBadge({ expireAt });
    expect(w.text()).toContain('待确认剩余');
    expect(w.text()).toContain('剩余 90s');
  });

  it('自定义 label 替换默认文案', () => {
    const expireAt = new Date(Date.now() + 90_000).toISOString();
    const w = mountBadge({ expireAt, label: '订单剩余' });
    expect(w.text()).toContain('订单剩余');
    expect(w.text()).not.toContain('待确认剩余');
    expect(w.text()).toContain('剩余 90s');
  });

  it('expireAt prop 变化触发剩余秒数立刻重算', async () => {
    const w = mountBadge({
      expireAt: new Date(Date.now() + 90_000).toISOString(),
    });
    expect(w.text()).toContain('剩余 90s');

    await w.setProps({
      expireAt: new Date(Date.now() + 30_000).toISOString(),
    });
    await flushPromises();
    expect(w.text()).toContain('剩余 30s');
    expect(w.text()).not.toContain('剩余 90s');
  });

  it('每秒 tick 一次：剩余秒数对应递减', async () => {
    const w = mountBadge({
      expireAt: new Date(Date.now() + 90_000).toISOString(),
    });
    expect(w.text()).toContain('剩余 90s');

    jest.advanceTimersByTime(1000);
    await flushPromises();
    expect(w.text()).toContain('剩余 89s');

    jest.advanceTimersByTime(3000);
    await flushPromises();
    expect(w.text()).toContain('剩余 86s');
  });

  it('剩余 ≥ 60s 时颜色 #1677ff（蓝）', () => {
    const w = mountBadge({
      expireAt: new Date(Date.now() + 90_000).toISOString(),
    });
    const root = w.find('.countdown-badge');
    expect(root.exists()).toBe(true);
    const style = root.attributes('style') || '';
    expect(style).toMatch(/#1677ff/i);
    expect(style).not.toMatch(/#ff4d4f/i);
  });

  it('剩余 < 60s 时颜色 #ff4d4f（红）', () => {
    const w = mountBadge({
      expireAt: new Date(Date.now() + 30_000).toISOString(),
    });
    const root = w.find('.countdown-badge');
    expect(root.exists()).toBe(true);
    const style = root.attributes('style') || '';
    expect(style).toMatch(/#ff4d4f/i);
    expect(style).not.toMatch(/#1677ff/i);
  });

  it('减到 0 时停止 tick，颜色保持红色', async () => {
    const w = mountBadge({
      expireAt: new Date(Date.now() + 2000).toISOString(),
    });
    expect(w.text()).toContain('剩余 2s');

    jest.advanceTimersByTime(2000);
    await flushPromises();
    expect(w.text()).toContain('剩余 0s');

    // 再推进 5s，应仍为 0（已停止 tick，不会变负数）
    jest.advanceTimersByTime(5000);
    await flushPromises();
    expect(w.text()).toContain('剩余 0s');
    expect(w.text()).not.toMatch(/剩余 -/);

    const style = w.find('.countdown-badge').attributes('style') || '';
    expect(style).toMatch(/#ff4d4f/i);
  });

  it('unmount 时 clearInterval 被调用（防内存泄漏）', () => {
    const clearSpy = jest.spyOn(global, 'clearInterval');
    const w = mountBadge({
      expireAt: new Date(Date.now() + 90_000).toISOString(),
    });
    const before = clearSpy.mock.calls.length;
    w.unmount();
    expect(clearSpy.mock.calls.length).toBeGreaterThan(before);
    clearSpy.mockRestore();
  });
});