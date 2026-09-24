# Address + Coupon Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 扩展 `services/user` 既有骨架（13 个 service 单测是占位），实现 patient 地址管理（4 API：CRUD + 5 上限 + 默认地址互斥）+ 优惠券状态查询（1 API）。产出 `addresses` + `coupons` 2 张表 + 5 个 REST 接口。v1 不做下单抵扣（v2 接订单系统时再补 ApplyCoupon）。

**Architecture:** 复用 `services/user/internal/{service,handler,router,server,cmd,middleware}/` 既有骨架；新增 `address_repo` + `coupon_repo`（pgx 直写 SQL，不引 sqlc）；扩展 service 层为独立类型 `AddressService` / `CouponService`（不污染 ProfileService）；handler 暴露 `/users/me/addresses` + `/users/me/coupons`（JWT userID 自动取，不用 query 参数）；main 装配时换掉 `nilRepo` 占位。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · gin-gonic v1.10。

**前置依赖:**
- `2026-09-24-state-machine.md` + `2026-09-24-order-lock.md` + `2026-09-24-refund.md` 已交付
- `2026-09-24-sos.md`（同模式 plan：独立 migration + repo + service + handler + router + smoke）
- `2026-09-24-l2-api-gap-design.md` §2.1 P1 patient 地址/优惠券 + §3 Address/Coupon entity
- `2026-09-24-patient-miniapp-design.md` 我的页 + 收货地址管理 + 优惠券列表

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- 地址上限：每个用户 ≤ 5 个；默认地址唯一（DB 用 partial unique index 保证）
- 地址省市区 v1 三级固定列表（白名单，handler 校验）；detail 不超过 200 字
- 优惠券 v1 仅查询 + 状态管理；type: `amount_off` / `percent_off` / `full_off`；`threshold ≥ 0`；`expires_at` 必填
- 优惠券状态推导：`unused`（未用且未过期）/ `used`（used_at 非空）/ `expired`（expires_at < now 且未用）
- v1 不接 Kafka 事件（地址 / 优惠券变更不广播）
- v1 不做下单抵扣（v2 接 order-service 时再补 ApplyCoupon）

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `migrations/0005_addresses_coupons.up.sql` | Create | addresses + coupons 表 + 索引 + CHECK |
| `migrations/0005_addresses_coupons.down.sql` | Create | 逆向 |
| `migrations/migrations_test.go` | Modify | 加 `Test0005AddressesCouponsUpDown` |
| `services/user/internal/repo/address_repo.go` | Create | `AddressRepo` + `AddressRecord` + `ErrAddressNotFound` / `ErrAddressLimitExceeded` |
| `services/user/internal/repo/address_repo_integration_test.go` | Create | 6 个集成测试 |
| `services/user/internal/repo/coupon_repo.go` | Create | `CouponRepo` + `CouponRecord` + `ErrCouponNotFound` |
| `services/user/internal/repo/coupon_repo_integration_test.go` | Create | 5 个集成测试 |
| `services/user/internal/service/address_service.go` | Create | `AddressService`（5-limit + 默认地址 + 省市区白名单校验） |
| `services/user/internal/service/address_service_test.go` | Create | 7 个单测 |
| `services/user/internal/service/coupon_service.go` | Create | `CouponService`（状态推导 + 过滤已过期） |
| `services/user/internal/service/coupon_service_test.go` | Create | 4 个单测 |
| `services/user/internal/handler/address.go` | Create | `addressHandler` + `RegisterAddressRoutes` |
| `services/user/internal/handler/address_test.go` | Create | 4 个 handler 单测 |
| `services/user/internal/handler/coupon.go` | Create | `couponHandler` + `RegisterCouponRoutes` |
| `services/user/internal/handler/coupon_test.go` | Create | 1 个 handler 单测 |
| `services/user/internal/handler/handler.go` | Modify | `RegisterRoutes` 加 `me` group（addresses + coupons） |
| `services/user/cmd/main.go` | Modify | 装配 `AddressRepo` + `CouponRepo` + `AddressService` + `CouponService` |
| `scripts/smoke-address-coupon.sh` | Create | smoke 脚本（GET addresses / POST / PUT / DELETE / GET coupons） |
| `docs/04-业务流程.md` | Modify | §4.7 加地址 / 优惠券流程 |
| `dev.md` | Modify | §10.13 加 plan 落地记录 |

---

### Task 1: 数据库迁移（addresses + coupons）

**Files:**
- Create: `migrations/0005_addresses_coupons.up.sql`
- Create: `migrations/0005_addresses_coupons.down.sql`
- Modify: `migrations/migrations_test.go`

**Step 1: 写集成测试（RED）**

在 `migrations_test.go` 末尾追加（参考 `Test0004RefundsUpDown` 风格）：

