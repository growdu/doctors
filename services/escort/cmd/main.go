// escort-service 入口。
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/server"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("escort")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	svc := service.New(nilRepo{}, nilPublisher{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.L().Info("escort-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("escort-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("escort-service stopped")
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

// nilRepo / nilPublisher 是占位实现；接 DB / Kafka 后替换。
type nilRepo struct{}

func (nilRepo) Create(ctx context.Context, e *service.Escort) error { return errNil }
func (nilRepo) GetByID(ctx context.Context, id int64) (*service.Escort, error) {
	return nil, errNil
}
func (nilRepo) GetByUserID(ctx context.Context, uid int64) (*service.Escort, error) {
	return nil, errNil
}
func (nilRepo) UpdateStatus(ctx context.Context, id int64, s string) error { return errNil }
func (nilRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	return errNil
}
func (nilRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	return errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishAvailabilityChanged(ctx context.Context, ev service.AvailabilityEvent) error {
	return errNil
}

var errNil = errors.New("escort: repo/publisher not wired (接 PG/Kafka 后替换)")