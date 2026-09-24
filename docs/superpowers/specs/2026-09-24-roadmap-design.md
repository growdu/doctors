# 陪诊师平台 v1.0~v3.0 全量路线图设计

> **日期**：2026-09-24
> **状态**：设计稿（待 review）
> **范围**：本 spec 是**总路线图**，不写任何代码；为后续每一份 feature spec 锚定范围、原则、切片方式与依赖关系。
> **关联**：docs/01~09 需求、docs/REVIEW-REPORT.md 评审报告、docs/superpowers/plans/2026-09-23-doctors-v1.md 已交付阶段 0~4 + 6 个 v1 骨架服务。

---

## 1. 背景与目标

后端 MVP（auth / order / match + 6 个 v1 骨架服务：user / escort / review / message / sos / payment）已经交付到 origin/main。但 `docs/REVIEW-REPORT.md` 指出 v1.0 距离可投产还有 **3 类硬伤**：

- **跨文档不一致**：状态机在 04 vs 07；API 路径在 06 vs 08；P0 范围在 03 vs 09。
- **核心业务规则颗粒度不足**：抢单并发 30s 锁单无落地；退款分段未给出；虚拟号 / SOS / 实名 / T+7 结算期退款处理细节缺失。
- **合规 / 运营支撑章节缺位**：埋点、客服 SLA、灰度回滚 SOP、AB 实验、数据迁移、灾备演练、保险接入无文档。

此外 docs/06 描述了完整三端（iOS/Android/小程序/管理后台）和 11 个业务域，而当前项目**完全没有前端代码**，且 6 个 v1 骨架服务业务逻辑大多为空。

**目标**：在不写代码的前提下，产出**可执行的总路线图**：每个 feature 一份 spec + 一份 plan，每份 plan 都是后续可以"按节奏 commit"的具体任务清单。

**非目标**（本期不做）：

- 不写任何生产代码
- 不重新评审已实现的后端（auth/order/match + 6 骨架）
- 不引入 docs/ 之外的新功能
- 不引入未评审的新技术栈

---

## 2. 设计原则（贯穿所有 spec）

| 原则 | 落地方式 |
| :-- | :-- |
| **高内聚低耦合** | 每个 feature 的 inbound / outbound contract 严格定义；跨服务通信全部走 `shared/contracts` 单一来源；业务规则不跨 service 包边界。 |
| **可扩展** | 状态机、抢单渠道、退款策略做成"字符串转换表 + 策略模式"；新增状态 / 渠道不改核心代码。 |
| **可测试** | 每个 service 必含单测（业务包 ≥ 80%）+ 集成（DB / Kafka / Redis，tag 隔离）+ e2e（smoke 脚本）。handler 层只做绑定；业务逻辑 100% 走 fake 替身。 |
| **可维护** | 每个大决策写到 `docs/superpowers/decisions/ADR-NNN-*.md`；CI 必跑 lint + vet + test；不引入新依赖需在 plan 里说明。 |
| **易用** | 每个 feature 给一段 PM 能跑的"演示脚本"；错误码统一在 `shared/errs`；OpenAPI 自动生成；trace_id 全链路贯穿。 |

---

## 3. 切片与组织

### 3.1 切片单位

每个 **feature** = 一个独立可交付单位，对应 **1 份 spec + 1 份 plan**。一个 feature 通常：

- 跨 1~3 个服务（如"退款"涉及 order + payment + billing 三方）
- 跨 1~2 端（如"评价双向"涉及患者小程序 + 陪诊 App + 管理后台）
- 有明确入口与出口契约

### 3.2 路线图分层（8 层）

