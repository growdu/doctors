// admin-service 入口。
//
// 设计要点：
//   - pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
//   - 内部 clients（order/refund/escort/user）从 cfg 读 baseURL；smoke 用占位。
//   - §32 Kafka 接入：cfg.Kafka.Brokers 空 → events.NopPublisher（dev / 单测友好）；
//     非空 → events.KafkaPublisher 真实 Kafka，发布多 topic（force_cancel/approve/reject 等）。
//   - service.New 装配；handler.RegisterRoutes 挂载；server.Run 启动。
//   - §31 接入：repo.NewWorkOrderRepo(pool) + repo.NewReportsRepo(pool) 接 admin_work_orders。
//   - 优雅停机：srv.RegisterShutdownHook 注册 otel-tracer + db-pool + kafka-publisher（LIFO）。
package main

import (
	"context"
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

	"github.com/growdu/doctors/services/admin/internal/clients"
	"github.com/growdu/doctors/services/admin/internal/events"
	"github.com/growdu/doctors/services/admin/internal/handler"
	"github.com/growdu/doctors/services/admin/internal/repo"
	"github.com/growdu/doctors/services/admin/internal/router"
	"github.com/growdu/doctors/services/admin/internal/server"
	"github.com/growdu/doctors/services/admin/internal/service"
	"github.com/growdu/doctors/shared/config"
	shareddb "github.com/growdu/doctors/shared/db"
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

	cfg, err := config.Load("admin")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("admin-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// 构造 pgxpool；DSN 缺失时降级为 nil（admin 仍有跨服务直读 client，可部分运行）。
	pool, err := buildPool(cfg)
	if err != nil {
		log.Fatalf("build pool: %v", err)
	}

	// Prometheus 业务指标（§29 metrics plan）：service_info + DB pool 采集。
	metrics.InitMetrics("admin-service", cfg.ServiceVersion,
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

	// work_orders / reports 仓储：pool != nil 走真实 PG；否则 nil 占位。
	var woRepo *repo.WorkOrderRepo
	var rrRepo *repo.ReportsRepo
	if pool != nil {
		woRepo = repo.NewWorkOrderRepo(pool)
		rrRepo = repo.NewReportsRepo(pool)
		logger.L().Info("admin-service: 2 repos wired (work_orders + reports)")
	} else {
		logger.L().Warn("admin-service: cfg.db.dsn empty; work_orders/reports repos not wired")
	}

	// 内部 clients（baseURL 从配置读；smoke 用占位）
	orderClient := clients.NewOrderClient(cfg.Admin.OrderBaseURL, 5*time.Second)
	refundClient := clients.NewRefundClient(cfg.Admin.RefundBaseURL, 5*time.Second)
	escortClient := clients.NewEscortClient(cfg.Admin.EscortBaseURL, 5*time.Second)
	userClient := clients.NewUserClient(cfg.Admin.UserBaseURL, 5*time.Second)

	// Publisher（Kafka 或 Nop）
	pub, kafkaPub := buildPublisher(cfg)

	svc := service.New(rrRepo, woRepo, orderClient, refundClient, escortClient, userClient, pub)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	// 优雅停机：先关 OTel tracer，再关 DB pool，最后关 Kafka publisher（LIFO）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if pool != nil {
		srv.RegisterShutdownHook("db-pool", func() error { pool.Close(); return nil })
	}
	if kafkaPub != nil {
		srv.RegisterShutdownHook("kafka-publisher", kafkaPub.Close)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("admin-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("admin-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("admin-service stopped")
}

// buildPool 根据 cfg.DB 构造 pgxpool；DSN 空时返回 (nil, nil)。
func buildPool(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DB.DSN == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return shareddb.NewPool(ctx, shareddb.Config{
		DSN:      cfg.DB.DSN,
		MaxConns: cfg.DB.MaxConns,
		MinConns: cfg.DB.MinConns,
	})
}

// buildPublisher 根据 cfg.Kafka 构造 publisher 与（可选）Kafka publisher 引用。
//
// §32 公共模式：
//   - Brokers 空 → *events.NopPublisher（dev 模式），kafkaPub=nil（无资源需释放）
//   - Brokers 非空 → *events.KafkaPublisher（多 topic：force_cancel/approve/reject/work_order），
//     kafkaPub=同对象供 shutdown hook 释放
func buildPublisher(cfg *config.Config) (service.Publisher, *events.KafkaPublisher) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Warn("admin-service: kafka.brokers empty; publisher disabled (fallback to NopPublisher)")
		return &events.NopPublisher{}, nil
	}
	kp := events.NewKafkaPublisher(cfg.Kafka.Brokers)
	logger.L().Info("admin-service: kafka publisher wired",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.Strings("topics", []string{
			"admin.order.force_cancelled",
			"admin.escort.approved",
			"admin.escort.rejected",
			"admin.refund.approved",
			"admin.refund.rejected",
			"admin.work_order.created",
		}))
	return kp, kp
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

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("admin-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}