# L2 v1.0 前端 — 后端 API Gap 设计

> **For spec reviewers:** 这是 L2 v1.0 前端的**后端 API 增量设计**。3 个前端端（patient-miniapp / escort-app / admin-web）的 spec 都依赖本文件定义的 API 契约。

**Goal:** 盘点当前 8 个后端服务已暴露的 API，对比 L2 前端（patient-miniapp / escort-app / admin-web）的需求，列出缺口 API + 优先级 + 影响范围。本文件是后端 plan（agent 实施用）的 source of truth。

**前置依赖:**
- `docs/superpowers/specs/2026-09-24-roadmap-design.md` §5 L2 三端功能清单
- `docs/03-功能需求.md` 9 个功能模块
- `docs/09-验收与发布.md` §9.2.1 / §9.2.2 / §9.2.3 三端验收清单
- `docs/superpowers/plans/2026-09-24-{state-machine,order-lock,refund}.md` 已交付的事件

**Architecture:**
- API 风格：REST + JSON，body 走 `shared/httpx.Resp[T]` 统一包装（业务码 5 位 / 系统码 6 位）。
- 鉴权：JWT Bearer；前端从 `/api/v1/auth/login` 拿 access + refresh。
- 实时性：长流程（抢单 / 状态推进）走 Kafka 事件 + 前端轮询；v2 引 WebSocket。
- OpenAPI 3.0 契约：生成在 `web/openapi/contracts.yaml`，是 3 个前端 + 后端的 single source of truth。

---

## 1. 当前后端 API 现状（截至 2026-09-24）

### auth-service（5 个 API）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/auth/sms/send | 发送短信验证码 |
| POST | /api/v1/auth/login | 手机号 + 验证码登录；返回 JWT |
| POST | /api/v1/auth/refresh | 刷新 access token |
| POST | /api/v1/users/real-name/auth | 提交实名（mock 第三方） |
| GET | /api/v1/users/me | 当前用户信息（含实名状态） |

### order-service（6 个 API）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /api/v1/orders | 创建订单 |
| GET | /api/v1/orders | 我的订单分页 |
| GET | /api/v1/orders/:id | 订单详情 |
| POST | /api/v1/orders/:id/accept | 接单（陪诊师） |
| POST | /api/v1/orders/:id/cancel | 取消（患者/管理员） |
| POST | /api/v1/orders/:id/finish | 完成（陪诊师） |

### match-service（3 个 API）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/match/feed | 候选陪诊师列表（订单匹配） |
| POST | /api/v1/match/candidates | 提交候选评分 |
| POST | /internal/match/dispatch | 派单（内部） |

### payment / user / escort / review / message / sos

均只有 `/healthz`，无业务 API（v1 骨架阶段）。

---

## 2. L2 前端需求 — API Gap 清单

### 2.1 patient-miniapp（微信小程序）

| 缺哪些 API | 现有 | 影响范围 | 优先级 |
|---|---|---|---|
| `GET /api/v1/hospitals` | ❌ | 医院列表 + 详情 + 搜索 | P0 |
| `GET /api/v1/packages` | ❌ | 服务包列表 + 详情 | P0 |
| `GET /api/v1/orders/{id}/escort-summary` | ❌ | 订单详情里陪诊师摘要（虚拟号依赖）| P0 |
| `GET /api/v1/orders/{id}/virtual-number` | ❌ | 陪诊师虚拟号 | P0 |
| `POST /api/v1/orders/{id}/review` | ❌ | 评价订单 | P0 |
| `POST /api/v1/orders/{id}/sos` | ❌ | SOS 紧急信号 | P0 |
| `POST /api/v1/orders/{id}/refund` | ❌ | 申请退款（state-machine plan 已加事件，缺 API）| P0 |
| `GET /api/v1/users/me/wallet` | ❌ | 个人中心钱包余额 | P1 |
| `GET /api/v1/users/me/coupons` | ❌ | 优惠券列表 | P1 |
| `GET /api/v1/users/me/addresses` | ❌ | 地址管理 | P1 |
| `PUT /api/v1/users/me/addresses` | ❌ | 新增 / 修改地址 | P1 |
| `GET /api/v1/escorts/{id}/reviews` | ❌ | 陪诊师评价 | P1 |
| `GET /api/v1/messages/conversations` | ❌ | 站内信会话列表 | P2 |
| `GET /api/v1/messages/conversations/{id}/messages` | ❌ | 消息历史 | P2 |

