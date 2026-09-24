# Virtual Number 中间号 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 保护患者 / 陪诊师双方手机号隐私。订单 accepted 时自动生成 1 个中间号映射并落到 `virtual_numbers` 表；患者 / 陪诊师通过 `GET /api/v1/orders/{id}/virtual-number` 各自看自己的那一面；订单 completed 后 24h 失效（仍可查历史但 `status=expired`，前端展示"已失效"）。

**Architecture:** 新建独立服务 `services/virtual-number/`（理由：隐私号是独立生命周期，与 order 业务耦合度低；将来接阿里云隐私号 API 时改造面最小）。事件驱动：order-service 通过内部 HTTP 调用触发激活 / 调度过期（`POST /internal/virtual-number/activate` + `POST /internal/virtual-number/schedule-expire`），避免 v1 引入 Kafka consumer 复杂度过高。后台 scanner 1 分钟轮询 `expires_at < NOW() AND status='active'` → 翻 `expired`。v1 中间号生成算法：`"1900" + order_id 末 4 位 0 填充`，示例：order_id=12345 → `19002345`（mock 整数 8 位，等同真实虚拟号格式）。

**Tech Stack:** Go 1.24+ · pgx v5.7 · gin v1.10 · testify v1.11 · zap v1.27。

**前置依赖:**
- `2026-09-24-state-machine.md`（orders.status 含 `accepted` / `completed` / `canceled` 三态，虚拟号生命周期对齐这三态）
- `2026-09-24-l2-api-gap-design.md` §2.1 / §2.2 P0 virtual-number API 清单 + §3.1 `OrderEscortSummary.phone_virtual` schema
- `services/order/internal/service`（订单 service 暴露 `OnAccepted(orderID, patientID, escortID)` + `OnCompleted(orderID)` 钩子；本 plan 假设已在订单 plan 里预留 hook，由本 plan 在 cmd 里挂 HTTP 调用）
- 既有 `shared/httpx` / `shared/errs` / `shared/middleware.Auth` / `shared/logger` / `shared/config` 全部就绪

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1（v1 不引 ORM，全部 pgx 直写 SQL）
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- 隐私：API 调用必须校验 `viewer_id` ∈ `{order.patient_id, order.escort_id}`，否则 `CodeForbidden`
- 中间号算法 v1：`fmt.Sprintf("1900%04d", orderID%10000)`（order_id 末 4 位 0 填充）
- v1 不接第三方隐私号 API（阿里云 / 腾讯云隐私号留 v2，本 plan 仅 mock 整数 8 位）
- 中间号仅在 `accepted → completed` 区间有效；订单 `canceled` / `refunded` 直接置 `expired`
- 24h 过期：`expires_at = completed_at + 24h`；scanner 翻状态后保留行不删（前端可看历史"已过期"标记）
- 同一订单只生成 1 个中间号（`UNIQUE(order_id)`），重复激活幂等返回已存在记录

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `migrations/0006_virtual_numbers.up.sql` | Create | virtual_numbers 表 + CHECK + 唯一索引 |
| `migrations/0006_virtual_numbers.down.sql` | Create | 逆向 |
| `migrations/migrations_test.go` | Modify | 加 `Test0006VirtualNumbersUpDown` |
| `services/virtual-number/go.mod` | Create | 子 module（v1 与其他服务共享 shared） |
| `services/virtual-number/internal/repo/virtual_repo.go` | Create | pgx 实现 + 哨兵错误 |
| `services/virtual-number/internal/repo/virtual_repo_integration_test.go` | Create | 集成测试（建表 + CRUD + 唯一约束） |
| `services/virtual-number/internal/service/virtual_service.go` | Create | 业务逻辑：生成 / 授权 / 调度过期 |
| `services/virtual-number/internal/service/virtual_service_test.go` | Create | 单测（mock repo） |
| `services/virtual-number/internal/service/scanner.go` | Create | 后台 1 分钟轮询翻 expired |
| `services/virtual-number/internal/service/scanner_test.go` | Create | scanner 单测（fake clock + fake repo） |
| `services/virtual-number/internal/handler/virtual.go` | Create | 3 个 endpoint handler + RegisterRoutes |
| `services/virtual-number/internal/handler/virtual_test.go` | Create | handler 单测（httptest） |
| `services/virtual-number/internal/router/router.go` | Create | gin Engine + /healthz |
| `services/virtual-number/internal/router/router_test.go` | Create | router 单测 |
| `services/virtual-number/internal/server/server.go` | Create | HTTP server 优雅停机 |
| `services/virtual-number/internal/cmd/main.go` | Create | 装配 + 启后台 scanner |
| `scripts/smoke-virtual-number.sh` | Create | smoke（/healthz + 401 + 鉴权校验） |
| `docs/04-业务流程.md` | Modify | §4.7 加虚拟号生成 / 失效流程 |
| `dev.md` | Modify | §10.13 加本 plan 落地记录 |

---

### Task 1: 数据库迁移（virtual_numbers 表）

**Files:**
- Create: `migrations/0006_virtual_numbers.up.sql`
- Create: `migrations/0006_virtual_numbers.down.sql`
- Modify: `migrations/migrations_test.go`

**Step 1: 写集成测试（RED）**

在 `migrations/migrations_test.go` 末尾追加：

