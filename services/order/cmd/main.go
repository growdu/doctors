// order-service 入口。
//
// 设计要点：
//  - 阶段 3.6 已实现 Accept 抢单；main 接入 PG（orders + order_events）。
//  - §4.2 order-lock plan：可选装配 RedisLocker / KafkaPublisher /
//    ExpiredLockScanner。配置存在就启用，否则降级为 NopLocker / NopPublisher / 不跑 scheduler。
//  - cfg.db.dsn 空 → pool=nil + warn log；orderRepo 用 nil 占位（业务 endpoint 调用时报错）。
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
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/order/internal/events"
	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/scheduler"
	"github.com/growdu/doctors/services/order/internal/server"
	"github.com/growdu/doctors/services/order/internal/service"
	"github.com/growdu/doctors/shared/config"
	shareddb "github.com/growdu/doctors/shared/db"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/lock"
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

	cfg, err := config.Load("order")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("order-service", cfg.Tracing.OTLPEndpoint,
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
	// pool != nil 时注入 StatProvider；nil 时跳过。
	metrics.InitMetrics("order-service", cfg.ServiceVersion,
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

	// orderRepo：pool != nil 走真实 PG（orders + order_events）；
	// 否则 nil 占位（路由仍生效，业务 endpoint 调用时返回 error）。
	var orderRepo service.OrderRepo
	if pool != nil {
		orderRepo = repo.NewOrderRepo(pool)
	} else {
		logger.L().Warn("order-service: cfg.db.dsn empty; order repo not wired (fallback to nil)")
		orderRepo = nilOrderRepo{}
	}

	// §4.2 可选装配：Redis 锁单 + Kafka 事件 + Scheduler
	locker, lockerCloser, redisClient := buildLocker(cfg)
	if closer, ok := lockerCloser.(interface{ Close() error }); ok && closer != nil {
		defer func() { _ = closer.Close() }()
	}
	publisher, pubCloser := buildPublisher(cfg)
	if pubCloser != nil {
		defer func() { _ = pubCloser.Close() }()
	}

	svc := service.New(orderRepo, unwiredUsers{}).
		WithLocker(locker).
		WithPublisher(publisher)
	h := handler.New(svc)

	// 依赖健康检查（/readyz）。order 有 DB + Redis（locker）+ Kafka publisher 三依赖。
	healthM := health.NewManager(health.WithTimeout(1 * time.Second))
	if pool != nil {
		healthM.MustRegister(health.NewPGPoolChecker("postgres-main", pool, time.Second))
		logger.L().Info("order-service: readyz registered checker: postgres-main")
	} else {
		logger.L().Warn("order-service: cfg.db.dsn empty; /readyz will fail-closed (postgres not configured)")
	}
	if redisClient != nil {
		healthM.MustRegister(health.NewRedisChecker("redis-main", redisClient, time.Second))
		logger.L().Info("order-service: readyz registered checker: redis-main")
	}
	if len(cfg.Kafka.Brokers) > 0 {
		healthM.MustRegister(health.NewKafkaBrokerChecker("kafka-brokers", cfg.Kafka.Brokers, 1*time.Second))
		logger.L().Info("order-service: readyz registered checker: kafka-brokers",
			zap.Strings("brokers", cfg.Kafka.Brokers))
	}

	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret, healthM)
	// 优雅停机：先关 OTel tracer，再关 DB pool，最后关 Kafka/Redis（LIFO）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if pool != nil {
		srv.RegisterShutdownHook("db-pool", func() error { pool.Close(); return nil })
	}
	if pubCloser != nil {
		srv.RegisterShutdownHook("kafka-publisher", func() error { return pubCloser.Close() })
	}
	if closer, ok := lockerCloser.(interface{ Close() error }); ok && closer != nil {
		srv.RegisterShutdownHook("redis-locker", closer.Close)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// §4.2 启动 ExpiredLockScanner（5s 扫一次过期锁单；orderRepo=nil 时不启动）
	if orderRepo != nil && svc != nil {
		go scheduler.NewExpiredLockScanner(orderRepo, svc, publisher, 5*time.Second).Run(ctx)
	}

	logger.FromContext(ctx).Info("order-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("order-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("order-service stopped")
}

// buildPool 根据 cfg.DB 构造 pgxpool；DSN 空时返回 (nil, nil) —— 调用方按"降级"
//
//	模式装配 nilOrderRepo，路由仍能注册（业务 endpoint 调用时才报错）。
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

// buildLocker 根据 cfg.Redis 配置构造 Locker；配置缺失 → NopLocker。
//
// 返回 (locker, closer, redisClient)：redisClient 用于 /readyz 的 redis checker；
// 配置缺失时 redisClient 为 nil（health checker 自动 skip）。
func buildLocker(cfg *config.Config) (lock.Locker, any, *redis.Client) {
	if cfg.Redis.Addr == "" {
		logger.L().Info("locker disabled (redis.addr empty); falling back to NopLocker")
		return lock.NopLocker{}, nil, nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
		DB:   cfg.Redis.DB,
	})
	logger.L().Info("locker enabled (redis SETNX)", zap.String("addr", cfg.Redis.Addr))
	return lock.NewRedisLocker(rdb), rdb, rdb
}

