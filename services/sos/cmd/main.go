// sos-service 入口。
//
// 设计要点：
//   - pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
//   - Kafka 配置缺失 → NopPublisher（dev / 单测友好）。
//   - service.New 装配；handler.RegisterRoutes 挂载；server.Run 启动。
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

	"github.com/growdu/doctors/services/sos/internal/handler"
	"github.com/growdu/doctors/services/sos/internal/router"
	"github.com/growdu/doctors/services/sos/internal/server"
	"github.com/growdu/doctors/services/sos/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("sos")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
	svc := service.New(nilRepo{}, nilLookup{}, nilPublisher{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("sos-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("sos-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("sos-service stopped")
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

// ---------- nilRepo / nilLookup / nilPublisher（main 装配占位；真实 pgx 接入留 v2） ----------

type nilRepo struct{}

func (nilRepo) Create(ctx context.Context, s *service.SOS) error     { return errNil }
func (nilRepo) GetByID(ctx context.Context, id int64) (*service.SOS, error) {
	return nil, errNil
}
func (nilRepo) UpdateStatus(ctx context.Context, id int64, status string) error { return errNil }
func (nilRepo) IsRecentDuplicate(ctx context.Context, orderID int64, since time.Time) (bool, error) {
	return false, errNil
}
func (nilRepo) List(ctx context.Context, f service.ListFilter) ([]*service.SOS, error) {
	return nil, errNil
}

type nilLookup struct{}

func (nilLookup) IsOrderActive(ctx context.Context, orderID int64) (bool, error) {
	return false, errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishSOSRaised(ctx context.Context, ev contracts.SOSRaisedEvent) error {
	return errNil
}

var errNil = errors.New("sos: repo/publisher not wired")
