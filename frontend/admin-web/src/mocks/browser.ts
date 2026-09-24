/**
 * MSW browser 启动入口（v2 增量适配）。
 *
 * 真实装配由 main.tsx 在 dev 环境调用 `worker.start({ onUnhandledRequest: 'warn' })`。
 *
 * 注意：本机当前未跑 `npx msw init public/` 生成 mockServiceWorker.js 二进制
 * （任务契约禁止 npm install / 跑工具链），所以 `worker.start()` 在没装 SW
 * 的环境下不会拦截请求；handler 文件本身完整可读，便于：
 *   1. 测试环境（vitest + msw/node）直接 `setupServer(...handlers)`；
 *   2. 后续用户在本地 `npm install` 后跑 `npx msw init public/` 即可启用 dev mock。
 *
 * 对应 spec：2026-09-24-admin-web-setup.md §Task 5
 */
import { setupWorker } from 'msw/browser';
import { handlers } from './handlers';

export const worker = setupWorker(...handlers);

export async function startMockServiceWorker(): Promise<void> {
  if (typeof window === 'undefined') return;
  await worker.start({
    onUnhandledRequest: 'bypass',
    serviceWorker: {
      url: '/mockServiceWorker.js',
    },
  });
}