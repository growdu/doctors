# patient-miniapp · 陪诊患者端

L2 v1.0 患者微信小程序 —— **uni-app + Vue 3 + uView Plus + Pinia** 骨架。

> 当前进度：**Task 1 ~ 3**（工程骨架 + manifest + uView Plus/Pinia + 拦截器）。
> 业务页面 / Pinia store / API 模块 / 测试框架 见后续 plan。

## 工程结构

```
frontend/patient-miniapp/
├── App.vue                    # uni-app 根组件（全局生命周期）
├── main.js                    # 入口：createSSRApp + Pinia + uView Plus + installInterceptors
├── manifest.json              # uni-app 应用清单（appid / app-plus 原生打包 / mp-weixin / h5）
├── pages.json                 # 路由声明 + easycom（uView Plus 自动注册）
├── package.json               # 依赖与 scripts
├── .gitignore                 # 忽略 node_modules / unpackage / dist / .hbuilderx
├── .npmrc                     # npm 镜像（npmmirror）
├── pages/
│   └── index/index.vue        # 占位首页（Task 3 落地）
└── utils/
    ├── auth.js                # token 持久化（uni.setStorageSync）
    └── request.js             # uni.request 封装 + 全局拦截器（X-Trace-Id / Authorization / 11001）
```

## 启动命令

```bash
# 1. 安装依赖（首次）
npm install

# 2. 开发态
npm run dev:h5              # H5 (Chrome 调试)
npm run dev:mp-weixin       # 微信小程序（微信开发者工具导入 dist/dev/mp-weixin）
npm run dev:app-plus        # 原生 App（Android Emu / iOS Sim）

# 3. 生产构建
npm run build:h5            # 输出 dist/build/h5
npm run build:mp-weixin     # 输出 dist/build/mp-weixin
npm run build:app-android   # 输出 dist/build/app-plus（Android APK，debug 签名）
npm run build:app-ios       # 输出 dist/build/app-plus（iOS IPA，debug 签名）

# 4. 类型检查
npm run typecheck
```

## 全局拦截器（utils/request.js）

| 阶段 | 行为 |
|---|---|
| 请求 | 自动加 `X-Trace-Id`（`mp-<ts13>-<rand6>`） + `Authorization: Bearer <token>` |
| 响应 | 业务码 `code === 0` 视为成功；其它一律抛 `ApiError` |
| 错误 | HTTP 401 / 业务码 `11001`（未登录）→ 清 token + `uni.reLaunch('/pages/auth/login')` |

## 后端对接

API baseURL：`https://api.dev.doctors.example.com/api/v1`（默认，可在 `utils/request.js` 顶部 `DEFAULT_BASE_URL` 覆盖）。

对接清单见 `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.1（共 14 个 P0 API）。

## 后续 plan

| Plan | 内容 |
|---|---|
| Task 4 | utils 层（trace / format / wx / uni-mock）+ 单测 |
| Task 5 | API client（trace_id / 401 重定向 / refresh） |
| Task 6 | 9 个 API 模块骨架 |
| Task 7 | Pinia 5 个 store（auth / order / hospital / wallet / message） |
| Task 8 | 8 个组件骨架 |
| Task 9 | i18n 占位 + Jest + Playwright e2e |
| Task 10 | OpenAPI codegen |
| Task 11 | GitHub Actions CI |
| Task 12 | 文档同步 |
| Task 13 | 选陪诊师流程实现 |
| Task 16~19 | v1.1 多端原生构建（Android / iOS） |

## 注意事项

- `manifest.json` 中 `appid` 暂为 `TOURIST_APPID` / `TOURIST_WX_APPID` 占位，上线前替换为真实 appid。
- app-plus 图标 / 启动图资源（`static/icons/*.png` / `static/splash/*`）在 Task 16 落地。
- 不要 push 到远端（本地 SSH 不通）；commit 后留用户手动 push。