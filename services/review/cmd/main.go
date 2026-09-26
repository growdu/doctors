// review-service 入口。
//
// 设计要点：
//   - pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
//   - Kafka 配置缺失 → NopPublisher（dev / 单测友好）。
//   - service.New 装配；handler.RegisterRoutes 挂载；server.Run 启动。
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/review/internal/handler"
	"github.com/growdu/doctors/services/review/internal/router"
	"github.com/growdu/doctors/services/review/internal/server"
	"github.com/growdu/doctors/services/review/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/logger"
	"github.com/growdu/doctors/shared/tracing"
)

func main() {
	// distroless HEALTHCHECK 旁路：在 :9090 起独立 http server，仅暴露 /healthz。
	healthzOnly := flag.Bool("healthz", false, "run healthz-only HTTP server on :9090 and exit")
	flag.Parse()
	if *healthzOnly {
		runHealthzServer()
		return
	}

	cfg, err := config.Load("review")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("review-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = traceShutdown(context.Background()) }()

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
	svc := service.New(nilRepo{}, nilPublisher{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("review-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("review-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("review-service stopped")
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

func (nilRepo) Create(ctx context.Context, r *service.Review) error { return errNil }
func (nilRepo) GetByID(ctx context.Context, id int64) (*service.Review, error) {
	return nil, errNil
}
func (nilRepo) GetByOrderID(ctx context.Context, oid int64) (*service.Review, error) {
	return nil, errNil
}
func (nilRepo) ListByEscort(ctx context.Context, eid int64, l, o int) ([]*service.Review, error) {
	return nil, errNil
}
func (nilRepo) List(ctx context.Context, f service.ListFilter) ([]*service.Review, error) {
	return nil, errNil
}
func (nilRepo) UpdateReply(ctx context.Context, id int64, reply string, adminID int64) error {
	return errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishOrderReviewed(ctx context.Context, ev contracts.OrderReviewedEvent) error {
	return errNil
}

// _ = time 防止未使用告警（main 暂时不直接用 time）
var _ = time.Second

var errNil = errors.New("review: repo/publisher not wired")

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("review-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}
