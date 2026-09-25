//go:build integration
// +build integration

package address

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

// setupPool 建表 + 清表。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err)
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
		  detail VARCHAR(200) NOT NULL,
		  lat DOUBLE PRECISION,
		  lng DOUBLE PRECISION,
		  is_default BOOLEAN NOT NULL DEFAULT FALSE,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX uq_addresses_user_default ON addresses(user_id) WHERE is_default = TRUE;
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DROP TABLE IF EXISTS addresses; DROP TABLE IF EXISTS users;`)
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

// TestRepo_Create_OK 验证 Create 路径。
func TestRepo_Create_OK(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800138001")
	r := NewRepo(pool)

	a := &Record{UserID: uid, Recipient: "张三", Phone: "13800138001", Detail: "东单9号"}
	require.NoError(t, r.Create(context.Background(), a))
	assert.NotZero(t, a.ID)
	assert.NotZero(t, a.CreatedAt)
	assert.False(t, a.IsDefault)
}

// TestRepo_CreateDefault_ClearsPrevious 验证 CreateDefault 切换默认。
func TestRepo_CreateDefault_ClearsPrevious(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800138002")
	r := NewRepo(pool)

	a1 := &Record{UserID: uid, Recipient: "甲", Phone: "13800138001", Detail: "a"}
	require.NoError(t, r.CreateDefault(context.Background(), a1))

	a2 := &Record{UserID: uid, Recipient: "乙", Phone: "13800138002", Detail: "b"}
	require.NoError(t, r.CreateDefault(context.Background(), a2))

	defaults, err := r.ListByUser(context.Background(), uid)
	require.NoError(t, err)
	var onlyDefault *Record
	for _, x := range defaults {
		if x.IsDefault {
			require.Nil(t, onlyDefault, "应有且只有 1 个默认")
			onlyDefault = x
		}
	}
	require.NotNil(t, onlyDefault)
	assert.Equal(t, a2.ID, onlyDefault.ID)
}

// TestRepo_Update_NotFound 验证 update 不存在 id 返回 ErrNotFound。
func TestRepo_Update_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewRepo(pool)
	err := r.Update(context.Background(), &Record{ID: 9999, UserID: 1, Recipient: "x", Phone: "13800138000", Detail: "y"})
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestRepo_GetByID_WrongUser 越权保护：别人的地址返回 NotFound。
func TestRepo_GetByID_WrongUser(t *testing.T) {
	pool := setupPool(t)
	u1 := seedUser(t, pool, "13800138003")
	u2 := seedUser(t, pool, "13800138004")
	r := NewRepo(pool)
	a := &Record{UserID: u1, Recipient: "u1", Phone: "13800138000", Detail: "x"}
	require.NoError(t, r.Create(context.Background(), a))

	_, err := r.GetByID(context.Background(), a.ID, u2)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestRepo_Delete_WrongUser 越权保护：删别人地址返回 NotFound。
func TestRepo_Delete_WrongUser(t *testing.T) {
	pool := setupPool(t)
	u1 := seedUser(t, pool, "13800138005")
	u2 := seedUser(t, pool, "13800138006")
	r := NewRepo(pool)
	a := &Record{UserID: u1, Recipient: "u1", Phone: "13800138000", Detail: "x"}
	require.NoError(t, r.Create(context.Background(), a))

	err := r.Delete(context.Background(), a.ID, u2)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestRepo_SetDefault_OK 验证默认切换。
func TestRepo_SetDefault_OK(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800138007")
	r := NewRepo(pool)

	a1 := &Record{UserID: uid, Recipient: "甲", Phone: "13800138001", Detail: "a"}
	a2 := &Record{UserID: uid, Recipient: "乙", Phone: "13800138002", Detail: "b"}
	require.NoError(t, r.Create(context.Background(), a1))
	require.NoError(t, r.Create(context.Background(), a2))

	require.NoError(t, r.SetDefault(context.Background(), a2.ID, uid))

	got, err := r.GetByID(context.Background(), a2.ID, uid)
	require.NoError(t, err)
	assert.True(t, got.IsDefault)

	got1, err := r.GetByID(context.Background(), a1.ID, uid)
	require.NoError(t, err)
	assert.False(t, got1.IsDefault)
}

// TestRepo_CountByUser 验证计数。
func TestRepo_CountByUser(t *testing.T) {
	pool := setupPool(t)
	uid := seedUser(t, pool, "13800138008")
	r := NewRepo(pool)

	n, err := r.CountByUser(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	for i := 0; i < 3; i++ {
		require.NoError(t, r.Create(context.Background(),
			&Record{UserID: uid, Recipient: "r", Phone: "13800138000", Detail: "d"}))
	}
	n, err = r.CountByUser(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
}