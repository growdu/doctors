# 状态机统一 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `docs/REVIEW-REPORT.md` C-01（04 与 07 状态机不一致）：新增 `pending_acceptance` / `settling` / `disputed` 三个中间态 + 抢单锁单字段（`lock_owner` / `lock_expire_at`），同步 docs 与代码。

**Architecture:** 在 `services/order/internal/state/machine.go` 中以字符串转换表追加新状态；`migrations/0003_orders_state.up.sql` 加列 + CHECK 约束；`order_repo` 增加 `LockForAccept` / `ReleaseLock` 方法；`service.Accept` 走"状态机 → repo"两步而非一锅烩。**不做**：抢单锁单 30s 的 Redis SETNX 与超时任务（见 §4.2 order-lock plan）；本 plan 只落地状态机 + 字段 + repo 适配层。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · golang-migrate（迁移脚本）。**TDD 全程**：每个新状态先写失败测试，再加转换。

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

---

## File Structure

| 路径 | 变更 | 职责 |
|------|------|------|
| `services/order/internal/state/machine.go` | Modify | 状态枚举 + transitions map + IsValid / IsTerminal；新增 `pending_acceptance` / `settling` / `disputed` |
| `services/order/internal/state/machine_test.go` | Modify | 表驱动覆盖新状态合法 / 非法转换 |
| `migrations/0003_orders_state.up.sql` | Create | 加 `lock_owner BIGINT` / `lock_expire_at TIMESTAMPTZ` 列；CHECK 约束加入新状态；加 `idx_orders_lock` |
| `migrations/0003_orders_state.down.sql` | Create | 逆向：删索引、删列、收紧 CHECK |
| `migrations/migrations_test.go` | Modify | `Test0003OrdersStateUpDown` 验证新字段与 CHECK |
| `services/order/internal/repo/order_repo.go` | Modify | `Order` struct 加 `LockOwner *int64` / `LockExpireAt any`；`UpdateStatus` 接受 `lockOwner *int64` 与 `lockExpireAt *time.Time`；新增 `LockForAccept` / `ReleaseLock` / `LockExpired` 三个方法 |
| `services/order/internal/repo/order_repo_integration_test.go` | Modify | 加 LockForAccept / ReleaseLock / LockExpired 集成测试 |
| `services/order/internal/service/accept.go` | Modify | 改走两步：①SELECT FOR UPDATE SKIP LOCKED WHERE status IN ('matching', 'pending_acceptance') → ②UPDATE 到 'pending_acceptance' + 写 lock_owner/lock_expire_at（**锁单具体超时逻辑在 order-lock plan**） |
| `services/order/internal/service/accept_integration_test.go` | Modify | 加 `pending_acceptance` 路径测试 |
| `docs/04-业务流程.md` | Modify | 状态机图加入三个新态；锁单流程引用本 plan |
| `docs/07-数据模型.md` | Modify | orders 表加 lock_owner / lock_expire_at；CHECK 加入新态 |

---

### Task 1: 状态枚举与转换表（machine.go）

**Files:**
- Modify: `services/order/internal/state/machine.go`
- Test: `services/order/internal/state/machine_test.go`

**Step 1: 写三个新状态的失败测试**

在 `machine_test.go` 末尾追加：

```go
// TestStatusPendingAcceptance_Exists 验证新状态在枚举中。
func TestStatusPendingAcceptance_Exists(t *testing.T) {
    assert.True(t, state.IsValid(state.StatusPendingAcceptance))
}

// TestCanTransition_Matching_ToPendingAcceptance 验证 matching → pending_acceptance 合法。
func TestCanTransition_Matching_ToPendingAcceptance(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusMatching, state.StatusPendingAcceptance))
}

// TestCanTransition_PendingAcceptance_ToAccepted 验证陪诊师 30s 内确认 → accepted。
func TestCanTransition_PendingAcceptance_ToAccepted(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusPendingAcceptance, state.StatusAccepted))
}

// TestCanTransition_PendingAcceptance_ToMatching 验证 30s 超时回退 matching。
func TestCanTransition_PendingAcceptance_ToMatching(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusPendingAcceptance, state.StatusMatching))
}

// TestCanTransition_Completed_ToSettling 验证结算态。
func TestCanTransition_Completed_ToSettling(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusCompleted, state.StatusSettling))
}

// TestCanTransition_Disputed_FromAnyActive 验证争议可从活动态转入。
func TestCanTransition_Disputed_FromAnyActive(t *testing.T) {
    for _, s := range []state.Status{
        state.StatusAccepted, state.StatusInService, state.StatusCompleted,
    } {
        assert.True(t, state.CanTransition(s, state.StatusDisputed),
            "%s → disputed 应合法", s)
    }
}

// TestCanTransition_Settling_ToClosed 验证结算后关闭。
func TestCanTransition_Settling_ToClosed(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusSettling, state.StatusClosed))
}

// TestCanTransition_Refunding_ToSettling 验证退款完成进入结算。
func TestCanTransition_Refunding_ToSettling(t *testing.T) {
    assert.True(t, state.CanTransition(state.StatusRefunding, state.StatusSettling))
}
```

