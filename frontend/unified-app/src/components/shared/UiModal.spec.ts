/**
 * UiModal 组件单测。
 *
 * 验证目标：
 *   - 默认不可见（visible=false）
 *   - visible=true 时显示
 *   - v-model:visible 双向绑定
 *   - mask 点击关闭（默认 closeOnMask=true）
 *   - closeOnMask=false 时 mask 不关闭
 *   - confirm / cancel 按钮 emit + 默认关闭
 *   - title / content prop + slot
 *   - confirmType 切换按钮样式
 */
import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import UiModal from './UiModal.vue';

describe('UiModal · visibility', () => {
  it('not rendered when visible is false', () => {
    const w = mount(UiModal);
    expect(w.find('[data-testid="ui-modal"]').exists()).toBe(false);
  });

  it('renders when visible is true', () => {
    const w = mount(UiModal, { props: { visible: true } });
    expect(w.find('[data-testid="ui-modal"]').exists()).toBe(true);
    expect(w.find('[data-testid="ui-modal-panel"]').exists()).toBe(true);
  });

  it('toggles visibility on prop change', async () => {
    const w = mount(UiModal, { props: { visible: false } });
    expect(w.find('[data-testid="ui-modal"]').exists()).toBe(false);
    await w.setProps({ visible: true });
    expect(w.find('[data-testid="ui-modal"]').exists()).toBe(true);
  });
});

describe('UiModal · mask close', () => {
  it('emits update:visible false + close on mask click', async () => {
    const w = mount(UiModal, { props: { visible: true } });
    await w.find('[data-testid="ui-modal-mask"]').trigger('click');
    expect(w.emitted('update:visible')![0]).toEqual([false]);
    expect(w.emitted('close')).toBeTruthy();
  });

  it('does not close when closeOnMask is false', async () => {
    const w = mount(UiModal, { props: { visible: true, closeOnMask: false } });
    await w.find('[data-testid="ui-modal-mask"]').trigger('click');
    expect(w.emitted('update:visible')).toBeUndefined();
    expect(w.emitted('close')).toBeUndefined();
  });

  it('panel click does not propagate to mask', async () => {
    const w = mount(UiModal, { props: { visible: true } });
    await w.find('[data-testid="ui-modal-panel"]').trigger('click');
    expect(w.emitted('update:visible')).toBeUndefined();
  });
});

describe('UiModal · buttons', () => {
  it('confirm emits confirm + close', async () => {
    const w = mount(UiModal, { props: { visible: true } });
    await w.find('[data-testid="ui-modal-confirm"]').trigger('click');
    expect(w.emitted('confirm')).toBeTruthy();
    expect(w.emitted('update:visible')![0]).toEqual([false]);
  });

  it('cancel emits cancel + close', async () => {
    const w = mount(UiModal, { props: { visible: true } });
    await w.find('[data-testid="ui-modal-cancel"]').trigger('click');
    expect(w.emitted('cancel')).toBeTruthy();
    expect(w.emitted('update:visible')![0]).toEqual([false]);
  });

  it('custom button labels', () => {
    const w = mount(UiModal, {
      props: { visible: true, confirmText: '提交', cancelText: '返回' },
    });
    expect(w.find('[data-testid="ui-modal-confirm"]').text()).toBe('提交');
    expect(w.find('[data-testid="ui-modal-cancel"]').text()).toBe('返回');
  });

  it('confirmType=danger applies danger class', () => {
    const w = mount(UiModal, {
      props: { visible: true, confirmType: 'danger' },
    });
    expect(w.find('[data-testid="ui-modal-confirm"]').classes()).toContain('ui-modal__btn--danger');
  });
});

describe('UiModal · title / content', () => {
  it('renders title prop', () => {
    const w = mount(UiModal, { props: { visible: true, title: '提示' } });
    expect(w.find('[data-testid="ui-modal-title"]').text()).toBe('提示');
  });

  it('renders title slot over prop', () => {
    const w = mount(UiModal, {
      props: { visible: true, title: 'fallback' },
      slots: { title: '<span data-testid="custom-title">自定义标题</span>' },
    });
    expect(w.find('[data-testid="custom-title"]').exists()).toBe(true);
  });

  it('renders content prop', () => {
    const w = mount(UiModal, { props: { visible: true, content: '确认删除？' } });
    expect(w.find('[data-testid="ui-modal-content"]').text()).toBe('确认删除？');
  });

  it('renders default slot', () => {
    const w = mount(UiModal, {
      props: { visible: true },
      slots: { default: '<p data-testid="custom-content">自定义内容</p>' },
    });
    expect(w.find('[data-testid="custom-content"]').exists()).toBe(true);
  });

  it('does not render title element when no title prop and no slot', () => {
    const w = mount(UiModal, { props: { visible: true } });
    expect(w.find('[data-testid="ui-modal-title"]').exists()).toBe(false);
  });
});