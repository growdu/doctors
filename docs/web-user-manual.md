# unified-app web 用户手册

> **适用范围**：v2 unified-app 的 **web (h5) 部分**（即 `pnpm build:h5` 产物）
> **角色**：患者 / 陪诊师 / 管理员（含 6 个 admin 子角色）
> **入口 URL**：`http://<host>/`（docker 部署端口 80；dev 本地端口 5174）
> **关联**：
> - 架构文档 — `docs/web-design.md`
> - 实施回溯 — `dev.md §45-§46`

---

## 1. 概述

### 1.1 这是什么

一个 web 应用，同时支持三类用户：
- 🩺 **患者**：下单陪诊服务、付款、评价、查看钱包、申请退款
- 🚑 **陪诊师**：抢单、查看任务、上报签到/签出、查看/提现收入
- 🛡️ **管理员**：审批订单/退款/陪诊师、查看报表、处理工单

同一份代码、同一套登录 token、同一域切换器在三种身份间切换。

### 1.2 三类用户能做什么（速览）

| 能力 | 患者 | 陪诊师 | 管理员 |
| --- | :---: | :---: | :---: |
| 登录 / 角色切换 | ✅ | ✅ | ✅ |
| 浏览医院 / 服务 | ✅ | — | ✅ |
| 创建 / 支付订单 | ✅ | — | ✅（强制取消） |
| 抢单池 / 任务管理 | — | ✅ | ✅（审批） |
| 钱包 / 提现 | ✅ | ✅ | ✅（审批提现） |
| 评价 | ✅ | — | ✅（审核占位） |
| SOS 紧急工单 | ✅ | — | ✅（占位） |
| 数据报表 | — | — | ✅ |

### 1.3 不能做什么（明确标注）

| 位置 | 限制 |
| --- | --- |
| 管理员「创建优惠券」 | v2 后端暂未上线，点击弹 toast「功能开发中」 |
| 管理员「群发消息」 | 同上 |
| 管理员「评价审核」 | 同上（仍可跳 patient 域查看） |
| 管理员「SOS 工单列表」 | 同上 |
| 管理员「系统设置」 | 仅展示只读环境信息 |
| 陪诊师「GPS 签到」 | 后端无 `/checkin` 端点，前端用位置上报模拟 |
| 陪诊师「审核状态查询」 | 后端无查询端点，前端用 24h mock 倒计时 |
| 陪诊师「1:1 聊天」 | 后端 message-service 无 chat 端点，仅展示通知详情 |

---

## 2. 通用操作（适用于所有用户）

### 2.1 登录

#### 首次登录

1. 打开首页 `/`（未登录状态显示 SMS 登录卡）
2. 输入 **11 位手机号**
3. 输入 **6 位验证码**（验证码会发到后端 `auth-service` 控制台日志，dev fake：任意 6 位数字可通过）
4. 点击「登录」
   - 系统自动发短信 → 校验 → 写 token → 拉用户画像 → 持久化到 `localStorage`
5. 登录成功后自动显示「选择进入域」面板

#### 已有账号

直接打开首页，已登录态自动恢复（`auth.bootstrap()` 从 localStorage 读 token + user）。

### 2.2 切换激活角色（多角色用户）

适用：你同时有 `patient` + `escort` + 任意 admin 子角色。

```
入口：首页 → 「切换激活角色（N）」按钮
   ↓
弹出 RoleSwitcherModal，列出你所有角色
   ↓
点击目标角色
   ↓
系统调 POST /api/v1/auth/switch-role { active }
   ↓
新 JWT 替换 + 拉新 MeResponse + 持久化
```

**注意**：
- 切换仅改 `active_role`（激活角色），不影响 `roles[]`（你的全部角色）
- 切换成功后域名访问权按新 active_role 重新计算
- 单角色账号弹层会提示「无需切换」

### 2.3 切换访问域（DomainSwitcher）

适用：在已登录状态下，从一个域跳到另一个域。

```
入口：首页 → 「选择进入域」3 卡片（患者域 / 陪诊师域 / 管理后台）
   ↓
点击目标域卡片
   ↓
判断该域是否可用（roles[] 包含该域代表 role）
   ├─ 可用 → 跳 /pages/{patient,escort,admin}/index
   └─ 不可用 → 显示 🔒「未解锁」（greyed out）
```

**重要区别**：
- **DomainSwitcher** 只**跳页**，不**切换角色**
- **RoleSwitcherModal** 才**切换激活角色**
- 两者配合：你可以先切换角色到 admin，再点 DomainSwitcher 跳 admin 域