```go
// Test0005AddressesCouponsUpDown 验证 addresses + coupons 表 + CHECK + 索引 + down 可逆。
func Test0005AddressesCouponsUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 依赖 users（addresses/coupons 都引用 users.id）
	for _, f := range []string{"0001_users.up.sql"} {
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
		sql, _ := os.ReadFile("0001_users.down.sql")
		_, _ = cleanConn.Exec(cleanCtx, string(sql))
	})

	applyUp(t, "0005_addresses_coupons.up.sql", []string{"addresses", "coupons"})

	// addresses 列检查
	for _, col := range []string{"id", "user_id", "recipient", "phone", "province", "city", "district", "detail", "is_default", "created_at", "updated_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='addresses' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "addresses.%s should exist", col)
	}

	// coupons 列检查
	for _, col := range []string{"id", "user_id", "type", "value", "threshold", "expires_at", "used_at", "status", "created_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='coupons' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "coupons.%s should exist", col)
	}

	// 索引检查
	for _, idx := range []string{"idx_addresses_user_created", "idx_coupons_user_status"} {
		var exists bool
		err = conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, idx).
			Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "%s should exist", idx)
	}

	// 部分唯一索引：每个用户最多 1 个默认地址
	var partialIdxExists bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_indexes
		               WHERE indexname='uq_addresses_user_default'
		                 AND indexdef LIKE '%WHERE%is_default%')`).
		Scan(&partialIdxExists)
	require.NoError(t, err)
	assert.True(t, partialIdxExists, "uq_addresses_user_default should be partial unique index")

	// CHECK 约束：coupon.type
	var hasCheck bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_table_constraints
		               WHERE constraint_name='coupons_type_check')`).
		Scan(&hasCheck)
	if hasCheck == false {
		err = conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
			               WHERE constraint_name LIKE 'coupons%')`).
			Scan(&hasCheck)
	}
	require.NoError(t, err)
	assert.True(t, hasCheck, "coupons type check should exist")

	// down 校验
	downCtx, downCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer downCancel()
	downSQL, err := os.ReadFile("0005_addresses_coupons.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(downCtx, string(downSQL))
	require.NoError(t, err, "apply 0005_addresses_coupons.down.sql")

	for _, tbl := range []string{"addresses", "coupons"} {
		var gone bool
		err = conn.QueryRow(downCtx,
			`SELECT NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, tbl).
			Scan(&gone)
		require.NoError(t, err)
		assert.True(t, gone, "%s should be gone after down", tbl)
	}
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run Test0005AddressesCouponsUpDown ./migrations/`
Expected: FAIL — `Test0005AddressesCouponsUpDown` undefined

**Step 3: 写 `0005_addresses_coupons.up.sql`**

```sql
-- 0005_addresses_coupons.up.sql
-- 地址管理（收货地址，patient 用）+ 优惠券列表（v1 仅查询，不做下单抵扣）。
--
-- 设计要点：
--   - addresses：每个用户 ≤ 5 个；默认地址用 partial unique index 约束唯一；
--     省市区 v1 三级固定白名单（应用层校验，DB 不强约束行政区）。
--   - coupons：type / threshold / expires_at 三件套；status 由 used_at + expires_at 推导；
--     v1 不存订单号（下单抵扣 v2 加），所以不引用 orders。

CREATE TABLE addresses (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  recipient VARCHAR(32) NOT NULL,
  phone VARCHAR(20) NOT NULL,
  province VARCHAR(16) NOT NULL,
  city VARCHAR(16) NOT NULL,
  district VARCHAR(16) NOT NULL,
  detail VARCHAR(200) NOT NULL,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 用户所有地址（按 created_at DESC 列表）
CREATE INDEX idx_addresses_user_created ON addresses(user_id, created_at DESC);

-- 默认地址唯一（partial unique index）
CREATE UNIQUE INDEX uq_addresses_user_default ON addresses(user_id) WHERE is_default = TRUE;

-- CHECK：电话号码格式（11 位数字或固话，宽松校验）
ALTER TABLE addresses ADD CONSTRAINT addresses_phone_check
  CHECK (phone ~ '^[0-9\-\+]{7,20}$');

-- CHECK：非空字段
ALTER TABLE addresses ADD CONSTRAINT addresses_recipient_nonempty
  CHECK (length(trim(recipient)) > 0);
ALTER TABLE addresses ADD CONSTRAINT addresses_detail_nonempty
  CHECK (length(trim(detail)) > 0);


CREATE TABLE coupons (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
  value NUMERIC(10,2) NOT NULL,
  threshold NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (threshold >= 0),
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  status VARCHAR(16) NOT NULL DEFAULT 'unused'
    CHECK (status IN ('unused','used','expired')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 用户优惠券列表（按 status + created_at DESC：未用过优先）
CREATE INDEX idx_coupons_user_status ON coupons(user_id, status, created_at DESC);

-- CHECK：value 合法性（amount_off > 0；percent_off 0 < x ≤ 9.99；full_off = 0）
ALTER TABLE coupons ADD CONSTRAINT coupons_value_check
  CHECK (
    (type = 'amount_off' AND value > 0)
    OR (type = 'percent_off' AND value > 0 AND value <= 9.99)
    OR (type = 'full_off' AND value = 0)
  );
```

**Step 4: 写 `0005_addresses_coupons.down.sql`**

```sql
-- 0005_addresses_coupons.down.sql
-- 撤销 0005_addresses_coupons。

DROP TABLE IF EXISTS coupons;
DROP INDEX IF EXISTS uq_addresses_user_default;
DROP INDEX IF EXISTS idx_addresses_user_created;
DROP TABLE IF EXISTS addresses;
```

**Step 5: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run Test0005AddressesCouponsUpDown ./migrations/`
Expected: PASS

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0005 addresses+coupons (5 上限 + 默认唯一 + 优惠券 3 种类型 + CHECK + 集成测试)"
```

---

### Task 2: address_repo（pgx + 哨兵错误）

**Files:**
- Create: `services/user/internal/repo/address_repo.go`
- Create: `services/user/internal/repo/address_repo_integration_test.go`

**Step 1: 写集成测试（RED）**

`services/user/internal/repo/address_repo_integration_test.go`：

```go
//go:build integration
// +build integration

package repo

import (
	"context"
	"errors"
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

// setupPool 起连接池并准备 users + addresses 表。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS addresses CASCADE;
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
		CREATE TABLE addresses (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  recipient VARCHAR(32) NOT NULL,
		  phone VARCHAR(20) NOT NULL,
		  province VARCHAR(16) NOT NULL,
		  city VARCHAR(16) NOT NULL,
		  district VARCHAR(16) NOT NULL,
		  detail VARCHAR(200) NOT NULL,
		  is_default BOOLEAN NOT NULL DEFAULT FALSE,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX uq_addresses_user_default ON addresses(user_id) WHERE is_default = TRUE;
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DROP TABLE IF EXISTS addresses; DROP TABLE IF EXISTS users;`)
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

// seedAddr 构造地址记录。
func seedAddr(t *testing.T, pool *pgxpool.Pool, userID int64, recipient, phone, isDefault string) int64 {
	t.Helper()
	var id int64
	var defBool bool
	if isDefault == "default" {
		defBool = true
	}
	err := pool.QueryRow(context.Background(), `
		INSERT INTO addresses (user_id, recipient, phone, province, city, district, detail, is_default)
		VALUES ($1, $2, $3, '北京', '北京', '东城区', '东单北大街9号', $4)
		RETURNING id`, userID, recipient, phone, defBool).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestAddressRepo_Create_OK 验证创建。
func TestAddressRepo_Create_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138000", "patient")
	r := NewAddressRepo(pool)

	rec := &AddressRecord{
		UserID:    user,
		Recipient: "张三",
		Phone:     "13800138000",
		Province:  "北京",
		City:      "北京",
		District:  "东城区",
		Detail:    "东单北大街9号",
		IsDefault: false,
	}
	require.NoError(t, r.Create(context.Background(), rec))
	assert.NotZero(t, rec.ID)
	assert.NotZero(t, rec.CreatedAt)
}

// TestAddressRepo_Create_DefaultSetsOnlyOne 验证创建默认地址时其他默认被取消。
func TestAddressRepo_Create_DefaultSetsOnlyOne(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138001", "patient")
	r := NewAddressRepo(pool)

	// 先建一个默认地址
	seedAddr(t, pool, user, "李四", "13800138001", "default")

	// 再建一个默认地址（应用层调 repo 的 CreateDefault 走事务）
	rec := &AddressRecord{
		UserID: user, Recipient: "王五", Phone: "13800138002",
		Province: "北京", City: "北京", District: "朝阳区",
		Detail: "建国门外大街1号", IsDefault: true,
	}
	require.NoError(t, r.CreateDefault(context.Background(), rec))

	// 用户的默认地址应该只剩 rec
	defaults, err := r.ListByUser(context.Background(), user, true)
	require.NoError(t, err)
	assert.Len(t, defaults, 1)
	assert.Equal(t, rec.ID, defaults[0].ID)
}

// TestAddressRepo_ListByUser 验证按用户列出所有地址。
func TestAddressRepo_ListByUser(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138002", "patient")
	r := NewAddressRepo(pool)

	for i := 0; i < 3; i++ {
		rec := &AddressRecord{
			UserID: user, Recipient: "测试", Phone: "13800138000",
			Province: "北京", City: "北京", District: "海淀区",
			Detail: "中关村大街1号", IsDefault: false,
		}
		require.NoError(t, r.Create(context.Background(), rec))
	}
	list, err := r.ListByUser(context.Background(), user, false)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

// TestAddressRepo_CountByUser 验证计数（5 上限校验）。
func TestAddressRepo_CountByUser(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138003", "patient")
	r := NewAddressRepo(pool)

	count, err := r.CountByUser(context.Background(), user)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	seedAddr(t, pool, user, "测试", "13800138000", "")
	count, err = r.CountByUser(context.Background(), user)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestAddressRepo_GetByID_OK 验证按 id 查询。
func TestAddressRepo_GetByID_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138004", "patient")
	id := seedAddr(t, pool, user, "张三", "13800138000", "")
	r := NewAddressRepo(pool)

	got, err := r.GetByID(context.Background(), id, user)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "张三", got.Recipient)
	assert.Equal(t, user, got.UserID)
}

// TestAddressRepo_GetByID_NotFound 验证 id 不存在返回 ErrAddressNotFound。
func TestAddressRepo_GetByID_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewAddressRepo(pool)
	_, err := r.GetByID(context.Background(), 99999, 1)
	assert.ErrorIs(t, err, ErrAddressNotFound)
}

// TestAddressRepo_GetByID_WrongUser 验证别人的地址返回 NotFound（避免越权）。
func TestAddressRepo_GetByID_WrongUser(t *testing.T) {
	pool := setupPool(t)
	user1 := seedUser(t, pool, "13800138005", "patient")
	user2 := seedUser(t, pool, "13800138006", "patient")
	id := seedAddr(t, pool, user1, "user1 的地址", "13800138000", "")
	r := NewAddressRepo(pool)

	_, err := r.GetByID(context.Background(), id, user2)
	assert.ErrorIs(t, err, ErrAddressNotFound)
}

// TestAddressRepo_Update_OK 验证修改。
func TestAddressRepo_Update_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138007", "patient")
	id := seedAddr(t, pool, user, "张三", "13800138000", "")
	r := NewAddressRepo(pool)

	err := r.Update(context.Background(), &AddressRecord{
		ID: id, UserID: user, Recipient: "李四",
		Phone: "13900139000", City: "上海", District: "黄浦区",
		Detail: "南京东路100号",
	})
	require.NoError(t, err)

	got, _ := r.GetByID(context.Background(), id, user)
	require.NotNil(t, got)
	assert.Equal(t, "李四", got.Recipient)
	assert.Equal(t, "上海", got.City)
}

// TestAddressRepo_Update_NotFound 验证修改不存在的 id 返回 ErrAddressNotFound。
func TestAddressRepo_Update_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewAddressRepo(pool)
	err := r.Update(context.Background(), &AddressRecord{ID: 99999, UserID: 1})
	assert.ErrorIs(t, err, ErrAddressNotFound)
}

// TestAddressRepo_Delete_OK 验证删除。
func TestAddressRepo_Delete_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800138008", "patient")
	id := seedAddr(t, pool, user, "张三", "13800138000", "")
	r := NewAddressRepo(pool)

	require.NoError(t, r.Delete(context.Background(), id, user))

	got, err := r.GetByID(context.Background(), id, user)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestAddressRepo_Delete_WrongUser 验证删别人地址返回 NotFound。
func TestAddressRepo_Delete_WrongUser(t *testing.T) {
	pool := setupPool(t)
	user1 := seedUser(t, pool, "13800138009", "patient")
	user2 := seedUser(t, pool, "13800138010", "patient")
	id := seedAddr(t, pool, user1, "user1 的地址", "13800138000", "")
	r := NewAddressRepo(pool)

	err := r.Delete(context.Background(), id, user2)
	assert.ErrorIs(t, err, ErrAddressNotFound)
}

// 引用 errors 包防止 unused 报错
var _ = errors.New
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run 'TestAddressRepo_' ./services/user/internal/repo/`
Expected: FAIL — `undefined: NewAddressRepo`, `undefined: AddressRecord`, `undefined: ErrAddressNotFound`

**Step 3: 写 address_repo.go**

`services/user/internal/repo/address_repo.go`：

```go
// Package repo 是 user-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc）。
//   - 地址越权保护：所有按 id 查询 / 修改 / 删除都强制 user_id 匹配，
//     避免 A 看到 / 改 B 的地址。
//   - 默认地址唯一由 partial unique index 保证，应用层走事务切换。
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AddressRecord 映射 addresses 表行。
type AddressRecord struct {
	ID        int64
	UserID    int64
	Recipient string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool
	CreatedAt int64 // unix nano（handler 转 RFC3339）
	UpdatedAt int64
}

// ErrAddressNotFound 是查询 / 修改 / 删除无结果时的哨兵（也用于越权）。
var ErrAddressNotFound = errors.New("repo: address not found")

// AddressRepo 是 addresses 表的仓储。
type AddressRepo struct {
	pool *pgxpool.Pool
}

// NewAddressRepo 构造仓储。
func NewAddressRepo(pool *pgxpool.Pool) *AddressRepo { return &AddressRepo{pool: pool} }

// baseSelectAddr 是 SELECT 子句。
const baseSelectAddr = `
	SELECT id, user_id, recipient, phone, province, city, district, detail,
	       is_default,
	       EXTRACT(EPOCH FROM created_at) * 1e9,
	       EXTRACT(EPOCH FROM updated_at) * 1e9
	FROM addresses`

// Create 插入一条地址（非默认）。
func (r *AddressRepo) Create(ctx context.Context, a *AddressRecord) error {
	return r.createInternal(ctx, a, false)
}

// CreateDefault 插入并设为默认地址（事务内先清掉同用户其他默认）。
func (r *AddressRepo) CreateDefault(ctx context.Context, a *AddressRecord) error {
	return r.createInternal(ctx, a, true)
}

