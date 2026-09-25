package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OverviewStats dashboard overview 聚合数据。
type OverviewStats struct {
	TodayGMV       float64   `json:"today_gmv"`
	TodayOrders    int       `json:"today_orders"`
	RefundRate     float64   `json:"refund_rate"`     // 0~1
	PendingEscorts int       `json:"pending_escorts"` // 待审核陪诊师
	PendingRefunds int       `json:"pending_refunds"` // 待审核退款
	OpenWorkOrders int       `json:"open_work_orders"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// ReportsRepo 是 dashboard 报表的直读仓储。
type ReportsRepo struct {
	pool *pgxpool.Pool
}

// NewReportsRepo 构造。
func NewReportsRepo(pool *pgxpool.Pool) *ReportsRepo { return &ReportsRepo{pool: pool} }

// Overview 返回 dashboard overview 聚合数据。
//
// 设计要点：
//   - 直读 PG（不走内部 client），因为聚合查询效率高于 N 次远程调用。
//   - 部分表（escort_profiles / refunds / work_orders）若不存在（其它 plan 未上线），
//     相关字段降级为 0，不 panic。
func (r *ReportsRepo) Overview(ctx context.Context) (*OverviewStats, error) {
	stats := &OverviewStats{GeneratedAt: time.Now()}

	// 今日 GMV + 订单数（排除 created / canceled）
	const q1 = `
		SELECT COALESCE(SUM(final_amount), 0), COUNT(*)
		FROM orders
		WHERE created_at >= CURRENT_DATE
		  AND status NOT IN ('created','canceled')`
	if err := r.pool.QueryRow(ctx, q1).Scan(&stats.TodayGMV, &stats.TodayOrders); err != nil {
		return nil, fmt.Errorf("overview gmv: %w", err)
	}

	// 退款率（近 30 天退款 / 总订单）；refunds 表不存在 → 0
	if exists(ctx, r.pool, "refunds") {
		_ = r.pool.QueryRow(ctx, `
			SELECT COALESCE(
			  (SELECT COUNT(*)::float FROM refunds
			     WHERE created_at >= NOW() - INTERVAL '30 days') /
			  NULLIF((SELECT COUNT(*) FROM orders
			           WHERE created_at >= NOW() - INTERVAL '30 days'), 0),
			  0
			)`).Scan(&stats.RefundRate)
	}

	// 待审核陪诊师（escort_profiles 表若存在）
	if exists(ctx, r.pool, "escort_profiles") {
		_ = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM escort_profiles WHERE audit_status = 'pending'`).
			Scan(&stats.PendingEscorts)
	}

	// 待审核退款
	if exists(ctx, r.pool, "refunds") {
		_ = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM refunds WHERE status = 'created'`).
			Scan(&stats.PendingRefunds)
	}

	// 未关闭工单
	if exists(ctx, r.pool, "work_orders") {
		_ = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM work_orders WHERE status NOT IN ('closed')`).
			Scan(&stats.OpenWorkOrders)
	}

	return stats, nil
}

// exists 判断表是否存在（避免硬依赖未上线的表）。
func exists(ctx context.Context, pool *pgxpool.Pool, table string) bool {
	var found bool
	_ = pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1)`,
		table).Scan(&found)
	return found
}