### 2.4 退出登录

```
首页 → 「退出登录」按钮
   ↓
auth.onUnauthorized() 清 token + user + 持久化
   ↓
reLaunch /pages/home/index（回到未登录态）
```

或**任意页面 API 返回 401** → 系统自动调 `onUnauthorized()` → 跳首页。

---

## 3. 患者域使用手册

### 3.1 入口

`/pages/patient/index`（4 快捷入口 + 推荐医院 + 推荐套餐）

### 3.2 主要功能

| 页面 | 路径 | 功能 |
| --- | --- | --- |
| **首页** | `/patient/index` | 4 快捷入口 + 推荐医院（前 3 个）+ 套餐瀑布流 |
| **选医院** | `/patient/hospitals/list` | 拉所有医院；按 city/keyword 过滤（前端） |
| **医院详情** | `/patient/hospitals/detail` | 医院信息 + 该院套餐列表 |
| **创建订单** | `/patient/order/create` | 选地址 + 时间 + 备注 + 优惠券 + 看候选陪诊师预览 → POST /orders |
| **支付** | `/patient/order/pay` | 选支付方式（默认 wallet）+ POST /payments + 扣钱包 |
| **订单列表** | `/patient/orders/index` | 4 状态 tab（全部 / 待接单 / 服务中 / 已完成 / 已取消） |
| **订单详情** | `/patient/order/detail` | 完整订单 + 状态机 + 取消按钮（pending/confirmed 时） |
| **抢单池** | `/patient/order/candidates` | patient 视角看订单的候选陪诊师（POST /match/candidates） |
| **钱包** | `/patient/wallet/index` | 余额 + 冻结 + 4 类型 tab 流水（全部 / 收入 / 提现 / 退款） |
| **优惠券** | `/patient/coupons/index` | 列出可领 + 跳详情；「领取」调 `/coupons/{id}/claim` |
| **消息** | `/patient/messages/index` | 4 tab（全部 / 系统 / 订单 / 支付）+ 未读小红点 |
| **通知** | `/patient/notifications/index` | 站内通知聚合 |
| **支持** | `/patient/support/index` | 客服联系 / FAQ 聚合入口 |
| **地址管理** | `/patient/address/list` + `/address/edit` | CRUD + 设为默认 |
| **个人中心** | `/patient/profile/index` | 头像/昵称 + 实名状态 + 退出 |
| **设置** | `/patient/settings/index` | 通知开关 / 隐私 / 关于 |
| **评价** | `/patient/reviews/{index,create}` | 我的评价 + 创建评价（订单完成后） |
| **SOS** | `/patient/sos/trigger` | 一键 SOS（POST /sos/trigger） |
| **退款申请** | `/patient/refund/apply` | 选订单 + 原因 + POST /payments/{id}/refund |

### 3.3 下单核心流程（一步一步）

```
1. 首页 → 「选医院下单」/ 或点某医院「预约陪诊」
        ↓
2. 选医院 + 套餐
        ↓
3. 创建订单页 (/order/create)：
   - 选地址（从已有地址选 / 跳地址编辑新建）
   - 选预约时间（文本输入 ISO8601）
   - 填备注（可选）
   - 看可用优惠券（自动匹配 ¥门槛）
   - 看候选陪诊师预览
        ↓
4. 点「提交订单」
   → POST /api/v1/orders { hospital_id, package_id, appointment_time, address, remark }
   → 跳转支付页
        ↓
5. 支付页 (/order/pay)：
   - 选支付方式（默认 wallet）
   - 点「确认支付」
   → POST /api/v1/payments { order_id, amount, method }
   → 钱包扣款（事务）
   → 跳订单详情
        ↓
6. 订单状态机自动流转：
   pending_escort → escort_confirmed（陪诊师接单）
                  → in_service（陪诊师开始服务）
                  → completed（陪诊师完成任务）
        ↓
7. 完成后 24h 内可评价
```

### 3.4 退款流程

```
订单详情 → 「申请退款」按钮（订单完成后 7 天内）
        ↓
/refund/apply 填原因
        ↓
POST /payments/{payment_id}/refund
        ↓
跳转「我的退款」列表（patient 域消息中心可见状态变更）
```

### 3.5 注意事项

- **实名认证**：高级操作（提现 / 大额订单）需先 `pages/patient/auth/real-name`
- **地址必填**：下单前必须有默认地址，否则 `pages/patient/address/edit`
- **钱包余额**：支付前会显示当前余额；不足时显示「余额不足」

