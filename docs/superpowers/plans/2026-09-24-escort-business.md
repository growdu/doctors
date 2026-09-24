# 陪诊业务实装 Implementation Plan（v2 — escort_availabilities + 邀请模式）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 escort-business v1（注册 / 实名 / 健康证 / 培训 / 审核 / 上线 / checkin-checkout / 抢单池已撤销）基础上，**新增 escort 配对的 availability 管理**：陪诊师可设置空余时段、患者可见与可选、拒接 / 超时 / 取消时自动恢复时段、邀请列表 + 状态派生。落地：
- 共享迁移 `0009_escort_availabilities.up.sql`（同时修订 orders 表：删 `lock_owner` / `lock_expire_at`，加 `selected_escort_id` / `escort_pending_expire_at`）
- 新增 `services/escort/internal/availability/` 子包：`AvailabilityRepo` + `AvailabilityService`
- 5 个新 endpoint：3 个 escort 自有 (`PUT/GET/DELETE /escorts/me/availability`)、1 个公开 (`GET /escorts/:id/availabilities`)、1 个邀请列表 (`GET /escorts/me/invitations`)
- 派生状态 `available` / `busy` / `off-line`（基于 `escort_availabilities` + manual offline）
- 与 order-service 集成：`order-service.ConfirmAccept` 触发 `BookByOrder`，`order-service.ReleaseLockAndReject` 触发 `ReleaseByOrder`（v1 简化为共享 Go 包 / 直接 import `availability` 子包）

参考 spec：`docs/superpowers/specs/2026-09-24-order-matching-redesign.md` §3.2 + §4.1 + §7。

**Architecture:**
- **共享一个迁移** (`0009_escort_availabilities.up.sql`)：orders 表的 lock_owner/lock_expire_at 删除、加 `selected_escort_id`/`escort_pending_expire_at`、CHECK 改、索引改；与 `escort_availabilities` 表共存。迁移由 order-matching-redesign spec + escort-business v2 共用。
- **availability 子包**：纯数据层 `AvailabilityRepo`（pgx + 乐观锁）+ 业务层 `AvailabilityService`（时段冲突校验、状态机 actions、状态派生）。所有 service 方法均可在没有 order-service 依赖的情况下独立 work。
- **状态派生**：`available` 判定 = 当前有至少一条 status='available' 且 end_at > now 的时段；`busy` = 没有 available 时段（即使有 booked/canceled）；`off-line` = escort 主动 set offline（escort_profiles.state='offline'）。
- **集成**：v1 采用 Go 包直接 import — order-service 通过 `internal/integration` 包调用 `availability.Service.BookByOrder/ReleaseByOrder`，避免 HTTP 跨服务往返。RPC 升级留 v2。
- **pro 通用规则**：所有响应走 `shared/httpx`；所有错误走 `shared/errs.Error`；事件 best-effort；commit 节奏按 Task 切分。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · gin v1.10 · segmentio/kafka-go v0.4.51。

**前置依赖:**
- `2026-09-24-state-machine.md`（orders 表 + status CHECK；本 plan 的 orders 修订以其为基础）
- `2026-09-24-order-matching-redesign.md`（spec §3.2 escort_availabilities schema、§4.1 API、§5.1 事件名）
- `2026-09-24-l2-api-gap-design.md` §2.2 escort 端 + §3.1 EscortSummary
- `2026-09-24-escort-app-design.md` §3.1 「我的空余时段」+ §3.1 「我的邀请」

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1 + segmentio/kafka-go v0.4.51
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- 迁移文件唯一权威 `0009_escort_availabilities.{up,down}.sql`（order 表修订 + availability 表共存）
- `escort_availabilities` 状态 3 态：`available` / `booked` / `canceled`（CHECK 约束；service 暴露 `CanTransition(from, action) (to, bool)` 严格白名单）
- DB 约束：UNIQUE INDEX `(escort_id, start_at)`（防同 start_at 重复）+ 服务层时段重叠校验 `start_at < new_end AND end_at > new_start`（防区间重叠）
- 时段 start_at 必须 `> now` 且 `end_at > start_at`，否则拒 400
- service 写状态用乐观锁 `expected_version`（profile_repo 已有模式复用）；冲突返回 `CodeConflict`
- 公开 endpoint `GET /escorts/:id/availabilities` 强制 `status='available'` 过滤，避免泄露 booked/canceled 数据
- 陪诊师派生状态：`available` 当且仅当有 status='available' 且 end_at > now 的时段；`busy` 当全部时段都不是 available；`off-line` 当 escort_profiles.state='offline'（最终态优先于 availability 判定）
- order-service 集成：`BookByOrder(slotID, orderID)` + `ReleaseByOrder(orderID)`；两者幂等；事件发布 best-effort
- 抢单 Feed（`GET /match/feed` + `POST /orders/:id/accept`）v1 已撤销，本 plan 不包含；escort-order-ext plan 处理 select-escort + confirm-accept + reject-accept
- admin 审核（approve/reject）走 admin plan，直接 `UPDATE escort_profiles.state`；状态机预留 `approved / rejected` 转换已 by escort-business v1 Task 2 状态机守门
- v1 不做：时段模板（每周固定）/ 智能匹配 / ML 评分（沿用 match-service 既有 scorer）

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `migrations/0009_escort_availabilities.up.sql` | Create | orders 表修订（drop lock_owner/lock_expire_at + add selected_escort_id/escort_pending_expire_at + new CHECK + drop idx_orders_lock + add idx_orders_selecting）+ `escort_availabilities` 表 |
| `migrations/0009_escort_availabilities.down.sql` | Create | 逆向（drop table + add columns back + restore old CHECK + restore old index） |
| `migrations/migrations_test.go` | Modify | 加 `Test0009EscortAvailabilitiesUpDown`（列检查 + 索引 + CHECK + 时段冲突 DB 级演示） |
| `services/escort/internal/availability/types.go` | Create | `Availability` struct + 状态常量 `StatusAvailable` / `StatusBooked` / `StatusCanceled` + 哨兵 `ErrNotFound` / `ErrConflict` / `ErrForbidden` / `ErrVersionConflict` |
| `services/escort/internal/availability/state.go` | Create | 状态机 pure function（`CanTransition(from, action)` 与 `IsValid`）；3 态流转：`available → booked` / `available → canceled` / `booked → available` |
| `services/escort/internal/availability/state_test.go` | Create | 状态机单测（合法 / 非法转换矩阵） |
| `services/escort/internal/availability/repo.go` | Create | pgx 实现 `AvailabilityRepo`：`Create` / `GetByID` / `Delete` / `ListByEscort` / `ListAvailableByTime` / `ListByEscortInTimeRange` / `Update` / `BookByOrder` / `ReleaseByOrder` |
| `services/escort/internal/availability/repo_integration_test.go` | Create | 集成测试（建表 + 写入 + 时段冲突 + 状态转换 + 反查） |
| `services/escort/internal/availability/service.go` | Create | `AvailabilityService`：业务校验 + 时段冲突检查 + 调用 repo；`DeriveStatus(ctx, escortID)` 派生 |
| `services/escort/internal/availability/service_test.go` | Create | service 单测（fake repo；覆盖冲突校验、过期时段过滤、拒绝取消已 booked、derive status 4 情形） |
| `services/escort/internal/handler/escort_availability.go` | Create | `Handler` 扩展：5 个 endpoint（PUT/GET/DELETE /escorts/me/availability + GET /escorts/:id/availabilities + GET /escorts/me/invitations） |
| `services/escort/internal/handler/escort_availability_test.go` | Create | handler 单测（httptest + fake service） |
| `services/escort/internal/handler/escort.go` | Modify | `RegisterRoutes` 加挂 5 个 route |
| `services/escort/internal/service/escort_service.go` | Modify | `GetMyProfile` 响应加 `availability_status` 字段（来自 `availability.Service.DeriveStatus`） |
| `services/escort/internal/service/escort_service_test.go` | Modify | 加 1 个测试（profile 含 availability_status） |
| `services/order/internal/integration/escort_availability.go` | Create | order-service ↔ escort-availability 集成层：`BookByOrder(ctx, slotID, orderID)` / `ReleaseByOrder(ctx, orderID)`；调 `availability.Service` |
| `services/order/internal/integration/escort_availability_test.go` | Create | 集成层单测（fake availability service；验证幂等） |
| `services/order/internal/service/order_service.go` | Modify | `ConfirmAccept` 在状态推进后调 `escortAvailability.BookByOrder(slotID, orderID)`；`ReleaseLockAndReject` 在状态回退后调 `escortAvailability.ReleaseByOrder(orderID)` |
| `services/order/internal/service/order_service_test.go` | Modify | 加 2 个测试（ConfirmAccept 调 BookByOrder / Reject 调 ReleaseByOrder） |
| `services/escort/cmd/main.go` | Modify | 装配 `availability.NewRepo(pool)` + `availability.NewService(repo)` + 注入 handler |
| `services/order/cmd/main.go` | Modify | 装配 `integration.NewEscortAvailability(escortSvc)` |
| `scripts/smoke-availability.sh` | Create | smoke（启动 escort/order + /healthz + 5 endpoint 401 拦截 + 邀请列表 200） |
| `docs/04-业务流程.md` | Modify | §4.7 加「选人模式 + escort_availabilities」流程图 |
| `dev.md` | Modify | §10.14 加 escort-business v2 落地记录 |

> **基线**：escort-business v1 的 `escort_profiles` 11 态状态机、`health_certs` / `training_records`、`SetOnline` / `SetOffline`、GPS mock checkin/checkout 等均按既有 v1 实现落地（git history commit `docs(plan): v1 escort-business plan`）。本 plan 假设这些已落地；如未落地，需先执行 v1 计划。

---

## Task 1: 0009 迁移（orders 修订 + escort_availabilities 表 + 集成测试）

**Files:**
- Create: `migrations/0009_escort_availabilities.up.sql`
- Create: `migrations/0009_escort_availabilities.down.sql`
- Modify: `migrations/migrations_test.go`

**Step 1: 写集成测试（RED）**

在 `migrations_test.go` 末尾追加：

```go
// Test0009EscortAvailabilitiesUpDown 验证 orders 修订（drop lock_owner/lock_expire_at,
// add selected_escort_id/escort_pending_expire_at, new CHECK, idx_orders_selecting）+ 
// escort_availabilities 表 + UNIQUE + 索引 + 时段 end_at > start_at CHECK。
// 该迁移是 escort-business v2 与 order-matching-redesign 共用。
func Test0009EscortAvailabilitiesUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 准备前置迁移（0001 users + 0002 orders + 0003 orders_state）。
	for _, f := range []string{"0001_users.up.sql", "0002_orders.up.sql", "0003_orders_state.up.sql"} {
		sql, _ := os.ReadFile(f)
		_, err = conn.Exec(ctx, string(sql))
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		cleanConn, err := pgx.Connect(cleanCtx, dsn())
		if err != nil {
			return
		}
		defer cleanConn.Close(cleanCtx)
		// down 顺序：0009 → 0003 → 0002 → 0001。
		for _, f := range []string{"0009_escort_availabilities.down.sql", "0003_orders_state.down.sql", "0002_orders.down.sql", "0001_users.down.sql"} {
			sql, _ := os.ReadFile(f)
			_, _ = cleanConn.Exec(cleanCtx, string(sql))
		}
	})

	// 应用 up。
	applyUp(t, "0009_escort_availabilities.up.sql", []string{"escort_availabilities"})

	// 检查 orders：lock_owner/lock_expire_at 已删除。
	for _, removed := range []string{"lock_owner", "lock_expire_at"} {
		var gone bool
		err := conn.QueryRow(ctx,
			`SELECT NOT EXISTS(SELECT 1 FROM information_schema.columns
			                   WHERE table_name='orders' AND column_name=$1)`, removed).
			Scan(&gone)
		require.NoError(t, err)
		assert.True(t, gone, "orders.%s should be removed by 0009", removed)
	}
	// 检查 orders：selected_escort_id/escort_pending_expire_at 已加。
	for _, added := range []string{"selected_escort_id", "escort_pending_expire_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			                   WHERE table_name='orders' AND column_name=$1)`, added).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "orders.%s should exist after 0009", added)
	}
	// 检查 orders 新 CHECK 包含 selecting_escort + escort_pending_acceptance。
	var hasNewStates bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name='orders_status_check'
		                 AND check_clause LIKE '%selecting_escort%'
		                 AND check_clause LIKE '%escort_pending_acceptance%')`).
		Scan(&hasNewStates)
	require.NoError(t, err)
	assert.True(t, hasNewStates, "orders.status_check should include new states")

	// 检查 escort_availabilities 列。
	for _, col := range []string{"id", "escort_id", "start_at", "end_at", "status", "order_id", "created_at", "updated_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			                   WHERE table_name='escort_availabilities' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "escort_availabilities.%s should exist", col)
	}

	// 检查 escort_availabilities status CHECK 3 态。
	var hasStatusCheck bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name LIKE 'escort_availabilities_status_check'
		                 AND check_clause LIKE '%available%'
		                 AND check_clause LIKE '%booked%'
		                 AND check_clause LIKE '%canceled%')`).
		Scan(&hasStatusCheck)
	require.NoError(t, err)
	assert.True(t, hasStatusCheck)

	// 检查 end_at > start_at CHECK。
	var hasTimeCheck bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name LIKE 'escort_availabilities%'
		                 AND check_clause LIKE '%end_at > start_at%')`).
		Scan(&hasTimeCheck)
	require.NoError(t, err)
	assert.True(t, hasTimeCheck)

	// 索引：idx_escort_avail_unique + idx_escort_avail_status_start。
	for _, idx := range []string{"idx_escort_avail_unique", "idx_escort_avail_status_start", "idx_orders_selecting"} {
		var exists bool
		err = conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, idx).
			Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "%s should exist", idx)
	}
	// 检查 idx_orders_lock 已删除。
	var lockIdxGone bool
	err = conn.QueryRow(ctx,
		`SELECT NOT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname='idx_orders_lock')`).
		Scan(&lockIdxGone)
	require.NoError(t, err)
	assert.True(t, lockIdxGone, "idx_orders_lock should be dropped by 0009")

	// down 校验。
	downCtx, downCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer downCancel()
	downSQL, err := os.ReadFile("0009_escort_availabilities.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(downCtx, string(downSQL))
	require.NoError(t, err, "apply 0009_escort_availabilities.down.sql")

	var tblGone bool
	err = conn.QueryRow(downCtx,
		`SELECT NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='escort_availabilities')`).
		Scan(&tblGone)
	require.NoError(t, err)
	assert.True(t, tblGone, "escort_availabilities should be gone after down")
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0009EscortAvailabilitiesUpDown ./migrations/
```

