// Package repo 是 admin-service 的数据访问层。
//
// 设计要点：
//   - admin 不持有业务数据，但持有 work_orders。
//   - 其他服务的数据通过 internal/clients 调用，不在本包。
//   - dashboard overview 走直读 PG（聚合查询效率高于 N 次远程调用）。
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkOrder 映射 work_orders 表行（admin-web 客服工单模块）。
//
// 字段含义：
//   - SubjectID + SubjectType：polymorphic 关联到订单 / 用户 / 陪诊师 / 系统；
//     SubjectType 为空表示系统级工单。
//   - AssigneeID：受理客服 / admin 的 user id。
//   - Resolution + ClosedAt：仅 status='closed' 时有值。
type WorkOrder struct {
	ID          int64
	UserID      int64
	Category    string
	Priority    string
	Status      string
	SubjectID   *int64
	SubjectType *string
	AssigneeID  *int64
	Title       string
	Content     string
	Resolution  *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	SLADueAt    *time.Time
}

// ErrWorkOrderNotFound 哨兵错误：工单不存在。
var ErrWorkOrderNotFound = errors.New("repo: work order not found")

// WorkOrderListFilter 是 List 的筛选条件。
type WorkOrderListFilter struct {
	Status      string
	Category    string
	AssigneeID  *int64
	SubjectType string
	Page        int
	PageSize    int
}

// WorkOrderRepo work_orders 仓储。
type WorkOrderRepo struct {
	pool *pgxpool.Pool
}

// NewWorkOrderRepo 构造仓储。
func NewWorkOrderRepo(pool *pgxpool.Pool) *WorkOrderRepo { return &WorkOrderRepo{pool: pool} }

// Create 插入工单；返回 ID + 时间戳。
func (r *WorkOrderRepo) Create(ctx context.Context, w *WorkOrder) error {
	if w.Status == "" {
		w.Status = "pending"
	}
	if w.Priority == "" {
		w.Priority = "P2"
	}
	const q = `
		INSERT INTO work_orders
		  (user_id, category, priority, status, subject_id, subject_type,
		   assignee_id, title, content, sla_due_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q,
		w.UserID, w.Category, w.Priority, w.Status,
		w.SubjectID, w.SubjectType,
		w.AssigneeID, w.Title, w.Content, w.SLADueAt,
	).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

// FindByID 按主键查询；不存在 → ErrWorkOrderNotFound。
func (r *WorkOrderRepo) FindByID(ctx context.Context, id int64) (*WorkOrder, error) {
	const q = `
		SELECT id, user_id, category, priority, status, subject_id, subject_type,
		       assignee_id, title, content, resolution,
		       created_at, updated_at, closed_at, sla_due_at
		FROM work_orders WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	w, err := scanWorkOrder(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}
	return w, nil
}

// List 按筛选条件分页查询（status / category / assignee / subject_type）。
func (r *WorkOrderRepo) List(ctx context.Context, f WorkOrderListFilter) ([]*WorkOrder, error) {
	q := `
		SELECT id, user_id, category, priority, status, subject_id, subject_type,
		       assignee_id, title, content, resolution,
		       created_at, updated_at, closed_at, sla_due_at
		FROM work_orders WHERE 1=1`
	args := []any{}
	i := 1
	if f.Status != "" {
		q += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, f.Status)
		i++
	}
	if f.Category != "" {
		q += fmt.Sprintf(" AND category = $%d", i)
		args = append(args, f.Category)
		i++
	}
	if f.AssigneeID != nil {
		q += fmt.Sprintf(" AND assignee_id = $%d", i)
		args = append(args, *f.AssigneeID)
		i++
	}
	if f.SubjectType != "" {
		q += fmt.Sprintf(" AND subject_type = $%d", i)
		args = append(args, f.SubjectType)
		i++
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list work orders: %w", err)
	}
	defer rows.Close()
	out := make([]*WorkOrder, 0, f.PageSize)
	for rows.Next() {
		w, err := scanWorkOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// Assign 分配客服（status → assigned）。
func (r *WorkOrderRepo) Assign(ctx context.Context, id, adminID int64) error {
	const q = `UPDATE work_orders SET assignee_id = $2, status = 'assigned',
	             updated_at = NOW() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, adminID)
	if err != nil {
		return fmt.Errorf("assign: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWorkOrderNotFound
	}
	return nil
}

// Resolve 关单（status → closed + resolution + closed_at）。
func (r *WorkOrderRepo) Resolve(ctx context.Context, id int64, resolution string) error {
	const q = `UPDATE work_orders SET status = 'closed', resolution = $2,
	             closed_at = NOW(), updated_at = NOW() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, resolution)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWorkOrderNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// scanWorkOrder 把 PG 行扫描成 *WorkOrder。
func scanWorkOrder(row rowScanner) (*WorkOrder, error) {
	w := &WorkOrder{}
	err := row.Scan(
		&w.ID, &w.UserID, &w.Category, &w.Priority, &w.Status,
		&w.SubjectID, &w.SubjectType, &w.AssigneeID,
		&w.Title, &w.Content, &w.Resolution,
		&w.CreatedAt, &w.UpdatedAt, &w.ClosedAt, &w.SLADueAt,
	)
	if err != nil {
		return nil, err
	}
	return w, nil
}