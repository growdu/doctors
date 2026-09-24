// user-service 入口。
//
// 阶段：仅装配 server + 假 repo；接 DB 后替换 nilRepo。
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/server"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("user")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	svc := service.New(nilRepo{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.L().Info("user-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("user-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("user-service stopped")
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

// nilRepo 占位实现：任何调用都返回错误，便于 dev 期间发现。
type nilRepo struct{}

func (nilRepo) GetByID(ctx context.Context, id int64) (*service.Profile, error) {
	return nil, errNil
}
func (nilRepo) UpdateNickname(ctx context.Context, id int64, n string) error { return errNil }
func (nilRepo) UpdateAvatar(ctx context.Context, id int64, u string) error  { return errNil }

var errNil = errors.New("user: repo not wired (接 pgxpool 后替换)")