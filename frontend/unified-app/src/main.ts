/**
 * unified-app 入口（v2 装配）。
 *
 * 装配链：
 *   - Pinia（auth store 注入）
 *   - uView Plus 全局组件
 *   - App.vue（路由 + home shell）
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