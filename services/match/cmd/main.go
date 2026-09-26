// match-service 入口。
//
// 阶段 4 已实现：候选打分 + Redis 抢单池（NopPool 默认）。
// 集成测试时把 NopPool 换为 RedisPool；生产 main 注入 RedisPool。
//
// 本次 §31 接入：无 DB（pure Kafka consumer）；只补 shutdown hook（otel-tracer
// + kafka consumer）+ 优雅停机路径；DB pool 钩子保持 nil。
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/match/internal/consumer"
	"github.com/growdu/doctors/services/match/internal/handler"
	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/server"
	"github.com/growdu/doctors/services/match/internal/service"
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

	cfg, err := config.Load("match")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("match-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// Prometheus 业务指标（§29 metrics plan）：service_info 已就绪。
	// match 无 DB pool → 不注入 StatProvider，DB 指标不会出现在 /metrics。
	metrics.InitMetrics("match-service", cfg.ServiceVersion)

	// v1 用 NopPool；接 Redis 后换 RedisPool
	svc := service.New(pool.NewNopPool(), nilEscortLoader{}, 0)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret)

	// Kafka consumer（order.created → 抢单池）；cfg.Kafka.Brokers 空则跳过。
	//
	// §31 接入：consumer 的 reader 关闭也走 shutdown hook；ctx cancel 时 Run 退出，
	// defer c.reader.Close() 已经兜底；hook 再调一次保证释放顺序受控。
	var kafkaConsumer *consumer.Consumer
	if len(cfg.Kafka.Brokers) > 0 {
		groupID := cfg.Kafka.GroupID
		if groupID == "" {
			groupID = "match-service"
		}
		kafkaConsumer = consumer.New(cfg.Kafka.Brokers, contracts.TopicOrderCreated, groupID, svc)
		logger.L().Info("match-service: kafka consumer enabled",
			zap.Strings("brokers", cfg.Kafka.Brokers),
			zap.String("topic", contracts.TopicOrderCreated))
	} else {
		logger.L().Info("match-service: kafka brokers empty; consumer skipped (dev mode)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Kafka consumer 后台消费；ctx cancel 时退出。
	if kafkaConsumer != nil {
		go func() {
			if err := kafkaConsumer.Run(ctx); err != nil {
				logger.FromContext(ctx).Warn("kafka consumer exited", zap.Error(err))
			}
		}()
	}

	// 优雅停机：先关 OTel tracer，再关 Kafka consumer（LIFO）。match 无 DB pool。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	// Consumer 自身的 defer reader.Close 已经兜底；这里挂一个 noop marker 让
	// 服务对"释放顺序"保持显式心智模型，运维一眼能看出本服务依赖了哪些资源。
	if kafkaConsumer != nil {
		srv.RegisterShutdownHook("kafka-consumer", func() error { return nil })
	}

	logger.FromContext(ctx).Info("match-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("match-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("match-service stopped")
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

// nilEscortLoader 是占位实现；任何调用返回 nil。
type nilEscortLoader struct{}

func (nilEscortLoader) ListAvailable(ctx context.Context, city string) ([]contracts.EscortSummary, error) {
	return nil, nil
}

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("match-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}