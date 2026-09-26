// auth-service 入口。
//
// 装配流程：
//  1. 读 config（service / http / db / auth 等）
//  2. 构造 pgxpool（cfg.db.dsn 空 → 降级 nil + warn log，路由仍生效但业务 endpoint 不可用）
//  3. 构造 user repo（pool == nil → 用 nilUserRepo 占位；否则走 PG + adapter）
//  4. 构造 sms / wxlogin / realname（默认 mock 实现）
//  5. 构造 service.Service + handler.Handler
//  6. 启动 server.Run(ctx)；注册 shutdown hook：db-pool、otel-tracer。
//
// §32 Kafka 接入策略：
//   - auth-service 是纯 JWT 签发 / 校验服务，不向任何 Kafka topic 发事件；
//   - 用户注册成功 / 实名完成事件由调用方（mobile / wxlogin / realname handler）
//     走 notification-service 直发；auth 不引入 Kafka 依赖；
//   - 因此本服务不实现 buildPublisher / 不注册 kafka-publisher shutdown hook。
//   - 若未来需要广播 UserRegisteredEvent 等事件，新增 kafkapublisher 包并按
//     §32 公共模式装配即可。
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

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/services/auth/internal/realname"
	"github.com/growdu/doctors/services/auth/internal/repo"
	"github.com/growdu/doctors/services/auth/internal/server"
	"github.com/growdu/doctors/services/auth/internal/service"
	"github.com/growdu/doctors/services/auth/internal/sms"
	"github.com/growdu/doctors/services/auth/internal/wxlogin"
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

	cfg, err := config.Load("auth")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	// 生产通过 cfg.Tracing.SamplingRatio（默认 1.0）和 cfg.ServiceVersion（默认 "dev"）注入采样 + 版本。
	traceShutdown, err := tracing.InitTracer("auth-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// 构造 pgxpool；DSN 缺失时降级为 nil（业务 endpoint 调用时由 nilUserRepo 返回 error）。
	pool, err := buildPool(cfg)
	if err != nil {
		log.Fatalf("build pool: %v", err)
	}
	if pool != nil {
		defer pool.Close()
	}

	// Prometheus 业务指标（§29 metrics plan）：service_info + DB pool 采集。
	// pool != nil 时注入 StatProvider；nil 时跳过（不写 DB 指标）。
	metrics.InitMetrics("auth-service", cfg.ServiceVersion,
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

	// 装配 service 依赖（本期均为 mock 实现）。
	smsSender := sms.NewLogSender()
	wxClient := wxlogin.NewMockClient()
	rnVerifier := realname.NewMockVerifier(cfg.Auth.JWTSecret) // salt 与 secret 共用，便于 dev

	// user repo：pool != nil 走真实 PG（adapter 把 repo.User 映射成 service.User）；
	// 否则 nil 占位（路由仍生效，业务 endpoint 会返回 5xx）。
	var userRepo service.UserRepo
	if pool != nil {
		userRepo = newUserRepoAdapter(repo.NewUserRepo(pool))
	} else {
		logger.L().Warn("auth-service: cfg.db.dsn empty; user repo not wired (fallback to nil)")
		userRepo = nilUserRepo{}
	}

	svc := service.New(userRepo, smsSender, wxClient, rnVerifier, cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)
	h := handler.New(svc)

	// 依赖健康检查（/readyz）。pool != nil → 注册 PG checker；nil → skip + warn；
	// cfg.Kafka.Brokers 非空 → 注册 Kafka broker checker；空 → skip + warn。
	// 注：auth-service §32 不接入 Kafka publisher/consumer，所以这里只跑 pool 一个 checker。
	healthM := health.NewManager(health.WithTimeout(1 * time.Second))
	if pool != nil {
		healthM.MustRegister(health.NewPGPoolChecker("postgres-main", pool, time.Second))
		logger.L().Info("auth-service: readyz registered checker: postgres-main")
	} else {
		logger.L().Warn("auth-service: cfg.db.dsn empty; /readyz will fail-closed (postgres not configured)")
	}
	if len(cfg.Kafka.Brokers) > 0 {
		healthM.MustRegister(health.NewKafkaBrokerChecker("kafka-brokers", cfg.Kafka.Brokers, 1*time.Second))
		logger.L().Info("auth-service: readyz registered checker: kafka-brokers",
			zap.Strings("brokers", cfg.Kafka.Brokers))
	}

	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret, healthM)
	// 优雅停机：先关 OTel tracer，再关 DB pool（按注册逆序 LIFO 执行）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if pool != nil {
		srv.RegisterShutdownHook("db-pool", func() error { pool.Close(); return nil })
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("auth-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("auth-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("auth-service stopped")
}

// buildPool 根据 cfg.DB 构造 pgxpool；DSN 空时返回 (nil, nil) —— 调用方按"降级"
//
//	模式装配 nilUserRepo，路由仍能注册（业务 endpoint 调用时才报错）。
//
// cfg.Tracing.OTLPEndpoint 非空 → 把 otelpgx tracer 注入 pcfg，让 SQL 自动写 span。
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
	// §34 OTel auto-instrumentation：endpoint 非空 → 注入 pgx tracer。
	// 注意：buildPool 早于 InitTracer 完成时（dev 模式 otlpEndpoint 空），
	// 此时全局 TracerProvider 是 Noop，OTel 零开销。
	pcfg, err := pgxpool.ParseConfig(poolCfg.DSN)
	if err == nil {
		tracing.WithPgxPool(pcfg)
		poolCfg.Tracer = pcfg.ConnConfig.Tracer
	}
	return shareddb.NewPool(ctx, poolCfg)
}

