//go:build integration
// +build integration

package pkg

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
		DROP TABLE IF EXISTS packages CASCADE;
		DROP TABLE IF EXISTS hospitals CASCADE;
		CREATE TABLE hospitals (
		  id BIGSERIAL PRIMARY KEY,
		  name VARCHAR(128) NOT NULL,
		  city_id BIGINT NOT NULL,
		  level VARCHAR(16) NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'active',
		  address VARCHAR(255) NOT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE packages (
		  id BIGSERIAL PRIMARY KEY,
		  hospital_id BIGINT NOT NULL REFERENCES hospitals(id),
		  name VARCHAR(128) NOT NULL,
		  type VARCHAR(16) NOT NULL,
		  duration_min INT NOT NULL,
		  price NUMERIC(10,2) NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'active',
		  description TEXT,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DROP TABLE IF EXISTS packages; DROP TABLE IF EXISTS hospitals;`)
		pool.Close()
	})
	return pool
}

func seedHospital(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO hospitals (name, city_id, level, address) VALUES ($1, 1, '3a', 'addr') RETURNING id`, name,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedPackage(t *testing.T, pool *pgxpool.Pool, hospitalID int64, name string, price float64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO packages (hospital_id, name, type, duration_min, price)
		VALUES ($1, $2, 'half_day', 240, $3) RETURNING id`,
		hospitalID, name, price,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestRepo_ListByHospital_OK 验证按医院列表。
func TestRepo_ListByHospital_OK(t *testing.T) {
	pool := setupPool(t)
	h1 := seedHospital(t, pool, "协和")
	seedPackage(t, pool, h1, "半日", 200)
	seedPackage(t, pool, h1, "全日", 380)
	h2 := seedHospital(t, pool, "301")
	seedPackage(t, pool, h2, "其它", 100)

	r := NewRepo(pool)
	list, err := r.ListByHospital(context.Background(), h1)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

// TestRepo_ListByHospital_Empty 验证无 active 返回空。
func TestRepo_ListByHospital_Empty(t *testing.T) {
	pool := setupPool(t)
	h1 := seedHospital(t, pool, "协和")
	// 建一个 inactive
	_, err := pool.Exec(context.Background(), `
		INSERT INTO packages (hospital_id, name, type, duration_min, price, status)
		VALUES ($1, 'x', 'half_day', 240, 100, 'inactive')`, h1)
	require.NoError(t, err)
	r := NewRepo(pool)
	list, err := r.ListByHospital(context.Background(), h1)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// TestRepo_GetByID_OK 验证按 id 查询。
func TestRepo_GetByID_OK(t *testing.T) {
	pool := setupPool(t)
	h1 := seedHospital(t, pool, "协和")
	pid := seedPackage(t, pool, h1, "x", 100)
	r := NewRepo(pool)
	p, err := r.GetByID(context.Background(), pid)
	require.NoError(t, err)
	assert.Equal(t, "x", p.Name)
	assert.InDelta(t, 100.0, p.Price, 0.01)
}

// TestRepo_GetByID_NotFound 验证不存在返回 ErrNotFound。
func TestRepo_GetByID_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewRepo(pool)
	_, err := r.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, ErrNotFound)
}