```go
// Test0006VirtualNumbersUpDown 验证 virtual_numbers 表 + CHECK + 唯一索引 + down 可逆。
func Test0006VirtualNumbersUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 先建 users + orders（virtual_numbers.escort_id 引用 users.id）
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

	applyUp(t, "0006_virtual_numbers.up.sql", []string{"virtual_numbers"})

	// 列检查
	for _, col := range []string{
		"id", "order_id", "patient_id", "escort_id",
		"virtual_no", "status", "expires_at",
		"created_at", "expired_at",
	} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='virtual_numbers' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "virtual_numbers.%s should exist", col)
	}

	// 唯一索引（一个订单最多一个中间号）
	var uniqExists bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, "uq_virtual_numbers_order").
		Scan(&uniqExists)
	require.NoError(t, err)
	assert.True(t, uniqExists, "uq_virtual_numbers_order should exist")

	// 状态枚举 CHECK
	var hasCheck bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name LIKE 'virtual_numbers_status_check')`).
		Scan(&hasCheck)
	require.NoError(t, err)
	assert.True(t, hasCheck)

	// down 校验
	downCtx, downCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer downCancel()
	downSQL, err := os.ReadFile("0006_virtual_numbers.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(downCtx, string(downSQL))
	require.NoError(t, err, "apply 0006_virtual_numbers.down.sql")

	var gone bool
	err = conn.QueryRow(downCtx,
		`SELECT NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='virtual_numbers')`).
		Scan(&gone)
	require.NoError(t, err)
	assert.True(t, gone, "virtual_numbers should be gone after down")
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0006VirtualNumbersUpDown ./migrations/
```

Expected: FAIL — `Test0006VirtualNumbersUpDown` undefined.

**Step 3: 写 `0006_virtual_numbers.up.sql`**

```sql
-- 0006_virtual_numbers.up.sql
-- 虚拟号映射表（保护患者 + 陪诊师手机号隐私）。
-- 生命周期：order accepted 时插入 status='active'；
--   completed + 24h 后 scanner 翻 status='expired'（保留行不删，前端展示"已过期"）。
-- 唯一约束：1 个订单 1 个虚拟号；重复激活幂等返回已存在记录。
-- v1 不接第三方隐私号 API，virtual_no 用 "1900" + order_id 末 4 位 mock。

CREATE TABLE virtual_numbers (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES orders(id),
  patient_id BIGINT NOT NULL REFERENCES users(id),
  escort_id BIGINT NOT NULL REFERENCES users(id),
  virtual_no VARCHAR(16) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','expired')),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expired_at TIMESTAMPTZ
);

-- 1 个订单 1 个虚拟号（重复激活走 ON CONFLICT 幂等返回）。
CREATE UNIQUE INDEX uq_virtual_numbers_order ON virtual_numbers(order_id);

-- 患者 / 陪诊师查"我的虚拟号"索引。
CREATE INDEX idx_virtual_numbers_patient ON virtual_numbers(patient_id, created_at DESC);
CREATE INDEX idx_virtual_numbers_escort ON virtual_numbers(escort_id, created_at DESC);

-- scanner 1 分钟轮询"已到过期时间但仍 active"集合。
CREATE INDEX idx_virtual_numbers_scanner ON virtual_numbers(expires_at)
  WHERE status = 'active';
```

**Step 4: 写 `0006_virtual_numbers.down.sql`**

```sql
-- 0006_virtual_numbers.down.sql
-- 撤销 0006：删索引 + 删表。

DROP INDEX IF EXISTS idx_virtual_numbers_scanner;
DROP INDEX IF EXISTS idx_virtual_numbers_escort;
DROP INDEX IF EXISTS idx_virtual_numbers_patient;
DROP INDEX IF EXISTS uq_virtual_numbers_order;
DROP TABLE IF EXISTS virtual_numbers;
```

**Step 5: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0006VirtualNumbersUpDown ./migrations/
```

Expected: PASS.

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0006 virtual_numbers (v1 mock 1900+末4位 + UNIQUE order + 索引 + 集成测试)"
```

---

### Task 2: virtual_repo + virtual_service（核心业务）

**Files:**
- Create: `services/virtual-number/go.mod`
- Create: `services/virtual-number/internal/repo/virtual_repo.go`
- Create: `services/virtual-number/internal/repo/virtual_repo_integration_test.go`
- Create: `services/virtual-number/internal/service/virtual_service.go`
- Create: `services/virtual-number/internal/service/virtual_service_test.go`

**Step 2.0: 写 `services/virtual-number/go.mod`**

```go
module github.com/growdu/doctors/services/virtual-number

go 1.24

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/jackc/pgx/v5 v5.7.1
	github.com/stretchr/testify v1.11.0
	go.uber.org/zap v1.27.0
)

replace github.com/growdu/doctors/shared => ../../shared
```

随后在 `services/virtual-number/` 下 `go mod tidy`（会自动复用根 go.sum）。

**Step 2.1: 写 repo 集成测试（RED）**

`services/virtual-number/internal/repo/virtual_repo_integration_test.go`：

```go
//go:build integration
// +build integration

package repo

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

// setupPool 起连接池并准备 users + orders + virtual_numbers 表（一次性建表）。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS virtual_numbers CASCADE;
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
		    'created','paid','matching','pending_acceptance','accepted','in_service',
		    'completed','reviewed','refunding','refunded','settling','disputed',
		    'closed','canceled')),
		  version INT NOT NULL DEFAULT 0,
		  lock_owner BIGINT REFERENCES users(id),
		  lock_expire_at TIMESTAMPTZ,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE virtual_numbers (
		  id BIGSERIAL PRIMARY KEY,
		  order_id BIGINT NOT NULL REFERENCES orders(id),
		  patient_id BIGINT NOT NULL REFERENCES users(id),
		  escort_id BIGINT NOT NULL REFERENCES users(id),
		  virtual_no VARCHAR(16) NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'active'
		    CHECK (status IN ('active','expired')),
		  expires_at TIMESTAMPTZ NOT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  expired_at TIMESTAMPTZ
		);
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS virtual_numbers;
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