// userRepoAdapter 把 repo.UserRepo（返回 *repo.User）适配成 service.UserRepo（返回 *service.User）。
//
//	职责单一：仅做字段映射；不引入业务逻辑。auth.repo 不依赖 auth.service，避免循环导入。
type userRepoAdapter struct{ r *repo.UserRepo }

func newUserRepoAdapter(r *repo.UserRepo) *userRepoAdapter { return &userRepoAdapter{r: r} }

func (a *userRepoAdapter) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	u := &repo.User{Phone: phone, Role: repo.Role(role), WxUnionID: unionid}
	if err := a.r.Create(ctx, u); err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (a *userRepoAdapter) FindByPhone(ctx context.Context, phone string) (*service.User, error) {
	u, err := a.r.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	return toServiceUser(u), nil
}

func (a *userRepoAdapter) FindByUnionID(ctx context.Context, unionid string) (*service.User, error) {
	u, err := a.r.FindByUnionID(ctx, unionid)
	if err != nil {
		return nil, err
	}
	return toServiceUser(u), nil
}

func (a *userRepoAdapter) FindByID(ctx context.Context, id int64) (*service.User, error) {
	u, err := a.r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toServiceUser(u), nil
}

func (a *userRepoAdapter) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return a.r.UpdateRealName(ctx, id, hash, tail)
}

// toServiceUser 把 repo.User 收敛成 service.User（只保留业务层需要的字段）。
func toServiceUser(u *repo.User) *service.User {
	return &service.User{
		ID:               u.ID,
		Phone:            u.Phone,
		Role:             string(u.Role),
		UnionID:          u.WxUnionID,
		RealNameVerified: u.RealNameVerified,
	}
}

// nilUserRepo 是为了让 main 在 DSN 缺失时仍能编译 / 启动；任何方法被调用
//
//	都返回 error，便于 dev 期间快速发现。
type nilUserRepo struct{}

func (nilUserRepo) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	return 0, errNilUserRepo
}
func (nilUserRepo) FindByPhone(ctx context.Context, phone string) (*service.User, error) {
	return nil, errNilUserRepo
}
func (nilUserRepo) FindByUnionID(ctx context.Context, unionid string) (*service.User, error) {
	return nil, errNilUserRepo
}
func (nilUserRepo) FindByID(ctx context.Context, id int64) (*service.User, error) {
	return nil, errNilUserRepo
}
func (nilUserRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return errNilUserRepo
}

var errNilUserRepo = &errNilSentinel{}

type errNilSentinel struct{}

func (e *errNilSentinel) Error() string {
	return "auth: user repo not wired (cfg.db.dsn empty; configure postgres dsn to enable)"
}

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

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("auth-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}