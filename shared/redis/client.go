// Package redis 提供 Redis 客户端封装。
//
// 设计要点：
//   - 暴露 *redis.Client，业务可直接使用 go-redis v9 API。
//   - 地址必须是 host:port 形式，不带 scheme。
//   - 集成测试用 //go:build integration 隔离。
package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 是 Redis 连接配置。
type Config struct {
	Addr     string
	Password string
	DB       int
}

// ApplyDefaults 填默认 DB。
func (c *Config) ApplyDefaults() {}

// ValidateAddr 校验 addr 是否合法。
func ValidateAddr(addr string) error {
	if addr == "" {
		return errors.New("redis: empty addr")
	}
	if strings.Contains(addr, "://") {
		return fmt.Errorf("redis: addr must not contain scheme, got %q", addr)
	}
	parts := strings.Split(addr, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("redis: addr must be host:port, got %q", addr)
	}
	return nil
}

// NewClient 构造 redis 客户端（不立即连接，懒加载）。
func NewClient(cfg Config) (*redis.Client, error) {
	if err := ValidateAddr(cfg.Addr); err != nil {
		return nil, err
	}
	c := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     20,
	})
	return c, nil
}

// HealthCheck 主动 ping。
func HealthCheck(ctx context.Context, c *redis.Client) error {
	cc, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Ping(cc).Err()
}