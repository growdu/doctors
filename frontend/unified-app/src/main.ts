/**
 * unified-app 入口（v2 装配 · 移动端兼容）。
 *
 * 装配链：
 *   - Pinia（auth store 注入）
 *   - App.vue（路由 + home shell）
 *
 * 端兼容：
 *   - h5        : createSSRApp（需要 SSR 配合 history router）
 *   - mp-weixin : createSSRApp（uni-app x vue3 默认）
 *   - app-plus  : createApp（不需要 SSR，避免 nvue 风格冲突）
 *
 * 跨域路由守卫：
 *   - /pages/patient/**：require roles 含 patient
 *   - /pages/escort/**：require roles 含 escort
 *   - /pages/admin/**：require roles 含 admin_*
 *
 * 对应 spec：2026-09-28-unified-app-v2.md §2.5
 */
import { createSSRApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';

export function createApp() {
  const app = createSSRApp(App);
  const pinia = createPinia();
  app.use(pinia);
  return { app, Pinia: pinia };
}