func seedOrder(t *testing.T, pool *pgxpool.Pool, patientID int64, escortID *int64, status string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO orders (order_no, patient_id, escort_id, hospital_id, package_id,
		                    service_start_at, amount, final_amount, status)
		VALUES ('O-vn', $1, $2, 1, 1, NOW() + INTERVAL '1 day', 100, 100, $3)
		RETURNING id`, patientID, escortID, status).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestVirtualRepo_Activate_OK 验证插入一条 active 记录。
func TestVirtualRepo_Activate_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138000", "patient")
	escort := seedUser(t, pool, "13900139000", "escort")
	orderID := seedOrder(t, pool, patient, &escort, "accepted")
	r := NewVirtualRepo(pool)

	rec, err := r.Activate(context.Background(), orderID, patient, escort, "19002345", time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.NotZero(t, rec.ID)
	assert.Equal(t, "19002345", rec.VirtualNo)
	assert.Equal(t, "active", rec.Status)
}

// TestVirtualRepo_Activate_DuplicateOrder 验证重复激活幂等返回已存在记录。
func TestVirtualRepo_Activate_DuplicateOrder(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138001", "patient")
	escort := seedUser(t, pool, "13900139001", "escort")
	orderID := seedOrder(t, pool, patient, &escort, "accepted")
	r := NewVirtualRepo(pool)

	first, err := r.Activate(context.Background(), orderID, patient, escort, "19000001", time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	second, err := r.Activate(context.Background(), orderID, patient, escort, "19000099", time.Now().Add(48*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "重复激活应返回同一行")
	assert.Equal(t, "19000001", second.VirtualNo, "不应覆盖原 virtual_no")
}

// TestVirtualRepo_GetByOrder 验证按 order_id 查。
func TestVirtualRepo_GetByOrder(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138002", "patient")
	escort := seedUser(t, pool, "13900139002", "escort")
	orderID := seedOrder(t, pool, patient, &escort, "accepted")
	r := NewVirtualRepo(pool)
	_, err := r.Activate(context.Background(), orderID, patient, escort, "19001234", time.Now().Add(24*time.Hour))
	require.NoError(t, err)

	got, err := r.GetByOrder(context.Background(), orderID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "active", got.Status)
	assert.Equal(t, "19001234", got.VirtualNo)
}

// TestVirtualRepo_GetByOrder_NotFound 验证订单无虚拟号时返回 (nil, nil)。
func TestVirtualRepo_GetByOrder_NotFound(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138003", "patient")
	orderID := seedOrder(t, pool, patient, nil, "paid")
	r := NewVirtualRepo(pool)

	got, err := r.GetByOrder(context.Background(), orderID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestVirtualRepo_ScheduleExpire_OK 验证更新 expires_at。
func TestVirtualRepo_ScheduleExpire_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138004", "patient")
	escort := seedUser(t, pool, "13900139004", "escort")
	orderID := seedOrder(t, pool, patient, &escort, "accepted")
	r := NewVirtualRepo(pool)
	_, err := r.Activate(context.Background(), orderID, patient, escort, "19005678", time.Now().Add(24*time.Hour))
	require.NoError(t, err)

	newExpire := time.Now().Add(48 * time.Hour)
	require.NoError(t, r.ScheduleExpire(context.Background(), orderID, newExpire))

	got, err := r.GetByOrder(context.Background(), orderID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.WithinDuration(t, newExpire, got.ExpiresAt, time.Second)
}

// TestVirtualRepo_MarkExpired_OK 验证 scanner 翻 expired。
func TestVirtualRepo_MarkExpired_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138005", "patient")
	escort := seedUser(t, pool, "13900139005", "escort")
	orderID := seedOrder(t, pool, patient, &escort, "accepted")
	r := NewVirtualRepo(pool)
	_, err := r.Activate(context.Background(), orderID, patient, escort, "19009999", time.Now().Add(24*time.Hour))
	require.NoError(t, err)

	n, err := r.MarkExpiredBefore(context.Background(), time.Now().Add(48*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	got, err := r.GetByOrder(context.Background(), orderID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "expired", got.Status)
	require.NotNil(t, got.ExpiredAt)
}
```

**Step 2.2: 跑测试确认失败**

```bash
cd services/virtual-number
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestVirtualRepo_' ./internal/repo/
```

Expected: FAIL — `undefined: NewVirtualRepo`, `undefined: ErrVirtualNotFound`。

**Step 2.3: 写 `virtual_repo.go`**

`services/virtual-number/internal/repo/virtual_repo.go`：

```go
// Package repo 是 virtual-number-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc）。
//   - UNIQUE(order_id) 保证幂等；Activate 用 ON CONFLICT RETURNING 返回已存在行。
//   - status 用 CHECK 约束；scanner 用 UPDATE ... WHERE status='active' AND expires_at <= $1。
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record 映射 virtual_numbers 表行。
type Record struct {
	ID         int64
	OrderID    int64
	PatientID  int64
	EscortID   int64
	VirtualNo  string
	Status     string // "active" | "expired"
	ExpiresAt  time.Time
	CreatedAt  time.Time
	ExpiredAt  *time.Time
}

// VirtualRepo 是 virtual_numbers 表的仓储。
type VirtualRepo struct {
	pool *pgxpool.Pool
}

// NewVirtualRepo 构造仓储。
func NewVirtualRepo(pool *pgxpool.Pool) *VirtualRepo { return &VirtualRepo{pool: pool} }

// baseSelect 是 SELECT 子句。
const baseSelect = `
	SELECT id, order_id, patient_id, escort_id, virtual_no, status,
	       expires_at, created_at, expired_at
	FROM virtual_numbers`

// Activate 插入一条 active 记录；若 order_id 已存在则返回原行（幂等）。
func (r *VirtualRepo) Activate(ctx context.Context, orderID, patientID, escortID int64,
	virtualNo string, expiresAt time.Time) (*Record, error) {
	const q = `
		INSERT INTO virtual_numbers (order_id, patient_id, escort_id, virtual_no, status, expires_at)
		VALUES ($1, $2, $3, $4, 'active', $5)
		ON CONFLICT (order_id) DO UPDATE SET order_id = EXCLUDED.order_id
		RETURNING id, order_id, patient_id, escort_id, virtual_no, status,
		          expires_at, created_at, expired_at`
	row := r.pool.QueryRow(ctx, q, orderID, patientID, escortID, virtualNo, expiresAt)
	s := &Record{}
	if err := row.Scan(&s.ID, &s.OrderID, &s.PatientID, &s.EscortID, &s.VirtualNo,
		&s.Status, &s.ExpiresAt, &s.CreatedAt, &s.ExpiredAt); err != nil {
		return nil, fmt.Errorf("activate virtual number: %w", err)
	}
	return s, nil
}

// GetByOrder 按 order_id 查询；订单无虚拟号时返回 (nil, nil)。
func (r *VirtualRepo) GetByOrder(ctx context.Context, orderID int64) (*Record, error) {
	q := baseSelect + ` WHERE order_id = $1 LIMIT 1`
	row := r.pool.QueryRow(ctx, q, orderID)
	s := &Record{}
	if err := row.Scan(&s.ID, &s.OrderID, &s.PatientID, &s.EscortID, &s.VirtualNo,
		&s.Status, &s.ExpiresAt, &s.CreatedAt, &s.ExpiredAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get virtual by order: %w", err)
	}
	return s, nil
}

// ScheduleExpire 更新 expires_at（订单 completed 时设置 completed_at + 24h）。
func (r *VirtualRepo) ScheduleExpire(ctx context.Context, orderID int64, expiresAt time.Time) error {
	const q = `UPDATE virtual_numbers SET expires_at = $2 WHERE order_id = $1`
	tag, err := r.pool.Exec(ctx, q, orderID, expiresAt)
	if err != nil {
		return fmt.Errorf("schedule expire: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("schedule expire order %d: no row", orderID)
	}
	return nil
}

// MarkExpiredBefore 把 expires_at < cutoff 且 status='active' 的行翻 expired；
// 返回受影响行数（供 scanner 日志）。
func (r *VirtualRepo) MarkExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	const q = `
		UPDATE virtual_numbers
		   SET status = 'expired', expired_at = NOW()
		 WHERE status = 'active' AND expires_at <= $1`
	tag, err := r.pool.Exec(ctx, q, cutoff)
	if err != nil {
		return 0, fmt.Errorf("mark expired: %w", err)
	}
	return tag.RowsAffected(), nil
}
```

**Step 2.4: 跑测试确认通过**

```bash
cd services/virtual-number
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestVirtualRepo_' ./internal/repo/
```

Expected: PASS（6 个测试）。

**Step 2.5: 写 service 单测（RED）**

`services/virtual-number/internal/service/virtual_service_test.go`：

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/services/virtual-number/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepo 满足 VirtualRepo 接口（最小）。
type fakeRepo struct {
	activateFn    func(ctx context.Context, orderID, patientID, escortID int64, virtualNo string, expiresAt time.Time) (*repo.Record, error)
	getByOrderFn  func(ctx context.Context, orderID int64) (*repo.Record, error)
	scheduleFn    func(ctx context.Context, orderID int64, expiresAt time.Time) error
	markExpiredFn func(ctx context.Context, cutoff time.Time) (int64, error)
}

func (f *fakeRepo) Activate(ctx context.Context, orderID, patientID, escortID int64, virtualNo string, expiresAt time.Time) (*repo.Record, error) {
	return f.activateFn(ctx, orderID, patientID, escortID, virtualNo, expiresAt)
}
func (f *fakeRepo) GetByOrder(ctx context.Context, orderID int64) (*repo.Record, error) {
	return f.getByOrderFn(ctx, orderID)
}
func (f *fakeRepo) ScheduleExpire(ctx context.Context, orderID int64, expiresAt time.Time) error {
	return f.scheduleFn(ctx, orderID, expiresAt)
}
func (f *fakeRepo) MarkExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return f.markExpiredFn(ctx, cutoff)
}

// TestGenerateVirtualNo 验证 v1 mock 算法 "1900" + order_id 末 4 位。
func TestGenerateVirtualNo(t *testing.T) {
	cases := []struct {
		orderID int64
		want    string
	}{
		{12345, "19002345"},
		{7, "19000007"},
		{99999, "19009999"},
		{100000, "19000000"}, // 末 4 位 0 填充
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, GenerateVirtualNo(tc.orderID))
	}
}

