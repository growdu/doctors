// Package address 实现 user-service 地址簿的数据访问层。
//
// 设计要点：
//   - 越权保护：所有按 id 操作都强制 user_id 匹配（防止 A 改 B 的地址）。
//   - 默认地址唯一由 partial unique index 保证；应用层走事务切换。
//   - CreatedAt / UpdatedAt 用 unix nano（handler 层转 RFC3339）。
package address

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record 映射 addresses 表行。
type Record struct {
	ID        int64
	UserID    int64
	Recipient string
	Phone     string
	Detail    string
	Lat       *float64
	Lng       *float64
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ErrNotFound 是查询/修改/删除无结果时的哨兵（也用于越权）。
var ErrNotFound = errors.New("address: not found")

// Repo 是 addresses 表的仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const baseSelect = `
	SELECT id, user_id, recipient, phone, detail, lat, lng, is_default,
	       created_at, updated_at
	FROM addresses`

// Create 插入一条地址（asDefault=false）。
func (r *Repo) Create(ctx context.Context, a *Record) error {
	return r.createInternal(ctx, a, false)
}

// CreateDefault 插入并设为默认（事务内先清同用户其他默认）。
func (r *Repo) CreateDefault(ctx context.Context, a *Record) error {
	return r.createInternal(ctx, a, true)
}

func (r *Repo) createInternal(ctx context.Context, a *Record, asDefault bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if asDefault {
		if _, err := tx.Exec(ctx,
			`UPDATE addresses SET is_default = FALSE, updated_at = NOW()
			   WHERE user_id = $1 AND is_default = TRUE`, a.UserID); err != nil {
			return fmt.Errorf("clear default: %w", err)
		}
	}

	const q = `
		INSERT INTO addresses (user_id, recipient, phone, detail, lat, lng, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	if err := tx.QueryRow(ctx, q,
		a.UserID, a.Recipient, a.Phone, a.Detail, a.Lat, a.Lng, asDefault,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return fmt.Errorf("insert address: %w", err)
	}
	a.IsDefault = asDefault
	return tx.Commit(ctx)
}

// ListByUser 列出某用户地址（默认地址优先 + 创建时间倒序）。
func (r *Repo) ListByUser(ctx context.Context, userID int64) ([]*Record, error) {
	q := baseSelect + ` WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer rows.Close()
	out := []*Record{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CountByUser 统计某用户地址数量（用于 5 上限校验）。
func (r *Repo) CountByUser(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM addresses WHERE user_id = $1`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count addresses: %w", err)
	}
	return n, nil
}

// GetByID 按 id 查询；userID 不匹配返回 NotFound（避免越权）。
func (r *Repo) GetByID(ctx context.Context, id, userID int64) (*Record, error) {
	q := baseSelect + ` WHERE id = $1 AND user_id = $2`
	a, err := scan(r.pool.QueryRow(ctx, q, id, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

// Update 修改地址（recipient/phone/detail/lat/lng）；userID 不匹配返回 NotFound。
// 不修改 is_default；如需切换默认请用 SetDefault。
func (r *Repo) Update(ctx context.Context, a *Record) error {
	const q = `
		UPDATE addresses
		   SET recipient = $3, phone = $4, detail = $5, lat = $6, lng = $7,
		       updated_at = NOW()
		 WHERE id = $1 AND user_id = $2
		 RETURNING updated_at`
	if err := r.pool.QueryRow(ctx, q,
		a.ID, a.UserID, a.Recipient, a.Phone, a.Detail, a.Lat, a.Lng,
	).Scan(&a.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("update address: %w", err)
	}
	return nil
}

// SetDefault 切换默认地址：事务内先清同用户其他默认，再 UPDATE 本条。
func (r *Repo) SetDefault(ctx context.Context, id, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. 清掉同用户其他默认
	if _, err := tx.Exec(ctx,
		`UPDATE addresses SET is_default = FALSE, updated_at = NOW()
		   WHERE user_id = $1 AND is_default = TRUE AND id <> $2`, userID, id); err != nil {
		return fmt.Errorf("clear default: %w", err)
	}
	// 2. 置本条为默认
	tag, err := tx.Exec(ctx,
		`UPDATE addresses SET is_default = TRUE, updated_at = NOW()
		   WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// Delete 删地址；userID 不匹配返回 NotFound。
func (r *Repo) Delete(ctx context.Context, id, userID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM addresses WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scan 把 pgx.Row 扫描到 Record。
func scan(row pgx.Row) (*Record, error) {
	a := &Record{}
	if err := row.Scan(
		&a.ID, &a.UserID, &a.Recipient, &a.Phone, &a.Detail,
		&a.Lat, &a.Lng, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return a, nil
}