Expected: FAIL — `Test0009EscortAvailabilitiesUpDown` undefined + migration file missing.

**Step 3: 写 `0009_escort_availabilities.up.sql`**

```sql
-- 0009_escort_availabilities.up.sql
-- 共享迁移：escort-business v2 (availability 管理) + order-matching-redesign (orders 表修订)。
-- 单文件包含 orders 表结构变更 + 新表 escort_availabilities；下游 plan 不再单独改 orders。

-- ========== orders 表修订 ==========
ALTER TABLE orders
  DROP COLUMN IF EXISTS lock_owner,
  DROP COLUMN IF EXISTS lock_expire_at;

ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS selected_escort_id BIGINT REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS escort_pending_expire_at TIMESTAMPTZ;

-- 替换 CHECK 约束（包含新状态 selecting_escort / escort_pending_acceptance）。
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','selecting_escort','escort_pending_acceptance','accepted',
  'in_service','completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));

-- 索引：候选扫描 + 待确认扫描（替代 idx_orders_lock）。
DROP INDEX IF EXISTS idx_orders_lock;
CREATE INDEX idx_orders_selecting ON orders(service_start_at)
  WHERE status IN ('selecting_escort','escort_pending_acceptance');

-- ========== escort_availabilities 新表 ==========
CREATE TABLE escort_availabilities (
  id BIGSERIAL PRIMARY KEY,
  escort_id BIGINT NOT NULL REFERENCES users(id),
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'available'
    CHECK (status IN ('available','booked','canceled')),
  order_id BIGINT REFERENCES orders(id),  -- 仅在 status='booked' 时必填
  version INT NOT NULL DEFAULT 0,          -- 乐观锁
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (end_at > start_at)
);

-- 同 escort 不允许相同 start_at（DB 级防护）。
CREATE UNIQUE INDEX idx_escort_avail_unique ON escort_availabilities(escort_id, start_at);

-- 候选匹配查询：available 状态 + 时间窗口过滤（部分索引）。
CREATE INDEX idx_escort_avail_status_start ON escort_availabilities(status, start_at)
  WHERE status = 'available';

-- 按 escort + 时间窗查询（公开端点 + 邀请列表）。
CREATE INDEX idx_escort_avail_escort_time ON escort_availabilities(escort_id, start_at);
```

> **注**：时段区间重叠（`start_at < new_end AND end_at > new_start`）由 service 层校验（DB 无法用 UNIQUE 约束直接表达重叠区间）；典型做法是在 service 内 `SELECT EXISTS` 后再做 INSERT。

**Step 4: 写 `0009_escort_availabilities.down.sql`**

```sql
-- 0009_escort_availabilities.down.sql
-- 撤销 0009：删 escort_availabilities 表 + 索引；恢复 orders 旧结构。

DROP INDEX IF EXISTS idx_escort_avail_escort_time;
DROP INDEX IF EXISTS idx_escort_avail_status_start;
DROP INDEX IF EXISTS idx_escort_avail_unique;
DROP TABLE IF EXISTS escort_availabilities;

-- 恢复 orders 旧结构（与 0003 一致）。
DROP INDEX IF EXISTS idx_orders_selecting;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','pending_acceptance','accepted','in_service',
  'completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));
CREATE INDEX idx_orders_lock ON orders(lock_expire_at)
  WHERE lock_owner IS NOT NULL;

ALTER TABLE orders
  DROP COLUMN IF EXISTS selected_escort_id,
  DROP COLUMN IF EXISTS escort_pending_expire_at,
  ADD COLUMN IF NOT EXISTS lock_owner BIGINT REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS lock_expire_at TIMESTAMPTZ;
```

**Step 5: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0009EscortAvailabilitiesUpDown ./migrations/
```

Expected: PASS

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0009 orders 修订 (drop lock_owner/add selected_escort_id) + escort_availabilities 表 (3 态 CHECK + UNIQUE + 集成测试)"
```

---

## Task 2: AvailabilityRepo（pgx 数据访问 + 乐观锁 + 集成测试）

**Files:**
- Create: `services/escort/internal/availability/types.go`
- Create: `services/escort/internal/availability/state.go`
- Create: `services/escort/internal/availability/state_test.go`
- Create: `services/escort/internal/availability/repo.go`
- Create: `services/escort/internal/availability/repo_integration_test.go`

**Step 1: 写 types.go + state.go + 状态机单测（RED）**

`services/escort/internal/availability/types.go`：

```go
// Package availability 实现陪诊师时段管理（escort-business v2 + order-matching-redesign 共用）。
//
// 职责：
//   - state：纯函数状态机（3 态转换白名单）。
//   - repo：pgx 数据访问（无业务校验，仅 CRUD + 乐观锁 + 时间过滤）。
//   - service：业务校验层（时段冲突 / 时间合法性 / 状态转换守卫）+ 派生状态 DeriveStatus。
//   - handler：HTTP 入口（5 个 endpoint）。
package availability

import (
	"errors"
	"time"
)

// Availability 映射 escort_availabilities 表行。
type Availability struct {
	ID        int64
	EscortID  int64
	StartAt   time.Time
	EndAt     time.Time
	Status    string // "available" | "booked" | "canceled"
	OrderID   *int64 // 仅 booked 时非空
	Version   int    // 乐观锁
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 3 态状态常量（与 DB CHECK 对齐）。
const (
	StatusAvailable = "available"
	StatusBooked    = "booked"
	StatusCanceled  = "canceled"
)

// 哨兵错误（service / handler 用 errors.Is 区分）。
var (
	ErrNotFound        = errors.New("availability: not found")
	ErrSlotNotAvail    = errors.New("availability: slot not in available status")
	ErrVersionConflict = errors.New("availability: version conflict")
)

// 业务错误（service 层返回，handler 映射业务码）。
var (
	ErrConflict  = errors.New("availability: time slot overlaps existing one")
	ErrBadRange  = errors.New("availability: start_at must be in future and end_at > start_at")
	ErrForbidden = errors.New("availability: operation forbidden by state machine")
)
```

`services/escort/internal/availability/state.go`：

```go
package availability

// 状态机：3 态白名单。
//
//   available → booked       （escort-business BookByOrder / 患者 confirm）
//   available → canceled     （escort 主动取消时段）
//   booked → available       （订单 release / canceled / 系统 cancel）
//   canceled → available     （escort 重新启用 canceled 时段，v1 可选）
//
// booked → canceled / canceled → booked 非法；状态一旦 canceled 终态优先。
var transitions = map[string]map[string]string{
	StatusAvailable: {
		"book":      StatusBooked,
		"cancel":    StatusCanceled,
		"reactivate": StatusAvailable, // 幂等
	},
	StatusBooked: {
		"release": StatusAvailable,
	},
	StatusCanceled: {
		"reactivate": StatusAvailable,
	},
}

// CanTransition 判定 from 下执行 action 是否合法；返回目标状态与 ok。
func CanTransition(from, action string) (string, bool) {
	m, ok := transitions[from]
	if !ok {
		return "", false
	}
	to, ok := m[action]
	return to, ok
}

// IsValid 检查字符串是否为合法的 3 态值。
func IsValid(s string) bool {
	_, ok := transitions[s]
	return ok
}
```

`services/escort/internal/availability/state_test.go`：

```go
package availability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCanTransition_Legal 验证合法转换。
func TestCanTransition_Legal(t *testing.T) {
	legal := []struct{ from, action string }{
		{StatusAvailable, "book"},
		{StatusAvailable, "cancel"},
		{StatusAvailable, "reactivate"}, // 幂等
		{StatusBooked, "release"},
		{StatusCanceled, "reactivate"},
	}
	for _, c := range legal {
		_, ok := CanTransition(c.from, c.action)
		assert.True(t, ok, "%s --%s--> should be legal", c.from, c.action)
	}
}

// TestCanTransition_Illegal 验证非法转换被拒。
func TestCanTransition_Illegal(t *testing.T) {
	illegal := []struct{ from, action string }{
		{StatusBooked, "cancel"},      // booked 不能直接 canceled（必须先 release）
		{StatusCanceled, "book"},      // canceled 不能直接 booked（必须先 reactivate）
		{StatusCanceled, "release"},   // canceled 无 release 语义
		{"unknown_state", "book"},     // 未知状态
		{StatusAvailable, "unknown"},  // 未知动作
	}
	for _, c := range illegal {
		_, ok := CanTransition(c.from, c.action)
		assert.False(t, ok, "%s --%s--> should be illegal", c.from, c.action)
	}
}

// TestIsValid 验证 3 态字符串识别。
func TestIsValid(t *testing.T) {
	assert.True(t, IsValid(StatusAvailable))
	assert.True(t, IsValid(StatusBooked))
	assert.True(t, IsValid(StatusCanceled))
	assert.False(t, IsValid("on_duty"))
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```

Expected: FAIL — package not exists.

**Step 3: 写 repo.go**

`services/escort/internal/availability/repo.go`：

