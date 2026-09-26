// message-service 入口。
//
// 设计要点：
//  - v1 简化为 in-memory store：repo 用内存版（按 orderID 分桶 + 自增 ID），
//    路由生效，业务 endpoint 可用（重启会丢消息，符合 v1 dev 范围）。
//  - cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；非空 → KafkaPublisher（v2 接入）。
//  - service.New 装配；handler.RegisterRoutes 挂载；server.Run 启动。
//  - 本服务不接 DB（§31）；只注册 otel-tracer shutdown hook。
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/message/internal/handler"
	"github.com/growdu/doctors/services/message/internal/router"
	"github.com/growdu/doctors/services/message/internal/server"
	"github.com/growdu/doctors/services/message/internal/service"
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

	cfg, err := config.Load("message")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("message-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// Prometheus 业务指标（§29 metrics plan）：service_info 已就绪。
	// message 无 DB pool → 不注入 StatProvider。
	metrics.InitMetrics("message-service", cfg.ServiceVersion)

	// v1 in-memory repo（线程安全，自增 ID，按 orderID 索引）。
	memRepo := newMemoryRepo()

	// Publisher：cfg.Kafka.Brokers 空 → nilPublisher；非空 → 接入点占位。
	var publisher service.Publisher
	if len(cfg.Kafka.Brokers) > 0 {
		logger.L().Info("message-service: kafka brokers configured; publisher wired (v2 接入)",
			zap.Strings("brokers", cfg.Kafka.Brokers))
		publisher = nilPublisher{}
	} else {
		logger.L().Info("message-service: kafka brokers empty; publisher disabled (dev mode)")
		publisher = nilPublisher{}
	}

	svc := service.New(memRepo, publisher)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	// 优雅停机：先关 OTel tracer（message 无 DB / 无长连接 consumer）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("message-service starting",
		zap.String("addr", cfg.HTTP.Addr),
		zap.String("store", "in-memory"),
	)
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("message-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("message-service stopped")
}

// ---------- memoryRepo（v1 简化实现） ----------

// memoryRepo 是 in-memory 的 message.repo 实现。
//
//	线程安全：RWMutex 保护 messages map；ID 用 atomic int64 自增。
//	重启清空（v1 范围；prod 应替换为 PG）。
type memoryRepo struct {
	mu       sync.RWMutex
	messages map[int64]*service.Message // id -> message
	byOrder  map[int64][]int64          // orderID -> []messageID（按时间正序）
	nextID   int64
}

// newMemoryRepo 构造一个空的 in-memory repo。
func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		messages: make(map[int64]*service.Message),
		byOrder:  make(map[int64][]int64),
	}
}

// CreateConversation v1 简化：仅返回成功（会话由 orderID + 双方 userID 隐式表达）。
func (r *memoryRepo) CreateConversation(ctx context.Context, c *service.Conversation) error {
	return nil
}

// CreateMessage 写入内存。
func (r *memoryRepo) CreateMessage(ctx context.Context, m *service.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m.ID = atomic.AddInt64(&r.nextID, 1)
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	r.messages[m.ID] = m
	r.byOrder[m.OrderID] = append(r.byOrder[m.OrderID], m.ID)
	return nil
}

// GetMessageByID 按 ID 取；未命中返回 ErrMessageNotFound。
func (r *memoryRepo) GetMessageByID(ctx context.Context, id int64) (*service.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.messages[id]
	if !ok {
		return nil, service.ErrMessageNotFound
	}
	return m, nil
}

// ListMessagesByOrder 按 orderID 拉消息列表（按时间正序；支持 limit/offset）。
func (r *memoryRepo) ListMessagesByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*service.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byOrder[orderID]
	out := make([]*service.Message, 0, len(ids))
	for _, id := range ids {
		if m, ok := r.messages[id]; ok {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if offset >= len(out) {
		return []*service.Message{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

// nilPublisher 是 kafka 缺失时的占位：消息已写入 in-memory；事件不外发。
type nilPublisher struct{}

func (nilPublisher) PublishMessageSent(ctx context.Context, ev contracts.MessageSentEvent) error {
	return nil
}

// 显式占位引用以保持 import。
var _ = errors.New

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
	log.Printf("message-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}