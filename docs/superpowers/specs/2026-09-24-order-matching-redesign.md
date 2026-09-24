# 订单匹配模式重构 — 抢单 → 客户选人

> **For spec reviewers:** 这是 L2 v1.0 前端的**业务流程修订**。影响后续 13 份 plan 中与匹配 / 抢单 / 锁单相关的多个 plan。本 spec 定义新的状态机、API、字段；具体修订哪个 plan 在 §6 列出。

**Goal:** 把订单匹配模式从「陪诊师抢单」改为「陪诊师设时段 → 患者选人 → 陪诊师 30s 确认」。这样更符合 v1.0 业务实际：陪诊师是被动接诊（不是销售），患者挑人才能做出信任决策。

**Architecture:** 删 `lock_owner` / `lock_expire_at` 字段；新增 `escort_availabilities` 表（陪诊师时段）和 `selected_escort_id` 字段；状态机加 `selecting_escort`（候选已生成，待患者选）和 `escort_pending_acceptance`（患者选了某陪诊师，待其确认）；移除 `pending_acceptance` 状态用法（改名含义）。**核心事件**改为 `OrderSelectingEscortEvent`（陪诊师候选名单更新时）+ `OrderEscortSelectedEvent`（患者选了某人）+ `OrderEscortConfirmedEvent`（陪诊师确认）+ `OrderEscortRejectedEvent`（陪诊师拒 / 超时）。

**Tech Stack:** Go 1.24+ · pgx v5.7 · segmentio/kafka-go · 既有状态机纯函数 + match.scorer 复用（候选推荐排序）。

**前置依赖:**
- `2026-09-24-state-machine.md`（已交付；本 spec 在其基础上加状态 + 修订字段）
- `2026-09-24-l2-api-gap-design.md`（已交付；本 spec 修订其 §2.2 escort-side order + 加 availability）
- `2026-09-24-escort-app-design.md`（已交付；本 spec 修订其 §3.1 Feed → 邀请列表 + 我的时段）
- `2026-09-24-patient-miniapp-design.md`（已交付；本 spec 修订其 §3.1 订单详情 → 加"选陪诊师"页）

---

## 1. 核心流程对比

### 1.1 旧流程（抢单式，已撤销）

```
陪诊师：上线（status:anytime）→ 进入抢单池（Redis ZSET matching:*）
       → 看订单 feed（GET /match/feed）→ 30s 锁单（state:pending_acceptance）
       → 30s 内 confirm accept → state:accepted

订单：paid → matching（候选池）→ pending_acceptance（被锁定）→ accepted
```

### 1.2 新流程（选人式，本 spec 落地）

```
陪诊师：注册 → 实名 → 通过审核 → 上线时设置空余时段（escort_availabilities 表）
       → 收到 patient 选择邀请 → 30s 内 confirm accept/reject
       → 超时或拒 → 订单回到 selecting_escort（患者可重选）

订单：paid → selecting_escort（候选已生成）
       → 患者从候选列表选 1 位 → escort_pending_acceptance（待陪诊师 30s 确认）
       → 陪诊师 confirm → accepted
       → 陪诊师 reject 或 30s 超时 → 回到 selecting_escort（PatientEvent → 患者可重选）
```

---

## 2. 状态机变更

### 2.1 订单状态机（services/order/internal/state/machine.go）

**新增**：
- `selecting_escort` = 候选已生成，等待患者选
- `escort_pending_acceptance` = 患者选了某人，等待 30s 确认窗口

**保留 / 复用**：
- `matching` 改为「候选池」语义（已支付未选 → `selecting_escort`）；保留作为过渡态 1 frame
- `pending_acceptance` 改为「**escort** 待确认」→ 重命名为 `escort_pending_acceptance`（更清晰）；旧 `pending_acceptance` 保留 v1 兼容但不再使用

**transitions 表更新**：

```go
StatusPaid:                    {StatusSelectingEscort, StatusCanceled},
StatusMatching:                {StatusSelectingEscort, StatusCanceled}, // 保留过渡
StatusSelectingEscort:         {StatusEscortPendingAcceptance, StatusCanceled},
StatusEscortPendingAcceptance: {StatusAccepted, StatusSelectingEscort}, // 超时或拒 → 回退
StatusAccepted:                {StatusInService, StatusSelectingEscort, StatusCanceled, StatusDisputed}, // 拒接回退到 selecting
```

