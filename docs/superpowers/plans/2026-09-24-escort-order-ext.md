# Order Service 陪诊视角扩展（escort list + checkin/checkout）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.2 escort 端 P0 缺口：`GET /api/v1/orders?role=escort&status=matching`（抢单池 Feed）+ `POST /api/v1/orders/:id/checkin`（到院签到）+ `POST /api/v1/orders/:id/checkout`（服务完成打卡）。复用既有 `state` 状态机 + `repo` 仓储 + `service` 业务编排；不抢 escort-business plan 的 escort 业务字段。

**Architecture:** 扩展既有 `services/order`（最小侵入）：迁移 `0006_orders_checkin.up.sql` 加 `checkin_at` / `checkout_at` 列；`repo.OrderRepo` 加 `ListMatching` / `ListByEscort` / `UpdateCheckin` / `UpdateCheckout`；`service.Service` 把 `List(uid, limit, offset)` 重构为 `List(uid, ListParams)`（patient 向后兼容） + 加 `Checkin` / `Checkout` 方法；`handler` 加 `?role=&status=` query 参数透传 + `POST /:id/checkin` + `POST /:id/checkout` 路由；状态机增加 `accepted → in_service` 与 `in_service → completed` 已有动作（无需修改 `transitions`，因为 `accepted → in_service` 和 `in_service → completed` 已存在于既有状态机 `state/machine.go`）。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · gin v1.10。

**前置依赖:**
- `2026-09-24-state-machine.md`（transitions 已含 `accepted → in_service` 与 `in_service → completed`）
- `2026-09-24-order-lock.md`（lock 字段已用 `lock_owner` / `lock_expire_at`，本 plan 不动）
- `2026-09-24-l2-api-gap-design.md` §2.2 P0 escort 端 API 清单

**不抢（边界）:**
- escort 实名 / 健康证 / 培训 / 上线 等业务字段（escort-business plan 处理）
- GPS 距离校验（escort-business plan 处理；本 plan v1 只验 status）
- 状态机本身（已有 `accepted → in_service` 与 `in_service → completed`，不修改 `transitions`）
- review 评价字段（review plan 处理）

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1 + gin v1.10
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- v1 checkin 只校验 `status='accepted'`，不引 GPS；escort 业务层的 GPS 校验在 escort-business plan 实现
- v1 checkout 只校验 `status='in_service'`，且 `checkin_at IS NOT NULL`
- `ListParams.Role='escort'&Status='matching'` → 抢单池（全局 matching 订单，不限定 escort_id）；其它 escort status → 仅本 escort 的订单
- `ListParams.Role='escort'&Status=''` → 仅本 escort 的全部订单
- 向后兼容：patient 不传 `role` 默认走 `ListByPatient`（与既有行为一致）
- 列新增：`checkin_at TIMESTAMPTZ` / `checkout_at TIMESTAMPTZ`（orders 表，迁移 `0006`）

---

## File Structure

| 路径 | 变更 | 职责 |
|------|------|------|
| `migrations/0006_orders_checkin.up.sql` | Create | orders 表加 `checkin_at` / `checkout_at` 列 + 索引 |
| `migrations/0006_orders_checkin.down.sql` | Create | 逆向 |
| `migrations/migrations_test.go` | Modify | 加 `Test0006OrdersCheckinUpDown` |
| `services/order/internal/repo/order_repo.go` | Modify | `Order` 加字段；`baseSelect` 加列；加 `ListMatching` / `ListByEscort` / `UpdateCheckin` / `UpdateCheckout` |
| `services/order/internal/repo/order_repo_integration_test.go` | Modify | 加 4 个集成测试（ListMatching / ListByEscort / UpdateCheckin / UpdateCheckout） |
| `services/order/internal/service/order_service.go` | Modify | `List` 改用 `ListParams`；加 `Checkin` / `Checkout`；`OrderRepo` 接口加 4 个方法 |
| `services/order/internal/service/order_service_test.go` | Modify | 既有测试适配新签名；加 Checkin / Checkout 单测 |
| `services/order/internal/handler/order.go` | Modify | `List` 解析 `role` / `status` query；加 `Checkin` / `Checkout` handler；`RegisterRoutes` 加 2 路由 |
| `services/order/internal/handler/order_test.go` | Modify | `fakeRepo` 加 4 方法；加 List 参数透传测试 + Checkin / Checkout handler 测试 |
| `docs/04-业务流程.md` | Modify | §5 陪诊端流程加 checkin / checkout |
| `dev.md` | Modify | §10.13 加 escort-order-ext plan 落地记录 |

---

### Task 1: 数据库迁移（orders 加 checkin_at / checkout_at）

**Files:**
- Create: `migrations/0006_orders_checkin.up.sql`
- Create: `migrations/0006_orders_checkin.down.sql`
- Modify: `migrations/migrations_test.go`

**Step 1: 写集成测试**

在 `migrations_test.go` 的 `Test0004RefundsUpDown` 之后追加：

```go
// Test0006OrdersCheckinUpDown 验证 0006_orders_checkin 加 checkin_at / checkout_at + 索引。
// 依赖 0001_users + 0002_orders + 0003_orders_state；测试结束回滚所有变更。
func Test0006OrdersCheckinUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 先建 users + orders + state（0006 依赖 lock_owner 等列）
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
		for _, f := range []string{"0003_orders_state.down.sql", "0002_orders.down.sql", "0001_users.down.sql"} {
			sql, _ := os.ReadFile(f)
			_, _ = cleanConn.Exec(cleanCtx, string(sql))
		}
	})

	applyUp(t, "0006_orders_checkin.up.sql", []string{"orders"})

	// 列检查
	for _, col := range []string{"checkin_at", "checkout_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='orders' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "orders.%s should exist", col)
	}

	// 索引检查（checkin_at 用于"陪诊师今日已签到订单"查询）
	var idxExists bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, "idx_orders_escort_checkin").
		Scan(&idxExists)
	require.NoError(t, err)
	assert.True(t, idxExists, "idx_orders_escort_checkin should exist")

	// down 校验
	downCtx, downCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer downCancel()
	downSQL, err := os.ReadFile("0006_orders_checkin.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(downCtx, string(downSQL))
	require.NoError(t, err, "apply 0006_orders_checkin.down.sql")

	for _, col := range []string{"checkin_at", "checkout_at"} {
		var gone bool
		err := conn.QueryRow(downCtx,
			`SELECT NOT EXISTS(SELECT 1 FROM information_schema.columns
			                   WHERE table_name='orders' AND column_name=$1)`, col).
			Scan(&gone)
		require.NoError(t, err)
		assert.True(t, gone, "orders.%s should be gone after down", col)
	}
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run Test0006OrdersCheckinUpDown ./migrations/`
Expected: FAIL — `Test0006OrdersCheckinUpDown` undefined

