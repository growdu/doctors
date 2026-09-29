// unified-app v2 · vite 配置（Phase 3.0.5 扩展）。
//
// 设计要点：
//   - ESM interop：@dcloudio/vite-plugin-uni 是 CJS，esModule 导入需穿透双层 default
//   - server.proxy：h5 dev 跨域路由到 11 个后端服务（auth :8081 / order :8082 / match :8083 / message :8084 / payment :8085 / review :8086 / sos :8087 / user :8088 / escort :8089 / wallet :8090 / admin :8091）
//   - alias '@' → src/
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
      // auth-service :8081 (默认 baseURL，auth/me/switch-role)
      '/api/v1/auth': { target: 'http://127.0.0.1:8081', changeOrigin: true, secure: false },
      '/api/v1/users': { target: 'http://127.0.0.1:8081', changeOrigin: true, secure: false },
      // order-service :8082
      '/api/v1/orders': { target: 'http://127.0.0.1:8082', changeOrigin: true, secure: false },
      // match-service :8083
      '/api/v1/match': { target: 'http://127.0.0.1:8083', changeOrigin: true, secure: false },
      // message-service :8084
      '/api/v1/messages': { target: 'http://127.0.0.1:8084', changeOrigin: true, secure: false },
      // payment-service :8085
      '/api/v1/payments': { target: 'http://127.0.0.1:8085', changeOrigin: true, secure: false },
      // review-service :8086
      '/api/v1/reviews': { target: 'http://127.0.0.1:8086', changeOrigin: true, secure: false },
      // sos-service :8087
      '/api/v1/sos': { target: 'http://127.0.0.1:8087', changeOrigin: true, secure: false },
      // user-service :8088
      '/api/v1/users/': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/coupons': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/me/': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/hospitals': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/packages': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/virtual-numbers': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      '/api/v1/address': { target: 'http://127.0.0.1:8088', changeOrigin: true, secure: false },
      // escort-service :8089
      '/api/v1/escorts': { target: 'http://127.0.0.1:8089', changeOrigin: true, secure: false },
      // wallet-service :8090
      '/api/v1/wallet': { target: 'http://127.0.0.1:8090', changeOrigin: true, secure: false },
      // admin-service :8091
      '/api/v1/admin': { target: 'http://127.0.0.1:8091', changeOrigin: true, secure: false },
      // mock service worker (admin-web MSW 兼容)
      '/mockServiceWorker.js': {
        target: 'http://127.0.0.1:5174',
        changeOrigin: false,
      },
    },
  },
};