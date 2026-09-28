/**
 * unified-app e2e smoke (h5 渲染 + 截图证明 spike)。
 *
 * 跑测：
 *   1. 启 auth-service + unified-app dev:h5（前提：§42 v2 multi-role 已 commit）
 *   2. 浏览器访问 http://127.0.0.1:5174/ → 看 SMS 登录表单
 *   3. 截图证明 spike 渲染正确
 *
 * 已知限制（spike 阶段）：
 *   - uni-app h5 的 `<uview>` 组件 placeholder 不被浏览器解析为原生 placeholder 属性
 *   - 改用 text 内容匹配（更兼容）
 */
import { test, expect } from '@playwright/test';

const BASE = process.env.UNIFIED_E2E_URL ?? 'http://127.0.0.1:5174';
const ALT_PORT = 'http://127.0.0.1:5175';

async function pickBase(): Promise<string> {
  // dev server 可能用 5174 或 5175（5174 占用时自动+1）；先 ping 5174，失败再 5175
  for (const url of [BASE, ALT_PORT]) {
    try {
      const resp = await fetch(url, { signal: AbortSignal.timeout(2000) });
      if (resp.ok) return url;
    } catch { /* try next */ }
  }
  throw new Error(`unified-app dev server not reachable at ${BASE} or ${ALT_PORT}`);
}

test('unified-app h5 spike 渲染 SMS 登录表单', async () => {
  const base = await pickBase();
  const { chromium } = await import('@playwright/test');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(base);

  // 等 home shell 渲染
  await expect(page.getByText('unified-app v2 spike')).toBeVisible({ timeout: 15000 });
  await expect(page.getByText('3 端合并 + 多角色 + 角色切换 demo')).toBeVisible();

  // SMS 登录表单存在（uni-app h5 组件渲染结构特殊，用最宽松的文本断言）
  await expect(page.getByText('手机号：')).toBeVisible();
  await expect(page.getByText('验证码：')).toBeVisible();
  // 按钮文本（在 uni-h5 button 内嵌 <text>，所以 *:has-text 也可能匹配不上；
  // 用 page.locator 全文检索最稳）
  const hasSendSms = await page.locator(':text("发验证码")').count();
  const hasLogin = await page.locator(':text("登录")').count();
  expect(hasSendSms).toBeGreaterThanOrEqual(1);
  expect(hasLogin).toBeGreaterThanOrEqual(1);

  // 截图 spike 状态
  await page.screenshot({ path: 'test-results/unified-app-spike-1-home.png', fullPage: true });
  console.log('[spike-1] 截图：test-results/unified-app-spike-1-home.png');

  await browser.close();
});

test('unified-app h5 跨域 proxy 到 :8081 auth-service', async () => {
  const base = await pickBase();
  const { chromium } = await import('@playwright/test');
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto(base);

  // 验证 proxy：page 内部 fetch /api 应转发到 :8081
  const resp = await page.evaluate(async (baseUrl) => {
    const r = await fetch(`${baseUrl}/api/v1/auth/sms/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone: '13900139011' }),
    });
    return { status: r.status, body: await r.json() };
  }, base);

  expect(resp.status).toBe(200);
  expect(resp.body.code).toBe(0); // auth-service 业务码 0 = ok

  await page.screenshot({ path: 'test-results/unified-app-spike-2-proxy.png', fullPage: true });
  console.log('[spike-2] 截图：test-results/unified-app-spike-2-proxy.png');

  await browser.close();
});