// Package metrics 提供 Prometheus 业务指标统一接入。
//
// 设计要点：
//   - 全局默认 Registry：业务自定义指标用 promauto 自动注册到 prometheus.DefaultRegisterer，
//     与 promhttp.Handler() 抓取端天然兼容，11 个服务无需额外注入。
//   - 6 个默认业务指标：HTTP 流量 / DB 连接池 / Kafka 消费滞后 / 服务自身元信息。
//   - InitMetrics(serviceName, version)：注入 service_info{version, go_version} 静态指标，
//     并启动 goroutine 每 30s 收集一次 DB pool 指标（StatProvider 注入式，nil 时跳过）。
//   - Middleware / GinMiddleware：自动记录 http_requests_total 与 http_request_duration_seconds。
//   - Handler()：直接返回 promhttp.Handler()，挂到 /metrics 由 Prometheus 抓取。
//
// 使用示例（main.go）：
//
//	metrics.InitMetrics("auth-service", cfg.ServiceVersion,
//	    metrics.WithDBStatProvider(func() []metrics.DBPoolStat{
//	        s := pool.Stat()
//	        return []metrics.DBPoolStat{{Name: "main", Acquired: s.AcquiredConns(), Idle: s.IdleConns(), TotalConns: s.TotalConns()}}
//	    }),
//	)
//	engine := gin.New()
//	engine.Use(metrics.GinMiddleware())
//	engine.GET("/metrics", gin.WrapH(metrics.Handler()))
package metrics

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPRequestsTotal 累计 HTTP 请求数（按 method/path/status 分桶）。
//
// 业务可读：rate(http_requests_total[5m]) QPS、按 status 维度拆分错误率。
var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests processed, labeled by method, path and status code.",
	},
	[]string{"method", "path", "status"},
)

// HTTPRequestDuration HTTP 请求耗时（method/path）。
//
// buckets 覆盖 5ms~5s：覆盖内网 RPC 典型 p99（< 250ms）和慢请求 (>1s)。
// P95/P99 计算：histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))
var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds, labeled by method and path.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
	},
	[]string{"method", "path"},
)

// DBPoolAcquiredConnections pgxpool / sql.DB 当前占用连接数（按 pool 名）。
var DBPoolAcquiredConnections = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "db_pool_acquired_connections",
		Help: "Number of acquired (in-use) connections in the pool.",
	},
	[]string{"pool"},
)

// DBPoolIdleConnections 池中空闲连接数。
var DBPoolIdleConnections = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "db_pool_idle_connections",
		Help: "Number of idle connections in the pool.",
	},
	[]string{"pool"},
)

// DBPoolTotalConnections 池总连接数（acquired + idle）。
var DBPoolTotalConnections = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "db_pool_total_connections",
		Help: "Total number of connections in the pool (acquired + idle).",
	},
	[]string{"pool"},
)

// KafkaConsumerLag Kafka 消费者滞后（topic + group）：未消费 offset 与最新 offset 之差。
var KafkaConsumerLag = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "kafka_consumer_lag",
		Help: "Kafka consumer lag (high watermark - committed offset), labeled by topic and group.",
	},
	[]string{"topic", "group"},
)

// RecoveryPanicsTotal panic 恢复计数（按 path 维度）。
//
// 由 shared/middleware.Recovery 在捕获到 panic 时 Inc。
// 对应 deploy/prometheus/alerts/panic_recovery.yaml 的 4 条告警规则
// （任意 panic → critical；同端点反复 panic；panic 速率 > 5/min；进程重启）。
//
// 标签仅 path（路由模板，如 "/api/v1/users/:id"），避免 label 基数爆炸。
var RecoveryPanicsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "recovery_panics_total",
		Help: "Total number of panics recovered by middleware.Recovery, labeled by request path.",
	},
	[]string{"path"},
)

// ReadyzCheckTotal /readyz 端点调用次数（按 check 名字 + 单次 ok/fail 状态）。
//
// 由 shared/health.ReadyzHandler 在入口（status="request"，无论 readyz 整体结果）
// + 每个 checker 调用后（status="ok" / "fail"）Inc。
//
// 标签：
//   - check：checker 名字（如 "postgres-main" / "kafka-brokers"）；请求级 Inc 时为
//     "_total"（避免误读为某个具体依赖）。
//   - status："ok" / "fail" / "request" 三选一。
//
// 对应 deploy/prometheus/alerts/general.yaml 的 ReadyzCheckFailure 告警
// （status="fail" 持续 1 分钟即触发 critical）。
var ReadyzCheckTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "readyz_check_total",
		Help: "Total number of /readyz invocations, labeled by check name and status (ok/fail/request).",
	},
	[]string{"check", "status"},
)

