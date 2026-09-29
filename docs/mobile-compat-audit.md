# unified-app 移动端兼容审计（mp-weixin / app-plus）

> **触发**：v2.0.0 已发布 web (h5)，用户要求"完善移动端代码"
> **范围**：74 page / 9 通用组件 / 11 API client / 3 store / 1 manifest / 1 pages.json
> **目标**：识别 web-only 代码，补齐 mp-weixin（微信小程序）+ app-plus（Android/iOS）的运行配置

---

## 1. 现状汇总（2026-09-29 扫描）

### 1.1 配置层

| 文件 | 现状 | 缺口 |
| --- | --- | --- |
| `src/manifest.json` | appid 占位 `__UNI__UNIFIED`；mp-weixin.appid `wxPLACEHOLDER` | 真实 appid、ios distribute、icons、launch image、modules、requiredPrivateInfos |
| `src/pages.json` | 74 路由，仅设 navigationBarTitleText；无 top-level `conditionCompile`；无 tabBar | app-plus / mp-weixin 条件编译（navigationStyle、pullRefresh、share、disableScroll）；tabBar（移动端底栏） |
| `src/uni.scss` | 全局工具类（page/card/title）已内联 | — |
| `vite.config.js` | dev proxy + alias；CSS preprocessor loadPaths | — |

### 1.2 代码层（web-only 热点）

| 位置 | 行号 | 问题 | 移动端兼容性 |
| --- | --- | --- | --- |
| `src/App.vue` | 27-32 | 用 `document.createElement('style')` 注入 `:root` CSS 变量 | mp-weixin / app 无 document，全局样式需走 `uni.scss` 预编译 |
| `src/main.ts` | 19 | `createSSRApp`（h5 模式需要 SSR，mp-weixin 兼容，app-plus 兼容但不需要） | 全兼容但需明确 SSR vs 非 SSR |
| `src/store/auth.ts` | 35-53 | 已有 `typeof uni !== 'undefined' ? uni.* : localStorage.*` fallback | ✅ 已修复 |
| `src/pages/patient/settings/index.vue` | 37, 56 | `globalThis.localStorage?.getItem/setItem` | mp-weixin 无 localStorage（用 `uni.setStorageSync`），需 fallback |
| `src/api/client.ts` | 29, 40 | `process.env.UNI_BASE_URL \|\| 'http://127.0.0.1:8081'` + `fetch` | mp-weixin 不能用 `fetch`，需用 `wx.request` 或保持 fetch（uni-app 已 polyfill）；app 端 fetch 正常 |
| `src/components/shared/RoleGuard.vue` | 41 | `<u-button>` 引用 uView Plus | ✅ uView Plus 已配置 easycom |
| `src/test/smoke.spec.ts` | 4 处 | `window.localStorage` | 测试代码 OK（happy-dom 环境） |

### 1.3 API 调用点统计

```
uni.* API 总调用点：148
- getStorageSync: 4
- setStorageSync: 3
- removeStorageSync: 2
- navigateTo: 14（最频繁）
- reLaunch: 12
- showToast: 18
- switchTab: 2
- navigateBack: 7
- getCurrentPages: 14
- getLocation: 1（escort/checkin）
- setClipboardData: 1
- getSystemInfoSync: 0
```

### 1.4 端差异优先级

| 优先级 | 项 | 说明 |
| --- | --- | --- |
| P0 | App.vue `document.*` | 移动端会 throw，必须条件编译 |
| P0 | patient/settings `localStorage` | 小程序崩溃 |
| P1 | manifest.json ios / app-plus | 没有 ios config 编译 ios 报错 |
| P1 | manifest.json appid | 提交审核需要真实 appid |
| P1 | pages.json tabBar | 移动端底栏导航是主流交互 |
| P2 | pages.json navigationStyle: custom | 沉浸式导航栏需要每页显式声明 |
| P2 | utils/storage.ts 抽象 | 重构减少 web-only 代码蔓延 |
| P3 | utils/share.ts | 移动端分享能力（patient 订单分享给陪诊师、escort 抢单分享） |
| P3 | utils/callPhone.ts | SOS 触发后小程序一键拨号 |

---

## 2. 实施批次

> **进度状态（2026-09-29 更新）**：批 1-4 代码全部完成（含 22 spec cases 验证），批 5 实际构建未跑（依赖微信开发者工具 CLI + HBuilderX 离线打包环境），改为 `mobile-integration.spec.ts` 跨模块静态分析 spec 替代。

