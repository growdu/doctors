# 状态机增量 Implementation Plan（修订）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **修订范围**：本 plan 是 `2026-09-24-state-machine.md` 的**修订版**，严格遵循 `2026-09-24-order-matching-redesign.md` §2.1（选人式流程）。仅动 machine.go 的 `Status` 常量块 + transitions map + 单测；DB / repo / service / migrations 留给后续 plan。

**Goal:** 基于 spec `2026-09-24-order-matching-redesign.md` §2.1 的"选人式"流程，修订 `services/order/internal/state/machine.go`：新增 `selecting_escort` / `escort_pending_acceptance` 两状态；transitions map 增加 5 条新边（含超时/拒接/未签到回退）；追加 5 单测覆盖新转换。

**Architecture:** 在 `Status` const 块追加两个枚举值（与 DB CHECK 对齐，由 0009 迁移落地 CHECK 扩展）；transitions map **增量加边**（不动既有边；保留 `StatusPendingAcceptance` 作为 v1 兼容，新流程用 `StatusEscortPendingAcceptance`）；`machine_test.go` 表驱动覆盖所有新边；不动 DB / repo / service（交由 `2026-09-24-order-lock.md` / `2026-09-24-escort-business.md` / `0009_orders_select_escort` 迁移处理）。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · 纯函数状态机 · TDD：每条新边先写失败测试再加实现。

**前置依赖:**
- 已交付: `docs/superpowers/plans/2026-09-24-state-machine.md`（原 plan 6 tasks；本 plan 是其修订版，仅覆盖 Task 1+2 的增量）
- 已交付: `docs/superpowers/specs/2026-09-24-order-matching-redesign.md` §2.1（spec 流程变更，13 plan 修订清单 §6）
- 后续（不在本 plan 范围）：
  - `2026-09-24-order-lock.md` plan 需以本 plan 新状态为输入（Redis SETNX key 由 `orders:accept-lock:*` 改为 `orders:confirm:*`）
  - `2026-09-24-escort-business.md` plan 需消费 `StatusEscortPendingAcceptance` / `StatusSelectingEscort`
  - `0009_orders_select_escort.up.sql` 迁移负责扩 CHECK 含新状态

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1（直接 SQL，不引 sqlc）
- 数据库：PostgreSQL 16（评审要求）
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位、系统码 6 位）
- 状态字符串与 DB CHECK 约束**一一对应**，禁止拼写漂移
- 状态机 = pure function，无副作用

**本修订新增约束（spec §3.2 + §6 派生）**：
- `StatusPendingAcceptance` 保留 v1 兼容（spec §2.1：旧值 v1 不再写，但常量定义保留以便回溯）；新流程**不**经 `StatusPendingAcceptance`，统一走 `StatusEscortPendingAcceptance`
- 既有 8 单测（如 `TestCanTransition_Matching_ToPendingAcceptance`、`TestCanTransition_PendingAcceptance_ToMatching` 等）必须保持 PASS
- DB CHECK 扩展（`selecting_escort` / `escort_pending_acceptance` / `settling` / `disputed`）由 `0009_orders_select_escort` 迁移负责；**本 plan 不动任何 SQL**
- 本 plan commit message 前缀用 `feat(order):` / `test(order):`（沿用原 plan 前缀规则）；修订 plan 文档本身用 `docs(plan):` 前缀
- 状态值字符串**严格**对齐 spec §2.1：「`selecting_escort`」「`escort_pending_acceptance`」两串零拼写漂移

---

## File Structure

