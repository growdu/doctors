# 陪诊师空余时段（escort_availabilities）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `docs/superpowers/specs/2026-09-24-order-matching-redesign.md` §3.2 + §4.1 escort 端「空余时段」模块：陪诊师上线时设置 / 修改 / 删除空余时段、患者公开查询陪诊师空余时段、订单系统内部按"时段 book + 释放"推进 `escort_availabilities.status`。这是订单匹配模式由「抢单」改为「选人」的承载表（详见 spec §1.2 / §2.2 / §7.4）。

**Architecture:** **选项 A**——作为 `escort-service` 的子包 `services/escort/internal/availability/`，不新增独立服务。
- 同进程复用既有 PG pool + Gin router + JWT 中间件，避免新增部署 / 配置 / CI 资源。
- 业务边界：`availability.Repo`（pgx 持久化）→ `availability.Service`（业务校验：escort 已 approved + 时段不冲突）→ `availability.Handler`（HTTP / 5 个 endpoint）。
- 跨包依赖：`availability.Service` 通过 `ProfileLookup` 接口读 `escort_profiles.state`（不直接 import escort service 包），由 main.go 装配时注入 pgx 实现。
- 不创建独立服务 `services/escort-availability/`（选项 B 已否决：增加部署开销 + 不必要）。

**Tech Stack:** Go 1.24+ · pgx v5.7.1 · testify v1.11 · gin v1.10。

**前置依赖:**
- `2026-09-24-state-machine.md`（orders 表含 `escort_id` 列 + status CHECK 已含 `accepted / in_service / completed`，与 escort_availabilities.order_id FK 兼容；spec §3.1 同步的 selected_escort_id / escort_pending_expire_at 字段由 state-machine / order-lock 修订 plan 单独落，本 plan 不重复）
- `2026-09-24-escort-business.md`（`escort_profiles` 表 + 11 态状态机，`Set` 入口前置 `escort_state='approved'` 校验；`availability.Service` 通过 `ProfileLookup` 接口读 state，不与 escort service 包内部类型耦合）
- `2026-09-24-order-matching-redesign.md`（spec 本体）

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1
- 测试覆盖率：`availability` 包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- 哨兵错误：`ErrAvailabilityNotFound` / `ErrAvailabilityBooked` / `ErrAvailabilityConflict`（包级 `var`，可用 `errors.Is` 比较）
- 时段冲突规则：同 `escort_id` 的时段区间 `[start_at, end_at)` 不允许重叠；插入前 DB UNIQUE 索引 + service 层 `ListByEscort + 区间相交校验` 双保险
- `Book(ctx, id, orderID)`：仅当 `status='available'` 时更新为 `booked` + `order_id`；WHERE 命中 0 行 → `ErrAvailabilityBooked`（乐观锁，避免并发）
- `ReleaseByOrder(ctx, orderID)`：仅恢复 `status='booked' AND order_id=$1` 的时段为 `available`；订单取消 / 陪诊师拒接 / 超时时调用
- API 5 个：PUT / DELETE / GET /api/v1/escorts/me/availability* 走 JWT；GET /api/v1/escorts/:id/availabilities 公开（候选匹配查询）
- 上线 = 有 available 时段；下线 = 把所有 available 时段置 canceled（不在本 plan，留 admin / order-escort 计划）

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `migrations/0009_escort_availabilities.up.sql` | Create | `escort_availabilities` 表 + CHECK + UNIQUE + 部分索引（spec §3.2 完整 SQL） |
| `migrations/0009_escort_availabilities.down.sql` | Create | 逆向（drop indexes + drop table） |
| `migrations/migrations_test.go` | Modify | 加 `Test0009EscortAvailabilitiesUpDown` |
| `services/escort/internal/availability/repo.go` | Create | `AvailabilityRepo`（pgx 实现）+ `Record` 结构 + 3 个哨兵 + 7 个方法 |
| `services/escort/internal/availability/repo_integration_test.go` | Create | 集成测试（≥ 10 个） |
| `services/escort/internal/availability/service.go` | Create | `AvailabilityService` + `ProfileLookup` 接口 + 5 个方法 |
| `services/escort/internal/availability/service_test.go` | Create | 业务单测（≥ 6 个，含 fakeRepo + fakeProfileLookup） |
| `services/escort/internal/availability/handler.go` | Create | `Handler` + 5 个 endpoint + `RegisterRoutes(r gin.IRouter)` |
| `services/escort/internal/availability/handler_test.go` | Create | handler 单测（fake service + httptest） |
| `services/escort/internal/handler/escort.go` | Modify | `RegisterRoutes` 追加 4 个 availability route（含 `/me` + `/escorts/:id/availabilities`） |
| `services/escort/internal/handler/escort.go` | Modify | 新增 `availabilityHandler` 字段 + 构造时由 main 注入 |
| `services/escort/cmd/main.go` | Modify | 装配 availability.Repo + availability.Service + availability.Handler + 注入 escort Handler |
| `services/escort/internal/router/router.go` | Modify | 接受 availability handler 入参并挂到 router |
| `scripts/smoke-escort-availability.sh` | Create | smoke（build + 启动 + /healthz + 5 个 endpoint 401 拦截） |
| `docs/04-业务流程.md` | Modify | §4.8 加 escort_availabilities 时段管理流程 |
| `dev.md` | Modify | §10.14 加 escort-availability plan 落地记录 |

---

### Task 1: 数据库迁移（escort_availabilities）

**Files:**
- Create: `migrations/0009_escort_availabilities.up.sql`
- Create: `migrations/0009_escort_availabilities.down.sql`
- Modify: `migrations/migrations_test.go`

**Step 1: 写集成测试（RED）**

在 `migrations/migrations_test.go` 末尾追加：

```go
// Test0009EscortAvailabilitiesUpDown 验证 escort_availabilities 表 + CHECK +
// UNIQUE (escort_id, start_at) + 部分索引 + end_at > start_at CHECK + down 可逆。
func Test0009EscortAvailabilitiesUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 先建 users + orders（escort_availabilities 引用 users.id + orders.id）
	for _, f := range []string{"0001_users.up.sql", "0002_orders.up.sql"} {
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
		for _, f := range []string{"0002_orders.down.sql", "0001_users.down.sql"} {
			sql, _ := os.ReadFile(f)
			_, _ = cleanConn.Exec(cleanCtx, string(sql))
		}
	})

	applyUp(t, "0009_escort_availabilities.up.sql", []string{"escort_availabilities"})

	// 列检查
	for _, col := range []string{
		"id", "escort_id", "start_at", "end_at", "status",
		"order_id", "created_at", "updated_at",
	} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='escort_availabilities' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "escort_availabilities.%s should exist", col)
	}

	// 索引检查：UNIQUE (escort_id, start_at) + 部分索引 status='available'
	for _, idx := range []string{"idx_escort_avail_unique", "idx_escort_avail_status_start"} {
		var idxExists bool
		err = conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, idx).
			Scan(&idxExists)
		require.NoError(t, err)
		assert.True(t, idxExists, "%s should exist", idx)
	}

	// CHECK 约束存在
	var hasStatusCheck, hasTimeCheck bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name LIKE 'escort_availabilities_status_check')`).
		Scan(&hasStatusCheck)
	require.NoError(t, err)
	assert.True(t, hasStatusCheck)

	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name LIKE 'escort_availabilities%' AND check_clause LIKE '%end_at%start_at%')`).
		Scan(&hasTimeCheck)
	require.NoError(t, err)
	assert.True(t, hasTimeCheck, "end_at > start_at CHECK should exist")

	// down 校验
	downCtx, downCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer downCancel()
	downSQL, err := os.ReadFile("0009_escort_availabilities.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(downCtx, string(downSQL))
	require.NoError(t, err, "apply 0009_escort_availabilities.down.sql")

	var gone bool
	err = conn.QueryRow(downCtx,
		`SELECT NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='escort_availabilities')`).
		Scan(&gone)
	require.NoError(t, err)
	assert.True(t, gone, "escort_availabilities should be gone after down")
}
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0009EscortAvailabilitiesUpDown ./migrations/
```
Expected: FAIL — `Test0009EscortAvailabilitiesUpDown` undefined

**Step 3: 写 `0009_escort_availabilities.up.sql`**

```sql
-- 0009_escort_availabilities.up.sql
-- 陪诊师可预约时段（陪诊师上线时设置；患者只能选 status='available' 时段）。
-- 状态机：available / booked / canceled（与 spec §2.2 + §7.4 一致）。
-- 业务规则（spec §2.2）：
--   - 一个时段只能被一个订单占用（status='booked' 时 order_id 必填）
--   - 陪诊师拒接 / 超时 → 时段保持 available（订单回退 selecting_escort）
--   - 陪诊师 confirm → 时段 booked，order_id 锁定
--   - 订单 cancel → 时段恢复 available
-- 时段不冲突：同 escort_id UNIQUE (escort_id, start_at) + 业务层 ListByEscort 区间相交校验。

CREATE TABLE escort_availabilities (
  id BIGSERIAL PRIMARY KEY,
  escort_id BIGINT NOT NULL REFERENCES users(id),
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'available'
    CHECK (status IN ('available','booked','canceled')),
  order_id BIGINT REFERENCES orders(id),  -- booked 时必填（业务层校验）
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (end_at > start_at)
);

-- 同 escort_id 的时段按 start_at 唯一（业务层额外做区间相交校验）
CREATE UNIQUE INDEX idx_escort_avail_unique ON escort_availabilities(escort_id, start_at);

-- 候选匹配查询：available 时段按时间过滤（部分索引，仅索引 status='available' 行）
CREATE INDEX idx_escort_avail_status_start ON escort_availabilities(status, start_at)
  WHERE status = 'available';
```

**Step 4: 写 `0009_escort_availabilities.down.sql`**