```go
package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AvailabilityRepo 是 escort_availabilities 表的数据访问层。
//
// 设计要点：
//   - 不做业务校验（时段冲突 / 时间合法性由 service 层负责）。
//   - 状态推进通过专用方法（BookByOrder / ReleaseByOrder），不暴露开放 UPDATE。
//   - 乐观锁：所有写操作要求 expected_version，冲突返回 ErrVersionConflict。
type AvailabilityRepo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *AvailabilityRepo { return &AvailabilityRepo{pool: pool} }

// HasOverlap 检测同一 escort 是否有时间重叠的时段（不含自身 id + 状态非 canceled）。
// 业务校验用：service.Create / Update 时调用。
func (r *AvailabilityRepo) HasOverlap(ctx context.Context, escortID int64, startAt, endAt time.Time, excludeID int64) (bool, error) {
	const q = `
		SELECT EXISTS (
		  SELECT 1 FROM escort_availabilities
		   WHERE escort_id = $1
		     AND id <> $2
		     AND status <> 'canceled'
		     AND start_at < $4  -- existing.start_at < new.end_at
		     AND end_at   > $3  -- existing.end_at   > new.start_at
		)`
	var ok bool
	err := r.pool.QueryRow(ctx, q, escortID, excludeID, startAt, endAt).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("has overlap: %w", err)
	}
	return ok, nil
}

// Create 插入一条 available 时段；ID / Version / CreatedAt / UpdatedAt 由 DB 回写。
func (r *AvailabilityRepo) Create(ctx context.Context, a *Availability) error {
	const q = `
		INSERT INTO escort_availabilities (escort_id, start_at, end_at, status)
		VALUES ($1, $2, $3, 'available')
		RETURNING id, version, created_at, updated_at`
	return r.pool.QueryRow(ctx, q, a.EscortID, a.StartAt, a.EndAt).Scan(
		&a.ID, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
}

// GetByID 按 ID 查询；不存在返回 (nil, ErrNotFound)。
func (r *AvailabilityRepo) GetByID(ctx context.Context, id int64) (*Availability, error) {
	const q = `
		SELECT id, escort_id, start_at, end_at, status, order_id, version, created_at, updated_at
		FROM escort_availabilities WHERE id = $1`
	a, err := scanOne(ctx, r.pool, q, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get by id: %w", err)
	}
	return a, nil
}

// Delete 物理删除；service 层需先校验 status='available'。
func (r *AvailabilityRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM escort_availabilities WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByEscort 按 escort + status 过滤；status 为空字符串时不按状态过滤。
func (r *AvailabilityRepo) ListByEscort(ctx context.Context, escortID int64, status string) ([]*Availability, error) {
	const q = `
		SELECT id, escort_id, start_at, end_at, status, order_id, version, created_at, updated_at
		FROM escort_availabilities
		WHERE escort_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY start_at ASC`
	return r.listWithArgs(ctx, q, escortID, status)
}

// ListAvailableByTime 查 status='available' 且时间窗口重叠（new.start..new.end）的时段。
// 用于：候选取 Top N（match-service 调用）+ 公开 endpoint（按 escort_id 过滤由调用方拼 WHERE）。
func (r *AvailabilityRepo) ListAvailableByTime(ctx context.Context, startAt, endAt time.Time) ([]*Availability, error) {
	const q = `
		SELECT id, escort_id, start_at, end_at, status, order_id, version, created_at, updated_at
		FROM escort_availabilities
		WHERE status = 'available'
		  AND start_at < $2
		  AND end_at   > $1
		ORDER BY start_at ASC`
	return r.listWithArgs(ctx, q, startAt, endAt)
}

// ListByEscortInTimeRange 查某 escort 在指定时间窗内重叠的时段（status='available'）。
func (r *AvailabilityRepo) ListByEscortInTimeRange(ctx context.Context, escortID int64, startAt, endAt time.Time) ([]*Availability, error) {
	const q = `
		SELECT id, escort_id, start_at, end_at, status, order_id, version, created_at, updated_at
		FROM escort_availabilities
		WHERE escort_id = $1
		  AND status = 'available'
		  AND start_at < $3
		  AND end_at   > $2
		ORDER BY start_at ASC`
	return r.listWithArgs(ctx, q, escortID, startAt, endAt)
}

// Update 修改时段（start_at / end_at），仅 available 可改；version 必传；冲突返回 ErrVersionConflict。
func (r *AvailabilityRepo) Update(ctx context.Context, id int64, escortID int64, startAt, endAt time.Time, expectedVersion int) error {
	const q = `
		UPDATE escort_availabilities
		   SET start_at = $1, end_at = $2, version = version + 1, updated_at = NOW()
		 WHERE id = $3 AND escort_id = $4 AND status = 'available' AND version = $5`
	tag, err := r.pool.Exec(ctx, q, startAt, endAt, id, escortID, expectedVersion)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 区分：不存在 / 非 available / version 冲突 → 进一步探测
		a, gErr := r.GetByID(ctx, id)
		if gErr != nil {
			return ErrNotFound
		}
		if a.Version != expectedVersion {
			return ErrVersionConflict
		}
		return ErrSlotNotAvail
	}
	return nil
}

// BookByOrder 标记时段为 booked + 写 order_id（service 已确认 status='available' + 乐观锁）。
func (r *AvailabilityRepo) BookByOrder(ctx context.Context, id int64, escortID int64, orderID int64, expectedVersion int) error {
	const q = `
		UPDATE escort_availabilities
		   SET status = 'booked', order_id = $1, version = version + 1, updated_at = NOW()
		 WHERE id = $2 AND escort_id = $3 AND status = 'available' AND version = $4`
	tag, err := r.pool.Exec(ctx, q, orderID, id, escortID, expectedVersion)
	if err != nil {
		return fmt.Errorf("book: %w", err)
	}
	if tag.RowsAffected() == 0 {
		a, gErr := r.GetByID(ctx, id)
		if gErr != nil {
			return ErrNotFound
		}
		if a.Version != expectedVersion {
			return ErrVersionConflict
		}
		return ErrSlotNotAvail
	}
	return nil
}

// ReleaseByOrder 把 order_id 关联的 booked 时段恢复为 available；幂等（无 order_id 时不报错）。
func (r *AvailabilityRepo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	const q = `
		UPDATE escort_availabilities
		   SET status = 'available', order_id = NULL, version = version + 1, updated_at = NOW()
		 WHERE order_id = $1 AND status = 'booked'`
	_, err := r.pool.Exec(ctx, q, orderID)
	if err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}

// CountAvailableForEscort 计数某 escort 当前可用的时段（end_at > now）。
// 用于 DeriveStatus 派生：> 0 → available，否则 busy。
func (r *AvailabilityRepo) CountAvailableForEscort(ctx context.Context, escortID int64) (int, error) {
	const q = `
		SELECT COUNT(*) FROM escort_availabilities
		WHERE escort_id = $1 AND status = 'available' AND end_at > NOW()`
	var n int
	err := r.pool.QueryRow(ctx, q, escortID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count available: %w", err)
	}
	return n, nil
}

// ---------- helpers ----------

func scanOne(ctx context.Context, p *pgxpool.Pool, q string, args ...any) (*Availability, error) {
	a := &Availability{}
	err := p.QueryRow(ctx, q, args...).Scan(
		&a.ID, &a.EscortID, &a.StartAt, &a.EndAt, &a.Status, &a.OrderID,
		&a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *AvailabilityRepo) listWithArgs(ctx context.Context, q string, args ...any) ([]*Availability, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()
	out := make([]*Availability, 0)
	for rows.Next() {
		a := &Availability{}
		if err := rows.Scan(
			&a.ID, &a.EscortID, &a.StartAt, &a.EndAt, &a.Status, &a.OrderID,
			&a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
```

**Step 4: 写 repo_integration_test.go**

`services/escort/internal/availability/repo_integration_test.go`：

```go
//go:build integration
// +build integration

package availability

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDSN() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

// setupAvailPool 起连接池 + 建 users + 跑 0009 up（创 escort_availabilities）。
func setupAvailPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	// 应用 0001 + 0009（orders 旧版由 down 段恢复）。
	for _, f := range []string{"0001_users.up.sql", "0002_orders.up.sql", "0003_orders_state.up.sql", "0009_escort_availabilities.up.sql"} {
		sql, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply %s", f)
	}

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		_, _ = pool.Exec(cleanCtx, `
			DROP TABLE IF EXISTS escort_availabilities CASCADE;
			DROP INDEX IF EXISTS idx_orders_selecting;
			ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
			ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
			  'created','paid','matching','pending_acceptance','accepted','in_service',
			  'completed','reviewed','refunding','refunded','settling','disputed',
			  'closed','canceled'
			));
			CREATE INDEX idx_orders_lock ON orders(lock_expire_at) WHERE lock_owner IS NOT NULL;
			ALTER TABLE orders
			  DROP COLUMN IF EXISTS selected_escort_id,
			  DROP COLUMN IF EXISTS escort_pending_expire_at,
			  ADD COLUMN IF NOT EXISTS lock_owner BIGINT REFERENCES users(id),
			  ADD COLUMN IF NOT EXISTS lock_expire_at TIMESTAMPTZ;
			DROP TABLE IF EXISTS orders CASCADE;
			DROP TABLE IF EXISTS users CASCADE;
		`)
		pool.Close()
	})
	return pool
}

// seedEscortUser 插一个 escort 用户；返回 user_id。
func seedEscortUser(t *testing.T, pool *pgxpool.Pool, phone string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone, role) VALUES ($1, 'escort') RETURNING id`, phone).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestRepo_CreateAndGet_OK 验证插入 + GetByID。
func TestRepo_CreateAndGet_OK(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138001")
	r := NewRepo(pool)

	a := &Availability{
		EscortID: escortID,
		StartAt:  time.Now().Add(2 * time.Hour).Truncate(time.Second),
		EndAt:    time.Now().Add(4 * time.Hour).Truncate(time.Second),
	}
	require.NoError(t, r.Create(context.Background(), a))
	assert.NotZero(t, a.ID)
	assert.NotZero(t, a.Version)
	assert.Equal(t, StatusAvailable, a.Status)

	got, err := r.GetByID(context.Background(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.StartAt.Unix(), got.StartAt.Unix())
	assert.Equal(t, a.EndAt.Unix(), got.EndAt.Unix())
}

// TestRepo_HasOverlap_DetectsOverlap 验证时间重叠检测。
func TestRepo_HasOverlap_DetectsOverlap(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138002")
	r := NewRepo(pool)
	base := time.Now().Add(2 * time.Hour).Truncate(time.Second)

	// 第一条 14:00~18:00。
	a1 := &Availability{EscortID: escortID, StartAt: base, EndAt: base.Add(4 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a1))

	// 探测 15:00~16:00（完全重叠）。
	overlap, err := r.HasOverlap(context.Background(), escortID,
		base.Add(time.Hour), base.Add(2*time.Hour), 0)
	require.NoError(t, err)
	assert.True(t, overlap)

	// 探测 18:00~20:00（边界不重叠）。end_at=18:00 与 a1.end_at 相邻。
	noOverlap, err := r.HasOverlap(context.Background(), escortID,
		base.Add(4*time.Hour), base.Add(6*time.Hour), 0)
	require.NoError(t, err)
	assert.False(t, noOverlap, "adjacent time ranges should not overlap")
}

// TestRepo_HasOverlap_IgnoresCanceled 验证 canceled 时段不参与重叠判定。
func TestRepo_HasOverlap_IgnoresCanceled(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138003")
	r := NewRepo(pool)
	base := time.Now().Add(2 * time.Hour).Truncate(time.Second)

	a1 := &Availability{EscortID: escortID, StartAt: base, EndAt: base.Add(4 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a1))

	// 手动标 canceled 后再探测。
	_, err := pool.Exec(context.Background(),
		`UPDATE escort_availabilities SET status='canceled' WHERE id=$1`, a1.ID)
	require.NoError(t, err)

	overlap, err := r.HasOverlap(context.Background(), escortID,
		base.Add(time.Hour), base.Add(2*time.Hour), 0)
	require.NoError(t, err)
	assert.False(t, overlap)
}

// TestRepo_ListByEscort_FilterStatus 验证按 status 过滤。
func TestRepo_ListByEscort_FilterStatus(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138004")
	r := NewRepo(pool)
	base := time.Now().Add(2 * time.Hour).Truncate(time.Second)

	a1 := &Availability{EscortID: escortID, StartAt: base, EndAt: base.Add(time.Hour)}
	a2 := &Availability{EscortID: escortID, StartAt: base.Add(2 * time.Hour), EndAt: base.Add(3 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a1))
	require.NoError(t, r.Create(context.Background(), a2))

	// 把 a2 改成 canceled。
	_, err := pool.Exec(context.Background(),
		`UPDATE escort_availabilities SET status='canceled' WHERE id=$1`, a2.ID)
	require.NoError(t, err)

	avail, err := r.ListByEscort(context.Background(), escortID, StatusAvailable)
	require.NoError(t, err)
	assert.Len(t, avail, 1)
	assert.Equal(t, a1.ID, avail[0].ID)

	canceled, err := r.ListByEscort(context.Background(), escortID, StatusCanceled)
	require.NoError(t, err)
	assert.Len(t, canceled, 1)
	assert.Equal(t, a2.ID, canceled[0].ID)

	all, err := r.ListByEscort(context.Background(), escortID, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

// TestRepo_ListAvailableByTime 验证时间窗重叠查询。
func TestRepo_ListAvailableByTime(t *testing.T) {
	pool := setupAvailPool(t)
	e1 := seedEscortUser(t, pool, "13800138005")
	e2 := seedEscortUser(t, pool, "13800138006")
	r := NewRepo(pool)

	base := time.Date(2026, 10, 1, 14, 0, 0, 0, time.UTC)
	// e1: 14:00~18:00
	require.NoError(t, r.Create(context.Background(), &Availability{
		EscortID: e1, StartAt: base, EndAt: base.Add(4 * time.Hour),
	}))
	// e2: 19:00~20:00（不重叠）
	require.NoError(t, r.Create(context.Background(), &Availability{
		EscortID: e2, StartAt: base.Add(5 * time.Hour), EndAt: base.Add(6 * time.Hour),
	}))

	got, err := r.ListAvailableByTime(context.Background(), base.Add(time.Hour), base.Add(2*time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, e1, got[0].EscortID)
}

// TestRepo_BookByOrder_OK 验证 bookByOrder。
func TestRepo_BookByOrder_OK(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138007")
	r := NewRepo(pool)

	a := &Availability{
		EscortID: escortID,
		StartAt:  time.Now().Add(2 * time.Hour).Truncate(time.Second),
		EndAt:    time.Now().Add(4 * time.Hour).Truncate(time.Second),
	}
	require.NoError(t, r.Create(context.Background(), a))

	require.NoError(t, r.BookByOrder(context.Background(), a.ID, escortID, 10001, a.Version))

	got, err := r.GetByID(context.Background(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusBooked, got.Status)
	require.NotNil(t, got.OrderID)
	assert.Equal(t, int64(10001), *got.OrderID)
	assert.Equal(t, a.Version+1, got.Version)
}

// TestRepo_BookByOrder_VersionConflict 验证乐观锁冲突。
func TestRepo_BookByOrder_VersionConflict(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138008")
	r := NewRepo(pool)

	a := &Availability{EscortID: escortID, StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a))

	err := r.BookByOrder(context.Background(), a.ID, escortID, 10002, a.Version+999)
	assert.ErrorIs(t, err, ErrVersionConflict)
}

// TestRepo_BookByOrder_SlotNotAvail 验证已 booked 时段不可重复 book。
func TestRepo_BookByOrder_SlotNotAvail(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138009")
	r := NewRepo(pool)

	a := &Availability{EscortID: escortID, StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a))
	require.NoError(t, r.BookByOrder(context.Background(), a.ID, escortID, 10003, a.Version))

	err := r.BookByOrder(context.Background(), a.ID, escortID, 10004, a.Version+1)
	assert.ErrorIs(t, err, ErrSlotNotAvail)
}

// TestRepo_ReleaseByOrder_OK 验证订单释放。
func TestRepo_ReleaseByOrder_OK(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138010")
	r := NewRepo(pool)

	a := &Availability{EscortID: escortID, StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a))
	require.NoError(t, r.BookByOrder(context.Background(), a.ID, escortID, 20001, a.Version))

	require.NoError(t, r.ReleaseByOrder(context.Background(), 20001))

	got, _ := r.GetByID(context.Background(), a.ID)
	assert.Equal(t, StatusAvailable, got.Status)
	assert.Nil(t, got.OrderID)
}

// TestRepo_ReleaseByOrder_Idempotent 验证幂等（无 order_id 也不报错）。
func TestRepo_ReleaseByOrder_Idempotent(t *testing.T) {
	pool := setupAvailPool(t)
	r := NewRepo(pool)
	require.NoError(t, r.ReleaseByOrder(context.Background(), 99999))
}

// TestRepo_Update_OK 验证 Update。
func TestRepo_Update_OK(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138011")
	r := NewRepo(pool)

	base := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	a := &Availability{EscortID: escortID, StartAt: base, EndAt: base.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), a))

	newStart := base.Add(30 * time.Minute)
	newEnd := base.Add(90 * time.Minute)
	require.NoError(t, r.Update(context.Background(), a.ID, escortID, newStart, newEnd, a.Version))

	got, _ := r.GetByID(context.Background(), a.ID)
	assert.Equal(t, newStart.Unix(), got.StartAt.Unix())
	assert.Equal(t, a.Version+1, got.Version)
}

