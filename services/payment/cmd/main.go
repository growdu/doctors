// payment-service 入口。
//
// 设计要点：
//   - 本期（§31）：构造 pgxpool（cfg.db.dsn 缺失 → 降级 nil + warn log）。
//   - refund.NewRepo(pool) 接入 refunds + refund_policies（v1 in-memory 实现；
//     v2 接 PG 时只换内部存储，签名不变）。
//   - payment 表 repo 暂保留 nilRepo 占位（v1 mock 渠道 + 内存记账即可）。
//   - §32 Kafka 接入：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；
//     非空 → kafkapublisher.Publisher 真实 Kafka，订阅 TopicPaymentCompleted/Refunded。
//   - 优雅停机：srv.RegisterShutdownHook 注册 otel-tracer + db-pool + kafka-publisher。
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

	"github.com/growdu/doctors/services/payment/internal/handler"
	"github.com/growdu/doctors/services/payment/internal/kafkapublisher"
	"github.com/growdu/doctors/services/payment/internal/refund"
	"github.com/growdu/doctors/services/payment/internal/router"
	"github.com/growdu/doctors/services/payment/internal/server"
	"github.com/growdu/doctors/services/payment/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
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

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// 构造 pgxpool；DSN 缺失时降级为 nil（路由仍生效，业务 endpoint 调用时报错）。
	pool, err := buildPool(cfg)
	if err != nil {
		log.Fatalf("build pool: %v", err)
	}
	if pool != nil {
		defer pool.Close()
	}

	// Prometheus 业务指标（§29 metrics plan）：service_info + DB pool 采集。
	// pool != nil 时注入 StatProvider；nil 时跳过。
	metrics.InitMetrics("payment-service", cfg.ServiceVersion,
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

	// refund 仓储：pool != nil 走 refund.NewRepo(pool)（v1 内存版，签名预留 PG 接入）；
	// 否则 nil 占位（refund 业务 endpoint 调用时报错）。
	var refundRepo refund.RefundRepo
	if pool != nil {
		refundRepo = refund.NewRepo(pool)
		logger.L().Info("payment-service: refund repo wired (in-memory v1)")
	} else {
		logger.L().Warn("payment-service: cfg.db.dsn empty; refund repo not wired (fallback to nil)")
		refundRepo = nilRefundRepo{}
	}

	// payment 仓储：v1 仍用 nilRepo 占位（mock 渠道）。v2 接 PG 时替换。
	// Kafka publisher：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；非空 → 真实 publisher。
	publisher, kafkaCloser := buildPublisher(cfg)
	svc := service.New(nilRepo{}, publisher, nilChannel{})
	h := handler.New(svc)

	// 依赖健康检查（/readyz）。pool/kafka brokers 各自可选——nil/空 → skip + warn。
	healthM := health.NewManager(health.WithTimeout(1 * time.Second))
	if pool != nil {
		healthM.MustRegister(health.NewPGPoolChecker("postgres-main", pool, time.Second))
		logger.L().Info("payment-service: readyz registered checker: postgres-main")
	} else {
		logger.L().Warn("payment-service: cfg.db.dsn empty; /readyz will fail-closed (postgres not configured)")
	}
	if len(cfg.Kafka.Brokers) > 0 {
		healthM.MustRegister(health.NewKafkaBrokerChecker("kafka-brokers", cfg.Kafka.Brokers, 1*time.Second))
		logger.L().Info("payment-service: readyz registered checker: kafka-brokers",
			zap.Strings("brokers", cfg.Kafka.Brokers))
	}

	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret, healthM))

	// 优雅停机：先关 OTel tracer，再关 DB pool，再关 Kafka publisher（LIFO）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if pool != nil {
		srv.RegisterShutdownHook("db-pool", func() error { pool.Close(); return nil })
	}
	if kafkaCloser != nil {
		srv.RegisterShutdownHook("kafka-publisher", kafkaCloser)
	}
	// 预留 shutdown hook 标记，让运维一眼看出 refund 仓储依赖 DB。
	if refundRepo != nil {
		srv.RegisterShutdownHook("refund-repo", func() error { return nil })
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("payment-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("payment-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("payment-service stopped")
}

// buildPool 根据 cfg.DB 构造 pgxpool；DSN 空时返回 (nil, nil) —— 调用方按"降级"
//
//	模式装配 nilRefundRepo，路由仍能注册（业务 endpoint 调用时才报错）。
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

// nilRefundRepo 占位：DSN 缺失时 refund 业务 endpoint 调用即返回 error。
//
//	有 pool 时使用 refund.NewRepo(pool)（in-memory v1）。
type nilRefundRepo struct{}

func (nilRefundRepo) Create(ctx context.Context, r *refund.Record) error { return errNilRefund }
func (nilRefundRepo) GetByOrderID(ctx context.Context, orderID int64) ([]*refund.Record, error) {
	return nil, errNilRefund
}
func (nilRefundRepo) UpdateStatus(ctx context.Context, id int64, status string, externalTxID string, failureReason string) error {
	return errNilRefund
}

// nilPublisher 占位：内存记账，不发外部事件。
type nilPublisher struct{}

func (nilPublisher) PublishPaymentCompleted(ctx context.Context, ev contracts.PaymentCompletedEvent) error {
	return nil
}
func (nilPublisher) PublishPaymentRefunded(ctx context.Context, ev contracts.PaymentRefundedEvent) error {
	return nil
}

// buildPublisher 根据 cfg.Kafka 构造 publisher 与可选 closer。
//
// §32 公共模式：Brokers 空 → *nilPublisher + nil closer；非空 → *kafkapublisher.Publisher + 同对象 closer。
func buildPublisher(cfg *config.Config) (service.Publisher, func() error) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Warn("payment-service: kafka.brokers empty; publisher disabled (fallback to nilPublisher)")
		return nilPublisher{}, nil
	}
	kp, err := kafkapublisher.New(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatalf("payment-service: build kafka publisher: %v", err)
	}
	logger.L().Info("payment-service: kafka publisher wired",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.Strings("topics", []string{contracts.TopicPaymentCompleted, contracts.TopicPaymentRefunded}))
	return kp, kp.Close
}

var (
	errNil         = errors.New("payment: repo not wired (接 pgxpool 后替换)")
	errNilRefund   = errors.New("payment: refund repo not wired (cfg.db.dsn empty)")
)

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