| 层 | 内容 | 阶段 | 切片方式 |
| :-- | :-- | :-- | :-- |
| **L0** 基础设施 | CI / CD / 可观测 / 灰度 / 部署 / OpenAPI 文档统一 | 全阶段渐进 | per-feature |
| **L1** v1.0 后端收敛 | 状态机统一 / 抢单锁单 / P0 业务域填实（refund / sos / wallet / work-order / virtual-number / bid-review） | v1.0 MVP 阻塞项 | per-feature |
| **L2** v1.0 前端 | 患者小程序 + 陪诊 App + 管理后台（先用 mock 后端打通） | v1.0 MVP 阻塞项 | per-feature × 端 |
| **L3** v1.0 真实集成 | 微信支付沙箱 / 短信 / 实名 / 推送 / 虚拟号 | v1.0 上线前 | per-feature（每个三方一个） |
| **L5** v1.1 | 优惠券 / 抢单池优化 / 工单 / 数据看板 | v1.1 | per-feature |
| **L6** v1.2 | 拉新活动 / 推送体系 / 行程上报 | v1.2 | per-feature |
| **L8** v2.0 | AI 匹配 / 医院深度合作 / 保险接入 | v2.0 | per-feature |
| **L9** v3.0 | 多区域 / 海外 | v3.0 | per-feature |

**为什么不按版本（v1.0 / v1.1 ...）切片？** 单个版本含多个 feature，跨版本做依赖追踪困难；按 feature 切片能确保每个 spec 是**自包含**的、可独立排期。

### 3.3 时间线（来自 docs/09.9）

| 版本 | 主要内容 |
| :-- | :-- |
| v1.0 | MVP：三端基础流程 + 后台核心 |
| v1.1 | 优惠券、抢单池、虚拟号、客服工单、数据看板 |
| v1.2 | 拉新活动、推送体系、行程上报 |
| v2.0 | AI 智能匹配、医院深度合作、保险接入 |
| v2.5 | 健康档案、积分商城、城市经理分级 |
| v3.0 | 多区域、海外 |

v2.5 暂未在本路线图展开（不在 docs/07 数据模型范围内），留作未来追加。

---

## 4. L1 v1.0 后端收敛 — 第一波详细 feature

每个 feature 一份 spec。L1 是当前最紧迫的（评审报告 Critical 全部归属）。

### 4.1 状态机统一

**目的**：解决 C-01（04 vs 07 状态机不一致）+ 引入评审要求的中间态。

**新增/调整状态**：

```
created → paid → matching → pending_acceptance → accepted → in_service
                                                                   ↓
                                                  (5min 不签到回退 matching)
                                                  ↓
                                                                completed → reviewed → closed
                                                                ↘
                                                                 refunding → refunded → closed
                                                                          ↓
                                                                  (中间) settling → closed
                                                                                ↓
                                                                  (中间) disputed → closed
```

**新增字段**：`orders.lock_owner` (BIGINT, escort_id)、`orders.lock_expire_at` (TIMESTAMPTZ)、`orders.version` 已在用。

**影响文件**：

- `services/order/internal/state/machine.go`：加 `pending_acceptance` / `settling` / `disputed`；`accepted → matching` 是合法的"回退"
- `migrations/0002_orders.up.sql`：加 lock_owner / lock_expire_at；CHECK 约束加新状态
- `services/order/internal/repo/order_repo.go`：增加 `LockForAccept` / `ReleaseLock` 方法
- `services/order/internal/service/accept.go`：引入 30s 锁单窗口
- `services/match/internal/pool/pool.go`：候选 TopN 与抢单状态联动

### 4.2 抢单锁单 30s（解决 C-04）

**目的**：高并发 N 个陪诊师抢同一单，恰好 1 个胜出 + 锁单 30s，期间其它人不能抢；超时回退 matching。

**三道防线**：

1. **状态机**：新增 `pending_acceptance` 中间态；只有 `matching` 可进 `pending_acceptance`
2. **Redis SETNX**：`orders:accept-lock:{id}` 30s TTL；SETNX 失败直接 409
3. **DB 唯一约束**：`(order_id, escort_id)` 在 order_locks 表唯一；过期定时任务清理

**伪代码**：

```go
// accept.go 核心逻辑
// 1. SETNX attempts: 在 Redis 设 30s 锁
// 2. SELECT FOR UPDATE on orders WHERE id=$1 AND status='matching'
// 3. UPDATE orders SET status='pending_acceptance', lock_owner=$2, lock_expire_at=NOW()+30s WHERE id=$1 AND version=$3
// 4. 异步任务：30s 后检查 lock_expire_at 已过 → UPDATE 回退 status='matching', lock_owner=NULL
// 5. notifyEscort(escort_id, "请在 30s 内确认接单")
```

