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
//
// 字段语义：
//   - DSN：必填；缺/非 postgres:// 前缀 → NewPool 返回 error（调用方按 DSN 缺失降级为 nil）。
//   - MaxConns / MinConns：连接池上下限；ApplyDefaults 兜底 10 / 2。
//   - ConnectTimeout：建立单条连接的拨号超时；默认 5s（pgxpool 自身隐式是
//     ctx 控制的，建议调用方传 ctx 控制 NewPool 整体超时；此处为单条 conn
//     的 TCP 拨号 / auth 阶段）。
//   - HealthCheckPeriod：默认 30s；设为 0 时关闭主动 ping。
//   - MaxConnLifetime / MaxConnIdleTime：默认 1h / 10m。
type Config struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	ConnectTimeout    time.Duration
	HealthCheckPeriod time.Duration
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
}

// ApplyDefaults 填默认值。
func (c *Config) ApplyDefaults() {
	if c.MaxConns <= 0 {
		c.MaxConns = 10
	}
	if c.MinConns <= 0 {
		c.MinConns = 2
	}
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	if c.HealthCheckPeriod <= 0 {
		c.HealthCheckPeriod = 30 * time.Second
	}
	if c.MaxConnLifetime <= 0 {
		c.MaxConnLifetime = time.Hour
	}
	if c.MaxConnIdleTime <= 0 {
		c.MaxConnIdleTime = 10 * time.Minute
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
	pcfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	pcfg.MaxConnLifetime = cfg.MaxConnLifetime
	pcfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	pcfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

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