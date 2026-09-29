/**
 * UiLoading 组件单测。
 *
 * 验证目标：
 *   - 3 种 size class 正确渲染
 *   - text prop 渲染文本
 *   - fullscreen 模式应用 fullscreen class
 */
import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import UiLoading from './UiLoading.vue';

describe('UiLoading · size', () => {
  it('applies size class to spinner', () => {
    for (const s of ['sm', 'base', 'lg'] as const) {
      const w = mount(UiLoading, { props: { size: s } });
      expect(w.find('[data-testid="ui-loading-spinner"]').classes()).toContain(`ui-loading__spinner--${s}`);
    }
  });

  it('default size is base', () => {
    const w = mount(UiLoading);
    expect(w.find('[data-testid="ui-loading-spinner"]').classes()).toContain('ui-loading__spinner--base');
  });
});

describe('UiLoading · text', () => {
  it('does not render text element when text is empty', () => {
    const w = mount(UiLoading);
    expect(w.find('[data-testid="ui-loading-text"]').exists()).toBe(false);
  });

  it('renders text when provided', () => {
    const w = mount(UiLoading, { props: { text: '加载中...' } });
    expect(w.find('[data-testid="ui-loading-text"]').text()).toBe('加载中...');
  });
});

describe('UiLoading · fullscreen', () => {
  it('does not apply fullscreen class by default', () => {
    const w = mount(UiLoading);
    expect(w.classes()).not.toContain('ui-loading--fullscreen');
  });

  it('applies fullscreen class when prop true', () => {
    const w = mount(UiLoading, { props: { fullscreen: true } });
    expect(w.classes()).toContain('ui-loading--fullscreen');
  });
});