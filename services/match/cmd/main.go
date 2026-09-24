// match-service 入口。
//
// 阶段 4 已实现：候选打分 + Redis 抢单池（NopPool 默认）。
// 集成测试时把 NopPool 换为 RedisPool；生产 main 注入 RedisPool。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/match/internal/handler"
	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/scorer"
	"github.com/growdu/doctors/services/match/internal/server"
	"github.com/growdu/doctors/services/match/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("match")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// v1 用 NopPool；接 Redis 后换 RedisPool
	svc := service.New(pool.NewNopPool(), nilEscortLoader{}, 0)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("match-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("match-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("match-service stopped")
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

// nilEscortLoader 是占位实现；任何调用返回 nil。
type nilEscortLoader struct{}

func (nilEscortLoader) ListAvailable(ctx context.Context, city string) ([]scorer.Escort, error) {
	return nil, nil
}