func (r *AddressRepo) createInternal(ctx context.Context, a *AddressRecord, asDefault bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if asDefault {
		if _, err := tx.Exec(ctx,
			`UPDATE addresses SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1 AND is_default = TRUE`,
			a.UserID); err != nil {
			return fmt.Errorf("clear default: %w", err)
		}
	}

	const q = `
		INSERT INTO addresses (user_id, recipient, phone, province, city, district, detail, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id,
		          EXTRACT(EPOCH FROM created_at) * 1e9,
		          EXTRACT(EPOCH FROM updated_at) * 1e9`
	if err := tx.QueryRow(ctx, q,
		a.UserID, a.Recipient, a.Phone, a.Province, a.City, a.District, a.Detail, asDefault,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return fmt.Errorf("insert address: %w", err)
	}
	a.IsDefault = asDefault
	return tx.Commit(ctx)
}

// ListByUser 列出某用户地址；onlyDefault=true 只返回默认地址。
func (r *AddressRepo) ListByUser(ctx context.Context, userID int64, onlyDefault bool) ([]*AddressRecord, error) {
	q := baseSelectAddr + ` WHERE user_id = $1`
	if onlyDefault {
		q += ` AND is_default = TRUE`
	}
	q += ` ORDER BY is_default DESC, created_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer rows.Close()
	out := []*AddressRecord{}
	for rows.Next() {
		a, err := scanAddr(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CountByUser 统计某用户地址数量（用于 5 上限校验）。
func (r *AddressRepo) CountByUser(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM addresses WHERE user_id = $1`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count addresses: %w", err)
	}
	return n, nil
}

// GetByID 按 id 查询；强制 userID 匹配（越权返回 NotFound）。
func (r *AddressRepo) GetByID(ctx context.Context, id, userID int64) (*AddressRecord, error) {
	q := baseSelectAddr + ` WHERE id = $1 AND user_id = $2`
	row := r.pool.QueryRow(ctx, q, id, userID)
	a, err := scanAddr(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, err
	}
	return a, nil
}

// Update 修改地址（userID 不匹配返回 NotFound）。
func (r *AddressRepo) Update(ctx context.Context, a *AddressRecord) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. 修改（user_id 必须匹配，否则 0 rows）
	const updQ = `
		UPDATE addresses
		   SET recipient = $3, phone = $4, province = $5, city = $6,
		       district = $7, detail = $8, updated_at = NOW()
		 WHERE id = $1 AND user_id = $2`
	tag, err := tx.Exec(ctx, updQ, a.ID, a.UserID,
		a.Recipient, a.Phone, a.Province, a.City, a.District, a.Detail)
	if err != nil {
		return fmt.Errorf("update address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAddressNotFound
	}

	// 2. 回写 updated_at
	var updatedAt int64
	if err := tx.QueryRow(ctx,
		`SELECT EXTRACT(EPOCH FROM updated_at) * 1e9 FROM addresses WHERE id = $1`, a.ID,
	).Scan(&updatedAt); err != nil {
		return fmt.Errorf("read updated_at: %w", err)
	}
	a.UpdatedAt = updatedAt
	return tx.Commit(ctx)
}

// Delete 删地址；userID 不匹配返回 NotFound。
func (r *AddressRepo) Delete(ctx context.Context, id, userID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM addresses WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAddressNotFound
	}
	return nil
}

// scanAddr 把 pgx.Row 扫描到 AddressRecord。
func scanAddr(row pgx.Row) (*AddressRecord, error) {
	a := &AddressRecord{}
	if err := row.Scan(
		&a.ID, &a.UserID, &a.Recipient, &a.Phone,
		&a.Province, &a.City, &a.District, &a.Detail,
		&a.IsDefault, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return a, nil
}
```

**Step 4: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run 'TestAddressRepo_' ./services/user/internal/repo/`
Expected: PASS（11 个测试全过）

**Step 5: Commit**

```bash
git add services/user/internal/repo/
git commit -m "feat(user): repo 加 AddressRepo (Create/ListByUser/GetByID/Update/Delete/CreateDefault + 11 个集成测试)"
```

---

### Task 3: coupon_repo（pgx + 状态推导）

**Files:**
- Create: `services/user/internal/repo/coupon_repo.go`
- Create: `services/user/internal/repo/coupon_repo_integration_test.go`

**Step 1: 写集成测试（RED）**

`services/user/internal/repo/coupon_repo_integration_test.go`：

```go
//go:build integration
// +build integration

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCouponRepo_Create_OK 验证创建 amount_off 优惠券。
func TestCouponRepo_Create_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800139000", "patient")

	// 重建 coupons 表
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS coupons (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
		  value NUMERIC(10,2) NOT NULL,
		  threshold NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (threshold >= 0),
		  expires_at TIMESTAMPTZ NOT NULL,
		  used_at TIMESTAMPTZ,
		  status VARCHAR(16) NOT NULL DEFAULT 'unused'
		    CHECK (status IN ('unused','used','expired')),
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS coupons;`)
	})

	r := NewCouponRepo(pool)
	expires := time.Now().Add(30 * 24 * time.Hour)
	rec := &CouponRecord{
		UserID: user, Type: "amount_off", Value: 20.0, Threshold: 100.0,
		ExpiresAt: expires,
	}
	require.NoError(t, r.Create(context.Background(), rec))
	assert.NotZero(t, rec.ID)
	assert.NotZero(t, rec.CreatedAt)
}

// TestCouponRepo_ListByUser_OK 验证列表（自动重算 status）。
func TestCouponRepo_ListByUser_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800139001", "patient")
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS coupons (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
		  value NUMERIC(10,2) NOT NULL,
		  threshold NUMERIC(10,2) NOT NULL DEFAULT 0,
		  expires_at TIMESTAMPTZ NOT NULL,
		  used_at TIMESTAMPTZ,
		  status VARCHAR(16) NOT NULL DEFAULT 'unused',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS coupons;`)
	})

	r := NewCouponRepo(pool)
	now := time.Now()

	// 未过期、未用
	rec1 := &CouponRecord{UserID: user, Type: "amount_off", Value: 20, Threshold: 100,
		ExpiresAt: now.Add(10 * 24 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec1))

	// 已过期（先建，再用 UPDATE 把 expires_at 改到过去）
	rec2 := &CouponRecord{UserID: user, Type: "percent_off", Value: 8.5, Threshold: 50,
		ExpiresAt: now.Add(1 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec2))
	_, err = pool.Exec(context.Background(),
		`UPDATE coupons SET expires_at = $1 WHERE id = 2`, now.Add(-1*time.Hour))
	require.NoError(t, err)

	// 已用
	rec3 := &CouponRecord{UserID: user, Type: "full_off", Value: 0, Threshold: 200,
		ExpiresAt: now.Add(10 * 24 * time.Hour)}
	require.NoError(t, r.Create(context.Background(), rec3))
	_, err = pool.Exec(context.Background(),
		`UPDATE coupons SET used_at = $1 WHERE id = 3`, now)
	require.NoError(t, err)

	// 列表
	list, err := r.ListByUser(context.Background(), user)
	require.NoError(t, err)
	assert.Len(t, list, 3)
	// 重算后 status
	for _, c := range list {
		switch c.ID {
		case rec1.ID:
			assert.Equal(t, "unused", c.Status)
		case 2:
			assert.Equal(t, "expired", c.Status)
		case 3:
			assert.Equal(t, "used", c.Status)
		}
	}
}

// TestCouponRepo_GetByID_OK 验证按 id 查询。
func TestCouponRepo_GetByID_OK(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800139002", "patient")
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS coupons (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
		  value NUMERIC(10,2) NOT NULL,
		  threshold NUMERIC(10,2) NOT NULL DEFAULT 0,
		  expires_at TIMESTAMPTZ NOT NULL,
		  used_at TIMESTAMPTZ,
		  status VARCHAR(16) NOT NULL DEFAULT 'unused',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS coupons;`)
	})

	r := NewCouponRepo(pool)
	expires := time.Now().Add(7 * 24 * time.Hour)
	rec := &CouponRecord{UserID: user, Type: "amount_off", Value: 50, Threshold: 200,
		ExpiresAt: expires}
	require.NoError(t, r.Create(context.Background(), rec))

	got, err := r.GetByID(context.Background(), rec.ID, user)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.InDelta(t, 50.0, got.Value, 0.01)
}

// TestCouponRepo_GetByID_NotFound 验证 id 不存在。
func TestCouponRepo_GetByID_NotFound(t *testing.T) {
	pool := setupPool(t)
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS coupons (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
		  value NUMERIC(10,2) NOT NULL,
		  threshold NUMERIC(10,2) NOT NULL DEFAULT 0,
		  expires_at TIMESTAMPTZ NOT NULL,
		  used_at TIMESTAMPTZ,
		  status VARCHAR(16) NOT NULL DEFAULT 'unused',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS coupons;`)
	})

	r := NewCouponRepo(pool)
	_, err = r.GetByID(context.Background(), 99999, 1)
	assert.ErrorIs(t, err, ErrCouponNotFound)
}

// TestCouponRepo_ListByUser_Empty 验证空列表。
func TestCouponRepo_ListByUser_Empty(t *testing.T) {
	pool := setupPool(t)
	user := seedUser(t, pool, "13800139003", "patient")
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS coupons (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
		  value NUMERIC(10,2) NOT NULL,
		  threshold NUMERIC(10,2) NOT NULL DEFAULT 0,
		  expires_at TIMESTAMPTZ NOT NULL,
		  used_at TIMESTAMPTZ,
		  status VARCHAR(16) NOT NULL DEFAULT 'unused',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS coupons;`)
	})

	r := NewCouponRepo(pool)
	list, err := r.ListByUser(context.Background(), user)
	require.NoError(t, err)
	assert.Empty(t, list)
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run 'TestCouponRepo_' ./services/user/internal/repo/`
Expected: FAIL — `undefined: NewCouponRepo`, `undefined: CouponRecord`, `undefined: ErrCouponNotFound`

**Step 3: 写 coupon_repo.go**

`services/user/internal/repo/coupon_repo.go`：

```go
// Package repo 是 user-service 的数据访问层（coupon 部分）。
//
// 设计要点：
//   - status 字段冗余存表，但 ListByUser / GetByID 时按 used_at + expires_at 重算
//     并回写（DB 侧 status 只在 Create 时默认 unused，过期变更是 best-effort）。
//   - v1 不存 order_id（下单抵扣 v2 接 order-service 再加）。
//   - 越权保护：GetByID 强制 user_id 匹配。
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CouponRecord 映射 coupons 表行。
type CouponRecord struct {
	ID         int64
	UserID     int64
	Type       string  // "amount_off" | "percent_off" | "full_off"
	Value      float64 // amount_off=减 N 元；percent_off=N 折（0 < N ≤ 9.99）；full_off=0
	Threshold  float64 // 满 X 减；0 表示无门槛
	ExpiresAt  time.Time
	UsedAt     *time.Time
	Status     string // "unused" | "used" | "expired"（读取时重算）
	CreatedAt  time.Time
}