---

## 4. 陪诊师域使用手册

### 4.1 入口

`/pages/escort/index`（3 入口卡片：抢单池 / 我的任务 / 个人中心）

> 首次登录需 SMS 验证码登录（与患者共用 auth-service）。escort 域也支持独立的 `/pages/escort/login` 入口。

### 4.2 主要功能

| 页面 | 路径 | 功能 |
| --- | --- | --- |
| **首页（工作台）** | `/escort/index` | 3 入口卡片 |
| **抢单池** | `/escort/invitations/index` | 拉候选订单（GET /match/feed）+ 接单/拒单 |
| **我的任务** | `/escort/orders/index` | 4 状态 tab（待服务/服务中/已完成/已取消） |
| **任务详情** | `/escort/order-detail` | 完整订单 + 「开始服务」/「完成任务」按钮 |
| **签到** | `/escort/checkin/index` | 拉 GPS + 上报位置（v2 mock） |
| **签出** | `/escort/checkout/index` | 填服务小结 + 完成订单 |
| **可用时段** | `/escort/availability/index` | 增删可接单时段（PUT /escorts/me/availabilities） |
| **培训记录** | `/escort/training/index` | 只读列表（GET /escorts/trainings） |
| **消息** | `/escort/message/{list,chat}` | 通知列表 + 详情 |
| **钱包首页** | `/escort/wallet/index` | 余额卡 + 4 tab 流水 |
| **提现申请** | `/escort/wallet/withdraw` | 金额 + 账号 + 账户名 → POST /escorts/me/wallet/withdraw |
| **个人中心** | `/escort/profile/index` | 基本信息 + 角色 + 退出 |
| **启动页** | `/escort/splash` | Auth Gate（已登录跳抢单池，否则跳 login） |
| **登录页** | `/escort/login` | SMS 登录 + 倒计时 60s |
| **审核中** | `/escort/audit-pending` | mock 24h 倒计时 + 返回首页 |

### 4.3 接单核心流程

```
1. 进入抢单池 /escort/invitations
        ↓
2. 看到候选订单列表（按 score + distance 排序）
   - 医院 ID + 预约时间 + 剩余席位 + 截止时间
        ↓
3. 选择「接单」或「拒单」
   ├─ 接单 → POST /api/v1/orders/{id}/confirm-accept
   │        → 订单状态: pending_escort → escort_confirmed
   │        → 自动从抢单池移除
   │
   └─ 拒单 → 弹 modal 收「拒单原因」
            → POST /api/v1/orders/{id}/reject-accept { reason }
            → 状态保持 pending_escort
        ↓
4. 跳「我的任务」 → 看到 escort_confirmed 订单
        ↓
5. 任务详情 /escort/order-detail
   - 「开始服务」→ POST /confirm-accept → in_service
   - 「完成任务」→ POST /finish → completed
        ↓
6. (可选) 签到 /escort/checkin 在到患者地点时
   - 系统拉 GPS → 调 UPDATE /escorts/me/location
   - mock 状态（v2 后端无专用 /checkin 端点）
        ↓
7. 签出 /escort/checkout
   - 填服务小结 → 触发 finish
```

### 4.4 设置可接单时段（重要）

陪诊师需先在 `/escort/availability` 设置可接时段才能被 patient 域抢单池看到。

```
/escort/availability
   ↓
点「+ 新增时段」
   ↓
输入 ISO8601 开始/结束时间 + 备注
   ↓
校验：end > start
   ↓
提交 → PUT /escorts/me/availabilities
```

### 4.5 提现流程

```
/escort/wallet → 点「提现」按钮
   ↓
/escort/wallet/withdraw
   - 输入金额（元，自动 ×100 转分）
   - 输入收款账号（银行卡 / 支付宝号）
   - 输入账户名
   - 校验：金额 ≤ 余额（balance > 0 时）
   ↓
提交 → POST /api/v1/escorts/me/wallet/withdraw
   ↓
toast「提现申请已提交」 + 跳回 /escort/wallet
```

### 4.6 注意事项

- **单角色用户**：escort 账号只能进 escort 域，不能进 patient/admin
- **审核中状态**：注册后未通过 admin 审核前，部分功能受限
- **可用时段**：未设置时段不会出现在 patient 抢单池

---

## 5. 管理员域使用手册

### 5.1 入口

