/**
 * UiCard 组件单测。
 *
 * 验证目标：
 *   - title 渲染到 header
 *   - extra slot 渲染到 header 右侧
 *   - body slot 渲染到中部
 *   - footer slot 渲染到底部
 *   - shadow 等级 class 切换
 *   - noPadding 模式去掉 body 内边距
 *   - 无 title 时不渲染 header
 */
import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import UiCard from './UiCard.vue';

describe('UiCard · slots', () => {
  it('renders body slot as default', () => {
    const w = mount(UiCard, { slots: { default: '<p data-testid="body-content">hello</p>' } });
    expect(w.find('[data-testid="body-content"]').exists()).toBe(true);
  });

  it('renders title when provided', () => {
    const w = mount(UiCard, { props: { title: '订单详情' } });
    expect(w.find('[data-testid="ui-card-title"]').text()).toBe('订单详情');
    expect(w.find('[data-testid="ui-card-header"]').exists()).toBe(true);
  });

  it('renders extra slot when provided', () => {
    const w = mount(UiCard, {
      props: { title: '订单' },
      slots: { extra: '<a data-testid="extra-link">查看全部</a>' },
    });
    expect(w.find('[data-testid="extra-link"]').exists()).toBe(true);
  });

  it('renders extra slot without title', () => {
    const w = mount(UiCard, {
      slots: { extra: '<a data-testid="extra-only">filter</a>' },
    });
    expect(w.find('[data-testid="extra-only"]').exists()).toBe(true);
    expect(w.find('[data-testid="ui-card-header"]').exists()).toBe(true);
  });

  it('does not render header when no title and no extra', () => {
    const w = mount(UiCard);
    expect(w.find('[data-testid="ui-card-header"]').exists()).toBe(false);
  });

  it('renders footer slot', () => {
    const w = mount(UiCard, {
      slots: { footer: '<button data-testid="footer-btn">确认</button>' },
    });
    expect(w.find('[data-testid="footer-btn"]').exists()).toBe(true);
  });

  it('does not render footer when slot empty', () => {
    const w = mount(UiCard);
    expect(w.find('[data-testid="ui-card-footer"]').exists()).toBe(false);
  });
});

describe('UiCard · shadow variants', () => {
  it('default shadow is sm', () => {
    const w = mount(UiCard);
    expect(w.classes()).toContain('ui-card--shadow-sm');
  });

  it('applies shadow class variants', () => {
    for (const s of ['sm', 'md', 'lg', 'none'] as const) {
      const w = mount(UiCard, { props: { shadow: s } });
      expect(w.classes()).toContain(`ui-card--shadow-${s}`);
    }
  });
});

describe('UiCard · noPadding', () => {
  it('applies no-padding class', () => {
    const w = mount(UiCard, { props: { noPadding: true } });
    expect(w.classes()).toContain('ui-card--no-padding');
  });

  it('does not apply no-padding class by default', () => {
    const w = mount(UiCard);
    expect(w.classes()).not.toContain('ui-card--no-padding');
  });
});