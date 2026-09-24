//go:build integration
// +build integration

// Accept 集成测试：需要 docker compose up 起 PG。
//
// 覆盖：
//   1. 单 escort 抢单成功
//   2. N 个并发抢单，恰好 1 个成功
//   3. 抢单失败（行不存在 / 状态非法 / version 冲突）
package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/shared/lock"
)

func testDSN() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

func setupAcceptPool(t *testing.T) (*pgxpool.Pool, int64, int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS refunds CASCADE;
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
		CREATE TABLE order_events (
		  id BIGSERIAL PRIMARY KEY,
		  order_id BIGINT NOT NULL REFERENCES orders(id),
		  from_status VARCHAR(24),
		  to_status VARCHAR(24) NOT NULL,
		  actor_id BIGINT,
		  payload JSONB,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	patient := int64(0)
	escort := int64(0)
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users (phone, role) VALUES ('p', 'patient') RETURNING id`).Scan(&patient))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO users (phone, role) VALUES ('e', 'escort') RETURNING id`).Scan(&escort))

	// 2026-09-24 fix：TestAccept_OnlyOneWins 用 50 个不同 escortID（1000+i），
	// 但 orders.escort_id 是 FK；提前批量建 50 个 escort 用户，避免 FK 违反。
	for i := 0; i < 50; i++ {
		_, err := pool.Exec(ctx,
			`INSERT INTO users (phone, role) VALUES ($1, 'escort') ON CONFLICT DO NOTHING`,
			fmt.Sprintf("e-bulk-%d", i))
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS order_events;
			DROP TABLE IF EXISTS orders;
			DROP TABLE IF EXISTS users;
		`)
		pool.Close()
	})
	return pool, patient, escort
}

func seedOrder(t *testing.T, pool *pgxpool.Pool, patient int64, status string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
		                    service_start_at, amount, final_amount, status)
		VALUES ('O-test', $1, 1, 1, NOW() + INTERVAL '1 day', 100, 100, $2)
		RETURNING id`, patient, status).Scan(&id)
	require.NoError(t, err)
	return id
}

// fakeUserLookup 对 Accept 路径完全不影响。
type fakeUserLookup struct{}

func (fakeUserLookup) FindByID(ctx context.Context, id int64) (*UserSnapshot, error) {
	return &UserSnapshot{ID: id, Role: "patient", RealNameVerified: true}, nil
}

// TestAccept_SingleEscort 验证单个 escort 抢单成功。
func TestAccept_SingleEscort(t *testing.T) {
	pool, patient, escort := setupAcceptPool(t)
	orderID := seedOrder(t, pool, patient, "matching")
	r := repo.NewOrderRepo(pool)
	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})

	res, err := svc.Accept(context.Background(), orderID, escort)
	require.NoError(t, err)
	assert.Equal(t, orderID, res.OrderID)
	assert.Equal(t, 1, res.Version)

	got, _ := r.FindByID(context.Background(), orderID)
	assert.Equal(t, "accepted", got.Status)
	assert.Equal(t, escort, *got.EscortID)
}

// TestAccept_OnlyOneWins 100 个并发 → 恰好 1 个成功。
func TestAccept_OnlyOneWins(t *testing.T) {
	pool, patient, _ := setupAcceptPool(t)
	orderID := seedOrder(t, pool, patient, "matching")
	r := repo.NewOrderRepo(pool)
	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})

	const N = 50 // 50 并发（PG 默认 max_conns 在 docker compose 里是 100，足够）

	// 2026-09-24 fix：orders.escort_id 是 FK，concurrent test 必须用真实 user ids。
	// setupAcceptPool 已批量建 50 个 bulk 用户，id 从 ~3 开始递增；这里再查一次取 ids。
	rows, err := pool.Query(context.Background(),
		`SELECT id FROM users WHERE phone LIKE 'e-bulk-%' ORDER BY id LIMIT $1`, N)
	require.NoError(t, err)
	defer rows.Close()
	escortIDs := make([]int64, 0, N)
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		escortIDs = append(escortIDs, id)
	}
	require.Len(t, escortIDs, N, "setupAcceptPool 必须预建 50 个 bulk 用户")

	var wg sync.WaitGroup
	wg.Add(N)
	var successCount, lockCount, otherErrs int32

	for i := 0; i < N; i++ {
		go func(escortID int64) {
			defer wg.Done()
			_, err := svc.Accept(context.Background(), orderID, escortID)
			switch {
			case err == nil:
				atomic.AddInt32(&successCount, 1)
			case errors.Is(err, ErrOrderLocked), errors.Is(err, ErrInvalidState), errors.Is(err, ErrVersionConflict):
				atomic.AddInt32(&lockCount, 1)
			default:
				atomic.AddInt32(&otherErrs, 1)
				t.Logf("unexpected err [%d]: %v", escortID, err)
			}
		}(escortIDs[i])
	}
	wg.Wait()

	assert.EqualValues(t, 1, successCount, "只能 1 个成功")
	assert.EqualValues(t, N-1, lockCount, "其余 N-1 应被 SKIP LOCKED 拒")
	assert.EqualValues(t, 0, otherErrs, "无其他错误")
}

