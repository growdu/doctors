// wallet-service 入口。
//
// 设计要点：
//   - 装配 config + logger + DB pool + repo + service + handler + router + server + scanner + Kafka consumer。
//   - cfg.DB.DSN 缺失 → 起空 pool；仅 /healthz 工作；业务 endpoint 调用会报错（v1 dev 阶段）。
//   - cfg.Kafka.Brokers 缺失 → Kafka 监听降级为 noop（不启动 consumer）。
//   - Scanner 间隔 = 1 分钟（v1 测试用；生产改 7*24h + daily tick，由环境变量 DOCTORS_WALLET_THRESHOLD 覆盖）。
//   - 优雅停机：ctx.Done 触发 server.Shutdown + scanner 退出。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/wallet/internal/handler"
	"github.com/growdu/doctors/services/wallet/internal/repo"
	"github.com/growdu/doctors/services/wallet/internal/router"
	"github.com/growdu/doctors/services/wallet/internal/server"
	"github.com/growdu/doctors/services/wallet/internal/service"
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

	cfg, err := config.Load("wallet")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("wallet-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}
	defer func() { _ = traceShutdown(context.Background()) }()

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	pool, err := buildPool(cfg)
	if err != nil {
		log.Fatalf("build pool: %v", err)
	}
	if pool != nil {
		defer pool.Close()
	}

	// Prometheus 业务指标（§29 metrics plan）：service_info + DB pool 采集。
	// pool 已构建 → 闭包在 30s ticker 中读取 pool.Stat()。
	metrics.InitMetrics("wallet-service", cfg.ServiceVersion,
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

	walletRepo := repo.NewWalletRepo(pool)
	svc := service.New(walletRepo)
	h := handler.New(svc, svc) // service 同时实现 Service + AdminSvc
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Kafka consumer（OrderCompleted → wallet.OnOrderCompleted + PaymentRefunded → wallet.OnRefund）
	consumeKafka(ctx, cfg, svc)

	// T+7 scanner
	threshold, tick := walletScannerParams(cfg)
	if pool != nil {
		go service.NewScanner(walletRepo, threshold, time.Now).WithTick(tick).Run(ctx)
	} else {
		logger.FromContext(ctx).Warn("wallet-service: no DB pool; scanner disabled")
	}

	logger.FromContext(ctx).Info("wallet-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("wallet-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("wallet-service stopped")
}

func buildPool(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DB.DSN == "" {
		logger.L().Warn("wallet-service: cfg.db.dsn empty; running in healthz-only mode")
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return pgxpool.New(ctx, cfg.DB.DSN)
}

// walletScannerParams 从 cfg 解析 scanner 间隔。v1 dev 默认 1 分钟；生产用 7*24h + daily tick。
func walletScannerParams(cfg *config.Config) (threshold, tick time.Duration) {
	threshold = 1 * time.Minute
	tick = 1 * time.Minute
	if v := os.Getenv("DOCTORS_WALLET_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			threshold = d
			tick = d
		}
	}
	return
}

// consumeKafka 启动 OrderCompleted + PaymentRefunded 消费者。
// cfg.Kafka.Brokers 空时跳过（dev 模式）。
func consumeKafka(ctx context.Context, cfg *config.Config, svc *service.Service) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Info("wallet-service: kafka brokers empty; consumer skipped")
		return
	}
	groupID := cfg.Kafka.GroupID
	if groupID == "" {
		groupID = "wallet-service"
	}
	go consumeTopic(ctx, cfg.Kafka.Brokers, groupID, contracts.TopicOrderCompleted, func(ctx context.Context, value []byte) error {
		var ev contracts.OrderCompletedEvent
		if err := json.Unmarshal(value, &ev); err != nil {
			logger.FromContext(ctx).Warn("unmarshal order.completed failed", zap.Error(err))
			return nil
		}
		amt, _ := decimal.NewFromString(formatFloat(ev.Amount))
		return svc.OnOrderCompleted(ctx, ev.EscortID, ev.OrderID, amt)
	})
	go consumeTopic(ctx, cfg.Kafka.Brokers, groupID, contracts.TopicPaymentRefunded, func(ctx context.Context, value []byte) error {
		var ev contracts.PaymentRefundedEvent
		if err := json.Unmarshal(value, &ev); err != nil {
			logger.FromContext(ctx).Warn("unmarshal payment.refunded failed", zap.Error(err))
			return nil
		}
		amt, _ := decimal.NewFromString(formatFloat(ev.Amount))
		return svc.OnRefund(ctx, ev.OrderID, ev.OrderID, amt)
	})
}

func consumeTopic(ctx context.Context, brokers []string, groupID, topic string, handle func(context.Context, []byte) error) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	defer func() { _ = r.Close() }()
	logger.FromContext(ctx).Info("wallet kafka consumer started", zap.String("topic", topic))
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.FromContext(ctx).Warn("kafka read failed", zap.String("topic", topic), zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		if err := handle(ctx, m.Value); err != nil {
			logger.FromContext(ctx).Warn("kafka handler failed", zap.String("topic", topic), zap.Error(err))
		}
	}
}

// formatFloat 把 float64 转为 decimal.NewFromString 可接受的字符串。
// 精度截断到 2 位小数（与 NUMERIC(10,2) 一致）；避免 strconv 的科学计数法 / NaN 问题。
func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
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

// 引用 strconv 避免 unused。
var _ = strconv.Itoa

// runHealthzServer 在 :9090 起独立 http server，仅暴露 /healthz。
// 用于 distroless 镜像的 Docker HEALTHCHECK：进程存活 → 200 OK。
func runHealthzServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("wallet-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}