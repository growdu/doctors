// order-service 入口。
//
// 阶段 3.6 已实现 Accept 抢单，但 main 还未注入真实 PG / TxRunner。
// 启动时使用 nil pool 占位；接 DB 后（阶段 3.9）替换。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/server"
	"github.com/growdu/doctors/services/order/internal/service"
	"github.com/growdu/doctors/shared/config"
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
	svc := service.New(orderRepo, unwiredUsers{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("order-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("order-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("order-service stopped")
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