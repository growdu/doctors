// Package db 提供 PostgreSQL 连接池与事务辅助。
//
// 设计要点：
//   - 仅暴露 *pgxpool.Pool，业务自行管理事务。
//   - DSN 必须以 postgres:// 或 postgresql:// 开头，避免误连到 MySQL。
//   - WithTx 用泛型 fn 包装事务，自动处理 commit/rollback。
//   - 集成测试用 //go:build integration 隔离，需 docker-compose up 后运行。
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config 是连接池配置。
type Config struct {
	DSN      string
	MaxConns int32
	MinConns int32
}

// ApplyDefaults 填默认值。
func (c *Config) ApplyDefaults() {
	if c.MaxConns <= 0 {
		c.MaxConns = 10
	}
	if c.MinConns <= 0 {
		c.MinConns = 2
	}
}

// ValidateDSN 校验 DSN 格式。
func ValidateDSN(dsn string) error {
	if dsn == "" {
		return errors.New("db: empty dsn")
	}
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		return fmt.Errorf("db: dsn must start with postgres:// or postgresql://, got: %q", prefixOf(dsn))
	}
	return nil
}

func prefixOf(s string) string {
	if len(s) > 32 {
		return s[:32] + "..."
	}
	return s
}

// NewPool 建立 pgx 连接池。
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if err := ValidateDSN(cfg.DSN); err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()

	pcfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	pcfg.MaxConns = cfg.MaxConns
	pcfg.MinConns = cfg.MinConns
	pcfg.HealthCheckPeriod = 30 * time.Second
	pcfg.MaxConnLifetime = time.Hour
	pcfg.MaxConnIdleTime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("db: new pool: %w", err)
	}
	return pool, nil
}

// HealthCheck 主动 ping 数据库。
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return pool.Ping(c)
}

// WithTx 在事务中执行 fn；fn 返回 error 时回滚，否则提交。
// 任意 panic 也会触发回滚。
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		if cErr := tx.Commit(ctx); cErr != nil {
			err = fmt.Errorf("db: commit: %w", cErr)
		}
	}()
	return fn(tx)
}