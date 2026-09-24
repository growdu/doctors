// auth-service 入口。
//
// 当前阶段只装配配置 + server，未接入 DB / Redis / Kafka。
// 后续阶段会在 initDep() 中加入 pool、redis client、kafka producer、各业务 service。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/auth/internal/server"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("auth")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	srv := server.New(cfg.HTTP.Addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("auth-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("auth-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("auth-service stopped")
}

// parseLevel 把 yaml 字符串映射为 zapcore.Level，未知值默认 info。
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