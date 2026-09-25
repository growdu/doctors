//go:build integration
// +build integration

package hospital

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
		DROP TABLE IF EXISTS hospitals;
		CREATE TABLE hospitals (
		  id BIGSERIAL PRIMARY KEY,
		  name VARCHAR(128) NOT NULL,
		  city_id BIGINT NOT NULL,
		  level VARCHAR(16) NOT NULL CHECK (level IN ('3a','3b','2a','2b','1','other')),
		  status VARCHAR(16) NOT NULL DEFAULT 'active'
		    CHECK (status IN ('active','inactive')),
		  address VARCHAR(255) NOT NULL,
		  lat DOUBLE PRECISION,
		  lng DOUBLE PRECISION,
		  phone VARCHAR(20),
		  departments TEXT,
		  description TEXT,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS hospitals;`)
		pool.Close()
	})
	return pool
}

func seedHospital(t *testing.T, pool *pgxpool.Pool, name string, cityID int64, level string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO hospitals (name, city_id, level, address)
		VALUES ($1, $2, $3, 'addr') RETURNING id`, name, cityID, level).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestRepo_List_Filter 验证 city_id + level + keyword 过滤。
func TestRepo_List_Filter(t *testing.T) {
	pool := setupPool(t)
	seedHospital(t, pool, "协和医院", 1, "3a")
	seedHospital(t, pool, "301 医院", 1, "3a")
	seedHospital(t, pool, "社区医院", 1, "1")
	seedHospital(t, pool, "上海中山", 2, "3a")
	r := NewRepo(pool)

	// 仅 city=1
	list, total, err := r.List(context.Background(), ListFilter{CityID: 1, Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, list, 3)

	// city=1 + level=3a
	list, total, err = r.List(context.Background(), ListFilter{CityID: 1, Level: "3a", Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, list, 2)

	// keyword=协和
	list, total, err = r.List(context.Background(), ListFilter{Keyword: "协和", Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "协和医院", list[0].Name)
}

// TestRepo_List_OnlyActive 验证 inactive 不出现在列表。
func TestRepo_List_OnlyActive(t *testing.T) {
	pool := setupPool(t)
	seedHospital(t, pool, "active 医院", 1, "3a")
	var id2 int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO hospitals (name, city_id, level, address, status)
		VALUES ('inactive 医院', 1, '3a', 'addr', 'inactive') RETURNING id`).Scan(&id2)
	require.NoError(t, err)
	r := NewRepo(pool)

	list, total, err := r.List(context.Background(), ListFilter{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, list, 1)
}

// TestRepo_GetByID_OK 验证按 id 查询。
func TestRepo_GetByID_OK(t *testing.T) {
	pool := setupPool(t)
	id := seedHospital(t, pool, "x", 1, "3a")
	r := NewRepo(pool)
	h, err := r.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "x", h.Name)
}

// TestRepo_GetByID_NotFound 验证不存在返回 ErrNotFound。
func TestRepo_GetByID_NotFound(t *testing.T) {
	pool := setupPool(t)
	r := NewRepo(pool)
	_, err := r.GetByID(context.Background(), 9999)
	assert.ErrorIs(t, err, ErrNotFound)
}