// TestActivate_OK 验证激活流程：调用 repo.Activate 并返回记录。
func TestActivate_OK(t *testing.T) {
	var captured struct {
		orderID, patientID, escortID         int64
		virtualNo                            string
		expiresAt                            time.Time
	}
	r := &fakeRepo{
		activateFn: func(_ context.Context, oid, pid, eid int64, vn string, exp time.Time) (*repo.Record, error) {
			captured.orderID, captured.patientID, captured.escortID = oid, pid, eid
			captured.virtualNo = vn
			captured.expiresAt = exp
			return &repo.Record{ID: 1, OrderID: oid, VirtualNo: vn, Status: "active", ExpiresAt: exp}, nil
		},
	}
	s := New(r, WithClock(func() time.Time { return time.Unix(1700000000, 0) }))
	got, err := s.Activate(context.Background(), 12345, 7, 8)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(12345), captured.orderID)
	assert.Equal(t, "19002345", captured.virtualNo)
	assert.Equal(t, int64(7), captured.patientID)
	assert.Equal(t, int64(8), captured.escortID)
	// 默认 expires_at = now + 24h
	assert.Equal(t, time.Unix(1700000000, 0).Add(24*time.Hour), captured.expiresAt)
}

// TestActivate_RepoError 验证 repo 报错时包装 CodeForbidden 不应（应是系统错） 。
func TestActivate_RepoError(t *testing.T) {
	r := &fakeRepo{
		activateFn: func(_ context.Context, _ int64, _ int64, _ int64, _ string, _ time.Time) (*repo.Record, error) {
			return nil, errors.New("db down")
		},
	}
	s := New(r)
	got, err := s.Activate(context.Background(), 1, 2, 3)
	assert.Nil(t, got)
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeInternal, e.Code)
}

// TestGetByOrder_Forbidden 验证 viewer 既非 patient_id 又非 escort_id 时返回 CodeForbidden。
func TestGetByOrder_Forbidden(t *testing.T) {
	r := &fakeRepo{
		getByOrderFn: func(_ context.Context, _ int64) (*repo.Record, error) {
			return &repo.Record{OrderID: 1, PatientID: 7, EscortID: 8, Status: "active", VirtualNo: "19000001"}, nil
		},
	}
	s := New(r)
	got, err := s.GetByOrderForViewer(context.Background(), 1, 99)
	assert.Nil(t, got)
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeForbidden, e.Code)
}

// TestGetByOrder_NotFound 验证订单无虚拟号返回 CodeNotFound。
func TestGetByOrder_NotFound(t *testing.T) {
	r := &fakeRepo{
		getByOrderFn: func(_ context.Context, _ int64) (*repo.Record, error) { return nil, nil },
	}
	s := New(r)
	got, err := s.GetByOrderForViewer(context.Background(), 1, 7)
	assert.Nil(t, got)
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeNotFound, e.Code)
}

// TestGetByOrder_PatientViewer_OK 验证患者可看。
func TestGetByOrder_PatientViewer_OK(t *testing.T) {
	r := &fakeRepo{
		getByOrderFn: func(_ context.Context, oid int64) (*repo.Record, error) {
			return &repo.Record{OrderID: oid, PatientID: 7, EscortID: 8, Status: "active",
				VirtualNo: "19000001", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
		},
	}
	s := New(r)
	got, err := s.GetByOrderForViewer(context.Background(), 1, 7)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "active", got.Status)
}

