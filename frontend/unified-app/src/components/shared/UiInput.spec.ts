/**
 * UiInput 组件单测。
 *
 * 验证目标：
 *   - v-model 双向绑定（update:modelValue emit）
 *   - input / textarea 渲染根据 type 切换
 *   - placeholder / maxlength / disabled 属性传递
 *   - clearable 模式显示 + 点击清空
 *   - error 状态显示错误文本 + 红色边框 class
 *   - label slot 与 label prop 互斥
 *   - disabled / readonly 状态
 */
import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import UiInput from './UiInput.vue';

describe('UiInput · type / element', () => {
  it('renders input by default (type=text)', () => {
    const w = mount(UiInput);
    expect(w.find('[data-testid="ui-input-inner"]').exists()).toBe(true);
    expect(w.find('[data-testid="ui-input-textarea"]').exists()).toBe(false);
  });

  it('renders textarea when type=textarea', () => {
    const w = mount(UiInput, { props: { type: 'textarea' } });
    expect(w.find('[data-testid="ui-input-textarea"]').exists()).toBe(true);
    expect(w.find('[data-testid="ui-input-inner"]').exists()).toBe(false);
  });

  it('passes input attributes', () => {
    const w = mount(UiInput, {
      props: { placeholder: '请输入手机号', maxlength: 11, type: 'tel' },
    });
    const input = w.find('[data-testid="ui-input-inner"]');
    expect(input.attributes('placeholder')).toBe('请输入手机号');
    expect(input.attributes('maxlength')).toBe('11');
    expect(input.attributes('type')).toBe('tel');
  });
});

describe('UiInput · v-model', () => {
  it('renders modelValue', () => {
    const w = mount(UiInput, { props: { modelValue: 'hello' } });
    expect((w.find('[data-testid="ui-input-inner"]').element as HTMLInputElement).value).toBe('hello');
  });

  it('emits update:modelValue on input', async () => {
    const w = mount(UiInput, { props: { modelValue: '' } });
    const input = w.find('[data-testid="ui-input-inner"]');
    await input.setValue('new value');
    expect(w.emitted('update:modelValue')).toBeTruthy();
    expect(w.emitted('update:modelValue')![0]).toEqual(['new value']);
  });
});

describe('UiInput · clearable', () => {
  it('does not show clear button by default', () => {
    const w = mount(UiInput, { props: { modelValue: 'abc' } });
    expect(w.find('[data-testid="ui-input-clear"]').exists()).toBe(false);
  });

  it('shows clear button when clearable and has value', () => {
    const w = mount(UiInput, { props: { modelValue: 'abc', clearable: true } });
    expect(w.find('[data-testid="ui-input-clear"]').exists()).toBe(true);
  });

  it('does not show clear button when empty even with clearable', () => {
    const w = mount(UiInput, { props: { modelValue: '', clearable: true } });
    expect(w.find('[data-testid="ui-input-clear"]').exists()).toBe(false);
  });

  it('clicking clear emits empty value', async () => {
    const w = mount(UiInput, { props: { modelValue: 'abc', clearable: true } });
    await w.find('[data-testid="ui-input-clear"]').trigger('click');
    expect(w.emitted('update:modelValue')![0]).toEqual(['']);
  });

  it('hides clear when disabled or readonly', () => {
    const w1 = mount(UiInput, { props: { modelValue: 'abc', clearable: true, disabled: true } });
    expect(w1.find('[data-testid="ui-input-clear"]').exists()).toBe(false);
    const w2 = mount(UiInput, { props: { modelValue: 'abc', clearable: true, readonly: true } });
    expect(w2.find('[data-testid="ui-input-clear"]').exists()).toBe(false);
  });
});

describe('UiInput · error / disabled', () => {
  it('shows error text when error prop set', () => {
    const w = mount(UiInput, { props: { error: '手机号格式不正确' } });
    expect(w.find('[data-testid="ui-input-error"]').text()).toBe('手机号格式不正确');
    expect(w.classes()).toContain('ui-input--error');
  });

  it('no error text when error prop empty', () => {
    const w = mount(UiInput);
    expect(w.find('[data-testid="ui-input-error"]').exists()).toBe(false);
    expect(w.classes()).not.toContain('ui-input--error');
  });

  it('applies disabled class when disabled', () => {
    const w = mount(UiInput, { props: { disabled: true } });
    expect(w.classes()).toContain('ui-input--disabled');
  });
});

describe('UiInput · label', () => {
  it('renders label prop', () => {
    const w = mount(UiInput, { props: { label: '手机号' } });
    expect(w.find('[data-testid="ui-input-label"]').text()).toBe('手机号');
  });

  it('renders label slot over prop', () => {
    const w = mount(UiInput, {
      props: { label: 'fallback' },
      slots: { label: '<span data-testid="custom-label">自定义</span>' },
    });
    expect(w.find('[data-testid="custom-label"]').exists()).toBe(true);
  });

  it('does not render label element when label prop empty and no slot', () => {
    const w = mount(UiInput);
    expect(w.find('[data-testid="ui-input-label"]').exists()).toBe(false);
  });
});