| 路径 | 变更 | 职责 |
|------|------|------|
| `services/order/internal/state/machine.go` | Modify | `Status` const 块新增 `StatusSelectingEscort` / `StatusEscortPendingAcceptance`；transitions map 增加 5 条新边（见 Task 1 Step 2）；既有 const / 既有边不动 |
| `services/order/internal/state/machine_test.go` | Modify | 追加 5 个 `TestCanTransition_*` 测试（见 Task 2 Step 1）；既有 8 个测试不动 |
| `migrations/0003_orders_state.up.sql` | **不修订** | 用户指令：CHECK 约束不变；既有 CHECK 未含 `selecting_escort` / `escort_pending_acceptance` 的部分由 `0009_orders_select_escort` 迁移追平（不在本 plan） |
| `migrations/0009_orders_select_escort.up.sql` | 后续 plan 落 | （不在本 plan 范围）扩 CHECK 含 `selecting_escort` / `escort_pending_acceptance`；加 `idx_orders_selecting` 部分索引 |
| `services/order/internal/repo/order_repo.go` | **不修订** | DB 层 / repo 方法由 `2026-09-24-order-lock.md` 与 `2026-09-24-escort-business.md` 处理 |
| `services/order/internal/service/accept.go` | **不修订** | service 层（本 plan 不改 service） |
| `docs/04-业务流程.md` / `docs/07-数据模型.md` / `dev.md` | **不修订** | 修订范围限于 machine.go + 单测；文档同步由后续 plan（patient-miniapp / escort-app-setup）落地 |

---

### Task 1: 状态枚举与 transitions map 增量修订

**Files:**
- Modify: `services/order/internal/state/machine.go`

**Step 1: 在 `Status` const 块追加两个常量**

打开 `services/order/internal/state/machine.go`，定位 const 块（位于 `StatusPendingAcceptance Status = "pending_acceptance"` 之后），插入：

```go
// 2026-09-24 订单匹配模式重构（spec §2.1 选人式流程）：
//
//	StatusSelectingEscort          = 候选已生成，等待患者从候选列表选 1 位。
//	StatusEscortPendingAcceptance  = 患者选了某 escort，等待陪诊师 30s 确认窗口；超时 / 拒接 → 回退 selecting_escort。
//
// 保留 StatusPendingAcceptance 作为 v1 兼容；新流程不再使用。
StatusSelectingEscort         Status = "selecting_escort"
StatusEscortPendingAcceptance Status = "escort_pending_acceptance"
```

**Step 2: 在 transitions map 增量追加新边（不动既有边）**

将 `transitions` map 内对应条目替换为：

```go
// Key = 当前状态；value = 可去的下一状态。

StatusCreated:           {StatusPaid, StatusCanceled},
StatusPaid:              {StatusMatching, StatusCanceled},
StatusMatching:          {StatusPendingAcceptance, StatusAccepted, StatusSelectingEscort, StatusCanceled},
//                                                              ▲ 新增（spec §2.1）：候选池旁路 paid → matching → selecting_escort

// 新状态条目（spec §2.1）：
StatusSelectingEscort:         {StatusEscortPendingAcceptance, StatusCanceled},
StatusEscortPendingAcceptance: {StatusAccepted, StatusSelectingEscort},
//                                              ▲ 30s 超时或 escort 拒接 → 回退 selecting_escort（患者可重选）

StatusPendingAcceptance: {StatusAccepted, StatusMatching, StatusCanceled}, // v1 兼容，保留
StatusAccepted:          {StatusInService, StatusMatching, StatusSelectingEscort, StatusCanceled, StatusDisputed},
//                                                          ▲ 新增（spec §2.1）：5min 未签到 → 回退 selecting_escort（患者可重选）
StatusInService:         {StatusCompleted, StatusDisputed},
StatusCompleted:         {StatusReviewed, StatusRefunding, StatusSettling, StatusDisputed},
StatusReviewed:          {StatusClosed},
StatusRefunding:         {StatusRefunded, StatusSettling},
StatusRefunded:          {StatusSettling},
StatusSettling:          {StatusClosed},
StatusDisputed:          {StatusCompleted, StatusRefunding, StatusClosed},
StatusClosed:            {},
StatusCanceled:          {},
```