**Step 3: 写 `0006_orders_checkin.up.sql`**

```sql
-- 0006_orders_checkin.up.sql
-- escort-order-ext plan: 陪诊端 checkin / checkout 时间戳。
-- checkin_at: 陪诊师到院签到（status=accepted → in_service）时写入。
-- checkout_at: 服务完成打卡（status=in_service → completed）时写入。
-- v1 不做 GPS 校验（escort-business plan 处理）；列只记录时间戳。

ALTER TABLE orders
  ADD COLUMN checkin_at TIMESTAMPTZ,
  ADD COLUMN checkout_at TIMESTAMPTZ;

-- 索引服务于"陪诊师今日已签到订单"等查询（escort-app 个人中心）。
-- 复合 (escort_id, checkin_at DESC) 比单独 (escort_id) 更高效。
CREATE INDEX idx_orders_escort_checkin ON orders(escort_id, checkin_at DESC)
  WHERE escort_id IS NOT NULL;
```

**Step 4: 写 `0006_orders_checkin.down.sql`**

```sql
-- 0006_orders_checkin.down.sql
-- 撤销 0006：删索引 + 删列。

DROP INDEX IF EXISTS idx_orders_escort_checkin;
ALTER TABLE orders DROP COLUMN IF EXISTS checkout_at;
ALTER TABLE orders DROP COLUMN IF EXISTS checkin_at;
```

**Step 5: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run Test0006OrdersCheckinUpDown ./migrations/`
Expected: PASS

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0006 orders checkin_at/checkout_at + idx_orders_escort_checkin"
```

---

### Task 2: repo 扩展 ListMatching / ListByEscort / UpdateCheckin / UpdateCheckout

**Files:**
- Modify: `services/order/internal/repo/order_repo.go`
- Modify: `services/order/internal/repo/order_repo_integration_test.go`

**Step 1: 写集成测试（RED）**

在 `services/order/internal/repo/order_repo_integration_test.go` 的 `setupPool` 函数里加 `checkin_at` / `checkout_at` 列定义（确保 schema 与迁移对齐）：

```go
// setupPool 中的 orders CREATE TABLE 块追加：
  checkin_at TIMESTAMPTZ,
  checkout_at TIMESTAMPTZ,
```

在文件末尾追加 4 个集成测试：

