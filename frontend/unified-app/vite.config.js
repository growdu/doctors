// unified-app v2 spike：3 域合并的 uni-app 配置。
// 验证模式：与 patient-miniapp 完全一致（import default + plugins: [uni()]）
// 加 server.proxy：h5 origin (5174) 跨域到 auth-service (8081) + order (8082) + match (8083)
import * as uniModule from '@dcloudio/vite-plugin-uni';
import path from 'node:path';

const uni = uniModule.default.default || uniModule.default;

export default {
  plugins: [uni()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
        secure: false,
      },
      '/mockServiceWorker.js': {
        target: 'http://127.0.0.1:5174',
        changeOrigin: false,
      },
    },
  },
};
