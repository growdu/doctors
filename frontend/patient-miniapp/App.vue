<script>
// App.vue — uni-app 根组件（不是普通 Vue 组件）
// 仅放全局生命周期钩子 + 全局样式；不写业务逻辑。
//
// Task 1~3 范围内只占位，后续 plan（Task 5 / Task 16）会扩展：
//   - installInterceptors 在 main.js 调，本文件不重复
//   - 平台条件编译 mock 支付回调（#ifdef APP-PLUS）
//   - 全局错误上报 / 埋点

export default {
  onLaunch() {
    // App 启动时跑一次（每个平台语义略不同：mp-weixin = 冷启动 / app-plus = 进程拉起）
    // eslint-disable-next-line no-console
    console.log('[patient-miniapp] onLaunch');
  },
  onShow() {
    // eslint-disable-next-line no-console
    console.log('[patient-miniapp] onShow');
  },
  onHide() {
    // eslint-disable-next-line no-console
    console.log('[patient-miniapp] onHide');
  },
};
</script>

<style>
/* 全局样式入口：后续 Task 2 引入 global.scss / variables.scss 在此追加 @import */
page {
  background-color: #f5f5f5;
  font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Helvetica Neue', sans-serif;
  font-size: 16px;
  color: #333;
}

/* uView Plus 主题变量可在 App.vue 覆盖，Task 2 完整化 */
</style>