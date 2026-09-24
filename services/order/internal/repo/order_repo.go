// Package repo 提供 order-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc 工具链）。
//   - UpdateStatus 用 version 做乐观锁；并发安全靠 DB 不靠 Redis。
//   - 状态变更走 UpdateStatus + InsertEvent；service 层负责把"原 status"先读出再调用，
//     同一事务由 shared/db.WithTx 包装（阶段 3.5 补）。
//
// v1.1（order-matching-redesign）：
//   - 删除 LockOwner / LockExpireAt 字段（v1 抢单锁单用）
//   - 新增 SelectedEscortID / EscortPendingExpireAt 字段（v1.1 选人 +30s 确认窗口用）
//   - 删 LockForAccept / ReleaseLock / LockExpired 3 方法
//   - 增 SelectForEscort / ConfirmByEscort / RejectByEscort / PendingExpired 4 方法
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Order 映射 orders 表行。
type Order struct {
	ID                     int64
	OrderNo                string
	PatientID              int64
	EscortID               *int64
	HospitalID             int64
	PackageID              int64
	ServiceStartAt         any
	Amount                 float64
	FinalAmount            float64
	Status                 string
	Version                int
	SelectedEscortID       *int64     // escort_pending_acceptance 已选陪诊师；nil 表示未选
	EscortPendingExpireAt  *time.Time // 30s 确认窗口超时时间；nil 表示未进 escort_pending_acceptance
}

// OrderEvent 映射 order_events 表行。
type OrderEvent struct {
	ID         int64
	OrderID    int64
	FromStatus *string
	ToStatus   string
	ActorID    *int64
	Payload    []byte
}

// ErrOrderNotFound 是查询无结果时的哨兵。
var ErrOrderNotFound = errors.New("repo: order not found")

// ErrVersionConflict 是乐观锁冲突（version 不匹配）。
var ErrVersionConflict = errors.New("repo: order version conflict")

// ErrInvalidStateForSelect 是 SelectForEscort 时订单状态不在 selecting_escort 的哨兵。
var ErrInvalidStateForSelect = errors.New("repo: order not in selecting_escort state for select")

// ErrInvalidStateForConfirm 是 ConfirmByEscort 时订单状态不在 escort_pending_acceptance 的哨兵。
var ErrInvalidStateForConfirm = errors.New("repo: order not in escort_pending_acceptance state for confirm")

// ErrInvalidStateForReject 是 RejectByEscort 时订单状态不在 escort_pending_acceptance 的哨兵。
var ErrInvalidStateForReject = errors.New("repo: order not in escort_pending_acceptance state for reject")

// ErrSelectedEscortMismatch 是 ConfirmByEscort 时 escortID != selected_escort_id 的哨兵。
var ErrSelectedEscortMismatch = errors.New("repo: escort_id does not match selected_escort_id")

// ErrInvitationExpired 是 ConfirmByEscort 时 escort_pending_expire_at 已过的哨兵。
var ErrInvitationExpired = errors.New("repo: escort invitation expired")

// OrderRepo 是 orders + order_events 表的仓储。
type OrderRepo struct {
	pool *pgxpool.Pool
}

// NewOrderRepo 构造仓储。
func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo { return &OrderRepo{pool: pool} }

// Create 插入新订单；ID / version 由 DB 回写。
func (r *OrderRepo) Create(ctx context.Context, o *Order) error {
	const q = `
		INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
		                    service_start_at, amount, final_amount, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, version`
	return r.pool.QueryRow(ctx, q,
		o.OrderNo, o.PatientID, o.HospitalID, o.PackageID,
		o.ServiceStartAt, o.Amount, o.FinalAmount, o.Status,
	).Scan(&o.ID, &o.Version)
}

// FindByID 按主键查找。
func (r *OrderRepo) FindByID(ctx context.Context, id int64) (*Order, error) {
	const q = baseSelect + ` WHERE id = $1 AND deleted_at IS NULL`
	return r.scanOne(r.pool.QueryRow(ctx, q, id))
}

