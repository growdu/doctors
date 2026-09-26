// review-service 入口。
//
// 设计要点：
//  - v1 简化为 in-memory store：review 按 orderID 唯一索引，自增 ID。
//  - 路由生效，业务 endpoint 可用（重启会丢评价，符合 v1 dev 范围）。
//  - §32 Kafka 接入：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；
//    非空 → kafkapublisher.Publisher 真实 Kafka，订阅 TopicOrderReviewed。
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

	"github.com/growdu/doctors/services/review/internal/handler"
	"github.com/growdu/doctors/services/review/internal/kafkapublisher"
	"github.com/growdu/doctors/services/review/internal/router"
	"github.com/growdu/doctors/services/review/internal/server"
	"github.com/growdu/doctors/services/review/internal/service"
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

	cfg, err := config.Load("review")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// OTel 全链路追踪（§26 tracing plan）：endpoint 留空 → Noop，零开销。
	traceShutdown, err := tracing.InitTracer("review-service", cfg.Tracing.OTLPEndpoint,
		tracing.WithSamplingRatio(cfg.Tracing.SamplingRatio),
		tracing.WithServiceVersion(cfg.ServiceVersion),
	)
	if err != nil {
		log.Fatalf("init tracer: %v", err)
	}

	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// Prometheus 业务指标（§29 metrics plan）：service_info 已就绪。
	// review 无 DB pool → 不注入 StatProvider。
	metrics.InitMetrics("review-service", cfg.ServiceVersion)

	// v1 in-memory repo：按 orderID 唯一，自增 ID，按 escortID 分桶。
	memRepo := newMemoryRepo()

	// Publisher：cfg.Kafka.Brokers 空 → nilPublisher（dev / 单测友好）；非空 → 真实 Kafka publisher。
	publisher, kafkaCloser := buildPublisher(cfg)

	svc := service.New(memRepo, publisher)
	h := handler.New(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(h, cfg.Auth.JWTSecret))

	// 优雅停机：先关 OTel tracer，再关 Kafka publisher（LIFO）。
	srv.RegisterShutdownHook("otel-tracer", func() error { return traceShutdown(context.Background()) })
	if kafkaCloser != nil {
		srv.RegisterShutdownHook("kafka-publisher", kafkaCloser)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.FromContext(ctx).Info("review-service starting",
		zap.String("addr", cfg.HTTP.Addr),
		zap.String("store", "in-memory"),
	)
	if err := srv.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("review-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.FromContext(ctx).Info("review-service stopped")
}

// ---------- memoryRepo（v1 简化实现） ----------

// memoryRepo 是 in-memory 的 review.repo 实现。
//
//	线程安全：RWMutex 保护 reviews map；ID 用 atomic int64 自增。
//	按 orderID 唯一（unique 约束）；重启清空（v1 范围；prod 应替换为 PG）。
type memoryRepo struct {
	mu       sync.RWMutex
	reviews  map[int64]*service.Review // id -> review
	byOrder  map[int64]int64          // orderID -> reviewID（唯一）
	byEscort map[int64][]int64        // escortID -> []reviewID（按时间倒序）
	nextID   int64
}

// newMemoryRepo 构造一个空的 in-memory repo。
func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		reviews:  make(map[int64]*service.Review),
		byOrder:  make(map[int64]int64),
		byEscort: make(map[int64][]int64),
	}
}

// Create 写入新评价；同 orderID 已存在 → ErrDuplicateReview。
func (r *memoryRepo) Create(ctx context.Context, rv *service.Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byOrder[rv.OrderID]; exists {
		return service.ErrDuplicateReview
	}
	rv.ID = atomic.AddInt64(&r.nextID, 1)
	if rv.CreatedAt.IsZero() {
		rv.CreatedAt = time.Now()
	}
	r.reviews[rv.ID] = rv
	r.byOrder[rv.OrderID] = rv.ID
	r.byEscort[rv.EscortID] = append(r.byEscort[rv.EscortID], rv.ID)
	return nil
}

// GetByID 按 ID 取；未命中返回 ErrReviewNotFound。
func (r *memoryRepo) GetByID(ctx context.Context, id int64) (*service.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rv, ok := r.reviews[id]
	if !ok {
		return nil, service.ErrReviewNotFound
	}
	return rv, nil
}

// GetByOrderID 按 orderID 取；未命中返回 ErrReviewNotFound。
func (r *memoryRepo) GetByOrderID(ctx context.Context, orderID int64) (*service.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byOrder[orderID]
	if !ok {
		return nil, service.ErrReviewNotFound
	}
	return r.reviews[id], nil
}

// ListByEscort 返回某 escort 的评价列表（按时间倒序；支持 limit/offset）。
func (r *memoryRepo) ListByEscort(ctx context.Context, escortID int64, limit, offset int) ([]*service.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byEscort[escortID]
	out := make([]*service.Review, 0, len(ids))
	for _, id := range ids {
		if rv, ok := r.reviews[id]; ok {
			out = append(out, rv)
		}
	}
	// 按时间倒序（最新在前）。
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if offset >= len(out) {
		return []*service.Review{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

// List 按过滤条件查询（v1 简化：仅按 escortID / orderID 索引 + MinRating 过滤）。
func (r *memoryRepo) List(ctx context.Context, f service.ListFilter) ([]*service.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*service.Review, 0, len(r.reviews))
	for _, rv := range r.reviews {
		if f.EscortID > 0 && rv.EscortID != f.EscortID {
			continue
		}
		if f.OrderID > 0 && rv.OrderID != f.OrderID {
			continue
		}
		if f.MinRating > 0 && rv.Rating < f.MinRating {
			continue
		}
		out = append(out, rv)
	}
	// 按时间倒序
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	// 简单分页
	page := f.Page
	if page <= 0 {
		page = 1
	}
	size := f.PageSize
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if offset >= len(out) {
		return []*service.Review{}, nil
	}
	end := offset + size
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

// UpdateReply 写入 admin 回复内容。
func (r *memoryRepo) UpdateReply(ctx context.Context, id int64, reply string, adminID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rv, ok := r.reviews[id]
	if !ok {
		return service.ErrReviewNotFound
	}
	if rv.RepliedAt != nil {
		return service.ErrDuplicateReview // 已回复过，复用现有 sentinel
	}
	now := time.Now()
	rv.Reply = reply
	rv.RepliedBy = adminID
	rv.RepliedAt = &now
	return nil
}

// nilPublisher 是 Kafka 缺失时的占位：事件不外发；评价已写入 in-memory。
type nilPublisher struct{}

func (nilPublisher) PublishOrderReviewed(ctx context.Context, ev contracts.OrderReviewedEvent) error {
	return nil
}

// buildPublisher 根据 cfg.Kafka 构造 publisher 与可选 closer。
//
// §32 公共模式：Brokers 空 → *nilPublisher + nil closer；非空 → *kafkapublisher.Publisher + 同对象 closer。
func buildPublisher(cfg *config.Config) (service.Publisher, func() error) {
	if len(cfg.Kafka.Brokers) == 0 {
		logger.L().Warn("review-service: kafka.brokers empty; publisher disabled (fallback to nilPublisher)")
		return nilPublisher{}, nil
	}
	kp, err := kafkapublisher.New(cfg.Kafka.Brokers, contracts.TopicOrderReviewed)
	if err != nil {
		log.Fatalf("review-service: build kafka publisher: %v", err)
	}
	logger.L().Info("review-service: kafka publisher wired",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("topic", contracts.TopicOrderReviewed))
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
	log.Printf("review-service healthz server listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}