// buildPublisher 根据 cfg.Kafka 配置构造 Publisher；配置缺失 → NopPublisher。
func buildPublisher(cfg *config.Config) (events.Publisher, events.Publisher) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Info("publisher disabled (kafka.brokers empty); falling back to NopPublisher")
		return &events.NopPublisher{}, nil
	}
	logger.L().Info("publisher enabled (kafka)", zap.Strings("brokers", cfg.Kafka.Brokers))
	p := events.NewKafkaPublisher(cfg.Kafka.Brokers)
	return p, p
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

// nilOrderRepo 是为了让 main 在 DSN 缺失时仍能编译 / 启动；任何方法被调用
//
//	都返回 error，便于 dev 期间快速发现。
type nilOrderRepo struct{}

func (nilOrderRepo) Create(ctx context.Context, o *repo.Order) error {
	return errNilOrderRepo
}
func (nilOrderRepo) FindByID(ctx context.Context, id int64) (*repo.Order, error) {
	return nil, errNilOrderRepo
}
func (nilOrderRepo) ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error) {
	return nil, errNilOrderRepo
}
func (nilOrderRepo) ListByEscort(ctx context.Context, escortID int64, statusFilter string, limit, offset int) ([]*repo.Order, error) {
	return nil, errNilOrderRepo
}
func (nilOrderRepo) UpdateStatus(ctx context.Context, id int64, to string, expectVersion int, escortID *int64) error {
	return errNilOrderRepo
}
func (nilOrderRepo) InsertEvent(ctx context.Context, orderID int64, from *string, to string, actorID *int64, payload []byte) error {
	return errNilOrderRepo
}
func (nilOrderRepo) ListEvents(ctx context.Context, orderID int64) ([]*repo.OrderEvent, error) {
	return nil, errNilOrderRepo
}
func (nilOrderRepo) SelectForEscort(ctx context.Context, id, escortID int64, expireAt time.Time, expectVersion int) error {
	return errNilOrderRepo
}
func (nilOrderRepo) ConfirmByEscort(ctx context.Context, id, escortID int64, now time.Time, expectVersion int) error {
	return errNilOrderRepo
}
func (nilOrderRepo) RejectByEscort(ctx context.Context, id, escortID int64, expectVersion int) error {
	return errNilOrderRepo
}
func (nilOrderRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
	return nil, errNilOrderRepo
}

var errNilOrderRepo = errOrderSentinel("order: order repo not wired (cfg.db.dsn empty; configure postgres dsn to enable)")

type errOrderSentinel string

func (e errOrderSentinel) Error() string { return string(e) }

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("order-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}