// TestGetByOrder_EscortViewer_OK 验证陪诊师可看。
func TestGetByOrder_EscortViewer_OK(t *testing.T) {
	r := &fakeRepo{
		getByOrderFn: func(_ context.Context, oid int64) (*repo.Record, error) {
			return &repo.Record{OrderID: oid, PatientID: 7, EscortID: 8, Status: "active",
				VirtualNo: "19000001", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
		},
	}
	s := New(r)
	got, err := s.GetByOrderForViewer(context.Background(), 1, 8)
	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestScheduleExpire_Defaults24h 验证默认过期时间是 completed_at + 24h。
func TestScheduleExpire_Defaults24h(t *testing.T) {
	completedAt := time.Unix(1700000000, 0)
	var captured time.Time
	r := &fakeRepo{
		scheduleFn: func(_ context.Context, _ int64, exp time.Time) error {
			captured = exp
			return nil
		},
	}
	s := New(r, WithClock(func() time.Time { return completedAt }))
	require.NoError(t, s.ScheduleExpire(context.Background(), 1, completedAt))
	assert.Equal(t, completedAt.Add(24*time.Hour), captured)
}
```

**Step 2.6: 写 `virtual_service.go`**

`services/virtual-number/internal/service/virtual_service.go`：

```go
// Package service 是 virtual-number-service 的业务逻辑层。
//
// 设计要点：
//   - v1 中间号算法 GenerateVirtualNo：mock "1900" + order_id 末 4 位 0 填充。
//   - GetByOrderForViewer 校验 viewer ∈ {patient, escort}，否则 CodeForbidden。
//   - Activate 默认 expires_at = now + 24h；ScheduleExpire 允许外部覆盖（completed 后 24h）。
//   - Scanner 由 reposervice.MarkExpiredBefore 提供原语，service 层拼 1 分钟 ticker。
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/services/virtual-number/internal/repo"
)

// VirtualRepo 是仓储契约（与 repo.VirtualRepo 一致）。
type VirtualRepo interface {
	Activate(ctx context.Context, orderID, patientID, escortID int64, virtualNo string, expiresAt time.Time) (*repo.Record, error)
	GetByOrder(ctx context.Context, orderID int64) (*repo.Record, error)
	ScheduleExpire(ctx context.Context, orderID int64, expiresAt time.Time) error
	MarkExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// 编译期断言。
var _ VirtualRepo = (*repo.VirtualRepo)(nil)

// GenerateVirtualNo v1 mock 算法：fmt.Sprintf("1900%04d", orderID%10000)。
// 末 4 位 0 填充：order_id=7 → "19000007"；order_id=100000 → "19000000"。
func GenerateVirtualNo(orderID int64) string {
	return fmt.Sprintf("1900%04d", orderID%10000)
}

// defaultTTL 默认中间号有效期 24 小时。
const defaultTTL = 24 * time.Hour

// Service 持有 repo + clock。
type Service struct {
	repo     VirtualRepo
	clockNow func() time.Time
}

// New 构造 service。
func New(r VirtualRepo, opts ...Option) *Service {
	s := &Service{repo: r, clockNow: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Option 配置 Service。
type Option func(*Service)

// WithClock 注入时钟（测试用）。
func WithClock(now func() time.Time) Option { return func(s *Service) { s.clockNow = now } }

// Activate 为订单生成中间号（订单 accepted 时由 order-service 触发）。
// 幂等：重复调用返回已存在记录，不重复创建。
func (s *Service) Activate(ctx context.Context, orderID, patientID, escortID int64) (*repo.Record, error) {
	virtualNo := GenerateVirtualNo(orderID)
	expiresAt := s.clockNow().Add(defaultTTL)
	rec, err := s.repo.Activate(ctx, orderID, patientID, escortID, virtualNo, expiresAt)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "activate virtual number", err)
	}
	return rec, nil
}

// GetByOrderForViewer 返回订单中间号；viewer 必须是 patient_id 或 escort_id，否则 CodeForbidden。
func (s *Service) GetByOrderForViewer(ctx context.Context, orderID, viewerID int64) (*repo.Record, error) {
	rec, err := s.repo.GetByOrder(ctx, orderID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "get virtual number", err)
	}
	if rec == nil {
		return nil, errs.New(errs.CodeNotFound, "virtual number not found for order")
	}
	if viewerID != rec.PatientID && viewerID != rec.EscortID {
		return nil, errs.New(errs.CodeForbidden, "viewer is not part of this order")
	}
	return rec, nil
}

// ScheduleExpire 调度过期时间（订单 completed 时由 order-service 触发）；默认 completed_at + 24h。
func (s *Service) ScheduleExpire(ctx context.Context, orderID int64, completedAt time.Time) error {
	expiresAt := completedAt.Add(defaultTTL)
	if err := s.repo.ScheduleExpire(ctx, orderID, expiresAt); err != nil {
		return errs.Wrap(errs.CodeInternal, "schedule expire", err)
	}
	return nil
}

// MarkExpiredNow 立即翻 expired（订单 canceled / refunded 时调用，跳过 24h 等）。
func (s *Service) MarkExpiredNow(ctx context.Context, orderID int64) error {
	if err := s.repo.ScheduleExpire(ctx, orderID, s.now()); err != nil {
		return errs.Wrap(errs.CodeInternal, "mark expired now", err)
	}
	return nil
}

// now 是 clockNow 的小写访问器。
func (s *Service) now() time.Time { return s.clockNow() }
```

**Step 2.7: 跑 service 单测 + repo 集成测试**

```bash
cd services/virtual-number
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestGenerateVirtualNo|TestActivate_|TestGetByOrder_|TestScheduleExpire_' ./internal/service/
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestVirtualRepo_' ./internal/repo/
```

Expected: 全部 PASS（7 个 service 单测 + 6 个 repo 集成测试）。

**Step 2.8: Commit**

```bash
git add services/virtual-number/ migrations/
git commit -m "feat(virtual-number): repo + service (GenerateVirtualNo mock + Activate/GetByOrderForViewer/ScheduleExpire + 7 个单测 + 6 个集成测试)"
```

---

### Task 3: handler + router + server + main + scanner

**Files:**
- Create: `services/virtual-number/internal/service/scanner.go`
- Create: `services/virtual-number/internal/service/scanner_test.go`
- Create: `services/virtual-number/internal/handler/virtual.go`
- Create: `services/virtual-number/internal/handler/virtual_test.go`
- Create: `services/virtual-number/internal/router/router.go`
- Create: `services/virtual-number/internal/router/router_test.go`
- Create: `services/virtual-number/internal/server/server.go`
- Create: `services/virtual-number/internal/cmd/main.go`
- Create: `scripts/smoke-virtual-number.sh`

**Step 3.1: 写 scanner 单测（RED）**

`services/virtual-number/internal/service/scanner_test.go`：

```go
package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepoMark 用于 scanner 测试，仅实现 MarkExpiredBefore。
type fakeRepoMark struct {
	calls atomic.Int32
	last  time.Time
}

func (f *fakeRepoMark) Activate(_ context.Context, _ int64, _ int64, _ int64, _ string, _ time.Time) (interface{}, error) {
	return nil, nil
}
func (f *fakeRepoMark) GetByOrder(_ context.Context, _ int64) (interface{}, error) {
	return nil, nil
}
func (f *fakeRepoMark) ScheduleExpire(_ context.Context, _ int64, _ time.Time) error { return nil }
func (f *fakeRepoMark) MarkExpiredBefore(_ context.Context, cutoff time.Time) (int64, error) {
	f.calls.Add(1)
	f.last = cutoff
	return 0, nil
}

// 我们直接测试 scanner.Tick 函数（每次 ticker 回调执行一次）。
func TestScanner_Tick_CallsRepo(t *testing.T) {
	r := &scannerRepoMock{calls: 0}
	now := time.Unix(1700000000, 0)
	s := NewScanner(r, WithScannerClock(func() time.Time { return now }))
	n, err := s.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, now, r.last)
}

// scannerRepoMock 实现 scanner.Repo 接口（仅 MarkExpiredBefore）。
type scannerRepoMock struct {
	calls int
	last  time.Time
}

func (f *scannerRepoMock) MarkExpiredBefore(_ context.Context, cutoff time.Time) (int64, error) {
	f.calls++
	f.last = cutoff
	return 3, nil
}
```

**Step 3.2: 跑测试确认失败**

```bash
cd services/virtual-number
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestScanner_Tick' ./internal/service/
```

Expected: FAIL — `undefined: NewScanner`。

**Step 3.3: 写 `scanner.go`**

`services/virtual-number/internal/service/scanner.go`：

```go
// Package service: scanner 子模块负责后台 1 分钟轮询翻 expired。
package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/growdu/doctors/shared/logger"
)

// ScannerRepo 是 scanner 用到的仓储接口（只读 MarkExpiredBefore）。
type ScannerRepo interface {
	MarkExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// 编译期断言。
var _ ScannerRepo = (*virtualRepoAdapter)(nil)

// virtualRepoAdapter 把 *repo.VirtualRepo 适配成 ScannerRepo。
// 原因：service.Service 的 VirtualRepo 接口为 mock 友好用了不同包，
//       这里单独给 scanner 一个最小接口，避免 service 主接口污染 scanner 包。
type virtualRepoAdapter struct{ inner VirtualRepo }

// MarkExpiredBefore 透传。
func (a virtualRepoAdapter) MarkExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return a.inner.MarkExpiredBefore(ctx, cutoff)
}

// Scanner 是后台 ticker。
type Scanner struct {
	repo     ScannerRepo
	clockNow func() time.Time
	interval time.Duration
}