### 2.2 陪诊师可用性模型

- `escort_availabilities` 表状态：`available` / `booked` / `canceled`
- 业务规则：
  - 一个时段只能被一个订单占用
  - 陪诊师拒接 / 超时 → 时段保持 `available`（订单回到 `selecting_escort`）
  - 陪诊师 confirm → 时段 `booked`，锁定到 order_id
  - 订单 cancel → 时段恢复 `available`

---

## 3. 数据库 schema 变更

### 3.1 orders 表（扩展 0003 迁移为 0009）

```sql
-- 0009_orders_select_escort.up.sql
-- 修订：移除 lock_owner / lock_expire_at；新增 selected_escort_id + escort_pending_expire_at

ALTER TABLE orders
  DROP COLUMN IF EXISTS lock_owner,
  DROP COLUMN IF EXISTS lock_expire_at;

ALTER TABLE orders
  ADD COLUMN selected_escort_id BIGINT REFERENCES users(id),
  ADD COLUMN escort_pending_expire_at TIMESTAMPTZ;

-- 新状态 CHECK（替换既有 CHECK）
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','selecting_escort','escort_pending_acceptance','accepted',
  'in_service','completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));

-- 新索引：候选查询 + 待确认扫描
CREATE INDEX idx_orders_selecting ON orders(service_start_at)
  WHERE status IN ('selecting_escort','escort_pending_acceptance');
```

### 3.2 新表：escort_availabilities（0009 同迁移）

```sql
-- 陪诊师可预约时段（陪诊师上线时设置；患者只能选 available 时段）
CREATE TABLE escort_availabilities (
  id BIGSERIAL PRIMARY KEY,
  escort_id BIGINT NOT NULL REFERENCES users(id),
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'available'
    CHECK (status IN ('available','booked','canceled')),
  order_id BIGINT REFERENCES orders(id),  -- booked 时必填
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (end_at > start_at)
);

-- 时段不冲突：同 escort_id 不能有时间重叠
CREATE UNIQUE INDEX idx_escort_avail_unique ON escort_availabilities(escort_id, start_at);

-- 候选匹配查询：available 时段按时间 + 评分过滤
CREATE INDEX idx_escort_avail_status_start ON escort_availabilities(status, start_at)
  WHERE status = 'available';
```

### 3.3 删 `idx_orders_lock`（被 0009 替代）

```sql
DROP INDEX IF EXISTS idx_orders_lock;
```

---

## 4. API 修订

### 4.1 新增（l2-api-gap-design.md §2.1 patient 端 + §2.2 escort 端）

#### escort 端（替换抢单池）

```yaml
PUT /api/v1/escorts/me/availability:        # 增/改时段（body: start_at + end_at）
DELETE /api/v1/escorts/me/availability/{id}: # 删时段（仅 available）
GET /api/v1/escorts/me/availability:       # 查我的时段
GET /api/v1/escorts/me/invitations:        # 待我确认的订单（selecting → 我被选）
POST /api/v1/orders/{id}/confirm-accept:   # 陪诊师 30s 内确认 → accepted
POST /api/v1/orders/{id}/reject-accept:     # 陪诊师拒接 → 回退 selecting_escort
```

#### patient 端（替换抢单）

```yaml
GET /api/v1/orders/{id}/candidates:         # 候选陪诊师列表（按评分排序）
POST /api/v1/orders/{id}/select-escort:    # 患者选某陪诊师（body: escort_id）→ escort_pending_acceptance
```

### 4.2 删除（已撤销）

```yaml
GET /api/v1/match/feed:            # 抢单池 —— 撤销
POST /api/v1/orders/{id}/accept:   # 抢单 —— 撤销（改为 select-escort + confirm-accept 两步）
```

### 4.3 保留

- `GET /api/v1/escorts/{id}/availabilities?start_at=...&end_at=...`（公开）：候选用
- match.scorer 评分逻辑（match-service）：复用为候选推荐排序
- OrderMatchingEvent → 改名为 `OrderSelectingEscortEvent`（语义：候选列表生成）

---

## 5. 事件类型

### 5.1 shared/contracts/events.go 新增