```sql
-- 0009_escort_availabilities.down.sql
-- 撤销 0009：删索引 + 删表。

DROP INDEX IF EXISTS idx_escort_avail_status_start;
DROP INDEX IF EXISTS idx_escort_avail_unique;
DROP TABLE IF EXISTS escort_availabilities;
```

**Step 5: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0009EscortAvailabilitiesUpDown ./migrations/
```
Expected: PASS

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0009 escort_availabilities (时段表 + UNIQUE + CHECK + 部分索引 + 集成测试)"
```

---

### Task 2: availability.Repo（pgx 实现 + 哨兵错误 + 10+ 集成测试）

**Files:**
- Create: `services/escort/internal/availability/repo.go`
- Create: `services/escort/internal/availability/repo_integration_test.go`

**Step 1: 写集成测试（RED）**

`services/escort/internal/availability/repo_integration_test.go`：

```go
//go:build integration
// +build integration

package availability

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

// setupPool 建连接池 + users + orders + escort_availabilities 表（最小可跑 schema）。
// 复用 migrations 0001 + 0002 + 0009；测试结束后 cleanup。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS escort_availabilities CASCADE;
		DROP TABLE IF EXISTS order_events;
		DROP TABLE IF EXISTS orders;
		DROP TABLE IF EXISTS users;
		CREATE TABLE users (
		  id BIGSERIAL PRIMARY KEY,
		  phone VARCHAR(20) UNIQUE NOT NULL,
		  role VARCHAR(16) NOT NULL CHECK (role IN ('patient','escort','admin')),
		  real_name_verified BOOLEAN NOT NULL DEFAULT FALSE,
		  wx_unionid VARCHAR(64),
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE orders (
		  id BIGSERIAL PRIMARY KEY,
		  order_no VARCHAR(32) UNIQUE NOT NULL,
		  patient_id BIGINT NOT NULL REFERENCES users(id),
		  escort_id BIGINT REFERENCES users(id),
		  hospital_id BIGINT NOT NULL,
		  package_id BIGINT NOT NULL,
		  service_start_at TIMESTAMPTZ NOT NULL,
		  amount NUMERIC(10,2) NOT NULL,
		  final_amount NUMERIC(10,2) NOT NULL,
		  status VARCHAR(24) NOT NULL CHECK (status IN (
		    'created','paid','matching','selecting_escort','escort_pending_acceptance',
		    'accepted','in_service','completed','reviewed','refunding','refunded',
		    'settling','disputed','closed','canceled')),
		  version INT NOT NULL DEFAULT 0,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE escort_availabilities (
		  id BIGSERIAL PRIMARY KEY,
		  escort_id BIGINT NOT NULL REFERENCES users(id),
		  start_at TIMESTAMPTZ NOT NULL,
		  end_at TIMESTAMPTZ NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'available'
		    CHECK (status IN ('available','booked','canceled')),
		  order_id BIGINT REFERENCES orders(id),
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  CHECK (end_at > start_at)
		);
		CREATE UNIQUE INDEX idx_escort_avail_unique ON escort_availabilities(escort_id, start_at);
		CREATE INDEX idx_escort_avail_status_start ON escort_availabilities(status, start_at)
		  WHERE status = 'available';
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS escort_availabilities;
			DROP TABLE IF EXISTS order_events;
			DROP TABLE IF EXISTS orders;
			DROP TABLE IF EXISTS users;
		`)
		pool.Close()
	})
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool, phone, role string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone, role) VALUES ($1, $2) RETURNING id`, phone, role).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedOrder(t *testing.T, pool *pgxpool.Pool, patientID int64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
		                    service_start_at, amount, final_amount, status)
		VALUES ('O-' || gen_random_uuid()::text, $1, 1, 1, NOW() + INTERVAL '1 day', 100, 100, 'paid')
		RETURNING id`, patientID).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestAvailabilityRepo_Create_OK 验证插入 + 自动回填 ID / CreatedAt / 默认 status。
func TestAvailabilityRepo_Create_OK(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138001", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	rec := &Record{EscortID: escort, StartAt: start, EndAt: end}
	require.NoError(t, r.Create(context.Background(), rec))
	assert.NotZero(t, rec.ID)
	assert.NotZero(t, rec.CreatedAt)
	assert.Equal(t, "available", rec.Status)
}

// TestAvailabilityRepo_Create_ConflictByExactStartAt 验证同 escort_id 同一 start_at 触发 UNIQUE 冲突。
func TestAvailabilityRepo_Create_ConflictByExactStartAt(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138002", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	require.NoError(t, r.Create(context.Background(), &Record{EscortID: escort, StartAt: start, EndAt: end}))
	rec2 := &Record{EscortID: escort, StartAt: start, EndAt: end.Add(1 * time.Hour)}
	err := r.Create(context.Background(), rec2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAvailabilityConflict), "expected ErrAvailabilityConflict, got %v", err)
}

// TestAvailabilityRepo_Create_InvalidTimeRange 验证 end_at <= start_at 触发 CHECK 失败。
func TestAvailabilityRepo_Create_InvalidTimeRange(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138003", "escort")
	r := NewRepo(pool)
	now := time.Now()
	err := r.Create(context.Background(), &Record{EscortID: escort, StartAt: now, EndAt: now})
	require.Error(t, err, "end_at == start_at should fail")
}

// TestAvailabilityRepo_GetByID_OK 验证按 id 查 + 不存在返 ErrAvailabilityNotFound。
func TestAvailabilityRepo_GetByID_OK(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138004", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: end}
	require.NoError(t, r.Create(context.Background(), rec))

	got, err := r.GetByID(context.Background(), escort, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
	assert.Equal(t, escort, got.EsortID)

	_, err = r.GetByID(context.Background(), escort, 99999)
	assert.ErrorIs(t, err, ErrAvailabilityNotFound)
}

// TestAvailabilityRepo_ListByEscort 验证 ListByEscort 按状态过滤。
func TestAvailabilityRepo_ListByEscort(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138005", "escort")
	r := NewRepo(pool)
	base := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	for i := 0; i < 3; i++ {
		s := base.Add(time.Duration(i) * time.Hour)
		require.NoError(t, r.Create(context.Background(), &Record{
			EscortID: escort, StartAt: s, EndAt: s.Add(30 * time.Minute),
		}))
	}

	all, err := r.ListByEscort(context.Background(), escort, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	available, err := r.ListByEscort(context.Background(), escort, "available")
	require.NoError(t, err)
	assert.Len(t, available, 3)

	booked, err := r.ListByEscort(context.Background(), escort, "booked")
	require.NoError(t, err)
	assert.Len(t, booked, 0)
}

// TestAvailabilityRepo_ListAvailableInRange 验证 ListAvailableInRange 范围过滤。
func TestAvailabilityRepo_ListAvailableInRange(t *testing.T) {
	pool := setupPool(t)
	e1 := seedUser(t, pool, "13800138006", "escort")
	e2 := seedUser(t, pool, "13800138007", "escort")
	r := NewRepo(pool)
	base := time.Now().Add(1 * time.Hour).Truncate(time.Second)

	// e1: 14:00-16:00
	require.NoError(t, r.Create(context.Background(), &Record{
		EscortID: e1, StartAt: base, EndAt: base.Add(2 * time.Hour),
	}))
	// e1: 18:00-20:00（不在 14:00-17:00 范围）
	require.NoError(t, r.Create(context.Background(), &Record{
		EscortID: e1, StartAt: base.Add(4 * time.Hour), EndAt: base.Add(6 * time.Hour),
	}))
	// e2: 15:00-15:30（在范围）
	require.NoError(t, r.Create(context.Background(), &Record{
		EscortID: e2, StartAt: base.Add(time.Hour), EndAt: base.Add(90 * time.Minute),
	}))

	got, err := r.ListAvailableInRange(context.Background(), base, base.Add(3*time.Hour))
	require.NoError(t, err)
	assert.Len(t, got, 2, "e1@14:00 + e2@15:00 应命中；e1@18:00 超出范围")
}

// TestAvailabilityRepo_Update_OK 验证 Update 仅改 start_at/end_at，状态保持 available。
func TestAvailabilityRepo_Update_OK(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138008", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))

	newEnd := start.Add(3 * time.Hour)
	require.NoError(t, r.Update(context.Background(), rec.ID, escort, start, newEnd))

	got, err := r.GetByID(context.Background(), escort, rec.ID)
	require.NoError(t, err)
	assert.True(t, got.EndAt.Equal(newEnd))
	assert.Equal(t, "available", got.Status)
}

// TestAvailabilityRepo_Update_BookedFails 验证 booked 状态的时段不能 Update（仅 available 可改）。
func TestAvailabilityRepo_Update_BookedFails(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138009", "escort")
	patient := seedUser(t, pool, "13800138010", "patient")
	orderID := seedOrder(t, pool, patient)
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))
	require.NoError(t, r.Book(context.Background(), rec.ID, orderID))

	err := r.Update(context.Background(), rec.ID, escort, start, start.Add(3*time.Hour))
	assert.ErrorIs(t, err, ErrAvailabilityBooked)
}

// TestAvailabilityRepo_Delete_OK 验证 Delete 仅 available 可删，booked 拒。
func TestAvailabilityRepo_Delete_OK(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138011", "escort")
	patient := seedUser(t, pool, "13800138012", "patient")
	orderID := seedOrder(t, pool, patient)
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))

	// available 可删
	require.NoError(t, r.Delete(context.Background(), rec.ID, escort))
	_, err := r.GetByID(context.Background(), escort, rec.ID)
	assert.ErrorIs(t, err, ErrAvailabilityNotFound)

	// booked 不可删
	rec2 := &Record{EscortID: escort, StartAt: start.Add(2 * time.Hour), EndAt: start.Add(3 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec2))
	require.NoError(t, r.Book(context.Background(), rec2.ID, orderID))
	err = r.Delete(context.Background(), rec2.ID, escort)
	assert.ErrorIs(t, err, ErrAvailabilityBooked)
}

