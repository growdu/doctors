// payment-service 入口。
//
// 设计要点：
//   - pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径（与 admin 一致）。
//   - cfg.Kafka.Brokers 空 → 不发事件（v1 暂不开 publisher）。
//   - service.New 装配；handler.RegisterRoutes 挂载；server.Run 启动。
//   - -healthz flag：distroless 镜像的 Docker HEALTHCHECK 旁路——起独立 :9090 HTTP server
//     持续返回 200 OK，跳过全部业务装配。
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

	"github.com/growdu/doctors/services/payment/internal/handler"
	"github.com/growdu/doctors/services/payment/internal/router"
	"github.com/growdu/doctors/services/payment/internal/server"
	"github.com/growdu/doctors/services/payment/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/logger"
	"github.com/growdu/doctors/shared/metrics"
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

	cfg, err := config.Load("payment")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("payment-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = traceShutdown(context.Background()) }()

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// Prometheus 业务指标（§29 metrics plan）：service_info + DB pool 采集。
	metrics.InitMetrics("payment-service", cfg.ServiceVersion)

	// pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
	svc := service.New(nilRepo{}, nilPublisher{}, nilChannel{})
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("payment-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("payment-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("payment-service stopped")
}

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("payment-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}

// ---------- 占位实现：业务未接通；smoke 不调用 ----------

// nilRepo 是为了让 main 能编译 / 启动；接入 pgxpool 后会替换。
// 任何方法被调用都返回错误，便于 dev 期间快速发现。
type nilRepo struct{}

func (nilRepo) Create(ctx context.Context, p *service.Payment) error { return errNil }
func (nilRepo) GetByID(ctx context.Context, id int64) (*service.Payment, error) {
	return nil, errNil
}
func (nilRepo) GetByOrderID(ctx context.Context, orderID int64) (*service.Payment, error) {
	return nil, errNil
}
func (nilRepo) UpdateStatus(ctx context.Context, id int64, status string, completedAt, refundedAt *time.Time, externalTxID string) error {
	return errNil
}

type nilChannel struct{}

func (nilChannel) CreateOutTradeNo(ctx context.Context, p *service.Payment) (string, error) {
	return "", errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishPaymentCompleted(ctx context.Context, ev contracts.PaymentCompletedEvent) error {
	return nil
}
func (nilPublisher) PublishPaymentRefunded(ctx context.Context, ev contracts.PaymentRefundedEvent) error {
	return nil
}

var errNil = errors.New("payment: repo not wired (接 pgxpool 后替换)")

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