**Step 2: 跑测试确认失败**

Run: `go test -count=1 -run 'TestStatusPendingAcceptance|TestCanTransition_(Matching_ToPending|PendingAcceptance_ToAccepted|PendingAcceptance_ToMatching|Completed_ToSettling|Disputed_FromAnyActive|Settling_ToClosed|Refunding_ToSettling)' ./services/order/internal/state/`
Expected: FAIL — `undefined: state.StatusPendingAcceptance`

**Step 3: 在 machine.go 加三个常量 + transitions 边**

```go
// 在 Status const 块追加
StatusPendingAcceptance Status = "pending_acceptance"
StatusSettling         Status = "settling"
StatusDisputed         Status = "disputed"

// 在 transitions map 追加 / 调整：
var transitions = map[Status][]Status{
    StatusCreated:   {StatusPaid, StatusCanceled},
    StatusPaid:      {StatusMatching, StatusCanceled},
    StatusMatching:  {StatusPendingAcceptance, StatusAccepted, StatusCanceled},
    StatusPendingAcceptance: {StatusAccepted, StatusMatching, StatusCanceled}, // 30s 锁单 / 超时回退 / 拒接
    StatusAccepted:  {StatusInService, StatusMatching, StatusCanceled, StatusDisputed},
    StatusInService: {StatusCompleted, StatusDisputed},
    StatusCompleted: {StatusReviewed, StatusRefunding, StatusSettling, StatusDisputed},
    StatusReviewed:  {StatusClosed},
    StatusRefunding: {StatusRefunded, StatusSettling},
    StatusRefunded:  {StatusSettling},
    StatusSettling:  {StatusClosed},
    StatusDisputed:  {StatusCompleted, StatusRefunding, StatusClosed},
    StatusClosed:    {},
    StatusCanceled:  {},
}
```

**Step 4: 跑测试确认通过**

Run: `go test -count=1 ./services/order/internal/state/`
Expected: PASS（包含 Task 1 新增的 8 个 + 既有 4 个用例）

**Step 5: Commit**

```bash
git add services/order/internal/state/
git commit -m "feat(order): 状态机加 pending_acceptance/settling/disputed 三态"
```

---

### Task 2: 数据库迁移（migrations/0003_orders_state）

**Files:**
- Create: `migrations/0003_orders_state.up.sql`
- Create: `migrations/0003_orders_state.down.sql`
- Modify: `migrations/migrations_test.go`（追加 `Test0003OrdersStateUpDown`）

**Step 1: 写集成测试**

在 `migrations_test.go` 追加（参考既有 `Test0002OrdersUpDown` 风格）：