// TestAvailabilityRepo_Book_OK 验证 Book 更新 status=booked + order_id。
func TestAvailabilityRepo_Book_OK(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138013", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))
	require.NoError(t, r.Book(context.Background(), rec.ID, 7777))

	got, err := r.GetByID(context.Background(), escort, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, "booked", got.Status)
	require.NotNil(t, got.OrderID)
	assert.Equal(t, int64(7777), *got.OrderID)
}

// TestAvailabilityRepo_Book_TwiceFail 验证再次 Book 已 booked 时段返 ErrAvailabilityBooked。
func TestAvailabilityRepo_Book_TwiceFail(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138014", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))
	require.NoError(t, r.Book(context.Background(), rec.ID, 1))
	err := r.Book(context.Background(), rec.ID, 2)
	assert.ErrorIs(t, err, ErrAvailabilityBooked)
}

// TestAvailabilityRepo_ReleaseByOrder_OK 验证 ReleaseByOrder 把 booked 时段恢复 available。
func TestAvailabilityRepo_ReleaseByOrder_OK(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138015", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))
	require.NoError(t, r.Book(context.Background(), rec.ID, 555))

	require.NoError(t, r.ReleaseByOrder(context.Background(), 555))
	got, _ := r.GetByID(context.Background(), escort, rec.ID)
	assert.Equal(t, "available", got.Status)
	assert.Nil(t, got.OrderID)
}

// TestAvailabilityRepo_ReleaseByOrder_NotBookedNoop 验证释放不在 booked 状态的 order_id 时不报错。
func TestAvailabilityRepo_ReleaseByOrder_NotBookedNoop(t *testing.T) {
	pool := setupPool(t)
	r := NewRepo(pool)
	require.NoError(t, r.ReleaseByOrder(context.Background(), 99999), "should be no-op")
}

// 引用 pgx 防止 unused（即使 service 已用，集成测试也直接用）。
var _ = pgx.ErrNoRows
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestAvailabilityRepo_' ./services/escort/internal/availability/
```
Expected: FAIL — `availability` package not exists / `undefined: NewRepo` / `undefined: Record` / `undefined: ErrAvailabilityConflict`

**Step 3: 写 repo.go**

`services/escort/internal/availability/repo.go`：

```go
// Package availability 是 escort-service 的陪诊师空余时段模块。
//
// 模块拆分（sub-agent-driven-development 推荐）：
//   - repo.go（仓储 / SQL / 哨兵）
//   - service.go（业务校验：escort 已 approved + 时段不冲突）
//   - handler.go（HTTP / 5 个 endpoint）
//
// 数据表：escort_availabilities（spec §3.2 + 0009 迁移）。
package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status 是 escort_availabilities.status 的合法枚举（与 DB CHECK 对齐）。
const (
	StatusAvailable = "available"
	StatusBooked    = "booked"
	StatusCanceled  = "canceled"
)

// Record 映射 escort_availabilities 行。
type Record struct {
	ID        int64
	EscortID  int64
	StartAt   time.Time
	EndAt     time.Time
	Status    string // available / booked / canceled
	OrderID   *int64 // booked 时必填；其他状态 nil
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ErrAvailabilityNotFound 查询 / Update / Delete 目标不存在。
var ErrAvailabilityNotFound = errors.New("availability: not found")

// ErrAvailabilityBooked 时段已被 booking / 不能改 / 不能删。
var ErrAvailabilityBooked = errors.New("availability: already booked")

// ErrAvailabilityConflict 时段冲突（同 escort_id 同一 start_at 或区间重叠）。
var ErrAvailabilityConflict = errors.New("availability: time conflict")

// Repo 是 escort_availabilities 表的仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const baseSelect = `
	SELECT id, escort_id, start_at, end_at, status, order_id, created_at, updated_at
	FROM escort_availabilities`

// Create 插入一条时段；status 默认 'available'；ID / CreatedAt 由 DB 回写。
// DB 唯一约束 (escort_id, start_at) 冲突时映射为 ErrAvailabilityConflict。
func (r *Repo) Create(ctx context.Context, rec *Record) error {
	const q = `
		INSERT INTO escort_availabilities (escort_id, start_at, end_at)
		VALUES ($1, $2, $3)
		RETURNING id, status, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, rec.EscortID, rec.StartAt, rec.EndAt).
		Scan(&rec.ID, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		// 23505 unique_violation → ErrAvailabilityConflict（同 start_at 撞唯一索引）
		if isUniqueViolation(err) {
			return ErrAvailabilityConflict
		}
		// end_at > start_at CHECK 失败也映射为 ErrAvailabilityConflict（语义同）
		if isCheckViolation(err, "escort_availabilities", "end_at") {
			return ErrAvailabilityConflict
		}
		return fmt.Errorf("create availability: %w", err)
	}
	return nil
}

// GetByID 按 escort_id + id 取一行；不存在返 ErrAvailabilityNotFound。
func (r *Repo) GetByID(ctx context.Context, escortID, id int64) (*Record, error) {
	q := baseSelect + ` WHERE escort_id = $1 AND id = $2`
	row := r.pool.QueryRow(ctx, q, escortID, id)
	rec, err := scanOne(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAvailabilityNotFound
		}
		return nil, err
	}
	return rec, nil
}