// ListByPatient 按患者分页查订单，按 created_at DESC。
func (r *OrderRepo) ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*Order, error) {
	const q = baseSelect + ` WHERE patient_id = $1 AND deleted_at IS NULL
	                         ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, patientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()
	out := make([]*Order, 0)
	for rows.Next() {
		o, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListByEscort 按 escort 分页查订单（v1.1 escort-order-ext plan）。
//   - statusFilter == "" → 查 escort_id = $1（已确认接单的订单）
//   - statusFilter == "invitations" → 查 selected_escort_id = $1 AND status = 'escort_pending_acceptance'（待确认邀请）
//   - 其他值 → 按精确 status 过滤
// 按 escort_pending_expire_at ASC（30s 倒计时用），无邀请时按 created_at DESC。
func (r *OrderRepo) ListByEscort(ctx context.Context, escortID int64, statusFilter string, limit, offset int) ([]*Order, error) {
	var q string
	var args []any
	switch statusFilter {
	case "invitations":
		q = baseSelect + ` WHERE selected_escort_id = $1
		                    AND status = 'escort_pending_acceptance'
		                    AND deleted_at IS NULL
		                    ORDER BY escort_pending_expire_at ASC
		                    LIMIT $2 OFFSET $3`
		args = []any{escortID, limit, offset}
	case "":
		q = baseSelect + ` WHERE escort_id = $1 AND deleted_at IS NULL
		                    ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []any{escortID, limit, offset}
	default:
		q = baseSelect + ` WHERE escort_id = $1 AND status = $2
		                    AND deleted_at IS NULL
		                    ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = []any{escortID, statusFilter, limit, offset}
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list by escort: %w", err)
	}
	defer rows.Close()
	out := make([]*Order, 0)
	for rows.Next() {
		o, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateStatus 用乐观锁更新状态；escortID 可空（accepted 时填）。
func (r *OrderRepo) UpdateStatus(ctx context.Context, id int64, toStatus string, expectVersion int, escortID *int64) error {
	const q = `
		UPDATE orders
		   SET status = $1, version = version + 1, updated_at = NOW(),
		       escort_id = COALESCE($2, escort_id)
		 WHERE id = $3 AND version = $4 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, toStatus, escortID, id, expectVersion)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrVersionConflict
	}
	return nil
}

