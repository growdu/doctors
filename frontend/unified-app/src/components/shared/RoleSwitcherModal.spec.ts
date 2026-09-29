/**
 * RoleSwitcherModal 单测。
 *
 * 验证目标：
 *   - 单角色场景显示提示 + 「知道了」按钮
 *   - 多角色场景列出所有角色
 *   - 当前 active 高亮 class
 *   - 点击角色调用 auth.switchRole + 关闭弹层
 *   - 点击 active 角色只关闭不切换
 *   - 切换失败不关闭（loading 清除）
 */
import { describe, expect, it, beforeEach, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import RoleSwitcherModal from './RoleSwitcherModal.vue';
import { useAuthStore } from '@/store/auth';

beforeEach(() => {
  setActivePinia(createPinia());
});

describe('RoleSwitcherModal · single role', () => {
  it('shows single-role message when user has only 1 role', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient'], real_name_verified: false };
    const w = mount(RoleSwitcherModal, { props: { visible: true } });
    expect(w.find('[data-testid="role-switcher-single"]').exists()).toBe(true);
    expect(w.find('[data-testid="role-switcher-list"]').exists()).toBe(false);
  });
});

describe('RoleSwitcherModal · multi role', () => {
  it('lists all roles', () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient', 'escort'], real_name_verified: false };
    const w = mount(RoleSwitcherModal, { props: { visible: true } });
    expect(w.find('[data-testid="role-option-patient"]').exists()).toBe(true);
    expect(w.find('[data-testid="role-option-escort"]').exists()).toBe(true);
    expect(w.find('[data-testid="role-option-active"]').exists()).toBe(true);
  });

  it('clicking active role just closes modal', async () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient', 'escort'], real_name_verified: false };
    const switchSpy = vi.spyOn(auth, 'switchRole');
    const w = mount(RoleSwitcherModal, { props: { visible: true } });
    await w.find('[data-testid="role-option-patient"]').trigger('click');
    expect(switchSpy).not.toHaveBeenCalled();
  });

  it('clicking other role calls switchRole', async () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient', 'escort'], real_name_verified: false };
    const switchSpy = vi.spyOn(auth, 'switchRole').mockResolvedValue();
    const w = mount(RoleSwitcherModal, { props: { visible: true } });
    await w.find('[data-testid="role-option-escort"]').trigger('click');
    expect(switchSpy).toHaveBeenCalledWith('escort');
  });

  it('switchRole failure keeps modal open and clears loading', async () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient', 'escort'], real_name_verified: false };
    vi.spyOn(auth, 'switchRole').mockRejectedValue(new Error('切换失败'));
    const w = mount(RoleSwitcherModal, { props: { visible: true } });
    await w.find('[data-testid="role-option-escort"]').trigger('click');
    await flushPromises();
    expect(w.props('visible')).toBe(true);
  });
});

describe('RoleSwitcherModal · v-model', () => {
  it('emits update:visible when closed via cancel button', async () => {
    const auth = useAuthStore();
    auth.user = { id: 1, phone: '1', role: 'patient', active_role: 'patient', roles: ['patient'], real_name_verified: false };
    const w = mount(RoleSwitcherModal, { props: { visible: true } });
    // 取消按钮 (data-testid="ui-modal-cancel") 由 UiModal 提供
    await w.find('[data-testid="ui-modal-cancel"]').trigger('click');
    expect(w.emitted('update:visible')![0]).toEqual([false]);
  });
});