```go
// TestOrderRepo_ListMatching_GlobalFeed 验证 ListMatching 返回全部 status='matching' 的订单。
func TestOrderRepo_ListMatching_GlobalFeed(t *testing.T) {
	pool := setupPool(t)
	patient1 := seedUser(t, pool, "13800200000", "patient")
	patient2 := seedUser(t, pool, "13800200001", "patient")
	r := NewOrderRepo(pool)

	// patient1 一个 matching + 一个 paid（不参与）
	o1 := sampleOrder(patient1)
	o1.OrderNo = "O-ext-001"
	require.NoError(t, r.Create(context.Background(), o1))
	require.NoError(t, r.UpdateStatus(context.Background(), o1.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o1.ID, "matching", 1, nil))

	// patient2 一个 matching
	o2 := sampleOrder(patient2)
	o2.OrderNo = "O-ext-002"
	require.NoError(t, r.Create(context.Background(), o2))
	require.NoError(t, r.UpdateStatus(context.Background(), o2.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o2.ID, "matching", 1, nil))

	list, err := r.ListMatching(context.Background(), 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 2, "两个 matching 订单应在抢单池")
}

// TestOrderRepo_ListMatching_ExcludesOthers 验证 ListMatching 不返回非 matching 订单。
func TestOrderRepo_ListMatching_ExcludesOthers(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800200010", "patient")
	escort := seedUser(t, pool, "13800200011", "escort")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	o.OrderNo = "O-ext-003"
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "accepted", 2, &escort))

	list, err := r.ListMatching(context.Background(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, list, "accepted 订单不应在抢单池")
}

// TestOrderRepo_ListByEscort 验证 ListByEscort 按 escort_id + status 过滤。
func TestOrderRepo_ListByEscort(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800200020", "patient")
	escort1 := seedUser(t, pool, "13800200021", "escort")
	escort2 := seedUser(t, pool, "13800200022", "escort")
	r := NewOrderRepo(pool)

	// escort1 接 2 单（1 个 accepted + 1 个 in_service）
	o1 := sampleOrder(patient)
	o1.OrderNo = "O-ext-004"
	require.NoError(t, r.Create(context.Background(), o1))
	require.NoError(t, r.UpdateStatus(context.Background(), o1.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o1.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o1.ID, "accepted", 2, &escort1))

	o2 := sampleOrder(patient)
	o2.OrderNo = "O-ext-005"
	require.NoError(t, r.Create(context.Background(), o2))
	require.NoError(t, r.UpdateStatus(context.Background(), o2.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o2.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o2.ID, "accepted", 2, &escort1))
	require.NoError(t, r.UpdateStatus(context.Background(), o2.ID, "in_service", 3, nil))

	// escort2 接 1 单（accepted）—— 不应出现在 escort1 列表
	o3 := sampleOrder(patient)
	o3.OrderNo = "O-ext-006"
	require.NoError(t, r.Create(context.Background(), o3))
	require.NoError(t, r.UpdateStatus(context.Background(), o3.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o3.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o3.ID, "accepted", 2, &escort2))

	// 只查 accepted 状态
	list, err := r.ListByEscort(context.Background(), escort1, "accepted", 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, o1.ID, list[0].ID)

	// 只查 in_service 状态
	list, err = r.ListByEscort(context.Background(), escort1, "in_service", 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, o2.ID, list[0].ID)

	// 不传 status：返回 escort1 全部订单（2 条）
	list, err = r.ListByEscort(context.Background(), escort1, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// TestOrderRepo_UpdateCheckin_OK 验证 accepted → in_service + 写 checkin_at。
func TestOrderRepo_UpdateCheckin_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800200030", "patient")
	escort := seedUser(t, pool, "13800200031", "escort")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	o.OrderNo = "O-ext-007"
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "accepted", 2, &escort))

	require.NoError(t, r.UpdateCheckin(context.Background(), o.ID, 2))

	got, _ := r.FindByID(context.Background(), o.ID)
	assert.Equal(t, "in_service", got.Status)
	require.NotNil(t, got.CheckinAt)
	assert.WithinDuration(t, time.Now(), *got.CheckinAt, 5*time.Second)
}

// TestOrderRepo_UpdateCheckin_WrongStatus 验证非 accepted 状态签到返回 ErrInvalidStateForCheckin。
func TestOrderRepo_UpdateCheckin_WrongStatus(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800200032", "patient")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient) // status='created'
	o.OrderNo = "O-ext-008"
	require.NoError(t, r.Create(context.Background(), o))

	err := r.UpdateCheckin(context.Background(), o.ID, 0)
	assert.ErrorIs(t, err, ErrInvalidStateForCheckin)
}

// TestOrderRepo_UpdateCheckout_OK 验证 in_service → completed + 写 checkout_at。
func TestOrderRepo_UpdateCheckout_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800200040", "patient")
	escort := seedUser(t, pool, "13800200041", "escort")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	o.OrderNo = "O-ext-009"
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "accepted", 2, &escort))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "in_service", 3, nil))

	require.NoError(t, r.UpdateCheckout(context.Background(), o.ID, 4))

	got, _ := r.FindByID(context.Background(), o.ID)
	assert.Equal(t, "completed", got.Status)
	require.NotNil(t, got.CheckoutAt)
	assert.WithinDuration(t, time.Now(), *got.CheckoutAt, 5*time.Second)
}

// TestOrderRepo_UpdateCheckout_NotInService 验证非 in_service 状态打卡返回 ErrInvalidStateForCheckout。
func TestOrderRepo_UpdateCheckout_NotInService(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800200042", "patient")
	escort := seedUser(t, pool, "13800200043", "escort")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	o.OrderNo = "O-ext-010"
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "paid", 0, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 1, nil))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "accepted", 2, &escort))

	err := r.UpdateCheckout(context.Background(), o.ID, 3)
	assert.ErrorIs(t, err, ErrInvalidStateForCheckout)
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run 'TestOrderRepo_(ListMatching|ListByEscort|UpdateCheckin|UpdateCheckout)' ./services/order/internal/repo/`
Expected: FAIL — `undefined: ListMatching`, `undefined: ListByEscort`, `undefined: UpdateCheckin`, `undefined: UpdateCheckout`, `undefined: ErrInvalidStateForCheckin`, `undefined: ErrInvalidStateForCheckout`

**Step 3: 扩展 order_repo.go**

`services/order/internal/repo/order_repo.go`：

```go
// Order 映射 orders 表行。CheckinAt / CheckoutAt 由 0006 迁移添加。
type Order struct {
	ID             int64
	OrderNo        string
	PatientID      int64
	EscortID       *int64
	HospitalID     int64
	PackageID      int64
	ServiceStartAt any // 借 `time.Time`；这里用 any 是为避免顶层 import 复杂化
	Amount         float64
	FinalAmount    float64
	Status         string
	Version        int
	LockOwner      *int64     // pending_acceptance 锁单持有者；nil 表示未锁
	LockExpireAt   *time.Time // 锁单超时时间；nil 表示未锁
	CheckinAt      *time.Time // 陪诊端 checkin 时间戳；nil = 未签到
	CheckoutAt     *time.Time // 陪诊端 checkout 时间戳；nil = 未打卡
}
```

哨兵（按既有 `ErrInvalidStateForLock` 紧邻追加）：

```go
// ErrInvalidStateForCheckin 是 checkin 时订单不在 accepted 的哨兵。
var ErrInvalidStateForCheckin = errors.New("repo: order not in accepted state for checkin")

// ErrInvalidStateForCheckout 是 checkout 时订单不在 in_service 的哨兵。
var ErrInvalidStateForCheckout = errors.New("repo: order not in in_service state for checkout")
```

`baseSelect` 加列：

```go
// baseSelect 是 SELECT 子句。
const baseSelect = `
	SELECT id, order_no, patient_id, escort_id, hospital_id, package_id,
	       service_start_at, amount, final_amount, status, version,
	       lock_owner, lock_expire_at, checkin_at, checkout_at
	FROM orders`
```

`scanOne` / `scanRow` 同步加列：

```go
// scanOne 把单行扫描为 *Order。
func (r *OrderRepo) scanOne(row pgx.Row) (*Order, error) {
	o := &Order{}
	if err := row.Scan(
		&o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
		&o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
		&o.LockOwner, &o.LockExpireAt, &o.CheckinAt, &o.CheckoutAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("scan order: %w", err)
	}
	return o, nil
}

// scanRow 把 pgx.Rows 的一行扫描为 *Order。
func (r *OrderRepo) scanRow(rows pgx.Rows) (*Order, error) {
	o := &Order{}
	if err := rows.Scan(
		&o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
		&o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
		&o.LockOwner, &o.LockExpireAt, &o.CheckinAt, &o.CheckoutAt,
	); err != nil {
		return nil, err
	}
	return o, nil
}
```

4 个新方法（追加到 `LockExpired` 之后）：