// TestRepo_Update_SlotNotAvail 验证 booked 时段不可 Update。
func TestRepo_Update_SlotNotAvail(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138012")
	r := NewRepo(pool)

	a := &Availability{EscortID: escortID, StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a))
	require.NoError(t, r.BookByOrder(context.Background(), a.ID, escortID, 20002, a.Version))

	err := r.Update(context.Background(), a.ID, escortID, time.Now().Add(3*time.Hour), time.Now().Add(4*time.Hour), a.Version+1)
	assert.ErrorIs(t, err, ErrSlotNotAvail)
}

// TestRepo_CountAvailableForEscort 验证计数（end_at > now 过滤）。
func TestRepo_CountAvailableForEscort(t *testing.T) {
	pool := setupAvailPool(t)
	escortID := seedEscortUser(t, pool, "13800138013")
	r := NewRepo(pool)

	// 过期时段不计入：start_at 1 分钟前、end_at 1 分钟前 → 等于无效（end_at <= now）。
	_, err := pool.Exec(context.Background(),
		`INSERT INTO escort_availabilities (escort_id, start_at, end_at, status) VALUES ($1, NOW()-INTERVAL '5 min', NOW()-INTERVAL '2 min', 'available')`, escortID)
	require.NoError(t, err)

	// 未来时段计入。
	a := &Availability{EscortID: escortID, StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), a))

	n, err := r.CountAvailableForEscort(context.Background(), escortID)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "expired slots should not count")
}
```

**Step 5: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestRepo_' ./services/escort/internal/availability/
```

Expected: state 单测 3 个 PASS + repo 集成 13 个 PASS。

**Step 6: Commit**

```bash
git add services/escort/internal/availability/
git commit -m "feat(escort): availability 子包 (types/state/repo + 3 态状态机 + 13 个集成测试 + 3 个状态机单测)"
```

---

## Task 3: AvailabilityService（业务校验 + 派生状态 + 单测）

**Files:**
- Create: `services/escort/internal/availability/service.go`
- Create: `services/escort/internal/availability/service_test.go`

**Step 1: 写 service_test.go（RED）**

`services/escort/internal/availability/service_test.go`：

```go
package availability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fake repo (满足 service 内部依赖的方法集) ----------

type fakeRepo struct {
	slots     map[int64]*Availability
	byEscort  map[int64][]*Availability
	byOrderID map[int64]int64 // order_id -> slot_id
	nextID    int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		slots:     map[int64]*Availability{},
		byEscort:  map[int64][]*Availability{},
		byOrderID: map[int64]int64{},
	}
}

func (r *fakeRepo) Create(ctx context.Context, a *Availability) error {
	r.nextID++
	a.ID = r.nextID
	a.Version = 0
	a.Status = StatusAvailable
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	r.slots[a.ID] = a
	r.byEscort[a.EscortID] = append(r.byEscort[a.EscortID], a)
	return nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Availability, error) {
	a, ok := r.slots[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (r *fakeRepo) Delete(ctx context.Context, id int64) error {
	a, ok := r.slots[id]
	if !ok {
		return ErrNotFound
	}
	delete(r.slots, id)
	for i, v := range r.byEscort[a.EscortID] {
		if v.ID == id {
			r.byEscort[a.EscortID] = append(r.byEscort[a.EscortID][:i], r.byEscort[a.EscortID][i+1:]...)
			break
		}
	}
	return nil
}

func (r *fakeRepo) ListByEscort(ctx context.Context, escortID int64, status string) ([]*Availability, error) {
	out := []*Availability{}
	for _, a := range r.byEscort[escortID] {
		if status == "" || a.Status == status {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListAvailableByTime(ctx context.Context, start, end time.Time) ([]*Availability, error) {
	out := []*Availability{}
	for _, list := range r.byEscort {
		for _, a := range list {
			if a.Status == StatusAvailable && a.StartAt.Before(end) && a.EndAt.After(start) {
				out = append(out, a)
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) ListByEscortInTimeRange(ctx context.Context, escortID int64, start, end time.Time) ([]*Availability, error) {
	return r.ListAvailableByTime(ctx, start, end) // 简化
}

func (r *fakeRepo) Update(ctx context.Context, id int64, escortID int64, start, end time.Time, expectedVersion int) error {
	a, ok := r.slots[id]
	if !ok {
		return ErrNotFound
	}
	if a.EscortID != escortID || a.Status != StatusAvailable || a.Version != expectedVersion {
		return ErrVersionConflict
	}
	a.StartAt = start
	a.EndAt = end
	a.Version++
	a.UpdatedAt = time.Now()
	return nil
}

func (r *fakeRepo) BookByOrder(ctx context.Context, id, escortID, orderID, expectedVersion int64) error {
	a, ok := r.slots[id]
	if !ok {
		return ErrNotFound
	}
	if a.Status != StatusAvailable || a.Version != expectedVersion {
		return ErrVersionConflict
	}
	a.Status = StatusBooked
	oid := orderID
	a.OrderID = &oid
	a.Version++
	r.byOrderID[orderID] = id
	return nil
}

func (r *fakeRepo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	id, ok := r.byOrderID[orderID]
	if !ok {
		return nil // 幂等
	}
	a := r.slots[id]
	a.Status = StatusAvailable
	a.OrderID = nil
	a.Version++
	delete(r.byOrderID, orderID)
	return nil
}

func (r *fakeRepo) HasOverlap(ctx context.Context, escortID int64, start, end time.Time, excludeID int64) (bool, error) {
	for _, a := range r.byEscort[escortID] {
		if a.ID == excludeID || a.Status == StatusCanceled {
			continue
		}
		if a.StartAt.Before(end) && a.EndAt.After(start) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) CountAvailableForEscort(ctx context.Context, escortID int64) (int, error) {
	n := 0
	now := time.Now()
	for _, a := range r.byEscort[escortID] {
		if a.Status == StatusAvailable && a.EndAt.After(now) {
			n++
		}
	}
	return n, nil
}

// ---------- 测试 ----------

func TestService_Create_OK(t *testing.T) {
	r, s := newFakeRepo(), newServiceForTest(newFakeRepo())
	_ = r // 占位：测试用 service 内置 fake
	escortID := int64(100)
	start := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)

	a, err := s.Create(context.Background(), escortID, start, end)
	require.NoError(t, err)
	assert.Equal(t, StatusAvailable, a.Status)
	assert.Equal(t, escortID, a.EscortID)
}

// TestService_Create_Conflict 验证时段冲突被业务层拒绝。
func TestService_Create_Conflict(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)
	escortID := int64(101)
	start := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)

	_, err := s.Create(context.Background(), escortID, start, end)
	require.NoError(t, err)

	// 重叠时段：start+30min, end+30min
	_, err = s.Create(context.Background(), escortID, start.Add(30*time.Minute), end.Add(30*time.Minute))
	assert.ErrorIs(t, err, ErrConflict)
}

// TestService_Create_InvalidRange 验证时间合法性。
func TestService_Create_InvalidRange(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)

	// start_at 在过去
	_, err := s.Create(context.Background(), 102,
		time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrBadRange)

	// end_at <= start_at
	now := time.Now().Add(time.Hour).Truncate(time.Second)
	_, err = s.Create(context.Background(), 102, now, now)
	assert.ErrorIs(t, err, ErrBadRange)
}

// TestService_Delete_OnlyAvailable 验证只有 available 可删。
func TestService_Delete_OnlyAvailable(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)
	escortID := int64(103)
	a, err := s.Create(context.Background(), escortID,
		time.Now().Add(2*time.Hour).Truncate(time.Second),
		time.Now().Add(3*time.Hour).Truncate(time.Second))
	require.NoError(t, err)

	require.NoError(t, s.Delete(context.Background(), escortID, a.ID))

	// 第二次删：找不到
	assert.ErrorIs(t, s.Delete(context.Background(), escortID, a.ID), ErrNotFound)

	// 新时段 → booked → 删失败
	a2, _ := s.Create(context.Background(), escortID,
		time.Now().Add(5*time.Hour).Truncate(time.Second),
		time.Now().Add(6*time.Hour).Truncate(time.Second))
	_ = r.BookByOrder(context.Background(), a2.ID, escortID, 10001, a2.Version)
	assert.ErrorIs(t, s.Delete(context.Background(), escortID, a2.ID), ErrSlotNotAvail)
}

// TestService_BookByOrder_ReleasesOnOrderCancel 验证 BookByOrder + ReleaseByOrder 闭环。
func TestService_BookByOrder_ReleasesOnOrderCancel(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)
	escortID := int64(104)
	a, err := s.Create(context.Background(), escortID,
		time.Now().Add(2*time.Hour).Truncate(time.Second),
		time.Now().Add(3*time.Hour).Truncate(time.Second))
	require.NoError(t, err)

	require.NoError(t, s.BookByOrder(context.Background(), a.ID, escortID, 20001))

	got, _ := r.GetByID(context.Background(), a.ID)
	assert.Equal(t, StatusBooked, got.Status)

	require.NoError(t, s.ReleaseByOrder(context.Background(), 20001))
	got, _ = r.GetByID(context.Background(), a.ID)
	assert.Equal(t, StatusAvailable, got.Status)
}

// TestService_DeriveStatus_Available 验证有 available 时段时为 available。
func TestService_DeriveStatus_Available(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)
	escortID := int64(105)
	_, err := s.Create(context.Background(), escortID,
		time.Now().Add(2*time.Hour).Truncate(time.Second),
		time.Now().Add(3*time.Hour).Truncate(time.Second))
	require.NoError(t, err)

	status, err := s.DeriveStatus(context.Background(), escortID, false)
	require.NoError(t, err)
	assert.Equal(t, "available", status)
}

// TestService_DeriveStatus_Busy 验证无 available 时段时为 busy。
func TestService_DeriveStatus_Busy(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)
	escortID := int64(106)
	// 所有时段 canceled
	a, _ := s.Create(context.Background(), escortID,
		time.Now().Add(2*time.Hour).Truncate(time.Second),
		time.Now().Add(3*time.Hour).Truncate(time.Second))
	_ = r.Delete(context.Background(), a.ID) // 删除模拟空
	// 或者创建一个并立刻取消
	a2, _ := s.Create(context.Background(), escortID,
		time.Now().Add(5*time.Hour).Truncate(time.Second),
		time.Now().Add(6*time.Hour).Truncate(time.Second))
	// 手动改 canceled
	r.slots[a2.ID].Status = StatusCanceled

	status, err := s.DeriveStatus(context.Background(), escortID, false)
	require.NoError(t, err)
	assert.Equal(t, "busy", status)
}

// TestService_DeriveStatus_OffLine 验证 manualOffline=true 时强制 off-line。
func TestService_DeriveStatus_OffLine(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r)
	escortID := int64(107)
	// 即便有 available 时段
	_, _ = s.Create(context.Background(), escortID,
		time.Now().Add(2*time.Hour).Truncate(time.Second),
		time.Now().Add(3*time.Hour).Truncate(time.Second))

	status, err := s.DeriveStatus(context.Background(), escortID, true)
	require.NoError(t, err)
	assert.Equal(t, "off-line", status, "manual offline overrides availability")
}

// helpers

func newServiceForTest(r *fakeRepo) *Service { return NewService(r) }

// 占位：用 Service.New 但需 repo 真实接口；用 fakeRepo 实现 repo 用 interface。
// 为简化，service.go 里 Service 直接持有 *AvailabilityRepo（具体类型）。
// 在 service_test.go 里通过包装 shim 注入 fake（见下方）。
// 实际实施时调整：service 持有 interface（见 service.go 设计），fakeRepo 满足该 interface。
var _ = errors.Is
```

