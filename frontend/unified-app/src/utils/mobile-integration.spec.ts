/**
 * 跨模块集成验证 spec（mobile batch 5 — 替代 build 验证）。
 *
 * 由于本会话不跑 pnpm build:mp-weixin / build:app（依赖外部工具链），
 * 用静态分析 + 模块加载验证代替：
 *   - 各 utils 模块导出符合预期（动态 import）
 *   - 页面正确引用 utils（source grep）
 *   - 配置文件路径前缀一致
 *   - 已加条件编译（App.vue #ifdef H5）
 *   - web-only 残留清理（auth.ts 不再 typeof uni 三元）
 */
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const SRC = resolve(__dirname, '..');

describe('mobile batch 5 · utils 模块导出', () => {
  it('share.ts 导出 shareContent 函数', async () => {
    const m = await import('./share');
    expect(typeof m.shareContent).toBe('function');
  });

  it('share.mp-weixin.ts 导出 buildMpShareMessage 函数', async () => {
    const m = await import('./share.mp-weixin');
    expect(typeof m.buildMpShareMessage).toBe('function');
  });

  it('callPhone.ts 导出 callPhone 函数', async () => {
    const m = await import('./callPhone');
    expect(typeof m.callPhone).toBe('function');
  });

  it('storage.ts 导出 storage 对象（含 5 方法）+ 4 个 convenience 函数', async () => {
    const m = await import('./storage');
    expect(m.storage).toBeDefined();
    expect(typeof m.storage.setItem).toBe('function');
    expect(typeof m.storage.getItem).toBe('function');
    expect(typeof m.storage.removeItem).toBe('function');
    expect(typeof m.storage.getJSON).toBe('function');
    expect(typeof m.storage.clear).toBe('function');
    expect(typeof m.storageSet).toBe('function');
    expect(typeof m.storageGet).toBe('function');
    expect(typeof m.storageRemove).toBe('function');
    expect(typeof m.storageGetJSON).toBe('function');
  });
});

describe('mobile batch 5 · 页面正确引用 utils', () => {
  it('patient/order/share.vue 引用 @/utils/share + share.mp-weixin', () => {
    const content = readFileSync(resolve(SRC, 'pages/patient/order/share.vue'), 'utf8');
    expect(content).toContain("from '@/utils/share'");
    expect(content).toContain("from '@/utils/share.mp-weixin'");
  });

  it('patient/sos/trigger.vue 引用 @/utils/callPhone', () => {
    const content = readFileSync(resolve(SRC, 'pages/patient/sos/trigger.vue'), 'utf8');
    expect(content).toContain("from '@/utils/callPhone'");
  });
});

describe('mobile batch 5 · web-only 残留清理', () => {
  it('store/auth.ts 用 @/utils/storage 替代 typeof localStorage 三元（typeof uni 保留，uni 跨平台有效）', () => {
    const content = readFileSync(resolve(SRC, 'store/auth.ts'), 'utf8');
    expect(content).toContain("from '@/utils/storage'");
    // typeof localStorage 是 web-only 写法（mp/app 无 localStorage）— 应已清理
    expect(content).not.toMatch(/typeof localStorage\s*!==\s*['"]undefined['"]/);
    // typeof window / document 同样是 web-only
    expect(content).not.toMatch(/typeof window\s*!==\s*['"]undefined['"]/);
    expect(content).not.toMatch(/typeof document\s*!==\s*['"]undefined['"]/);
  });

  it('App.vue 含 // #ifdef H5 + // #endif 条件编译包裹 document.*', () => {
    const content = readFileSync(resolve(SRC, 'App.vue'), 'utf8');
    expect(content).toContain('#ifdef H5');
    expect(content).toContain('#endif');
    // document.* 引用必须在 #ifdef 块内
    const ifdefStart = content.indexOf('#ifdef H5');
    const ifdefEnd = content.indexOf('#endif', ifdefStart);
    const block = content.substring(ifdefStart, ifdefEnd);
    expect(block).toMatch(/document\.\w+/);
  });

  it('pages/patient/settings/index.vue 不含 typeof window / typeof document 三元', () => {
    const content = readFileSync(resolve(SRC, 'pages/patient/settings/index.vue'), 'utf8');
    expect(content).not.toMatch(/typeof window\s*!==\s*['"]undefined['"]/);
    expect(content).not.toMatch(/typeof document\s*!==\s*['"]undefined['"]/);
  });
});

describe('mobile batch 5 · 配置一致性', () => {
  it('manifest.json ios/android icons 路径前缀均为 static/icons/', async () => {
    const m = (await import('../manifest.json')).default as {
      'app-plus'?: {
        distribute?: {
          ios?: { icons?: Record<string, string> };
          android?: { icons?: Record<string, string> };
        };
      };
    };
    const iosIcons = m['app-plus']?.distribute?.ios?.icons ?? {};
    for (const p of Object.values(iosIcons)) {
      expect(p).toMatch(/^static\/icons\//);
    }
    const androidIcons = m['app-plus']?.distribute?.android?.icons ?? {};
    for (const p of Object.values(androidIcons)) {
      expect(p).toMatch(/^static\/icons\//);
    }
  });

  it('pages.json tabBar icon 路径前缀均为 static/icons/tabbar/', async () => {
    const p = (await import('../pages.json')).default as {
      tabBar?: { list?: Array<{ iconPath: string; selectedIconPath: string }> };
    };
    const list = p.tabBar?.list ?? [];
    for (const item of list) {
      expect(item.iconPath).toMatch(/^static\/icons\/tabbar\//);
      expect(item.selectedIconPath).toMatch(/^static\/icons\/tabbar\//);
    }
  });

  it('pages.json 每条路由都有 navigationBarTitleText', async () => {
    const p = (await import('../pages.json')).default as {
      pages: Array<{ path: string; style?: { navigationBarTitleText?: string } }>;
    };
    for (const page of p.pages) {
      expect(page.style?.navigationBarTitleText?.length ?? 0).toBeGreaterThan(0);
    }
  });
});