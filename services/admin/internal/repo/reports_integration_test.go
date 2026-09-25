//go:build integration
// +build integration

package repo

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReports_Overview_GMV 验证今日 GMV 计算。
func TestReports_Overview_GMV(t *testing.T) {
	pool := setupAdminPool(t)
	patient := seedUser(t, pool, "13800138010", "patient")

	// 创建 2 笔今日 paid 订单
	for i := 0; i < 2; i++ {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
			                    service_start_at, amount, final_amount, status, created_at)
			VALUES ($1, $2, 1, 1, NOW(), 100, 100, 'paid', NOW())`,
			fmt.Sprintf("o-%d", i+1), patient)
		require.NoError(t, err)
	}
	// 1 笔昨日
	_, err := pool.Exec(context.Background(), `
		INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
		                    service_start_at, amount, final_amount, status, created_at)
		VALUES ('o-old', $1, 1, 1, NOW() - INTERVAL '1 day',
		        100, 100, 'paid', NOW() - INTERVAL '1 day')`,
		patient)
	require.NoError(t, err)

	r := NewReportsRepo(pool)
	stats, err := r.Overview(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 200.0, stats.TodayGMV, 0.01, "今日 GMV 应该是 200")
	assert.Equal(t, 2, stats.TodayOrders)
}

// TestReports_Overview_RefundRate 验证退款率（无 refunds 表 → 0）。
func TestReports_Overview_RefundRate(t *testing.T) {
	pool := setupAdminPool(t)
	patient := seedUser(t, pool, "13800138011", "patient")
	for i := 0; i < 10; i++ {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
			                    service_start_at, amount, final_amount, status)
			VALUES ($1, $2, 1, 1, NOW(), 100, 100, 'paid')`,
			fmt.Sprintf("o-%d", i+1), patient)
		require.NoError(t, err)
	}
	r := NewReportsRepo(pool)
	stats, err := r.Overview(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 0.0, stats.RefundRate, 0.01, "无退款表 → 0%")
}

// TestReports_Overview_PendingEscorts_NoTable 验证 escort_profiles 不存在时返回 0。
func TestReports_Overview_PendingEscorts_NoTable(t *testing.T) {
	pool := setupAdminPool(t)
	r := NewReportsRepo(pool)
	stats, err := r.Overview(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, stats.PendingEscorts)
}

// TestReports_Overview_OpenWorkOrders 验证 OpenWorkOrders 计数（work_orders 表若存在）。
func TestReports_Overview_OpenWorkOrders(t *testing.T) {
	pool := setupAdminPool(t)
	user := seedUser(t, pool, "13800138012", "patient")
	r := NewReportsRepo(pool)

	// 创建 2 个 pending + 1 个 closed → OpenWorkOrders 应 = 2
	for i := 0; i < 2; i++ {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO work_orders (user_id, category, status, title, content)
			VALUES ($1, 'complaint', 'pending', $2, 'x')`,
			user, fmt.Sprintf("t-%d", i))
		require.NoError(t, err)
	}
	_, err := pool.Exec(context.Background(), `
		INSERT INTO work_orders (user_id, category, status, title, content, closed_at)
		VALUES ($1, 'complaint', 'closed', 'closed', 'x', NOW())`,
		user)
	require.NoError(t, err)

	stats, err := r.Overview(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, stats.OpenWorkOrders)
}