```go
// Test0003OrdersStateUpDown 验证 0003 加列 + CHECK + 索引。
// 依赖 0001_users + 0002_orders；测试结束回滚所有变更。
func Test0003OrdersStateUpDown(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    conn, err := pgx.Connect(ctx, dsn())
    require.NoError(t, err)
    defer conn.Close(ctx)

    // 先建 users + orders
    usersSQL, _ := os.ReadFile("0001_users.up.sql")
    _, err = conn.Exec(ctx, string(usersSQL))
    require.NoError(t, err)
    ordersSQL, _ := os.ReadFile("0002_orders.up.sql")
    _, err = conn.Exec(ctx, string(ordersSQL))
    require.NoError(t, err)
    t.Cleanup(func() {
        down, _ := os.ReadFile("0002_orders.down.sql")
        _, _ = conn.Exec(context.Background(), string(down))
        down, _ = os.ReadFile("0001_users.down.sql")
        _, _ = conn.Exec(context.Background(), string(down))
    })

    applyUp(t, "0003_orders_state.up.sql", []string{"orders"})

    // 列检查
    for _, col := range []string{"lock_owner", "lock_expire_at"} {
        var found bool
        err := conn.QueryRow(ctx,
            `SELECT EXISTS(SELECT 1 FROM information_schema.columns
                           WHERE table_name='orders' AND column_name=$1)`, col).
            Scan(&found)
        require.NoError(t, err)
        assert.True(t, found, "orders.%s should exist", col)
    }

    // 索引检查
    var idxExists bool
    err = conn.QueryRow(ctx,
        `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, "idx_orders_lock").
        Scan(&idxExists)
    require.NoError(t, err)
    assert.True(t, idxExists, "idx_orders_lock should exist")

    // CHECK 约束含新状态
    var hasCheck bool
    err = conn.QueryRow(ctx,
        `SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
                       WHERE constraint_name LIKE 'orders_status_check')`).
        Scan(&hasCheck)
    require.NoError(t, err)
    assert.True(t, hasCheck)

    applyDown(t, "0003_orders_state.down.sql", []string{"orders"})
}
```

**Step 2: 跑测试确认失败**

Run: `go test -tags=integration -run Test0003OrdersStateUpDown ./migrations/`
Expected: FAIL — `Test0003OrdersStateUpDown` 不存在 / 文件未创建

**Step 3: 写迁移文件 `migrations/0003_orders_state.up.sql`**

```sql
-- 0003_orders_state.up.sql
-- 状态机统一：加锁单字段 + CHECK 约束包含新状态。
-- 注意：这是兼容迁移；老数据 status 不在新 CHECK 列表里会失败，需先跑数据迁移脚本（生产单独排期）。

ALTER TABLE orders
  ADD COLUMN lock_owner BIGINT REFERENCES users(id),
  ADD COLUMN lock_expire_at TIMESTAMPTZ;

-- 索引服务于"查锁单即将到期"任务（§4.2 order-lock plan）。
CREATE INDEX idx_orders_lock ON orders(lock_expire_at)
  WHERE lock_owner IS NOT NULL;

-- 替换 CHECK 约束（Postgres 改 CHECK 约束要先 DROP 再 ADD）。
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','pending_acceptance','accepted','in_service',
  'completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));
```

**Step 4: 写 `migrations/0003_orders_state.down.sql`**

```sql
-- 0003_orders_state.down.sql
DROP INDEX IF EXISTS idx_orders_lock;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','accepted','in_service','completed',
  'reviewed','refunding','refunded','closed','canceled'
));
ALTER TABLE orders DROP COLUMN IF EXISTS lock_expire_at;
ALTER TABLE orders DROP COLUMN IF EXISTS lock_owner;
```

**Step 5: 跑测试确认通过（需 docker compose up）**

Run: `go test -tags=integration -run Test0003OrdersStateUpDown ./migrations/`
Expected: PASS

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0003_orders_state 加 lock_owner/lock_expire_at + CHECK 扩展"
```

---

### Task 3: order_repo 加 LockOwner / LockExpireAt 字段 + LockForAccept / ReleaseLock

**Files:**
- Modify: `services/order/internal/repo/order_repo.go`
- Modify: `services/order/internal/repo/order_repo_integration_test.go`

**Step 1: 写集成测试**

在 `order_repo_integration_test.go` 追加：

