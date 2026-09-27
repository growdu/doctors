/**
 * admin-web v1 e2e: 登录 → 进 Orders 页面 → 看到 mock 订单列表 → 点详情。
 *
 * 流程：
 *  1. 打开 / → AuthBootstrap 检查无 session → 跳 /login
 *  2. 用 'super' / 任意密码登录（authStore.login 是纯 mock，按用户名前缀决定 role）
 *  3. 登录后跳到 /orders（默认路由）
 *  4. 等待 OrderListPage 渲染表格 + MSW 返回的 mock 订单
 *  5. 点第一条订单行的"详情"按钮 → 进 OrderDetailPage
 *  6. 验证关键字段渲染
 *
 * 关键覆盖链路：
 *  - main.tsx 装配链（MSW + ConfigProvider + Router）
 *  - MSW service worker 注册（mockServiceWorker.js 已在 public/ 下）
 *  - authStore.login + bootstrap 持久化
 *  - OrderListPage 调 /api/v1/admin/orders（被 MSW 拦截）
 *  - OrderDetailPage 调 /api/v1/admin/orders/:id（被 MSW 拦截）
 */
import { test, expect } from '@playwright/test';

const BASE = 'http://127.0.0.1:5173';

test.describe('admin-web v1 e2e smoke', () => {
  test('login → Orders list → Order detail (MSW 全链路)', async ({ page }) => {
    // 1. 进首页 → 没 session 跳 /login
    await page.goto(BASE + '/');

    // 等 AuthBootstrap 跑完 + 重定向发生
    await expect(page).toHaveURL(/\/login$/, { timeout: 15_000 });

    // 2. 填用户名 + 密码（'super' 前缀 → super_admin role，绕过 11003 拦截）
    await page.getByLabel(/用户名|username/i).first().fill('super');
    await page.getByLabel(/密码|password/i).first().fill('mock-password');

    // 点登录按钮
    const submitBtn = page.getByRole('button', { name: /登\s*录|login|sign\s*in/i }).first();
    await submitBtn.click();

    // 3. 登录成功 → 默认跳 /dashboard（router: / → <Navigate to="/dashboard" />）
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 });

    // 4. 手动导航到 /orders（侧边栏菜单 → OrderListPage）
    await page.goto(BASE + '/orders');

    // 等 OrderListPage 渲染（标题 + 表格 + mock 订单至少一条）
    // 用 MSW seed 数据：`mockOrders` 至少有一条 seeded record
    await expect(page.getByText(/订单|Orders?/i).first()).toBeVisible({ timeout: 15_000 });

    // 等表格出现 — 任意 data-testid 或 ant-table-row
    const firstRow = page.locator('.ant-table-row').first();
    await expect(firstRow).toBeVisible({ timeout: 15_000 });

    // 验证 MSW 真的拦截了 /api/v1/admin/orders（不是 vite proxy 兜底）
    const apiResponse = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/admin/orders') && resp.status() === 200,
      { timeout: 15_000 },
    );
    expect(apiResponse.request().method()).toBe('GET');

    // 5. 点第一行的"详情"按钮（按钮文字以"详情"开头，可能拼"详情 > "）
    const detailLink = firstRow.getByRole('link', { name: /详情/ }).first()
      .or(firstRow.getByRole('button', { name: /详情/ }).first());
    await detailLink.click();

    // 6. 进 OrderDetailPage — URL 含 /orders/<id>
    await expect(page).toHaveURL(/\/orders\/\d+/, { timeout: 10_000 });

    // 详情页应显示订单 ID 或"订单详情"标题（OrderDetailPage 用 data-testid 锚定）
    await expect(page.getByTestId('order-detail-page')).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByText(/订单\s*\d+/i).first()).toBeVisible({
      timeout: 10_000,
    });

    // 验证详情页内容：基础信息 card（AntCard title="基础信息"）+ 订单号
    await expect(page.getByText('基础信息').first()).toBeVisible({ timeout: 5_000 });
  });

  test('MSW service worker 已注册（拦截 /api/v1/admin/orders）', async ({ page }) => {
    await page.goto(BASE + '/');
    // 给 main.tsx + MSW 启动 + SW 注册充足时间
    await page.waitForTimeout(3000);

    // 浏览器控制台应有 MSW 启动日志（"[MSW] Mocking enabled"）
    // 用 page.on('console') 在 beforeAll 收集断言，这里用 page.evaluate 读 navigator.serviceWorker
    const swRegistered = await page.evaluate(async () => {
      if (!('serviceWorker' in navigator)) return false;
      const regs = await navigator.serviceWorker.getRegistrations();
      return regs.some((r) => r.active?.scriptURL.includes('mockServiceWorker.js'));
    });
    expect(swRegistered).toBe(true);
  });
});