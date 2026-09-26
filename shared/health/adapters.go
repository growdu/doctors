// Package health —— 依赖适配器（PG / Redis / Kafka）。
//
// 设计要点：
//   - 每个 Checker 都接受一个 "Name + 资源 + ping timeout"；
//   - 资源依赖通过参数注入（interface / brokers []string），不强制
//     调用方使用某个具体包路径——便于测试和未来替换。
//   - 不在 import 路径上耦合业务包：shared/health 不 import shared/db /
//     shared/redis / shared/kafka。Checker 通过 pgxpool.Pool / redis.Client /
//     拨号函数抽象，调用方在自己包里完成 NewPool/NewClient 后再注入。

package health

import (
	"context"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// pgPoolPinger 是 pgxpool 的最小接口（pool 本身已实现 Ping(ctx)）。
//
// 抽象成 interface 让 NewPGPoolChecker 既能接受 *pgxpool.Pool，
// 也能在测试中传入 fake（避免真实 DB）。
type pgPoolPinger interface {
	Ping(context.Context) error
}

// NewPGPoolChecker 构造 PostgreSQL 健康检查。
//
// name：稳定标识（如 "postgres-main"），同时用于 /readyz JSON + 监控 label。
// pool：通常是 shared/db.NewPool 的返回值；可为 nil（main 启动期 DSN 缺失时）。
// timeout：单次 Ping 的超时（与 K8s readinessProbe.timeoutSeconds 配套）。
func NewPGPoolChecker(name string, pool pgPoolPinger, timeout time.Duration) Checker {
	return CheckFunc{
		NameFn: func() string { return name },
		CheckFn: func(ctx context.Context) error {
			if pool == nil {
				return errResourceNil
			}
			c, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return pool.Ping(c)
		},
	}
}

// Compile-time guard：保证 shared/db.NewPool 返回的 *pgxpool.Pool 满足 pgPoolPinger。
var _ pgPoolPinger = (*pgxpool.Pool)(nil)

// NewRedisChecker 构造 Redis 健康检查。
//
// 抽象成 interface（仅要求 Ping(ctx) *redis.StatusCmd），让 fake 在测试里替换。
type redisClientPinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// NewRedisChecker 构造 Redis 健康检查。
//
// name / client / timeout 语义同 NewPGPoolChecker。
func NewRedisChecker(name string, client redisClientPinger, timeout time.Duration) Checker {
	return CheckFunc{
		NameFn: func() string { return name },
		CheckFn: func(ctx context.Context) error {
			if client == nil {
				return errResourceNil
			}
			c, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return client.Ping(c).Err()
		},
	}
}

// kafkaDialer 是 kafka-go broker TCP 连通的最小接口（net.Dial 即可）。
//
// 抽象成 interface：测试里可注入"必失败"的 dialer 模拟 broker 宕机。
type kafkaDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// defaultDialer 复用 net.Dialer；生产代码不需要关心（直接传 nil 即可）。
type defaultDialer struct{ d net.Dialer }

func (d *defaultDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.d.DialContext(ctx, network, address)
}

// NewKafkaBrokerChecker 构造 Kafka broker TCP 连通检查。
//
// brokers：bootstrap broker 列表（cfg.Kafka.Brokers）。
//   - 任一 broker 连通 → OK
//   - 全部失败 → 返回最后一个 error
// timeout：单次 dial 超时（建议 ≥ 500ms；K8s 内网通常 < 50ms，1s 留 20× 余量）。
func NewKafkaBrokerChecker(name string, brokers []string, timeout time.Duration) Checker {
	return newKafkaBrokerChecker(name, brokers, timeout, &defaultDialer{d: net.Dialer{}})
}

// newKafkaBrokerChecker 是 NewKafkaBrokerChecker 的"可注入 dialer"形态（测试用）。
func newKafkaBrokerChecker(name string, brokers []string, timeout time.Duration, dialer kafkaDialer) Checker {
	return CheckFunc{
		NameFn: func() string { return name },
		CheckFn: func(ctx context.Context) error {
			if len(brokers) == 0 {
				return errResourceNil
			}
			c, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			var lastErr error
			for _, b := range brokers {
				conn, err := dialer.DialContext(c, "tcp", b)
				if err == nil {
					_ = conn.Close()
					return nil
				}
				lastErr = err
			}
			if lastErr == nil {
				return errNoBrokers
			}
			return lastErr
		},
	}
}

// errResourceNil 标记资源未注入（main 在 pool/client == nil 时跳过注册 Checker；
// 但保留这个分支，让调用方可选地"强制注册一个总会失败的 checker"用于灰度）。
var errResourceNil = &resourceNilError{}

// errNoBrokers 标记 brokers 列表为空（极小概率，因为 NewKafkaBrokerChecker 已检查）。
var errNoBrokers = errResourceNil

type resourceNilError struct{}

func (e *resourceNilError) Error() string {
	return "health: resource not configured (nil or empty)"
}

// IsResourceNil 便于调用方判断"是 nil 资源触发的失败 vs 真实 ping 失败"。
func IsResourceNil(err error) bool {
	return errorsIs(err, errResourceNil)
}

// errorsIs 抽出来避免循环 import（errors 标准库函数）。
func errorsIs(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}