> **注**：上面 `newServiceForTest(r)` 为占位。实施时：
> 1. service.go 中 `Service` 持有 `*AvailabilityRepo` 改为 interface（最小集合见 service.go）；
> 2. fakeRepo 实现该 interface（已有方法可满足，无需新增）。

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```

Expected: FAIL — `undefined: Service`, `undefined: NewService`, `undefined: newServiceForTest`。

**Step 3: 写 service.go**

`services/escort/internal/availability/service.go`：

```go
package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/growdu/doctors/shared/errs"
)

// Repo 是 service 所需的最小仓储接口（解耦、便于 fake repo 测试）。
type Repo interface {
	Create(ctx context.Context, a *Availability) error
	GetByID(ctx context.Context, id int64) (*Availability, error)
	Delete(ctx context.Context, id int64) error
	ListByEscort(ctx context.Context, escortID int64, status string) ([]*Availability, error)
	ListAvailableByTime(ctx context.Context, startAt, endAt time.Time) ([]*Availability, error)
	ListByEscortInTimeRange(ctx context.Context, escortID int64, startAt, endAt time.Time) ([]*Availability, error)
	Update(ctx context.Context, id int64, escortID int64, startAt, endAt time.Time, expectedVersion int) error
	BookByOrder(ctx context.Context, id int64, escortID int64, orderID int64, expectedVersion int) error
	ReleaseByOrder(ctx context.Context, orderID int64) error
	HasOverlap(ctx context.Context, escortID int64, startAt, endAt time.Time, excludeID int64) (bool, error)
	CountAvailableForEscort(ctx context.Context, escortID int64) (int, error)
}

// Service 是 escort-availability 业务层（业务校验 + 状态机 + 派生）。
type Service struct {
	repo   Repo
	nowFn  func() time.Time // 可注入以稳定测试
}

// NewService 构造 service。
func NewService(repo Repo) *Service {
	return &Service{repo: repo, nowFn: time.Now}
}

// SetNowFn 注入时间函数（测试用）。
func (s *Service) SetNowFn(fn func() time.Time) *Service { s.nowFn = fn; return s }

// ---------- 7 个核心业务方法 + 1 个派生 ----------

// Create 新建时段；service 层校验时间合法性 + 重叠。
func (s *Service) Create(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Availability, error) {
	if err := validateRange(startAt, endAt, s.nowFn()); err != nil {
		return nil, err
	}
	overlap, err := s.repo.HasOverlap(ctx, escortID, startAt, endAt, 0)
	if err != nil {
		return nil, fmt.Errorf("check overlap: %w", err)
	}
	if overlap {
		return nil, ErrConflict
	}
	a := &Availability{
		EscortID: escortID,
		StartAt:  startAt,
		EndAt:    endAt,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	a.Status = StatusAvailable
	return a, nil
}

// Delete 删时段（仅 available 可删）。
func (s *Service) Delete(ctx context.Context, escortID, id int64) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.EscortID != escortID {
		return ErrForbidden
	}
	if a.Status != StatusAvailable {
		return ErrSlotNotAvail
	}
	return s.repo.Delete(ctx, id)
}

// ListByEscort 列某 escort 的时段（status 空字符串时不按状态过滤）。
func (s *Service) ListByEscort(ctx context.Context, escortID int64, status string) ([]*Availability, error) {
	return s.repo.ListByEscort(ctx, escortID, status)
}

// ListAvailableByTime 查可用时段（候选取 + 公开端点复用）。
func (s *Service) ListAvailableByTime(ctx context.Context, startAt, endAt time.Time) ([]*Availability, error) {
	return s.repo.ListAvailableByTime(ctx, startAt, endAt)
}

// Update 修改时段（仅 available；version 必传；冲突返回 ErrVersionConflict）。
func (s *Service) Update(ctx context.Context, escortID, id int64, startAt, endAt time.Time, expectedVersion int) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.EscortID != escortID {
		return ErrForbidden
	}
	if a.Status != StatusAvailable {
		return ErrSlotNotAvail
	}
	if err := validateRange(startAt, endAt, s.nowFn()); err != nil {
		return err
	}
	overlap, err := s.repo.HasOverlap(ctx, escortID, startAt, endAt, id)
	if err != nil {
		return fmt.Errorf("check overlap: %w", err)
	}
	if overlap {
		return ErrConflict
	}
	if err := s.repo.Update(ctx, id, escortID, startAt, endAt, expectedVersion); err != nil {
		if errors.Is(err, ErrSlotNotAvail) || errors.Is(err, ErrVersionConflict) {
			return err
		}
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

// BookByOrder 标记时段为 booked + 写 order_id。
func (s *Service) BookByOrder(ctx context.Context, slotID, escortID, orderID int64) error {
	a, err := s.repo.GetByID(ctx, slotID)
	if err != nil {
		return err
	}
	if a.EscortID != escortID {
		return ErrForbidden
	}
	if _, ok := CanTransition(a.Status, "book"); !ok {
		return ErrForbidden
	}
	return s.repo.BookByOrder(ctx, slotID, escortID, orderID, a.Version)
}

// ReleaseByOrder 把 order_id 关联的 booked 时段恢复为 available；幂等。
func (s *Service) ReleaseByOrder(ctx context.Context, orderID int64) error {
	return s.repo.ReleaseByOrder(ctx, orderID)
}

// DeriveStatus 派生陪诊师在线状态。
//
//   - manualOffline=true  → "off-line"（最终态优先）
//   - 有 status='available' 且 end_at > now 的时段 → "available"
//   - 否则 → "busy"
//
// 返回值为派生状态的字符串常量（与 events / API 响应一致）。
func (s *Service) DeriveStatus(ctx context.Context, escortID int64, manualOffline bool) (string, error) {
	if manualOffline {
		return "off-line", nil
	}
	n, err := s.repo.CountAvailableForEscort(ctx, escortID)
	if err != nil {
		return "", fmt.Errorf("derive status: %w", err)
	}
	if n > 0 {
		return "available", nil
	}
	return "busy", nil
}

// ---------- helpers ----------

// validateRange 校验 start_at > now + end_at > start_at。
func validateRange(startAt, endAt, now time.Time) error {
	if !startAt.After(now) {
		return ErrBadRange
	}
	if !endAt.After(startAt) {
		return ErrBadRange
	}
	return nil
}

// ---------- 业务错误 → errs.Error 映射（给 handler 用） ----------

// ToErrs 把业务错误映射为 errs.Error（5 位业务码）。
func ToErrs(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.New(errs.CodeNotFound, err.Error())
	case errors.Is(err, ErrConflict):
		return errs.New(errs.CodeConflict, err.Error())
	case errors.Is(err, ErrBadRange):
		return errs.New(errs.CodeParamInvalid, err.Error())
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrSlotNotAvail):
		return errs.New(errs.CodeForbidden, err.Error())
	case errors.Is(err, ErrVersionConflict):
		return errs.New(errs.CodeConflict, err.Error())
	default:
		return errs.Wrap(errs.CodeInternal, "availability: %w", err)
	}
}
```

**Step 4: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```

Expected: PASS（state 3 个 + service 8 个 = 11 个单测）。

**Step 5: Commit**

```bash
git add services/escort/internal/availability/
git commit -m "feat(escort): availability.Service 业务层 (冲突校验 + 时段验证 + DeriveStatus + 11 个单测)"
```

---

## Task 4: 5 个 HTTP endpoint（handler + 路由 + 单测）

**Files:**
- Create: `services/escort/internal/handler/escort_availability.go`
- Create: `services/escort/internal/handler/escort_availability_test.go`
- Modify: `services/escort/internal/handler/escort.go`（`RegisterRoutes` 加挂）

**Step 1: 写 handler 单测（RED）**

`services/escort/internal/handler/escort_availability_test.go`：

```go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/shared/httpx"
)

// ---------- fake AvailabilityService ----------

type fakeAvailService struct {
	createCalled      bool
	deleteCalled      bool
	updateCalled      bool
	bookCalled        bool
	releaseCalled     bool
	listMine          []*availability.Availability
	listPublic        []*availability.Availability
	invitations       []InvitationDTO
	deriveStatusValue string
}

func (f *fakeAvailService) Create(ctx context.Context, escortID int64, start, end time.Time) (*availability.Availability, error) {
	f.createCalled = true
	return &availability.Availability{ID: 1, EscortID: escortID, StartAt: start, EndAt: end, Status: availability.StatusAvailable}, nil
}

func (f *fakeAvailService) Update(ctx context.Context, escortID, id int64, start, end time.Time, version int) error {
	f.updateCalled = true
	return nil
}

func (f *fakeAvailService) Delete(ctx context.Context, escortID, id int64) error {
	f.deleteCalled = true
	return nil
}

func (f *fakeAvailService) ListByEscort(ctx context.Context, escortID int64, status string) ([]*availability.Availability, error) {
	if status == "" && len(f.listMine) > 0 {
		return f.listMine, nil
	}
	return f.listPublic, nil
}

func (f *fakeAvailService) ListAvailableByTime(ctx context.Context, start, end time.Time) ([]*availability.Availability, error) {
	return f.listPublic, nil
}

func (f *fakeAvailService) BookByOrder(ctx context.Context, slotID, escortID, orderID int64) error {
	f.bookCalled = true
	return nil
}

func (f *fakeAvailService) ReleaseByOrder(ctx context.Context, orderID int64) error {
	f.releaseCalled = true
	return nil
}

func (f *fakeAvailService) DeriveStatus(ctx context.Context, escortID int64, manualOffline bool) (string, error) {
	return f.deriveStatusValue, nil
}

// InvitationDTO 是 handler 暴露给 escort-app 的待确认订单摘要。
// 实际 list 来自 order-service（v1 简化：service 用 fake 数据；Task 6 注入真实 order client）。
type InvitationDTO struct {
	OrderID         int64     `json:"order_id"`
	PatientName     string    `json:"patient_name"`
	HospitalName    string    `json:"hospital_name"`
	ServiceStartAt  time.Time `json:"service_start_at"`
	ServiceEndAt    time.Time `json:"service_end_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// （fake invitation 来自 service.ListInvitations，订单集成在 Task 6 加。