**影响**：参考 L1.4 引入新的 `OrderLockScheduler` 异步任务。

### 4.3 退款分段

**目的**：解决 C-04 + 评审 I-05 "退款分段留扩展点"。

**退款阶梯**：

| 触发点 | 退款比例 | 路径 |
| :-- | :-- | :-- |
| 患者付款后 5 分钟内取消 | 100% | 微信支付原路退 |
| 5 分钟 ~ 陪诊师接单前 | 100% | 同上 |
| 陪诊师接单后、服务开始前 | 95%（陪诊师补偿 5%） | 同上 |
| 服务已开始 | 0% 退款，需走投诉流程 | 走 dispute 服务 |

**扩展点**：退款比例由 `refund_policies` 表配置；支持按"取消时机 × 距离服务开始时长"做策略。

### 4.4 SOS 一键报警（评审 P0）

**触发**：订单 status=in_service 时，患者/陪诊师 App 长按 SOS 按钮 3s。

**流程**：

```
SOS 按钮 → sos-service.Raise()
  → 入库 sos_records (lat/lng/note)
  → 发 SOSRaisedEvent
  → 通知紧急联系人（短信 + 推送）
  → 通知管理后台（SOS 工单优先级 P0）
  → 自动录音（虚拟号通道录音）
  → 30s 内无响应 → 通知警方（v2.0）
```

**幂等**：同订单 5 分钟内只发一次（避免误触）。

### 4.5 虚拟号

**目的**：保护双方真实手机号。

**实现**：集成第三方虚拟号 API（如阿里云隐私号）。订单详情页展示虚拟号；拨打虚拟号自动转接到双方真实号。

**回写**：`orders.virtual_number VARCHAR(20)`；服务开始时分配；服务结束释放。

**录音**：通话期间全程录音（评审 I-08 合规要求）；存 OSS；7 天后自动归档。

### 4.6 钱包 T+7 结算

**目的**：解决 B-05 资金流。

**模型**：

```
billings:
- id / order_id / escort_id / amount / status
- status: 'pending' → 'frozen'（服务完成时）→ 'available'（+7 天）→ 'withdrawn' / 'refunded'
- frozen_at / available_at / withdrawn_at
```

**流程**：

1. 订单 `completed` → 写 billings (frozen)
2. 7 天后定时任务 → billings.status='available'
3. 陪诊师申请提现 → billings.status='withdrawing' → 调微信企业付款 → billings.status='withdrawn' / 'failed'
4. 提现失败回退 'available'

**风险**：T+7 期间退款怎么办？→ 退款时 billings.status='refunded'，从 frozen 池里扣减。

### 4.7 工单 + 客服首响 SLA

**目的**：评审 I-09 客服 SLA 监控。

**模型**：

```
work_orders:
- id / order_id / user_id / category / priority / status
- first_responded_at / resolved_at / sla_deadline
- status: 'pending' → 'assigned' → 'in_progress' → 'resolved' / 'closed'
- priority: 'P0' (SOS) / 'P1' (退款) / 'P2' (一般)
```

**SLA**：

| 优先级 | 首响 | 解决 |
| :-- | :-- | :-- |
| P0 | 30s | 1 小时 |
| P1 | 5 分钟 | 4 小时 |
| P2 | 1 小时 | 24 小时 |

**埋点**：每个工单创建时上报 `work_order.created{priority}` metric；sla_deadline 用 prometheus alarm。

### 4.8 双向评价

**目的**：C-09 双向匿名评价。

**已有**：review-service.CreateReview（patient → order）。

**新增**：

- escort → patient 评价：`reviews.reviewer_role='escort'`
- 双方评价完成后 → orders.status='reviewed' → 24h 后 'closed'
- 评价显示阶段：服务端聚合 + 匿名 7 天后公开

---

## 5. L2 v1.0 前端 — 第二波 feature

每个端一份 spec；每个端用 **mock 后端优先**打通端到端，再切真后端。

### 5.1 患者小程序（微信）

**功能清单**（来自 09.2.1）：