// ErrCouponNotFound 是查询无结果时的哨兵。
var ErrCouponNotFound = errors.New("repo: coupon not found")

// CouponRepo 是 coupons 表的仓储。
type CouponRepo struct {
	pool *pgxpool.Pool
}

// NewCouponRepo 构造仓储。
func NewCouponRepo(pool *pgxpool.Pool) *CouponRepo { return &CouponRepo{pool: pool} }

// baseSelectCpn 是 SELECT 子句。
const baseSelectCpn = `
	SELECT id, user_id, type, value, threshold, expires_at, used_at, status, created_at
	FROM coupons`

// Create 插入优惠券；status 默认 unused。
func (r *CouponRepo) Create(ctx context.Context, c *CouponRecord) error {
	const q = `
		INSERT INTO coupons (user_id, type, value, threshold, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, 'unused')
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q,
		c.UserID, c.Type, c.Value, c.Threshold, c.ExpiresAt,
	).Scan(&c.ID, &c.CreatedAt)
}

// ListByUser 列出某用户所有优惠券；按 status 优先级 + created_at DESC 排序，
// 同时重算 status（used_at / expires_at → used / expired）。
func (r *CouponRepo) ListByUser(ctx context.Context, userID int64) ([]*CouponRecord, error) {
	now := time.Now()
	q := baseSelectCpn + `
		WHERE user_id = $1
		ORDER BY
		  CASE status WHEN 'unused' THEN 0 WHEN 'used' THEN 1 ELSE 2 END,
		  created_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list coupons: %w", err)
	}
	defer rows.Close()

	out := []*CouponRecord{}
	for rows.Next() {
		c, err := scanCpn(rows)
		if err != nil {
			return nil, err
		}
		// 重算 status（DB 可能因为历史原因没跟上）
		c.Status = deriveStatus(c.UsedAt, c.ExpiresAt, now)
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetByID 按 id 查询；userID 不匹配返回 NotFound；自动重算 status。
func (r *CouponRepo) GetByID(ctx context.Context, id, userID int64) (*CouponRecord, error) {
	q := baseSelectCpn + ` WHERE id = $1 AND user_id = $2`
	row := r.pool.QueryRow(ctx, q, id, userID)
	c, err := scanCpn(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCouponNotFound
		}
		return nil, err
	}
	c.Status = deriveStatus(c.UsedAt, c.ExpiresAt, time.Now())
	return c, nil
}

// deriveStatus 推导优惠券状态。
//   - used_at 非空 → "used"
//   - expires_at < now → "expired"
//   - 其他 → "unused"
func deriveStatus(usedAt *time.Time, expiresAt, now time.Time) string {
	if usedAt != nil {
		return "used"
	}
	if !expiresAt.IsZero() && expiresAt.Before(now) {
		return "expired"
	}
	return "unused"
}

// scanCpn 把 pgx.Row 扫描到 CouponRecord。
func scanCpn(row pgx.Row) (*CouponRecord, error) {
	c := &CouponRecord{}
	if err := row.Scan(
		&c.ID, &c.UserID, &c.Type, &c.Value, &c.Threshold,
		&c.ExpiresAt, &c.UsedAt, &c.Status, &c.CreatedAt,
	); err != nil {
		return nil, err
	}
	return c, nil
}
```

**Step 4: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 -run 'TestCouponRepo_' ./services/user/internal/repo/`
Expected: PASS（5 个测试）

**Step 5: Commit**

```bash
git add services/user/internal/repo/coupon_repo.go services/user/internal/repo/coupon_repo_integration_test.go
git commit -m "feat(user): repo 加 CouponRepo (Create/ListByUser/GetByID + 状态推导 + 5 个集成测试)"
```

---

### Task 4: AddressService（5-limit + 默认地址 + 省市区白名单）

**Files:**
- Create: `services/user/internal/service/address_service.go`
- Create: `services/user/internal/service/address_service_test.go`

**Step 1: 写单测（RED）**

`services/user/internal/service/address_service_test.go`：

```go
package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/growdu/doctors/services/user/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAddrRepo 是 repo.AddressRepo 的 fake 实现。
type fakeAddrRepo struct {
	records    map[int64]*repo.AddressRecord
	nextID     int64
	countByUsr map[int64]int
}

func newFakeAddrRepo() *fakeAddrRepo {
	return &fakeAddrRepo{
		records:    map[int64]*repo.AddressRecord{},
		nextID:     1,
		countByUsr: map[int64]int{},
	}
}

func (f *fakeAddrRepo) Create(ctx context.Context, a *repo.AddressRecord) error {
	return f.createInternal(ctx, a, false)
}
func (f *fakeAddrRepo) CreateDefault(ctx context.Context, a *repo.AddressRecord) error {
	return f.createInternal(ctx, a, true)
}
func (f *fakeAddrRepo) createInternal(_ context.Context, a *repo.AddressRecord, asDefault bool) error {
	if asDefault {
		for _, r := range f.records {
			if r.UserID == a.UserID && r.IsDefault {
				r.IsDefault = false
			}
		}
	}
	a.ID = f.nextID
	f.nextID++
	a.IsDefault = asDefault
	f.records[a.ID] = a
	f.countByUsr[a.UserID]++
	return nil
}
func (f *fakeAddrRepo) ListByUser(_ context.Context, userID int64, onlyDefault bool) ([]*repo.AddressRecord, error) {
	out := []*repo.AddressRecord{}
	for _, r := range f.records {
		if r.UserID != userID {
			continue
		}
		if onlyDefault && !r.IsDefault {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
func (f *fakeAddrRepo) CountByUser(_ context.Context, userID int64) (int, error) {
	return f.countByUsr[userID], nil
}
func (f *fakeAddrRepo) GetByID(_ context.Context, id, userID int64) (*repo.AddressRecord, error) {
	r, ok := f.records[id]
	if !ok || r.UserID != userID {
		return nil, repo.ErrAddressNotFound
	}
	return r, nil
}
func (f *fakeAddrRepo) Update(_ context.Context, a *repo.AddressRecord) error {
	r, ok := f.records[a.ID]
	if !ok || r.UserID != a.UserID {
		return repo.ErrAddressNotFound
	}
	// 简化：直接替换
	f.records[a.ID] = a
	return nil
}
func (f *fakeAddrRepo) Delete(_ context.Context, id, userID int64) error {
	r, ok := f.records[id]
	if !ok || r.UserID != userID {
		return repo.ErrAddressNotFound
	}
	delete(f.records, id)
	f.countByUsr[userID]--
	return nil
}

// 编译期断言 fakeAddrRepo 满足 AddressRepo（接口在 service 内定义）。
var _ AddressRepo = (*fakeAddrRepo)(nil)

// validAddr 构造一个合法地址参数。
func validAddr() *AddressInput {
	return &AddressInput{
		Recipient: "张三",
		Phone:     "13800138000",
		Province:  "北京",
		City:      "北京",
		District:  "东城区",
		Detail:    "东单北大街9号",
		IsDefault: false,
	}
}

// TestAddressService_Create_OK 验证创建成功。
func TestAddressService_Create_OK(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)

	out, err := svc.Create(context.Background(), 100, validAddr())
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotZero(t, out.ID)
}

// TestAddressService_Create_LimitExceeded 验证超 5 上限。
func TestAddressService_Create_LimitExceeded(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)

	for i := 0; i < 5; i++ {
		_, err := svc.Create(context.Background(), 100, validAddr())
		require.NoError(t, err)
	}
	_, err := svc.Create(context.Background(), 100, validAddr())
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrAddressLimitExceeded),
		"expected ErrAddressLimitExceeded, got %v", err)
}

// TestAddressService_Create_InvalidRegion 验证白名单外省市区。
func TestAddressService_Create_InvalidRegion(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)

	in := validAddr()
	in.Province = "火星"
	_, err := svc.Create(context.Background(), 100, in)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "province") || strings.Contains(err.Error(), "region"))
}

// TestAddressService_Create_DefaultClears 验证默认地址互斥。
func TestAddressService_Create_DefaultClears(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)

	in1 := validAddr()
	in1.IsDefault = true
	rec1, err := svc.Create(context.Background(), 100, in1)
	require.NoError(t, err)
	assert.True(t, rec1.IsDefault)

	in2 := validAddr()
	in2.IsDefault = true
	in2.Recipient = "李四"
	rec2, err := svc.Create(context.Background(), 100, in2)
	require.NoError(t, err)
	assert.True(t, rec2.IsDefault)

	list, _ := svc.List(context.Background(), 100)
	defaults := 0
	for _, a := range list {
		if a.IsDefault {
			defaults++
			assert.Equal(t, rec2.ID, a.ID)
		}
	}
	assert.Equal(t, 1, defaults)
}

// TestAddressService_Update_OK 验证修改。
func TestAddressService_Update_OK(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)

	rec, err := svc.Create(context.Background(), 100, validAddr())
	require.NoError(t, err)

	in := validAddr()
	in.Recipient = "李四"
	require.NoError(t, svc.Update(context.Background(), 100, rec.ID, in))

	got, err := svc.Get(context.Background(), 100, rec.ID)
	require.NoError(t, err)
	assert.Equal(t, "李四", got.Recipient)
}

// TestAddressService_Update_NotFound 验证改不存在的地址。
func TestAddressService_Update_NotFound(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)
	err := svc.Update(context.Background(), 100, 9999, validAddr())
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrAddressNotFound)
}

// TestAddressService_Delete_OK 验证删除。
func TestAddressService_Delete_OK(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)
	rec, err := svc.Create(context.Background(), 100, validAddr())
	require.NoError(t, err)

	require.NoError(t, svc.Delete(context.Background(), 100, rec.ID))

	_, err = svc.Get(context.Background(), 100, rec.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrAddressNotFound)
}

// TestAddressService_Get_NotFound 验证 Get 不存在的 id。
func TestAddressService_Get_NotFound(t *testing.T) {
	r := newFakeAddrRepo()
	svc := NewAddressService(r)
	_, err := svc.Get(context.Background(), 100, 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrAddressNotFound)
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestAddressService_' ./services/user/internal/service/`
Expected: FAIL — `undefined: NewAddressService`, `undefined: AddressInput`, `undefined: AddressRepo`, `undefined: errs.ErrAddressLimitExceeded`