`/pages/admin/index`（聚合占位页，显示欢迎 + 阶段说明）
实际业务从 `/pages/admin/dashboard/index` 仪表盘开始。

### 5.2 子角色权限矩阵

| 子角色 | 缩写 | 主要权限 |
| --- | --- | --- |
| `super_admin` | 超管 | 所有权限 |
| `order_admin` | 订单 | 订单管理 + 强制取消 |
| `refund_admin` | 退款 | 退款审批 |
| `audit_admin` | 审核 | 陪诊审核 |
| `cs` | 客服 | 工单管理 |
| `viewer` | 只读 | 所有 read-only（不可操作） |
| `finance_admin` | 财务 | 财务（v2 端点有限） |

切换到 admin 后，系统按 `active_role` 显示你能操作的按钮。`viewer` 看到所有按钮但不显示（隐藏）。

### 5.3 主要功能

| 页面 | 路径 | 功能 | 主要操作 |
| --- | --- | --- | --- |
| **仪表盘** | `/admin/dashboard/index` | 6 项指标卡（总订单/总金额/今日订单/今日 GMV/在线陪诊师/工单） | 点击跳对应列表 |
| **仪表盘详情** | `/admin/dashboard/detail` | 单指标展开 | — |
| **订单概览** | `/admin/orders/index` | 3 指标卡（待接单/服务中/已完成） | 跳列表 |
| **订单列表** | `/admin/orders/list` | 5 状态 tab + 拉所有角色订单 | 跳详情 |
| **订单详情** | `/admin/orders/detail` | 完整订单 + admin 专属操作 | 「强制取消」+ 输入原因 |
| **陪诊管理** | `/admin/escorts/index` | 陪诊师管理聚合入口 | 跳审核队列 |
| **陪诊审核** | `/admin/escorts/pending-audit` | 待审陪诊师列表 | 「通过」/「拒绝」+ 原因 |
| **陪诊详情** | `/admin/escorts/detail` | 陪诊师资料 + 资质材料 | — |
| **退款管理** | `/admin/refunds/index` | 3 状态卡 + 跳列表 | — |
| **退款列表** | `/admin/refunds/list` | 4 tab + 拉所有退款 | 「通过」+ 「驳回」+ 原因 |
| **退款详情** | `/admin/refunds/detail` | 完整退款 + 操作 | — |
| **用户管理** | `/admin/users/index` | 4 role tab + 列表 | 实名徽章 |
| **医院管理** | `/admin/hospitals/index` | 关键词搜索 + 展开套餐 | — |
| **套餐管理** | `/admin/packages/index` | 选医院 → 套餐列表 | — |
| **工单管理** | `/admin/work-orders/index` | 4 状态 tab + 「创建工单」 | — |
| **财务管理** | `/admin/finance/index` | 3 概览卡 + 4 类型流水 | — |
| **数据报表** | `/admin/reports/index` | 5 项核心指标 + 「导出报表」（占位） | — |
| **审计管理** | `/admin/audit/index` | 3 聚合入口 | 跳各审批页 |
| **优惠券管理** | `/admin/coupons/index` | **占位**：跳转 patient 域 | 「功能开发中」toast |
| **站内信管理** | `/admin/messages/index` | **占位**：跳转 patient 域 | 同上 |
| **评价管理** | `/admin/reviews/index` | **占位**：跳转 patient 域 | 同上 |
| **SOS 管理** | `/admin/sos/index` | **占位**：跳转 patient 域 | 同上 |
| **系统设置** | `/admin/settings/index` | 仅展示 version/env/login 状态 | — |
| **登录入口** | `/admin/login/index` | SMS 登录 + 6 demo 账号一键填充 | — |
| **个人中心** | `/admin/profile/index` | 用户卡 + 角色信息 + 退出 | — |

### 5.4 核心操作流程

#### 5.4.1 陪诊审核

```
/admin/escorts → 待审队列 /admin/escorts/pending-audit
        ↓
看到 PendingEscort 列表（id/nickname/phone/提交时间）
        ↓
点击「通过」/「拒绝」
   ├─ 通过 → 直接 POST /api/v1/admin/escorts/{id}/approve → 从列表移除
   │
   └─ 拒绝 → 弹 modal 输入原因（必填）
            → POST /api/v1/admin/escorts/{id}/reject { reason }
            → 从列表移除
```

#### 5.4.2 退款审批