// InsertEvent 写入一条 order_events。
func (r *OrderRepo) InsertEvent(ctx context.Context, orderID int64, fromStatus *string, toStatus string, actorID *int64, payload []byte) error {
	const q = `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, payload)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, q, orderID, fromStatus, toStatus, actorID, payload)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// ListEvents 取一个订单的所有事件，按时间正序。
func (r *OrderRepo) ListEvents(ctx context.Context, orderID int64) ([]*OrderEvent, error) {
	const q = `
		SELECT id, order_id, from_status, to_status, actor_id, payload
		  FROM order_events WHERE order_id = $1 ORDER BY created_at ASC, id ASC`
	rows, err := r.pool.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	out := make([]*OrderEvent, 0)
	for rows.Next() {
		var e OrderEvent
		if err := rows.Scan(&e.ID, &e.OrderID, &e.FromStatus, &e.ToStatus, &e.ActorID, &e.Payload); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// SelectForEscort 在 selecting_escort 状态下：写 selected_escort_id + escort_pending_expire_at + 切到 escort_pending_acceptance。
// 校验 version + status='selecting_escort'。
func (r *OrderRepo) SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'escort_pending_acceptance',
		       selected_escort_id = $1,
		       escort_pending_expire_at = $2,
		       version = version + 1,
		       updated_at = NOW()
		 WHERE id = $3 AND version = $4 AND status = 'selecting_escort' AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, escortID, expireAt, id, expectVersion)
	if err != nil {
		return fmt.Errorf("select for escort: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var curStatus string
		var curVer int
		_ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
		if curVer != expectVersion {
			return ErrVersionConflict
		}
		return ErrInvalidStateForSelect
	}
	return nil
}

// ConfirmByEscort 在 escort_pending_acceptance 状态下：写 escort_id = selected_escort_id + 切到 accepted。
// 校验：
//   - status='escort_pending_acceptance'
//   - selected_escort_id = $1（防止误确认）
//   - escort_pending_expire_at > now()（未超时）
func (r *OrderRepo) ConfirmByEscort(ctx context.Context, id int64, escortID int64, now time.Time, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'accepted',
		       escort_id = $1,
		       selected_escort_id = NULL,
		       escort_pending_expire_at = NULL,
		       version = version + 1,
		       updated_at = NOW()
		 WHERE id = $2 AND version = $3
		   AND status = 'escort_pending_acceptance'
		   AND selected_escort_id = $1
		   AND deleted_at IS NULL
		   AND (escort_pending_expire_at IS NULL OR escort_pending_expire_at > $4)`
	tag, err := r.pool.Exec(ctx, q, escortID, id, expectVersion, now)
	if err != nil {
		return fmt.Errorf("confirm by escort: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 区分原因：超时 / selected_escort_id 不匹配 / 版本冲突 / 状态非法
		var curStatus string
		var curVer int
		var curSelected *int64
		var curExpire *time.Time
		_ = r.pool.QueryRow(ctx, "SELECT status, version, selected_escort_id, escort_pending_expire_at FROM orders WHERE id=$1", id).
			Scan(&curStatus, &curVer, &curSelected, &curExpire)
		if curVer != expectVersion {
			return ErrVersionConflict
		}
		if curStatus != "escort_pending_acceptance" {
			return ErrInvalidStateForConfirm
		}
		if curSelected == nil || *curSelected != escortID {
			return ErrSelectedEscortMismatch
		}
		if curExpire != nil && !curExpire.After(now) {
			return ErrInvitationExpired
		}
		return ErrInvalidStateForConfirm
	}
	return nil
}

// RejectByEscort 在 escort_pending_acceptance 状态下：清 selected_escort_id / escort_pending_expire_at + 回退 selecting_escort。
// 校验 status + selected_escort_id（防止误拒）。
func (r *OrderRepo) RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'selecting_escort',
		       selected_escort_id = NULL,
		       escort_pending_expire_at = NULL,
		       version = version + 1,
		       updated_at = NOW()
		 WHERE id = $1 AND version = $2
		   AND status = 'escort_pending_acceptance'
		   AND (selected_escort_id = $3 OR selected_escort_id IS NULL)
		   AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, expectVersion, escortID)
	if err != nil {
		return fmt.Errorf("reject by escort: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var curStatus string
		var curVer int
		_ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
		if curVer != expectVersion {
			return ErrVersionConflict
		}
		return ErrInvalidStateForReject
	}
	return nil
}

// PendingExpired 返回已过期的 escort_pending_acceptance 订单（按 escort_pending_expire_at 升序）；给 §4.2 定时任务用。
func (r *OrderRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*Order, error) {
	const q = baseSelect + ` WHERE status = 'escort_pending_acceptance'
	                              AND selected_escort_id IS NOT NULL
	                              AND escort_pending_expire_at IS NOT NULL
	                              AND escort_pending_expire_at < $1
	                              ORDER BY escort_pending_expire_at ASC
	                              LIMIT $2`
	rows, err := r.pool.Query(ctx, q, now, limit)
	if err != nil {
		return nil, fmt.Errorf("find expired invitations: %w", err)
	}
	defer rows.Close()
	out := make([]*Order, 0)
	for rows.Next() {
		o, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// baseSelect 是 SELECT 子句（v1.1：selected_escort_id / escort_pending_expire_at 替换 lock_owner / lock_expire_at）。
const baseSelect = `
	SELECT id, order_no, patient_id, escort_id, hospital_id, package_id,
	       service_start_at, amount, final_amount, status, version,
	       selected_escort_id, escort_pending_expire_at
	FROM orders`

// scanOne 把单行扫描为 *Order。
func (r *OrderRepo) scanOne(row pgx.Row) (*Order, error) {
	o := &Order{}
	if err := row.Scan(
		&o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
		&o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
		&o.SelectedEscortID, &o.EscortPendingExpireAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("scan order: %w", err)
	}
	return o, nil
}

// scanRow 把 pgx.Rows 的一行扫描为 *Order。
func (r *OrderRepo) scanRow(rows pgx.Rows) (*Order, error) {
	o := &Order{}
	if err := rows.Scan(
		&o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
		&o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
		&o.SelectedEscortID, &o.EscortPendingExpireAt,
	); err != nil {
		return nil, err
	}
	return o, nil
}