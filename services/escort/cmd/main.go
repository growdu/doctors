// escort-service 入口。
//
// 设计要点：
//   - pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
//   - availability 子包独立装配（availability.NewService(nilRepo)）。
//   - service.New 装配 + WithQualificationRepo / WithTrainingRepo 注入；handler + availability handler 挂载；server.Run 启动。
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

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/router"
	"github.com/growdu/doctors/services/escort/internal/server"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/config"
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

	cfg, err := config.Load("escort")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("escort-service", cfg.Tracing.OTLPEndpoint,
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
	svc := service.New(nilRepo{}, nilPublisher{}).
		WithQualificationRepo(nilQualRepo{}).
		WithTrainingRepo(nilTrainingRepo{})

	// availability 子包
	availSvc := availability.NewService(nilAvailabilityRepo{})
	availH := availability.NewHandler(availSvc)

	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.NewWithPublic(h, availH, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("escort-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("escort-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("escort-service stopped")
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

// ---------- nilRepo / nilPublisher / nilQualRepo / nilTrainingRepo / nilAvailabilityRepo ----------

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
func (nilRepo) UpdateCity(ctx context.Context, id int64, city string) error { return errNil }
func (nilRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	return errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishAvailabilityChanged(ctx context.Context, ev service.AvailabilityEvent) error {
	return errNil
}

type nilQualRepo struct{}

func (nilQualRepo) Create(ctx context.Context, q *service.Qualification) error { return errNil }
func (nilQualRepo) GetByID(ctx context.Context, id, escortID int64) (*service.Qualification, error) {
	return nil, errNil
}
func (nilQualRepo) ListByEscort(ctx context.Context, escortID int64) ([]*service.Qualification, error) {
	return nil, errNil
}
func (nilQualRepo) Update(ctx context.Context, q *service.Qualification) error { return errNil }
func (nilQualRepo) Delete(ctx context.Context, id, escortID int64) error       { return errNil }

type nilTrainingRepo struct{}

func (nilTrainingRepo) Create(ctx context.Context, t *service.Training) error { return errNil }
func (nilTrainingRepo) ListByEscort(ctx context.Context, escortID int64) ([]*service.Training, error) {
	return nil, errNil
}

// nilAvailabilityRepo 占位：真实实现见 escort/internal/availability/repo.go。
// 这里只暴露构造签名所需的最少方法以满足装配；nil 时调用即 panic（与 admin 一致）。
type nilAvailabilityRepo struct{}

func (nilAvailabilityRepo) Create(ctx context.Context, escortID int64, startAt, endAt time.Time) (*availability.Availability, error) {
	return nil, errNil
}
func (nilAvailabilityRepo) FindByID(ctx context.Context, id int64) (*availability.Availability, error) {
	return nil, errNil
}
func (nilAvailabilityRepo) Delete(ctx context.Context, id, escortID int64) error { return errNil }
func (nilAvailabilityRepo) ListByEscort(ctx context.Context, escortID int64) ([]*availability.Availability, error) {
	return nil, errNil
}
func (nilAvailabilityRepo) ListAvailableByTime(ctx context.Context, startAt, endAt time.Time, limit int) ([]*availability.Availability, error) {
	return nil, errNil
}
func (nilAvailabilityRepo) BookByOrder(ctx context.Context, id, orderID int64) error { return errNil }
func (nilAvailabilityRepo) ReleaseByOrder(ctx context.Context, orderID int64) error   { return errNil }

var errNil = errors.New("escort: repo/publisher not wired (接 PG/Kafka 后替换)")

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("escort-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}
