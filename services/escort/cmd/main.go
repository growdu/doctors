// escort-service 入口。
//
// 装配 4 套仓储：profile / qualification / training / availability。
// DSN 缺失 → 各模块 nil 占位 + warn log（路由仍生效，业务 endpoint 调用时报错）。
//
// §32 Kafka 接入：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；
//
//	非空 → kafkapublisher.Publisher 真实 Kafka，订阅 TopicEscortAvailable/Unavailable。
//
// 优雅停机：srv.RegisterShutdownHook 注册 otel-tracer + db-pool + kafka-publisher。
//
// 注意：users 表由 auth-service 持有；escort 通过 auth / gRPC 获取 user_id 信息。
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

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/kafkapublisher"
	"github.com/growdu/doctors/services/escort/internal/router"
	"github.com/growdu/doctors/services/escort/internal/server"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/config"
	shareddb "github.com/growdu/doctors/shared/db"
	"github.com/growdu/doctors/shared/health"
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

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// 构造 pgxpool；DSN 缺失时降级为 nil。
	pool, err := buildPool(cfg)
	if err != nil {
		log.Fatalf("build pool: %v", err)
	}
	if pool != nil {
		defer pool.Close()
	}

	// Prometheus 业务指标（§29 metrics plan）：service_info + DB pool 采集。
	metrics.InitMetrics("escort-service", cfg.ServiceVersion,
		metrics.WithDBStatProvider(func() []metrics.DBPoolStat {
			if pool == nil {
				return nil
			}
			s := pool.Stat()
			return []metrics.DBPoolStat{{
				Name:       "main",
				Acquired:   s.AcquiredConns(),
				Idle:       s.IdleConns(),
				TotalConns: s.TotalConns(),
			}}
		}),
	)

	// 4 套仓储：pool != nil → 各 New(pool)；否则 nil 占位。
	var (
		profileRepo       service.EscortRepo
		qualificationRepo service.QualificationRepo
		trainingRepo      service.TrainingRepo
	)
	if pool != nil {
		profileRepo = service.NewProfileRepo(pool)
		qualificationRepo = service.NewQualificationRepo(pool)
		trainingRepo = service.NewTrainingRepo(pool)
		logger.L().Info("escort-service: 3 in-memory repos wired (profile/qual/training)")
	} else {
		profileRepo = nilProfileRepo{}
		qualificationRepo = nilQualRepo{}
		trainingRepo = nilTrainingRepo{}
		logger.L().Warn("escort-service: cfg.db.dsn empty; 3 repos not wired (fallback to nil)")
	}

	// Publisher：cfg.Kafka.Brokers 空 → nilPublisher；非空 → 真实 Kafka publisher。
	publisher, kafkaCloser := buildPublisher(cfg)
	svc := service.New(profileRepo, publisher).
		WithQualificationRepo(qualificationRepo).
		WithTrainingRepo(trainingRepo)

	// availability 仓储：pool != nil → availability.New(pool)；否则 nil 占位。
	var availRepo availability.AvailabilityRepo
	if pool != nil {
		availRepo = availability.New(pool)
		logger.L().Info("escort-service: availability repo wired (pgx)")
	} else {
		availRepo = nilAvailabilityRepo{}
		logger.L().Warn("escort-service: cfg.db.dsn empty; availability repo not wired (fallback to nil)")
	}

	availSvc := availability.NewService(availRepo)
	availH := availability.NewHandler(availSvc)

	h := handler.New(svc)

	// 依赖健康检查（/readyz）。escort 有 DB + Kafka publisher。
	healthM := health.NewManager(health.WithTimeout(1 * time.Second))
	if pool != nil {
		healthM.MustRegister(health.NewPGPoolChecker("postgres-main", pool, time.Second))
		logger.L().Info("escort-service: readyz registered checker: postgres-main")
	} else {
		logger.L().Warn("escort-service: cfg.db.dsn empty; /readyz will fail-closed (postgres not configured)")
	}
	if len(cfg.Kafka.Brokers) > 0 {
		healthM.MustRegister(health.NewKafkaBrokerChecker("kafka-brokers", cfg.Kafka.Brokers, 1*time.Second))
		logger.L().Info("escort-service: readyz registered checker: kafka-brokers",
			zap.Strings("brokers", cfg.Kafka.Brokers))
	}

	srv := server.New(cfg.HTTP.Addr, router.NewWithPublic(h, availH, cfg.Auth.JWTSecret, healthM))

	// 优雅停机：先关 OTel tracer，再关 DB pool，再关 Kafka publisher（LIFO）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if pool != nil {
		srv.RegisterShutdownHook("db-pool", func() error { pool.Close(); return nil })
	}
	if kafkaCloser != nil {
		srv.RegisterShutdownHook("kafka-publisher", kafkaCloser)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("escort-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("escort-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("escort-service stopped")
}

// buildPool 根据 cfg.DB 构造 pgxpool；DSN 空时返回 (nil, nil)。
//
// §34 OTel auto-instrumentation：注入 otelpgx tracer。
func buildPool(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DB.DSN == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	poolCfg := shareddb.Config{
		DSN:      cfg.DB.DSN,
		MaxConns: cfg.DB.MaxConns,
		MinConns: cfg.DB.MinConns,
	}
	pcfg, err := pgxpool.ParseConfig(poolCfg.DSN)
	if err == nil {
		tracing.WithPgxPool(pcfg)
		poolCfg.Tracer = pcfg.ConnConfig.Tracer
	}
	return shareddb.NewPool(ctx, poolCfg)
}

// ---------- nil 占位（业务接口 fallback；调用即返回 errNil） ----------

type nilProfileRepo struct{}

func (nilProfileRepo) Create(ctx context.Context, e *service.Escort) error { return errNil }
func (nilProfileRepo) GetByID(ctx context.Context, id int64) (*service.Escort, error) {
	return nil, errNil
}
func (nilProfileRepo) GetByUserID(ctx context.Context, uid int64) (*service.Escort, error) {
	return nil, errNil
}
func (nilProfileRepo) UpdateStatus(ctx context.Context, id int64, s string) error { return errNil }
func (nilProfileRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	return errNil
}
func (nilProfileRepo) UpdateCity(ctx context.Context, id int64, city string) error { return errNil }
func (nilProfileRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	return errNil
}

type nilPublisher struct{}

func (nilPublisher) PublishAvailabilityChanged(ctx context.Context, ev service.AvailabilityEvent) error {
	return nil
}

// buildPublisher 根据 cfg.Kafka 构造 publisher 与可选 closer。
//
// §32 公共模式：Brokers 空 → *nilPublisher + nil closer；非空 → *kafkapublisher.Publisher + 同对象 closer。
func buildPublisher(cfg *config.Config) (service.Publisher, func() error) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Warn("escort-service: kafka.brokers empty; publisher disabled (fallback to nilPublisher)")
		return nilPublisher{}, nil
	}
	kp, err := kafkapublisher.New(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatalf("escort-service: build kafka publisher: %v", err)
	}
	logger.L().Info("escort-service: kafka publisher wired",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.Strings("topics", []string{"escort.available", "escort.unavailable"}))
	return kp, kp.Close
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

// nilAvailabilityRepo 占位：DSN 缺失时 availability 业务 endpoint 调用即返回 errNil。
//
//	真实实现见 escort/internal/availability/repo.go（pgx 直写 escort_availabilities）。
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
func (nilAvailabilityRepo) BookByOrder(ctx context.Context, id, orderID int64) error {
	return errNil
}
func (nilAvailabilityRepo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	return errNil
}

var errNil = errors.New("escort: repo not wired (cfg.db.dsn empty; configure postgres dsn to enable)")

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