- 注册 / 登录（手机号 + 微信）
- 实名认证（mock）
- 浏览医院 / 服务包（筛选 + 搜索）
- 下单 → 微信支付（沙箱） → 支付成功
- 订单详情展示陪诊师虚拟号
- 申请退款 → 自动原路退回
- 服务完成 → 评价 → 双向评价展示
- 个人中心、优惠券、地址管理
- SOS 一键报警

**技术栈**：uni-app（Vue 3） + uView Plus；状态管理 Pinia；HTTP uni-request（拦截器加 trace_id）。

**目录**：`web/patient-miniapp/`。

### 5.2 陪诊师 App（iOS + Android）

**功能清单**（09.2.2）：

- 注册 → 实名 → 上传健康证 → 培训考核 → 协议签署
- 提交审核 → 管理员审核 → 通过
- 上线后进入抢单池 / 可被派单
- 接单 → 到院签到（GPS 校验）→ 打卡 → 结束服务
- 钱包可见收入 / T+7 冻结 / 提现申请
- 培训课程学习 / 考核通过有记录

**技术栈**：React Native（iOS + Android 共享）；push 用 Firebase / 极光。

**目录**：`mobile/escort-app/`。

### 5.3 管理后台 Web

**功能清单**（09.2.3）：

- 仪表盘真实指标
- 医院 / 服务包 / 优惠券 CRUD
- 陪诊师审核工作流
- 订单监控 + 改派 / 退款
- 财务对账 + 提现审核
- 工单 + 客服首响 SLA 监控

**技术栈**：Vue 3 + Element Plus + Vite + Pinia。

**目录**：`web/admin/`。

---

## 6. L3 v1.0 真实集成

每个三方一个 spec。每个集成方需先在 dev 环境用真实数据迁移，集成后再切 staging / 沙箱。

| feature | 集成商 | 沙箱 / 真渠道 |
| :-- | :-- | :-- |
| 微信登录 | 微信开放平台 | 沙箱 |
| 微信支付 V3 | 微信支付 | 沙箱（商户号） |
| 实名 | 公安二要素 / 旷视 | mock → 沙箱 → 真 |
| 短信 | 阿里云 / 腾讯云 | dev mock → 真 |
| 推送 | 极光 / Firebase | dev mock → 真 |
| 虚拟号 | 阿里云隐私号 | dev mock → 真 |
| 地图 | 高德 / 腾讯 | 沙箱 |
| 录音 | 七牛 OSS | 真 |

---

## 7. L0 基础设施（全阶段渐进）

| feature | 范围 |
| :-- | :-- |
| CI | GitHub Actions：lint + vet + test + build |
| CD | 镜像构建 + staging/prod 灰度；Helm chart |
| 可观测 | TraceID 全链路（OpenTelemetry）；metrics（Prometheus）；log（结构化 JSON） |
| 灰度 | 城市 / 用户标签 / App 版本三维灰度；秒级回滚 |
| OpenAPI 一致性 | OpenAPI 规范生成 → docs/api/openapi.yaml；评审脚本对比 docs/06 vs docs/08 |
| 配置中心 | Nacos 接入；环境变量集中；配置变更审计 |
| 灾备 | PG 主从 + 每日快照演练；Redis AOF + RDB |
| 文档 | dev.md 每周更新；ADR 大决策；spec / plan 索引 |

---

## 8. L5+ 后续版本

每版本一份 spec，再细分为 feature：

- **L5 v1.1**：优惠券 / 抢单池优化 / 工单 / 数据看板（4 feature）
- **L6 v1.2**：拉新活动 / 推送体系 / 行程上报（3 feature）
- **L8 v2.0**：AI 匹配 / 医院深度合作 / 保险接入（3 feature）
- **L9 v3.0**：多区域 / 海外（2 feature）

具体 spec 在到达版本时再展开。本路线图先锚定 v1.0 的 L0~L3。

---

## 9. 测试矩阵（每 spec 必填）

