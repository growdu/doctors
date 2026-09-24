// main.js
// uni-app 入口：createSSRApp + Pinia + uView Plus + 全局拦截器
//
// 范围：Task 1~3 初始化阶段。
//   - Pinia: Vue 3 配套状态管理（Task 7 才会真正落地 5 个 store，本文件先注册）
//   - uView Plus: 通过 app.use() 注入全局组件；easycom 已在 pages.json 配置
//   - installInterceptors: 全局请求/响应拦截（X-Trace-Id / Authorization / 11001 reLaunch）
//
// 后续 plan（Task 5/7/9）会追加 i18n / 全局错误边界 / 上报埋点。

import { createSSRApp } from 'vue';
import { createPinia } from 'pinia';

// uview-plus 2.x 的 Vue 3 plugin 入口。easycom 已负责按需注册组件，
// 这里 app.use() 主要作用是注册全局 $u / mixin / prototype。
// @ts-ignore - uview-plus 暂未发布完整 d.ts
import uviewPlus from 'uview-plus';

import App from './App.vue';
import { installInterceptors } from './utils/request.js';

// 1) 安装全局 HTTP 拦截器（必须在 createApp 之前，保证后续任何请求都走新逻辑）
installInterceptors();

/**
 * uni-app 标准 createApp 工厂（Vue 3 + createSSRApp）。
 * 平台分支：H5 走 SSR；mp-weixin / app-plus 客户端走 hydration。
 */
export function createApp() {
  const app = createSSRApp(App);

  // 2) Pinia
  const pinia = createPinia();
  app.use(pinia);

  // 3) uView Plus
  // @ts-ignore
  app.use(uviewPlus);

  return {
    app,
    Pinia: pinia,
  };
}