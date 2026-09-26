// sos-service 入口。
//
// 设计要点：
//  - v1 简化为 in-memory store：SOS 按 ID 索引 + 按 orderID 反查去重。
//  - 路由生效，业务 endpoint 可用（重启会丢 SOS 记录，符合 v1 dev 范围）。
//  - §32 Kafka 接入：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；
//    非空 → kafkapublisher.Publisher 真实 Kafka，订阅 TopicSOSRaised。
//  - 本服务不接 DB（§31）；注册 otel-tracer + kafka-publisher shutdown hook。
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

	"github.com/growdu/doctors/services/sos/internal/handler"
	"github.com/growdu/doctors/services/sos/internal/kafkapublisher"
	"github.com/growdu/doctors/services/sos/internal/router"
	"github.com/growdu/doctors/services/sos/internal/server"
	"github.com/growdu/doctors/services/sos/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/contracts"
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

	cfg, err := config.Load("sos")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("sos-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// Prometheus 业务指标（§29 metrics plan）：service_info 已就绪。
	// sos 无 DB pool → 不注入 StatProvider。
	metrics.InitMetrics("sos-service", cfg.ServiceVersion)

	// v1 in-memory repo：SOS 自增 ID；同 orderID 5min 内去重。
	memRepo := newMemoryRepo()

	// 订单状态查询：v1 用 activeLookup{} 返回 true（dev 模式）。
	// 接 order-service 后改用 gRPC / 直连。
	orderLookup := activeLookup{}

	// Publisher：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；非空 → 真实 Kafka publisher。
	publisher, kafkaCloser := buildPublisher(cfg)

	svc := service.New(memRepo, orderLookup, publisher)
	h := handler.New(svc)

	// 依赖健康检查（/readyz）。sos 无 DB；kafka brokers 非空时注册 checker。
	healthM := health.NewManager(health.WithTimeout(1 * time.Second))
	if len(cfg.Kafka.Brokers) > 0 {
		healthM.MustRegister(health.NewKafkaBrokerChecker("kafka-brokers", cfg.Kafka.Brokers, 1*time.Second))
		logger.L().Info("sos-service: readyz registered checker: kafka-brokers",
			zap.Strings("brokers", cfg.Kafka.Brokers))
	} else {
		logger.L().Warn("sos-service: kafka.brokers empty; /readyz will fail-closed (no dependencies registered)")
	}

	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret, healthM))

	// 优雅停机：先关 OTel tracer，再关 Kafka publisher（LIFO）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if kafkaCloser != nil {
		srv.RegisterShutdownHook("kafka-publisher", kafkaCloser)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("sos-service starting",
		zap.String("addr", cfg.HTTP.Addr),
		zap.String("store", "in-memory"),
	)
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("sos-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("sos-service stopped")
}

// ---------- memoryRepo（v1 简化实现） ----------

// memoryRepo 是 in-memory 的 sos.repo 实现。
//
//	线程安全：RWMutex 保护 sos map；ID 用 atomic int64 自增。
//	IsRecentDuplicate 通过遍历全表（v1 简化；v2 应建索引）。
type memoryRepo struct {
	mu     sync.RWMutex
	sos    map[int64]*service.SOS
	nextID int64
}

// newMemoryRepo 构造一个空的 in-memory repo。
func newMemoryRepo() *memoryRepo {
	return &memoryRepo{sos: make(map[int64]*service.SOS)}
}

// Create 写入新 SOS 记录。
func (r *memoryRepo) Create(ctx context.Context, s *service.SOS) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.ID = atomic.AddInt64(&r.nextID, 1)
	if s.RaisedAt.IsZero() {
		s.RaisedAt = time.Now()
	}
	if s.Status == "" {
		s.Status = "raised"
	}
	r.sos[s.ID] = s
	return nil
}

// GetByID 按 ID 取；未命中返回 ErrSOSNotFound。
func (r *memoryRepo) GetByID(ctx context.Context, id int64) (*service.SOS, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sos[id]
	if !ok {
		return nil, service.ErrSOSNotFound
	}
	return s, nil
}

// UpdateStatus 更新 SOS 状态（raised → handling → resolved）。
func (r *memoryRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sos[id]
	if !ok {
		return service.ErrSOSNotFound
	}
	s.Status = status
	return nil
}

// IsRecentDuplicate 5min 内同 orderID 是否已有 SOS（防误触）。
func (r *memoryRepo) IsRecentDuplicate(ctx context.Context, orderID int64, since time.Time) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sos {
		if s.OrderID == orderID && s.RaisedAt.After(since) {
			return true, nil
		}
	}
	return false, nil
}

// List 按过滤条件查询（v1 简化：status / orderID 过滤；时间倒序；分页）。
func (r *memoryRepo) List(ctx context.Context, f service.ListFilter) ([]*service.SOS, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*service.SOS, 0, len(r.sos))
	for _, s := range r.sos {
		if f.Status != "" && s.Status != f.Status {
			continue
		}
		if f.OrderID > 0 && s.OrderID != f.OrderID {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RaisedAt.After(out[j].RaisedAt) })
	limit := f.PageSize
	if limit <= 0 {
		limit = 20
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	if offset >= len(out) {
		return []*service.SOS{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

// activeLookup v1 占位：所有订单视为 active。接 order-service 后替换。
type activeLookup struct{}

func (activeLookup) IsOrderActive(ctx context.Context, orderID int64) (bool, error) {
	return true, nil
}

// nilPublisher 是 Kafka 缺失时的占位：事件不外发；SOS 已写入 in-memory。
type nilPublisher struct{}

func (nilPublisher) PublishSOSRaised(ctx context.Context, ev contracts.SOSRaisedEvent) error {
	return nil
}

// buildPublisher 根据 cfg.Kafka 构造 publisher 与可选 closer。
//
// §32 公共模式：Brokers 空 → *nilPublisher + nil closer；非空 → *kafkapublisher.Publisher + 同对象 closer。
func buildPublisher(cfg *config.Config) (service.Publisher, func() error) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Warn("sos-service: kafka.brokers empty; publisher disabled (fallback to nilPublisher)")
		return nilPublisher{}, nil
	}
	kp, err := kafkapublisher.New(cfg.Kafka.Brokers, contracts.TopicSOSRaised)
	if err != nil {
		log.Fatalf("sos-service: build kafka publisher: %v", err)
	}
	logger.L().Info("sos-service: kafka publisher wired",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("topic", contracts.TopicSOSRaised))
	return kp, kp.Close
}

// 引用占位 errors（避免 unused）。
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
	log.Printf("sos-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}