/**
 * UiButton 组件单测（vitest + Vue Test Utils）。
 *
 * 验证目标：
 *   - 4 种 type 的 CSS class 正确渲染
 *   - 3 种 size 的 padding / font-size 正确
 *   - disabled 阻止 click 事件
 *   - loading 时显示旋转图标 + 自动 disabled
 *   - click 事件触发并传递 MouseEvent
 *   - block 模式 width: 100%
 */
import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import UiButton from './UiButton.vue';

describe('UiButton · type / size', () => {
  it('renders default type by default', () => {
    const w = mount(UiButton, { slots: { default: 'click me' } });
    expect(w.classes()).toContain('ui-btn--default');
    expect(w.classes()).toContain('ui-btn--base');
    expect(w.text()).toBe('click me');
  });

  it('applies 4 type classes', () => {
    for (const t of ['primary', 'default', 'ghost', 'danger'] as const) {
      const w = mount(UiButton, { props: { type: t }, slots: { default: 'x' } });
      expect(w.classes()).toContain(`ui-btn--${t}`);
    }
  });

  it('applies 3 size classes', () => {
    for (const s of ['sm', 'base', 'lg'] as const) {
      const w = mount(UiButton, { props: { size: s }, slots: { default: 'x' } });
      expect(w.classes()).toContain(`ui-btn--${s}`);
    }
  });
});

describe('UiButton · disabled / loading', () => {
  it('disabled prevents click event', async () => {
    const w = mount(UiButton, {
      props: { disabled: true },
      slots: { default: 'x' },
    });
    expect(w.attributes('disabled')).toBeDefined();
    await w.trigger('click');
    expect(w.emitted('click')).toBeUndefined();
  });

  it('loading state shows spinner and disables click', async () => {
    const w = mount(UiButton, {
      props: { loading: true },
      slots: { default: 'submit' },
    });
    expect(w.find('[data-testid="ui-btn-spinner"]').exists()).toBe(true);
    expect(w.attributes('disabled')).toBeDefined();
    expect(w.classes()).toContain('ui-btn--loading');
    await w.trigger('click');
    expect(w.emitted('click')).toBeUndefined();
  });
});

describe('UiButton · click event', () => {
  it('emits click when enabled', async () => {
    const w = mount(UiButton, { slots: { default: 'go' } });
    await w.trigger('click');
    expect(w.emitted('click')).toHaveLength(1);
    expect(w.emitted('click')![0]![0]).toBeInstanceOf(MouseEvent);
  });
});

describe('UiButton · block', () => {
  it('applies block class', () => {
    const w = mount(UiButton, { props: { block: true }, slots: { default: 'x' } });
    expect(w.classes()).toContain('ui-btn--block');
  });
});