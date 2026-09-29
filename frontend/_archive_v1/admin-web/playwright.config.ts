/**
 * admin-web Playwright e2e 配置（v1 最小可用）。
 *
 * 设计要点：
 * - 仅依赖 dev server（vite dev）+ MSW（service worker 拦截 /api/*）。
 * - 不依赖 Go 后端启动（migrations 占位时也能跑通 e2e）。
 * - 默认 baseURL 指向本地 vite dev 端口。
 * - webServer 自动拉起 dev server（pnpm dev），跑完即关。
 *
 * 对应 spec：docs/superpowers/plans/2026-09-23-doctors-v1.md §E2E 测试基础设施
 */
import { defineConfig, devices } from '@playwright/test';

const PORT = 5173;
const BASE_URL = `http://127.0.0.1:${PORT}`;

export default defineConfig({
  testDir: './e2e',
  // 默认 30s 单测超时，dev server 冷启 → 60s
  timeout: 60_000,
  expect: { timeout: 10_000 },
  // 失败时保留 trace 便于排查
  use: {
    baseURL: BASE_URL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    // MSW service worker 注册需要同源；用 127.0.0.1 而不是 localhost
    extraHTTPHeaders: {},
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // dev server 自动拉起
  webServer: {
    // 显式 npx vite，避免 pnpm 在 root workspace 找不到 dev script。
    command: 'npx vite --host 127.0.0.1 --port 5173',
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    stdout: 'ignore',
    stderr: 'pipe',
  },
  // 单线程跑（避免多 worker 同时启 dev server）
  workers: 1,
  // 报告器：本地默认 line，CI 用 html
  reporter: process.env.CI ? [['html'], ['github']] : 'list',
});