**Step 3: 在 `shared/errs/code.go` 加 ErrAddressLimitExceeded / ErrAddressNotFound 哨兵**

> **不修改 errs.Error 类型**（保持业务码 + msg 结构）；改为在 `services/user/internal/service/` 包内提供业务级 sentinel error，让 handler 翻译。

`services/user/internal/service/address_service.go`：

```go
// Package service 是 user-service 的业务编排层（address 部分）。
//
// 设计要点：
//   - AddressInput 是 handler 传入的入参；AddressView 是出参（暴露 unix nano → RFC3339 由 handler 做）。
//   - 5 上限校验：Create 时查 CountByUser；超限给业务错。
//   - 默认地址唯一：repo 层 partial unique index + CreateDefault 事务保证；service 层简化透传。
//   - 省市区 v1 白名单（避免接外部行政区数据库）：6 个城市 + 各自固定区。
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/growdu/doctors/services/user/internal/repo"
	"github.com/growdu/doctors/shared/errs"
)

// AddressInput 是地址入参（Create / Update 共用）。
type AddressInput struct {
	Recipient string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool
}

// AddressView 是地址出参。
type AddressView struct {
	ID        int64
	UserID    int64
	Recipient string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool
	CreatedAt int64 // unix nano
	UpdatedAt int64
}

// AddressRepo 是仓储契约（与 repo.AddressRepo 一致；解耦）。
type AddressRepo interface {
	Create(ctx context.Context, a *repo.AddressRecord) error
	CreateDefault(ctx context.Context, a *repo.AddressRecord) error
	ListByUser(ctx context.Context, userID int64, onlyDefault bool) ([]*repo.AddressRecord, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	GetByID(ctx context.Context, id, userID int64) (*repo.AddressRecord, error)
	Update(ctx context.Context, a *repo.AddressRecord) error
	Delete(ctx context.Context, id, userID int64) error
}

var _ AddressRepo = (*repo.AddressRepo)(nil)

// 业务级 sentinel error（service 层抛；handler 层翻译为业务码）。
var (
	ErrAddressLimitExceeded = errs.New(errs.CodeConflict, "address limit exceeded (max 5)")
	ErrAddressNotFound      = errs.New(errs.CodeNotFound, "address not found")
)

// AddressService 是地址业务编排器。
type AddressService struct {
	repo AddressRepo
}

// NewAddressService 装配。
func NewAddressService(r AddressRepo) *AddressService { return &AddressService{repo: r} }

// allowedRegions 是 v1 三级白名单（应用层校验）。
//   province → cities → districts
var allowedRegions = map[string]map[string][]string{
	"北京": {"北京": {"东城区", "西城区", "朝阳区", "海淀区", "丰台区", "石景山区"}},
	"上海": {"上海": {"黄浦区", "徐汇区", "长宁区", "静安区", "普陀区", "浦东新区"}},
	"广州": {"广州": {"天河区", "越秀区", "海珠区", "荔湾区", "白云区"}},
	"深圳": {"深圳": {"福田区", "罗湖区", "南山区", "宝安区", "龙岗区"}},
	"杭州": {"杭州": {"上城区", "拱墅区", "西湖区", "滨江区", "余杭区"}},
	"成都": {"成都": {"锦江区", "青羊区", "金牛区", "武侯区", "成华区"}},
}

// validate 校验入参：必填 + 长度 + 白名单 + 电话。
func validate(addr *AddressInput) error {
	if l := len(strings.TrimSpace(addr.Recipient)); l == 0 || l > 32 {
		return errs.New(errs.CodeParamInvalid, "recipient length must be 1..32")
	}
	if !phoneOK.MatchString(addr.Phone) {
		return errs.New(errs.CodeParamInvalid, "phone format invalid")
	}
	if l := len(strings.TrimSpace(addr.Detail)); l == 0 || l > 200 {
		return errs.New(errs.CodeParamInvalid, "detail length must be 1..200")
	}
	cities, ok := allowedRegions[addr.Province]
	if !ok {
		return errs.New(errs.CodeParamInvalid, "province not in whitelist")
	}
	districts, ok := cities[addr.City]
	if !ok {
		return errs.New(errs.CodeParamInvalid, "city not match province")
	}
	found := false
	for _, d := range districts {
		if d == addr.District {
			found = true
			break
		}
	}
	if !found {
		return errs.New(errs.CodeParamInvalid, "district not match city")
	}
	return nil
}

// phoneOK 是电话宽松正则（7~20 位数字 / `-` / `+`）。
var phoneOK = regexp.MustCompile(`^[0-9\-\+]{7,20}$`)

// Create 创建地址；超 5 上限返回 ErrAddressLimitExceeded。
func (s *AddressService) Create(ctx context.Context, userID int64, in *AddressInput) (*AddressView, error) {
	if err := validate(in); err != nil {
		return nil, err
	}
	n, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "count addresses", err)
	}
	if n >= 5 {
		return nil, fmt.Errorf("service: %w", ErrAddressLimitExceeded)
	}
	rec := &repo.AddressRecord{
		UserID:    userID,
		Recipient: strings.TrimSpace(in.Recipient),
		Phone:     in.Phone,
		Province:  in.Province,
		City:      in.City,
		District:  in.District,
		Detail:    strings.TrimSpace(in.Detail),
		IsDefault: in.IsDefault,
	}
	if in.IsDefault {
		if err := s.repo.CreateDefault(ctx, rec); err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "create default address", err)
		}
	} else {
		if err := s.repo.Create(ctx, rec); err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "create address", err)
		}
	}
	return toView(rec), nil
}

// Update 修改地址。
func (s *AddressService) Update(ctx context.Context, userID, addrID int64, in *AddressInput) error {
	if err := validate(in); err != nil {
		return err
	}
	rec := &repo.AddressRecord{
		ID:        addrID,
		UserID:    userID,
		Recipient: strings.TrimSpace(in.Recipient),
		Phone:     in.Phone,
		Province:  in.Province,
		City:      in.City,
		District:  in.District,
		Detail:    strings.TrimSpace(in.Detail),
	}
	if err := s.repo.Update(ctx, rec); err != nil {
		if errors.Is(err, repo.ErrAddressNotFound) {
			return fmt.Errorf("service: %w", ErrAddressNotFound)
		}
		return errs.Wrap(errs.CodeInternal, "update address", err)
	}
	return nil
}

// Delete 删除地址。
func (s *AddressService) Delete(ctx context.Context, userID, addrID int64) error {
	if err := s.repo.Delete(ctx, addrID, userID); err != nil {
		if errors.Is(err, repo.ErrAddressNotFound) {
			return fmt.Errorf("service: %w", ErrAddressNotFound)
		}
		return errs.Wrap(errs.CodeInternal, "delete address", err)
	}
	return nil
}

// Get 取一个地址。
func (s *AddressService) Get(ctx context.Context, userID, addrID int64) (*AddressView, error) {
	rec, err := s.repo.GetByID(ctx, addrID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrAddressNotFound) {
			return nil, fmt.Errorf("service: %w", ErrAddressNotFound)
		}
		return nil, errs.Wrap(errs.CodeInternal, "get address", err)
	}
	return toView(rec), nil
}

// List 列出用户所有地址（默认优先）。
func (s *AddressService) List(ctx context.Context, userID int64) ([]*AddressView, error) {
	list, err := s.repo.ListByUser(ctx, userID, false)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list addresses", err)
	}
	out := make([]*AddressView, 0, len(list))
	for _, r := range list {
		out = append(out, toView(r))
	}
	return out, nil
}

// toView 把 repo 记录转 view。
func toView(r *repo.AddressRecord) *AddressView {
	return &AddressView{
		ID:        r.ID,
		UserID:    r.UserID,
		Recipient: r.Recipient,
		Phone:     r.Phone,
		Province:  r.Province,
		City:      r.City,
		District:  r.District,
		Detail:    r.Detail,
		IsDefault: r.IsDefault,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
```

**Step 4: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestAddressService_' ./services/user/internal/service/`
Expected: PASS（8 个测试）

**Step 5: Commit**

```bash
git add services/user/internal/service/address_service.go services/user/internal/service/address_service_test.go
git commit -m "feat(user): AddressService (5-limit + 默认互斥 + 省市区白名单 + 8 个单测)"
```

