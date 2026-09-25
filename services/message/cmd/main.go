// message-service 入口。
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

	"github.com/growdu/doctors/services/message/internal/handler"
	"github.com/growdu/doctors/services/message/internal/router"
	"github.com/growdu/doctors/services/message/internal/server"
	"github.com/growdu/doctors/services/message/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("message")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
	svc := service.New(nilRepo{}, nilPublisher{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("message-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("message-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("message-service stopped")
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

// ---------- nilRepo / nilPublisher（main 装配占位；真实 pgx 接入留 v2） ----------

type nilRepo struct{}

func (nilRepo) CreateConversation(ctx context.Context, c *service.Conversation) error {
	return errNil
}
func (nilRepo) CreateMessage(ctx context.Context, m *service.Message) error { return errNil }
func (nilRepo) GetMessageByID(ctx context.Context, id int64) (*service.Message, error) {
	return nil, errNil
}
func (nilRepo) ListMessagesByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*service.Message, error) {
	return nil, errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishMessageSent(ctx context.Context, ev contracts.MessageSentEvent) error {
	return errNil
}

// _ = time 防止未使用告警（main 暂时不直接用 time）
var _ = time.Second

var errNil = errors.New("message: repo/publisher not wired")
