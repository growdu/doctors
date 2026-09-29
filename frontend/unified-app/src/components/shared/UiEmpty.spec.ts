/**
 * UiEmpty 组件单测。
 *
 * 验证目标：
 *   - 默认 props 正确（icon / title）
 *   - 自定义 icon / title / description 渲染
 *   - action slot 正确渲染
 */
import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import UiEmpty from './UiEmpty.vue';

describe('UiEmpty · default props', () => {
  it('renders default icon and title', () => {
    const w = mount(UiEmpty);
    expect(w.find('[data-testid="ui-empty-icon"]').text()).toBe('📭');
    expect(w.find('[data-testid="ui-empty-title"]').text()).toBe('暂无数据');
    expect(w.find('[data-testid="ui-empty-description"]').exists()).toBe(false);
  });
});

describe('UiEmpty · custom props', () => {
  it('renders custom icon and title', () => {
    const w = mount(UiEmpty, { props: { icon: '🔍', title: '未找到结果' } });
    expect(w.find('[data-testid="ui-empty-icon"]').text()).toBe('🔍');
    expect(w.find('[data-testid="ui-empty-title"]').text()).toBe('未找到结果');
  });

  it('renders description when provided', () => {
    const w = mount(UiEmpty, { props: { description: '换个关键词试试' } });
    expect(w.find('[data-testid="ui-empty-description"]').text()).toBe('换个关键词试试');
  });
});

describe('UiEmpty · action slot', () => {
  it('renders action slot content', () => {
    const w = mount(UiEmpty, {
      slots: { action: '<button data-testid="retry">重试</button>' },
    });
    expect(w.find('[data-testid="retry"]').exists()).toBe(true);
  });

  it('does not render action wrapper when slot empty', () => {
    const w = mount(UiEmpty);
    expect(w.find('.ui-empty__action').exists()).toBe(false);
  });
});