//go:build integration
// +build integration

// order_repo 集成测试：需要 docker compose up 起 PG。
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

// setupPool 起连接池并准备 users + orders + order_events 表。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

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
		  from_status VARCHAR(16),
		  to_status VARCHAR(16) NOT NULL,
		  actor_id BIGINT,
		  payload JSONB,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS order_events;
			DROP TABLE IF EXISTS orders;
			DROP TABLE IF EXISTS users;
		`)
		pool.Close()
	})
	return pool
}

// seedUser 插入一个 patient 并返回 id。
func seedUser(t *testing.T, pool *pgxpool.Pool, phone, role string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone, role) VALUES ($1, $2) RETURNING id`, phone, role).Scan(&id)
	require.NoError(t, err)
	return id
}

// sampleOrder 构造一个最小可用的 Order，patientID 必填。
func sampleOrder(patientID int64) *Order {
	return &Order{
		OrderNo:        "O20260924-001",
		PatientID:      patientID,
		HospitalID:     100,
		PackageID:      1,
		ServiceStartAt: time.Now().Add(24 * time.Hour),
		Amount:         200.00,
		FinalAmount:    200.00,
		Status:         "created",
	}
}

func TestOrderRepo_CreateAndFindByID(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138000", "patient")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))
	assert.NotZero(t, o.ID)

	got, err := r.FindByID(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, o.OrderNo, got.OrderNo)
	assert.Equal(t, patient, got.PatientID)
	assert.Equal(t, "created", got.Status)
	assert.Equal(t, 0, got.Version)
}

func TestOrderRepo_ListByPatient(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138001", "patient")
	r := NewOrderRepo(pool)

	for i := 0; i < 3; i++ {
		o := sampleOrder(patient)
		o.OrderNo = "O20260924-00" + string(rune('1'+i))
		require.NoError(t, r.Create(context.Background(), o))
	}

	list, err := r.ListByPatient(context.Background(), patient, 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestOrderRepo_UpdateStatusWithVersion(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138002", "patient")
	escort := seedUser(t, pool, "13800138003", "escort")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))

	// 第一次更新成功
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "paid", 0, nil))
	// 同版本号再更新应失败（乐观锁）
	err := r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil)
	assert.Error(t, err)
	// 用正确版本号再更新应成功
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 1, nil))

	// 验证 status 与 version 都对
	got, _ := r.FindByID(context.Background(), o.ID)
	assert.Equal(t, "matching", got.Status)
	assert.Equal(t, 2, got.Version)

	// accept：传入 escortID
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "accepted", 2, &escort))
	got, _ = r.FindByID(context.Background(), o.ID)
	assert.Equal(t, "accepted", got.Status)
	assert.Equal(t, escort, *got.EscortID)
}

func TestOrderRepo_InsertEvent(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800138004", "patient")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))

	from, to := "created", "paid"
	require.NoError(t, r.InsertEvent(context.Background(), o.ID, &from, to, &patient, nil))

	events, err := r.ListEvents(context.Background(), o.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, &from, events[0].FromStatus)
	assert.Equal(t, to, events[0].ToStatus)
}

// ===== 状态机统一 plan: LockForAccept / ReleaseLock / LockExpired =====

func TestOrderRepo_LockForAccept_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800139000", "patient")
	escort := seedUser(t, pool, "13800139001", "escort")
	r := NewOrderRepo(pool)

	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))

	err := r.LockForAccept(context.Background(), o.ID, escort, time.Now().Add(30*time.Second), 1)
	require.NoError(t, err)

	got, _ := r.FindByID(context.Background(), o.ID)
	assert.Equal(t, "pending_acceptance", got.Status)
	require.NotNil(t, got.LockOwner)
	assert.Equal(t, escort, *got.LockOwner)
	require.NotNil(t, got.LockExpireAt)
	assert.True(t, got.LockExpireAt.After(time.Now()))
}

func TestOrderRepo_LockForAccept_VersionMismatch(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800139002", "patient")
	r := NewOrderRepo(pool)
	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))

	// version 999 不匹配
	err := r.LockForAccept(context.Background(), o.ID, 1, time.Now(), 999)
	assert.ErrorIs(t, err, ErrVersionConflict)
}

func TestOrderRepo_LockForAccept_WrongStatus(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800139003", "patient")
	r := NewOrderRepo(pool)
	o := sampleOrder(patient) // status='created'
	require.NoError(t, r.Create(context.Background(), o))

	err := r.LockForAccept(context.Background(), o.ID, 1, time.Now(), 0)
	assert.ErrorIs(t, err, ErrInvalidStateForLock)
}

func TestOrderRepo_ReleaseLock_OK(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800139004", "patient")
	escort := seedUser(t, pool, "13800139005", "escort")
	r := NewOrderRepo(pool)
	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))
	require.NoError(t, r.LockForAccept(context.Background(), o.ID, escort, time.Now().Add(30*time.Second), 1))

	require.NoError(t, r.ReleaseLock(context.Background(), o.ID, 2))
	got, _ := r.FindByID(context.Background(), o.ID)
	assert.Equal(t, "matching", got.Status)
	assert.Nil(t, got.LockOwner)
	assert.Nil(t, got.LockExpireAt)
}

func TestOrderRepo_LockExpired_FindsExpiring(t *testing.T) {
	pool := setupPool(t)
	patient := seedUser(t, pool, "13800139006", "patient")
	escort := seedUser(t, pool, "13800139007", "escort")
	r := NewOrderRepo(pool)
	o := sampleOrder(patient)
	require.NoError(t, r.Create(context.Background(), o))
	require.NoError(t, r.UpdateStatus(context.Background(), o.ID, "matching", 0, nil))
	// 锁在过去 = 已过期
	require.NoError(t, r.LockForAccept(context.Background(), o.ID, escort, time.Now().Add(-1*time.Hour), 1))

	expired, err := r.LockExpired(context.Background(), time.Now(), 10)
	require.NoError(t, err)
	assert.Len(t, expired, 1)
	assert.Equal(t, o.ID, expired[0].ID)
}