// admin-service 入口。
//
// 设计要点：
//   - pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
//   - 内部 clients（order/refund/escort/user）从 cfg 读 baseURL；smoke 用占位。
//   - Kafka 配置缺失 → NopPublisher（dev / 单测友好）。
//   - service.New 装配；handler.RegisterRoutes 挂载；server.Run 启动。
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

	"github.com/growdu/doctors/services/admin/internal/clients"
	"github.com/growdu/doctors/services/admin/internal/events"
	"github.com/growdu/doctors/services/admin/internal/handler"
	"github.com/growdu/doctors/services/admin/internal/repo"
	"github.com/growdu/doctors/services/admin/internal/router"
	"github.com/growdu/doctors/services/admin/internal/server"
	"github.com/growdu/doctors/services/admin/internal/service"
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

	cfg, err := config.Load("admin")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("admin-service", cfg.Tracing.OTLPEndpoint)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = traceShutdown(context.Background()) }()

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// pool=nil：路由生效；业务调用会 panic；smoke 不走业务路径。
	woRepo := repo.NewWorkOrderRepo(nil)
	rrRepo := repo.NewReportsRepo(nil)

	// 内部 clients（baseURL 从配置读；smoke 用占位）
	orderClient := clients.NewOrderClient(cfg.Admin.OrderBaseURL, 5*time.Second)
	refundClient := clients.NewRefundClient(cfg.Admin.RefundBaseURL, 5*time.Second)
	escortClient := clients.NewEscortClient(cfg.Admin.EscortBaseURL, 5*time.Second)
	userClient := clients.NewUserClient(cfg.Admin.UserBaseURL, 5*time.Second)

	// Publisher（Kafka 或 Nop）
	var pub service.Publisher
	if len(cfg.Kafka.Brokers) > 0 {
		kp := events.NewKafkaPublisher(cfg.Kafka.Brokers)
		defer func() { _ = kp.Close() }()
		pub = kp
	} else {
		pub = &events.NopPublisher{}
	}

	svc := service.New(rrRepo, woRepo, orderClient, refundClient, escortClient, userClient, pub)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("admin-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("admin-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("admin-service stopped")
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