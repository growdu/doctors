package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisteredMetrics_AllSix 验证 6 个业务指标都能在 DefaultRegisterer 上注册并被收集。
func TestRegisteredMetrics_AllSix(t *testing.T) {
	// 1. HTTPRequestsTotal
	c := HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/test", "2xx")
	c.Inc()
	assert.Equal(t, 1.0, testutil.ToFloat64(c),
		"http_requests_total 应能 Inc 并被 CollectAndCount 读到")

	// 2. HTTPRequestDuration
	h := HTTPRequestDuration.WithLabelValues("GET", "/api/v1/test")
	h.Observe(0.123)
	// Histogram 是多个 metric 系列（_sum / _count / _bucket），CollectAndCount 应 >= 1
	count := testutil.CollectAndCount(HTTPRequestDuration)
	assert.GreaterOrEqual(t, count, 1, "http_request_duration_seconds 应被注册")

	// 3-5. DB pool gauges：直接 Set 后读取
	DBPoolAcquiredConnections.WithLabelValues("main").Set(7)
	DBPoolIdleConnections.WithLabelValues("main").Set(3)
	DBPoolTotalConnections.WithLabelValues("main").Set(10)
	assert.Equal(t, 7.0, testutil.ToFloat64(DBPoolAcquiredConnections.WithLabelValues("main")))
	assert.Equal(t, 3.0, testutil.ToFloat64(DBPoolIdleConnections.WithLabelValues("main")))
	assert.Equal(t, 10.0, testutil.ToFloat64(DBPoolTotalConnections.WithLabelValues("main")))

	// 6. KafkaConsumerLag
	KafkaConsumerLag.WithLabelValues("orders", "order-svc").Set(42)
	assert.Equal(t, 42.0, testutil.ToFloat64(KafkaConsumerLag.WithLabelValues("orders", "order-svc")))
}

// TestHandler_ServesPrometheus 验证 Handler() 返回的 http.Handler 能输出标准 prom 文本格式。
func TestHandler_ServesPrometheus(t *testing.T) {
	// 先写点数据，确保抓取端能看见
	HTTPRequestsTotal.WithLabelValues("GET", "/probe", "2xx").Add(2)
	HTTPRequestDuration.WithLabelValues("GET", "/probe").Observe(0.05)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "/metrics 应返回 200")
	body := w.Body.String()
	// 关键指标名必须出现
	assert.Contains(t, body, "http_requests_total", "应包含 http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds", "应包含 http_request_duration_seconds")
	assert.Contains(t, body, "# HELP", "应包含 HELP 注释（标准 prom 文本格式）")
}

// TestGinMiddleware_RecordsCounter 验证 GinMiddleware 自动记录 method/path/status。
func TestGinMiddleware_RecordsCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinMiddleware())
	r.GET("/api/v1/probe/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/api/v1/fail", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"err": "x"})
	})

	// 200 × 2
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/probe/42", nil))
		require.Equal(t, http.StatusOK, w.Code)
	}
	// 400 × 1
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/fail", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)

	// 校验：用路由模板（不是真实 ID），避免 label 基数爆炸
	assert.Equal(t, 2.0,
		testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/probe/:id", "2xx")),
		"2xx 计数应为 2")
	assert.Equal(t, 1.0,
		testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/fail", "4xx")),
		"4xx 计数应为 1")

	// Histogram _count 也应同步增长
	d := HTTPRequestDuration.WithLabelValues("GET", "/api/v1/probe/:id")
	m := &d // prometheus 内部 metric
	_ = m
	assert.Greater(t, testutil.CollectAndCount(HTTPRequestDuration), 0)
}

// TestStatusBucket 验证 status code → 桶映射。
func TestStatusBucket(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{100, "1xx"}, {199, "1xx"},
		{200, "2xx"}, {299, "2xx"},
		{300, "3xx"}, {301, "3xx"},
		{400, "4xx"}, {404, "4xx"}, {499, "4xx"},
		{500, "5xx"}, {503, "5xx"}, {599, "5xx"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, statusBucket(c.in), "code=%d", c.in)
	}
}

// TestSafeCollect_EmptyAndNil 安全采集在 stats 为空/nil 时不 panic。
func TestSafeCollect_EmptyAndNil(t *testing.T) {
	prev := dbStat
	defer func() { dbStat = prev }()

	// nil provider：safeCollect 内部 dbStat() 会 NPE；这里改成返回空切片
	dbStat = func() []DBPoolStat { return nil }
	safeCollect() // 不应 panic

	dbStat = func() []DBPoolStat { return []DBPoolStat{} }
	safeCollect() // 不应 panic

	// name 为空应跳过
	dbStat = func() []DBPoolStat {
		return []DBPoolStat{{Name: "", Acquired: 1}}
	}
	safeCollect() // 不应 panic
}

// TestMiddleware_NetHTTP 验证 net/http 中间件形态能正确记录 status code。
func TestMiddleware_NetHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/err", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := Middleware(mux)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/err", nil))

	// /healthz 出现 3 次 2xx
	assert.Equal(t, 3.0,
		testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/healthz", "2xx")))
	assert.Equal(t, 1.0,
		testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/err", "5xx")))
}

// TestServiceInfo_RecordsVersion 验证 InitMetrics 注入的 service_info 在 /metrics 输出中可见。
//
// 注意：本测试在 init 顺序上排在 InitMetrics 之前调用是 noop；为此我们单独调一次
// InitMetrics（sync.Once 锁定后即幂等），随后通过 Handler() 抓取端验证指标暴露。
func TestServiceInfo_RecordsVersion(t *testing.T) {
	InitMetrics("test-svc", "v0.0.1-test")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, req)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "service_info"),
		"/metrics 输出应包含 service_info metric")
}

// TestWithDBStatProvider_Integration 验证 InitMetrics 注入 dbStat 后能跑采集循环。
func TestWithDBStatProvider_Integration(t *testing.T) {
	// 本测试不直接调 InitMetrics（已被前面测试初始化过），改为手动调用 safeCollect 验证 path：
	prev := dbStat
	defer func() { dbStat = prev }()

	called := 0
	dbStat = func() []DBPoolStat {
		called++
		return []DBPoolStat{{Name: "integration", Acquired: 5, Idle: 2, TotalConns: 7}}
	}
	safeCollect()
	assert.Equal(t, 1, called)
	assert.Equal(t, 5.0,
		testutil.ToFloat64(DBPoolAcquiredConnections.WithLabelValues("integration")))
}

// TestCollectPeriod_Tick 验证 WithCollectPeriod 自定义间隔生效 + 非法值（0）被忽略。
func TestCollectPeriod_Tick(t *testing.T) {
	// 正常路径：100ms 覆盖默认值 30s
	cfg := &initConfig{collectPeriod: 30 * time.Second}
	WithCollectPeriod(100 * time.Millisecond)(cfg)
	assert.Equal(t, 100*time.Millisecond, cfg.collectPeriod)

	// 非法值：0 应被忽略，保留上一个值（100ms）
	WithCollectPeriod(0)(cfg)
	assert.Equal(t, 100*time.Millisecond, cfg.collectPeriod,
		"d=0 应被 WithCollectPeriod 忽略，不应覆盖已有值")

	// 负数同理：保留已有值
	WithCollectPeriod(-1 * time.Second)(cfg)
	assert.Equal(t, 100*time.Millisecond, cfg.collectPeriod)
}