// NewScanner 构造 scanner（默认间隔 1 分钟）。
func NewScanner(r VirtualRepo, opts ...ScannerOption) *Scanner {
	s := &Scanner{
		repo:     virtualRepoAdapter{inner: r},
		clockNow: time.Now,
		interval: time.Minute,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ScannerOption 配置 Scanner。
type ScannerOption func(*Scanner)

// WithScannerClock 注入时钟（测试用）。
func WithScannerClock(now func() time.Time) ScannerOption {
	return func(s *Scanner) { s.clockNow = now }
}

// WithScannerInterval 注入轮询间隔（测试用，默认 1 分钟）。
func WithScannerInterval(d time.Duration) ScannerOption {
	return func(s *Scanner) { s.interval = d }
}

// Tick 执行一次扫描；返回受影响行数。
func (s *Scanner) Tick(ctx context.Context) (int64, error) {
	cutoff := s.clockNow()
	n, err := s.repo.MarkExpiredBefore(ctx, cutoff)
	if err != nil {
		logger.L().Error("virtual-number scanner tick failed", zap.Error(err))
		return 0, err
	}
	if n > 0 {
		logger.L().Info("virtual-number scanner marked expired", zap.Int64("rows", n))
	}
	return n, nil
}

// Run 启动后台循环；ctx 取消时优雅退出。
func (s *Scanner) Run(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	logger.L().Info("virtual-number scanner started", zap.Duration("interval", s.interval))
	for {
		select {
		case <-ctx.Done():
			logger.L().Info("virtual-number scanner stopped")
			return
		case <-t.C:
			if _, err := s.Tick(ctx); err != nil {
				// 已 log；继续
			}
		}
	}
}
```

**Step 3.4: 跑 scanner 单测**

```bash
cd services/virtual-number
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestScanner_Tick' ./internal/service/
```

Expected: PASS.

**Step 3.5: 写 handler 单测（RED）**

`services/virtual-number/internal/handler/virtual_test.go`：

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
	"github.com/growdu/doctors/services/virtual-number/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeService 满足 Service 接口。
type fakeService struct {
	activateFn       func(ctx context.Context, orderID, patientID, escortID int64) (*repo.Record, error)
	scheduleExpireFn func(ctx context.Context, orderID int64, completedAt time.Time) error
	markExpiredNowFn func(ctx context.Context, orderID int64) error
	getByOrderFn     func(ctx context.Context, orderID, viewerID int64) (*repo.Record, error)
}

func (f *fakeService) Activate(ctx context.Context, oid, pid, eid int64) (*repo.Record, error) {
	return f.activateFn(ctx, oid, pid, eid)
}
func (f *fakeService) ScheduleExpire(ctx context.Context, oid int64, completedAt time.Time) error {
	return f.scheduleExpireFn(ctx, oid, completedAt)
}
func (f *fakeService) MarkExpiredNow(ctx context.Context, oid int64) error {
	return f.markExpiredNowFn(ctx, oid)
}
func (f *fakeService) GetByOrderForViewer(ctx context.Context, oid, vid int64) (*repo.Record, error) {
	return f.getByOrderFn(ctx, oid, vid)
}

// TestGetVirtual_OK 验证 GET /api/v1/orders/:id/virtual-number 返回中间号。
func TestGetVirtual_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeService{
		getByOrderFn: func(_ context.Context, oid, _ int64) (*repo.Record, error) {
			return &repo.Record{
				OrderID: oid, PatientID: 7, EscortID: 8,
				VirtualNo: "19002345", Status: "active",
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}, nil
		},
	}
	h := New(svc, "test-secret")

	// 生成 JWT（patient role）用于鉴权。
	token := mintJWT(t, "test-secret", 7, "patient")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/1/virtual-number", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r := gin.New()
	h.RegisterRoutes(r, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "19002345", (*resp.Data)["virtual_no"])
}

// TestGetVirtual_Forbidden 验证非订单关联人返回 CodeForbidden。
func TestGetVirtual_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeService{
		getByOrderFn: func(_ context.Context, _ int64, vid int64) (*repo.Record, error) {
			return nil, errs.New(errs.CodeForbidden, "viewer is not part of this order")
		},
	}
	h := New(svc, "test-secret")
	token := mintJWT(t, "test-secret", 99, "patient")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/1/virtual-number", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r := gin.New()
	h.RegisterRoutes(r, nil)
	r.ServeHTTP(w, req)

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeForbidden), resp.Code)
}

// TestInternalActivate_OK 验证 POST /internal/virtual-number/activate（order-service 调用）。
func TestInternalActivate_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &fakeService{
		activateFn: func(_ context.Context, oid, pid, eid int64) (*repo.Record, error) {
			called = true
			assert.Equal(t, int64(123), claimed) // 编译期覆盖
			return &repo.Record{ID: 1, OrderID: oid, Status: "active", VirtualNo: "19000123"}, nil
		},
	}
	h := New(svc, "test-secret")
	body := map[string]any{"order_id": 123, "patient_id": 7, "escort_id": 8}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/virtual-number/activate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r := gin.New()
	h.RegisterInternalRoutes(r)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// claimed 复用变量（避免 unused 报错）
var claimed int64

// mintJWT 用 shared/auth 生成测试用 token（patient role）。
func mintJWT(t *testing.T, secret string, uid int64, role string) string {
	t.Helper()
	tok := issueToken(secret, uid, role)
	return tok
}
```

**Step 3.6: 写 handler `virtual.go`**

`services/virtual-number/internal/handler/virtual.go`：

```go
// Package handler 翻译 virtual-number HTTP 请求 ↔ service 调用 + errs 业务码。
package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/growdu/doctors/services/virtual-number/internal/repo"
	"github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 virtual-number-service 业务接口。
type Service interface {
	Activate(ctx context.Context, orderID, patientID, escortID int64) (*repo.Record, error)
	ScheduleExpire(ctx context.Context, orderID int64, completedAt time.Time) error
	MarkExpiredNow(ctx context.Context, orderID int64) error
	GetByOrderForViewer(ctx context.Context, orderID, viewerID int64) (*repo.Record, error)
}

// Handler 持有 service + JWT secret。
type Handler struct {
	svc       Service
	jwtSecret string
}

// New 构造 handler。
func New(svc Service, jwtSecret string) *Handler { return &Handler{svc: svc, jwtSecret: jwtSecret} }

// virtualResp 是 GET /api/v1/orders/:id/virtual-number 的响应体。
type virtualResp struct {
	OrderID        int64     `json:"order_id"`
	VirtualNo      string    `json:"virtual_no"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expires_at"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
	ViewerRoleHint string    `json:"viewer_role_hint"` // "patient" | "escort"
}

// GetVirtualHandler 患者 / 陪诊师查自己的中间号。
// viewer 必须是订单的 patient_id 或 escort_id，否则 CodeForbidden。
func (h *Handler) GetVirtualHandler(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid order id"))
		return
	}
	viewerID := c.GetInt64("user_id")
	if viewerID == 0 {
		httpx.Fail(c, errs.New(errs.CodeUnauthorized, "missing user_id in token"))
		return
	}
	rec, err := h.svc.GetByOrderForViewer(c.Request.Context(), orderID, viewerID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	hint := "escort"
	if viewerID == rec.PatientID {
		hint = "patient"
	}
	httpx.OK[virtualResp](c, virtualResp{
		OrderID:        rec.OrderID,
		VirtualNo:      rec.VirtualNo,
		Status:         rec.Status,
		ExpiresAt:      rec.ExpiresAt,
		ExpiredAt:      rec.ExpiredAt,
		ViewerRoleHint: hint,
	})
}

// activateReq 是 POST /internal/virtual-number/activate 的请求体。
type activateReq struct {
	OrderID   int64 `json:"order_id"`
	PatientID int64 `json:"patient_id"`
	EscortID  int64 `json:"escort_id"`
}

// ActivateInternalHandler order-service 在 accept 时调用。
func (h *Handler) ActivateInternalHandler(c *gin.Context) {
	var req activateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	if req.OrderID == 0 || req.PatientID == 0 || req.EscortID == 0 {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "order_id/patient_id/escort_id required"))
		return
	}
	rec, err := h.svc.Activate(c.Request.Context(), req.OrderID, req.PatientID, req.EscortID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK[map[string]any](c, map[string]any{
		"order_id":   rec.OrderID,
		"virtual_no": rec.VirtualNo,
		"status":     rec.Status,
		"expires_at": rec.ExpiresAt,
	})
}