```go
// ListMatching 返回 status='matching' 的订单（抢单池 Feed；不限定 escort_id）。
// 按 service_start_at ASC 排序：最先开始的单排前面（陪诊师优先接急单）。
func (r *OrderRepo) ListMatching(ctx context.Context, limit, offset int) ([]*Order, error) {
	const q = baseSelect + ` WHERE status = 'matching' AND deleted_at IS NULL
	                         ORDER BY service_start_at ASC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list matching: %w", err)
	}
	defer rows.Close()
	out := make([]*Order, 0)
	for rows.Next() {
		o, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListByEscort 按 escort_id + 可选 status 过滤；status 为空时返回 escort 全部订单。
// 按 service_start_at DESC 排序：最近的服务排前面。
func (r *OrderRepo) ListByEscort(ctx context.Context, escortID int64, status string, limit, offset int) ([]*Order, error) {
	q := baseSelect + ` WHERE escort_id = $1 AND deleted_at IS NULL`
	args := []any{escortID}
	if status != "" {
		q += ` AND status = $2`
		args = append(args, status)
		q += ` ORDER BY service_start_at DESC LIMIT $3 OFFSET $4`
		args = append(args, limit, offset)
	} else {
		q += ` ORDER BY service_start_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list by escort: %w", err)
	}
	defer rows.Close()
	out := make([]*Order, 0)
	for rows.Next() {
		o, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateCheckin 把 accepted 状态订单签到 → in_service + 写 checkin_at。
// v1 不引 GPS 校验（escort-business plan 处理）；只校验 status=accepted。
func (r *OrderRepo) UpdateCheckin(ctx context.Context, id int64, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'in_service', checkin_at = NOW(),
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND version = $2 AND status = 'accepted' AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, expectVersion)
	if err != nil {
		return fmt.Errorf("update checkin: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 区分版本冲突与状态非法
		var curStatus string
		var curVer int
		_ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
		if curVer != expectVersion {
			return ErrVersionConflict
		}
		return ErrInvalidStateForCheckin
	}
	return nil
}

// UpdateCheckout 把 in_service 状态订单打卡 → completed + 写 checkout_at。
// 校验 status=in_service 且 checkin_at IS NOT NULL（先签到才能打卡）。
func (r *OrderRepo) UpdateCheckout(ctx context.Context, id int64, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'completed', checkout_at = NOW(),
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND version = $2 AND status = 'in_service'
		       AND checkin_at IS NOT NULL AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, expectVersion)
	if err != nil {
		return fmt.Errorf("update checkout: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var curStatus string
		var curVer int
		_ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
		if curVer != expectVersion {
			return ErrVersionConflict
		}
		return ErrInvalidStateForCheckout
	}
	return nil
}
```

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run 'TestOrderRepo_(ListMatching|ListByEscort|UpdateCheckin|UpdateCheckout)' ./services/order/internal/repo/
```
Expected: PASS（7 个测试全过）

**Step 5: 跑全量集成测试，确认不破坏既有 6 个**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 ./services/order/internal/repo/`
Expected: PASS（既有 6 + 新增 7 = 13 个）

**Step 6: Commit**

```bash
git add services/order/internal/repo/
git commit -m "feat(order): repo 加 ListMatching / ListByEscort / UpdateCheckin / UpdateCheckout (7 集成测试)"
```

---

### Task 3: service 扩展 ListParams + Checkin / Checkout

**Files:**
- Modify: `services/order/internal/service/order_service.go`
- Modify: `services/order/internal/service/order_service_test.go`

**Step 1: 扩展 service 测试（先 fake repo 加方法 + 写新单测）**

在 `services/order/internal/service/order_service_test.go` 的 `fakeRepo`（既有 fake 实现 OrderRepo 接口）追加 4 个方法（参考 `LockForAccept` 等的实现风格，按内存 map 模拟）：

```go
// 抢单池 Feed（不限定 escort_id）：列出所有 status='matching' 的订单。
func (r *fakeRepo) ListMatching(ctx context.Context, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.Status == "matching" {
			out = append(out, o)
		}
	}
	return out, nil
}

// 陪诊师视角：按 escort_id + 可选 status 过滤。
func (r *fakeRepo) ListByEscort(ctx context.Context, escortID int64, status string, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.EscortID == nil || *o.EscortID != escortID {
			continue
		}
		if status != "" && o.Status != status {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

// 签到：accepted → in_service + 写 checkin_at。
func (r *fakeRepo) UpdateCheckin(ctx context.Context, id int64, expectVersion int) error {
	o, ok := r.orders[id]
	if !ok {
		return repo.ErrOrderNotFound
	}
	if o.Version != expectVersion {
		return repo.ErrVersionConflict
	}
	if o.Status != "accepted" {
		return repo.ErrInvalidStateForCheckin
	}
	now := time.Now()
	o.Status = "in_service"
	o.CheckinAt = &now
	o.Version++
	return nil
}

// 打卡：in_service → completed + 写 checkout_at。
func (r *fakeRepo) UpdateCheckout(ctx context.Context, id int64, expectVersion int) error {
	o, ok := r.orders[id]
	if !ok {
		return repo.ErrOrderNotFound
	}
	if o.Version != expectVersion {
		return repo.ErrVersionConflict
	}
	if o.Status != "in_service" {
		return repo.ErrInvalidStateForCheckout
	}
	now := time.Now()
	o.Status = "completed"
	o.CheckoutAt = &now
	o.Version++
	return nil
}
```

> **说明**：既有 fakeRepo 的 OrderRepo 接口实现（Create / FindByID / ListByPatient / UpdateStatus / InsertEvent / ListEvents / LockForAccept / ReleaseLock / LockExpired）在既有文件中已存在；本 Task 只追加 4 个新方法。

接着在文件末尾追加 service 层的单测：

```go
// TestService_ListEscortMatching 验证 role=escort&status=matching 走 ListMatching。
func TestService_ListEscortMatching(t *testing.T) {
	fr := newFakeRepo()
	users := map[int64]*service.UserSnapshot{
		1: {ID: 1, Role: "escort", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	// 注入一个 matching 订单
	o := &repo.Order{OrderNo: "O-svc-001", PatientID: 99, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100, Status: "matching", Version: 0}
	require.NoError(t, fr.Create(context.Background(), o))

	list, err := svc.List(context.Background(), 1, service.ListParams{
		Role: "escort", Status: "matching", Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Len(t, list, 1, "matching 订单应在抢单池")
}

// TestService_ListEscortMyOrders 验证 role=escort 不传 status 走 ListByEscort。
func TestService_ListEscortMyOrders(t *testing.T) {
	fr := newFakeRepo()
	escort := int64(1)
	users := map[int64]*service.UserSnapshot{
		1: {ID: escort, Role: "escort", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	// 注入 escort 已接订单
	eid := escort
	o := &repo.Order{OrderNo: "O-svc-002", PatientID: 99, EscortID: &eid, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100, Status: "accepted", Version: 0}
	require.NoError(t, fr.Create(context.Background(), o))

	list, err := svc.List(context.Background(), escort, service.ListParams{
		Role: "escort", Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	assert.Len(t, list, 1, "escort 自己的订单应返回")
}

// TestService_ListPatientBackwards 验证不传 role 默认走 ListByPatient（向后兼容）。
func TestService_ListPatientBackwards(t *testing.T) {
	fr := newFakeRepo()
	patient := int64(1)
	users := map[int64]*service.UserSnapshot{
		1: {ID: patient, Role: "patient", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	o := &repo.Order{OrderNo: "O-svc-003", PatientID: patient, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100, Status: "created", Version: 0}
	require.NoError(t, fr.Create(context.Background(), o))

	list, err := svc.List(context.Background(), patient, service.ListParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

// TestService_Checkin_OK 验证 accepted 状态签到成功。
func TestService_Checkin_OK(t *testing.T) {
	fr := newFakeRepo()
	escort := int64(2)
	users := map[int64]*service.UserSnapshot{
		2: {ID: escort, Role: "escort", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	eid := escort
	o := &repo.Order{ID: 100, OrderNo: "O-svc-004", PatientID: 99, EscortID: &eid, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100, Status: "accepted", Version: 5}
	fr.orders[o.ID] = o
	fr.events = nil

	err := svc.Checkin(context.Background(), o.ID, escort)
	require.NoError(t, err)
	assert.Equal(t, "in_service", o.Status)
	require.NotNil(t, o.CheckinAt)
	require.NotEmpty(t, fr.events, "InsertEvent 应被调用")
	assert.Equal(t, "in_service", fr.events[0].ToStatus)
}

// TestService_Checkin_NotEscort 验证非 escort 签到返回 CodeForbidden。
func TestService_Checkin_NotEscort(t *testing.T) {
	fr := newFakeRepo()
	users := map[int64]*service.UserSnapshot{
		1: {ID: 1, Role: "patient", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	err := svc.Checkin(context.Background(), 100, 1)
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeForbidden, e.Code)
}

// TestService_Checkin_WrongState 验证非 accepted 状态签到返回 CodeConflict。
func TestService_Checkin_WrongState(t *testing.T) {
	fr := newFakeRepo()
	escort := int64(2)
	users := map[int64]*service.UserSnapshot{
		2: {ID: escort, Role: "escort", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	o := &repo.Order{ID: 101, OrderNo: "O-svc-005", PatientID: 99, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100, Status: "created", Version: 0}
	fr.orders[o.ID] = o

	err := svc.Checkin(context.Background(), o.ID, escort)
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeConflict, e.Code)
}

// TestService_Checkout_OK 验证 in_service + 已签到 → completed。
func TestService_Checkout_OK(t *testing.T) {
	fr := newFakeRepo()
	escort := int64(2)
	users := map[int64]*service.UserSnapshot{
		2: {ID: escort, Role: "escort", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	now := time.Now().Add(-time.Minute)
	eid := escort
	o := &repo.Order{ID: 200, OrderNo: "O-svc-006", PatientID: 99, EscortID: &eid, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100,
		Status: "in_service", Version: 7, CheckinAt: &now}
	fr.orders[o.ID] = o

	err := svc.Checkout(context.Background(), o.ID, escort)
	require.NoError(t, err)
	assert.Equal(t, "completed", o.Status)
	require.NotNil(t, o.CheckoutAt)
}

// TestService_Checkout_NotCheckedIn 验证未签到的订单打卡返回 CodeConflict。
func TestService_Checkout_NotCheckedIn(t *testing.T) {
	fr := newFakeRepo()
	escort := int64(2)
	users := map[int64]*service.UserSnapshot{
		2: {ID: escort, Role: "escort", RealNameVerified: true},
	}
	svc := service.New(fr, &fakeUserLookup{users: users})

	eid := escort
	o := &repo.Order{ID: 201, OrderNo: "O-svc-007", PatientID: 99, EscortID: &eid, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100,
		Status: "in_service", Version: 7, CheckinAt: nil}
	fr.orders[o.ID] = o

	err := svc.Checkout(context.Background(), o.ID, escort)
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeConflict, e.Code)
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestService_(ListEscortMatching|ListEscortMyOrders|ListPatientBackwards|Checkin|Checkout)' ./services/order/internal/service/`
Expected: FAIL — `undefined: service.ListParams`, `undefined: Service.Checkin`, `undefined: Service.Checkout`

**Step 3: 扩展 order_service.go**

`OrderRepo` 接口加 4 个方法（紧邻既有 `LockExpired` 之后）：

```go
// OrderRepo 是仓储最小契约。
type OrderRepo interface {
	Create(ctx context.Context, o *repo.Order) error
	FindByID(ctx context.Context, id int64) (*repo.Order, error)
	ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error)
	// escort 视角：抢单池（全局 matching）+ 陪诊师查看自己的订单
	ListMatching(ctx context.Context, limit, offset int) ([]*repo.Order, error)
	ListByEscort(ctx context.Context, escortID int64, status string, limit, offset int) ([]*repo.Order, error)
	UpdateStatus(ctx context.Context, id int64, to string, expectVersion int, escortID *int64) error
	// 签到 / 打卡
	UpdateCheckin(ctx context.Context, id int64, expectVersion int) error
	UpdateCheckout(ctx context.Context, id int64, expectVersion int) error
	InsertEvent(ctx context.Context, orderID int64, from *string, to string, actorID *int64, payload []byte) error
	ListEvents(ctx context.Context, orderID int64) ([]*repo.OrderEvent, error)
	// 状态机统一 plan: 锁单三件套
	LockForAccept(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error
	ReleaseLock(ctx context.Context, id int64, expectVersion int) error
	LockExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error)
}
```

新增 `ListParams` + 重构 `List`：

```go
// ListParams 是 List 的参数。
//   - Role: "patient"（默认）或 "escort"
//   - Status: 可选；空字符串表示全部状态
//   - 配合 role=escort&status=matching 即"抢单池 Feed"（不限定 escort_id）
type ListParams struct {
	Role   string
	Status string
	Limit  int
	Offset int
}

// List 返回订单分页。
//   - role=escort&status=matching → 抢单池 Feed（全局 matching 订单，不限定 escort_id）
//   - role=escort&其它 status     → 仅本 escort 的订单
//   - role=patient（默认）         → 患者自己订单（向后兼容既有行为）
func (s *Service) List(ctx context.Context, userID int64, p ListParams) ([]*repo.Order, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	switch p.Role {
	case "escort":
		if p.Status == "matching" {
			return s.orders.ListMatching(ctx, p.Limit, p.Offset)
		}
		return s.orders.ListByEscort(ctx, userID, p.Status, p.Limit, p.Offset)
	default: // patient
		return s.orders.ListByPatient(ctx, userID, p.Limit, p.Offset)
	}
}
```

新增 `Checkin` / `Checkout`（紧邻 `Finish` 之后）：

```go
// Checkin 陪诊师到院签到：accepted → in_service + 写 checkin_at。
// v1 不引 GPS 校验（escort-business plan 处理）；仅校验 status=accepted。
func (s *Service) Checkin(ctx context.Context, orderID, actorID int64) error {
	u, err := s.users.FindByID(ctx, actorID)
	if err != nil || u == nil {
		return errs.New(errs.CodeUnauthorized, "actor not found")
	}
	if u.Role != "escort" {
		return errs.New(errs.CodeForbidden, "only escort can checkin")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.EscortID == nil || *o.EscortID != actorID {
		return errs.New(errs.CodeForbidden, "not your order")
	}
	from := state.Status(o.Status)
	to := state.StatusInService
	if !state.CanTransition(from, to) {
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot checkin from %s", from))
	}
	if err := s.orders.UpdateCheckin(ctx, orderID, o.Version); err != nil {
		if err == repo.ErrInvalidStateForCheckin {
			return errs.New(errs.CodeConflict, "order not in accepted state")
		}
		if err == repo.ErrVersionConflict {
			return errs.New(errs.CodeConflict, "order version conflict")
		}
		return errs.Wrap(errs.CodeInternal, "update checkin", err)
	}
	fromStr := string(from)
	actor := actorID
	if err := s.orders.InsertEvent(ctx, orderID, &fromStr, string(to), &actor, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}
	return nil
}

// Checkout 陪诊师服务完成打卡：in_service → completed + 写 checkout_at。
// 校验 checkin_at IS NOT NULL（先签到才能打卡）。
func (s *Service) Checkout(ctx context.Context, orderID, actorID int64) error {
	u, err := s.users.FindByID(ctx, actorID)
	if err != nil || u == nil {
		return errs.New(errs.CodeUnauthorized, "actor not found")
	}
	if u.Role != "escort" {
		return errs.New(errs.CodeForbidden, "only escort can checkout")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.EscortID == nil || *o.EscortID != actorID {
		return errs.New(errs.CodeForbidden, "not your order")
	}
	from := state.Status(o.Status)
	to := state.StatusCompleted
	if !state.CanTransition(from, to) {
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot checkout from %s", from))
	}
	if err := s.orders.UpdateCheckout(ctx, orderID, o.Version); err != nil {
		if err == repo.ErrInvalidStateForCheckout {
			return errs.New(errs.CodeConflict, "order not in in_service state or not checked in")
		}
		if err == repo.ErrVersionConflict {
			return errs.New(errs.CodeConflict, "order version conflict")
		}
		return errs.Wrap(errs.CodeInternal, "update checkout", err)
	}
	fromStr := string(from)
	actor := actorID
	if err := s.orders.InsertEvent(ctx, orderID, &fromStr, string(to), &actor, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}
	return nil
}
```

> **既有 List 签名变更影响**：既有 `handler.order.go` 的 `h.svc.List(c.Request.Context(), uid, 20, 0)` 需要在 Task 4 中改为 `h.svc.List(c.Request.Context(), uid, service.ListParams{...})`。

**Step 4: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./services/order/internal/service/`
Expected: PASS（既有 + 8 个新单测）

**Step 5: Commit**

```bash
git add services/order/internal/service/
git commit -m "feat(order): service 加 ListParams + Checkin / Checkout (8 个单测 + fakeRepo 扩 4 方法)"
```

---

### Task 4: handler 扩展 List 参数 + Checkin / Checkout 路由

**Files:**
- Modify: `services/order/internal/handler/order.go`
- Modify: `services/order/internal/handler/order_test.go`

**Step 1: 写 handler 单测（RED）**

在 `services/order/internal/handler/order_test.go` 既有 fakeRepo 块追加 4 个新方法（按 service 层 fake 同样的代码）：

```go
// 抢单池 Feed（与 service 层 fake 一致）。
func (r *fakeRepo) ListMatching(ctx context.Context, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.Status == "matching" {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListByEscort(ctx context.Context, escortID int64, status string, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.EscortID == nil || *o.EscortID != escortID {
			continue
		}
		if status != "" && o.Status != status {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func (r *fakeRepo) UpdateCheckin(ctx context.Context, id int64, expectVersion int) error {
	o, ok := r.orders[id]
	if !ok {
		return repo.ErrOrderNotFound
	}
	if o.Version != expectVersion {
		return repo.ErrVersionConflict
	}
	if o.Status != "accepted" {
		return repo.ErrInvalidStateForCheckin
	}
	now := time.Now()
	o.Status = "in_service"
	o.CheckinAt = &now
	o.Version++
	return nil
}

func (r *fakeRepo) UpdateCheckout(ctx context.Context, id int64, expectVersion int) error {
	o, ok := r.orders[id]
	if !ok {
		return repo.ErrOrderNotFound
	}
	if o.Version != expectVersion {
		return repo.ErrVersionConflict
	}
	if o.Status != "in_service" {
		return repo.ErrInvalidStateForCheckout
	}
	now := time.Now()
	o.Status = "completed"
	o.CheckoutAt = &now
	o.Version++
	return nil
}
```

在文件末尾追加 handler 单测：

```go
// TestList_EscortMatching 验证 GET /api/v1/orders?role=escort&status=matching 路由到 ListMatching。
func TestList_EscortMatching(t *testing.T) {
	r, fr, _ := newTestServer()

	// 注入一个 matching 订单
	o := &repo.Order{
		OrderNo: "O-handler-001", PatientID: 99, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100,
		Status: "matching", Version: 0,
	}
	require.NoError(t, fr.Create(context.Background(), o))

	tok := signTestToken(t, 2, "escort")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders?role=escort&status=matching", tok, nil)
	assert.Equal(t, 0, resp.Code, resp.Message)
	require.NotNil(t, resp.Data["orders"])
	orders, _ := resp.Data["orders"].([]any)
	assert.Len(t, orders, 1)
}

// TestList_EscortMyOrders 验证 GET /api/v1/orders?role=escort 列出 escort 自己的订单。
func TestList_EscortMyOrders(t *testing.T) {
	r, fr, _ := newTestServer()

	escortID := int64(2)
	o := &repo.Order{
		OrderNo: "O-handler-002", PatientID: 99, EscortID: &escortID, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100,
		Status: "accepted", Version: 0,
	}
	require.NoError(t, fr.Create(context.Background(), o))

	tok := signTestToken(t, escortID, "escort")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders?role=escort", tok, nil)
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestCheckin_OK 验证 POST /api/v1/orders/:id/checkin 走 escort + accepted → in_service。
func TestCheckin_OK(t *testing.T) {
	r, fr, _ := newTestServer()
	escortID := int64(2)

	o := &repo.Order{ID: 300, OrderNo: "O-handler-003", PatientID: 99, EscortID: &escortID, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100,
		Status: "accepted", Version: 0}
	fr.orders[o.ID] = o

	tok := signTestToken(t, escortID, "escort")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/300/checkin", tok, nil)
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.Equal(t, "in_service", o.Status)
	require.NotNil(t, o.CheckinAt)
}

// TestCheckin_NotEscort 验证 patient 调 checkin 返回 CodeForbidden。
func TestCheckin_NotEscort(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/300/checkin", tok, nil)
	assert.NotEqual(t, 0, resp.Code)
}

// TestCheckout_OK 验证 POST /api/v1/orders/:id/checkout 走 escort + 已签到 → completed。
func TestCheckout_OK(t *testing.T) {
	r, fr, _ := newTestServer()
	escortID := int64(2)
	now := time.Now().Add(-time.Minute)

	o := &repo.Order{ID: 400, OrderNo: "O-handler-004", PatientID: 99, EscortID: &escortID, HospitalID: 1, PackageID: 1,
		ServiceStartAt: time.Now().Add(time.Hour), Amount: 100, FinalAmount: 100,
		Status: "in_service", Version: 5, CheckinAt: &now}
	fr.orders[o.ID] = o

	tok := signTestToken(t, escortID, "escort")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/400/checkout", tok, nil)
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.Equal(t, "completed", o.Status)
	require.NotNil(t, o.CheckoutAt)
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestList_EscortMatching|TestList_EscortMyOrders|TestCheckin_OK|TestCheckin_NotEscort|TestCheckout_OK' ./services/order/internal/handler/`
Expected: FAIL — `undefined: ListMatching`, `undefined: ListByEscort`, `undefined: UpdateCheckin`, `undefined: UpdateCheckout`（fakeRepo 不满足接口）

**Step 3: 扩展 handler/order.go**

`RegisterRoutes` 加 2 路由：

```go
// RegisterRoutes 把 order 路由挂到 RouterGroup。
// 中间件（如 Auth）已在 RouterGroup 上挂好。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	orders := r.Group("/orders")
	orders.POST("", h.Create)
	orders.GET("", h.List)
	orders.GET("/:id", h.Get)
	orders.POST("/:id/accept", h.Accept)
	orders.POST("/:id/cancel", h.Cancel)
	orders.POST("/:id/finish", h.Finish)
	// escort 端：签到 + 打卡（escort-order-ext plan）
	orders.POST("/:id/checkin", h.Checkin)
	orders.POST("/:id/checkout", h.Checkout)
}
```

`List` handler 改成读 `role` / `status` query：

```go
// List GET /api/v1/orders?role=escort&status=matching
//   - role=patient（默认）：返回患者自己订单
//   - role=escort&status=matching：抢单池 Feed
//   - role=escort&其它 status：escort 自己订单
func (h *Handler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	list, err := h.svc.List(c.Request.Context(), uid, service.ListParams{
		Role:   c.Query("role"),
		Status: c.Query("status"),
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"orders": list})
}
```

加 `Checkin` / `Checkout` handler（紧邻 `Finish` 之后）：

```go
// Checkin POST /api/v1/orders/{id}/checkin 陪诊师到院签到（accepted → in_service）。
func (h *Handler) Checkin(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can checkin"))
		return
	}
	if err := h.svc.Checkin(c.Request.Context(), id, uid); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": id, "status": "in_service"})
}

// Checkout POST /api/v1/orders/{id}/checkout 陪诊师服务完成打卡（in_service → completed）。
func (h *Handler) Checkout(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can checkout"))
		return
	}
	if err := h.svc.Checkout(c.Request.Context(), id, uid); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": id, "status": "completed"})
}
```

> **既有 List 调整**：原 `h.svc.List(c.Request.Context(), uid, 20, 0)` 改为新签名；既有 handler 单测（如 `TestList_OK` 之类）如有直接调 handler 的，需要同步加 query 参数；本 Task 主要新增 escort 端测试，既有 patient 单测不受影响（默认 role=patient）。

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./services/order/internal/handler/
```
Expected: PASS（既有 + 5 个新 handler 单测）

**Step 5: 全量回归 service + handler**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./services/order/...`
Expected: PASS（既有单测 + 新增 13 个全过）

**Step 6: Commit**

```bash
git add services/order/internal/handler/
git commit -m "feat(order): handler 加 List role/status 参数 + POST /:id/checkin + POST /:id/checkout (5 单测)"
```

---

### Task 5: 文档同步 + dev.md

**Files:**
- Modify: `docs/04-业务流程.md`
- Modify: `dev.md`

**Step 1: `docs/04-业务流程.md` §5 加陪诊端流程**

```markdown
### 5.x 陪诊端订单流程（escort-order-ext plan）

1. 陪诊师 App 启动 → 轮询 `GET /api/v1/orders?role=escort&status=matching`（每 5s）
2. 抢单：`POST /api/v1/orders/:id/accept` → 进入 pending_acceptance（30s 锁单）→ 确认或超时
3. accepted 后陪诊师前往医院 → `POST /api/v1/orders/:id/checkin` → status=in_service + checkin_at
4. 服务完成 → `POST /api/v1/orders/:id/checkout` → status=completed + checkout_at
5. 完成后由患者 review（review plan 实现的 `POST /api/v1/orders/:id/review` 触发评分）

> v1 不做 GPS 距离校验（escort-business plan 处理）；checkin / checkout 仅校验 status。
```

**Step 2: `dev.md` §10.13 加落地记录**

```markdown
### 10.13 escort-order-ext（2026-09-24 escort-order-ext plan）

解决 L2 v1.0 P0 缺口：escort 端抢单池 / 签到 / 打卡。

**落地 commits（4 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(migrations) | 0006 orders checkin_at / checkout_at + idx_orders_escort_checkin |
| feat(order) | repo 加 ListMatching / ListByEscort / UpdateCheckin / UpdateCheckout |
| feat(order) | service 加 ListParams + Checkin / Checkout |
| feat(order) | handler 加 ?role=&status= + POST /:id/checkin + POST /:id/checkout |

**API 增量**（3 个）：

- `GET /api/v1/orders?role=escort&status=matching` — 抢单池 Feed（不限定 escort_id）
- `GET /api/v1/orders?role=escort&status=accepted` — escort 自己的已接订单
- `POST /api/v1/orders/:id/checkin` — 陪诊师到院签到（accepted → in_service）
- `POST /api/v1/orders/:id/checkout` — 陪诊师服务完成打卡（in_service → completed）

> 既有 `GET /api/v1/orders` 不传 role 仍走患者视图（向后兼容）。

**未做（留给 escort-business / review plan）**：

- GPS 距离校验（escort-business plan 实现 `confirm-via-GPS` 校验层）
- 双向评价（review plan 实现 `POST /api/v1/orders/:id/review`）
- 抢单池 WebSocket 推送（v2）
```

**Step 3: Commit**

```bash
git add docs/ dev.md
git commit -m "docs: 04 §5 陪诊端流程 + dev.md 10.13 escort-order-ext 落地"
```

---

### Task 6: 全量回归 + push

```bash
# 清干净 PG（避免 0006 列约束影响既有测试）
docker exec doctors-postgres psql -U doctors -d doctors -c "DROP TABLE IF EXISTS refunds CASCADE; DROP TABLE IF EXISTS refund_policies CASCADE; DROP TABLE IF EXISTS order_events CASCADE; DROP TABLE IF EXISTS orders CASCADE; DROP TABLE IF EXISTS users CASCADE;" || true

# 跑全部单测
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./shared/... ./services/...

# 跑全部集成测试
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 ./migrations/... ./services/order/...

# 跑既有 smoke 脚本
bash scripts/smoke-order.sh

# push
git push origin main
```

Expected: 全部 PASS + smoke OK + pushed.

---

## Self-Review

- ✅ **Spec 覆盖**: l2-api-gap-design.md §2.2 P0 escort 端 3 个新 API（GET ?role=escort&status=matching / POST /:id/checkin / POST /:id/checkout） + 既有 GET 不破坏 patient 视图
- ✅ **无占位符**: 每个 Task 有完整代码与命令
- ✅ **类型一致**: `repo.Order` 在 Task 2 加 `CheckinAt` / `CheckoutAt`；`OrderRepo` 接口在 Task 3 加 4 方法；`service.ListParams` / `Checkin` / `Checkout` 一致；handler / service / fake 三层方法名一致
- ✅ **测试矩阵**: Task 1 集成 + Task 2 集成（7 个）+ Task 3 单元（8 个）+ Task 4 单元（5 个）+ Task 6 全量
- ✅ **边界**: 不抢 escort-business / review plan；v1 不引 GPS 校验；用既有状态机 `accepted → in_service` 与 `in_service → completed` 不改 transitions；向后兼容 patient 默认走 ListByPatient

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-escort-order-ext.md`。
> 当前为 plan_all 模式 → 进入实施阶段需要用户决策。

**下一步选项**：
1. **立即执行**（subagent-driven 或 inline 执行）—— 我开始实施 Task 1~6
2. **暂停 + review** —— 你 review 此 plan 后告诉我调整
3. **继续产 plan** —— 接着出 hospital-package / escort-business / wallet / review / message / address-coupon / admin 等后端 plan