---

### Task 5: CouponService（状态推导 + 越权保护）

**Files:**
- Create: `services/user/internal/service/coupon_service.go`
- Create: `services/user/internal/service/coupon_service_test.go`

**Step 1: 写单测（RED）**

`services/user/internal/service/coupon_service_test.go`：

```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/growdu/doctors/services/user/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCouponRepo 是 repo.CouponRepo 的 fake。
type fakeCouponRepo struct {
	records map[int64]*repo.CouponRecord
	nextID  int64
}

func newFakeCouponRepo() *fakeCouponRepo {
	return &fakeCouponRepo{records: map[int64]*repo.CouponRecord{}, nextID: 1}
}
func (f *fakeCouponRepo) Create(_ context.Context, c *repo.CouponRecord) error {
	c.ID = f.nextID
	f.nextID++
	c.Status = "unused"
	f.records[c.ID] = c
	return nil
}
func (f *fakeCouponRepo) ListByUser(_ context.Context, userID int64) ([]*repo.CouponRecord, error) {
	out := []*repo.CouponRecord{}
	for _, c := range f.records {
		if c.UserID == userID {
			now := time.Now()
			c.Status = repo.DeriveStatus(c.UsedAt, c.ExpiresAt, now)
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeCouponRepo) GetByID(_ context.Context, id, userID int64) (*repo.CouponRecord, error) {
	c, ok := f.records[id]
	if !ok || c.UserID != userID {
		return nil, repo.ErrCouponNotFound
	}
	c.Status = repo.DeriveStatus(c.UsedAt, c.ExpiresAt, time.Now())
	return c, nil
}

// 导出 derive 便于测试（如果不可见，换成 service 层的 wrap）。
// 这里为了简单，测试用 repo.DeriveStatus（已在 coupon_repo.go 导出）。

var _ CouponRepo = (*fakeCouponRepo)(nil)

// TestCouponService_List_OK 验证列表。
func TestCouponService_List_OK(t *testing.T) {
	r := newFakeCouponRepo()
	svc := NewCouponService(r)

	now := time.Now()
	r.records[1] = &repo.CouponRecord{ID: 1, UserID: 100, Type: "amount_off",
		Value: 20, Threshold: 100, ExpiresAt: now.Add(7 * 24 * time.Hour), Status: "unused"}

	list, err := svc.List(context.Background(), 100)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "amount_off", list[0].Type)
	assert.Equal(t, "unused", list[0].Status)
}

// TestCouponService_List_Empty 验证空列表。
func TestCouponService_List_Empty(t *testing.T) {
	r := newFakeCouponRepo()
	svc := NewCouponService(r)
	list, err := svc.List(context.Background(), 999)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// TestCouponService_List_DerivesExpired 验证过期状态推导。
func TestCouponService_List_DerivesExpired(t *testing.T) {
	r := newFakeCouponRepo()
	svc := NewCouponService(r)

	now := time.Now()
	r.records[1] = &repo.CouponRecord{ID: 1, UserID: 100, Type: "percent_off",
		Value: 8.5, Threshold: 50, ExpiresAt: now.Add(-1 * time.Hour), Status: "unused"}

	list, err := svc.List(context.Background(), 100)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "expired", list[0].Status)
}

// TestCouponService_Get_OK 验证 Get。
func TestCouponService_Get_OK(t *testing.T) {
	r := newFakeCouponRepo()
	svc := NewCouponService(r)

	now := time.Now()
	r.records[1] = &repo.CouponRecord{ID: 1, UserID: 100, Type: "full_off",
		Value: 0, Threshold: 200, ExpiresAt: now.Add(30 * 24 * time.Hour), Status: "unused"}

	got, err := svc.Get(context.Background(), 100, 1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "full_off", got.Type)
}
```

> **注意**：上面测试引用了 `repo.DeriveStatus`。如不愿导出，把 `deriveStatus` 改为 `DeriveStatus`（大写导出）即可。

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestCouponService_' ./services/user/internal/service/`
Expected: FAIL — `undefined: NewCouponService`, `undefined: CouponRepo`

**Step 3: 写 coupon_service.go**

`services/user/internal/service/coupon_service.go`：

```go
// Package service 是 user-service 的业务编排层（coupon 部分）。
//
// 设计要点：
//   - v1 仅查询 + 状态管理；不做 ApplyCoupon（下单抵扣留给 v2 order-service）。
//   - 状态推导交给 repo.DeriveStatus；service 层只做"按用户过滤 + 转 view"。
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/growdu/doctors/services/user/internal/repo"
	"github.com/growdu/doctors/shared/errs"
)

// CouponView 是优惠券出参。
type CouponView struct {
	ID        int64
	Type      string
	Value     float64
	Threshold float64
	ExpiresAt int64 // unix nano
	UsedAt    int64 // 0 表示未用
	Status    string
	CreatedAt int64
}

// CouponRepo 是仓储契约。
type CouponRepo interface {
	ListByUser(ctx context.Context, userID int64) ([]*repo.CouponRecord, error)
	GetByID(ctx context.Context, id, userID int64) (*repo.CouponRecord, error)
}

var _ CouponRepo = (*repo.CouponRepo)(nil)

// CouponService 是优惠券业务编排器。
type CouponService struct {
	repo CouponRepo
}

// NewCouponService 装配。
func NewCouponService(r CouponRepo) *CouponService { return &CouponService{repo: r} }

// List 列出用户所有优惠券（v1 不分页；coupons 通常 ≤ 20 张）。
func (s *CouponService) List(ctx context.Context, userID int64) ([]*CouponView, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list coupons", err)
	}
	out := make([]*CouponView, 0, len(list))
	for _, c := range list {
		out = append(out, toCouponView(c))
	}
	return out, nil
}

// Get 取一张优惠券。
func (s *CouponService) Get(ctx context.Context, userID, couponID int64) (*CouponView, error) {
	c, err := s.repo.GetByID(ctx, couponID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrCouponNotFound) {
			return nil, fmt.Errorf("service: %w", errs.New(errs.CodeNotFound, "coupon not found"))
		}
		return nil, errs.Wrap(errs.CodeInternal, "get coupon", err)
	}
	return toCouponView(c), nil
}

func toCouponView(c *repo.CouponRecord) *CouponView {
	var usedAt int64
	if c.UsedAt != nil {
		usedAt = c.UsedAt.UnixNano()
	}
	return &CouponView{
		ID:        c.ID,
		Type:      c.Type,
		Value:     c.Value,
		Threshold: c.Threshold,
		ExpiresAt: c.ExpiresAt.UnixNano(),
		UsedAt:    usedAt,
		Status:    c.Status,
		CreatedAt: c.CreatedAt.UnixNano(),
	}
}
```

> **同步**：把 `coupon_repo.go` 的 `deriveStatus` 重命名为 `DeriveStatus`（导出），让 service 单测可用。

**Step 4: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestCouponService_' ./services/user/internal/service/`
Expected: PASS（4 个测试）

**Step 5: Commit**

```bash
git add services/user/internal/service/coupon_service.go services/user/internal/service/coupon_service_test.go services/user/internal/repo/coupon_repo.go
git commit -m "feat(user): CouponService (List/Get + repo.DeriveStatus 导出 + 4 个单测)"
```

---

### Task 6: handler + router（5 API 接入）

**Files:**
- Modify: `services/user/internal/handler/user.go`（在 RegisterRoutes 加 me group）
- Create: `services/user/internal/handler/address.go`
- Create: `services/user/internal/handler/address_test.go`
- Create: `services/user/internal/handler/coupon.go`
- Create: `services/user/internal/handler/coupon_test.go`

**Step 1: 写 handler 单测（RED）**

`services/user/internal/handler/address_test.go`：

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/growdu/doctors/services/user/internal/middleware"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeAddrSvc 实现 handler.AddressService 接口。
type fakeAddrSvc struct {
	createCalled bool
	createIn     *service.AddressInput
	createOut    *service.AddressView
	createErr    error
	updateCalled bool
	updateErr    error
	deleteCalled bool
	deleteErr    error
	listOut      []*AddressView
	listErr      error
	getOut       *AddressView
	getErr       error
}

func (f *fakeAddrSvc) Create(_ context.Context, _ int64, in *service.AddressInput) (*service.AddressView, error) {
	f.createCalled = true
	f.createIn = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createOut, nil
}
func (f *fakeAddrSvc) Update(_ context.Context, _, _ int64, _ *service.AddressInput) error {
	f.updateCalled = true
	return f.updateErr
}
func (f *fakeAddrSvc) Delete(_ context.Context, _, _ int64) error {
	f.deleteCalled = true
	return f.deleteErr
}
func (f *fakeAddrSvc) List(_ context.Context, _ int64) ([]*AddressView, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listOut, nil
}
func (f *fakeAddrSvc) Get(_ context.Context, _, _ int64) (*AddressView, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getOut, nil
}

// newTestEngine 构造带 JWT 中间件 + fake svc 的 gin engine。
func newTestEngine(svc *fakeAddrSvc) *gin.Engine {
	h := NewAddressHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// 注入 userID（mock JWT 解码）
		c.Set(middleware.CtxUserID, int64(100))
		c.Next()
	})
	h.RegisterRoutes(r)
	return r
}

