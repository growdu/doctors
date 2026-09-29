/**
 * DomainSwitcher 单测。
 *
 * 验证目标：
 *   - 3 张卡片渲染（patient / escort / admin）
 *   - currentDomain 高亮 class 切换
 *   - 用户 roles 不含某域代表 role 时，lock 图标 + disabled class
 *   - 点击可用域 emit('select', id)
 *   - 点击未解锁域不 emit
 */
import { describe, expect, it, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import DomainSwitcher from './DomainSwitcher.vue';
import { useAuthStore } from '@/store/auth';

// auth store 不依赖网络，纯本地；用真 store
beforeEach(() => {
  setActivePinia(createPinia());
});

describe('DomainSwitcher · 3 cards render', () => {
  it('renders patient + escort + admin cards', () => {
    const w = mount(DomainSwitcher);
    expect(w.find('[data-testid="domain-card-patient"]').exists()).toBe(true);
    expect(w.find('[data-testid="domain-card-escort"]').exists()).toBe(true);
    expect(w.find('[data-testid="domain-card-admin"]').exists()).toBe(true);
  });
});

describe('DomainSwitcher · availability based on roles', () => {
  it('all 3 cards disabled when user has no roles', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: '', active_role: '', roles: [], real_name_verified: false };
    const w = mount(DomainSwitcher);
    expect(w.find('[data-testid="domain-card-patient"]').classes()).toContain('domain-switcher__card--disabled');
    expect(w.find('[data-testid="domain-card-escort"]').classes()).toContain('domain-switcher__card--disabled');
    expect(w.find('[data-testid="domain-card-admin"]').classes()).toContain('domain-switcher__card--disabled');
    expect(w.findAll('[data-testid="domain-card-lock"]')).toHaveLength(3);
  });

  it('patient card available when user has patient role', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient'], real_name_verified: false };
    const w = mount(DomainSwitcher);
    expect(w.find('[data-testid="domain-card-patient"]').classes()).not.toContain('domain-switcher__card--disabled');
    expect(w.find('[data-testid="domain-card-escort"]').classes()).toContain('domain-switcher__card--disabled');
  });

  it('escort card available when user has escort role', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'escort', active_role: 'escort', roles: ['escort'], real_name_verified: false };
    const w = mount(DomainSwitcher);
    expect(w.find('[data-testid="domain-card-escort"]').classes()).not.toContain('domain-switcher__card--disabled');
  });

  it('admin card available for any admin_* role', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'order_admin', active_role: 'order_admin', roles: ['order_admin'], real_name_verified: false };
    const w = mount(DomainSwitcher);
    expect(w.find('[data-testid="domain-card-admin"]').classes()).not.toContain('domain-switcher__card--disabled');
  });
});

describe('DomainSwitcher · currentDomain highlight', () => {
  it('applies active class to currentDomain card', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient', 'escort'], real_name_verified: false };
    const w = mount(DomainSwitcher, { props: { currentDomain: 'patient' } });
    expect(w.find('[data-testid="domain-card-patient"]').classes()).toContain('domain-switcher__card--active');
    expect(w.find('[data-testid="domain-card-escort"]').classes()).not.toContain('domain-switcher__card--active');
  });

  it('no active class when currentDomain undefined', () => {
    const w = mount(DomainSwitcher);
    expect(w.findAll('.domain-switcher__card--active')).toHaveLength(0);
  });
});

describe('DomainSwitcher · click behavior', () => {
  it('emits select on click for available domain', async () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient'], real_name_verified: false };
    const w = mount(DomainSwitcher);
    await w.find('[data-testid="domain-card-patient"]').trigger('click');
    expect(w.emitted('select')![0]).toEqual(['patient']);
  });

  it('does not emit select for locked domain', async () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient'], real_name_verified: false };
    const w = mount(DomainSwitcher);
    await w.find('[data-testid="domain-card-escort"]').trigger('click');
    expect(w.emitted('select')).toBeUndefined();
  });
});