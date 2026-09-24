//go:build integration
// +build integration

package repo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDSN() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

// setupWalletPool 起连接池并准备 users + orders + wallets + withdrawals + billings。
// 集成测试自带表，便于独立运行；docker compose up + DOCTORS_TEST_DSN 环境变量可切换。
func setupWalletPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS billings CASCADE;
		DROP TABLE IF EXISTS withdrawals CASCADE;
		DROP TABLE IF EXISTS wallets CASCADE;
		DROP TABLE IF EXISTS sos_records CASCADE;
		DROP TABLE IF EXISTS refunds CASCADE;
		DROP TABLE IF EXISTS refund_policies CASCADE;
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
		  completed_at TIMESTAMPTZ,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE wallets (
		  user_id BIGINT PRIMARY KEY REFERENCES users(id),
		  balance NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
		  frozen NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (frozen >= 0),
		  total_earned NUMERIC(10,2) NOT NULL DEFAULT 0,
		  total_withdrawn NUMERIC(10,2) NOT NULL DEFAULT 0,
		  currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE withdrawals (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  amount NUMERIC(10,2) NOT NULL CHECK (amount > 0),
		  channel VARCHAR(16) NOT NULL CHECK (channel IN ('wx','alipay','mock')),
		  account VARCHAR(64) NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'pending'
		    CHECK (status IN ('pending','approved','paid','rejected')),
		  external_tx_id VARCHAR(64),
		  failure_reason VARCHAR(255),
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  reviewed_at TIMESTAMPTZ,
		  reviewed_by BIGINT REFERENCES users(id),
		  paid_at TIMESTAMPTZ
		);
		CREATE TABLE billings (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  order_id BIGINT REFERENCES orders(id),
		  type VARCHAR(32) NOT NULL CHECK (type IN (
		    'order_income','frozen_release','withdraw','withdraw_refund',
		    'refund_deduct','admin_adjust')),
		  amount NUMERIC(10,2) NOT NULL,
		  balance_after NUMERIC(10,2) NOT NULL,
		  frozen_after NUMERIC(10,2) NOT NULL,
		  note VARCHAR(255),
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS billings;
			DROP TABLE IF EXISTS withdrawals;
			DROP TABLE IF EXISTS wallets;
			DROP TABLE IF EXISTS sos_records;
			DROP TABLE IF EXISTS refunds CASCADE;
			DROP TABLE IF EXISTS refund_policies CASCADE;
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

func seedOrder(t *testing.T, pool *pgxpool.Pool, patientID int64, completedAt *time.Time) int64 {
	t.Helper()
	var id int64
	var err error
	if completedAt != nil {
		err = pool.QueryRow(context.Background(), `
			INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
			                    service_start_at, amount, final_amount, status, completed_at)
			VALUES ('O-w', $1, 1, 1, NOW(), 300, 300, 'completed', $2)
			RETURNING id`, patientID, *completedAt).Scan(&id)
	} else {
		err = pool.QueryRow(context.Background(), `
			INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
			                    service_start_at, amount, final_amount, status)
			VALUES ('O-w', $1, 1, 1, NOW(), 300, 300, 'completed')
			RETURNING id`, patientID).Scan(&id)
	}
	require.NoError(t, err)
	return id
}

// TestWalletRepo_GetOrCreate 验证 GetOrCreate 自动建行。
func TestWalletRepo_GetOrCreate(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138000", "escort")
	r := NewWalletRepo(pool)

	w, err := r.GetOrCreate(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, uid, w.UserID)
	assert.True(t, w.Balance.IsZero(), "初始 balance=0")
	assert.True(t, w.Frozen.IsZero(), "初始 frozen=0")
	assert.Equal(t, "CNY", w.Currency)
	assert.NotZero(t, w.CreatedAt)

	// 第二次 GetOrCreate 不应重复 INSERT
	w2, err := r.GetOrCreate(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, w.CreatedAt.Unix(), w2.CreatedAt.Unix(), "二次调用返回同一行")
}

// TestWalletRepo_Get_NoneWhenMissing 验证 Get 无记录时返回 nil,nil。
func TestWalletRepo_Get_NoneWhenMissing(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138001", "escort")
	r := NewWalletRepo(pool)

	w, err := r.Get(context.Background(), uid)
	require.NoError(t, err)
	assert.Nil(t, w)
}

// TestWalletRepo_FreezeIncome 验证给 frozen 加钱并写 billings。
func TestWalletRepo_FreezeIncome(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138002", "escort")
	orderID := seedOrder(t, pool, uid, nil)
	r := NewWalletRepo(pool)

	err := r.FreezeIncome(context.Background(), uid, orderID, decimal.NewFromInt(300))
	require.NoError(t, err)

	w, _ := r.Get(context.Background(), uid)
	require.NotNil(t, w)
	assert.True(t, w.Balance.IsZero())
	assert.True(t, w.Frozen.Equal(decimal.NewFromInt(300)), "frozen 应为 300")
	assert.True(t, w.TotalEarned.Equal(decimal.NewFromInt(300)))

	// billings 也落了一条
	var billCount int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM billings WHERE user_id=$1 AND type='order_income'`, uid).Scan(&billCount)
	require.NoError(t, err)
	assert.Equal(t, 1, billCount)
}

// TestWalletRepo_DeductFrozenForRefund 验证退款扣 frozen。
func TestWalletRepo_DeductFrozenForRefund(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138003", "escort")
	orderID := seedOrder(t, pool, uid, nil)
	r := NewWalletRepo(pool)
	require.NoError(t, r.FreezeIncome(context.Background(), uid, orderID, decimal.NewFromInt(300)))

	err := r.DeductFrozenForRefund(context.Background(), uid, orderID, decimal.NewFromInt(150))
	require.NoError(t, err)

	w, _ := r.Get(context.Background(), uid)
	require.NotNil(t, w)
	assert.True(t, w.Frozen.Equal(decimal.NewFromInt(150)))
}

// TestWalletRepo_UnfreezeToBalance 验证 T+7 把 frozen 转 balance。
func TestWalletRepo_UnfreezeToBalance(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138004", "escort")
	orderID := seedOrder(t, pool, uid, nil)
	r := NewWalletRepo(pool)
	require.NoError(t, r.FreezeIncome(context.Background(), uid, orderID, decimal.NewFromInt(300)))

	err := r.UnfreezeToBalance(context.Background(), uid, orderID, decimal.NewFromInt(300))
	require.NoError(t, err)

	w, _ := r.Get(context.Background(), uid)
	require.NotNil(t, w)
	assert.True(t, w.Frozen.IsZero())
	assert.True(t, w.Balance.Equal(decimal.NewFromInt(300)))
}

// TestWalletRepo_CreateWithdrawal 验证创建提现工单 + 扣 balance + 写 billings。
func TestWalletRepo_CreateWithdrawal(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138005", "escort")
	r := NewWalletRepo(pool)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), uid, 0, decimal.NewFromInt(500)))

	w := &Withdrawal{
		UserID:  uid,
		Amount:  decimal.NewFromInt(200),
		Channel: "wx",
		Account: "138****0000",
		Status:  "pending",
	}
	require.NoError(t, r.CreateWithdrawal(context.Background(), w))
	assert.NotZero(t, w.ID)

	wallet, _ := r.Get(context.Background(), uid)
	require.NotNil(t, wallet)
	assert.True(t, wallet.Balance.Equal(decimal.NewFromInt(300)), "balance 500-200=300")
	assert.True(t, wallet.TotalWithdrawn.Equal(decimal.NewFromInt(200)))
}

// TestWalletRepo_ApproveAndMarkPaid 验证审核通过 + 打款完成。
func TestWalletRepo_ApproveAndMarkPaid(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138006", "escort")
	r := NewWalletRepo(pool)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), uid, 0, decimal.NewFromInt(200)))
	w := &Withdrawal{UserID: uid, Amount: decimal.NewFromInt(200), Channel: "wx", Account: "138****0000", Status: "pending"}
	require.NoError(t, r.CreateWithdrawal(context.Background(), w))

	require.NoError(t, r.MarkWithdrawalApproved(context.Background(), w.ID, uid))
	require.NoError(t, r.MarkWithdrawalPaid(context.Background(), w.ID, "MOCK-TX-1"))

	got, err := r.GetWithdrawal(context.Background(), w.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "paid", got.Status)
	assert.Equal(t, "MOCK-TX-1", got.ExternalTxID)
	assert.NotNil(t, got.PaidAt)
}

// TestWalletRepo_ListTransactions 验证账单分页。
func TestWalletRepo_ListTransactions(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138007", "escort")
	orderID := seedOrder(t, pool, uid, nil)
	r := NewWalletRepo(pool)
	require.NoError(t, r.FreezeIncome(context.Background(), uid, orderID, decimal.NewFromInt(300)))
	require.NoError(t, r.UnfreezeToBalance(context.Background(), uid, orderID, decimal.NewFromInt(300)))

	txs, err := r.ListTransactions(context.Background(), uid, 10, 0)
	require.NoError(t, err)
	assert.Len(t, txs, 2, "1 笔 order_income + 1 笔 frozen_release")
}

// TestWalletRepo_ListCompletedOrdersBefore 验证 scanner 用的"已完成的订单"。
func TestWalletRepo_ListCompletedOrdersBefore(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138008", "escort")
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	recentTime := time.Now().Add(-1 * time.Hour)
	oldOrder := seedOrder(t, pool, uid, &oldTime)
	recentOrder := seedOrder(t, pool, uid, &recentTime)
	r := NewWalletRepo(pool)
	require.NoError(t, r.FreezeIncome(context.Background(), uid, oldOrder, decimal.NewFromInt(100)))
	require.NoError(t, r.FreezeIncome(context.Background(), uid, recentOrder, decimal.NewFromInt(200)))

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	orders, err := r.ListCompletedOrdersBefore(context.Background(), cutoff, 50)
	require.NoError(t, err)
	assert.Len(t, orders, 1, "只有 10 天前的订单在 cutoff 之前")
	assert.Equal(t, oldOrder, orders[0].OrderID)
	assert.True(t, orders[0].Amount.Equal(decimal.NewFromInt(100)))
}

// TestWalletRepo_FreezeIncome_NoWalletError 验证 GetOrCreate 应在 UPDATE 之前自动建行。
func TestWalletRepo_FreezeIncome_NoWalletError(t *testing.T) {
	pool := setupWalletPool(t)
	uid := seedUser(t, pool, "13800138009", "escort")
	r := NewWalletRepo(pool)

	// 用户没 wallet 行；FreezeIncome 应先 GetOrCreate 再 UPDATE。
	err := r.FreezeIncome(context.Background(), uid, 1, decimal.NewFromInt(100))
	require.NoError(t, err, "GetOrCreate 应在 UPDATE 之前自动建行")
}