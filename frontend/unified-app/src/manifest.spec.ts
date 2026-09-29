/**
 * manifest.json 结构校验（mobile batch 2a）。
 *
 * 验证目标：uni-app 多端发布所需的最小配置字段。
 * 不验证 icon 文件物理存在（icons 由设计师提交，build 阶段拷贝）。
 * 不触发 build — 仅做静态结构校验，替代实际 build 验证。
 */
import { describe, it, expect } from 'vitest';
import manifest from './manifest.json';

describe('manifest.json · iOS distribute 配置', () => {
  it('基本字段：name / appid / vueVersion 完整', () => {
    expect(manifest.name).toBeTypeOf('string');
    expect(manifest.name.length).toBeGreaterThan(0);
    expect(manifest.appid).toBeTypeOf('string');
    expect(manifest.vueVersion).toBe('3');
  });

  it('ios.appid 是合法 Bundle ID（反向域名）', () => {
    const ios = manifest['app-plus']?.distribute?.ios;
    expect(ios?.appid).toBeTypeOf('string');
    expect(ios?.appid).toMatch(/^[a-z][a-z0-9-]*(\.[a-z0-9-]+)+$/);
  });

  it('ios.idfa = false（不启用广告追踪）', () => {
    const ios = manifest['app-plus']?.distribute?.ios;
    expect(ios?.idfa).toBe(false);
  });

  it('ios.requiredDeviceCapabilities 包含 arm64', () => {
    const caps = manifest['app-plus']?.distribute?.ios?.requiredDeviceCapabilities;
    expect(Array.isArray(caps)).toBe(true);
    expect(caps).toContain('arm64');
  });

  it('ios.icons 含 app / app@2x / app@3x（最低发布要求）', () => {
    const icons = manifest['app-plus']?.distribute?.ios?.icons;
    expect(icons?.app).toBeTypeOf('string');
    expect(icons?.['app@2x']).toBeTypeOf('string');
    expect(icons?.['app@3x']).toBeTypeOf('string');
  });

  it('ios.infoPlist.NSAppTransportSecurity 允许 http（开发需要）+ 白名单 localhost / 127.0.0.1', () => {
    const ats = manifest['app-plus']?.distribute?.ios?.infoPlist?.NSAppTransportSecurity;
    expect(ats?.NSAllowsArbitraryLoads).toBe(true);
    expect(ats?.NSExceptionDomains?.localhost?.NSExceptionAllowsInsecureHTTPLoads).toBe(true);
    expect(ats?.NSExceptionDomains?.['127.0.0.1']?.NSExceptionAllowsInsecureHTTPLoads).toBe(true);
  });

  it('ios.infoPlist.LSRequiresIPhoneOS = true', () => {
    expect(manifest['app-plus']?.distribute?.ios?.infoPlist?.LSRequiresIPhoneOS).toBe(true);
  });

  it('ios.infoPlist.UIRequiredDeviceCapabilities 与 ios.requiredDeviceCapabilities 一致', () => {
    const ios = manifest['app-plus']?.distribute?.ios;
    expect(ios?.infoPlist?.UIRequiredDeviceCapabilities).toEqual(ios?.requiredDeviceCapabilities);
  });

  it('ios.infoPlist.UISupportedInterfaceOrientations 仅竖屏（v1 锁定竖屏）', () => {
    const orients = manifest['app-plus']?.distribute?.ios?.infoPlist?.UISupportedInterfaceOrientations;
    expect(orients).toEqual(['UIInterfaceOrientationPortrait']);
  });
});