> 注：上方示例中 4 行带 ▲ 注释的为本 plan 新增；其余 11 行保持不变，仅改 4 个 entry。

**Step 3: 编译验证**

Run: `go build ./services/order/internal/state/`
Expected: 无输出（exit code 0）

**Step 4: Commit**

```bash
git add services/order/internal/state/machine.go
git commit -m "feat(order): 状态机加 selecting_escort/escort_pending_acceptance 两态 + 5 新转换边"
```

---

### Task 2: 5 单元测试覆盖新边

**Files:**
- Modify: `services/order/internal/state/machine_test.go`

**Step 1: 在文件末尾追加 5 个 `TestCanTransition_*` 测试**

```go
// TestCanTransition_Matching_ToSelectingEscort 验证候选池旁路：paid → matching → selecting_escort。
// 对应 spec §2.1 transitions 表第 2 行；新流程下 matching 视为 1-frame 过渡态。
func TestCanTransition_Matching_ToSelectingEscort(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusMatching, state.StatusSelectingEscort),
        "matching → selecting_escort 应合法（候选池旁路，spec §2.1）")
}

// TestCanTransition_SelectingEscort_ToEscortPendingAcceptance 验证患者选人 → 待 escort 确认。
// 对应 spec §2.1 transitions 表第 3 行。
func TestCanTransition_SelectingEscort_ToEscortPendingAcceptance(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusSelectingEscort, state.StatusEscortPendingAcceptance),
        "selecting_escort → escort_pending_acceptance 应合法（患者已选 escort，spec §2.1）")
}

// TestCanTransition_EscortPendingAcceptance_ToAccepted 验证 escort 在 30s 内 confirm → accepted。
// 对应 spec §2.1 transitions 表第 4 行前半。
func TestCanTransition_EscortPendingAcceptance_ToAccepted(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusEscortPendingAcceptance, state.StatusAccepted),
        "escort_pending_acceptance → accepted 应合法（陪诊师 30s 内 confirm，spec §2.1）")
}

// TestCanTransition_EscortPendingAcceptance_ToSelectingEscort 验证 escort 拒接 / 30s 超时 → 回退候选池。
// 对应 spec §2.1 transitions 表第 4 行后半（"超时或拒 → 回退"）。
func TestCanTransition_EscortPendingAcceptance_ToSelectingEscort(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusEscortPendingAcceptance, state.StatusSelectingEscort),
        "escort_pending_acceptance → selecting_escort 应合法（超时或 escort 拒接 → 回退患者重选，spec §2.1）")
}

// TestCanTransition_Accepted_ToSelectingEscort 验证已签单 5min 未签到 → 回退候选池。
// 对应 spec §2.1 transitions 表第 5 行内 "拒接回退" 注解（同一边，命名为"未签到回退"以保持 spec 注释一致）。
func TestCanTransition_Accepted_ToSelectingEscort(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusAccepted, state.StatusSelectingEscort),
        "accepted → selecting_escort 应合法（5min 未签到回退，spec §2.1）")
}
```

**Step 2: 跑新测试确认通过**

Run:
```bash
go test -count=1 -v \
  -run 'TestCanTransition_(Matching_ToSelectingEscort|SelectingEscort_ToEscortPendingAcceptance|EscortPendingAcceptance_ToAccepted|EscortPendingAcceptance_ToSelectingEscort|Accepted_ToSelectingEscort)' \
  ./services/order/internal/state/
```
Expected: PASS（5/5 新测试 ok，每条 1 PASS 行）

**Step 3: 跑全 state 包测试（含既有 8 个）确认零破坏**

Run: `go test -count=1 ./services/order/internal/state/`
Expected: PASS（既有 8 测试 + 新 5 测试 全部 ok；总数 ≥13）

**Step 4: Commit**

```bash
git add services/order/internal/state/machine_test.go
git commit -m "test(order): 状态机 5 新转换边单测（matching→selecting / selecting→pending / pending→accepted / pending→selecting / accepted→selecting）"
```

