---
## 21. patient-miniapp 8 核心业务页接 API（patient-miniapp v1 plan §M1-M8）

**目标**：按 patient-miniapp v1 plan 落地 5 个新 API 模块 + 5 个新 Pinia store + 8 个真实业务页（首页 / 医院列表 / 医院详情 / 个人中心 / 优惠券中心 / 地址管理 / 评价创建 / 订单创建）。

**8 个 commit**：

| commit | 内容 | 新增测试 |
| :-- | :-- | :--: |
| `43fcd61` | `feat(patient-miniapp)` API client 5 个新模块 + index 聚合 | 38 |
| `5223b0d` | `feat(patient-miniapp)` Pinia 5 个新 store + loading/error | 41 |
| `6e022a0` | 首页（hospital 推荐 + 4 快捷入口 + 公告）| 9 |
| `b20b327` | 医院列表 + 医院详情（含服务包占位）| 15 |
| `4fd9581` | 个人中心（hero + 4 订单状态 tile + 设置 menu + 退出登录）| 10 |
| `c99dd20` | 优惠券中心（领券 + 我的券 tab + status 子 tab）| 7 |
| `8f880d8` | 地址管理 CRUD + 默认地址 + 评价创建（5 星）| 15 |
| `decf2d5` | 订单创建页（医院/服务包/时间/联系人/地址/优惠券 + 折扣计算）+ pages.json | 9 |

**关键设计**：

1. **API 模式**：`utils/request.js` 的 `request({ url, method, data, query })` + 缺参兜底 + 后端字段名 snake_case 最小透传
2. **store 模式**：`useXxxStore()` + `loading / error / data` + 动态 `import('@/api/xxx.js')`（与 v1.1 `stores/order.js` 一致）
3. **页面模式**：uView Plus `u-card` + `u-skeleton` + `u-empty` + `u-button` + `u-search` + `u-tabs` + `u-rate`（已通过 easycom 自动注册）
4. **测试模式**：jest.doMock 注入 fake store + uView Plus 全 stub

**累计测试用例**：18 个测试文件 / **144 个 it() 块**（API 38 + store 41 + page 65）。

**Plan 偏差**：

1. **服务包 `packages` 字段**：v1 后端 hospital 模块未返；`detail.vue` 与 `order/create.vue` 按 `packages=[]` 兜底渲染，v2 接 services/user/internal/pkg 后扩展
2. **`getMyProfile`**：用户信息走现有 `useAuthStore().fetchMe()` 通道（依赖 `@/api/auth.js`，v1.1 占位）
3. **订单创建页 `onSubmit`**：v1 仅做 toast + redirectTo（无后端 `POST /orders` 提交；order-service 已有 handler，后续 plan 接入即可）
4. **pages.json 路径约定**：原 v1.1 把 `src/pages/order/index.vue` 注册成 `pages/order/index`；新页沿用相同约定（uni-app 按 `src/pages/` 前缀解析）。未实际跑 uni-app build 验证
5. **vue-jest 未装**：测试是契约样，与项目既有约定一致

**未做（留给后续）**：

- 接入 `POST /orders` 真实创建订单（order-service 已有 handler，frontend `order/create.vue` onSubmit 改为真实提交）
- 接入 `services/user/internal/pkg` 的服务包 API
- 补 `src/api/auth.js`（`loginByPhone` + `fetchMe`）让 auth store 的 fetchMe 真实可达
- 跑 uni-app build 验证 pages.json 路径解析
- 引入 vue-jest 把 .vue 单测转为可执行

**端到端联通（v1.3 目标）**：

- patient-miniapp 8 核心页：首页（医院推荐 + 4 快捷入口）→ 医院列表 → 医院详情 → 订单创建（医院/服务包/时间/地址/优惠券）→ 选陪诊师（v1.1）→ 订单详情（v1.1）→ 服务执行 → 评价（新增）→ 个人中心 / 优惠券 / 地址管理
- 共享 X-Trace-Id：`mp-{ms}-{rand6}`（与 escort-app `escort-`、admin-web 不同）
- 后端 11 个 Go 服务 59 包 0 FAIL + patient-miniapp 144 测试
