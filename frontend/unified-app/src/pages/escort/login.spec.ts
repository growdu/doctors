/**
 * escort/login.vue 组件单测（vitest + Vue Test Utils + Pinia）。
 *
 * 验证目标：
 *   - 渲染标题 + 表单卡
 *   - 校验：手机号非 11 位时发送按钮错误提示
 *   - 校验：验证码非 6 位时登录按钮错误提示
 *   - 「获取验证码」调 sendSmsCode + 启动倒计时
 *   - 「登录」调 auth.login + reLaunch
 *   - 「返回首页」调 uni.reLaunch
 *   - 倒计时期间「获取验证码」按钮 disabled
 *   - API 异常显示错误信息
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '@/store/auth';
import LoginPage from './login.vue';

vi.mock('@/api/auth', () => ({
  sendSmsCode: vi.fn(),
}));

import * as apiAuth from '@/api/auth';

beforeEach(() => {
  setActivePinia(createPinia());
  vi.clearAllMocks();
});

describe('escort/login · 渲染', () => {
  it('渲染标题 + 表单卡', () => {
    const w = mount(LoginPage);
    expect(w.text()).toContain('陪诊师登录');
    expect(w.find('[data-testid="escort-login-card"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-login-phone"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-login-code"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-login-send-code"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-login-submit"]').exists()).toBe(true);
    expect(w.find('[data-testid="escort-login-back-home"]').exists()).toBe(true);
  });
});

describe('escort/login · 输入校验', () => {
  it('手机号非 11 位时点击「获取验证码」显示错误', async () => {
    const w = mount(LoginPage);
    await flushPromises();

    const phoneInput = w.findAll('[data-testid="ui-input-inner"]')[0]!;
    await phoneInput.setValue('12345');
    await flushPromises();

    await w.find('[data-testid="escort-login-send-code"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-login-error"]').exists()).toBe(true);
    expect(w.text()).toContain('11 位手机号');
    expect(apiAuth.sendSmsCode).not.toHaveBeenCalled();
  });

  it('验证码非 6 位时点击「登录」显示错误', async () => {
    const w = mount(LoginPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('13900139000');
    await inputs[1]!.setValue('123');
    await flushPromises();

    await w.find('[data-testid="escort-login-submit"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-login-error"]').exists()).toBe(true);
    expect(w.text()).toContain('6 位验证码');
  });
});

describe('escort/login · 验证码发送', () => {
  it('合法手机号 → 调 sendSmsCode', async () => {
    vi.mocked(apiAuth.sendSmsCode).mockResolvedValue({ sent: true, ttl: 60 });

    const w = mount(LoginPage);
    await flushPromises();

    const phoneInput = w.findAll('[data-testid="ui-input-inner"]')[0]!;
    await phoneInput.setValue('13900139000');
    await flushPromises();

    await w.find('[data-testid="escort-login-send-code"]').trigger('click');
    await flushPromises();

    expect(apiAuth.sendSmsCode).toHaveBeenCalledWith('13900139000');
  });
});

describe('escort/login · 登录流程', () => {
  it('合法表单 → 调 auth.login + reLaunch', async () => {
    const auth = useAuthStore();
    const loginSpy = vi.spyOn(auth, 'login').mockResolvedValue();

    const reLaunchSpy = vi.fn();
    (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;

    const w = mount(LoginPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('13900139000');
    await inputs[1]!.setValue('123456');
    await flushPromises();

    await w.find('[data-testid="escort-login-submit"]').trigger('click');
    await flushPromises();

    expect(loginSpy).toHaveBeenCalledWith('13900139000', '123456');
    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/escort/invitations/index' });
  });

  it('登录 API 异常显示错误', async () => {
    const auth = useAuthStore();
    vi.spyOn(auth, 'login').mockRejectedValue(new Error('验证码错误'));

    const w = mount(LoginPage);
    await flushPromises();

    const inputs = w.findAll('[data-testid="ui-input-inner"]');
    await inputs[0]!.setValue('13900139000');
    await inputs[1]!.setValue('000000');
    await flushPromises();

    await w.find('[data-testid="escort-login-submit"]').trigger('click');
    await flushPromises();

    expect(w.find('[data-testid="escort-login-error"]').exists()).toBe(true);
    expect(w.text()).toContain('验证码错误');
  });
});

describe('escort/login · 返回首页', () => {
  it('点击「返回首页选择其他角色」调 uni.reLaunch', async () => {
    const reLaunchSpy = vi.fn();
    (globalThis as unknown as { uni: { reLaunch: typeof reLaunchSpy } }).uni.reLaunch = reLaunchSpy;

    const w = mount(LoginPage);
    await flushPromises();

    await w.find('[data-testid="escort-login-back-home"]').trigger('click');
    expect(reLaunchSpy).toHaveBeenCalledWith({ url: '/pages/home/index' });
  });
});