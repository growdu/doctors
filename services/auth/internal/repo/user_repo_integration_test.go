//go:build integration
// +build integration

// user_repo 集成测试：需要 docker compose up 起 PG。
//
// 每个测试独立创建 schema + 表，结束后清理，避免互相污染。
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

// setupPool 起一个连接池并准备干净的 users 表。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS users;
		CREATE TABLE users (
		  id BIGSERIAL PRIMARY KEY,
		  phone VARCHAR(20) UNIQUE NOT NULL,
		  role VARCHAR(16) NOT NULL CHECK (role IN ('patient','escort','admin')),
		  nickname VARCHAR(64),
		  avatar_url VARCHAR(255),
		  real_name_verified BOOLEAN NOT NULL DEFAULT FALSE,
		  id_card_hash VARCHAR(64),
		  id_card_tail VARCHAR(8),
		  wx_unionid VARCHAR(64) UNIQUE,
		  wx_openid_mini VARCHAR(64),
		  wx_openid_app VARCHAR(64),
		  status SMALLINT NOT NULL DEFAULT 1,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
	`)
	require.NoError(t, err, "create users table")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS users`)
		pool.Close()
	})
	return pool
}

func TestUserRepo_CreateAndFindByPhone(t *testing.T) {
	pool := setupPool(t)
	r := NewUserRepo(pool)

	u := &User{
		Phone: "13800138000",
		Role:  RolePatient,
	}
	err := r.Create(context.Background(), u)
	require.NoError(t, err)
	assert.NotZero(t, u.ID, "ID should be assigned")

	got, err := r.FindByPhone(context.Background(), "13800138000")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, "13800138000", got.Phone)
	assert.Equal(t, RolePatient, got.Role)
}

func TestUserRepo_FindByPhone_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewUserRepo(pool)

	_, err := r.FindByPhone(context.Background(), "999")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_FindByUnionID(t *testing.T) {
	pool := setupPool(t)
	r := NewUserRepo(pool)

	u := &User{
		Phone:    "13800138001",
		Role:     RolePatient,
		WxUnionID: stringPtr("wx-abc-001"),
	}
	require.NoError(t, r.Create(context.Background(), u))

	got, err := r.FindByUnionID(context.Background(), "wx-abc-001")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	_, err = r.FindByUnionID(context.Background(), "nope")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_UpdateRealName(t *testing.T) {
	pool := setupPool(t)
	r := NewUserRepo(pool)

	u := &User{Phone: "13800138002", Role: RolePatient}
	require.NoError(t, r.Create(context.Background(), u))

	err := r.UpdateRealName(context.Background(), u.ID, "hash-xxx", "1234")
	require.NoError(t, err)

	got, err := r.FindByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.True(t, got.RealNameVerified)
	assert.Equal(t, "hash-xxx", got.IDCardHash)
	assert.Equal(t, "1234", got.IDCardTail)
}

func stringPtr(s string) *string { return &s }