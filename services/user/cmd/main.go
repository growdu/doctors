// user-service 入口。
//
// 装配 5 张表仓储：address / coupon / hospital / pkg / virtualnumber。
// DSN 缺失 → 各模块 nil 占位 + warn log（路由仍生效，业务 endpoint 调用时报错）。
//
// §32 Kafka 接入：virtualnumber 模块新增 kafkapublisher.Publisher；
//
//	cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；
//	非空 → 真实 Kafka publisher，订阅 TopicVirtualNumberAllocated/Released。
//
// 优雅停机：srv.RegisterShutdownHook 注册 otel-tracer + db-pool + kafka-publisher。
//
// 注意：用户 profile（users 表）由 auth-service 持有；user-service 通过 auth / gRPC 获取，
//
//	不在本服务装配 PG。
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

	"github.com/growdu/doctors/services/user/internal/address"
	"github.com/growdu/doctors/services/user/internal/coupon"
	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/hospital"
	pkgpkg "github.com/growdu/doctors/services/user/internal/pkg"
	"github.com/growdu/doctors/services/user/internal/server"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/services/user/internal/virtualnumber"
	vnkafkapublisher "github.com/growdu/doctors/services/user/internal/virtualnumber/kafkapublisher"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
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

	cfg, err := config.Load("user")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("user-service", cfg.Tracing.OTLPEndpoint,
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
	metrics.InitMetrics("user-service", cfg.ServiceVersion,
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

	// 5 张表仓储：pool != nil → 各 NewRepo(pool)；否则 nil 占位。
	var (
		addrRepo     address.Repository
		couponRepo   coupon.Repository
		hospitalRepo hospital.Repository
		pkgRepo      pkgpkg.Repository
		vnRepo       virtualnumber.Repository
	)
	if pool != nil {
		addrRepo = address.NewRepo(pool)
		couponRepo = coupon.NewRepo(pool)
		hospitalRepo = hospital.NewRepo(pool)
		pkgRepo = pkgpkg.NewRepo(pool)
		vnRepo = virtualnumber.NewRepo(pool)
		logger.L().Info("user-service: 5 repos wired (address/coupon/hospital/pkg/virtualnumber)")
	} else {
		addrRepo = nilAddrRepo{}
		couponRepo = nilCouponRepo{}
		hospitalRepo = nilHospitalRepo{}
		pkgRepo = nilPackageRepo{}
		vnRepo = nilVNRepo{}
		logger.L().Warn("user-service: cfg.db.dsn empty; 5 repos not wired (fallback to nil)")
	}

	// profile（users 表）暂用 nil 占位：auth-service 持有 users 表，本服务走 auth/gRPC。
	var profileRepo service.ProfileRepo = nilProfileRepo{}

	profileSvc := service.New(profileRepo)
	addrSvc := address.NewService(addrRepo)
	couponSvc := coupon.NewService(couponRepo)
	hospitalSvc := hospital.NewService(hospitalRepo)
	pkgSvc := pkgpkg.NewService(pkgRepo)
	// virtualnumber publisher：cfg.Kafka.Brokers 空 → nilPublisher；非空 → 真实 Kafka。
	vnPublisher, kafkaCloser := buildVNPublisher(cfg)
	vnSvc := virtualnumber.NewService(vnRepo).WithPublisher(vnPublisher)

	h := handler.New(profileSvc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret, addrSvc, couponSvc, hospitalSvc, pkgSvc, vnSvc)

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

	logger.FromContext(ctx).Info("user-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("user-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("user-service stopped")
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

// buildVNPublisher 根据 cfg.Kafka 构造 virtualnumber publisher 与可选 closer。
//
// §32 公共模式：Brokers 空 → *virtualnumber.nilPublisher（dev）+ nil closer；
//
//	非空 → *vnkafkapublisher.Publisher + 同对象 closer。
func buildVNPublisher(cfg *config.Config) (virtualnumber.Publisher, func() error) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Warn("user-service: kafka.brokers empty; virtualnumber publisher disabled (fallback to nil)")
		// 复用 virtualnumber 包的 nil 占位（NewService 已默认注入；这里返回 nil 让 main 跳过 WithPublisher）。
		return nil, nil
	}
	kp, err := vnkafkapublisher.New(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatalf("user-service: build kafka publisher: %v", err)
	}
	logger.L().Info("user-service: virtualnumber kafka publisher wired",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.Strings("topics", []string{contracts.TopicVirtualNumberAllocated, contracts.TopicVirtualNumberReleased}))
	return kp, kp.Close
}

// ---------- nil 占位（业务接口 fallback；调用即返回 errNil） ----------

type nilProfileRepo struct{}

func (nilProfileRepo) GetByID(ctx context.Context, id int64) (*service.Profile, error) {
	return nil, errNil
}
func (nilProfileRepo) UpdateNickname(ctx context.Context, id int64, n string) error { return errNil }
func (nilProfileRepo) UpdateAvatar(ctx context.Context, id int64, u string) error  { return errNil }

type nilAddrRepo struct{}

func (nilAddrRepo) Create(ctx context.Context, a *address.Record) error { return errNil }
func (nilAddrRepo) CreateDefault(ctx context.Context, a *address.Record) error { return errNil }
func (nilAddrRepo) ListByUser(ctx context.Context, userID int64) ([]*address.Record, error) {
	return nil, errNil
}
func (nilAddrRepo) CountByUser(ctx context.Context, userID int64) (int, error) { return 0, errNil }
func (nilAddrRepo) GetByID(ctx context.Context, id, userID int64) (*address.Record, error) {
	return nil, errNil
}
func (nilAddrRepo) Update(ctx context.Context, a *address.Record) error { return errNil }
func (nilAddrRepo) SetDefault(ctx context.Context, id, userID int64) error { return errNil }
func (nilAddrRepo) Delete(ctx context.Context, id, userID int64) error  { return errNil }

type nilCouponRepo struct{}

func (nilCouponRepo) ListActive(ctx context.Context, limit, offset int) ([]*coupon.Record, error) {
	return nil, errNil
}
func (nilCouponRepo) GetByID(ctx context.Context, id int64) (*coupon.Record, error) {
	return nil, errNil
}
func (nilCouponRepo) Claim(ctx context.Context, userID, couponID int64) (*coupon.UserCoupon, error) {
	return nil, errNil
}
func (nilCouponRepo) ListByUser(ctx context.Context, userID int64) ([]*coupon.UserCouponWithTemplate, error) {
	return nil, errNil
}
func (nilCouponRepo) MarkUsed(ctx context.Context, id, userID int64) error { return errNil }

type nilHospitalRepo struct{}

func (nilHospitalRepo) List(ctx context.Context, f hospital.ListFilter) ([]*hospital.Record, int, error) {
	return nil, 0, errNil
}
func (nilHospitalRepo) GetByID(ctx context.Context, id int64) (*hospital.Record, error) {
	return nil, errNil
}

type nilPackageRepo struct{}

func (nilPackageRepo) ListByHospital(ctx context.Context, hospitalID int64) ([]*pkgpkg.Record, error) {
	return nil, errNil
}
func (nilPackageRepo) GetByID(ctx context.Context, id int64) (*pkgpkg.Record, error) {
	return nil, errNil
}

type nilVNRepo struct{}

func (nilVNRepo) Allocate(ctx context.Context, in virtualnumber.AllocateInput) (*virtualnumber.Record, error) {
	return nil, errNil
}
func (nilVNRepo) GetByID(ctx context.Context, id int64) (*virtualnumber.Record, error) {
	return nil, errNil
}
func (nilVNRepo) Release(ctx context.Context, id int64, reason string) (*virtualnumber.Record, error) {
	return nil, errNil
}

var errNil = errors.New("user: repo not wired (cfg.db.dsn empty; configure postgres dsn to enable)")

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
	log.Printf("user-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}