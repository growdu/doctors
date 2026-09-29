// unified-app v2 · vite 配置（Phase 3.0.5/6）。
//
// 关键决策：sass @import 路径用 loadPaths + 短文件名（'styles/tokens'），
// 让 uni-app sass-loader 在处理 scoped style 块时能解析，
// 避免 @ 别名（@/...）或 @use 在 sass 内部行号偏移导致的失败。
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
  css: {
    preprocessorOptions: {
      scss: {
        loadPaths: [path.resolve(__dirname, './src')],
        silenceDeprecations: ['legacy-js-api', 'import'],
      },
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5174,
    proxy: {
      '/api/v1/auth': { target: 'http://127.0.0.1:8081', changeOrigin: true, secure: false },
      '/api/v1/users': { target: 'http://127.0.0.1:8081', changeOrigin: true, secure: false },
      '/api/v1/orders': { target: 'http://127.0.0.1:8082', changeOrigin: true, secure: false },
      '/api/v1/match': { target: 'http://127.0.0.1:8083', changeOrigin: true, secure: false },
      '/api/v1/messages': { target: 'http://127.0.0.1:8084', changeOrigin: true, secure: false },
      '/api/v1/payments': { target: 'http://127.0.0.1:8085', changeOrigin: true, secure: false },
      '/api/v1/reviews': { target: 'http://127.0.0.1:8086', changeOrigin: true, secure: false },
      '/api/v1/sos': { target: 'http://127.0.0.1:8087', changeOrigin: true, secure: false },
      '/api/v1/coupons': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/me/': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/hospitals': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/packages': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/virtual-numbers': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/address': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/escorts': { target: 'http://127.0.0.1:8089', changeOrigin: true, secure: false },
      '/api/v1/wallet': { target: 'http://127.0.0.1:8090', changeOrigin: true, secure: false },
      '/api/v1/admin': { target: 'http://127.0.0.1:8091', changeOrigin: true, secure: false },
      '/mockServiceWorker.js': {
        target: 'http://127.0.0.1:5174',
        changeOrigin: false,
      },
    },
  },
};