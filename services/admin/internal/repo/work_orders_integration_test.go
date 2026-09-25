//go:build integration
// +build integration

package repo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDSN 读环境变量；缺省用本地 docker 默认 DSN。
func testDSN() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

// setupAdminPool 起一个干净的 PG 测试 schema：users / orders / work_orders。
func setupAdminPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS work_orders CASCADE;
		DROP TABLE IF EXISTS orders CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
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
		  status VARCHAR(24) NOT NULL,
		  version INT NOT NULL DEFAULT 0,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE work_orders (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  category VARCHAR(32) NOT NULL CHECK (category IN (
		    'complaint','refund_consult','escort_issue','system_bug','other')),
		  priority VARCHAR(2) NOT NULL DEFAULT 'P2'
		    CHECK (priority IN ('P0','P1','P2','P3')),
		  status VARCHAR(16) NOT NULL DEFAULT 'pending'
		    CHECK (status IN ('pending','assigned','in_progress','resolved','closed')),
		  subject_id BIGINT,
		  subject_type VARCHAR(16) CHECK (subject_type IN ('order','user','escort','system','other') OR subject_type IS NULL),
		  assignee_id BIGINT REFERENCES users(id),
		  title VARCHAR(128) NOT NULL,
		  content TEXT NOT NULL,
		  resolution TEXT,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  closed_at TIMESTAMPTZ,
		  sla_due_at TIMESTAMPTZ
		);
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS work_orders;
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

// TestWorkOrderRepo_CreateAndGet 验证创建 + 按 ID 查询。
func TestWorkOrderRepo_CreateAndGet(t *testing.T) {
	pool := setupAdminPool(t)
	user := seedUser(t, pool, "13800138000", "patient")
	r := NewWorkOrderRepo(pool)

	now := time.Now()
	subjectID := int64(42)
	subjectType := "order"
	w := &WorkOrder{
		UserID:      user,
		Category:    "complaint",
		Priority:    "P1",
		Status:      "pending",
		SubjectID:   &subjectID,
		SubjectType: &subjectType,
		Title:       "服务投诉",
		Content:     "陪诊师迟到",
		SLADueAt:    &now,
	}
	require.NoError(t, r.Create(context.Background(), w))
	assert.NotZero(t, w.ID)

	got, err := r.FindByID(context.Background(), w.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "complaint", got.Category)
	assert.Equal(t, "P1", got.Priority)
	assert.Equal(t, "pending", got.Status)
	require.NotNil(t, got.SubjectID)
	assert.Equal(t, int64(42), *got.SubjectID)
	require.NotNil(t, got.SubjectType)
	assert.Equal(t, "order", *got.SubjectType)
}

// TestWorkOrderRepo_List_FilterByStatus 验证按状态筛选。
func TestWorkOrderRepo_List_FilterByStatus(t *testing.T) {
	pool := setupAdminPool(t)
	user := seedUser(t, pool, "13800138001", "patient")
	r := NewWorkOrderRepo(pool)

	for i := 0; i < 3; i++ {
		w := &WorkOrder{
			UserID:   user,
			Category: "refund_consult",
			Priority: "P2",
			Status:   "pending",
			Title:    "refund",
			Content:  "x",
		}
		require.NoError(t, r.Create(context.Background(), w))
	}
	w := &WorkOrder{
		UserID:   user,
		Category: "other",
		Priority: "P3",
		Status:   "closed",
		Title:    "ok",
		Content:  "x",
	}
	require.NoError(t, r.Create(context.Background(), w))

	pending, err := r.List(context.Background(), WorkOrderListFilter{
		Status:   "pending",
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Len(t, pending, 3)
}

// TestWorkOrderRepo_Assign 验证分配客服。
func TestWorkOrderRepo_Assign(t *testing.T) {
	pool := setupAdminPool(t)
	user := seedUser(t, pool, "13800138002", "patient")
	cs := seedUser(t, pool, "13800138003", "admin")
	r := NewWorkOrderRepo(pool)

	w := &WorkOrder{
		UserID: user, Category: "system_bug", Priority: "P0",
		Status: "pending", Title: "bug", Content: "x",
	}
	require.NoError(t, r.Create(context.Background(), w))
	require.NoError(t, r.Assign(context.Background(), w.ID, cs))

	got, _ := r.FindByID(context.Background(), w.ID)
	require.NotNil(t, got)
	require.NotNil(t, got.AssigneeID)
	assert.Equal(t, cs, *got.AssigneeID)
	assert.Equal(t, "assigned", got.Status)
}

// TestWorkOrderRepo_Resolve 验证关单 + 写 resolution。
func TestWorkOrderRepo_Resolve(t *testing.T) {
	pool := setupAdminPool(t)
	user := seedUser(t, pool, "13800138004", "patient")
	r := NewWorkOrderRepo(pool)

	w := &WorkOrder{
		UserID: user, Category: "complaint", Priority: "P1",
		Status: "in_progress", Title: "x", Content: "x",
	}
	require.NoError(t, r.Create(context.Background(), w))
	require.NoError(t, r.Resolve(context.Background(), w.ID, "已退款"))

	got, _ := r.FindByID(context.Background(), w.ID)
	require.NotNil(t, got)
	assert.Equal(t, "closed", got.Status)
	assert.Equal(t, "已退款", *got.Resolution)
	require.NotNil(t, got.ClosedAt)
}

// TestWorkOrderRepo_NotFound 验证 ErrWorkOrderNotFound。
func TestWorkOrderRepo_NotFound(t *testing.T) {
	pool := setupAdminPool(t)
	r := NewWorkOrderRepo(pool)
	_, err := r.FindByID(context.Background(), 99999)
	assert.True(t, errors.Is(err, ErrWorkOrderNotFound))
}