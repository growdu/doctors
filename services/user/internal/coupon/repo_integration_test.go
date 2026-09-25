//go:build integration
// +build integration

package coupon

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

func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS user_coupons CASCADE;
		DROP TABLE IF EXISTS coupons CASCADE;
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
		CREATE TABLE coupons (
		  id BIGSERIAL PRIMARY KEY,
		  name VARCHAR(64) NOT NULL,
		  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
		  value NUMERIC(10,2) NOT NULL,
		  threshold NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (threshold >= 0),
		  valid_from TIMESTAMPTZ NOT NULL,
		  valid_until TIMESTAMPTZ NOT NULL,
		  stock INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
		  status VARCHAR(16) NOT NULL DEFAULT 'active',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE user_coupons (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  coupon_id BIGINT NOT NULL REFERENCES coupons(id),
		  status VARCHAR(16) NOT NULL DEFAULT 'unused',
		  expires_at TIMESTAMPTZ NOT NULL,
		  used_at TIMESTAMPTZ,
		  claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX uq_user_coupons_user_coupon ON user_coupons(user_id, coupon_id);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DROP TABLE IF EXISTS user_coupons; DROP TABLE IF EXISTS coupons; DROP TABLE IF EXISTS users;`)
		pool.Close()
	})
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool, phone string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone, role) VALUES ($1, 'patient') RETURNING id`, phone).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedCoupon(t *testing.T, pool *pgxpool.Pool, name string, stock int) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO coupons (name, type, value, threshold, valid_from, valid_until, stock)
		VALUES ($1, 'amount_off', 20, 100, NOW(), NOW() + INTERVAL '30 days', $2)
		RETURNING id`, name, stock).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestRepo_Claim_OK 验证领券流程。
func TestRepo_Claim_OK(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800139001")
	cid := seedCoupon(t, pool, "x", 5)
	r := NewRepo(pool)

	uc, err := r.Claim(context.Background(), uid, cid)
	require.NoError(t, err)
	assert.NotZero(t, uc.ID)
	assert.Equal(t, "unused", uc.Status)
}

// TestRepo_Claim_AlreadyClaimed 重复领取返回 ErrAlreadyClaimed。
func TestRepo_Claim_AlreadyClaimed(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800139002")
	cid := seedCoupon(t, pool, "x", 5)
	r := NewRepo(pool)

	_, err := r.Claim(context.Background(), uid, cid)
	require.NoError(t, err)
	_, err = r.Claim(context.Background(), uid, cid)
	assert.ErrorIs(t, err, ErrAlreadyClaimed)
}

// TestRepo_Claim_NoStock 库存不足返回 ErrNoStock。
func TestRepo_Claim_NoStock(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800139003")
	cid := seedCoupon(t, pool, "x", 0)
	r := NewRepo(pool)

	_, err := r.Claim(context.Background(), uid, cid)
	assert.ErrorIs(t, err, ErrNoStock)
}

// TestRepo_Claim_NotFound 券不存在返回 ErrCouponNotFound。
func TestRepo_Claim_NotFound(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800139004")
	r := NewRepo(pool)

	_, err := r.Claim(context.Background(), uid, 9999)
	assert.ErrorIs(t, err, ErrCouponNotFound)
}

// TestRepo_MarkUsed_OK 验证核销。
func TestRepo_MarkUsed_OK(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800139005")
	cid := seedCoupon(t, pool, "x", 5)
	r := NewRepo(pool)
	uc, err := r.Claim(context.Background(), uid, cid)
	require.NoError(t, err)
	require.NoError(t, r.MarkUsed(context.Background(), uc.ID, uid))
}

// TestRepo_MarkUsed_WrongUser 越权核销返回 NotFound。
func TestRepo_MarkUsed_WrongUser(t *testing.T) {
	pool := setupPool(t)
	u1 := seedUser(t, pool, "13800139006")
	u2 := seedUser(t, pool, "13800139007")
	cid := seedCoupon(t, pool, "x", 5)
	r := NewRepo(pool)
	uc, _ := r.Claim(context.Background(), u1, cid)
	err := r.MarkUsed(context.Background(), uc.ID, u2)
	assert.ErrorIs(t, err, ErrUserCouponNotFound)
}

// TestRepo_ListByUser_ReDeriveStatus 验证状态重算（expired → expired）。
func TestRepo_ListByUser_ReDeriveStatus(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800139008")
	cid := seedCoupon(t, pool, "x", 5)
	r := NewRepo(pool)
	_, err := r.Claim(context.Background(), uid, cid)
	require.NoError(t, err)
	// 把 user_coupon.expires_at 改到过去
	_, err = pool.Exec(context.Background(),
		`UPDATE user_coupons SET expires_at = $1 WHERE user_id = $2`, time.Now().Add(-time.Hour), uid)
	require.NoError(t, err)
	list, err := r.ListByUser(context.Background(), uid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "expired", list[0].Status)
}