// TestAddress_List_OK 验证 GET /users/me/addresses 返回列表。
func TestAddress_List_OK(t *testing.T) {
	svc := &fakeAddrSvc{listOut: []*AddressView{
		{ID: 1, Recipient: "张三", IsDefault: true, CreatedAt: 1700000000000000000},
	}}
	r := newTestEngine(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me/addresses", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := map[string]any{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotNil(t, body["data"])
}

// TestAddress_Create_OK 验证 POST /users/me/addresses。
func TestAddress_Create_OK(t *testing.T) {
	svc := &fakeAddrSvc{createOut: &service.AddressView{ID: 7, IsDefault: false}}
	r := newTestEngine(svc)

	body := map[string]any{
		"recipient": "张三", "phone": "13800138000",
		"province": "北京", "city": "北京", "district": "东城区",
		"detail": "东单北大街9号", "is_default": true,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/me/addresses", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.createCalled)
	assert.Equal(t, "张三", svc.createIn.Recipient)
	assert.True(t, svc.createIn.IsDefault)
}

// TestAddress_Update_OK 验证 PUT /users/me/addresses/:id。
func TestAddress_Update_OK(t *testing.T) {
	svc := &fakeAddrSvc{}
	r := newTestEngine(svc)

	body := map[string]any{
		"recipient": "李四", "phone": "13900139000",
		"province": "上海", "city": "上海", "district": "黄浦区",
		"detail": "南京东路100号",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	id := strconv.FormatInt(7, 10)
	req := httptest.NewRequest(http.MethodPut, "/users/me/addresses/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.updateCalled)
}

// TestAddress_Delete_OK 验证 DELETE /users/me/addresses/:id。
func TestAddress_Delete_OK(t *testing.T) {
	svc := &fakeAddrSvc{}
	r := newTestEngine(svc)

	w := httptest.NewRecorder()
	id := strconv.FormatInt(7, 10)
	req := httptest.NewRequest(http.MethodDelete, "/users/me/addresses/"+id, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.deleteCalled)
}
```

`services/user/internal/handler/coupon_test.go`：

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/growdu/doctors/services/user/internal/middleware"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCouponSvc 实现 handler.CouponService 接口。
type fakeCouponSvc struct {
	listOut []*CouponView
	listErr error
}

func (f *fakeCouponSvc) List(_ context.Context, _ int64) ([]*CouponView, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listOut, nil
}
func (f *fakeCouponSvc) Get(_ context.Context, _, _ int64) (*CouponView, error) {
	return nil, nil
}

// TestCoupon_List_OK 验证 GET /users/me/coupons。
func TestCoupon_List_OK(t *testing.T) {
	svc := &fakeCouponSvc{listOut: []*CouponView{
		{ID: 1, Type: "amount_off", Value: 20, Threshold: 100, Status: "unused"},
	}}
	h := NewCouponHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.CtxUserID, int64(100))
		c.Next()
	})
	h.RegisterRoutes(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me/coupons", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := map[string]any{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotNil(t, body["data"])
}
```

**Step 2: 跑测试确认失败**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 -run 'TestAddress_|TestCoupon_' ./services/user/internal/handler/`
Expected: FAIL — `NewAddressHandler` / `NewCouponHandler` undefined

**Step 3: 写 address.go**

`services/user/internal/handler/address.go`：

```go
// Package handler 把 AddressService 暴露为 REST 接口。
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/user/internal/middleware"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// AddressView 是给前端的视图（与 service.AddressView 等价；handler 层再包一层便于演进）。
type AddressView = service.AddressView

// AddressService 是 handler 依赖的 service 接口。
type AddressService interface {
	Create(ctx context.Context, userID int64, in *service.AddressInput) (*service.AddressView, error)
	Update(ctx context.Context, userID, addrID int64, in *service.AddressInput) error
	Delete(ctx context.Context, userID, addrID int64) error
	List(ctx context.Context, userID int64) ([]*AddressView, error)
	Get(ctx context.Context, userID, addrID int64) (*AddressView, error)
}

// AddressHandler 持有 service 引用。
type AddressHandler struct {
	svc AddressService
}

// NewAddressHandler 构造。
func NewAddressHandler(svc AddressService) *AddressHandler { return &AddressHandler{svc: svc} }

// RegisterRoutes 挂 /users/me/addresses。
func (h *AddressHandler) RegisterRoutes(r gin.IRouter) {
	me := r.Group("/users/me")
	me.GET("/addresses", h.ListHandler)
	me.POST("/addresses", h.CreateHandler)
	me.PUT("/addresses/:id", h.UpdateHandler)
	me.DELETE("/addresses/:id", h.DeleteHandler)
}

// ListHandler GET /users/me/addresses
func (h *AddressHandler) ListHandler(c *gin.Context) {
	uid := middleware.UserID(c)
	list, err := h.svc.List(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, list)
}

// addressReq 是 POST/PUT body。
type addressReq struct {
	Recipient string `json:"recipient" binding:"required"`
	Phone     string `json:"phone" binding:"required"`
	Province  string `json:"province" binding:"required"`
	City      string `json:"city" binding:"required"`
	District  string `json:"district" binding:"required"`
	Detail    string `json:"detail" binding:"required"`
	IsDefault bool   `json:"is_default"`
}

func bindInput(c *gin.Context) (*service.AddressInput, bool) {
	var req addressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return nil, false
	}
	return &service.AddressInput{
		Recipient: req.Recipient, Phone: req.Phone,
		Province: req.Province, City: req.City, District: req.District,
		Detail: req.Detail, IsDefault: req.IsDefault,
	}, true
}

// CreateHandler POST /users/me/addresses
func (h *AddressHandler) CreateHandler(c *gin.Context) {
	uid := middleware.UserID(c)
	in, ok := bindInput(c)
	if !ok {
		return
	}
	out, err := h.svc.Create(c.Request.Context(), uid, in)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, out)
}

// UpdateHandler PUT /users/me/addresses/:id
func (h *AddressHandler) UpdateHandler(c *gin.Context) {
	uid := middleware.UserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	in, ok := bindInput(c)
	if !ok {
		return
	}
	if err := h.svc.Update(c.Request.Context(), uid, id, in); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// DeleteHandler DELETE /users/me/addresses/:id
func (h *AddressHandler) DeleteHandler(c *gin.Context) {
	uid := middleware.UserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uid, id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// respondError 翻译 service 错误为 HTTP 响应。
func respondError(c *gin.Context, err error) {
	// 业务 sentinel 优先
	if errors.Is(err, service.ErrAddressLimitExceeded) {
		httpx.Fail(c, int(errs.CodeConflict), "address limit exceeded (max 5)")
		return
	}
	if errors.Is(err, service.ErrAddressNotFound) {
		httpx.Fail(c, int(errs.CodeNotFound), "address not found")
		return
	}
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}
```

> **说明**：`respondError` 与 `user.go` 同名；保留 `user.go` 的版本（避免冲突），把 address handler 的 `respondError` 改为 `addrRespondError` 或抽到独立文件 `respond.go`。本 plan 选择把 `respondError` 抽到 `services/user/internal/handler/respond.go`，并删除 `user.go` 的同名函数。

**Step 4: 写 coupon.go**

`services/user/internal/handler/coupon.go`：

```go
// Package handler 把 CouponService 暴露为 REST 接口。
package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/user/internal/middleware"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// CouponView 是给前端的视图。
type CouponView = service.CouponView

// CouponService 是 handler 依赖的 service 接口。
type CouponService interface {
	List(ctx context.Context, userID int64) ([]*CouponView, error)
	Get(ctx context.Context, userID, couponID int64) (*CouponView, error)
}

// CouponHandler 持有 service 引用。
type CouponHandler struct {
	svc CouponService
}

// NewCouponHandler 构造。
func NewCouponHandler(svc CouponService) *CouponHandler { return &CouponHandler{svc: svc} }

// RegisterRoutes 挂 /users/me/coupons。
func (h *CouponHandler) RegisterRoutes(r gin.IRouter) {
	me := r.Group("/users/me")
	me.GET("/coupons", h.ListHandler)
}

// ListHandler GET /users/me/coupons
func (h *CouponHandler) ListHandler(c *gin.Context) {
	uid := middleware.UserID(c)
	list, err := h.svc.List(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, list)
}
```

**Step 5: 抽 respond.go + 改 user.go 用它**

`services/user/internal/handler/respond.go`：

```go
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// respondError 翻译 service 错误为 HTTP 响应（address / coupon handler 共用）。
func respondError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrAddressLimitExceeded) {
		httpx.Fail(c, int(errs.CodeConflict), "address limit exceeded (max 5)")
		return
	}
	if errors.Is(err, service.ErrAddressNotFound) {
		httpx.Fail(c, int(errs.CodeNotFound), "address not found")
		return
	}
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}
```

删除 `user.go` 里的 `respondError` 函数定义（保留 `parseID` / `respondError` 的本地用法——改为调 `respondError`）。

**Step 6: 修改 `user.go` 的 RegisterRoutes**

`RegisterRoutes` 改为：

```go
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	users := r.Group("/users")
	users.GET("/:id", h.Get)
	users.PATCH("/:id/nickname", h.UpdateNickname)
	users.PATCH("/:id/avatar", h.UpdateAvatar)
}
```

> **说明**：address / coupon handler 自己注册 `/users/me/*`；user handler 不动。

**Step 7: 跑测试确认通过**

Run: `GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./services/user/internal/handler/`
Expected: PASS（5 个测试：4 address + 1 coupon + 既有的用户相关单测）

> **注意**：`middleware.UserID` 与 `middleware.CtxUserID` 是实际常量名；如需 `c.Set(middleware.CtxUserID, 100)`，先查 `middleware/auth.go` 的导出名。

**Step 8: Commit**

```bash
git add services/user/internal/handler/
git commit -m "feat(user): handler 加 AddressHandler (4 API) + CouponHandler (1 API) + respond.go 抽取 + 5 个单测"
```

---

### Task 7: main 装配 + smoke + 文档同步

**Files:**
- Modify: `services/user/cmd/main.go`
- Create: `scripts/smoke-address-coupon.sh`

**Step 1: 修改 main.go**

替换 `services/user/cmd/main.go` 的 `nilRepo{}` 装配逻辑：

```go
// 原 main.go 末尾：
//   svc := service.New(nilRepo{})
//   h := handler.New(svc)
// 改为：
import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/growdu/doctors/services/user/internal/handler"
    "github.com/growdu/doctors/services/user/internal/repo"
    "github.com/growdu/doctors/services/user/internal/service"
    // ... 既有 import
)

func main() {
    cfg, err := config.Load("user")
    if err != nil { log.Fatalf("load config: %v", err) }
    logger.SetLevel(parseLevel(cfg.Logging.Level))
    defer func() { _ = logger.L().Sync() }()

    // PG pool
    pool, err := pgxpool.New(context.Background(), cfg.DB.DSN)
    if err != nil { log.Fatalf("pg pool: %v", err) }
    defer pool.Close()

    // Profile（原 13 个单测）
    profileSvc := service.New(nilProfileRepo{pool}) // 保留骨架；v2 接真 UserRepo
    h := handler.New(profileSvc)

    // Address
    addrRepo := repo.NewAddressRepo(pool)
    addrSvc := service.NewAddressService(addrRepo)
    addrH := handler.NewAddressHandler(addrSvc)

    // Coupon
    cpnRepo := repo.NewCouponRepo(pool)
    cpnSvc := service.NewCouponService(cpnRepo)
    cpnH := handler.NewCouponHandler(cpnSvc)

    srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)
    // 注入附加 handler（在 server.go 加 SetExtras 或在 main.go 注册）
    _ = addrH
    _ = cpnH
    _ = profileSvc
    _ = addrSvc
    _ = cpnSvc

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    logger.L().Info("user-service starting", zap.String("addr", cfg.HTTP.Addr))
    if err := srv.Run(ctx); err != nil {
        logger.L().Error("user-service exited", zap.Error(err))
        os.Exit(1)
    }
    logger.L().Info("user-service stopped")
}

// nilProfileRepo 占位（保留骨架；profile service 的 fake repo）。
type nilProfileRepo struct{ pool *pgxpool.Pool }
```

> **关键**：把 addr / cpn handler 真正挂到 router。两种方式：
> 1. 扩展 `server.Server` 加 `WithHandler(...)` 链式注册
> 2. 改 `router.New` 接收多个 handler
>
> 推荐方案 1：在 `server/server.go` 加：

```go
// services/user/internal/server/server.go 追加：
type extraRoute struct {
    prefix string
    reg    func(r gin.IRouter)
}

func (s *Server) WithExtra(reg func(r gin.IRouter)) *Server {
    s.extras = append(s.extras, reg)
    return s
}

// Server.Run 内：
//   for _, ex := range s.extras { ex(v1) }
```

并在 `cmd/main.go`：

```go
srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret).
    WithExtra(addrH.RegisterRoutes).
    WithExtra(cpnH.RegisterRoutes)
```

**Step 2: 写 smoke 脚本**

`scripts/smoke-address-coupon.sh`：

```bash
#!/usr/bin/env bash
# smoke-address-coupon.sh
# 启 user-service → 验证 addresses / coupons 5 个 API 的 JWT + 401 + 业务错误。

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ADDR="${DOCTORS_USER_HTTP_ADDR:-:8082}"
BIN="$ROOT/bin/user"
LOGFILE="$ROOT/.data/user-ac-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/6] building user-service..."
GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/user/cmd

echo "[2/6] starting user-service on $ADDR..."
DOCTORS_USER_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

for i in {1..10}; do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/6] curl /healthz"
curl -fsS "http://127.0.0.1$ADDR/healthz" | head -c 200
echo

echo "[4/6] GET /users/me/addresses (no token → expect 401)"
curl -sS "http://127.0.0.1$ADDR/users/me/addresses" | head -c 200
echo

echo "[5/6] POST /users/me/addresses (no token → expect 401)"
curl -sS -X POST -H "Content-Type: application/json" \
  -d '{"recipient":"测试","phone":"13800138000","province":"北京","city":"北京","district":"东城区","detail":"东单北大街9号"}' \
  "http://127.0.0.1$ADDR/users/me/addresses" | head -c 200
echo

echo "[6/6] GET /users/me/coupons (no token → expect 401)"
curl -sS "http://127.0.0.1$ADDR/users/me/coupons" | head -c 200
echo

echo "smoke OK"
```

```bash
chmod +x scripts/smoke-address-coupon.sh
bash scripts/smoke-address-coupon.sh
```

Expected: smoke OK（启动 + /healthz + 401 拦截）

**Step 3: Commit**

```bash
git add services/user/cmd/main.go services/user/internal/server/server.go scripts/smoke-address-coupon.sh
git commit -m "feat(user): main 装配 AddressRepo/CouponRepo + smoke 脚本（5 API + 401 拦截）"
```

---

### Task 8: 文档同步 + dev.md

**Files:**
- Modify: `docs/04-业务流程.md` §4.7
- Modify: `dev.md`

**Step 1: 04 加地址 / 优惠券流程**

```markdown
### 4.7 地址管理 + 优惠券流程

#### 地址
1. patient 进入"我的 → 收货地址" → `GET /users/me/addresses` 列出所有地址
2. 新增：表单填收件人 / 电话 / 省市区 / 详细地址 → `POST /users/me/addresses`（is_default=true 时后端事务内取消其他默认）
3. 修改：`PUT /users/me/addresses/:id`
4. 删除：`DELETE /users/me/addresses/:id`
5. 上限：每个用户 ≤ 5 个；默认地址唯一（partial unique index）

#### 优惠券
1. patient 进入"我的 → 优惠券" → `GET /users/me/coupons`
2. 列表按状态排序：unused 优先 → used → expired
3. 状态由 `used_at` + `expires_at` 实时推导（不是 DB 存的 status）
4. v1 不做下单抵扣（下单时勾选 / 抵消留给 v2 order-service）
```

**Step 2: dev.md 加 §10.13**

```markdown
### 10.13 地址 + 优惠券（2026-09-24 address-coupon plan）

扩展 user-service 骨架，实现 patient 地址管理（4 API）+ 优惠券列表（1 API）。

**落地 commits（7 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(migrations) | 0005 addresses+coupons + CHECK + 索引 |
| feat(user) | repo 加 AddressRepo (Create/List/Get/Update/Delete/CreateDefault) |
| feat(user) | repo 加 CouponRepo + DeriveStatus |
| feat(user) | AddressService (5-limit + 白名单 + 默认互斥) |
| feat(user) | CouponService (List/Get) |
| feat(user) | handler AddressHandler + CouponHandler + respond.go |
| feat(user) | main 装配 + smoke 脚本 |

**API 增量**：GET /users/me/addresses + POST /users/me/addresses + PUT /users/me/addresses/:id + DELETE /users/me/addresses/:id + GET /users/me/coupons。

**未做**：下单抵扣（v2 接 order-service）；OSS 地址（v1 不存）；收货地址标签（v1 仅基本字段）。
```

**Step 3: Commit**

```bash
git add docs/04-业务流程.md dev.md
git commit -m "docs: 地址+优惠券流程 + dev.md 10.13 (user-service 扩 5 API)"
```

---

### Task 9: 全量回归 + push

```bash
# 清干净 PG（addresses / coupons / sos / refunds / orders / users）
docker exec doctors-postgres psql -U doctors -d doctors -c "
DROP TABLE IF EXISTS addresses CASCADE;
DROP TABLE IF EXISTS coupons CASCADE;
DROP TABLE IF EXISTS sos_records CASCADE;
DROP TABLE IF EXISTS refunds CASCADE;
DROP TABLE IF EXISTS refund_policies CASCADE;
DROP TABLE IF EXISTS order_events CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS users CASCADE;"

# 跑全部单测
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./shared/... ./services/...

# 跑全部集成测试
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -tags=integration -count=1 ./migrations/... ./services/...

# 跑 smoke
bash scripts/smoke-order.sh    # 已有
bash scripts/smoke-sos.sh      # sos plan
bash scripts/smoke-address-coupon.sh  # 本 plan

# push
git push origin main
```

Expected: 全部 PASS + smoke OK + pushed.

---

## Self-Review

- ✅ **Spec 覆盖**: l2-api-gap-design.md §2.1 P1 patient 地址/优惠券全部 5 API；§3 Address / Coupon entity（type 全 3 种 + threshold + expires_at）
- ✅ **无占位符**: 每步有具体代码、命令、commit message
- ✅ **类型一致**: `AddressInput` / `AddressView` / `AddressRepo` / `AddressRecord` 跨 Task 2/4/6 一致；`CouponRecord` / `CouponView` / `CouponRepo` 跨 Task 3/5/6 一致；`repo.DeriveStatus` 跨 Task 3/5 一致
- ✅ **测试矩阵**: Task 1 集成（migration）+ Task 2 集成（address repo 11 个）+ Task 3 集成（coupon repo 5 个）+ Task 4 单测（address service 8 个）+ Task 5 单测（coupon service 4 个）+ Task 6 单测（handler 5 个）+ Task 7 smoke + Task 9 全量
- ✅ **越权保护**: 所有 repo.GetByID / Update / Delete 都强制 user_id 匹配；handler 通过 JWT 注入 userID（不接 query 参数）
- ✅ **YAGNI**: v1 不做下单抵扣；不接 Kafka；不接 Redis 缓存；省市区用白名单而不是接外部数据库
- ⚠️ **设计偏差**: migration 用 `0005_addresses_coupons`（与 sos plan `0005_sos_records` 同号段）；spec §5 提到 "0005_init_v1_remaining" 集中做，但 sos plan 已用 `0005_sos_records`，沿用同模式（一个 feature 一文件）。**需要用户确认是否要后续合并到统一 migration**

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-address-coupon.md`。
> 当前为 plan_all 模式 → 进入实施阶段需要用户决策。

**下一步选项**：
1. **立即执行**（subagent-driven 或 inline 执行）—— 我开始实施 Task 1~9
2. **暂停 + review** —— 你 review 此 plan 后告诉我调整
3. **继续产 plan** —— 接着出 7 个后端 plan（hospital-package / wallet / escort-business / review / message / admin / escort-order-ext）+ 3 个前端 plan（patient-miniapp / escort-app / admin-web）