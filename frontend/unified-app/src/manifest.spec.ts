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

describe('manifest.json · Android distribute 配置', () => {
  it('android.minSdkVersion / targetSdkVersion 合规（21+/34）', () => {
    const android = manifest['app-plus']?.distribute?.android;
    expect(android?.minSdkVersion).toBeGreaterThanOrEqual(21);
    expect(android?.targetSdkVersion).toBe(34);
  });

  it('android.abiFilters 包含主流 ABI（armeabi-v7a + arm64-v8a）', () => {
    const abi = manifest['app-plus']?.distribute?.android?.abiFilters;
    expect(abi).toContain('arm64-v8a');
    expect(abi).toContain('armeabi-v7a');
  });

  it('android.permissions 含 INTERNET / ACCESS_NETWORK_STATE / 定位权限（escort 签到用）', () => {
    const perms = manifest['app-plus']?.distribute?.android?.permissions as string[];
    expect(perms.some((p) => p.includes('android.permission.INTERNET'))).toBe(true);
    expect(perms.some((p) => p.includes('android.permission.ACCESS_NETWORK_STATE'))).toBe(true);
    expect(perms.some((p) => p.includes('android.permission.ACCESS_COARSE_LOCATION'))).toBe(true);
    expect(perms.some((p) => p.includes('android.permission.ACCESS_FINE_LOCATION'))).toBe(true);
  });

  it('android.icons 含 6 尺寸（ldpi → xxxhdpi，覆盖全 DPI 设备）', () => {
    const icons = manifest['app-plus']?.distribute?.android?.icons;
    expect(icons?.ldpi).toBeTypeOf('string');
    expect(icons?.mdpi).toBeTypeOf('string');
    expect(icons?.hdpi).toBeTypeOf('string');
    expect(icons?.xhdpi).toBeTypeOf('string');
    expect(icons?.xxhdpi).toBeTypeOf('string');
    expect(icons?.xxxhdpi).toBeTypeOf('string');
  });
});

describe('manifest.json · mp-weixin 配置', () => {
  it('mp-weixin.appid 是 wx 开头 32 字符串（占位 / 真实）', () => {
    const mp = manifest['mp-weixin'];
    expect(mp?.appid).toMatch(/^wx[a-zA-Z0-9_]+$/);
    // 占位标记（v3 替换为真实 appid 后会失效，需同步更新此 spec）
    expect(mp?.appid).toBe('wxPLACEHOLDER');
  });

  it('mp-weixin.setting.urlCheck = false（dev 用，prod 应打开）', () => {
    expect(manifest['mp-weixin']?.setting?.urlCheck).toBe(false);
  });

  it('mp-weixin.lazyCodeLoading / optimizeMemory 启用（性能优化）', () => {
    expect(manifest['mp-weixin']?.lazyCodeLoading).toBe(true);
    expect(manifest['mp-weixin']?.optimizeMemory).toBe(true);
  });

  it('mp-weixin.requiredPrivateInfos 为空数组（v1 不使用 location/chooseMedia 等）', () => {
    expect(manifest['mp-weixin']?.requiredPrivateInfos).toEqual([]);
  });
});