// DBPoolStat 是单条 pool 指标的快照；由 StatProvider 批量返回。
type DBPoolStat struct {
	Name       string // pool 标识（如 "main" / "wallet"）
	Acquired   int32  // 当前占用连接数
	Idle       int32  // 当前空闲连接数
	TotalConns int32  // 总连接数 = Acquired + Idle
}

// StatProvider 暴露 DB pool 指标采集函数；main 启动时注入。
// 30s 一次的 collect 协程会调用它；返回 nil/empty 表示跳过（未接 DB）。
type StatProvider func() []DBPoolStat

var (
	initOnce sync.Once
	dbStat   StatProvider
)

// InitOption 允许自定义 InitMetrics 行为。
type InitOption func(*initConfig)

type initConfig struct {
	dbStat        StatProvider
	collectPeriod time.Duration
}

// WithDBStatProvider 注入 DB pool 指标采集函数。
// 示例：func() []DBPoolStat { s := pool.Stat(); return []DBPoolStat{{Name:"main", Acquired:s.AcquiredConns(), ...}} }
func WithDBStatProvider(p StatProvider) InitOption {
	return func(c *initConfig) { c.dbStat = p }
}

// WithCollectPeriod 自定义 DB pool 采集间隔（默认 30s）。
func WithCollectPeriod(d time.Duration) InitOption {
	return func(c *initConfig) {
		if d > 0 {
			c.collectPeriod = d
		}
	}
}

// InitMetrics 初始化 service_info + 启动 DB pool 收集 goroutine。
//
//   - serviceName：注入到 service_info{service="<name>"} 标签（与 OTel resource 一致）。
//   - version：服务构建版本（来自 cfg.ServiceVersion，默认 "dev"）。
//
// 重复调用幂等（sync.Once）：第二次 noop；方便测试。
func InitMetrics(serviceName, version string, opts ...InitOption) {
	initOnce.Do(func() {
		cfg := &initConfig{collectPeriod: 30 * time.Second}
		for _, o := range opts {
			o(cfg)
		}
		dbStat = cfg.dbStat

		// service_info：单时间序列 Gauge，值恒为 1，标签为静态元信息。
		// Prometheus 习惯用 gauge=1 暴露服务 build metadata，便于 alertmanager 关联。
		promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "service_info",
				Help: "Service build information; value is always 1, labels carry static metadata.",
			},
			[]string{"service", "version", "go_version"},
		).With(prometheus.Labels{
			"service":    serviceName,
			"version":    version,
			"go_version": runtime.Version(),
		}).Set(1)

		// DB pool 采集 goroutine：dbStat 为 nil 时不启动。
		if dbStat != nil {
			go runDBStatCollector(cfg.collectPeriod)
		}
	})
}

// runDBStatCollector 每 period 调用 dbStat() 并写入三个 gauge。
//
// 出错（panic）由 recover 兜底：单次采集失败不应杀掉整个采集协程。
func runDBStatCollector(period time.Duration) {
	t := time.NewTicker(period)
	defer t.Stop()
	// 启动后立即采一次，避免首次 /metrics 拉取拿到空值。
	safeCollect()
	for range t.C {
		safeCollect()
	}
}

func safeCollect() {
	defer func() { _ = recover() }()
	stats := dbStat()
	for _, s := range stats {
		if s.Name == "" {
			continue
		}
		DBPoolAcquiredConnections.WithLabelValues(s.Name).Set(float64(s.Acquired))
		DBPoolIdleConnections.WithLabelValues(s.Name).Set(float64(s.Idle))
		DBPoolTotalConnections.WithLabelValues(s.Name).Set(float64(s.TotalConns))
	}
}

// Handler 返回 Prometheus 抓取端 http.Handler。
// 直接挂到 gin 路由：r.GET("/metrics", gin.WrapH(metrics.Handler()))。
func Handler() http.Handler {
	return promhttp.Handler()
}

// GinMiddleware 是 Gin 适配的中间件。
//
// 推荐在 gin.Engine 上 Use（早于业务 middleware），以保证 4xx/5xx 也被统计。
// path 取 FullPath() 模板（如 "/api/v1/users/:id"），避免高基数爆炸；
// 真实 ID 在 label 聚合后被屏蔽，但能保证 Prometheus label 集合稳定可控。
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := statusBucket(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}

// Middleware 是 net/http 形态的中间件（兜底，非 gin 框架也能用）。
//
// path 只能用 Request.URL.Path（无路由模板），基数会比 Gin 版本大；
// gin 项目请优先用 GinMiddleware。
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		method := r.Method
		path := r.URL.Path
		status := statusBucket(rw.status)

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	})
}

// statusRecorder 捕获下游写入的 HTTP status code（默认 200）。
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	return r.ResponseWriter.Write(b)
}

// statusBucket 把 int status code 归到 1xx/2xx/3xx/4xx/5xx 桶：
// 减少 label 基数，便于告警规则聚合（rate(...{status="5xx"})）。
func statusBucket(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