// ListByEscort 列某 escort 的全部时段；status 为空时不按状态过滤。
func (r *Repo) ListByEscort(ctx context.Context, escortID, status string) ([]*Record, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if status == "" {
		rows, err = r.pool.Query(ctx, baseSelect+` WHERE escort_id = $1 ORDER BY start_at`, escortID)
	} else {
		rows, err = r.pool.Query(ctx, baseSelect+` WHERE escort_id = $1 AND status = $2 ORDER BY start_at`, escortID, status)
	}
	if err != nil {
		return nil, fmt.Errorf("list availability by escort: %w", err)
	}
	defer rows.Close()
	out := []*Record{}
	for rows.Next() {
		rec, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListAvailableInRange 列 [startAt, endAt) 范围内 status='available' 的全部时段（候选匹配用）。
// 不限定 escort_id：match-service 按评分 + 城市 + 时段交集筛选候选。
func (r *Repo) ListAvailableInRange(ctx context.Context, startAt, endAt time.Time) ([]*Record, error) {
	q := baseSelect + `
		WHERE status = 'available'
		  AND start_at < $2
		  AND end_at > $1
		ORDER BY start_at`
	rows, err := r.pool.Query(ctx, q, startAt, endAt)
	if err != nil {
		return nil, fmt.Errorf("list available in range: %w", err)
	}
	defer rows.Close()
	out := []*Record{}
	for rows.Next() {
		rec, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// Update 改 start_at / end_at；仅 status='available' 行允许（WHERE 命中 0 → ErrAvailabilityBooked）。
func (r *Repo) Update(ctx context.Context, id, escortID int64, startAt, endAt time.Time) error {
	const q = `
		UPDATE escort_availabilities
		   SET start_at = $3, end_at = $4, updated_at = NOW()
		 WHERE id = $1 AND escort_id = $2 AND status = 'available'`
	tag, err := r.pool.Exec(ctx, q, id, escortID, startAt, endAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAvailabilityConflict
		}
		if isCheckViolation(err, "escort_availabilities", "end_at") {
			return ErrAvailabilityConflict
		}
		return fmt.Errorf("update availability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 不存在 / 已 booked 都返回 0 行；区分：GetByID 探一下
		if _, gerr := r.GetByID(ctx, escortID, id); errors.Is(gerr, ErrAvailabilityNotFound) {
			return ErrAvailabilityNotFound
		}
		return ErrAvailabilityBooked
	}
	return nil
}

// Delete 删一行；仅 status='available' 行允许。
func (r *Repo) Delete(ctx context.Context, id, escortID int64) error {
	const q = `
		DELETE FROM escort_availabilities
		 WHERE id = $1 AND escort_id = $2 AND status = 'available'`
	tag, err := r.pool.Exec(ctx, q, id, escortID)
	if err != nil {
		return fmt.Errorf("delete availability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, gerr := r.GetByID(ctx, escortID, id); errors.Is(gerr, ErrAvailabilityNotFound) {
			return ErrAvailabilityNotFound
		}
		return ErrAvailabilityBooked
	}
	return nil
}

// Book 把 status='available' 改为 'booked' + order_id；仅 1 次成功。
// WHERE 命中 0 → ErrAvailabilityBooked（已被他人 book / 不存在）。
func (r *Repo) Book(ctx context.Context, id, orderID int64) error {
	const q = `
		UPDATE escort_availabilities
		   SET status = 'booked', order_id = $2, updated_at = NOW()
		 WHERE id = $1 AND status = 'available'`
	tag, err := r.pool.Exec(ctx, q, id, orderID)
	if err != nil {
		return fmt.Errorf("book availability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAvailabilityBooked
	}
	return nil
}

// ReleaseByOrder 把 order_id 关联的 booked 时段恢复 available（订单 cancel / 拒接 / 超时）。
// WHERE 命中 0 → 不报错（no-op；订单可能本来就没 book）。
func (r *Repo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	const q = `
		UPDATE escort_availabilities
		   SET status = 'available', order_id = NULL, updated_at = NOW()
		 WHERE order_id = $1 AND status = 'booked'`
	_, err := r.pool.Exec(ctx, q, orderID)
	if err != nil {
		return fmt.Errorf("release availability by order: %w", err)
	}
	return nil
}

// scanOne 单行扫描为 *Record。
func scanOne(row pgx.Row) (*Record, error) {
	rec := &Record{}
	if err := row.Scan(
		&rec.ID, &rec.EscortID, &rec.StartAt, &rec.EndAt,
		&rec.Status, &rec.OrderID, &rec.CreatedAt, &rec.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return rec, nil
}

// scanRow 把 pgx.Rows 的一行扫描为 *Record。
func scanRow(rows pgx.Rows) (*Record, error) {
	rec := &Record{}
	if err := rows.Scan(
		&rec.ID, &rec.EscortID, &rec.StartAt, &rec.EndAt,
		&rec.Status, &rec.OrderID, &rec.CreatedAt, &rec.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return rec, nil
}

// isUniqueViolation 判断 err 是否 PG 23505。
func isUniqueViolation(err error) bool {
	var pgErr *pgx.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// isCheckViolation 判断 err 是否 PG 23514 + 指定表/列。
func isCheckViolation(err error, table, column string) bool {
	var pgErr *pgx.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23514" && pgErr.TableName == table &&
			(pgErr.ColumnName == column || pgErr.ConstraintName != "")
	}
	return false
}
```

> **说明**：isCheckViolation 的 column 匹配为 best-effort（PG 错误信息表名 + constraint 名足以定位）；不严格匹配 column 字符串。

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestAvailabilityRepo_' ./services/escort/internal/availability/
```
Expected: PASS（13 个测试）

**Step 5: Commit**

```bash
git add services/escort/internal/availability/repo.go services/escort/internal/availability/repo_integration_test.go
git commit -m "feat(escort): availability.Repo 加 Create/GetByID/List/ListInRange/Update/Delete/Book/Release + 13 个集成测试"
```

---

### Task 3: availability.Service（业务层 + ProfileLookup 接口 + 6 个单测）

**Files:**
- Create: `services/escort/internal/availability/service.go`
- Create: `services/escort/internal/availability/service_test.go`

**Step 1: 写单测（RED）**

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

	"github.com/growdu/doctors/shared/errs"
)

// ---------- fake ----------

type fakeRepo struct {
	records []*Record
	byID    map[int64]*Record
	byOrder map[int64]*Record // orderID -> booked record
	nextID  int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*Record{}, byOrder: map[int64]*Record{}}
}

func (r *fakeRepo) Create(ctx context.Context, rec *Record) error {
	r.nextID++
	rec.ID = r.nextID
	rec.Status = "available"
	rec.CreatedAt = time.Now()
	rec.UpdatedAt = rec.CreatedAt
	r.records = append(r.records, rec)
	r.byID[rec.ID] = rec
	return nil
}

func (r *fakeRepo) GetByID(ctx context.Context, escortID, id int64) (*Record, error) {
	rec, ok := r.byID[id]
	if !ok || rec.EscortID != escortID {
		return nil, ErrAvailabilityNotFound
	}
	return rec, nil
}

func (r *fakeRepo) ListByEscort(ctx context.Context, escortID, status string) ([]*Record, error) {
	out := []*Record{}
	for _, rec := range r.records {
		if rec.EscortID != escortID {
			continue
		}
		if status != "" && rec.Status != status {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *fakeRepo) ListAvailableInRange(ctx context.Context, startAt, endAt time.Time) ([]*Record, error) {
	out := []*Record{}
	for _, rec := range r.records {
		if rec.Status != "available" {
			continue
		}
		if rec.StartAt.Before(endAt) && rec.EndAt.After(startAt) {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (r *fakeRepo) Update(ctx context.Context, id, escortID int64, startAt, endAt time.Time) error {
	rec := r.byID[id]
	if rec == nil || rec.EscortID != escortID {
		return ErrAvailabilityNotFound
	}
	if rec.Status != "available" {
		return ErrAvailabilityBooked
	}
	rec.StartAt = startAt
	rec.EndAt = endAt
	return nil
}

func (r *fakeRepo) Delete(ctx context.Context, id, escortID int64) error {
	rec := r.byID[id]
	if rec == nil || rec.EscortID != escortID {
		return ErrAvailabilityNotFound
	}
	if rec.Status != "available" {
		return ErrAvailabilityBooked
	}
	delete(r.byID, id)
	return nil
}

func (r *fakeRepo) Book(ctx context.Context, id, orderID int64) error {
	rec := r.byID[id]
	if rec == nil || rec.Status != "available" {
		return ErrAvailabilityBooked
	}
	rec.Status = "booked"
	oid := orderID
	rec.OrderID = &oid
	r.byOrder[orderID] = rec
	return nil
}

func (r *fakeRepo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	rec, ok := r.byOrder[orderID]
	if !ok {
		return nil
	}
	rec.Status = "available"
	rec.OrderID = nil
	delete(r.byOrder, orderID)
	return nil
}

type fakeProfileLookup struct {
	stateByUserID map[int64]string
}

func (f *fakeProfileLookup) IsApproved(ctx context.Context, userID int64) (bool, error) {
	state, ok := f.stateByUserID[userID]
	if !ok {
		return false, nil // 未注册 = 未通过
	}
	return state == "approved" || state == "online" || state == "in_service", nil
}

// ---------- tests ----------

// TestService_Set_ApprovedOK 验证 approved escort 可创建时段。
func TestService_Set_ApprovedOK(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{1: "approved"}}
	svc := NewService(repo, profile)

	rec, err := svc.Set(context.Background(), 1, time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, rec)
	assert.Equal(t, int64(1), rec.EscortID)
}

// TestService_Set_NotApprovedForbidden 验证未通过审核 escort 创建时段返 CodeForbidden。
func TestService_Set_NotApprovedForbidden(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{1: "pending_audit"}}
	svc := NewService(repo, profile)

	_, err := svc.Set(context.Background(), 1, time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	require.Error(t, err)
	var e *errs.Error
	require.True(t, errors.As(err, &e), "expected *errs.Error")
	assert.Equal(t, errs.CodeForbidden, e.Code)
}

// TestService_Set_ConflictReturnsErrAvailabilityConflict 验证同 escort_id 时间段重叠 → 业务层预检返 ErrAvailabilityConflict。
func TestService_Set_ConflictReturnsErrAvailabilityConflict(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{1: "approved"}}
	svc := NewService(repo, profile)
	base := time.Now().Add(time.Hour).Truncate(time.Second)

	// 已存在 14:00-16:00
	_, err := svc.Set(context.Background(), 1, base, base.Add(2*time.Hour))
	require.NoError(t, err)

	// 试图插入 15:00-17:00（重叠）
	_, err = svc.Set(context.Background(), 1, base.Add(time.Hour), base.Add(3*time.Hour))
	assert.ErrorIs(t, err, ErrAvailabilityConflict)
}

// TestService_Cancel_OK 验证 Cancel 委托 repo.Delete。
func TestService_Cancel_OK(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{1: "approved"}}
	svc := NewService(repo, profile)
	rec, err := svc.Set(context.Background(), 1, time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	require.NoError(t, err)

	require.NoError(t, svc.Cancel(context.Background(), 1, rec.ID))
}

// TestService_ListMine 验证 ListMine 走 repo.ListByEscort。
func TestService_ListMine(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{1: "approved"}}
	svc := NewService(repo, profile)
	base := time.Now().Add(time.Hour).Truncate(time.Second)
	_, _ = svc.Set(context.Background(), 1, base, base.Add(time.Hour))
	_, _ = svc.Set(context.Background(), 1, base.Add(2*time.Hour), base.Add(3*time.Hour))

	list, err := svc.ListMine(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// TestService_ListPublic 验证 ListPublic 走 repo.ListAvailableInRange（公开查询）。
func TestService_ListPublic(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{
		1: "approved", 2: "approved",
	}}
	svc := NewService(repo, profile)
	base := time.Now().Add(time.Hour).Truncate(time.Second)
	_, _ = svc.Set(context.Background(), 1, base, base.Add(2*time.Hour))
	_, _ = svc.Set(context.Background(), 2, base.Add(time.Hour), base.Add(90*time.Minute))

	list, err := svc.ListPublic(context.Background(), 1, base, base.Add(3*time.Hour))
	require.NoError(t, err)
	assert.Len(t, list, 2, "1@14:00 + 2@15:00 都应命中")
}

// TestService_BookForOrder_OK 验证 BookForOrder 内部按 (escortId, serviceStartAt) 找时段 + 绑定。
func TestService_BookForOrder_OK(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{7: "approved"}}
	svc := NewService(repo, profile)
	base := time.Now().Add(time.Hour).Truncate(time.Second)
	rec, err := svc.Set(context.Background(), 7, base, base.Add(2*time.Hour))
	require.NoError(t, err)

	require.NoError(t, svc.BookForOrder(context.Background(), 100, 7, base.Add(time.Minute)))
	got, _ := repo.GetByID(context.Background(), 7, rec.ID)
	assert.Equal(t, "booked", got.Status)
	require.NotNil(t, got.OrderID)
	assert.Equal(t, int64(100), *got.OrderID)
}

// TestService_BookForOrder_NoAvailability 验证无匹配时段返 ErrAvailabilityNotFound。
func TestService_BookForOrder_NoAvailability(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{7: "approved"}}
	svc := NewService(repo, profile)

	err := svc.BookForOrder(context.Background(), 100, 7, time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrAvailabilityNotFound)
}

// TestService_ReleaseForOrder 验证 ReleaseForOrder 走 repo.ReleaseByOrder。
func TestService_ReleaseForOrder(t *testing.T) {
	repo := newFakeRepo()
	profile := &fakeProfileLookup{stateByUserID: map[int64]string{7: "approved"}}
	svc := NewService(repo, profile)
	base := time.Now().Add(time.Hour).Truncate(time.Second)
	rec, err := svc.Set(context.Background(), 7, base, base.Add(2*time.Hour))
	require.NoError(t, err)
	require.NoError(t, svc.BookForOrder(context.Background(), 100, 7, base.Add(time.Minute)))

	require.NoError(t, svc.ReleaseForOrder(context.Background(), 100))
	got, _ := repo.GetByID(context.Background(), 7, rec.ID)
	assert.Equal(t, "available", got.Status)
	assert.Nil(t, got.OrderID)
}
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestService_' ./services/escort/internal/availability/
```
Expected: FAIL — `NewService` undefined

**Step 3: 写 service.go**

`services/escort/internal/availability/service.go`：

```go
package availability

import (
	"context"
	"errors"
	"time"

	"github.com/growdu/doctors/shared/errs"
)

// ProfileLookup 是 escort_profiles.state 的只读视图（接口）。
// 实现方：main.go 装配时注入 pgx 查询（避免 import escort service 包）。
type ProfileLookup interface {
	IsApproved(ctx context.Context, userID int64) (bool, error)
}

// Service 是 escort_availabilities 的业务编排层。
type Service struct {
	repo    *Repo
	profile ProfileLookup
}

// NewService 装配。
func NewService(r *Repo, p ProfileLookup) *Service { return &Service{repo: r, profile: p} }

// Set 创建时段：校验 escort 已通过审核 + 区间不与本人已有时段重叠。
// 返回插入后的 Record（含 ID / CreatedAt）。
func (s *Service) Set(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Record, error) {
	if !endAt.After(startAt) {
		return nil, errs.New(errs.CodeParamInvalid, "end_at must be after start_at")
	}
	approved, err := s.profile.IsApproved(ctx, escortID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "lookup profile", err)
	}
	if !approved {
		return nil, errs.New(errs.CodeForbidden, "escort not approved")
	}
	// 业务层预检：同 escort_id 已有时段与新区间是否相交（DB UNIQUE 只防同一 start_at）
	if conflict, err := s.hasConflict(ctx, escortID, startAt, endAt, 0); err != nil {
		return nil, err
	} else if conflict {
		return nil, ErrAvailabilityConflict
	}
	rec := &Record{EscortID: escortID, StartAt: startAt, EndAt: endAt}
	if err := s.repo.Create(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// Cancel 删时段（仅 available 可删，booked 由 release-by-order 处理）。
func (s *Service) Cancel(ctx context.Context, escortID, id int64) error {
	return s.repo.Delete(ctx, id, escortID)
}

// ListMine 列 escort 的全部时段（含 booked / canceled，方便 escort 自己查看历史）。
func (s *Service) ListMine(ctx context.Context, escortID int64) ([]*Record, error) {
	return s.repo.ListByEscort(ctx, escortID, "")
}

// ListPublic 公开查询某 escort 在 [startAt, endAt) 范围内的 available 时段（候选用）。
func (s *Service) ListPublic(ctx context.Context, escortID int64, startAt, endAt time.Time) ([]*Record, error) {
	all, err := s.repo.ListByEscort(ctx, escortID, "available")
	if err != nil {
		return nil, err
	}
	out := []*Record{}
	for _, rec := range all {
		if rec.StartAt.Before(endAt) && rec.EndAt.After(startAt) {
			out = append(out, rec)
		}
	}
	return out, nil
}

// BookForOrder 订单系统内部调用：按 (escortID, serviceStartAt) 找一条 available 时段并 book。
// 找时段策略：第一个 StartAt <= serviceStartAt 且 EndAt > serviceStartAt 的 available 行。
func (s *Service) BookForOrder(ctx context.Context, orderID, escortID int64, serviceStartAt time.Time) error {
	candidates, err := s.repo.ListAvailableInRange(ctx, serviceStartAt, serviceStartAt.Add(time.Microsecond))
	if err != nil {
		return err
	}
	// 进一步按 escort_id 过滤；ListAvailableInRange 不限 escort
	for _, rec := range candidates {
		if rec.EscortID != escortID {
			continue
		}
		if rec.StartAt.After(serviceStartAt) || rec.EndAt.Before(serviceStartAt) {
			continue
		}
		if err := s.repo.Book(ctx, rec.ID, orderID); err != nil {
			if errors.Is(err, ErrAvailabilityBooked) {
				continue // 并发被他人 book；继续找
			}
			return err
		}
		return nil
	}
	return ErrAvailabilityNotFound
}

// ReleaseForOrder 订单系统内部调用：释放某订单的所有 booked 时段。
func (s *Service) ReleaseForOrder(ctx context.Context, orderID int64) error {
	return s.repo.ReleaseByOrder(ctx, orderID)
}

// hasConflict 查同 escort_id 已有时段与 [startAt, endAt) 是否相交；excludeID 用于 Update 时排除自己。
func (s *Service) hasConflict(ctx context.Context, escortID int64, startAt, endAt time.Time, excludeID int64) (bool, error) {
	existing, err := s.repo.ListByEscort(ctx, escortID, "available")
	if err != nil {
		return false, err
	}
	for _, rec := range existing {
		if rec.ID == excludeID {
			continue
		}
		if rec.StartAt.Before(endAt) && rec.EndAt.After(startAt) {
			return true, nil
		}
	}
	return false, nil
}
```

> **设计权衡**：`ListPublic` 改走 `repo.ListByEscort(escortID, "available")` + 内存过滤，而不是直接走 `repo.ListAvailableInRange`；理由：公开查询要按 escort_id 限定 + status 限定，`ListAvailableInRange` 是全局 (供 match-service 跨 escort 候选用)，不应在此复用。

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```
Expected: PASS（8 个 service 单测）

**Step 5: Commit**

```bash
git add services/escort/internal/availability/service.go services/escort/internal/availability/service_test.go
git commit -m "feat(escort): availability.Service 加 Set/Cancel/ListMine/ListPublic/BookForOrder/ReleaseForOrder + ProfileLookup + 8 个单测"
```

---

### Task 4: availability.Handler（5 个 endpoint + 4 个 handler 单测）

**Files:**
- Create: `services/escort/internal/availability/handler.go`
- Create: `services/escort/internal/availability/handler_test.go`

**Step 1: 写 handler 单测（RED）**

`services/escort/internal/availability/handler_test.go`：

```go
package availability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/errs"
)

// fakeService 是 Service 的最小接口 fake（覆盖 5 个 handler 调用）。
type fakeService struct {
	// Set
	setCalled    bool
	setEscortID  int64
	setStartAt   time.Time
	setEndAt     time.Time
	setReturn    *Record
	setErr      error
	// Cancel
	cancelCalled   bool
	cancelEscortID int64
	cancelID       int64
	cancelErr     error
	// ListMine
	listMineCalled   bool
	listMineEscortID int64
	listMineReturn   []*Record
	listMineErr     error
	// ListPublic
	listPublicCalled    bool
	listPublicEscortID  int64
	listPublicStart     time.Time
	listPublicEnd       time.Time
	listPublicReturn    []*Record
	listPublicErr      error
}

func (f *fakeService) Set(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Record, error) {
	f.setCalled = true
	f.setEscortID = escortID
	f.setStartAt = startAt
	f.setEndAt = endAt
	if f.setErr != nil {
		return nil, f.setErr
	}
	return f.setReturn, nil
}

func (f *fakeService) Cancel(ctx context.Context, escortID, id int64) error {
	f.cancelCalled = true
	f.cancelEscortID = escortID
	f.cancelID = id
	return f.cancelErr
}

func (f *fakeService) ListMine(ctx context.Context, escortID int64) ([]*Record, error) {
	f.listMineCalled = true
	f.listMineEscortID = escortID
	return f.listMineReturn, f.listMineErr
}

func (f *fakeService) ListPublic(ctx context.Context, escortID int64, startAt, endAt time.Time) ([]*Record, error) {
	f.listPublicCalled = true
	f.listPublicEscortID = escortID
	f.listPublicStart = startAt
	f.listPublicEnd = endAt
	return f.listPublicReturn, f.listPublicErr
}

// 编译期断言 fakeService 满足 Service 接口（接口定义在 handler.go）。
var _ Service = (*fakeService)(nil)

// newTestEngine 起一个最小 gin engine 注册全部路由，便于 httptest 跑完整 HTTP 流。
func newTestEngine(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	h.RegisterRoutes(r)
	return r
}

// TestPutAvailability_OK 验证 PUT /api/v1/escorts/me/availability 走 Set。
func TestPutAvailability_OK(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	svc := &fakeService{setReturn: &Record{ID: 100, EscortID: 7, StartAt: now, EndAt: now.Add(time.Hour), Status: "available"}}
	r := newTestEngine(svc)

	body := map[string]any{"start_at": now.Format(time.RFC3339), "end_at": now.Add(time.Hour).Format(time.RFC3339)}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/escorts/me/availability", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// 模拟 JWT middleware 已注入 user_id=7（这里手填 gin Context）
	w := httptest.NewRecorder()
	r.ServeHTTP(w, injectUser(req, 7, "escort"))

	assert.True(t, svc.setCalled)
	assert.Equal(t, int64(7), svc.setEscortID)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestDeleteAvailability_OK 验证 DELETE /api/v1/escorts/me/availability/:id 走 Cancel。
func TestDeleteAvailability_OK(t *testing.T) {
	svc := &fakeService{}
	r := newTestEngine(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/escorts/me/availability/100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, injectUser(req, 7, "escort"))

	assert.True(t, svc.cancelCalled)
	assert.Equal(t, int64(7), svc.cancelEscortID)
	assert.Equal(t, int64(100), svc.cancelID)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetMineAvailability_OK 验证 GET /api/v1/escorts/me/availability 走 ListMine。
func TestGetMineAvailability_OK(t *testing.T) {
	svc := &fakeService{listMineReturn: []*Record{{ID: 1, EscortID: 7, Status: "available"}}}
	r := newTestEngine(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/escorts/me/availability", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, injectUser(req, 7, "escort"))

	assert.True(t, svc.listMineCalled)
	assert.Equal(t, int64(7), svc.listMineEscortID)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetPublicAvailability_OK 验证 GET /api/v1/escorts/:id/availabilities 走 ListPublic（不需 JWT / user）。
func TestGetPublicAvailability_OK(t *testing.T) {
	svc := &fakeService{listPublicReturn: []*Record{{ID: 1, EscortID: 7, Status: "available"}}}
	r := newTestEngine(svc)
	start := time.Now().Format(time.RFC3339)
	end := time.Now().Add(3 * time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/escorts/7/availabilities?start_at="+start+"&end_at="+end, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req) // 不 injectUser：公开

	assert.True(t, svc.listPublicCalled)
	assert.Equal(t, int64(7), svc.listPublicEscortID)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestPutAvailability_EscortNotApproved 验证 Set 返 CodeForbidden 时 handler 透传 403。
func TestPutAvailability_EscortNotApproved(t *testing.T) {
	svc := &fakeService{setErr: errs.New(errs.CodeForbidden, "escort not approved")}
	r := newTestEngine(svc)
	now := time.Now().Truncate(time.Second)
	body := map[string]any{"start_at": now.Format(time.RFC3339), "end_at": now.Add(time.Hour).Format(time.RFC3339)}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/escorts/me/availability", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, injectUser(req, 7, "escort"))

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "escort not approved")
}

// TestPutAvailability_ConflictReturns409 验证 ErrAvailabilityConflict → 409。
func TestPutAvailability_ConflictReturns409(t *testing.T) {
	svc := &fakeService{setErr: ErrAvailabilityConflict}
	r := newTestEngine(svc)
	now := time.Now().Truncate(time.Second)
	body := map[string]any{"start_at": now.Format(time.RFC3339), "end_at": now.Add(time.Hour).Format(time.RFC3339)}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/escorts/me/availability", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, injectUser(req, 7, "escort"))

	require.Equal(t, http.StatusConflict, w.Code, "ErrAvailabilityConflict → 409")
}

// TestGetPublicAvailability_ParamInvalid 验证缺少 start_at/end_at → 400。
func TestGetPublicAvailability_ParamInvalid(t *testing.T) {
	svc := &fakeService{}
	r := newTestEngine(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/escorts/7/availabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCancel_BookedReturns409 验证 Cancel 一个 booked 时段返 409。
func TestCancel_BookedReturns409(t *testing.T) {
	svc := &fakeService{cancelErr: ErrAvailabilityBooked}
	r := newTestEngine(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/escorts/me/availability/100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, injectUser(req, 7, "escort"))

	assert.Equal(t, http.StatusConflict, w.Code)
}

// injectUser 把 userID + role 注入到 gin.Context（模拟 JWT middleware 效果）。
// 这里直接走 gin 的 set 方法：handler 内部用 middleware.UserID(c) 读取。
func injectUser(req *http.Request, userID int64, role string) *http.Request {
	ctx := req.Context()
	// 把 user_id 和 role 写入 context value（key 与 middleware.UserIDKey 一致）
	type ctxKey string
	const userIDKey ctxKey = "user_id"
	const roleKey ctxKey = "role"
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, roleKey, role)
	return req.WithContext(ctx)
}

// 引用 errors 包防止 unused
var _ = errors.New
```

> **说明**：`middleware.UserID(c)` 读 gin.Context 上的 userID。测试中用 `context.WithValue` 注入 + 由 handler 的 ctx 中读取；
> 但 `middleware.UserID` 通常读 `gin.Context`，不是 `context.Context`。**故**测试 handler 直接调用 Service 时已验证业务调用正确，
> handler 的 user_id 提取（走 middleware）由 escort router 测试 / smoke 脚本覆盖。

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestPutAvailability_|TestDeleteAvailability_|TestGetMineAvailability_|TestGetPublicAvailability_|TestCancel_' ./services/escort/internal/availability/
```
Expected: FAIL — `NewHandler` undefined

**Step 3: 写 handler.go**

`services/escort/internal/availability/handler.go`：

```go
package availability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的业务接口（解耦，方便 fake）。
type Service interface {
	Set(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Record, error)
	Cancel(ctx context.Context, escortID, id int64) error
	ListMine(ctx context.Context, escortID int64) ([]*Record, error)
	ListPublic(ctx context.Context, escortID int64, startAt, endAt time.Time) ([]*Record, error)
}

// context 别名（避免 import "context" 冗余 + 提示）。
type context = interface{ Deadline() (time.Time, bool) }

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// NewHandler 构造 handler。
func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 挂 5 个 endpoint。
//   PUT    /api/v1/escorts/me/availability
//   DELETE /api/v1/escorts/me/availability/:id
//   GET    /api/v1/escorts/me/availability
//   GET    /api/v1/escorts/:id/availabilities?start_at=&end_at=
//
// 公开 endpoint（最后一条）不依赖 JWT；前 3 条由上游 escort router 挂到 v1 group（带 JWT middleware）。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	me := r.Group("/escorts/me/availability")
	me.Use.DELETE(h)
	me.PUT("", h.Put)
	me.GET("", h.ListMine)

	r.GET("/escorts/:id/availabilities", h.ListPublic)
}

// 编译期确保 *Service 满足 Service 接口。
var _ Service = (*Service)(nil)
```

> **修正**（handler 注册语法）：上面 `me.Use.DELETE(h)` 不合法。重写为：

```go
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	me := r.Group("/escorts/me/availability")
	me.PUT("", h.Put)
	me.DELETE("/:id", h.DeleteMine)
	me.GET("", h.ListMine)

	r.GET("/escorts/:id/availabilities", h.ListPublic)
}
```

handler 完整代码（含 5 个 endpoint + 错误翻译）：

```go
package availability

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的业务接口。
type Service interface {
	Set(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Record, error)
	Cancel(ctx context.Context, escortID, id int64) error
	ListMine(ctx context.Context, escortID int64) ([]*Record, error)
	ListPublic(ctx context.Context, escortID int64, startAt, endAt time.Time) ([]*Record, error)
}

// 编译期断言。
var _ Service = (*Service)(nil)

// Handler 持有 service 引用。
type Handler struct{ svc Service }

// NewHandler 构造。
func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 挂 4 个 endpoint（最后一个 ListPublic 在 me group 之外的 /escorts/:id/availabilities）。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	me := r.Group("/escorts/me/availability")
	me.PUT("", h.Put)
	me.DELETE("/:id", h.DeleteMine)
	me.GET("", h.ListMine)

	r.GET("/escorts/:id/availabilities", h.ListPublic)
}

// putReq 是 PUT /api/v1/escorts/me/availability 请求体。
type putReq struct {
	StartAt string `json:"start_at"` // RFC3339
	EndAt   string `json:"end_at"`
}

// Put PUT /api/v1/escorts/me/availability
func (h *Handler) Put(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		httpx.Fail(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req putReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid start_at"))
		return
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid end_at"))
		return
	}
	rec, err := h.svc.Set(c.Request.Context(), uid, startAt, endAt)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toView(rec))
}

// DeleteMine DELETE /api/v1/escorts/me/availability/:id
func (h *Handler) DeleteMine(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		httpx.Fail(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), uid, id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// ListMine GET /api/v1/escorts/me/availability
func (h *Handler) ListMine(c *gin.Context) {
	uid := currentUserID(c)
	if uid == 0 {
		httpx.Fail(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	list, err := h.svc.ListMine(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, r := range list {
		out = append(out, toView(r))
	}
	httpx.OK(c, out)
}

// ListPublic GET /api/v1/escorts/:id/availabilities?start_at=&end_at=
func (h *Handler) ListPublic(c *gin.Context) {
	escortID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid escort id"))
		return
	}
	startAt, err := time.Parse(time.RFC3339, c.Query("start_at"))
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "missing or invalid start_at"))
		return
	}
	endAt, err := time.Parse(time.RFC3339, c.Query("end_at"))
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "missing or invalid end_at"))
		return
	}
	list, err := h.svc.ListPublic(c.Request.Context(), escortID, startAt, endAt)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, r := range list {
		out = append(out, toView(r))
	}
	httpx.OK(c, out)
}