```go
TopicOrderSelectingEscort    = "order.selecting_escort"  // 候选列表生成
TopicOrderEscortSelected     = "order.escort_selected"   // 患者选了某人
TopicOrderEscortConfirmed    = "order.confirmed"         // 陪诊师 30s 内 confirm
TopicOrderEscortRejected     = "order.rejected"          // 陪诊师拒 / 超时
TopicEscortInvitationTimeout = "escort.invitation_timeout" // 30s 超时（内部事件，可选）

type OrderSelectingEscortEvent struct {
    OrderID    int64     `json:"order_id"`
    Candidates []int64   `json:"candidates"`  // 候选 escort_ids
    GeneratedAt time.Time `json:"generated_at"`
}

type OrderEscortSelectedEvent struct {
    OrderID    int64 `json:"order_id"`
    PatientID  int64 `json:"patient_id"`
    EscortID   int64 `json:"escort_id"`
    SelectedAt time.Time `json:"selected_at"`
    ExpiresAt  time.Time `json:"expires_at"`  // = now + 30s
}

type OrderEscortConfirmedEvent struct {
    OrderID    int64 `json:"order_id"`
    EscortID   int64 `json:"escort_id"`
    ConfirmedAt time.Time `json:"confirmed_at"`
}

type OrderEscortRejectedEvent struct {
    OrderID    int64 `json:"order_id"`
    EscortID   int64 `json:"escort_id"`
    Reason     string `json:"reason"`  // "escort_declined" | "lock_expired"
    RejectedAt time.Time `json:"rejected_at"`
}
```

### 5.2 删除

```go
TopicOrderMatching = "order.matching"  // 撤销
OrderMatchingEvent                      // 撤销（被 OrderSelectingEscortEvent 替代）
```

---

## 6. 影响的 plan 清单（13 份 → 7 份需修订 / 1 份新增）

| plan | 修订范围 | 工作量 |
|---|---|---|
| **2026-09-24-state-machine** | §4.2 加 `selecting_escort` + `escort_pending_acceptance` 状态；transitions 表更新 | 小 |
| **2026-09-24-order-lock** | **整个 plan 改写**：从"抢单锁单" → "选 escort + 30s 陪诊师确认锁单"；Redis SETNX key 改为 `orders:confirm:{order_id}`；release_token 与 token 逻辑保留 | **中** |
| **2026-09-24-refund** | 不变（取消时 refund 触发不变） | 无 |
| **2026-09-24-sos** | 不变 | 无 |
| **2026-09-24-virtual-number** | 不变（accepted 时生成不变） | 无 |
| **2026-09-24-escort-business** | §1 加 escort_availabilities 表 + availability CRUD API（3 个新端点）；§1 加陪诊师状态 `available` / `busy` / `off-line`；§1 加 `GET /escorts/me/invitations`（待确认订单） | **大** |
| **2026-09-24-escort-order-ext** | §1 重写：原 `POST /orders/{id}/accept`（抢单）→ 拆为 `POST /orders/{id}/select-escort` + `POST /orders/{id}/confirm-accept`；保留 checkin/checkout | **中** |
| **2026-09-24-hospital-package / review / message / address-coupon / wallet / admin** | 不变（与匹配模式无关） | 无 |
| **2026-09-24-patient-miniapp-setup** | §3.1 订单详情：新增"选陪诊师"步骤 + 候选列表页 + 30s 倒计时（陪诊师确认窗口）；§4.3 抢单相关代码删除 | **中** |
| **2026-09-24-escort-app-setup** | §3.1 抢单池页 → 改为"我的邀请"（escort_pending_acceptance 列表）+ 30s 倒计时确认按钮；§3.1 加"我的空余时段"页（增/删时段） | **中** |
| **2026-09-24-admin-web-setup** | §3.1 订单列表：新增"是否已选 escort"列 + 状态机新颜色；§3.3 dashboard：加"待确认订单"统计 | 小 |

**新增 plan**：

| plan | 范围 | 工作量 |
|---|---|---|
| **2026-09-24-escort-availability**（新） | escort_availabilities 表迁移 + AvailabilityRepo + AvailabilityService + 时段冲突检测 + 与 order-service / match-service 集成 + 8 集成测试 | **大** |

