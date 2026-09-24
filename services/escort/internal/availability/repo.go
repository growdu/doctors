package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo 是 escort_availabilities 表的 pgx 仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// New 构造仓储。
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// Create 插入新时段；时段冲突返回 ErrAvailabilityConflict。
func (r *Repo) Create(ctx context.Context, escortID int64, startAt, endAt time.Time) (*Availability, error) {
	// 1. 时段冲突检测（同一 escort，status != canceled，时间重叠）
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM escort_availabilities
			 WHERE escort_id = $1 AND deleted_at IS NULL
			   AND status != 'canceled'
			   AND start_at < $3 AND end_at > $2
		)`, escortID, startAt, endAt).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check conflict: %w", err)
	}
	if exists {
		return nil, ErrAvailabilityConflict
	}

	a := &Availability{
		EscortID: escortID,
		StartAt:  startAt,
		EndAt:    endAt,
		Status:   StatusAvailable,
	}
	const q = `
		INSERT INTO escort_availabilities (escort_id, start_at, end_at, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	err = r.pool.QueryRow(ctx, q, a.EscortID, a.StartAt, a.EndAt, string(a.Status)).Scan(&a.ID)
	if err != nil {
		return nil, fmt.Errorf("create availability: %w", err)
	}
	return a, nil
}

// FindByID 按主键查。
func (r *Repo) FindByID(ctx context.Context, id int64) (*Availability, error) {
	const q = baseSelect + ` WHERE id = $1 AND deleted_at IS NULL`
	return r.scanOne(r.pool.QueryRow(ctx, q, id))
}

// Delete 软删除时段（仅 owner + status=available 可删）。
func (r *Repo) Delete(ctx context.Context, id int64, escortID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE escort_availabilities
		   SET deleted_at = NOW()
		 WHERE id = $1 AND escort_id = $2 AND status = 'available' AND deleted_at IS NULL`,
		id, escortID)
	if err != nil {
		return fmt.Errorf("delete availability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotOwner
	}
	return nil
}

// ListByEscort 返回某 escort 的所有未删除时段，按 start_at ASC。
func (r *Repo) ListByEscort(ctx context.Context, escortID int64) ([]*Availability, error) {
	const q = baseSelect + ` WHERE escort_id = $1 AND deleted_at IS NULL
	                          ORDER BY start_at ASC`
	rows, err := r.pool.Query(ctx, q, escortID)
	if err != nil {
		return nil, fmt.Errorf("list by escort: %w", err)
	}
	defer rows.Close()
	out := make([]*Availability, 0)
	for rows.Next() {
		a, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAvailableByTime 返回时段与 [startAt, endAt) 重叠的 available 时段。
// 用于 order-service 在生成候选时筛选"该时段可服务的 escort"。
func (r *Repo) ListAvailableByTime(ctx context.Context, startAt, endAt time.Time, limit int) ([]*Availability, error) {
	const q = baseSelect + ` WHERE status = 'available'
	                          AND start_at < $2 AND end_at > $1
	                          AND deleted_at IS NULL
	                          ORDER BY start_at ASC LIMIT $3`
	rows, err := r.pool.Query(ctx, q, startAt, endAt, limit)
	if err != nil {
		return nil, fmt.Errorf("list available by time: %w", err)
	}
	defer rows.Close()
	out := make([]*Availability, 0)
	for rows.Next() {
		a, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// BookByOrder 把 available 时段切到 booked + 写 booked_order_id（order-service ConfirmAccept 调）。
func (r *Repo) BookByOrder(ctx context.Context, id int64, orderID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE escort_availabilities
		   SET status = 'booked', booked_order_id = $2, updated_at = NOW()
		 WHERE id = $1 AND status = 'available' AND deleted_at IS NULL`,
		id, orderID)
	if err != nil {
		return fmt.Errorf("book availability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotBookable
	}
	return nil
}

// ReleaseByOrder 把 booked 时段恢复为 available + 清 booked_order_id（order-service 取消/拒接时调）。
// v1 简化：不删除时段，只恢复 available 状态（让陪诊师可重新接受其它订单）。
func (r *Repo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE escort_availabilities
		   SET status = 'available', booked_order_id = NULL, updated_at = NOW()
		 WHERE booked_order_id = $1 AND deleted_at IS NULL`,
		orderID)
	if err != nil {
		return fmt.Errorf("release availability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAvailabilityNotFound
	}
	return nil
}

// baseSelect 是 SELECT 子句。
const baseSelect = `
	SELECT id, escort_id, start_at, end_at, status, booked_order_id
	FROM escort_availabilities`

func (r *Repo) scanOne(row pgx.Row) (*Availability, error) {
	a := &Availability{}
	if err := row.Scan(&a.ID, &a.EscortID, &a.StartAt, &a.EndAt, &a.Status, &a.BookedOrderID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAvailabilityNotFound
		}
		return nil, fmt.Errorf("scan availability: %w", err)
	}
	return a, nil
}

func (r *Repo) scanRow(rows pgx.Rows) (*Availability, error) {
	a := &Availability{}
	if err := rows.Scan(&a.ID, &a.EscortID, &a.StartAt, &a.EndAt, &a.Status, &a.BookedOrderID); err != nil {
		return nil, err
	}
	return a, nil
}