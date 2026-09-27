import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    open: false,
    // e2e fallback: vite proxy 把 /api/* 转发到本地 Go admin service。
    // 默认情况下，dev 环境 MSW service worker 会拦截 /api/*，
    // 但当 SW 注册失败 / e2e runner 想跑真后端时可走这里。
    // 目标端口参考 .env.example → VITE_ADMIN_API_PROXY_TARGET（默认 8080）。
    proxy: {
      '/api': {
        target: process.env.VITE_ADMIN_API_PROXY_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
      '/mockServiceWorker.js': {
        target: 'http://127.0.0.1:5173',
        changeOrigin: false,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    chunkSizeWarningLimit: 800,
  },
});