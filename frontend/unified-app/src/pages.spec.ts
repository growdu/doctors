/**
 * pages.json 结构校验（mobile batch 3a）。
 *
 * 验证目标：tabBar 配置完整性 + 与 pages[] 路由一致性。
 * 不触发 build — 仅做静态结构校验。
 */
import { describe, it, expect } from 'vitest';
import pagesJson from './pages.json';

describe('pages.json · tabBar 配置（patient 域）', () => {
  const tabBar = pagesJson.tabBar;
  const pagePaths = new Set((pagesJson.pages as { path: string }[]).map((p) => p.path));

  it('tabBar.color / selectedColor / backgroundColor 存在', () => {
    expect(tabBar?.color).toMatch(/^#[0-9A-Fa-f]{6}$/);
    expect(tabBar?.selectedColor).toMatch(/^#[0-9A-Fa-f]{6}$/);
    expect(tabBar?.backgroundColor).toMatch(/^#[0-9A-Fa-f]{6}$/);
  });

  it('tabBar.list 含 4 项（首页/订单/消息/我的）', () => {
    expect(Array.isArray(tabBar?.list)).toBe(true);
    expect((tabBar?.list as unknown[]).length).toBe(4);
  });

  it('tabBar 每项含 pagePath / text / iconPath / selectedIconPath', () => {
    const list = tabBar?.list as Array<Record<string, string>>;
    for (const item of list) {
      expect(item.pagePath).toBeTypeOf('string');
      expect(item.text.length).toBeGreaterThan(0);
      expect(item.iconPath).toMatch(/^static\/icons\/tabbar\/.+\.png$/);
      expect(item.selectedIconPath).toMatch(/^static\/icons\/tabbar\/.+-active\.png$/);
    }
  });

  it('tabBar.pagePath 引用的路由都存在于 pages[]（无 build-time missing 错误）', () => {
    const list = tabBar?.list as Array<{ pagePath: string }>;
    for (const item of list) {
      expect(pagePaths.has(item.pagePath)).toBe(true);
    }
  });

  it('tabBar 4 项顺序：首页 → 订单 → 消息 → 我的', () => {
    const list = tabBar?.list as Array<{ text: string }>;
    expect(list.map((i) => i.text)).toEqual(['首页', '订单', '消息', '我的']);
  });
});

describe('pages.json · globalStyle 基础配置', () => {
  it('globalStyle.navigationBarTextStyle 合法（black/white）', () => {
    expect(['black', 'white']).toContain(pagesJson.globalStyle?.navigationBarTextStyle);
  });

  it('globalStyle.backgroundColor 与 navigationBarBackgroundColor 不同（区分内容/标题栏）', () => {
    expect(pagesJson.globalStyle?.backgroundColor).not.toBe(pagesJson.globalStyle?.navigationBarBackgroundColor);
  });
});

describe('pages.json · 路由完整性', () => {
  it('pages[] 含 75 条路由（patient 32 + escort 16 + admin 26 + home 1）', () => {
    expect(pagesJson.pages.length).toBe(75);
  });

  it('每条路由都有 style.navigationBarTitleText（移动端导航一致性）', () => {
    for (const p of pagesJson.pages) {
      expect((p as { style?: { navigationBarTitleText?: string } }).style?.navigationBarTitleText).toBeTypeOf('string');
      expect(((p as { style?: { navigationBarTitleText?: string } }).style?.navigationBarTitleText ?? '').length).toBeGreaterThan(0);
    }
  });

  it('pages[] 中无重复 path', () => {
    const paths = (pagesJson.pages as { path: string }[]).map((p) => p.path);
    expect(new Set(paths).size).toBe(paths.length);
  });
});

describe('pages.json · 关键页移动端样式（mobile batch 3b）', () => {
  const findPage = (path: string) =>
    (pagesJson.pages as Array<{ path: string; style?: Record<string, unknown> }>).find((p) => p.path === path);

  it('patient/order/list: enablePullDownRefresh（订单下拉刷新）', () => {
    const style = findPage('pages/patient/order/list')?.style;
    expect(style?.enablePullDownRefresh).toBe(true);
  });

  it('patient/message/list: enablePullDownRefresh（消息下拉刷新）', () => {
    const style = findPage('pages/patient/message/list')?.style;
    expect(style?.enablePullDownRefresh).toBe(true);
  });

  it('patient/wallet/index: enablePullDownRefresh（钱包下拉刷新）', () => {
    const style = findPage('pages/patient/wallet/index')?.style;
    expect(style?.enablePullDownRefresh).toBe(true);
  });

  it('patient/sos/trigger: navigationStyle = custom（紧急页全屏）+ 红底色', () => {
    const style = findPage('pages/patient/sos/trigger')?.style;
    expect(style?.navigationStyle).toBe('custom');
    expect(style?.backgroundColor).toMatch(/^#[0-9A-Fa-f]{6}$/);
  });

  it('escort/invitations/index: enablePullDownRefresh + app-plus.titleNView.buttons（抢单池）', () => {
    const style = findPage('pages/escort/invitations/index')?.style as Record<string, unknown> & {
      'app-plus'?: { titleNView?: { buttons?: unknown[] } };
    };
    expect(style?.enablePullDownRefresh).toBe(true);
    expect(Array.isArray(style?.['app-plus']?.titleNView?.buttons)).toBe(true);
    expect((style?.['app-plus']?.titleNView?.buttons ?? []).length).toBeGreaterThan(0);
  });

  it('escort/orders/index: enablePullDownRefresh（任务列表）', () => {
    const style = findPage('pages/escort/orders/index')?.style;
    expect(style?.enablePullDownRefresh).toBe(true);
  });
});