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
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/order/internal/repo"
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
		  status VARCHAR(16) NOT NULL CHECK (status IN (
		    'created','paid','matching','accepted','in_service','completed',
		    'reviewed','refunding','refunded','closed','canceled')),
		  version INT NOT NULL DEFAULT 0,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE order_events (
		  id BIGSERIAL PRIMARY KEY,
		  order_id BIGINT NOT NULL REFERENCES orders(id),
		  from_status VARCHAR(16),
		  to_status VARCHAR(16) NOT NULL,
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
				t.Logf("unexpected err: %v", err)
			}
		}(int64(1000 + i))
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