```
/admin/refunds/list
        ↓
按状态 tab 过滤（pending/approved/rejected/all）
        ↓
对 pending 项点「通过」或「驳回」
   ├─ 通过 → 弹 modal 输入「审批备注」（可选） → POST /approve
   │
   └─ 驳回 → 弹 modal 输入「驳回原因」（必填） → POST /reject { reason }
```

#### 5.4.3 强制取消订单

```
/admin/orders → 订单列表 → 订单详情
        ↓
点击「强制取消订单」按钮（订单状态：pending_escort / escort_confirmed / in_service）
        ↓
弹 modal 输入「取消原因」
        ↓
POST /api/v1/admin/orders/{id}/force-cancel { reason }
        ↓
订单状态 → cancelled，刷新详情
```

> 注意：admin 强制取消与 patient 自助取消的 API 不同：前者走 admin-service 专用端点，后者走 order-service。

#### 5.4.4 创建工单

```
/admin/work-orders
        ↓
点击「+ 创建工单」
        ↓
弹 modal 选择类型（投诉/退款/咨询/其他）+ 标题 + 详细内容
        ↓
POST /api/v1/admin/work-orders
        ↓
工单插入列表首部
```

### 5.5 Admin demo 账号（开发用）

`/admin/login` 页面提供 6 个 demo 账号一键填充（避免 dev 时手输验证码）：

```
super_admin → 全部权限
order_admin → 订单 + 工单
refund_admin → 退款
audit_admin → 陪诊审核
cs         → 工单
viewer     → 只读
```

dev 环境 SMS 验证码任意 6 位数字可通过（`auth-service` mock）。

### 5.6 注意事项

- **多角色账号**：如果你同时是 patient + super_admin，可先在角色切换弹层选「超级管理员」+ DomainSwitcher 跳 admin 域
- **强制取消**：仅 pending_escort / escort_confirmed / in_service 三状态可强制取消；completed / cancelled 不显示按钮
- **占位页**：coupons / messages / reviews / sos / settings 5 页点击操作按钮会弹「功能开发中」toast + 自动跳 patient 域查看数据

---

## 6. 常见问题 FAQ

### Q1：登录后刷新页面就退出？

A：检查浏览器是否禁用了 localStorage。`auth.bootstrap()` 依赖 `localStorage.getItem('unified.auth')`。

### Q2：「401 未授权」自动跳首页？

A：token 过期或被踢出。重新登录即可。具体见 `src/api/client.ts:50-53`。

### Q3：admin 域点「创建优惠券」无反应？

A：v2 后端未上线该端点，是已知的占位（见 `pages/admin/coupons/index.vue:7`）。会跳 patient 域看数据。

### Q4：陪诊师看不到抢单池有订单？

A：检查是否设置可用时段（`/escort/availability`）。未设置时段的陪诊师不会进入 patient 的候选池。

### Q5：escort 域名用户能否看 patient 订单？

A：不能。3 域严格隔离（RoleGuard 拦截）。需切换 active_role 到 patient 才能看。

### Q6：dev 环境下 SMS 验证码是什么？

A：任意 6 位数字。`auth-service` 在 dev 模式直接返回 success（看 `services/auth/internal/handler/auth.go` 的 dev 分支）。

### Q7：docker 启动后访问 `localhost` 看不到 web？

A：unified-app 容器映射端口 80。检查 `docker ps` 是否 doctors-unified-app 已 up，`docker logs doctors-unified-app` 看启动日志。

### Q8：build:h5 报「@/api/xxx」找不到？

A：检查 `vite.config.js:13-17` 的 `@` alias 配置 + tsconfig.json 的 `baseUrl` / `paths`。

---

## 7. 联系反馈

- **架构问题** → 见 `docs/web-design.md` 的「已知占位 / Mock」清单
- **API 端点缺失** → 在 `dev.md` 或 `services/<name>/internal/handler/` 提 issue
- **前端 bug** → 看对应 page 的 `.spec.ts` 是否有覆盖；缺则补 test
- **设计决策** → 见 `docs/superpowers/specs/2026-09-28-unified-app-design.md`

---

## 8. 参考链接

| 用途 | 文档 |
| --- | --- |
| 架构与流程 | `docs/web-design.md` |
| 设计决策源 | `docs/superpowers/specs/2026-09-28-unified-app-design.md` |
| 实施回溯 | `dev.md §45-§46` |
| plan | `docs/superpowers/plans/2026-09-28-unified-app-v2.md` |
| 启动 + dev fake | `docs/01-项目概述.md` `docs/02-角色与权限.md` `docs/04-业务流程.md` |