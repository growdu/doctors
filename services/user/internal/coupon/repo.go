// Package coupon 实现 user-service 优惠券的数据访问层。
//
// 设计要点：
//   - coupons 表是券模板（平台发券），user_coupons 是用户领取实例。
//   - 越权保护：所有按 id 查询/修改/删除都强制 user_id 匹配。
//   - status 由 used_at + expires_at 推导（应用层）。
package coupon

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------- coupon template ----------

// Record 是 coupons 表行（平台发券）。
type Record struct {
	ID         int64
	Name       string
	Type       string  // amount_off | percent_off | full_off
	Value      float64
	Threshold  float64
	ValidFrom  time.Time
	ValidUntil time.Time
	Stock      int
	Status     string  // active | inactive
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ErrCouponNotFound 是查询无结果哨兵。
var ErrCouponNotFound = errors.New("coupon: template not found")

// ErrNoStock 是库存不足哨兵。
var ErrNoStock = errors.New("coupon: stock exhausted")

// ---------- user coupon ----------

// UserCoupon 是 user_coupons 表行（用户领取实例）。
type UserCoupon struct {
	ID         int64
	UserID     int64
	CouponID   int64
	Status     string  // unused | used | expired（读取时重算）
	ExpiresAt  time.Time
	UsedAt     *time.Time
	ClaimedAt  time.Time
}

// ErrUserCouponNotFound 是 user_coupon 查询无结果哨兵。
var ErrUserCouponNotFound = errors.New("coupon: user coupon not found")

// ErrAlreadyClaimed 是重复领取哨兵（uq_user_coupons_user_coupon 触发）。
var ErrAlreadyClaimed = errors.New("coupon: already claimed")

// Repo 是优惠券 + 用户券仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// ---------- coupon template queries ----------

const baseSelectCpn = `
	SELECT id, name, type, value, threshold, valid_from, valid_until,
	       stock, status, created_at, updated_at
	FROM coupons`

// ListActive 列出 active 优惠券模板（按 valid_from DESC + id DESC）。
func (r *Repo) ListActive(ctx context.Context, limit, offset int) ([]*Record, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	q := baseSelectCpn + ` WHERE status = 'active' ORDER BY valid_from DESC, id DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list coupons: %w", err)
	}
	defer rows.Close()
	out := []*Record{}
	for rows.Next() {
		c, err := scanCpn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetByID 按 id 查询券模板。
func (r *Repo) GetByID(ctx context.Context, id int64) (*Record, error) {
	q := baseSelectCpn + ` WHERE id = $1`
	c, err := scanCpn(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCouponNotFound
		}
		return nil, err
	}
	return c, nil
}

// DecrStock 原子扣减 stock（事务内调用，返回新 stock）。
// 若 stock 已为 0 返回 ErrNoStock；不存在返回 ErrCouponNotFound。
func (r *Repo) DecrStock(ctx context.Context, id int64) (int, error) {
	const q = `
		UPDATE coupons
		   SET stock = stock - 1, updated_at = NOW()
		 WHERE id = $1 AND status = 'active' AND stock > 0
		 RETURNING stock`
	var newStock int
	err := r.pool.QueryRow(ctx, q, id).Scan(&newStock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 区分两种 NotFound
			exists := 0
			_ = r.pool.QueryRow(ctx, `SELECT 1 FROM coupons WHERE id = $1`, id).Scan(&exists)
			if exists == 0 {
				return 0, ErrCouponNotFound
			}
			return 0, ErrNoStock
		}
		return 0, fmt.Errorf("decr stock: %w", err)
	}
	return newStock, nil
}

// ---------- user coupon ----------

const baseSelectUC = `
	SELECT id, user_id, coupon_id, status, expires_at, used_at, claimed_at
	FROM user_coupons`

// Claim 创建 user_coupon 行（在事务内扣减 stock + 插入）。
// 已领取返回 ErrAlreadyClaimed。
func (r *Repo) Claim(ctx context.Context, userID, couponID int64) (*UserCoupon, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. 拿券模板（拿 expires_at）
	var expiresAt time.Time
	var status string
	err = tx.QueryRow(ctx,
		`SELECT valid_until, status FROM coupons WHERE id = $1 FOR UPDATE`, couponID,
	).Scan(&expiresAt, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCouponNotFound
		}
		return nil, fmt.Errorf("read coupon: %w", err)
	}
	if status != "active" {
		return nil, ErrNoStock
	}

	// 2. 扣减 stock
	tag, err := tx.Exec(ctx,
		`UPDATE coupons SET stock = stock - 1, updated_at = NOW()
		   WHERE id = $1 AND stock > 0`, couponID)
	if err != nil {
		return nil, fmt.Errorf("decr stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNoStock
	}

	// 3. INSERT user_coupon
	uc := &UserCoupon{
		UserID:    userID,
		CouponID:  couponID,
		Status:    "unused",
		ExpiresAt: expiresAt,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO user_coupons (user_id, coupon_id, status, expires_at)
		VALUES ($1, $2, 'unused', $3)
		RETURNING id, claimed_at`,
		userID, couponID, expiresAt,
	).Scan(&uc.ID, &uc.ClaimedAt)
	if err != nil {
		// 23505 unique_violation
		if isUniqueViolation(err) {
			return nil, ErrAlreadyClaimed
		}
		return nil, fmt.Errorf("insert user_coupon: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return uc, nil
}

// ListByUser 列出某用户的 user_coupons + 关联 coupon 模板；重算 status。
func (r *Repo) ListByUser(ctx context.Context, userID int64) ([]*UserCouponWithTemplate, error) {
	q := `
		SELECT uc.id, uc.user_id, uc.coupon_id, uc.status, uc.expires_at, uc.used_at, uc.claimed_at,
		       c.id, c.name, c.type, c.value, c.threshold
		FROM user_coupons uc
		JOIN coupons c ON c.id = uc.coupon_id
		WHERE uc.user_id = $1
		ORDER BY uc.claimed_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list user_coupons: %w", err)
	}
	defer rows.Close()
	out := []*UserCouponWithTemplate{}
	now := time.Now()
	for rows.Next() {
		uc := &UserCouponWithTemplate{}
		if err := rows.Scan(
			&uc.ID, &uc.UserID, &uc.CouponID, &uc.Status, &uc.ExpiresAt, &uc.UsedAt, &uc.ClaimedAt,
			&uc.Coupon.ID, &uc.Coupon.Name, &uc.Coupon.Type, &uc.Coupon.Value, &uc.Coupon.Threshold,
		); err != nil {
			return nil, err
		}
		uc.Status = deriveStatus(uc.UsedAt, uc.ExpiresAt, now)
		out = append(out, uc)
	}
	return out, rows.Err()
}

// MarkUsed 把 user_coupon 标记为 used；userID 不匹配返回 NotFound；used_at 为空返回 Conflict。
func (r *Repo) MarkUsed(ctx context.Context, id, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. 拿状态（锁）
	var status string
	var expiresAt time.Time
	var usedAt *time.Time
	err = tx.QueryRow(ctx,
		`SELECT status, expires_at, used_at FROM user_coupons
		   WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, userID,
	).Scan(&status, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserCouponNotFound
		}
		return fmt.Errorf("read user_coupon: %w", err)
	}

	// 2. 校验
	if usedAt != nil {
		return ErrAlreadyClaimed // 已被使用
	}
	if status == "expired" || (!expiresAt.IsZero() && time.Now().After(expiresAt)) {
		return ErrUserCouponNotFound // expired / 越权保护也复用这个
	}

	// 3. 标记 used
	_, err = tx.Exec(ctx,
		`UPDATE user_coupons SET status = 'used', used_at = NOW()
		   WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("mark used: %w", err)
	}
	return tx.Commit(ctx)
}

// ---------- helpers ----------

// scanCpn 扫描券模板。
func scanCpn(row pgx.Row) (*Record, error) {
	c := &Record{}
	if err := row.Scan(
		&c.ID, &c.Name, &c.Type, &c.Value, &c.Threshold,
		&c.ValidFrom, &c.ValidUntil, &c.Stock, &c.Status,
		&c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return c, nil
}

// deriveStatus 按 used_at + expires_at 推导 status（unused / used / expired）。
func deriveStatus(usedAt *time.Time, expiresAt time.Time, now time.Time) string {
	if usedAt != nil {
		return "used"
	}
	if !expiresAt.IsZero() && now.After(expiresAt) {
		return "expired"
	}
	return "unused"
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

// UserCouponWithTemplate 是 ListByUser 返回的复合视图（含券模板）。
type UserCouponWithTemplate struct {
	UserCoupon
	Coupon Record `json:"coupon"`
}