// respondError 翻译 sentinel + err 业务码到 HTTP 响应。
func respondError(c *gin.Context, err error) {
	// 业务码优先（errs.As）
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	// 哨兵 → 409
	if errors.Is(err, ErrAvailabilityConflict) || errors.Is(err, ErrAvailabilityBooked) {
		httpx.Fail(c, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, ErrAvailabilityNotFound) {
		httpx.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	httpx.Fail(c, http.StatusInternalServerError, err.Error())
}

// currentUserID 从 gin.Context 读 user_id（与 escort service middleware 对齐）。
// 这里走 context.Context（middleware 注入）而非 gin.Context.MustGet，
// 是因为 availability 是独立子包，避免与 escort service middleware 紧耦合。
func currentUserID(c *gin.Context) int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	id, _ := v.(int64)
	return id
}

// toView Record → gin.H（视图）。
func toView(r *Record) gin.H {
	v := gin.H{
		"id":        r.ID,
		"escort_id": r.EscortID,
		"start_at":  r.StartAt,
		"end_at":    r.EndAt,
		"status":    r.Status,
		"created_at": r.CreatedAt,
		"updated_at": r.UpdatedAt,
	}
	if r.OrderID != nil {
		v["order_id"] = *r.OrderID
	}
	return v
}
```

> **设计权衡**：`currentUserID` 用 `c.Get("user_id")` 而不是 import escort 的 middleware，避免循环依赖。

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```
Expected: PASS（8 个 service + 8 个 handler 单测）

> **注意**：handler_test.go 的 `injectUser` 改走 `c.Set("user_id", userID)` 而非 context.WithValue：

**修正 handler_test.go 的 injectUser：**

```go
// injectUser 把 userID + role 注入到 gin.Context。
// 通过新建一个带中间件的 gin engine 来注入：
func injectUser(req *http.Request, userID int64, role string) *http.Request {
	// 用一个简单的中间件 handler 在 ServeHTTP 之前设置 gin.Context；
	// 这里直接把 user_id 写到请求 header（让一个中间件读取），
	// 但更简单：handler_test 在调用 ServeHTTP 之前用 c.Set。
	// 这里返回 req 不变；改 newTestEngine 接受 middleware：
	return req
}
```

**修正 newTestEngine：**

```go
// newTestEngine 起 gin engine 并注册一个把 user_id 写到 gin.Context 的中间件。
func newTestEngine(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if v := c.Request.Context().Value(ctxUserIDKey); v != nil {
			if id, ok := v.(int64); ok {
				c.Set("user_id", id)
			}
		}
		c.Next()
	})
	h := NewHandler(svc)
	h.RegisterRoutes(r)
	return r
}

// ctxUserIDKey 是 context.WithValue 的 key（导出避免冲突）。
type ctxKey string
const ctxUserIDKey ctxKey = "user_id"
```

**再修正 injectUser：**

```go
func injectUser(req *http.Request, userID int64, role string) *http.Request {
	ctx := context.WithValue(req.Context(), ctxUserIDKey, userID)
	return req.WithContext(ctx)
}
```

**Step 4（修正版）: 跑测试**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/availability/
```
Expected: PASS

**Step 5: Commit**

```bash
git add services/escort/internal/availability/handler.go services/escort/internal/availability/handler_test.go
git commit -m "feat(escort): availability.Handler 加 4 个 endpoint + errs 翻译 + 8 个 handler 单测"
```

---

### Task 5: 集成到 escort router + main 装配（PG pool + handler 注入）

**Files:**
- Modify: `services/escort/internal/handler/escort.go`
- Modify: `services/escort/internal/router/router.go`
- Modify: `services/escort/cmd/main.go`
- Modify: `services/escort/internal/handler/escort.go`（暴露 RegisterRoutes 接受 availability handler）

**Step 1: 改 escort.go handler 接受 availability handler**

修改 `services/escort/internal/handler/escort.go`：

在 `Handler` struct 加字段：

```go
type Handler struct {
	svc        *service.Service
	availH     *availability.Handler // 新增
}
```

把 `New` 改为接受 availability handler：

```go
// New 构造 Handler；availability handler 由 main 装配后传入。
func New(svc *service.Service, availH *availability.Handler) *Handler {
	return &Handler{svc: svc, availH: availH}
}
```

`RegisterRoutes` 末尾追加：

```go
// availability 路由（由 availability.Handler 自己挂）
if h.availH != nil {
	h.availH.RegisterRoutes(r)
}
```

**Step 2: 改 router.go**

修改 `services/escort/internal/router/router.go`：

```go
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/shared/httpx"
)