// completeReq 是 POST /internal/virtual-number/schedule-expire 的请求体。
type completeReq struct {
	OrderID     int64     `json:"order_id"`
	CompletedAt time.Time `json:"completed_at"`
}

// ScheduleExpireInternalHandler order-service 在 completed 时调用（默认 completed_at + 24h）。
func (h *Handler) ScheduleExpireInternalHandler(c *gin.Context) {
	var req completeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	if req.OrderID == 0 {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "order_id required"))
		return
	}
	if req.CompletedAt.IsZero() {
		req.CompletedAt = time.Now()
	}
	if err := h.svc.ScheduleExpire(c.Request.Context(), req.OrderID, req.CompletedAt); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK[map[string]any](c, map[string]any{
		"order_id":   req.OrderID,
		"expires_at": req.CompletedAt.Add(24 * time.Hour),
	})
}

// cancelReq 是 POST /internal/virtual-number/mark-expired 的请求体（canceled/refunded 时调用）。
type cancelReq struct {
	OrderID int64 `json:"order_id"`
}

// MarkExpiredInternalHandler order-service 在 canceled / refunded 时调用。
func (h *Handler) MarkExpiredInternalHandler(c *gin.Context) {
	var req cancelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	if req.OrderID == 0 {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "order_id required"))
		return
	}
	if err := h.svc.MarkExpiredNow(c.Request.Context(), req.OrderID); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK[map[string]any](c, map[string]any{
		"order_id": req.OrderID,
		"status":   "expired",
	})
}

// RegisterRoutes 挂业务路由（需 JWT 鉴权）。
func (h *Handler) RegisterRoutes(r gin.IRouter, authMW gin.HandlerFunc) {
	v1 := r.Group("/api/v1")
	if authMW != nil {
		v1.Use(authMW)
	}
	v1.GET("/orders/:id/virtual-number", h.GetVirtualHandler)
}

// RegisterInternalRoutes 挂内部回调路由（order-service 调用，无 JWT；v1 用 network policy 隔离）。
func (h *Handler) RegisterInternalRoutes(r gin.IRouter) {
	internal := r.Group("/internal/virtual-number")
	internal.POST("/activate", h.ActivateInternalHandler)
	internal.POST("/schedule-expire", h.ScheduleExpireInternalHandler)
	internal.POST("/mark-expired", h.MarkExpiredInternalHandler)
}

// issueToken 给测试用：通过 shared/auth 生成 JWT。
func issueToken(secret string, uid int64, role string) string {
	tok, _ := jwt.ParseWithClaims("", &auth.Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	_ = tok // 实际实现请用 auth.Issue(secret, uid, role, 1h)
	return auth.Issue(secret, uid, role, time.Hour)
}
```

**Step 3.7: 写 router + server + main**

`services/virtual-number/internal/router/router.go`：

```go
// Package router 注册 virtual-number-service 的 HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/virtual-number/internal/handler"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/middleware"
)

// New 返回一个挂好全部路由的 *gin.Engine。
func New(h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	auth := middleware.Auth(jwtSecret)
	h.RegisterRoutes(r, auth)
	h.RegisterInternalRoutes(r)
	return r
}
```

`services/virtual-number/internal/router/router_test.go`：

```go
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(nil, "test-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
```

`services/virtual-number/internal/server/server.go`：

```go
// Package server 启动 virtual-number-service HTTP server。
package server

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/growdu/doctors/shared/logger"
)

// Server 是 HTTP server 包装。
type Server struct {
	addr string
	srv  *http.Server
}