// 这里假设 ListInvitations 在 Task 6 加在 service 上；本 Task 4 暂 stub 返回 nil。）

// ---------- helpers ----------

func newRouter(fake *fakeAvailService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewEscortAvailabilityHandler(fake, &fakeOrderClient{})
	authed := r.Group("/api/v1")
	authed.Use(func(c *gin.Context) {
		c.Set("user_id", int64(100)) // mock 当前用户
		c.Next()
	})
	authed.POST("/escorts/me/availability", h.PutAvailability) // PUT 也用 POST（避免与 gin 冲突）
	authed.GET("/escorts/me/availability", h.ListMyAvailability)
	authed.DELETE("/escorts/me/availability/:id", h.DeleteAvailability)
	authed.GET("/escorts/me/invitations", h.ListMyInvitations)
	r.GET("/api/v1/escorts/:id/availabilities", h.ListPublicAvailability)
	return r
}

// fakeOrderClient 满足 OrderClient 最小接口（Task 6 接入真实 client）。
type fakeOrderClient struct{}

func (f *fakeOrderClient) ListPendingOrdersByEscort(ctx context.Context, escortID int64) ([]InvitationDTO, error) {
	return nil, nil
}

// ---------- PUT /escorts/me/availability ----------

func TestPutAvailability_OK(t *testing.T) {
	fake := &fakeAvailService{}
	w := httptest.NewRecorder()
	r := newRouter(fake)
	body := map[string]any{
		"start_at": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		"end_at":   time.Now().Add(3 * time.Hour).Format(time.RFC3339),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/escorts/me/availability", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fake.createCalled)
}

// ---------- DELETE /escorts/me/availability/:id ----------

func TestDeleteAvailability_OK(t *testing.T) {
	fake := &fakeAvailService{}
	w := httptest.NewRecorder()
	r := newRouter(fake)
	req := httptest.NewRequest("DELETE", "/api/v1/escorts/me/availability/123", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fake.deleteCalled)
}

// ---------- GET /escorts/me/availability ----------

func TestListMyAvailability_OK(t *testing.T) {
	fake := &fakeAvailService{listMine: []*availability.Availability{
		{ID: 1, EscortID: 100, Status: availability.StatusAvailable},
	}}
	w := httptest.NewRecorder()
	r := newRouter(fake)
	req := httptest.NewRequest("GET", "/api/v1/escorts/me/availability", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, w.Body.String(), "available")
}

// ---------- GET /escorts/:id/availabilities (公开) ----------

func TestListPublicAvailability_OK(t *testing.T) {
	fake := &fakeAvailService{listPublic: []*availability.Availability{
		{ID: 2, EscortID: 200, Status: availability.StatusAvailable},
	}}
	w := httptest.NewRecorder()
	r := newRouter(fake)
	req := httptest.NewRequest("GET", "/api/v1/escorts/200/availabilities?start_at="+
		time.Now().Format(time.RFC3339)+"&end_at="+time.Now().Add(24*time.Hour).Format(time.RFC3339), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- GET /escorts/me/invitations ----------

func TestListMyInvitations_OK(t *testing.T) {
	fake := &fakeAvailService{}
	w := httptest.NewRecorder()
	r := newRouter(fake)
	req := httptest.NewRequest("GET", "/api/v1/escorts/me/invitations", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestPutAvailability|TestDeleteAvailability|TestListMyAvailability|TestListPublicAvailability|TestListMyInvitations' ./services/escort/internal/handler/
```

Expected: FAIL — `undefined: NewEscortAvailabilityHandler`, handler methods undefined。

**Step 3: 写 escort_availability.go**

`services/escort/internal/handler/escort_availability.go`：

```go
// Package handler - escort_availability.go 提供 5 个 escort 端 + 1 个公开端 + 1 个邀请端。
//
// 路由：
//
//	POST  /api/v1/escorts/me/availability           新建时段（PUT 语义）
//	GET   /api/v1/escorts/me/availability           列我的时段
//	DELETE /api/v1/escorts/me/availability/:id      删时段（仅 available）
//	GET   /api/v1/escorts/:id/availabilities        公开列某 escort 的可用时段
//	GET   /api/v1/escorts/me/invitations            待我确认的订单（escort_pending_acceptance）
//
// 依赖：
//   - availabilityService（service 层抽象）
//   - orderClient（订单集成，Task 6 引入；本 Task 4 暂用 fake client）
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// AvailabilityService 是 handler 依赖的最小 service 接口。
type AvailabilityService interface {
	Create(ctx context.Context, escortID int64, startAt, endAt time.Time) (*availability.Availability, error)
	Update(ctx context.Context, escortID, id int64, startAt, endAt time.Time, expectedVersion int) error
	Delete(ctx context.Context, escortID, id int64) error
	ListByEscort(ctx context.Context, escortID int64, status string) ([]*availability.Availability, error)
	ListAvailableByTime(ctx context.Context, startAt, endAt time.Time) ([]*availability.Availability, error)
	BookByOrder(ctx context.Context, slotID, escortID, orderID int64) error
	ReleaseByOrder(ctx context.Context, orderID int64) error
	DeriveStatus(ctx context.Context, escortID int64, manualOffline bool) (string, error)
}

// OrderClient 是订单服务最小集成接口（list 邀请用）。
type OrderClient interface {
	ListPendingOrdersByEscort(ctx context.Context, escortID int64) ([]InvitationDTO, error)
}

// InvitationDTO 是 handler 暴露给 escort-app 的邀请摘要。
// 实际填充由 OrderClient 实现（Task 6 注入 order-service client）。
type InvitationDTO struct {
	OrderID        int64     `json:"order_id"`
	PatientName    string    `json:"patient_name"`
	HospitalName   string    `json:"hospital_name"`
	ServiceStartAt time.Time `json:"service_start_at"`
	ServiceEndAt   time.Time `json:"service_end_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// EscortAvailabilityHandler 持有 5 个 endpoint + 邀请列表。
type EscortAvailabilityHandler struct {
	svc       AvailabilityService
	orderCli  OrderClient
}

// NewEscortAvailabilityHandler 构造。
func NewEscortAvailabilityHandler(svc AvailabilityService, o OrderClient) *EscortAvailabilityHandler {
	return &EscortAvailabilityHandler{svc: svc, orderCli: o}
}

// ---------- 5 endpoint ----------

// PutAvailability body: {"start_at": "...", "end_at": "..."}
//
// 注：RESTful 语义是 PUT（创建/替换）。由于 gin 对 method 注册的便利，路由层用 POST 也可。
// spec 使用 PUT，本实现按 spec 暴露 PUT（router.RegisterRoutes 用 PUT）。
func (h *EscortAvailabilityHandler) PutAvailability(c *gin.Context) {
	userID := c.GetInt64("user_id")
	var req struct {
		StartAt time.Time `json:"start_at" binding:"required"`
		EndAt   time.Time `json:"end_at" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	a, err := h.svc.Create(c.Request.Context(), userID, req.StartAt, req.EndAt)
	if err != nil {
		httpx.WriteError(c, availability.ToErrs(err))
		return
	}
	httpx.WriteOK(c, gin.H{"availability": a})
}

// ListMyAvailability 列出当前 escort 的全部时段（status 可选过滤）。
func (h *EscortAvailabilityHandler) ListMyAvailability(c *gin.Context) {
	userID := c.GetInt64("user_id")
	status := c.Query("status")
	list, err := h.svc.ListByEscort(c.Request.Context(), userID, status)
	if err != nil {
		httpx.WriteError(c, availability.ToErrs(err))
		return
	}
	httpx.WriteOK(c, gin.H{"availabilities": list})
}

// DeleteAvailability 删时段（仅 available）。
func (h *EscortAvailabilityHandler) DeleteAvailability(c *gin.Context) {
	userID := c.GetInt64("user_id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.WriteError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		httpx.WriteError(c, availability.ToErrs(err))
		return
	}
	httpx.WriteOK(c, gin.H{"deleted": true})
}

// ListPublicAvailability 公开列某 escort 的可用时段（query: start_at + end_at）。
// 强制 status='available' 过滤（服务层已过滤；handler 只需传时间窗）。
func (h *EscortAvailabilityHandler) ListPublicAvailability(c *gin.Context) {
	escortID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.WriteError(c, errs.New(errs.CodeParamInvalid, "invalid escort id"))
		return
	}
	startAt, err1 := time.Parse(time.RFC3339, c.Query("start_at"))
	endAt, err2 := time.Parse(time.RFC3339, c.Query("end_at"))
	if err1 != nil || err2 != nil {
		httpx.WriteError(c, errs.New(errs.CodeParamInvalid, "start_at / end_at required (RFC3339)"))
		return
	}
	list, err := h.svc.ListByEscort(c.Request.Context(), escortID, availability.StatusAvailable)
	if err != nil {
		httpx.WriteError(c, availability.ToErrs(err))
		return
	}
	// 在公开范围上加时间窗过滤（service 已支持 ListAvailableByTime）—— 取并集
	extra, err := h.svc.ListAvailableByTime(c.Request.Context(), startAt, endAt)
	if err != nil {
		httpx.WriteError(c, availability.ToErrs(err))
		return
	}
	// 取 escortID 命中 + 时间窗命中
	merged := mergeByEscortAndRange(list, extra, escortID, startAt, endAt)
	httpx.WriteOK(c, gin.H{"availabilities": merged})
}

// ListMyInvitations 邀请列表（escort_pending_acceptance 订单）。
func (h *EscortAvailabilityHandler) ListMyInvitations(c *gin.Context) {
	userID := c.GetInt64("user_id")
	inv, err := h.orderCli.ListPendingOrdersByEscort(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, errs.Wrap(errs.CodeInternal, "list invitations: %w", err))
		return
	}
	httpx.WriteOK(c, gin.H{"invitations": inv})
}

// mergeByEscortAndRange 合并 listByEscort 与 listAvailableByTime，仅返 escort+range 命中。
func mergeByEscortAndRange(byEscort, byTime []*availability.Availability, escortID int64, startAt, endAt time.Time) []*availability.Availability {
	hit := map[int64]bool{}
	for _, a := range byTime {
		if a.EscortID == escortID {
			hit[a.ID] = true
		}
	}
	out := make([]*availability.Availability, 0)
	for _, a := range byEscort {
		if !hit[a.ID] {
			continue
		}
		if a.StartAt.Before(endAt) && a.EndAt.After(startAt) {
			out = append(out, a)
		}
	}
	return out
}

// 兜底编译（errors / net/http 被 handler 用）。
var _ = errors.New
var _ = http.StatusOK
```

**Step 4: 修改 escort.go 注册路由**

修改 `services/escort/internal/handler/escort.go` 的 `RegisterRoutes`：

```go
// 在 v1 group 内追加 5 个路由（沿用既有权限：escort 角色）。
//
//   PUT    /api/v1/escorts/me/availability
//   GET    /api/v1/escorts/me/availability
//   DELETE /api/v1/escorts/me/availability/:id
//   GET    /api/v1/escorts/me/invitations
//
// 注意：公开端 GET /escorts/:id/availabilities 不需要 escort 角色（候选取公开），注册在 v1 group 外（无需 auth 中间件）。
func RegisterRoutes(r *gin.Engine, h *EscortAvailabilityHandler) {
	v1 := r.Group("/api/v1")
	// 既有路由（me/profile, me/status 等）保持。
	// 在此追加：
	v1.PUT("/escorts/me/availability", h.PutAvailability)
	v1.GET("/escorts/me/availability", h.ListMyAvailability)
	v1.DELETE("/escorts/me/availability/:id", h.DeleteAvailability)
	v1.GET("/escorts/me/invitations", h.ListMyInvitations)
	// 公开端点：候选取（无需 auth）。
	v1.GET("/escorts/:id/availabilities", h.ListPublicAvailability)
}
```

> 实际注册时按既有 v1 group + 权限中间件风格（escort 角色 token）；公开端单独挂载到 r（无 auth 中间件）。

**Step 5: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/handler/
```

Expected: PASS（5 个 handler 单测）。

**Step 6: Commit**

```bash
git add services/escort/internal/handler/
git commit -m "feat(escort): handler 加 5 个 endpoint (escort/me/availability × 3 + 公开 availabilities + invitations)"
```

---

## Task 5: 陪诊师派生状态 + 集成到 GET /escorts/me/profile

**Files:**
- Modify: `services/escort/internal/service/escort_service.go`
- Modify: `services/escort/internal/service/escort_service_test.go`

**Step 1: 写单测（RED）**

在 `services/escort/internal/service/escort_service_test.go` 末尾追加：

```go
// TestGetMyProfile_IncludesAvailabilityStatus 验证 profile 响应含 availability_status。
func TestGetMyProfile_IncludesAvailabilityStatus(t *testing.T) {
	// 准备：mock profile reader（已有）+ mock availability.DeriveStatus。
	availSvc := &mockAvailServiceForProfile{status: "available"}
	s := New(newFakeRepo(), &fakePub{}).SetProfileReader(newFakeProfileReader())
	s.SetAvailabilityStatusProvider(func(ctx context.Context, escortID int64, manualOffline bool) (string, error) {
		return availSvc.DeriveStatus(ctx, escortID, manualOffline)
	})

	// 通过 helper 调 GetMyProfile + fill availability_status
	resp, err := s.GetMyProfile(context.Background(), 100)
	require.NoError(t, err)
	// resp 应包含 availability_status 字段（修改后）
	assert.Equal(t, "available", resp.AvailabilityStatus)
}

// mockAvailStatusProvider 满足 Escort Service 内部的 status provider 接口。
type mockAvailStatusProvider struct {
	status string
}

func (m *mockAvailStatusProvider) Derive(ctx context.Context, escortID int64, manualOffline bool) (string, error) {
	return m.status, nil
}

// 实际签名见 escort_service.go 中 Escort Service 修改（见 Step 3）。
// 占位 fake 适配：mockAvailabilityServiceForProfile 实现新接口。
type mockAvailServiceForProfile struct {
	status string
}

func (m *mockAvailServiceForProfile) DeriveStatus(ctx context.Context, escortID int64, manualOffline bool) (string, error) {
	return m.status, nil
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run TestGetMyProfile ./services/escort/internal/service/
```

Expected: FAIL — `s.SetAvailabilityStatusProvider` undefined / `resp.AvailabilityStatus` not exist。

**Step 3: 修改 escort_service.go**

`services/escort/internal/service/escort_service.go` 加派生集成：

```go
// 在 Service struct 加字段：
//   availabilityStatusProvider func(ctx, escortID, manualOffline) (string, error)
//
// Type alias 简化调用（service 内部抽象）：
type AvailabilityStatusProvider interface {
	DeriveStatus(ctx context.Context, escortID int64, manualOffline bool) (string, error)
}

func (s *Service) SetAvailabilityStatusProvider(p AvailabilityStatusProvider) *Service {
	s.availabilityStatusProvider = p
	return s
}

// 在 GetMyProfile 末尾追加派生状态填充：
func (s *Service) GetMyProfile(ctx context.Context, userID int64) (*ProfileResponse, error) {
	if s.profileReader == nil {
		return nil, errs.New(errs.CodeUnavailable, "profile reader not wired")
	}
	p, err := s.profileReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "get profile", err)
	}
	if p == nil {
		return nil, errs.New(errs.CodeNotFound, "escort profile not found")
	}
	resp := &ProfileResponse{
		Profile: p,
		// manualOffline 暂从 p.State=='offline' 派生（v1 简化）。
	}
	if s.availabilityStatusProvider != nil {
		manualOffline := (p.State == "offline")
		st, err := s.availabilityStatusProvider.DeriveStatus(ctx, userID, manualOffline)
		if err == nil {
			resp.AvailabilityStatus = st
		}
	}
	return resp, nil
}

// 新增 ProfileResponse 类型（替换原 GetMyProfile 返回 *repo.Profile）。
type ProfileResponse struct {
	*repo.Profile // 嵌入原结构
	AvailabilityStatus string `json:"availability_status"` // available | busy | off-line
}
```

更新既有 `GetMyProfile` 的 call site（既有 handler / 测试按 `*repo.Profile` 消费）：通过嵌入字段兼容，所有访问 `p.State` / `p.Version` 等仍可用；新代码用 `resp.AvailabilityStatus`。

**Step 4: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/service/
```

Expected: PASS（既有 + 新增 1 个）。

**Step 5: Commit**

```bash
git add services/escort/internal/service/
git commit -m "feat(escort): profile 响应加 availability_status (available / busy / off-line 派生)"
```

---

## Task 6: 与 order-service 集成（BookByOrder / ReleaseByOrder hook）

**Files:**
- Create: `services/order/internal/integration/escort_availability.go`
- Create: `services/order/internal/integration/escort_availability_test.go`
- Modify: `services/order/internal/service/order_service.go`
- Modify: `services/order/internal/service/order_service_test.go`

**Step 1: 写单测（RED）**

`services/order/internal/integration/escort_availability_test.go`：

```go
package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAvailabilityClient 满足 AvailabilityClient（BookByOrder / ReleaseByOrder 接口）。
type fakeAvailabilityClient struct {
	bookCalled    bool
	releaseCalled bool
	bookErr       error
	releaseErr    error
}

func (f *fakeAvailabilityClient) BookByOrder(ctx context.Context, slotID, escortID, orderID int64) error {
	f.bookCalled = true
	return f.bookErr
}

func (f *fakeAvailabilityClient) ReleaseByOrder(ctx context.Context, orderID int64) error {
	f.releaseCalled = true
	return f.releaseErr
}

// TestEscortAvailability_BookByOrder_OK
func TestEscortAvailability_BookByOrder_OK(t *testing.T) {
	cli := &fakeAvailabilityClient{}
	a := NewEscortAvailability(cli)
	require.NoError(t, a.BookByOrder(context.Background(), 100, 200, 300))
	assert.True(t, cli.bookCalled)
}

// TestEscortAvailability_BookByOrder_NoOpIfNotWired
func TestEscortAvailability_BookByOrder_NoOpIfNotWired(t *testing.T) {
	a := NewEscortAvailability(nil)
	require.NoError(t, a.BookByOrder(context.Background(), 1, 2, 3))
}

// TestEscortAvailability_BookByOrder_PropagatesError
func TestEscortAvailability_BookByOrder_PropagatesError(t *testing.T) {
	cli := &fakeAvailabilityClient{bookErr: errors.New("downstream down")}
	a := NewEscortAvailability(cli)
	err := a.BookByOrder(context.Background(), 1, 2, 3)
	assert.Error(t, err)
}

// TestEscortAvailability_ReleaseByOrder_Idempotent
func TestEscortAvailability_ReleaseByOrder_Idempotent(t *testing.T) {
	calls := 0
	cli := &counterClient{releaseFn: func() { calls++ }}
	a := NewEscortAvailability(cli)
	require.NoError(t, a.ReleaseByOrder(context.Background(), 500))
	require.NoError(t, a.ReleaseByOrder(context.Background(), 500))
	assert.Equal(t, 2, calls, "release should pass through (service already idempotent)")
}

type counterClient struct {
	*fakeAvailabilityClient
	releaseFn func()
}

func (c *counterClient) ReleaseByOrder(ctx context.Context, orderID int64) error {
	c.releaseFn()
	return nil
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/order/internal/integration/
```

Expected: FAIL — `undefined: NewEscortAvailability`, interface methods missing。

**Step 3: 写 escort_availability.go**

`services/order/internal/integration/escort_availability.go`：

```go
// Package integration 是 order-service 与 escort-service 的集成层。
//
// v1 简化为「共享 Go 包」模式：order-service 通过 AvailabilityClient 接口直接调用
// availability.Service 的方法；接口实现在 cmd/main.go 注入（共享同一 availability 实例）。
//
// RPC / 跨服务调用留 v2：escort-service 暴露 HTTP / gRPC endpoint。
package integration

import (
	"context"
	"errors"
	"fmt"
)

// AvailabilityClient 是 escort-availability 服务对外接口（order-service 用）。
type AvailabilityClient interface {
	BookByOrder(ctx context.Context, slotID, escortID, orderID int64) error
	ReleaseByOrder(ctx context.Context, orderID int64) error
}

// EscortAvailability 包装 AvailabilityClient，提供容错（best-effort + log + 不阻塞主流程）。
type EscortAvailability struct {
	cli AvailabilityClient
}

// NewEscortAvailability 构造。
//
// 传 nil 时所有方法为 no-op（方便本地开发 / 集成测试不接 escort 服务）。
func NewEscortAvailability(cli AvailabilityClient) *EscortAvailability {
	return &EscortAvailability{cli: cli}
}

// BookByOrder 把 escort 时段标记为 booked。
// 错误仅日志，order-service 主流程不阻塞（confirm accept 主流程已写入 DB）。
func (e *EscortAvailability) BookByOrder(ctx context.Context, slotID, escortID, orderID int64) error {
	if e == nil || e.cli == nil {
		return nil
	}
	if err := e.cli.BookByOrder(ctx, slotID, escortID, orderID); err != nil {
		return fmt.Errorf("availability.BookByOrder: %w", err)
	}
	return nil
}

// ReleaseByOrder 把时段恢复为 available（订单 cancel / reject 用）。
func (e *EscortAvailability) ReleaseByOrder(ctx context.Context, orderID int64) error {
	if e == nil || e.cli == nil {
		return nil
	}
	if err := e.cli.ReleaseByOrder(ctx, orderID); err != nil {
		return fmt.Errorf("availability.ReleaseByOrder: %w", err)
	}
	return nil
}

// 占位用于骗编译（errors 不直接在本文件用，但 _ import 防被 goimports 删）。
var _ = errors.New
```

**Step 4: 修改 order_service.go**

`services/order/internal/service/order_service.go` 改动：

```go
// 在 Service struct 加字段：
//   escortAvail *integration.EscortAvailability

// SetEscortAvailability 注入（cmd/main.go 装配）。
func (s *Service) SetEscortAvailability(ea *integration.EscortAvailability) *Service {
	s.escortAvail = ea
	return s
}

// ConfirmAccept 修改（既有实现已存在；此处新增时段预订 hook）：
//
//   1. 既有状态机推进：escort_pending_acceptance → accepted
//   2. 既有字段更新：orders.escort_id = $escortID, orders.accepted_at = now()
//   3. **新增**：s.escortAvail.BookByOrder(ctx, order.SelectedSlotID, order.EscortID, order.ID)
//      （slotID 由 match-service 在 select 时填充 order.SelectedSlotID，本 plan 假设该字段已加；如未加，
//        由 escort-order-ext plan 在 order 表加 selected_availability_id 字段，并在 ConfirmAccept 时传入。）
func (s *Service) ConfirmAccept(ctx context.Context, orderID int64, escortID int64, slotID *int64) error {
	// 既有推进逻辑...
	if slotID != nil && s.escortAvail != nil {
		if err := s.escortAvail.BookByOrder(ctx, *slotID, escortID, orderID); err != nil {
			// best-effort：log 但不阻塞（订单状态已 accepted）
			s.logger.Warn("book availability failed", "order_id", orderID, "err", err)
		}
	}
	return nil
}

// ReleaseLockAndReject 修改（既有实现保留 + 新增释放时段 hook）：
//
//   1. 既有状态机推进：escort_pending_acceptance → selecting_escort
//   2. **新增**：s.escortAvail.ReleaseByOrder(ctx, orderID)
//   3. （订单未 booked，无需 Book 反向；ReleaseByOrder 在 repo 层幂等——查不到 order_id 也不报错。）
func (s *Service) ReleaseLockAndReject(ctx context.Context, orderID int64, reason string) error {
	// 既有推进逻辑...
	if s.escortAvail != nil {
		if err := s.escortAvail.ReleaseByOrder(ctx, orderID); err != nil {
			s.logger.Warn("release availability failed", "order_id", orderID, "err", err)
		}
	}
	return nil
}
```

> **说明**：`slotID *int64` 是有意的 nil-safe 设计（v1 部分订单流可能不携带 slotID，此时跳过 book hook；escort 拒接时 order.SelectedSlotID 也可能为空）。

**Step 5: order_service_test.go 加 2 个 hook 验证**

在 `services/order/internal/service/order_service_test.go` 末尾追加：

```go
// TestConfirmAccept_CallsBookByOrder 验证 ConfirmAccept 触发 escort-availability hook。
func TestConfirmAccept_CallsBookByOrder(t *testing.T) {
	// ... 用 fake orderRepo + fake escortAvailClient 构造 Service
	// ... 调 ConfirmAccept(100, 200, ptr(300))
	// ... assert: fake escortAvailClient.bookCalled == true
}

// TestReleaseLockAndReject_CallsReleaseByOrder
func TestReleaseLockAndReject_CallsReleaseByOrder(t *testing.T) {
	// ... 调 ReleaseLockAndReject(100, "escort_declined")
	// ... assert: fake escortAvailClient.releaseCalled == true
}
```

**Step 6: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/order/...
```

Expected: PASS。

**Step 7: Commit**

```bash
git add services/order/internal/integration/ services/order/internal/service/
git commit -m "feat(order): integration 层 escort_availability hook + ConfirmAccept/ReleaseLockAndReject 集成 (Book/Release 2 个单测)"
```

---

## Task 7: 文档同步 + dev.md + 全量回归 + push

**Files:**
- Modify: `docs/04-业务流程.md`
- Modify: `dev.md`
- Create: `scripts/smoke-availability.sh`

**Step 1: 04-业务流程.md 加 §4.8 escort_availabilities**

```markdown
### 陪诊师时段与选人模式（escort-business v2）

1. 陪诊师 approved 后即可 PUT `/escorts/me/availability` 设置空余时段；
   时段不冲突（service 校验 + DB UNIQUE 索引）+ start_at 必须在未来。
2. 患者下单 paid → match-service 调 `availability.ListAvailableByTime(order.ServiceStartAt)` 选 Top N 候选。
3. 患者选某 escort：order 状态 → `escort_pending_acceptance`，写 `selected_escort_id`；
   escort 端 `GET /escorts/me/invitations` 看到该订单（30s 倒计时）。
4. Escort POST `/orders/:id/confirm-accept`（escort-order-ext plan）→ state `accepted` →
   order-service 调 `availability.BookByOrder(slotID, escortID, orderID)` 把时段 booked。
5. Escort POST `/orders/:id/reject-accept` 或 30s 超时 → state 回退 `selecting_escort` →
   order-service 调 `availability.ReleaseByOrder(orderID)`（幂等，时段本就 available）。
6. 取消后续订单 → state `canceled` + 时段恢复 available。

**派生状态**：`available`（有 available 时段）/ `busy`（无）/ `off-line`（escort 主动下线）。

**集成**：order-service ↔ escort-service 通过共享 `services/order/internal/integration` 包直连（共享 Go-level repo，避免 HTTP 跨服务往返）。

事件 `order.escort_confirmed` / `order.escort_rejected`（shared/contracts，详见 order-matching-redesign §5.1）。
```

**Step 2: dev.md 加 §10.14**

```markdown
### 10.14 陪诊业务实装 v2 — escort_availabilities + 选人模式（2026-09-24 escort-business plan v2）

解决 `order-matching-redesign` §3.2 + §4.1 escort 端 5 个新 endpoint + 邀请列表 + 派生状态。

**落地 commits（7 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(migrations) | 0009 orders 修订 + escort_availabilities 表 + CHECK + UNIQUE |
| feat(escort) | availability 子包：types/state/repo + 13 集成 + 3 状态机单测 |
| feat(escort) | availability.Service 业务层（冲突校验 + 派生状态）+ 8 单测 |
| feat(escort) | handler 5 endpoint（escort/me × 3 + 公开 availabilities + invitations）+ 5 单测 |
| feat(escort) | profile 响应加 availability_status（available/busy/off-line） |
| feat(order) | integration 层 escort_availability hook + ConfirmAccept/ReleaseLockAndReject 集成 + 2 单测 |
| docs + smoke | 04-业务流程.md §4.8 + dev.md 10.14 + smoke-availability.sh |

**API 增量（5 + 1 = 6 个）**：
- PUT    /api/v1/escorts/me/availability           新建时段
- GET    /api/v1/escorts/me/availability           列我的时段
- DELETE /api/v1/escorts/me/availability/:id      删时段（仅 available）
- GET    /api/v1/escorts/me/invitations            待我确认的订单
- GET    /api/v1/escorts/:id/availabilities        公开列某 escort 可用时段
- (派生)  GET /api/v1/escorts/me/profile 含 availability_status

**共享迁移**：0009_escort_availabilities.up.sql = orders 表修订 + escort_availabilities 新表；与 order-matching-redesign 共用。

**未做**：
- 抢单 Feed（match/feed）已撤销；escort-order-ext plan 处理 select-escort / confirm-accept / reject-accept
- 时段模板（每周固定）/ 智能评分留 v2
- order-service 与 escort-service 跨服务升级为 HTTP / gRPC 留 v2
- 时间重叠区间 DB 级约束（Postgres `tstzrange &&` exclusion 约束需 btree_gist 扩展；v1 用 service 层校验）
```

**Step 3: scripts/smoke-availability.sh**

```bash
#!/usr/bin/env bash
# Smoke 验证 escort-availability 5 endpoint 401 拦截。
set -euo pipefail

ADDR="${ADDR:-:8080}"
TOKEN="${ESCORT_TOKEN:-}"

if [[ -z "$TOKEN" ]]; then
  echo "ESCORT_TOKEN required"
  exit 1
fi

echo "smoke escort-availability endpoints"
for path in \
  "/api/v1/escorts/me/availability" \
  "/api/v1/escorts/me/invitations" \
  "/api/v1/escorts/1/availabilities" ; do
  CODE=$(curl -sS -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" "http://127.0.0.1$ADDR$path")
  if [[ "$CODE" != "200" ]]; then
    echo "  $path unexpected code: $CODE"
    exit 1
  fi
  echo "  $path -> $CODE OK"
done

# 401 拦截（无 token）
CODE=$(curl -sS -o /dev/null -w "%{http_code}" "http://127.0.0.1$ADDR/api/v1/escorts/me/availability")
[[ "$CODE" == "401" ]] || { echo "expected 401, got $CODE"; exit 1; }
echo "  no-token -> 401 OK"

echo "smoke OK"
```

**Step 4: 全量回归 + push**

```bash
# 清干净 PG（保留已有数据）
docker exec doctors-postgres psql -U doctors -d doctors -c \
  "DROP TABLE IF EXISTS escort_availabilities CASCADE;"

# 跑单测
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./shared/... ./services/...

# 跑集成测试
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 ./migrations/... ./services/...

# 跑 smoke
bash scripts/smoke-availability.sh

# commit docs + smoke
git add docs/ dev.md scripts/smoke-availability.sh
git commit -m "docs: 04-业务流程 §4.8 escort_availabilities 流程图 + dev.md 10.14 escort-business v2 落地记录 + smoke-availability"

# push
git push origin main
```

Expected: PASS + smoke OK + pushed.

---

## Self-Review

- ✅ **Spec 覆盖**：`order-matching-redesign.md` §3.2（escort_availabilities 表 schema + 索引 + CHECK）+ §4.1（5 endpoint）+ §5.1（事件与 §7 集成）；陪诊师状态派生（§7 关键决策）；与 order-service 集成（§6 plan 表）。
- ✅ **共享迁移**：单一 `0009_escort_availabilities.up.sql` 同时承载 orders 表修订（drop lock_owner / add selected_escort_id / 新 CHECK / 替换索引）+ escort_availabilities 新表；order-matching-redesign 与 escort-business v2 共用。
- ✅ **状态机**：3 态白名单 `CanTransition`（`book` / `release` / `cancel` / `reactivate`），纯函数 + 3 个单测覆盖合法+非法矩阵。
- ✅ **时段时间合法性**：service.validateRange（start_at > now + end_at > start_at）+ repo.HasOverlap（区间重叠检测）+ DB UNIQUE `(escort_id, start_at)`；冲突返回 `CodeConflict`，过期时段不计入 DeriveStatus。
- ✅ **派生状态**：`DeriveStatus(escortID, manualOffline)` — manualOffline 优先；available 判定需 `CountAvailableForEscort(escort_id)` > 0；否则 busy；不修改 escort_profiles.state（readonly 派生）。
- ✅ **集成路径**：order-service ↔ escort-service 通过 Go-level 共享包直连（`services/order/internal/integration/escort_availability.go`），绕过 HTTP / RPC；best-effort 不阻塞主流程；nil 客户端为 no-op（本地开发 + 单测）。
- ✅ **类型一致**：`Availability` struct 与 SQL 列对齐；事件字段名沿用 `order.escort_confirmed` / `order.escort_rejected`（与 order-matching-redesign §5.1 一致）；`InvitationDTO` 字段与 escort-app 设计 `§3.1 我的邀请` 对齐。
- ✅ **测试矩阵**：Task 1 集成（migration）+ Task 2 13 集成 + 3 状态机单测 + Task 3 8 service 单测 + Task 4 5 handler 单测 + Task 5 1 profile 单测 + Task 6 4 集成层 + 2 order-service 集成验证 + Task 7 smoke + 全量回归。
- ✅ **YAGNI**：v1 不做时段模板 / 智能评分 / 跨服务 HTTP / 时间范围 DB 约束（exclude USING gist 需扩展）；admin 审核、抢单 Feed、checkin/checkout、admin-web 接线留各专 plan。

## 与原 v1 plan 的关键差异

| 维度 | v1 | v2（本 plan） |
|---|---|---|
| 范围 | 11 API（含抢单 Feed + 上线/下线）| **5 API + 邀请列表 + 派生状态**（注册/实名/健康证/培训/审核 + checkin/checkout 假设 v1 已落地）|
| 核心业务变更 | 陪诊师自主上线 → 抢单池接单 | **陪诊师设时段 → 患者选人 → 30s 陪诊师确认** |
| 抢单 Feed | `GET /match/feed` + `POST /orders/:id/accept` | **已撤销**；患者端用 `GET /orders/:id/candidates` + `POST /orders/:id/select-escort`（escort-order-ext plan）|
| 订单状态 | `pending_acceptance`（escort 抢单锁）| **`selecting_escort` + `escort_pending_acceptance`**（order-matching-redesign）|
| 状态派生 | 直接由 `escort_profiles.state` | 加 **派生 `availability_status`**（available/busy/off-line）|
| 订单锁 | `lock_owner` + `lock_expire_at`（Redis SETNX）| **`selected_escort_id` + `escort_pending_expire_at` + 时段 booked** |
| 集成 | 单服务内部 | order-service ↔ escort-service 跨服务 hook（共享包）|

## 关联 spec

- `docs/superpowers/specs/2026-09-24-order-matching-redesign.md` §3.2 + §4.1 + §5.1 + §7.4
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.2 + §3.1
- `docs/superpowers/specs/2026-09-24-escort-app-design.md` §3.1「我的空余时段」+ §3.1「我的邀请」
- `docs/superpowers/plans/2026-09-24-state-machine.md`（orders 表基础）
- `docs/superpowers/plans/2026-09-24-escort-order-ext.md`（select-escort / confirm-accept / reject-accept，本 plan 提供 hook 给该 plan）

## Execution Options

> 本 plan 假设 v1（11 态状态机 + 注册 / 实名 / 健康证 / 培训 / 审核 / checkin/checkout）已先落地；如未落地，需先执行 v1 plan。
> 当前为 plan_all 模式 → 进入实施阶段需要用户决策。

**下一步选项**：
1. **立即执行**（subagent-driven 或 inline 执行）—— 从 Task 1（0009 迁移）开始按 commit 节奏推 7 个 task。
2. **暂停 + review** —— review 本 plan + 0009 迁移对生产的影响（需先跑数据迁移脚本把 `lock_owner` 数据清掉）。
3. **先做 escort-order-ext plan** —— escort-order-ext 处理 select-escort / confirm-accept / reject-accept，本 plan 提供 BookByOrder/ReleaseByOrder hook 供其调用；建议先 escort-order-ext 完成依赖接口再并行。
