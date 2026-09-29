/**
 * Vitest 配置：unified-app v2 测试基础设施。
 *
 * 设计目标：
 *   - happy-dom 环境（Vue 组件 DOM 渲染 + localStorage 等浏览器 API）
 *   - 路径别名 @/ 同步 vite.config.js（避免重复定义）
 *   - uni-app 全局 mock：test/setup.ts 注入（uni.* / getCurrentPages 等）
 *   - 组件测试单文件 .spec.ts 后缀，与业务 .ts 同源
 *
 * 对应：dev.md §43.3.3（spike 阶段 uview-plus bug 临时移除，后续组件库逐步回归）
 */
import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import path from 'node:path';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    environmentOptions: {
      jsdom: {
        // jsdom 默认 url=about:blank 时禁用 localStorage；给个 fake http URL 让 localStorage 启用
        url: 'http://localhost/',
      },
    },
    globals: true,
    setupFiles: ['./test/setup.ts'],
    include: [
      'src/**/*.spec.ts',
      'src/**/*.test.ts',
      'src/**/*.spec.tsx',
      'src/**/*.test.tsx',
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/**/*.{ts,vue}'],
      exclude: ['src/**/*.spec.ts', 'src/**/*.test.ts', 'src/types/**'],
    },
  },
});