```go
func TestOrderRepo_LockForAccept_OK(t *testing.T) {
    pool := setupPool(t)
    patient := seedUser(t, pool, "13800139000", "patient")
    escort := seedUser(t, pool, "13800139001", "escort")
    r := NewOrderRepo(pool)

    o := sampleOrder(patient)
    require.NoError(t, r.Create(context.Background(), o))
    require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))

    err := r.LockForAccept(context.Background(), o.ID, escort, time.Now().Add(30*time.Second), 1)
    require.NoError(t, err)

    got, _ := r.FindByID(context.Background(), o.ID)
    assert.Equal(t, "pending_acceptance", got.Status)
    assert.Equal(t, escort, *got.LockOwner)
    assert.True(t, got.LockExpireAt.After(time.Now()))
}

func TestOrderRepo_LockForAccept_VersionMismatch(t *testing.T) {
    pool := setupPool(t)
    patient := seedUser(t, pool, "13800139002", "patient")
    r := NewOrderRepo(pool)
    o := sampleOrder(patient)
    require.NoError(t, r.Create(context.Background(), o))
    require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))

    // version 999 不匹配
    err := r.LockForAccept(context.Background(), o.ID, 1, time.Now(), 999)
    assert.ErrorIs(t, err, ErrVersionConflict)
}

func TestOrderRepo_LockForAccept_WrongStatus(t *testing.T) {
    pool := setupPool(t)
    patient := seedUser(t, pool, "13800139003", "patient")
    r := NewOrderRepo(pool)
    o := sampleOrder(patient) // status='created'
    require.NoError(t, r.Create(context.Background(), o))

    err := r.LockForAccept(context.Background(), o.ID, 1, time.Now(), 0)
    assert.ErrorIs(t, err, ErrInvalidStateForLock)
}

func TestOrderRepo_ReleaseLock_OK(t *testing.T) {
    pool := setupPool(t)
    patient := seedUser(t, pool, "13800139004", "patient")
    escort := seedUser(t, pool, "13800139005", "escort")
    r := NewOrderRepo(pool)
    o := sampleOrder(patient)
    require.NoError(t, r.Create(context.Background(), o))
    require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))
    require.NoError(t, r.LockForAccept(context.Background(), o.ID, escort, time.Now().Add(30*time.Second), 1))

    require.NoError(t, r.ReleaseLock(context.Background(), o.ID, 1))
    got, _ := r.FindByID(context.Background(), o.ID)
    assert.Equal(t, "matching", got.Status)
    assert.Nil(t, got.LockOwner)
    assert.Nil(t, got.LockExpireAt)
}

func TestOrderRepo_LockExpired_FindsExpiring(t *testing.T) {
    pool := setupPool(t)
    patient := seedUser(t, pool, "13800139006", "patient")
    escort := seedUser(t, pool, "13800139007", "escort")
    r := NewOrderRepo(pool)
    o := sampleOrder(patient)
    require.NoError(t, r.Create(context.Background(), o))
    require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))
    // 锁在过去 = 已过期
    require.NoError(t, r.LockForAccept(context.Background(), o.ID, escort, time.Now().Add(-1*time.Hour), 1))

    expired, err := r.LockExpired(context.Background(), time.Now(), 10)
    require.NoError(t, err)
    assert.Len(t, expired, 1)
    assert.Equal(t, o.ID, expired[0].ID)
}
```

**Step 2: 跑测试确认失败（编译错误）**

Run: `go test -tags=integration -run 'TestOrderRepo_LockForAccept|TestOrderRepo_ReleaseLock|TestOrderRepo_LockExpired' ./services/order/internal/repo/`
Expected: FAIL — `undefined: ErrInvalidStateForLock`

**Step 3: 在 order_repo.go 实现**

修改 `Order` struct：

```go
type Order struct {
    ID             int64
    OrderNo        string
    PatientID      int64
    EscortID       *int64
    HospitalID     int64
    PackageID      int64
    ServiceStartAt any  // 实际是 time.Time；保留 any 是为简化集成测试
    Amount         float64
    FinalAmount    float64
    Status         string
    Version        int
    LockOwner      *int64    // 新增
    LockExpireAt   any       // 新增（实际是 *time.Time）
}
```

修改 `baseSelect` 加 `lock_owner, lock_expire_at`；修改 `scanOne` / `scanRow` 加这两列。

新增错误：

```go
var ErrInvalidStateForLock = errors.New("repo: order not in matching state for lock")
```

新增方法：