### 批 1：基础设施（条件编译 + 抽象层）✅ 已完成
- 修 App.vue 的 `document.*` → `// #ifdef H5`
- 修 main.ts：h5 用 SSR，app 用非 SSR
- 修 patient/settings：localStorage fallback uni API
- 新建 `src/utils/storage.ts`：抽象 set/get/remove
- 重构 auth.ts、settings/index.vue 使用 utils/storage.ts

### 批 2：manifest.json 补全 ✅ 已完成（2a + 2b 两个 commit）
- 真实 appid 占位说明（README 标注待替换）
- ios distribute：bundleId、version、requiredDeviceCapabilities、infoPlist NSAppTransportSecurity
- app-plus icons + splash（5+ 图标尺寸）
- mp-weixin appid + setting.es6 + optimize
- app-plus modules: Push、Share、OAuth（可选）
- requiredPrivateInfos（定位 / 通讯录）

### 批 3：pages.json 移动端配置 ✅ 已完成（3a + 3b 两个 commit）
- top-level conditionCompile（mp-weixin / app-plus / h5 条件块）
- tabBar 配置（移动端底栏）
- 关键页 `style.navigationStyle: custom`（沉浸式）
- 关键页 `style.enablePullDownRefresh: true`
- 关键页 `style.app-plus.titleNView`（原生导航栏配置）
- 关键页 mp-weixin `disableScroll: true`

### 批 4：移动端特有 page / 组件 ✅ 已完成（4a + 4b + 4c + 4d 四个 commit）
- `src/pages/patient/order/share.vue`：分享订单详情给陪诊师（app-plus + mp-weixin）
- `src/utils/share.ts`：uni.share API + h5 navigator.share fallback
- `src/utils/callPhone.ts`：uni.makePhoneCall + h5 tel: link
- SOS trigger 加「一键拨号 120」按钮

### 批 5：构建验证 ⚠️ 代码层完成（mobile-integration.spec.ts 12 cases）；实际 build 待外部工具

**已完成**：
- `src/utils/mobile-integration.spec.ts`：12 个跨模块静态验证 case
  - 各 utils 模块导出完整性（share / share.mp-weixin / callPhone / storage 4 模块）
  - 页面正确引用 utils（source grep）
  - web-only 残留清理（auth.ts 不再有 typeof localStorage/window/document）
  - App.vue `#ifdef H5` 包裹 `document.*` 验证
  - manifest.json / pages.json 路径前缀一致性

**未完成（依赖外部工具链，不在本机）**：
- `pnpm build:mp-weixin`：需要微信开发者工具 CLI（未安装）
- `pnpm build:app`：需要 HBuilderX 离线打包 + Android SDK（未安装）
- 不实际跑 HBuilderX 离线打包（需要 macOS + Xcode + Android SDK）

**图标资源待补（PNG 文件）**：
- `static/icons/ios/*.png`（11 个 iOS 图标尺寸）
- `static/icons/android/ic_app_*.png`（6 个 DPI）
- `static/icons/tabbar/*.png`（8 个 tabBar 图标）
- 设计师出图后直接放进 `frontend/unified-app/src/static/icons/`，无需改配置（路径已在 manifest.json / pages.json 引用）

---

## 3. 与 v1 移动端功能对照

| v1 功能 | v2 web 是否实现 | v2 移动端目标 |
| --- | :---: | :---: |
| 微信小程序登录（wx.login） | ❌（用 SMS） | ✅ 保留 SMS（统一登录） |
| 小程序分享给好友（onShareAppMessage） | ❌ | ✅ 新建 patient/order/share |
| 一键拨号（SOS） | ❌ | ✅ SOS trigger 加 callPhone |
| 微信支付 | ❌（v2 用钱包） | — |
| 位置授权弹窗（wx.getSetting） | ⚠️ fallback 默认 | ✅ callPhone/location 加权限检查 |
| App 推送（Push） | ❌ | ⏳ 留 v3 |
| 分享到朋友圈 | ❌ | ⏳ 留 v3 |
| 二维码扫码（scanCode） | ❌ | ⏳ 留 v3 |

---

## 4. 不在本次范围

- HBuilderX 离线打包（需要 macOS + Xcode + Android SDK，本会话无法验证）
- 微信开发者工具真机调试（需要 mp-weixin 编译产物 + IDE 打开）
- App Store / Google Play 提交审核（需要真实 appid + 签名）
- 推送服务对接（Push 模块 v3 计划）

---

## 5. 参考

- uni-app 跨端条件编译：`https://uniapp.dcloud.net.cn/tutorial/platform.html`
- uni-app manifest.json：`https://uniapp.dcloud.net.cn/tutorial/app-config.html`
- pages.json tabBar：`https://uniapp.dcloud.net.cn/tutorial/page-json.html`
- uni API 文档：`https://uniapp.dcloud.net.cn/api/`