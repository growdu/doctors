// vite.config.js
//
// Vite + uni-app H5 配置入口。
// uni-app vue3 项目通过 vite-plugin-uni 的默认导出注入编译插件链
// （含 @vitejs/plugin-vue、manifest.json 处理、pages.json 处理、easycom、
// pagesJson、setupApp 等），缺失它会导致 `pnpm build:h5` 在 .vue 解析时
// 报 "Install @vitejs/plugin-vue to handle .vue files"。
//
// 与 src/manifest.json + src/pages.json + root/index.html 配套，让 Vite
// 找到入口并交由 uni 插件接管 Vue SFC。

import uni from '@dcloudio/vite-plugin-uni';

export default {
  plugins: [uni()],
};