```go
// LockForAccept 在 matching 状态下加锁单 + 状态切到 pending_acceptance。
// 同时校验 version；lockExpireAt 写入 orders 表。
func (r *OrderRepo) LockForAccept(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
    const q = `
        UPDATE orders
           SET status = 'pending_acceptance', lock_owner = $1, lock_expire_at = $2,
               version = version + 1, updated_at = NOW()
         WHERE id = $3 AND version = $4 AND status = 'matching' AND deleted_at IS NULL`
    tag, err := r.pool.Exec(ctx, q, escortID, expireAt, id, expectVersion)
    if err != nil {
        return fmt.Errorf("lock for accept: %w", err)
    }
    if tag.RowsAffected() == 0 {
        // 区分版本冲突与状态非法
        var curStatus string
        var curVer int
        _ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
        if curVer != expectVersion {
            return ErrVersionConflict
        }
        return ErrInvalidStateForLock
    }
    return nil
}

// ReleaseLock 把 pending_acceptance 回退到 matching；清除锁单字段。
func (r *OrderRepo) ReleaseLock(ctx context.Context, id int64, expectVersion int) error {
    const q = `
        UPDATE orders
           SET status = 'matching', lock_owner = NULL, lock_expire_at = NULL,
               version = version + 1, updated_at = NOW()
         WHERE id = $1 AND status = 'pending_acceptance' AND version = $2 AND deleted_at IS NULL`
    tag, err := r.pool.Exec(ctx, q, id, expectVersion)
    if err != nil {
        return fmt.Errorf("release lock: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return ErrVersionConflict
    }
    return nil
}

// LockExpired 返回已过期的锁单（limit by 排序）；给 §4.2 定时任务用。
func (r *OrderRepo) LockExpired(ctx context.Context, now time.Time, limit int) ([]*Order, error) {
    const q = baseSelect + ` WHERE status = 'pending_acceptance'
                                AND lock_expire_at IS NOT NULL AND lock_expire_at < $1
                                ORDER BY lock_expire_at ASC
                                LIMIT $2`
    rows, err := r.pool.Query(ctx, q, now, limit)
    if err != nil {
        return nil, fmt.Errorf("find expired locks: %w", err)
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
```

**Step 4: 跑测试确认通过**

Run: `go test -tags=integration -run 'TestOrderRepo_LockForAccept|TestOrderRepo_ReleaseLock|TestOrderRepo_LockExpired' ./services/order/internal/repo/`
Expected: PASS

**Step 5: Commit**

```bash
git add services/order/internal/repo/
git commit -m "feat(order): repo 加 LockForAccept / ReleaseLock / LockExpired 三方法"
```

---

### Task 4: service.Accept 走两步（matching → pending_acceptance → accepted）

**Files:**
- Modify: `services/order/internal/service/accept.go`
- Modify: `services/order/internal/service/accept_integration_test.go`

> **范围说明**：本 Task 只把现有 Accept 拆成"先 LockForAccept，再 ConfirmAccept（更新到 accepted）"。**锁单 30s 超时与 Redis SETNX 留给 §4.2 plan**。本 Task 落地的 `LockForAccept` 调用就是为了给 §4.2 留好钩子。

**Step 1: 写集成测试**

在 `accept_integration_test.go` 追加：

```go
// TestAccept_LockThenConfirm_OK 验证 LockForAccept 成功后再 confirm。
func TestAccept_LockThenConfirm_OK(t *testing.T) {
    pool, patient, escort := setupAcceptPool(t)
    orderID := seedOrder(t, pool, patient, "matching")
    r := repo.NewOrderRepo(pool)
    svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})

    // 第一步：陪诊师拿锁单
    err := svc.TryLock(context.Background(), orderID, escort, 30*time.Second)
    require.NoError(t, err)

    // 第二步：30s 内 confirm
    got, err := svc.ConfirmAccept(context.Background(), orderID, escort)
    require.NoError(t, err)
    assert.Equal(t, "accepted", got.Status)
}

// TestAccept_LockFailsOnConflict 验证另一个 escort 抢不到锁单。
func TestAccept_LockFailsOnConflict(t *testing.T) {
    pool, patient, escort1 := setupAcceptPool(t)
    escort2 := escort1 + 100
    orderID := seedOrder(t, pool, patient, "matching")
    r := repo.NewOrderRepo(pool)
    svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})

    require.NoError(t, svc.TryLock(context.Background(), orderID, escort1, 30*time.Second))
    err := svc.TryLock(context.Background(), orderID, escort2, 30*time.Second)
    assert.ErrorIs(t, err, ErrInvalidStateForLock, "已被锁单的订单不能被另一 escort 再锁")
}

// TestAccept_ReleaseLock 验证拒接后回退 matching。
func TestAccept_ReleaseLock(t *testing.T) {
    pool, patient, escort := setupAcceptPool(t)
    orderID := seedOrder(t, pool, patient, "matching")
    r := repo.NewOrderRepo(pool)
    svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})

    require.NoError(t, svc.TryLock(context.Background(), orderID, escort, 30*time.Second))
    require.NoError(t, svc.ReleaseAcceptLock(context.Background(), orderID, escort))

    got, _ := r.FindByID(context.Background(), orderID)
    assert.Equal(t, "matching", got.Status)
    assert.Nil(t, got.LockOwner)
}
```

