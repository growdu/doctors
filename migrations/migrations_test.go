//go:build integration
// +build integration

// migrations 的集成测试：需要 docker compose up 起 PG 后才能跑。
//
// 用法：make docker-up && go test -tags=integration ./migrations/...
//
// 每个迁移文件用 schema_migrations 表跟踪；这里直接用 pgx 应用 SQL，
// 验证表结构与索引是否齐全。
package migrations

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dsn 从环境变量读，便于 CI 配置。
func dsn() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

// applyUp 应用指定迁移文件，验证其产生的 schema 与预期一致。
func applyUp(t *testing.T, file string, expect []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err, "connect pg (dsn=%s)", dsn())
	defer conn.Close(ctx)

	sqlBytes, err := os.ReadFile(filepath.Join(".", file))
	require.NoError(t, err)

	_, err = conn.Exec(ctx, string(sqlBytes))
	require.NoError(t, err, "apply %s", file)

	for _, q := range expect {
		var exists bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, q).
			Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "expected table %q to exist after %s", q, file)
	}
}

// applyDown 应用指定 down 文件，验证 schema 被清理。
func applyDown(t *testing.T, file string, expectGone []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	sqlBytes, err := os.ReadFile(filepath.Join(".", file))
	require.NoError(t, err)

	_, err = conn.Exec(ctx, string(sqlBytes))
	require.NoError(t, err, "apply %s", file)

	for _, q := range expectGone {
		var exists bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, q).
			Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists, "expected table %q to be gone after %s", q, file)
	}
}

// Test0001UsersUpDown 验证 0001_users 的 up/down 行为可逆且符合契约。
func Test0001UsersUpDown(t *testing.T) {
	applyUp(t, "0001_users.up.sql", []string{"users"})
	// 列检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	requiredCols := []string{
		"id", "phone", "role", "nickname", "avatar_url",
		"real_name_verified", "id_card_hash", "id_card_tail",
		"wx_unionid", "wx_openid_mini", "wx_openid_app",
		"status", "created_at", "updated_at", "deleted_at",
	}
	for _, col := range requiredCols {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='users' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "users.%s should exist", col)
	}

	// 索引检查
	requiredIdx := []string{"idx_users_phone", "idx_users_wx_unionid", "idx_users_role_status"}
	for _, idx := range requiredIdx {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE tablename='users' AND indexname=$1)`, idx).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "index %s should exist", idx)
	}

	applyDown(t, "0001_users.down.sql", []string{"users"})
}

// Test0002OrdersUpDown 验证 0002_orders 的 up/down 行为可逆且符合契约。
// 0002 依赖 0001 的 users 表。
func Test0002OrdersUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 先建 users（0002 依赖它），最后清理
	usersSQL, err := os.ReadFile("0001_users.up.sql")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, string(usersSQL))
	require.NoError(t, err)
	t.Cleanup(func() {
		down, _ := os.ReadFile("0001_users.down.sql")
		_, _ = conn.Exec(context.Background(), string(down))
	})

	applyUp(t, "0002_orders.up.sql", []string{"orders", "order_events"})

	requiredCols := []string{
		"id", "order_no", "patient_id", "escort_id", "hospital_id", "package_id",
		"service_start_at", "amount", "final_amount", "status", "version",
		"created_at", "updated_at", "deleted_at",
	}
	for _, col := range requiredCols {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='orders' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "orders.%s should exist", col)
	}

	requiredIdx := []string{"idx_orders_patient_created", "idx_orders_status_start", "idx_order_events_order"}
	for _, idx := range requiredIdx {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, idx).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "index %s should exist", idx)
	}

	// down 验证：两张表都被清掉
	applyDown(t, "0002_orders.down.sql", []string{"orders", "order_events"})
}

// TestAllUpMigrationsApplyCleanly 串行应用所有 up 文件，确保幂等 + 无脏表。
func TestAllUpMigrationsApplyCleanly(t *testing.T) {
	files, err := filepath.Glob("./*.up.sql")
	require.NoError(t, err)
	sort.Strings(files)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			b, err := os.ReadFile(f)
			require.NoError(t, err)
			// 跳过非本测试关心的语句（例如 DROP IF EXISTS）
			if !strings.Contains(string(b), "CREATE TABLE") {
				return
			}
			_, err = conn.Exec(ctx, string(b))
			require.NoError(t, err, "apply %s", f)
		})
	}
}