// TestAccept_OrderNotExist 订单 id 不存在。
func TestAccept_OrderNotExist(t *testing.T) {
	pool, _, escort := setupAcceptPool(t)
	r := repo.NewOrderRepo(pool)
	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})
	_, err := svc.Accept(context.Background(), 99999, escort)
	assert.ErrorIs(t, err, ErrOrderLocked, "FOR UPDATE SKIP LOCKED 对不存在行也返回 0 行")
}

// TestAccept_InvalidState 订单处于 created（不可直接 accept）状态。
func TestAccept_InvalidState(t *testing.T) {
	pool, patient, escort := setupAcceptPool(t)
	orderID := seedOrder(t, pool, patient, "created") // 非法状态
	r := repo.NewOrderRepo(pool)
	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool})
	_, err := svc.Accept(context.Background(), orderID, escort)
	assert.ErrorIs(t, err, ErrInvalidState)
}

// ===== 状态机统一 plan: TryLock + ConfirmAccept + ReleaseAcceptLock =====

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
// ===== §4.2 order-lock plan: TryLock 接 Redis SETNX =====

// TestTryLock_NopLockerAllowsTry 验证 Redis 不可用时（NopLocker）仍可走 DB 锁。
func TestTryLock_NopLockerAllowsTry(t *testing.T) {
	pool, patient, escort := setupAcceptPool(t)
	orderID := seedOrder(t, pool, patient, "matching")
	r := repo.NewOrderRepo(pool)
	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(lock.NopLocker{})

	require.NoError(t, svc.TryLock(context.Background(), orderID, escort, 30*time.Second))
}

// TestTryLock_RedisSetNX_BlocksSecondEscort 验证 Redis SETNX 阻止并发。
func TestTryLock_RedisSetNX_BlocksSecondEscort(t *testing.T) {
	pool, patient, escort1 := setupAcceptPool(t)
	escort2 := pickBulkEscort(t, pool, patient) // 用预建 bulk 用户；FK 不会触发
	orderID := seedOrder(t, pool, patient, "matching")
	r := repo.NewOrderRepo(pool)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	locker := lock.NewRedisLocker(rdb)

	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker)

	require.NoError(t, svc.TryLock(context.Background(), orderID, escort1, 30*time.Second))
	err = svc.TryLock(context.Background(), orderID, escort2, 30*time.Second)
	assert.ErrorIs(t, err, ErrLockTaken, "已被 SETNX 锁的订单不能被另一 escort 再锁")
}

// TestTryLock_RedisExpires_AllowsSecond 验证 Redis TTL 到期后另一 escort 能过 SETNX 关。
//
// 注：DB 层 escort1 已经把 status 切到 pending_acceptance，escort2 仍会被 DB
// 状态校验拒（ErrInvalidStateForLock）；这正是 plan 设计的"DB 兜底"——
// scheduler 必须先 ReleaseAcceptLock 释放 escort1 的 DB 锁单，escort2 才能再锁。
// 本测试只断言 SETNX 第一道闸已被 TTL 释放（用 assert.True 探针）。
func TestTryLock_RedisExpires_AllowsSecond(t *testing.T) {
	pool, patient, escort1 := setupAcceptPool(t)
	escort2 := pickBulkEscort(t, pool, patient)
	orderID := seedOrder(t, pool, patient, "matching")
	r := repo.NewOrderRepo(pool)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	locker := lock.NewRedisLocker(rdb)

	svc := New(r, fakeUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker)

	require.NoError(t, svc.TryLock(context.Background(), orderID, escort1, 100*time.Millisecond))
	// 加速 miniredis 时间（FastForward 让 TTL 立即到期）
	mr.FastForward(150 * time.Millisecond)
	err = svc.TryLock(context.Background(), orderID, escort2, 30*time.Second)
	// TTL 到期 → SETNX 通过 → 但 DB 还锁着 → ErrInvalidStateForLock（DB 兜底）
	assert.ErrorIs(t, err, ErrInvalidStateForLock,
		"TTL 到期后 SETNX 通过；DB 仍处于 pending_acceptance，被 DB 兜底拒")
}

// pickBulkEscort 从 setupAcceptPool 预建的 bulk 用户里取一个 escort id（满足 lock_owner FK）。
func pickBulkEscort(t *testing.T, pool *pgxpool.Pool, patient int64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE phone LIKE 'e-bulk-%' AND id <> $1 ORDER BY id LIMIT 1`, patient).
		Scan(&id)
	require.NoError(t, err)
	return id
}
