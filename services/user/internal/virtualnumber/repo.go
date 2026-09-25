// Package virtualnumber 实现 user-service 虚拟号的数据访问层。
//
// 设计要点：
//   - v1 mock：分配时从一段号池（17x 开头 + 8 位数字）生成 phone；
//   - 同一 order_id 仅允许 1 条 active（DB partial unique index 保证）；
//   - status 重算：expire_at < now 且未 released → expired。
package virtualnumber

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record 映射 virtual_numbers 表行。
type Record struct {
	ID         int64
	OrderID    int64
	PatientID  int64
	EscortID   int64
	Phone      string
	Status     string  // active | expired | released
	ExpireAt   time.Time
	ReleasedAt *time.Time
	CreatedAt  time.Time
}

// ErrNotFound 是查询无结果哨兵。
var ErrNotFound = errors.New("virtualnumber: not found")

// ErrDuplicateActive 是同订单已有 active 时的哨兵（partial unique 触发）。
var ErrDuplicateActive = errors.New("virtualnumber: order already has active virtual number")

// Repo 是 virtual_numbers 表的仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const baseSelect = `
	SELECT id, order_id, patient_id, escort_id, phone, status,
	       expire_at, released_at, created_at
	FROM virtual_numbers`

// AllocateInput 是 Allocate 入参。
type AllocateInput struct {
	OrderID   int64
	PatientID int64
	EscortID  int64
	ExpireAt  time.Time
	// Phone 可选；空则自动生成（mock 17x 号段）
	Phone string
}

// Allocate 分配虚拟号：事务内 INSERT + 选 phone（同 order_id 已 active 返回 ErrDuplicateActive）。
func (r *Repo) Allocate(ctx context.Context, in AllocateInput) (*Record, error) {
	phone := in.Phone
	if phone == "" {
		phone = generatePhone()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO virtual_numbers (order_id, patient_id, escort_id, phone, expire_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at`
	v := &Record{
		OrderID:   in.OrderID,
		PatientID: in.PatientID,
		EscortID:  in.EscortID,
		Phone:     phone,
		Status:    "active",
		ExpireAt:  in.ExpireAt,
	}
	err = tx.QueryRow(ctx, q,
		in.OrderID, in.PatientID, in.EscortID, phone, in.ExpireAt,
	).Scan(&v.ID, &v.Status, &v.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateActive
		}
		return nil, fmt.Errorf("insert virtual_number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return v, nil
}

// GetByID 按 id 查询；过期重算 status。
func (r *Repo) GetByID(ctx context.Context, id int64) (*Record, error) {
	q := baseSelect + ` WHERE id = $1`
	v, err := scan(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	v.Status = deriveStatus(v.Status, v.ReleasedAt, v.ExpireAt, time.Now())
	return v, nil
}

// Release 把虚拟号标记为 released；返回更新后的 record；id 不存在返回 NotFound。
func (r *Repo) Release(ctx context.Context, id int64, reason string) (*Record, error) {
	const q = `
		UPDATE virtual_numbers
		   SET status = 'released', released_at = NOW()
		 WHERE id = $1 AND status = 'active'
		 RETURNING id, order_id, patient_id, escort_id, phone, status,
		           expire_at, released_at, created_at`
	v, err := scan(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("release virtual_number: %w", err)
	}
	return v, nil
}

// ---------- helpers ----------

// scan 把 pgx.Row 扫描到 Record。
func scan(row pgx.Row) (*Record, error) {
	v := &Record{}
	if err := row.Scan(
		&v.ID, &v.OrderID, &v.PatientID, &v.EscortID, &v.Phone, &v.Status,
		&v.ExpireAt, &v.ReleasedAt, &v.CreatedAt,
	); err != nil {
		return nil, err
	}
	return v, nil
}

// deriveStatus 重算 status（expire_at < now → expired）。
func deriveStatus(current string, releasedAt *time.Time, expireAt, now time.Time) string {
	if current == "released" {
		return "released"
	}
	if releasedAt != nil {
		return "released"
	}
	if !expireAt.IsZero() && now.After(expireAt) {
		return "expired"
	}
	return "active"
}

// generatePhone 生成 mock 虚拟号：17 + 9 位数字（11 位）。
func generatePhone() string {
	const prefix = "17"
	n := rand.Int63n(1_000_000_000) // 0..999_999_999
	return fmt.Sprintf("%s%09d", prefix, n)
}

// isUniqueViolation 判断 PG 唯一约束冲突（23505）。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type pgErr interface {
		SQLState() string
	}
	var pe pgErr
	if errors.As(err, &pe) {
		return pe.SQLState() == "23505"
	}
	return false
}