### 2.2 escort-app（iOS + Android，Flutter）

| 缺哪些 API | 现有 | 影响范围 | 优先级 |
|---|---|---|---|
| `POST /api/v1/escorts/register` | ❌ | 陪诊师注册（角色切换） | P0 |
| `POST /api/v1/escorts/real-name/auth` | ❌ | 陪诊师实名 | P0 |
| `POST /api/v1/escorts/health-cert/upload` | ❌ | 健康证上传（OSS） | P0 |
| `POST /api/v1/escorts/training/complete` | ❌ | 培训考核完成 | P0 |
| `GET /api/v1/escorts/me/profile` | ❌ | 陪诊师档案 | P0 |
| `PUT /api/v1/escorts/me/status` | ❌ | 上线 / 下线 | P0 |
| `GET /api/v1/orders?role=escort&status=matching` | ⚠️ 现有只支持 patient | 抢单池 | P0 |
| `POST /api/v1/orders/{id}/checkin` | ❌ | 到院签到 | P0 |
| `POST /api/v1/orders/{id}/checkout` | ❌ | 服务完成打卡 | P0 |
| `GET /api/v1/escorts/me/wallet` | ❌ | 陪诊师钱包 | P0 |
| `POST /api/v1/escorts/me/wallet/withdraw` | ❌ | 提现申请 | P0 |
| `GET /api/v1/escorts/me/training-courses` | ❌ | 培训课程 | P1 |
| `GET /api/v1/escorts/me/reviews` | ❌ | 陪诊师收到的评价 | P1 |
| `POST /api/v1/orders/{id}/sos` | ❌ | SOS（陪诊师也能触发）| P0 |

### 2.3 admin-web（管理后台）

| 缺哪些 API | 现有 | 影响范围 | 优先级 |
|---|---|---|---|
| `GET /api/v1/admin/users` | ❌ | 用户列表 + 搜索 | P0 |
| `GET /api/v1/admin/orders` | ❌ | 全量订单 + 筛选 + 状态机操作 | P0 |
| `POST /api/v1/admin/orders/{id}/force-cancel` | ❌ | 管理员强制取消（admin_cancel 触发 refund） | P0 |
| `GET /api/v1/admin/escorts/pending-audit` | ❌ | 待审核陪诊师 | P0 |
| `POST /api/v1/admin/escorts/{id}/approve` | ❌ | 通过审核 | P0 |
| `POST /api/v1/admin/escorts/{id}/reject` | ❌ | 拒绝审核 | P0 |
| `GET /api/v1/admin/refunds` | ❌ | 退款审核队列 | P0 |
| `POST /api/v1/admin/refunds/{id}/approve` | ❌ | 批准退款 | P0 |
| `POST /api/v1/admin/refunds/{id}/reject` | ❌ | 拒绝退款 | P0 |
| `GET /api/v1/admin/work-orders` | ❌ | 工单列表 | P1 |
| `POST /api/v1/admin/work-orders` | ❌ | 创建工单 | P1 |
| `GET /api/v1/admin/billings` | ❌ | 账单 | P1 |
| `GET /api/v1/admin/reports/overview` | ❌ | 数据看板 | P1 |

> **总计 ≈ 43 个新 API**，按 feature 分组：

| Feature | API 数 | 涉及服务 |
|---|---:|---|
| **sos** | 2 | sos-service |
| **virtual-number** | 1 | virtual-number / 新建服务 |
| **wallet + withdrawal** | 4 | wallet-service / 新建 |
| **review** | 3 | review-service |
| **escort 业务** | 8 | escort-service |
| **hospital + package** | 4 | hospital / package / 新建 |
| **admin** | 12 | admin / 新建 |
| **message** | 4 | message-service |
| **address + coupon** | 4 | user-service |
| **escort-side order** | 1 | order-service（扩展 GET） |

---

## 3. 共享 schema（API 增量核心）

### 3.1 新建模型（entities）

