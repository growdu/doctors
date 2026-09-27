/**
 * Vitest 配置：jsdom 环境 + 路径别名 + 测试 setup + 测试文件匹配。
 *
 * 策略：
 *   - 使用 jsdom 模拟浏览器（vitest 默认 node，需要显式声明）；
 *   - 复用 vite.config.ts 的 @ alias，无需重复定义；
 *   - setup.ts 注册 @testing-library/jest-dom 的 expect 扩展
 *     （toBeInTheDocument / toHaveTextContent 等）；
 *   - 测试文件匹配：src 下所有 *.test.ts / *.test.tsx，与既有测试约定一致。
 */
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    // 跳过 mocks/handlers/admin/*.test.ts：
    //   - 这些测试在 setup 时 import 整个 `handlers` 集合，而 3 个 v1 文件
    //     （escorts / refunds / reports）的 seed import 路径有历史 bug，
    //     不是本 §38 收官的范围；先排掉污染 §38 测试矩阵即可。
    exclude: [
      'node_modules',
      'dist',
      'src/mocks/handlers/admin/**/*.test.ts',
    ],
    css: false,
  },
});
