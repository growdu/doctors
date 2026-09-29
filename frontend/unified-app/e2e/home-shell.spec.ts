/**
 * unified-app e2e spike 2 — HomeShell + 角色切换全链路（Phase 3.0.7）。
 *
 * 跑测路径（待 CI 集成，dev.md §44.7 P2）：
 *   1. 启动 auth-service + unified-app dev:h5（前提：§42 v2 multi-role 已 commit）
 *   2. Playwright 访问 http://127.0.0.1:5174/
 *   3. 注入 mock auth state（page.addInitScript），跳过真 SMS 登录
 *   4. 验证：HomeShell 渲染 + 角色切换弹层打开 + 列表多角色 + 点击切换
 *   5. 截图证据：test-results/unified-app-spike-3-home.png + spike-4-roleswitcher.png
 *
 * 已知限制：
 *   - 当前 e2e 不依赖真后端（page.addInitScript 注入 mock auth store 状态）
 *   - dev server 端口自动检测（5174 / 5175 fallback）
 *   - uni-app h5 部分 `<text>` 组件渲染结构特殊，用宽松文本匹配
 */
import { test, expect } from '@playwright/test';

const BASE = process.env.UNIFIED_E2E_URL ?? 'http://127.0.0.1:5174';
const ALT_PORT = 'http://127.0.0.1:5175';

async function pickBase(): Promise<string> {
  for (const url of [BASE, ALT_PORT]) {
    try {
      const resp = await fetch(url, { signal: AbortSignal.timeout(2000) });
      if (resp.ok) return url;
    } catch { /* try next */ }
  }
  throw new Error(`unified-app dev server not reachable at ${BASE} or ${ALT_PORT}`);
}

/**
 * 注入 multi-role auth 状态（patient + escort + admin），跳过 SMS 登录。
 * 与 src/store/auth.ts 的 bootstrap() 行为对齐：从 localStorage 恢复。
 */
const MOCK_MULTI_ROLE = `
(() => {
  const STORAGE_KEY = 'unified.auth';
  const data = {
    token: 'mock.multi-role.token',
    user: {
      id: 1,
      phone: '13800138000',
      role: 'patient',
      active_role: 'patient',
      roles: ['patient', 'escort', 'order_admin'],
      real_name_verified: false,
    },
  };
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch {}
})();
`;

test('HomeShell 渲染：用户卡 + 域切换器 + 退出按钮', async () => {
  const base = await pickBase();
  const { chromium } = await import('@playwright/test');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.addInitScript(MOCK_MULTI_ROLE);
  await page.goto(base);

  // 顶部品牌卡
  await expect(page.getByText('Doctors 统一 App')).toBeVisible({ timeout: 15000 });

  // 用户卡（注入 mock 后应可见）
  await expect(page.locator('[data-testid="home-user-phone"]')).toBeVisible();
  await expect(page.locator('[data-testid="home-user-active"]')).toContainText('patient');
  await expect(page.locator('[data-testid="home-user-roles"]')).toBeVisible();

  // 角色切换按钮（显示总角色数）
  await expect(page.locator('[data-testid="home-role-switch-btn"]')).toContainText('切换激活角色');
  await expect(page.locator('[data-testid="home-role-switch-btn"]')).toContainText('(3)');

  // 域切换器：3 域都可见
  await expect(page.locator('[data-testid="domain-card-patient"]')).toBeVisible();
  await expect(page.locator('[data-testid="domain-card-escort"]')).toBeVisible();
  await expect(page.locator('[data-testid="domain-card-admin"]')).toBeVisible();

  // 截图 HomeShell spike 状态
  await page.screenshot({ path: 'test-results/unified-app-spike-3-home.png', fullPage: true });
  console.log('[spike-3] 截图：test-results/unified-app-spike-3-home.png');

  await browser.close();
});

test('RoleSwitcherModal 打开 + 列出多角色 + 当前 active ✓', async () => {
  const base = await pickBase();
  const { chromium } = await import('@playwright/test');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.addInitScript(MOCK_MULTI_ROLE);
  await page.goto(base);

  // 点击「切换激活角色」按钮
  await page.locator('[data-testid="home-role-switch-btn"]').click();

  // 弹层列表可见
  await expect(page.locator('[data-testid="role-switcher-list"]')).toBeVisible({ timeout: 5000 });

  // 3 个角色项
  await expect(page.locator('[data-testid="role-option-patient"]')).toBeVisible();
  await expect(page.locator('[data-testid="role-option-escort"]')).toBeVisible();
  await expect(page.locator('[data-testid="role-option-order_admin"]')).toBeVisible();

  // 当前 active 角色显示 ✓
  await expect(page.locator('[data-testid="role-option-active"]')).toBeVisible();
  await expect(page.locator('[data-testid="role-option-active"]')).toHaveText('✓');

  // 截图弹层状态
  await page.screenshot({ path: 'test-results/unified-app-spike-4-roleswitcher.png', fullPage: true });
  console.log('[spike-4] 截图：test-results/unified-app-spike-4-roleswitcher.png');

  await browser.close();
});

test('DomainSwitcher 按角色可用性：multi-role 用户 3 域全开', async () => {
  const base = await pickBase();
  const { chromium } = await import('@playwright/test');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.addInitScript(MOCK_MULTI_ROLE);
  await page.goto(base);

  // 3 个域卡片都不可锁（无 disabled class）
  for (const id of ['patient', 'escort', 'admin']) {
    const cls = await page.locator(`[data-testid="domain-card-${id}"]`).getAttribute('class');
    expect(cls, `domain ${id} should not have disabled class`).not.toContain('disabled');
    await expect(page.locator(`[data-testid="domain-card-${id}"]`)).toBeVisible();
  }

  // 锁图标（未解锁）应为 0
  expect(await page.locator('[data-testid="domain-card-lock"]').count()).toBe(0);

  await browser.close();
});