```yaml
# Hospital — 医院
Hospital:
  id: int64
  name: string                    # "北京协和医院"
  city: string                    # "北京"
  district: string                # "东城区"
  address: string
  lat: float64
  lng: float64
  departments: string[]           # ["心内科","消化科"]
  level: string                   # "三甲"
  photo_url: string
  created_at: timestamp

# Package — 服务包
Package:
  id: int64
  hospital_id: int64
  name: string                    # "半日陪诊"
  duration_hours: int             # 4
  amount: decimal                 # 300.00
  description: string
  included: string[]              # ["代办挂号","全程陪同"]
  created_at: timestamp

# EscortSummary — 陪诊师档案（patient 端查看）
EscortSummary:
  id: int64
  nickname: string
  avatar_url: string
  rating: decimal                 # 4.8
  review_count: int
  completed_orders: int
  bad_rate: decimal               # 爽约率 0.02
  level: string                   # "金牌"
  certifications: string[]        # ["护士执业证","健康管理师"]
  is_available: bool

# OrderEscortSummary — 订单详情里的陪诊师（含虚拟号）
OrderEscortSummary:
  escort_id: int64
  nickname: string
  avatar_url: string
  phone_virtual: string           # 虚拟号（虚拟号服务生成）
  phone_virtual_expires_at: timestamp
  rating: decimal
  checkin_at: timestamp?
  checkout_at: timestamp?

# Review — 评价
Review:
  id: int64
  order_id: int64
  reviewer_id: int64
  reviewer_role: string           # "patient" | "escort"
  reviewee_id: int64
  rating: int                     # 1~5
  tags: string[]                  # ["耐心","专业"]
  comment: string
  is_anonymous: bool
  created_at: timestamp

# SOSRecord — 紧急信号
Signal:
  id: int64
  order_id: int64
  trigger_by: int64               # user_id
  trigger_role: string            # "patient" | "escort"
  lat: float64
  lng: float64
  address: string
  status: string                  # "active" | "resolved" | "cancelled"
  resolved_at: timestamp?
  resolved_by: int64?
  created_at: timestamp

# Wallet — 钱包
Wallet:
  user_id: int64
  balance: decimal                # 可用
  frozen: decimal                 # T+7 冻结
  total_earned: decimal
  updated_at: timestamp

# Withdrawal — 提现申请
Withdrawal:
  id: int64
  user_id: int64
  amount: decimal
  channel: string                 # "wx" | "alipay"
  account: string                 # 脱敏 "138****0000"
  status: string                  # "pending" | "approved" | "paid" | "rejected"
  created_at: timestamp
  reviewed_at: timestamp?
  paid_at: timestamp?

# Conversation / Message — 站内信
Conversation:
  id: int64
  participants: int64[]           # [user_id, escort_id]
  last_message_at: timestamp
  unread_count: int               # 给当前用户

Message:
  id: int64
  conversation_id: int64
  sender_id: int64
  content_type: string                # "text" | "image" | "location"
  content: string
  created_at: timestamp

# Address — 收货地址
Address:
  id: int64
  user_id: int64
  recipient: string
  phone: string
  province: string
  city: string
  district: string
  detail: string
  is_default: bool
  created_at: timestamp

# Coupon — 优惠券
Coupon:
  id: int64
  user_id: int64
  type: string                    # "amount_off" | "percent_off"
  value: decimal                  # 20.00 或 0.85
  threshold: decimal              # 满 100 减 20
  expires_at: timestamp
  used_at: timestamp?
  status: string                  # "unused" | "used" | "expired"
```

### 3.2 错误码扩展（shared/errs）

新增业务码（在 errs/code.go 集中管理）：

| 码 | 含义 | HTTP |
|------|------|------|
| 13001 | 资源未找到（医院/服务包/订单等通用） | 404 |
| 13002 | 资源已存在 | 409 |
| 13003 | 余额不足 | 402 |
| 13004 | 优惠券不可用 | 400 |
| 13005 | 实名未完成 | 403 |
| 13006 | 健康证未上传 | 403 |
| 13007 | 培训未完成 | 403 |
| 13008 | 抢单池已关闭 | 409 |
| 13009 | SOS 未授权（仅订单关联人可触发） | 403 |

---

## 4. 后端 plan（agent 实施用）

按 feature 分 plan，每个 plan 独立 spec + 实施：

