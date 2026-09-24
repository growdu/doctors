// review-service 入口。
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

	"github.com/growdu/doctors/services/review/internal/handler"
	"github.com/growdu/doctors/services/review/internal/server"
	"github.com/growdu/doctors/services/review/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("review")
	if err != nil { log.Fatalf("load config: %v", err) }
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	svc := service.New(nilRepo{}, nilPublisher{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.L().Info("review-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil { logger.L().Error("review-service exited", zap.Error(err)); os.Exit(1) }
	logger.L().Info("review-service stopped")
}

func parseLevel(s string) zapcore.Level {
	switch s { case "debug": return zapcore.DebugLevel; case "warn": return zapcore.WarnLevel; case "error": return zapcore.ErrorLevel; default: return zapcore.InfoLevel }
}

type nilRepo struct{}
func (nilRepo) Create(ctx context.Context, r *service.Review) error { return errNil }
func (nilRepo) GetByOrderID(ctx context.Context, oid int64) (*service.Review, error) { return nil, errNil }
func (nilRepo) ListByEscort(ctx context.Context, eid int64, l, o int) ([]*service.Review, error) { return nil, errNil }

type nilPublisher struct{}
func (nilPublisher) PublishOrderReviewed(ctx context.Context, ev contracts.OrderReviewedEvent) error { return errNil }

var errNil = errors.New("review: repo/publisher not wired")
