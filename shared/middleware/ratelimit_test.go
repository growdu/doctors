package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/middleware"
)

// okHandler 返回 200，测试时仅关心是否"放行"。
func okHandler(c *gin.Context) {
	httpx.OK[any](c, gin.H{"hi": 1})
}

// newLimiterEngine 装配一个挂好 RateLimit 的 gin engine，并把响应解到 body。
func newLimiterEngine(t *testing.T, opts ...middleware.RateLimitOption) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RateLimit(opts...))
	r.GET("/p", okHandler)
	return r
}

// makeReq 构造指定 client IP 的 GET 请求。
func makeReq(ip string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.RemoteAddr = ip + ":12345"
	return req
}

// doAndDecode 跑一次 ServeHTTP，解析 httpx.Resp。
func doAndDecode(t *testing.T, h http.Handler, ip string) httpx.Resp[any] {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeReq(ip))
	var resp httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

// TestRateLimit_BurstAllowsAndBlocksAfterwards 验证 burst 内放行 + 超限 429。
//
// rate=100/s burst=5：连发 7 个请求，前 5 个 200，后 2 个 429。
func TestRateLimit_BurstAllowsAndBlocksAfterwards(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(100),
		middleware.WithRateLimitBurst(5),
	)

	const ip = "10.1.1.1"
	var allowed, blocked int
	for i := 0; i < 7; i++ {
		resp := doAndDecode(t, r, ip)
		switch resp.Code {
		case 0:
			allowed++
		case int(errs.CodeRateLimit):
			blocked++
		}
	}
	assert.Equal(t, 5, allowed, "burst=5 应放行 5 个")
	assert.Equal(t, 2, blocked, "剩余 2 个应为 429")
}

// TestRateLimit_DifferentIPsHaveIndependentBuckets 验证不同 IP 独立计数。
//
// 单 IP 限流不会污染另一 IP。
func TestRateLimit_DifferentIPsHaveIndependentBuckets(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(100),
		middleware.WithRateLimitBurst(1), // 极小 burst 让限流更易触发
	)

	const ipA = "10.1.1.1"
	const ipB = "10.1.1.2"

	// A 已经耗尽：第 1 个 OK，第 2 个 429
	respA1 := doAndDecode(t, r, ipA)
	respA2 := doAndDecode(t, r, ipA)
	assert.Equal(t, 0, respA1.Code, "A-1 通过")
	assert.Equal(t, int(errs.CodeRateLimit), respA2.Code, "A-2 被限流")

	// B 第一次仍能通过 —— 证明 A 没把 B 的桶带走
	respB1 := doAndDecode(t, r, ipB)
	assert.Equal(t, 0, respB1.Code, "B-1 通过（A 被限流不影响 B）")

	// B 第二次也耗尽 → 429
	respB2 := doAndDecode(t, r, ipB)
	assert.Equal(t, int(errs.CodeRateLimit), respB2.Code)
}

// TestRateLimit_RateAndBurstSlowRefill 模拟 rate=0.1/s burst=2：
//
// 前 2 个通过 → 第 3 个 429（短时间未补充）。
//
// 这是文档示例："rate=1/10s burst=2：前 2 个通过，第 3 个 429"。
func TestRateLimit_RateAndBurstSlowRefill(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(0.1), // 1 token / 10s
		middleware.WithRateLimitBurst(2),
	)
	const ip = "10.2.2.2"

	r1 := doAndDecode(t, r, ip)
	r2 := doAndDecode(t, r, ip)
	r3 := doAndDecode(t, r, ip)

	assert.Equal(t, 0, r1.Code, "第 1 个通过（burst=2）")
	assert.Equal(t, 0, r2.Code, "第 2 个通过（burst=2）")
	assert.Equal(t, int(errs.CodeRateLimit), r3.Code,
		"第 3 个 0.1s 内必 429（rate=0.1 ≈ 1 token / 10s）")
}

// TestRateLimit_HTTPStatus429 验证超限时 HTTP 状态码为 429（不是 200）。
func TestRateLimit_HTTPStatus429(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(0.1),
		middleware.WithRateLimitBurst(1),
	)
	const ip = "10.3.3.3"

	// 抢占仅有的 1 个 token
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, makeReq(ip))
	require.Equal(t, http.StatusOK, w1.Code)

	// 第二个请求耗尽桶 → 429
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, makeReq(ip))
	assert.Equal(t, http.StatusTooManyRequests, w2.Code,
		"超限必须返回 HTTP 429")

	// 业务码
	var resp httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeRateLimit), resp.Code,
		"业务码必须为 errs.CodeRateLimit(13001)")
	assert.NotEmpty(t, resp.TraceID, "响应须带 trace_id")
}

// TestRateLimit_ConcurrentRequests 验证并发请求不破坏桶语义。
//
// 50 goroutine × 1 个 IP = 50 个请求；burst=20 → 至少放行 20，其余被限流；
// 由于 100 req/s 在 50 个 goroutine 排队期间可能补充 1~2 个 token，
// 这里只断言"放行数 ≥ burst"和"总数守恒"，避免时间窗口抖动造成的 flaky。
func TestRateLimit_ConcurrentRequests(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(100),
		middleware.WithRateLimitBurst(20),
	)

	const ip = "10.4.4.4"
	const total = 50
	var wg sync.WaitGroup
	codes := make(chan int, total)
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r.ServeHTTP(w, makeReq(ip))
			var resp httpx.Resp[any]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
				codes <- resp.Code
			}
		}()
	}
	wg.Wait()
	close(codes)

	var allowed, blocked int
	for c := range codes {
		switch c {
		case 0:
			allowed++
		case int(errs.CodeRateLimit):
			blocked++
		}
	}
	assert.Equal(t, total, allowed+blocked, "allowed+blocked 必须守恒为 total")
	assert.GreaterOrEqual(t, allowed, 20, "burst=20 → 至少放行 20 个")
	assert.GreaterOrEqual(t, blocked, 1, "必有一部分被限流")
}

// TestRateLimit_CustomKeyFunc 验证 WithRateLimitKeyFunc 按 header 取 key。
func TestRateLimit_CustomKeyFunc(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(100),
		middleware.WithRateLimitBurst(1),
		middleware.WithRateLimitKeyFunc(func(c *gin.Context) string {
			return c.GetHeader("X-Tenant-ID")
		}),
	)

	doReq := func(tenant string) httpx.Resp[any] {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("X-Tenant-ID", tenant)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var resp httpx.Resp[any]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return resp
	}

	// tenant-A 已耗尽 → 第 2 个 429
	assert.Equal(t, 0, doReq("A").Code)
	assert.Equal(t, int(errs.CodeRateLimit), doReq("A").Code)

	// tenant-B 独立
	assert.Equal(t, 0, doReq("B").Code, "B 不应被 A 限流")
}

// 旧日志：避免 time / strconv 未用报错