**Step 2: 跑测试确认失败**

Run: `go test -tags=integration -run 'TestAccept_LockThenConfirm|TestAccept_LockFailsOnConflict|TestAccept_ReleaseLock' ./services/order/internal/service/`
Expected: FAIL — `undefined: service.TryLock`

**Step 3: 在 service 包定义新错误 + 三个新方法**

在 `accept.go` 顶部新增错误：

```go
var ErrInvalidStateForLock = repo.ErrInvalidStateForLock  // 直接复用 repo 的错误
```

在 service struct / New / WithTx 后面追加：

```go
// TryLock 陪诊师尝试拿锁单（§4.2 order-lock plan 会在此前后加 Redis SETNX）。
func (s *Service) TryLock(ctx context.Context, orderID, escortID int64, ttl time.Duration) error {
    if orderID == 0 || escortID == 0 {
        return errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
    }
    o, err := s.orders.FindByID(ctx, orderID)
    if err != nil {
        if err == repo.ErrOrderNotFound {
            return errs.New(errs.CodeNotFound, "order not found")
        }
        return errs.Wrap(errs.CodeInternal, "find order", err)
    }
    if o.Status != string(state.StatusMatching) {
        return errs.New(errs.CodeConflict, fmt.Sprintf("cannot lock from status %s", o.Status))
    }
    expireAt := s.clockNow().Add(ttl)
    if err := s.orders.LockForAccept(ctx, orderID, escortID, expireAt, o.Version); err != nil {
        return errs.Wrap(errs.CodeInternal, "lock for accept", err)
    }
    return nil
}

// ReleaseAcceptLock 拒接 / 超时 → 回退到 matching。
func (s *Service) ReleaseAcceptLock(ctx context.Context, orderID, escortID int64) error {
    o, err := s.orders.FindByID(ctx, orderID)
    if err != nil {
        return errs.Wrap(errs.CodeInternal, "find order", err)
    }
    if o.Status != string(state.StatusPendingAcceptance) {
        return errs.New(errs.CodeConflict, "order not in pending_acceptance")
    }
    if err := s.orders.ReleaseLock(ctx, orderID, o.Version); err != nil {
        return errs.Wrap(errs.CodeInternal, "release lock", err)
    }
    return nil
}

// ConfirmAccept 陪诊师在锁单窗口内确认 → 改 accepted；escort_id 写入。
func (s *Service) ConfirmAccept(ctx context.Context, orderID, escortID int64) (*repo.Order, error) {
    o, err := s.orders.FindByID(ctx, orderID)
    if err != nil {
        return nil, errs.Wrap(errs.CodeInternal, "find order", err)
    }
    if o.Status != string(state.StatusPendingAcceptance) {
        return nil, errs.New(errs.CodeConflict, "order not in pending_acceptance")
    }
    if o.LockOwner == nil || *o.LockOwner != escortID {
        return nil, errs.New(errs.CodeForbidden, "lock owner mismatch")
    }
    if err := s.orders.UpdateStatus(ctx, orderID, string(state.StatusAccepted), o.Version, &escortID); err != nil {
        return nil, errs.Wrap(errs.CodeInternal, "update status", err)
    }
    // 写 order_event
    actor := escortID
    from := string(state.StatusPendingAcceptance)
    if err := s.orders.InsertEvent(ctx, orderID, &from, string(state.StatusAccepted), &actor, nil); err != nil {
        return nil, errs.Wrap(errs.CodeInternal, "insert event", err)
    }
    return s.orders.FindByID(ctx, orderID)
}
```

**Step 4: 保留旧 Accept 但标记 deprecated**

> 注：旧的 `Accept(orderID, escortID)` 内部走"一次 SELECT+UPDATE"，会被 §4.2 的 Redis SETNX 版本替代。**本期保留兼容，下个 plan 删除**。

在 `accept.go` 顶部加 deprecation 注释：

```go
// Accept 保留兼容：单次 SELECT+UPDATE；§4.2 order-lock plan 引入 Redis SETNX 后删除。
//
// Deprecated: 新流程走 TryLock + ConfirmAccept；保留 1 个版本兼容。
```

**Step 5: 跑测试确认通过**

Run: `go test -tags=integration -run 'TestAccept_LockThenConfirm|TestAccept_LockFailsOnConflict|TestAccept_ReleaseLock' ./services/order/internal/service/`
Expected: PASS