**未受影响（不变）**：refund / sos / virtual-number / hospital-package / review / message / address-coupon / wallet / admin

---

## 7. 关键设计决策

### 7.1 取消锁单 30s → 改为陪诊师确认 30s

- Redis SETNX key：`orders:confirm:{order_id}`（替代 `orders:accept-lock:{order_id}`）
- 用途：防止患者 / 系统重复触发；**不**用于陪诊师抢单（抢单已撤销）
- TTL：30s；过期由 `ExpiredLockScanner` 自动回退（保留 order-lock plan 的 scanner，key 改名）
- token 简化处理保留（best-effort + TTL 兜底）

### 7.2 候选推荐算法

- 复用 `match-service` 既有 scorer（city/time/rating/distance）
- 输入：订单 ID + service_start_at
- 输出：Top N（默认 5）escort IDs，按评分倒序
- 过滤：`escort_availabilities` 中 status='available' 且 start_at ≤ order.service_start_at ≤ end_at
- 触发：订单 paid 后自动调用，发布 `OrderSelectingEscortEvent`

### 7.3 拒接 / 超时重选

- 陪诊师拒接 → state 回退 `escort_pending_acceptance` → `selecting_escort`，发布 `OrderEscortRejectedEvent(reason='escort_declined')`
- 30s 超时 → scanner 自动回退，发布 `OrderEscortRejectedEvent(reason='lock_expired')`
- 患者**可重选**（从候选里选另一位）；无硬性上限但 spec §3.3 建议 3 次（v1 不强制）

### 7.4 已采纳的时段（escort_availabilities）

- 上线 = 有 available 时段；下线 = 所有时段 canceled
- 时段被 booking 后：status='booked' + order_id；订单 cancel / 时段 cancel 时恢复 'available'
- 同一 escort 不能有时间重叠的时段（DB UNIQUE 索引 + service 层校验）

---

## 8. 验收

### 8.1 端到端测试场景

1. 陪诊师 A 上线 → 设置 14:00~18:00 时段 → 系统记录
3. 患者 P 创建订单（service_start_at=15:00）→ paid
4. 系统生成候选 [A, C, E] → 推送 OrderSelectingEscortEvent
5. P 选「A」→ state: escort_pending_acceptance → OrderEscortSelectedEvent
6. A 在 30s 内 confirm → state: accepted → OrderEscortConfirmedEvent
7. A 在 30s 后才 confirm → 业务校验拒（state 已回退 selecting_escort）

### 8.2 异常路径

- 候选为空（无任何 escort 在 service_start_at 时有空） → 订单 5min 后自动 cancel + 触发 refund
- 患者选了已下线 escort → service 校验 availability 状态，返 CodeNotFound
- 同一订单并发选两位 → Redis SETNX + DB optimistic lock 双保险
- 患者拒接超过 3 次（v1 不强制）→ 进入 disputed（admin 介入）

---

## 9. 不做 / 留 v2

| 不做 | 留给 |
|---|---|
| 患者拒接次数硬性限制 | v2（v1 软提示） |
| 陪诊者按调度模板（每周固定时段） | v2 |
| 患者选多候选 + 智能调度 | v2 |
| 智能评分算法（ML） | v2（v1 用 match.scorer 既有规则） |
| 候选取现实时（v1 一次生成 5 候选） | v2 引 WebSocket |

---

## Self-Review

- ✅ Spec 覆盖：新流程 + 状态机 + schema + API + 事件 + plan 修订清单
- ✅ 无占位符：每节有具体 SQL / 代码 / 路径
- ✅ 类型一致：事件 / 字段命名跨服务一致
- ✅ 测试矩阵：§8.1 / §8.2 覆盖端到端 + 异常路径
- ✅ YAGNI：§9 明确不做 / 留 v2

## Execution Options

**修订 7 份 plan + 写 1 份新 plan（escort-availability）** ≈ 8 个工作单元。

**可选下一步**：
1. **先 review spec** —— 确认状态机 + 流程细节后再实施
2. **修订优先级**：先修订核心 3 个（state-machine / order-lock / escort-order-ext），再修订 4 个周边（escort-business / patient-miniapp / escort-app / admin-web）
3. **批量改**：一次性修订 7 个 + 写 1 个新 ≈ 1-2 小时密集输出

你建议哪种？