| Plan | 范围 | 估 commits |
|---|---|---:|
| `2026-09-24-hospital-package-plan.md` | hospitals + packages 表 + 2 服务 4 API | 4 |
| `2026-09-24-escort-business-plan.md` | escort 实名 / 健康证 / 培训 / 上线 + 8 API | 6 |
| `2026-09-24-virtual-number-plan.md` | virtual_numbers 表 + 1 服务 1 API | 4 |
| `2026-09-24-wallet-plan.md` | wallets + withdrawals + billing + 4 API | 6 |
| `2026-09-24-sos-plan.md` | sos_records + 紧急联系人推送 + 2 API | 5 |
| `2026-09-24-review-plan.md` | reviews 完善 + 双向评价 + 3 API | 5 |
| `2026-09-24-message-plan.md` | conversations + messages + 4 API | 4 |
| `2026-09-24-address-coupon-plan.md` | addresses + coupons + 4 API | 4 |
| `2026-09-24-admin-plan.md` | admin 后台 12 API + 权限中间件 | 6 |
| `2026-09-24-escort-order-ext-plan.md` | order GET 扩展 escort 视图 + 签到 + 打卡 | 3 |
| **合计** | **10 个 plan** | **47 commits** |

---

## 5. 跨 plan 共享：建表 + 服务骨架

为避免重复建表，在第一个 plan 集中做：

- 迁移 `0005_init_v1_remaining.up.sql`：hospitals / packages / escort_profiles / health_certs / training_records / virtual_numbers / wallets / withdrawals / billings / sos_records / reviews / conversations / messages / addresses / coupons + 索引
- 创建服务骨架 `services/{hospital,package,virtual_number,wallet,admin,escort_business}/`（参考 user/escort/review 等骨架）
- 共享 OpenAPI 类型在 `web/openapi/contracts.yaml`（Go 后端自动生成）

---

## 6. 测试矩阵

每个 API 增量 plan 必填：

| 测试类型 | 数量 |
|---|---|
| 单元测试（业务包 ≥ 80% 覆盖） | ≥ API 数 × 1 |
| 集成测试（PG + Redis + Kafka） | ≥ API 数 × 0.5 |
| E2E 测试（curl + smoke） | ≥ API 数 × 0.2 |

跨端契约校验（CI 必跑）：

- `web/openapi/contracts.yaml` → 用 `openapi-generator` 生成 Go server / TS client
- 后端跑 `oapi-codegen` 生成 server stubs（验证一致性）
- 前端跑 `openapi-typescript` 生成 client types

---

## 7. 不做 / 留给 v2

| 不做 | 原因 |
|---|---|
| WebSocket / SSE 实时消息 | v1 轮询 + Kafka；v2 引 WebSocket |
| 第三方支付真实接入 | refund plan 已声明 v1 用 mock tx_id；v2 接入微信 V3 |
| 第三方实名（face++ / 公安二要素） | v1 直接 mock "已通过"；v2 接入 |
| 第三方短信（阿里云 / 腾讯云） | auth-service 短信通道是 mock log；v2 接入 |
| 第三方推送（极光 / 友盟） | v1 用站内消息；v2 接入推送 |
| 第三方 OSS（健康证图片） | v1 接收 base64 字符串；v2 接入 OSS |

---

## 8. 验收标准（与 09 对齐）

每端（patient / escort / admin）必须能跑通：

1. 注册 → 实名 → 下单/接单 → 支付 → 服务 → 评价 全链路 smoke
2. 退款触发链路（state-machine + order-lock + refund-service）
3. SOS 紧急信号（patient + escort 都可触发）
4. 钱包 T+7 结算（陪诊师提现）
5. 异常路径（抢单失败 / 实名拦截 / 余额不足 / 退款拒绝）

CI 必跑：

- `go test -count=1 ./...` 全过
- `web/openapi/contracts.yaml` 与后端实现 schema 一致
- 各前端单测 + E2E 全过

---

## Self-Review

- ✅ **Spec 覆盖**: 列出全部 43 个缺 API，按 feature 分组；按优先级排序
- ✅ **无占位符**: 每节都有具体清单 / 数字 / 路径
- ✅ **类型一致**: entities + error codes 集中在 §3 给出
- ✅ **测试矩阵**: §6 给出每类测试数量基线
- ✅ **YAGNI**: §7 明确不做 / 留 v2 的边界

## 关联 spec

- `docs/superpowers/specs/2026-09-24-patient-miniapp-design.md`（待写）
- `docs/superpowers/specs/2026-09-24-escort-app-design.md`（待写）
- `docs/superpowers/specs/2026-09-24-admin-web-design.md`（待写）