**Step 6: Commit**

```bash
git add services/order/internal/service/
git commit -m "feat(order): service 加 TryLock/ReleaseAcceptLock/ConfirmAccept 三方法（为 §4.2 锁单铺路）"
```

---

### Task 5: 文档同步（docs/04 + docs/07）

**Files:**
- Modify: `docs/04-业务流程.md` §4.4 状态机图
- Modify: `docs/07-数据模型.md` §7.3.1 orders 表

**Step 1: 更新 04 状态机图**

把 §4.4 状态机 ASCII 图替换为：

```
                   ┌─ refunded ─┐
                   │             ▼
reviewing ─→ refunding ─→ settling ─→ closed
                                  ▲
                                  │
                                (disputed)
                                  ▲
                                  │
                          completed ─→ settling ─→ closed
                                  ▲
                                  │
                          (active dispute)

匹配：created → paid → matching → pending_acceptance → accepted
                                                  │
                                                  ↓ (锁单 30s 超时)
                                              matching
```

**Step 2: 更新 07 orders 表结构**

加：

```sql
lock_owner BIGINT REFERENCES users(id),     -- pending_acceptance 期间的持有者
lock_expire_at TIMESTAMPTZ,                  -- 锁单超时时间
```

并在状态枚举行加 `pending_acceptance / settling / disputed`。

**Step 3: 在 docs/superpowers/specs/2026-09-24-roadmap-design.md §4.1 加 commit 引用**

```markdown
**落地 commit**：
- 2026-09-24 feat(order): 状态机加 pending_acceptance/settling/disputed 三态
- 2026-09-24 feat(migrations): 0003_orders_state
- 2026-09-24 feat(order): repo 加 LockForAccept / ReleaseLock / LockExpired
- 2026-09-24 feat(order): service 加 TryLock/ReleaseAcceptLock/ConfirmAccept
```

**Step 4: Commit**

```bash
git add docs/
git commit -m "docs: 状态机统一 - 04 流程图 + 07 表结构 + roadmap 落地引用"
```

---

### Task 6: 全量回归测试 + smoke

**Files:**
- Modify: `scripts/smoke-order.sh`（加 30s 锁单路径 curl 演示）

**Step 1: 全量跑一次**

Run: `go test -count=1 ./shared/... ./services/...`
Expected: 37+ 包全部 PASS

Run: `go test -tags=integration -count=1 ./migrations/... ./services/...`
Expected: 全部 PASS（需要 docker compose up）

**Step 2: smoke 验证**

Run: `bash scripts/smoke-order.sh`
Expected: smoke OK（启动 + 鉴权拦截；本 plan 不改业务流）

**Step 3: dev.md 追加一行**

```markdown
| 3.10 | 状态机统一 + 锁单字段 (本 plan) | 4 commits | `feat(order): 状态机 + 锁单字段` |
```

**Step 4: Commit**

```bash
git add dev.md scripts/
git commit -m "docs: dev.md 状态机统一记录 + smoke 校验"
```

---

## Self-Review

- ✅ Spec 覆盖：评审 C-01（状态机不一致）— Task 1+2+5 落地；C-04 部分（只状态机部分；锁单 30s 与 Redis SETNX 留给 §4.2 plan）。
- ✅ 无占位符：每个 Step 都有具体代码与命令。
- ✅ 类型一致：`ErrInvalidStateForLock` 在 Task 3 定义，Task 4 复用；`Order.LockOwner` / `LockExpireAt` 在 Task 3 加入，Task 4 引用。
- ✅ 测试矩阵：Task 1 单元 + Task 2 集成 + Task 3 集成 + Task 4 集成 + Task 6 全量回归。
- ✅ YAGNI：本期不引入 Redis SETNX（§4.2 才做）；不引入异步任务（§4.2 才做）。

## 执行选项

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-state-machine.md`。
> 但因为你在 design 阶段选了 **plan_all（不实现）**，所以本 plan **不会被自动执行**。

**下一步选项**：
1. **继续 plan_all 模式**：我立即产出下一份 plan（§4.2 order-lock，含 Redis SETNX + 超时任务）
2. **停止 plan**：你 review 这份 plan 后告诉我调整；或你直接开始某个 plan 的实施（暂停 plan_all，进入 subagent-driven-development）