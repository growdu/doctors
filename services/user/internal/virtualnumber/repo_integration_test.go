//go:build integration
// +build integration

package virtualnumber

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
		DROP TABLE IF EXISTS virtual_numbers CASCADE;
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
		CREATE TABLE virtual_numbers (
		  id BIGSERIAL PRIMARY KEY,
		  order_id BIGINT NOT NULL,
		  patient_id BIGINT NOT NULL REFERENCES users(id),
		  escort_id BIGINT NOT NULL REFERENCES users(id),
		  phone VARCHAR(20) NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'active'
		    CHECK (status IN ('active','expired','released')),
		  expire_at TIMESTAMPTZ NOT NULL,
		  released_at TIMESTAMPTZ,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX uq_virtual_numbers_order_active ON virtual_numbers(order_id)
		  WHERE status = 'active';
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DROP TABLE IF EXISTS virtual_numbers; DROP TABLE IF EXISTS users;`)
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

// TestRepo_Allocate_OK 验证分配。
func TestRepo_Allocate_OK(t *testing.T) {
	pool := setupPool(t)
	p := seedUser(t, pool, "13800139001", "patient")
	e := seedUser(t, pool, "13800139002", "escort")
	r := NewRepo(pool)

	v, err := r.Allocate(context.Background(), AllocateInput{
		OrderID: 100, PatientID: p, EscortID: e,
		ExpireAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	assert.NotZero(t, v.ID)
	assert.NotEmpty(t, v.Phone)
}

// TestRepo_Allocate_DuplicateOrder 验证同 order 已有 active 返回 ErrDuplicateActive。
func TestRepo_Allocate_DuplicateOrder(t *testing.T) {
	pool := setupPool(t)
	p := seedUser(t, pool, "13800139003", "patient")
	e := seedUser(t, pool, "13800139004", "escort")
	r := NewRepo(pool)
	in := AllocateInput{
		OrderID: 200, PatientID: p, EscortID: e,
		ExpireAt: time.Now().Add(time.Hour),
	}
	_, err := r.Allocate(context.Background(), in)
	require.NoError(t, err)
	_, err = r.Allocate(context.Background(), in)
	assert.ErrorIs(t, err, ErrDuplicateActive)
}

// TestRepo_GetByID_OK 验证查询。
func TestRepo_GetByID_OK(t *testing.T) {
	pool := setupPool(t)
	p := seedUser(t, pool, "13800139005", "patient")
	e := seedUser(t, pool, "13800139006", "escort")
	r := NewRepo(pool)
	v, _ := r.Allocate(context.Background(), AllocateInput{
		OrderID: 300, PatientID: p, EscortID: e,
		ExpireAt: time.Now().Add(time.Hour),
	})
	got, err := r.GetByID(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, v.ID, got.ID)
}

// TestRepo_GetByID_NotFound 验证不存在返回 ErrNotFound。
func TestRepo_GetByID_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewRepo(pool)
	_, err := r.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestRepo_Release_OK 验证释放。
func TestRepo_Release_OK(t *testing.T) {
	pool := setupPool(t)
	p := seedUser(t, pool, "13800139007", "patient")
	e := seedUser(t, pool, "13800139008", "escort")
	r := NewRepo(pool)
	v, _ := r.Allocate(context.Background(), AllocateInput{
		OrderID: 400, PatientID: p, EscortID: e,
		ExpireAt: time.Now().Add(time.Hour),
	})
	released, err := r.Release(context.Background(), v.ID, "order_completed")
	require.NoError(t, err)
	assert.Equal(t, "released", released.Status)
	assert.NotNil(t, released.ReleasedAt)
}