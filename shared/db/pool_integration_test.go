//go:build integration
// +build integration

package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/db"
)

func dsn() string {
	if v := os.Getenv("DOCTORS_DB_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@localhost:5432/doctors?sslmode=disable"
}

// TestIntegration_NewPoolAndPing 真实起 pgxpool 并 ping。
// 运行：go test -tags=integration ./shared/db/...
func TestIntegration_NewPoolAndPing(t *testing.T) {
	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.Config{DSN: dsn(), MaxConns: 5})
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, db.HealthCheck(ctx, pool))
}

// TestIntegration_WithTxCommitRollback 真实事务提交流转。
func TestIntegration_WithTxCommitRollback(t *testing.T) {
	ctx := context.Background()
	pool, err := db.NewPool(ctx, db.Config{DSN: dsn(), MaxConns: 2})
	require.NoError(t, err)
	defer pool.Close()

	_, _ = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS t_tx_test (id INT PRIMARY KEY, note TEXT)`)
	_, _ = pool.Exec(ctx, `TRUNCATE t_tx_test`)

	// commit
	require.NoError(t, db.WithTx(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO t_tx_test(id, note) VALUES (1, 'commit')`)
		return err
	}))

	// rollback
	require.NoError(t, db.WithTx(ctx, pool, func(tx pgx.Tx) error {
		_, _ = tx.Exec(ctx, `INSERT INTO t_tx_test(id, note) VALUES (2, 'rollback')`)
		return pgx.ErrTxClosed // 触发回滚
	}))

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM t_tx_test`).Scan(&n))
	require.Equal(t, 1, n, "应当只有 1 条（rollback 那条未提交）")
}