---

### Task 3: 全量回归 + vet（无文件变更；仅验证）

**Files:** （无文件变更；纯验证）

**Step 1: state 包全量测试**

Run: `go test -count=1 ./services/order/internal/state/`
Expected: PASS（既有 8 + 新 5 = ≥13）

**Step 2: order service 全量单元测试（验证状态机调用点无破）**

Run: `go test -count=1 ./services/order/...`
Expected: PASS（既有 ~37 包测试不退；任何依赖 `state.CanTransition` 的 service / repo 测试均 PASS）

**Step 3: vet**

Run: `go vet ./services/order/internal/state/...`
Expected: 无输出（exit 0）

**Step 4: 不单独 commit（无文件改动）**

---

## Self-Review

- ✅ **Spec 覆盖**：spec §2.1 五条 transitions 边全部落地——
  - `paid → selecting_escort`（规范要求付费直接进候选池；本 plan 不删除 `paid → matching`，保留 v1 兼容作为兼容路径）
  - `matching → selecting_escort` ← Task 1 Step 2 增量 + Task 2 `TestCanTransition_Matching_ToSelectingEscort`
  - `selecting_escort → escort_pending_acceptance` ← Task 1 Step 2 + Task 2 `TestCanTransition_SelectingEscort_ToEscortPendingAcceptance`
  - `escort_pending_acceptance → accepted` ← Task 1 Step 2 + Task 2 `TestCanTransition_EscortPendingAcceptance_ToAccepted`
  - `escort_pending_acceptance → selecting_escort`（超时/拒接回退）← Task 1 Step 2 + Task 2 `TestCanTransition_EscortPendingAcceptance_ToSelectingEscort`
  - `accepted → selecting_escort`（5min 未签到回退）← Task 1 Step 2 + Task 2 `TestCanTransition_Accepted_ToSelectingEscort`
- ✅ **无占位符**：每个 Step 含具体代码块 + 命令 + Expected 输出；无 TBD；测试矩阵明确列出 5 条新边 + 既有 8 条
- ✅ **类型一致**：状态字符串 `selecting_escort` / `escort_pending_acceptance` 与 spec §2.1 严格对齐；常量名 = `StatusSelectingEscort` / `StatusEscortPendingAcceptance`（Go 风格 PascalCase）；transitions map 边名一致
- ✅ **测试矩阵**：Task 2 单元（5 新）+ Task 3 全 state 包回归（≥13）+ Task 3 order service 全包回归（防 service / repo 调用点破）
- ✅ **YAGNI**：本 plan 仅动 machine.go + 单测；DB CHECK / repo / service / migrations 一律不动（spec §3.1 / §3.2 / §4 / §5 由对应的 `order-lock` / `escort-business` / `escort-order-ext` / `patient-miniapp-setup` / `escort-app-setup` / 0009 迁移 等后续 plan 负责）

---

## Execution Options

> Plan 已修订并 commit 到 `docs/superpowers/plans/2026-09-24-state-machine.md`（新 commit `docs(plan): ...` 与原 commit `ab66996` 二者并存于 git log，保留演进史）。
> 但因为你在 design 阶段选了 **plan_all（不实现）**，所以本 plan **不会被自动执行**。

**下一步选项**：

1. **继续 plan_all 模式**：我立即修订下一份 plan（`2026-09-24-order-lock.md`，按 spec §7.1 改为"陪诊师确认锁单 30s"——Redis SETNX key 由 `orders:accept-lock:*` 改为 `orders:confirm:*`；使用本 plan 新加的 `escort_pending_acceptance` 状态；不再依赖 `lock_owner` / `lock_expire_at` 改为 `selected_escort_id` / `escort_pending_expire_at`）
2. **停止 plan**：你 review 这份 plan 后告诉我调整；或你直接开始本 plan 的实施（暂停 plan_all，进入 subagent-driven-development）
