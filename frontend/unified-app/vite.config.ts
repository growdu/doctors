import { defineConfig } from 'vite';
import uni from '@dcloudio/vite-plugin-uni';
import path from 'node:path';

// unified-app v2 spike：3 域合并的 uni-app 配置。
// 编译目标：h5（PC web）+ mp-weixin（小程序）+ app（Android/iOS）。
// 后续按 plan §3.1 Phase 2 把 admin / patient / escort 三域路由挂上。
export default defineConfig({
  plugins: [uni()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    chunkSizeWarningLimit: 800,
  },
});