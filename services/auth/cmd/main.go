// auth-service 入口。
//
// 装配流程：
//  1. 读 config（service / http / db / auth 等）
//  2. 构造 user repo（pgx pool）
//  3. 构造 sms / wxlogin / realname（默认 mock 实现）
//  4. 构造 service.Service + handler.Handler
//  5. 启动 server.Run(ctx)
//
// 本期没有真实 DB 连接（业务层用 fake 不合适），所以 main 不直连 pgxpool：
// pgxpool 仍需要等到业务层真正接入 repo（阶段 2.9 smoke）。
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

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/services/auth/internal/realname"
	"github.com/growdu/doctors/services/auth/internal/server"
	"github.com/growdu/doctors/services/auth/internal/service"
	"github.com/growdu/doctors/services/auth/internal/sms"
	"github.com/growdu/doctors/services/auth/internal/wxlogin"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
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

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// 装配 service 依赖（本期均为 mock 实现）
	smsSender := sms.NewLogSender()
	wxClient := wxlogin.NewMockClient()
	rnVerifier := realname.NewMockVerifier(cfg.Auth.JWTSecret) // salt 与 secret 共用，便于 dev

	// user repo 需要 pgxpool；当前阶段未接真 DB，业务层走 fake 路径。
	// 这里把 repo 设为 nil，业务层若尝试调用会 panic；smoke 阶段会替换为真 repo。
	// 为让 main 启动不至于 panic，这里组装一个 nil 哨兵：
	svc := service.New(nilRepo{}, smsSender, wxClient, rnVerifier, cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)
	h := handler.New(svc)

	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("auth-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("auth-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("auth-service stopped")
}

// nilRepo 是为了让 main 能编译 / 启动；接入 pgxpool 后会替换。
// 任何方法被调用都返回错误，便于 dev 期间快速发现。
type nilRepo struct{}

func (nilRepo) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	return 0, errNilRepo
}
func (nilRepo) FindByPhone(ctx context.Context, phone string) (*service.User, error) {
	return nil, errNilRepo
}
func (nilRepo) FindByUnionID(ctx context.Context, unionid string) (*service.User, error) {
	return nil, errNilRepo
}
func (nilRepo) FindByID(ctx context.Context, id int64) (*service.User, error) {
	return nil, errNilRepo
}
func (nilRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return errNilRepo
}

var errNilRepo = &errsSentinel{}

type errsSentinel struct{}

func (e *errsSentinel) Error() string { return "auth: user repo not wired (阶段 2.9 后接通)" }

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

// 显式保留 time 引用避免 lint
var _ = time.Second