| 测试类型 | 工具 / 范围 | 覆盖目标 |
| :-- | :-- | :-- |
| 单元测试 | testify + gomock | 业务包 ≥ 80%；pure function 100% |
| 集成测试 | dockertest（PG/Redis/Kafka）+ `//go:build integration` | 关键路径：抢单锁单 / 退款 / 状态机迁移 |
| e2e smoke | shell + curl | 每个 feature 给一个 `smoke-<feature>.sh` |
| 性能测试 | k6 / vegeta | L1 收敛 + L3 集成完后做；P95 < 200ms / P99 < 500ms |
| 渗透测试 | OWASP Top 10 | L3 集成完后做一次 |
| 端到端 Playwright | web/ 驱动 | 管理后台关键路径 |

---

## 10. 文档目录（最终态）

```
docs/
├── 01-09 需求文档（已存在）
├── REVIEW-REPORT.md（已存在）
├── superpowers/
│   ├── 2026-09-23-doctors-v1.md          # 已交付的 MVP 计划
│   ├── 2026-09-24-roadmap-design.md       # 本设计（本文件）
│   ├── overview.md                        # 总览时间线 + 依赖图
│   ├── specs/
│   │   ├── 2026-09-24-state-machine-design.md
│   │   ├── 2026-09-24-order-lock-design.md
│   │   ├── 2026-09-24-refund-design.md
│   │   ├── 2026-09-24-sos-design.md
│   │   ├── 2026-09-24-virtual-number-design.md
│   │   ├── 2026-09-24-wallet-design.md
│   │   ├── 2026-09-24-work-order-design.md
│   │   ├── 2026-09-24-bid-review-design.md
│   │   ├── 2026-09-24-patient-miniapp-design.md
│   │   ├── 2026-09-24-escort-app-design.md
│   │   ├── 2026-09-24-admin-web-design.md
│   │   ├── 2026-09-24-observability-design.md
│   │   └── ...
│   ├── plans/
│   │   └── ... (每个 spec 一份 plan)
│   └── decisions/
│       └── ADR-NNN-*.md (大决策)
└── diagrams/
    └── ... (架构图)
```

---

## 11. 风险与回退

| 风险 | 应对 |
| :-- | :-- |
| spec 与已交付代码脱节 | spec 必含"现状"章节描述既有实现，避免重写 |
| feature 之间循环依赖 | 写明"必须先完成的 feature"，拓扑排序 |
| 评审报告的 I-01~I-10 未跟进 | 每个 L1 spec 必含 "评审回复" 章节 |
| 三方接入延期 | dev mock 优先；真渠道在后 |
| 状态机修改影响线上数据 | 兼容迁移：新状态列加 DEFAULT，旧状态写明触发点 |
| 抢单锁单并发引发超卖 | 三道防线（SETNX + DB + state machine）+ 集成测试 50 并发 |

---

## 12. 明确不做的事

- 不在本期写任何生产代码
- 不重新评审已实现的 backend（auth/order/match + 6 骨架服务）
- 不引入 docs/ 之外的新功能（如"礼品卡" / "抽奖" 等需求）
- 不写 ADR（除非后续某个 feature 决策需要追溯）

---

## 13. 待确认问题（写代码前 review）

1. **状态机命名**：用 `pending_acceptance` 还是 `accepting`？（建议 pending_acceptance 更明确表达"已抢到但陪诊师未确认"的语义）
2. **退款阶梯的 5% 陪诊师补偿**：是否需要按城市 / 套餐类型分级？（MVP 建议固定 5%）
3. **T+7 起始点**：从 `completed` 起算 vs 从 `reviewed` 起算？建议 `completed`（评审通过后即冻结，投诉必走 dispute）
4. **虚拟号录音存储**：是否走阿里云 OSS？合规审查周期？（评审 I-08 要求所有通话录音存证）
5. **前端 monorepo**：把三个端（patient/escort/admin）放在一个 monorepo 还是三个独立 repo？（建议 monorepo：共享 design token / API client / 类型）

---

## 14. 下一步

1. 用户 review 本 spec → 修订
2. 用户确认后，按 brainstorming → writing-plans 流程，逐个出每份 feature spec + plan
3. 本期产出顺序建议：
   - **spec 优先**：state-machine → order-lock → refund → sos → virtual-number → wallet → work-order → 三个前端 → 集成
   - **每份 spec 落地**：用户 review → 写 plan → commit（不写代码）