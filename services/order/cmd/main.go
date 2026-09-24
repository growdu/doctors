// order-service 入口。
//
// 设计要点：
//   - 阶段 3.6 已实现 Accept 抢单；main 骨架保留 nilRepo / unwiredUsers 占位，
//     接入 PG / UserLookup 后替换。
//   - §4.2 order-lock plan（2026-09-24）：可选装配 RedisLocker / KafkaPublisher /
//     ExpiredLockScanner。配置存在就启用，否则降级为 NopLocker / NopPublisher / 不跑 scheduler。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/scheduler"
	"github.com/growdu/doctors/services/order/internal/server"
	"github.com/growdu/doctors/services/order/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/lock"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("order")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// pool=nil：路由生效，业务调用会 panic；smoke 不走业务路径。
	orderRepo := repo.NewOrderRepo(nil)

	// §4.2 可选装配：Redis 锁单 + Kafka 事件 + Scheduler
	locker, lockerCloser := buildLocker(cfg)
	defer func() {
		if closer, ok := lockerCloser.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()
	publisher, pubCloser := buildPublisher(cfg)
	defer func() {
		if pubCloser != nil {
			_ = pubCloser.Close()
		}
	}()

	svc := service.New(orderRepo, unwiredUsers{}).
		WithLocker(locker).
		WithPublisher(publisher)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// §4.2 启动 ExpiredLockScanner（5s 扫一次过期锁单；orderRepo=nil 时不启动）
	if orderRepo != nil && svc != nil {
		go scheduler.NewExpiredLockScanner(orderRepo, svc, publisher, 5*time.Second).Run(ctx)
	}

	logger.L().Info("order-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("order-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("order-service stopped")
}

// buildLocker 根据 cfg.Redis 配置构造 Locker；配置缺失 → NopLocker。
func buildLocker(cfg *config.Config) (lock.Locker, any) {
	if cfg.Redis.Addr == "" {
		logger.L().Info("locker disabled (redis.addr empty); falling back to NopLocker")
		return lock.NopLocker{}, nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
		DB:   cfg.Redis.DB,
	})
	logger.L().Info("locker enabled (redis SETNX)", zap.String("addr", cfg.Redis.Addr))
	return lock.NewRedisLocker(rdb), rdb
}

// buildPublisher 根据 cfg.Kafka 配置构造 Publisher；配置缺失 → NopPublisher。
func buildPublisher(cfg *config.Config) (events.Publisher, events.Publisher) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Info("publisher disabled (kafka.brokers empty); falling back to NopPublisher")
		return &events.NopPublisher{}, nil
	}
	logger.L().Info("publisher enabled (kafka)", zap.Strings("brokers", cfg.Kafka.Brokers))
	p := events.NewKafkaPublisher(cfg.Kafka.Brokers)
	return p, p
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// unwiredUsers 是 UserLookup 占位实现；任何调用返回 nil。
// 接 DB 后会被 auth.UserRepo 替换（本地直连或 gRPC）。
type unwiredUsers struct{}

func (unwiredUsers) FindByID(ctx context.Context, id int64) (*service.UserSnapshot, error) {
	return nil, nil
}