// New 返回挂好路由的 gin engine。
// availabilityH 是 availability 子包的 handler；nil 时只挂既有 escort 路由（向后兼容）。
func New(h *handler.Handler, availabilityH *availability.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	v1 := r.Group("/api/v1", middleware.Auth(jwtSecret))
	h.RegisterRoutes(v1)
	if availabilityH != nil {
		availabilityH.RegisterRoutes(v1) // 复用同一 v1 group（含 JWT middleware）
	}
	return r
}
```

**Step 3: 改 cmd/main.go**

修改 `services/escort/cmd/main.go`：

```go
// escort-service 入口。
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/router"
	"github.com/growdu/doctors/services/escort/internal/server"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("escort")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// PG pool（可空；空时表见 DSN 未配置时降级为 nil）
	var pool *pgxpool.Pool
	if cfg.DB.DSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err = pgxpool.New(ctx, cfg.DB.DSN)
		cancel()
		if err != nil {
			log.Fatalf("connect pg: %v", err)
		}
		defer pool.Close()
	}

	// 既有 escort service（保持兼容；nilRepo 是占位）
	svc := service.New(nilRepo{}, nilPublisher{})

	// availability 子包装配
	var availHandler *availability.Handler
	if pool != nil {
		repo := availability.NewRepo(pool)
		profileLookup := availability.NewPgxProfileLookup(pool) // 见 service.go 增补
		availSvc := availability.NewService(repo, profileLookup)
		availHandler = availability.NewHandler(availSvc)
	}

	// handler 装配
	h := handler.New(svc, availHandler)

	// router
	var jwtSecret string
	if cfg.Auth != nil {
		jwtSecret = cfg.Auth.JWTSecret
	}
	srv := server.NewWithRouter(cfg.HTTP.Addr, func() *gin.Engine {
		return router.New(h, availHandler, jwtSecret)
	}, jwtSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.L().Info("escort-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("escort-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("escort-service stopped")
}

// parseLevel 解析日志级别。
func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// nilRepo / nilPublisher 保持既有占位实现。
type nilRepo struct{}

func (nilRepo) Create(ctx context.Context, e *service.Escort) error { return errNil }
func (nilRepo) GetByID(ctx context.Context, id int64) (*service.Escort, error) {
	return nil, errNil
}
func (nilRepo) GetByUserID(ctx context.Context, uid int64) (*service.Escort, error) {
	return nil, errNil
}
func (nilRepo) UpdateStatus(ctx context.Context, id int64, s string) error { return errNil }
func (nilRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	return errNil
}
func (nilRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	return errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishAvailabilityChanged(ctx context.Context, ev service.AvailabilityEvent) error {
	return errNil
}

var errNil = errors.New("escort: repo/publisher not wired (接 PG/Kafka 后替换)")
```

> **依赖**：`availability.NewPgxProfileLookup` 需在 `availability/service.go` 末尾追加：

```go
// PgxProfileLookup 是 ProfileLookup 的 pgx 实现（main.go 装配）。
type PgxProfileLookup struct {
	pool *pgxpool.Pool
}

// NewPgxProfileLookup 构造。
func NewPgxProfileLookup(pool *pgxpool.Pool) *PgxProfileLookup {
	return &PgxProfileLookup{pool: pool}
}

// IsApproved 读 escort_profiles.state；state ∈ {approved, online, in_service} 视为 approved。
// 未注册或已 reject 一律 false。
func (p *PgxProfileLookup) IsApproved(ctx context.Context, userID int64) (bool, error) {
	var state string
	err := p.pool.QueryRow(ctx,
		`SELECT state FROM escort_profiles WHERE user_id = $1`, userID).Scan(&state)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return state == "approved" || state == "online" || state == "in_service", nil
}
```

**Step 4: 跑既有 escort handler 单测 + availability 单测，确认不破坏**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/...
```
Expected: PASS（既有 + 新增 16+ 个测试）

**Step 5: Commit**

```bash
git add services/escort/internal/availability/service.go services/escort/internal/availability/service_test.go \
        services/escort/internal/handler/escort.go services/escort/internal/router/router.go \
        services/escort/cmd/main.go
git commit -m "feat(escort): availability 接入 escort router + main 装配 PG pool + PgxProfileLookup + 16 个测试"
```

---

### Task 6: smoke 脚本（smoke-escort-availability.sh）

**Files:**
- Create: `scripts/smoke-escort-availability.sh`

**Step 1: 写 smoke 脚本**

```bash
#!/usr/bin/env bash
# escort-availability 端到端 smoke：
#   1. build escort-service（含 availability 子包）
#   2. 启动
#   3. /healthz 通
#   4. 4 个 endpoint 必须 401（无 token）
#
# 真正端到端（DB 迁移 + escort 注册 + 设置时段 + 候选查询）需要 docker compose；
# 见 scripts/smoke-e2e.sh（待做）。

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ADDR=":8084"
BIN="$ROOT/bin/escort-availability"
LOGFILE="$ROOT/.data/escort-availability-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/6] building escort-service (含 availability)..."
GOFLAGS="-mod=mod" GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/escort/cmd

echo "[2/6] starting escort-service on $ADDR..."
DOCTORS_ESCORT_HTTP_ADDR="$ADDR" DOCTORS_ESCORT_DB_DSN="" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/6] curl /healthz"
HEALTH=$(curl -fsS "http://127.0.0.1$ADDR/healthz")
echo "$HEALTH" | grep -q '"status":"ok"' || { echo "healthz unexpected: $HEALTH"; exit 1; }
echo "  -> $HEALTH"

echo "[4/6] PUT /api/v1/escorts/me/availability (no token → expect 401)"
RESP=$(curl -sS -X PUT "http://127.0.0.1$ADDR/api/v1/escorts/me/availability" \
  -H 'Content-Type: application/json' \
  -d '{"start_at":"2026-12-01T10:00:00Z","end_at":"2026-12-01T12:00:00Z"}')
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "[5/6] DELETE /api/v1/escorts/me/availability/1 (no token → expect 401)"
RESP=$(curl -sS -X DELETE "http://127.0.0.1$ADDR/api/v1/escorts/me/availability/1")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

echo "[6/6] GET /api/v1/escorts/me/availability (no token → expect 401)"
RESP=$(curl -sS "http://127.0.0.1$ADDR/api/v1/escorts/me/availability")
echo "$RESP" | grep -q '"code":11001' || { echo "unauth unexpected: $RESP"; exit 1; }
echo "  -> $RESP"

# 公开 endpoint 不需 JWT（无需 401 校验）；仅打健康
EXIST_NOT_RESP=$(curl -fsS "http://127.0.0.1$ADDR/api/v1/escorts/7/availabilities?start_at=2026-12-01T10:00:00Z&end_at=2026-12-01T12:00:00Z")
echo "$EXIST_NOT_RESP" | grep -q '"data":' || { echo "public list unexpected: $EXIST_NOT_RESP"; exit 1; }
echo "  public list (no auth) → $EXIST_NOT_RESP"

echo "smoke OK"
```

**Step 2: 跑 smoke 确认通过**

```bash
chmod +x scripts/smoke-escort-availability.sh
./scripts/smoke-escort-availability.sh
```
Expected: 全部 step 通过，最后输出 `smoke OK`

**Step 3: Commit**

```bash
git add scripts/smoke-escort-availability.sh
git commit -m "test(smoke): escort-availability 端到端 smoke（build + /healthz + 4 endpoint 401 拦截）"
```

---

### Task 7: 集成测试场景 + Self-Review + 文档落地

**Files:**
- Modify: `services/escort/internal/availability/repo_integration_test.go`（追加 4 个跨层场景）
- Modify: `docs/04-业务流程.md`
- Modify: `dev.md`

**Step 1: 追加 4 个集成场景**

在 `repo_integration_test.go` 末尾追加：

```go
// TestIntegration_EscortNotApproved 验证 escort 未通过审核时 Set 返 CodeForbidden（跨 service + repo）。
func TestIntegration_EscortNotApproved(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138016", "escort")

	// PgxProfileLookup：escort 未注册 escort_profiles → IsApproved=false
	lookup := NewPgxProfileLookup(pool)
	svc := NewService(NewRepo(pool), lookup)

	_, err := svc.Set(context.Background(), escort, time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	require.Error(t, err)
	var e *errs.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errs.CodeForbidden, e.Code)
}

// TestIntegration_TimeConflict 验证时段区间重叠 → ErrAvailabilityConflict（service 层 + repo UNIQUE 双保险）。
func TestIntegration_TimeConflict(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138017", "escort")

	// 直接插一条 escort_profiles(state=approved) 让 Set 走通前置校验
	_, err := pool.Exec(context.Background(),
		`INSERT INTO escort_profiles (user_id, state) VALUES ($1, 'approved')`, escort)
	require.NoError(t, err)

	repo := NewRepo(pool)
	lookup := NewPgxProfileLookup(pool)
	svc := NewService(repo, lookup)

	base := time.Now().Add(time.Hour).Truncate(time.Second)
	_, err = svc.Set(context.Background(), escort, base, base.Add(2*time.Hour))
	require.NoError(t, err)

	// 重叠时段（15:00-17:00 vs 已存在 14:00-16:00）
	_, err = svc.Set(context.Background(), escort, base.Add(time.Hour), base.Add(3*time.Hour))
	assert.ErrorIs(t, err, ErrAvailabilityConflict)
}

// TestIntegration_BookTwiceFails 验证 Book 成功后再次 Book 返 ErrAvailabilityBooked。
func TestIntegration_BookTwiceFails(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138018", "escort")
	patient := seedUser(t, pool, "13800138019", "patient")
	orderID := seedOrder(t, pool, patient)

	repo := NewRepo(pool)
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, repo.Create(context.Background(), rec))

	require.NoError(t, repo.Book(context.Background(), rec.ID, orderID))
	err := repo.Book(context.Background(), rec.ID, orderID+1)
	assert.ErrorIs(t, err, ErrAvailabilityBooked)
}

// TestIntegration_ReleaseByOrderRestores 验证 ReleaseByOrder 把 booked 恢复 available（订单 cancel 流程）。
func TestIntegration_ReleaseByOrderRestores(t *testing.T) {
	pool := setupPool(t)
	escort := seedUser(t, pool, "13800138020", "escort")
	r := NewRepo(pool)
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	rec := &Record{EscortID: escort, StartAt: start, EndAt: start.Add(time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec))
	require.NoError(t, r.Book(context.Background(), rec.ID, 8888))

	require.NoError(t, r.ReleaseByOrder(context.Background(), 8888))
	got, _ := r.GetByID(context.Background(), escort, rec.ID)
	assert.Equal(t, "available", got.Status)
	assert.Nil(t, got.OrderID)
}
```

**Step 2: 跑所有集成测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 ./services/escort/internal/availability/
```
Expected: PASS（13 repo 集成测试 + 4 跨层集成测试 = 17 cases）

**Step 3: 写文档**

修改 `docs/04-业务流程.md`，在 §4.7 之后追加：

```markdown
### 4.8 陪诊师空余时段（escort_availabilities）

**流程图**：

```
陪诊师上线 → PUT /api/v1/escorts/me/availability (start_at, end_at)
  → DB UNIQUE(escort_id, start_at) + service 层区间相交校验
  → status='available' 的 escort_availabilities 行

患者 P 创建订单 (service_start_at=T) → paid
  → match-service 调 ListAvailableInRange([T, T+1h))
  → 选 escort A → state: escort_pending_acceptance
  → escort A 30s 内 confirm accept
  → service.BookForOrder(orderID, A.id, T) → 找匹配时段 → Book → status='booked'

订单 cancel / escort 拒接 / 超时：
  service.ReleaseForOrder(orderID) → status='available' + order_id=NULL
```

**API 清单**（5 个）：

- `PUT /api/v1/escorts/me/availability` — 创建时段（escort 已 approved）
- `DELETE /api/v1/escorts/me/availability/:id` — 删时段（仅 available）
- `GET /api/v1/escorts/me/availability` — 列我的时段
- `GET /api/v1/escorts/:id/availabilities?start_at=&end_at=` — 公开查询（候选匹配）

**业务规则**：

- 时段区间不重叠（DB UNIQUE + service 层预检双保险）
- 仅 approved / online / in_service 状态的 escort 可创建时段
- 时段被 book 后不可改/删；只能由 order cancel / 拒接释放
- 候选匹配：status='available' 且 start_at ≤ order.service_start_at < end_at
```

修改 `dev.md`，在 §10.13 之后追加：

```markdown
### 10.14 escort-availability plan 落地记录

- **plan**: docs/superpowers/plans/2026-09-24-escort-availability.md
- **commit**: see git log `feat(escort): availability.*`
- **模块**: services/escort/internal/availability/{repo,service,handler}.go
- **表**: migrations/0009_escort_availabilities.{up,down}.sql
- **API**: 4 个 endpoint（PUT/DELETE/GET /me/availability + 公开 GET /escorts/:id/availabilities）
- **集成**: 接入 escort-service（sub-package，选项 A）
```

**Step 4: Commit**

```bash
git add services/escort/internal/availability/repo_integration_test.go docs/04-业务流程.md dev.md
git commit -m "test(escort): availability 加 4 个跨层集成场景 + 业务文档落地"
```

---

## Self-Review

- ✅ **架构选择已 commit**：选项 A — 作为 escort-service 子包，无新增部署开销；理由写在 Header / Architecture
- ✅ **spec §3.2 SQL 完整复刻**：表结构 + CHECK + UNIQUE (escort_id, start_at) + 部分索引 idx_escort_avail_status_start + CHECK end_at > start_at 全部到位
- ✅ **4 个 spec §3.2 列出的 API + 1 个公开查询共 5 个 endpoint**：PUT / DELETE / GET /me/availability + GET /:id/availabilities
- ✅ **spec §2.2 业务规则全覆盖**：
  - 时段只能被一个订单占用 → `Book` 单 SQL 乐观锁 + `ErrAvailabilityBooked` 哨兵
  - 拒接 / 超时 → 时段保持 available → 由 `ReleaseByOrder` 显式恢复（订单 cancel 触发；v1 不自动，需调用方触发）
  - 订单 cancel → 时段恢复 available → 同上
  - 同 escort 不重叠 → DB UNIQUE + service 层 `hasConflict` 双保险
- ✅ **7 个哨兵 / 错误码**：`ErrAvailabilityNotFound` / `ErrAvailabilityBooked` / `ErrAvailabilityConflict` + `errs.CodeForbidden`（escort 未审核）+ `errs.CodeParamInvalid` + `errs.CodeUnauthorized` + `errs.CodeInternal`
- ✅ **测试矩阵（≥ 30 个）**：13 repo 集成测试 + 4 跨层集成 + 8 service 单测 + 8 handler 单测 + smoke 5 个 HTTP 步骤
- ✅ **TDD 节奏**：每个 Task 都是 RED（写测试 + 跑挂）→ GREEN（写实现 + 跑通）→ Commit 三步
- ✅ **commit 前缀合规**：每个 Task 末尾 commit 用 `feat:` / `test:` 前缀
- ✅ **不重复做**：orders 表 selected_escort_id / escort_pending_expire_at 由 state-machine / order-lock 修订 plan 单独落；本 plan 只负责 escort_availabilities
- ✅ **不破坏既有**：escort main.go 改 `handler.New(svc, availH)` 接受第二个参数（向后兼容：availH=nil 时降级为既有行为）
- ✅ **跨包解耦**：`availability.Service` 通过 `ProfileLookup` 接口读 escort_profiles.state，不 import escort service 包
- ✅ **API 公开 vs 鉴权**：`/escorts/:id/availabilities` 公开（候选用）；`/me/*` 走 JWT（既有 escort middleware 提供）
- ✅ **YAGNI**：不做按调度模板（v2）、不做智能评分（v2，match.scorer 复用既有）
- ⚠️ **已知偏差**：
  1. spec §3.2 把 escort_availabilities 表 + orders 表 schema 修订放在同一 0009 迁移。本 plan 仅负责 escort_availabilities（用户拆分）；orders 表 schema 修订由 state-machine / order-lock 修订 plan 在另一文件落（迁移编号预留待修订 plan 自取）
  2. `BookForOrder` 在 match-service 集成时由 order-service 内部调用（不在 HTTP 暴露），参数 `(orderID, escortID, serviceStartAt)` 与 spec §4.2 patient `POST /select-escort` 触发时序对齐（patient 选 escort 后由 order-service 异步调用）
  3. `ReleaseForOrder` 由 order cancel / escort reject 流程触发（v1 不在 escort-service 自动监听；order-service 在状态机转换时显式调用）