// Package repo 提供 order-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc 工具链）。
//   - UpdateStatus 用 version 做乐观锁；并发安全靠 DB 不靠 Redis。
//   - 状态变更走 UpdateStatus + InsertEvent；service 层负责把"原 status"先读出再调用，
//     同一事务由 shared/db.WithTx 包装（阶段 3.5 补）。
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
	ID             int64
	OrderNo        string
	PatientID      int64
	EscortID       *int64
	HospitalID     int64
	PackageID      int64
	ServiceStartAt any // 借 `time.Time`；这里用 any 是为避免顶层 import 复杂化
	Amount         float64
	FinalAmount    float64
	Status         string
	Version        int
	LockOwner      *int64     // pending_acceptance 锁单持有者；nil 表示未锁
	LockExpireAt   *time.Time // 锁单超时时间；nil 表示未锁
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

// ErrInvalidStateForLock 是 LockForAccept 时订单状态不在 matching 的哨兵。
var ErrInvalidStateForLock = errors.New("repo: order not in matching state for lock")

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

// LockForAccept 在 matching 状态下加锁单 + 状态切到 pending_acceptance。
// 同时校验 version；lockExpireAt 写入 orders 表。
func (r *OrderRepo) LockForAccept(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'pending_acceptance', lock_owner = $1, lock_expire_at = $2,
		       version = version + 1, updated_at = NOW()
		 WHERE id = $3 AND version = $4 AND status = 'matching' AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, escortID, expireAt, id, expectVersion)
	if err != nil {
		return fmt.Errorf("lock for accept: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 区分版本冲突与状态非法
		var curStatus string
		var curVer int
		_ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
		if curVer != expectVersion {
			return ErrVersionConflict
		}
		return ErrInvalidStateForLock
	}
	return nil
}

// ReleaseLock 把 pending_acceptance 回退到 matching；清除锁单字段。
func (r *OrderRepo) ReleaseLock(ctx context.Context, id int64, expectVersion int) error {
	const q = `
		UPDATE orders
		   SET status = 'matching', lock_owner = NULL, lock_expire_at = NULL,
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND status = 'pending_acceptance' AND version = $2 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, expectVersion)
	if err != nil {
		return fmt.Errorf("release lock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrVersionConflict
	}
	return nil
}

// LockExpired 返回已过期的锁单（按 lock_expire_at 升序）；给 §4.2 定时任务用。
func (r *OrderRepo) LockExpired(ctx context.Context, now time.Time, limit int) ([]*Order, error) {
	const q = baseSelect + ` WHERE status = 'pending_acceptance'
	                              AND lock_expire_at IS NOT NULL AND lock_expire_at < $1
	                              ORDER BY lock_expire_at ASC
	                              LIMIT $2`
	rows, err := r.pool.Query(ctx, q, now, limit)
	if err != nil {
		return nil, fmt.Errorf("find expired locks: %w", err)
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

// baseSelect 是 SELECT 子句。
const baseSelect = `
	SELECT id, order_no, patient_id, escort_id, hospital_id, package_id,
	       service_start_at, amount, final_amount, status, version,
	       lock_owner, lock_expire_at
	FROM orders`

// scanOne 把单行扫描为 *Order。
func (r *OrderRepo) scanOne(row pgx.Row) (*Order, error) {
	o := &Order{}
	if err := row.Scan(
		&o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
		&o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
		&o.LockOwner, &o.LockExpireAt,
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
		&o.LockOwner, &o.LockExpireAt,
	); err != nil {
		return nil, err
	}
	return o, nil
}