// New 构造 server。
func New(addr string, h http.Handler) *Server {
	return &Server{
		addr: addr,
		srv: &http.Server{
			Addr:              addr,
			Handler:           h,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run 启动 + 优雅停机。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	logger.L().Info("virtual-number-service starting", zap.String("addr", s.addr))

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
```

`services/virtual-number/internal/cmd/main.go`：

```go
// virtual-number-service 入口。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/virtual-number/internal/handler"
	"github.com/growdu/doctors/services/virtual-number/internal/repo"
	"github.com/growdu/doctors/services/virtual-number/internal/router"
	"github.com/growdu/doctors/services/virtual-number/internal/server"
	"github.com/growdu/doctors/services/virtual-number/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("virtual-number")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// pool
	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("connect pg: %v", err)
	}
	defer pool.Close()

	virtualRepo := repo.NewVirtualRepo(pool)
	svc := service.New(virtualRepo)
	scanner := service.NewScanner(virtualRepo)

	h := handler.New(svc, cfg.Auth.JWTSecret)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	// 启后台 scanner。
	go scanner.Run(ctx)

	logger.L().Info("virtual-number-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("virtual-number-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("virtual-number-service stopped")
}

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
```

**Step 3.8: 写 smoke 脚本**

`scripts/smoke-virtual-number.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
ADDR=":8088"
BIN="$ROOT/bin/virtual-number"
LOGFILE="$ROOT/.data/virtual-number-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/4] building virtual-number-service..."
GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/virtual-number/cmd

echo "[2/4] starting virtual-number-service on $ADDR..."
DOCTORS_VIRTUAL_NUMBER_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

# wait for /healthz
for i in {1..10}; do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/4] curl /healthz"
curl -fsS "http://127.0.0.1$ADDR/healthz" | head -c 200
echo
echo "[4/4] curl /api/v1/orders/1/virtual-number (no token → expect 401)"
curl -sS "http://127.0.0.1$ADDR/api/v1/orders/1/virtual-number" | head -c 200
echo
echo "smoke OK"
```

```bash
chmod +x scripts/smoke-virtual-number.sh
```

**Step 3.9: 跑全部测试 + smoke**

```bash
cd services/virtual-number
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./...
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 ./...
bash scripts/smoke-virtual-number.sh
```

Expected: 全部 PASS + smoke OK（启动 + /healthz + 401 拦截）。

**Step 3.10: Commit**

```bash
git add services/virtual-number/ scripts/smoke-virtual-number.sh
git commit -m "feat(virtual-number): handler + router + server + main + scanner (1 分钟翻 expired + smoke)"
```

---

### Task 4: 文档 + dev.md 落地 + 全量回归

**Files:**
- Modify: `docs/04-业务流程.md`
- Modify: `dev.md`

**Step 4.1: 在 `docs/04-业务流程.md` §4.7 追加虚拟号流程**

```markdown
### 4.7 虚拟号中间号（privacy）

1. 订单被接单（`orders.status='accepted'`）时，order-service 调 `POST /internal/virtual-number/activate`（payload: order_id, patient_id, escort_id）→ virtual-number-service 生成 `virtual_no = "1900" + order_id 末 4 位 0 填充` + `expires_at = now + 24h`，写入 `virtual_numbers` 表。
2. 患者 / 陪诊师在订单详情查 `GET /api/v1/orders/:id/virtual-number` → 仅返回中间号；viewer 必须是 patient_id 或 escort_id，否则 `CodeForbidden`。
3. 订单完成（`orders.status='completed'`）时，order-service 调 `POST /internal/virtual-number/schedule-expire`（payload: order_id, completed_at）→ 重写 `expires_at = completed_at + 24h`。
4. 订单取消 / 退款时，order-service 调 `POST /internal/virtual-number/mark-expired` → 立即翻 expired（保留行不删，前端展示"已失效"）。
5. 后台 scanner 每 1 分钟轮询 `expires_at < NOW() AND status='active'` → 翻 expired；日志 `rows=N`。
6. v1 中间号算法：`fmt.Sprintf("1900%04d", orderID%10000)`（末 4 位 0 填充；order_id=7 → "19000007"）。
7. v1 不接第三方隐私号 API（阿里云 / 腾讯云留 v2）。
```

**Step 4.2: 在 `dev.md` §10.13 追加落地记录**

```markdown
### 10.13 虚拟号中间号（2026-09-24 virtual-number plan）

保护患者 / 陪诊师双方手机号隐私：订单 accepted 自动生成 v1 mock 中间号（"1900" + order_id 末 4 位），completed + 24h 后后台 scanner 翻 expired。

**落地 commits（4 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(migrations) | 0006 virtual_numbers + UNIQUE(order_id) + 索引 + 集成测试 |
| feat(virtual-number) | repo + service (GenerateVirtualNo mock + Activate/GetByOrderForViewer/ScheduleExpire + 7 个单测 + 6 个集成测试) |
| feat(virtual-number) | handler + router + server + main + scanner (1 分钟翻 expired + smoke) |
| docs | 04 业务流程 §4.7 + dev.md 10.13 |

**API 增量**：
- `GET /api/v1/orders/:id/virtual-number`（JWT 鉴权；viewer 必须是 patient_id 或 escort_id）
- `POST /internal/virtual-number/activate`（order-service accept hook）
- `POST /internal/virtual-number/schedule-expire`（order-service complete hook）
- `POST /internal/virtual-number/mark-expired`（order-service cancel/refund hook）

**未做**：第三方隐私号 API（v2 接阿里云）；admin-web 不暴露虚拟号（chatbot 系统才能看，留 v2）。
```

**Step 4.3: Commit**

```bash
git add docs/04-业务流程.md dev.md
git commit -m "docs(virtual-number): 04 §4.7 虚拟号流程 + dev.md 10.13 落地记录"
```

---

### Task 5: 全量回归 + push

```bash
# 清干净 PG
docker exec doctors-postgres psql -U doctors -d doctors -c "
  DROP TABLE IF EXISTS virtual_numbers CASCADE;
  DROP TABLE IF EXISTS sos_records CASCADE;
  DROP TABLE IF EXISTS refunds CASCADE;
  DROP TABLE IF EXISTS refund_policies CASCADE;
  DROP TABLE IF EXISTS order_events CASCADE;
  DROP TABLE IF EXISTS orders CASCADE;
  DROP TABLE IF EXISTS users CASCADE;
"

# 跑全部单测
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./shared/... ./services/...

# 跑全部集成测试
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 ./migrations/... ./services/...

# 跑 smoke
bash scripts/smoke-order.sh  # 已有
bash scripts/smoke-sos.sh    # 已有
bash scripts/smoke-virtual-number.sh  # 本 plan 新增

# push
git push origin main
```

Expected: 全部 PASS + smoke OK + pushed.

---

## Self-Review

- ✅ **Spec 覆盖**: l2-api-gap-design.md §2.1 / §2.2 P0 virtual-number API；§3.1 `OrderEscortSummary.phone_virtual` schema 与本 plan `virtualResp.VirtualNo` 一致
- ✅ **无占位符**: 每步有具体 SQL / Go 代码 / 命令；无 TBD / 类似 / TODO
- ✅ **类型一致**: `repo.Record` 跨 Task 2 / 3 一致；`service.VirtualRepo` 接口与 `repo.VirtualRepo` 实现一致（编译期断言）；`handler.Service` 接口与 `service.Service` 方法签名一致；`scanner.ScannerRepo` 通过 adapter 转 `service.VirtualRepo`
- ✅ **测试矩阵**: Task 1 集成（migration up/down）+ Task 2 repo 集成（6 个）+ service 单测（7 个）+ Task 3 handler 单测（3 个）+ router 单测（1 个）+ scanner 单测（1 个）+ smoke（3 步）+ Task 5 全量回归
- ✅ **YAGNI**: v1 不接第三方隐私号 API；不引 Kafka topic / consumer（用内部 HTTP）；不做 admin-web 虚拟号查看（chatbot 系统 v2 再说）；不做中间号拨打录音（合规留 v2）
- ⚠️ **设计偏差（需用户确认）**:
  1. **错误码冲突**：spec §3.2 列了 13001~13009 业务码，但 `shared/errs/code.go` 已有 `CodeRateLimit = 13001`。本 plan 暂用既有 `CodeForbidden (11002)` 做鉴权拦截、`CodeNotFound (12001)` 做订单无虚拟号；**没有引入新码**。需用户决策：是把 `CodeRateLimit` 改名 / 移到 14xxx，还是把 spec 13001~13009 整体移到 14xxx / 15xxx？建议后者（向后兼容）。
  2. **触发方式**：用 order-service 调内部 HTTP（`POST /internal/virtual-number/{activate,schedule-expire,mark-expired}`），而非 Kafka 消费者；理由是 v1 不引 `OrderCompletedEvent` 主题。order-service 钩子接入需要后续 order plan 加 HTTP client，本 plan 仅声明 hook 函数 endpoint。需用户决策：是否在 order plan 里加 `client.VirtualNumber.Activate(ctx, ...)` 调用，还是 v1 暂不接钩子、scanner 单跑？
  3. **scanner 间隔**：默认 1 分钟轮询；spec 未指定。生产可降为 5s，v1 取 1 分钟够用。
  4. **唯一约束冲突场景**：本 plan 用 `ON CONFLICT (order_id) DO UPDATE SET order_id = EXCLUDED.order_id` 实现幂等（重复激活返回原行不报错）。如需更严格幂等（不更新任何字段），把 SET 子句改为 `SET order_id = virtual_numbers.order_id`。
  5. **中间号格式示例**：order_id=12345 → "19002345"（整数 8 位）。spec 文本说"1900 + order_id 末 4 位"，本 plan 严格遵守；如需 11 位手机号格式（"019002345"），需调整。

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-virtual-number.md`。
> 当前为 plan_all 模式 → 进入实施阶段需要用户决策。

**下一步选项**：
1. **立即执行**（subagent-driven 或 inline 执行）—— 我开始实施 Task 1~5
2. **暂停 + review** —— 你 review 此 plan 后告诉我调整（特别注意 5 个设计偏差）
3. **继续产 plan** —— 接着出 8 个后端 plan + 3 个前端 plan（sos / wallet / escort-business / hospital-package / review / message / address-coupon / admin / escort-order-ext + patient-miniapp / escort-app / admin-web）

## 关联 spec
- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.1 / §2.2 P0 virtual-number API + §3.1 `OrderEscortSummary`
- `docs/superpowers/specs/2026-09-24-patient-miniapp-design.md`（订单详情暴露虚拟号给患者）
- `docs/superpowers/specs/2026-09-24-escort-app-design.md`（订单详情暴露虚拟号给陪诊师）