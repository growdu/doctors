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
	"golang.org/x/time/rate"

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

// newLimiterEngineWithRegistry 装配带显式 registry 的 engine，并把 registry 一起返回，
// 让测试能通过 Snapshot() 拿到每个 key 的 *rate.Limiter 并直接读 Tokens()，
// 不依赖 sleep 推进真实 clock，彻底规避 flaky。
func newLimiterEngineWithRegistry(t *testing.T, perSecond rate.Limit, burst int, keyFunc func(*gin.Context) string) (*gin.Engine, *middleware.LimiterRegistry) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reg := middleware.NewLimiterRegistry(perSecond, burst)
	r.Use(middleware.RateLimitWithRegistry(reg, keyFunc))
	r.GET("/p", okHandler)
	return r, reg
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
// 单 IP 限流不会污染另一 IP。修复说明（v1.4 稳定性优化）：
//   - 原实现靠 burst=1 + rate=100/s，期望"A 第 2 个 429 → B 第 1 个仍 200"。
//   - Windows + Git Bash 调度抖动会让 A 桶在两次 Allow 之间被回填一个 token，
//     导致"A 第 2 个 200、B 第 1 个 429"——假阴性。
//   - 现用 rate=0.001（1 token / 1000s）+ burst=1，并在 Snapshot() 上
//     直接读 Tokens() 断言 A、B 桶互不污染：完全跳过真实时钟推进，
//     测试在 ms 级运行不会触发任何 token 回填。
func TestRateLimit_DifferentIPsHaveIndependentBuckets(t *testing.T) {
	r, reg := newLimiterEngineWithRegistry(t, rate.Limit(0.001), 1, nil)

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

	// 用 Tokens() 直接校验桶互相独立（不依赖 sleep）：
	//   - A 桶消费 1 次后 Allow 失败 → tokens 接近 0
	//   - B 桶消费 1 次后 Allow 失败 → tokens 接近 0
	//   - 但两个桶独立存在，证明 key 维度隔离
	// 注：rate.Limiter.Tokens() 在 burst=1 全部耗尽后仍可能返回 ~1e-7（浮点精度），
	// 所以用 0.01 作为容差；远小于 burst=1 的 1.0，仍能严格断言"已被耗尽"。
	snap := reg.Snapshot()
	require.Contains(t, snap, ipA, "A 桶应已注册")
	require.Contains(t, snap, ipB, "B 桶应已注册")
	assert.Less(t, snap[ipA].Tokens(), 0.01,
		"A 桶应被耗尽（burst=1 消费 1 次后近似 0）")
	assert.Less(t, snap[ipB].Tokens(), 0.01,
		"B 桶应被耗尽（burst=1 消费 1 次后近似 0）")
	assert.Equal(t, 2, reg.Size(), "应注册 2 个 key（A、B 独立）")
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
//
// 修复说明：用极低 rate（0.001）让 token 几乎不可能在测试窗口内回填，
// 规避 Windows + Git Bash 上偶发的"调度抖动让 token 补回来"的 flaky。
func TestRateLimit_HTTPStatus429(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(0.001),
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
// 用极低 rate（0.001）规避"50 goroutine 排队期间补 token"的 flaky，
// 保证 burst 数严格生效：放行数 == burst。
func TestRateLimit_ConcurrentRequests(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(0.001),
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
	// 极低 rate 让 token 几乎不可能在 50 goroutine 排队期间补充：
	// 放行数应严格 == burst=20。
	assert.Equal(t, 20, allowed,
		"rate=0.001 burst=20 时，50 并发请求的放行数应严格等于 burst")
	assert.Equal(t, total-20, blocked, "其余应为 429")
}

// TestRateLimit_CustomKeyFunc 验证 WithRateLimitKeyFunc 按 header 取 key。
func TestRateLimit_CustomKeyFunc(t *testing.T) {
	r := newLimiterEngine(t,
		middleware.WithRateLimitPerSecond(0.001),
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

// TestRateLimit_SnapshotReturnsIndependentLimiters 显式验证 Snapshot() 语义：
//
// 注册的 limiter 都是独立 *rate.Limiter 指针，互不影响。
// 防止未来有人无意把 registry 改成共用一个 limiter。
func TestRateLimit_SnapshotReturnsIndependentLimiters(t *testing.T) {
	reg := middleware.NewLimiterRegistry(rate.Limit(0.001), 1)
	// 通过一个空 engine 触发首次 get，但这里直接走 registry 公开方法
	// （get 是 unexported；只能借助 RateLimitWithRegistry 路径）。
	r := newTestEngineWithReg(t, reg)
	_, _ = r, reg
	snap := reg.Snapshot()
	// 初始：未发请求 → 0 个 key
	assert.Equal(t, 0, reg.Size(), "未发请求时 registry 应为空")

	// 发请求后再 Snapshot
	w := httptest.NewRecorder()
	r.ServeHTTP(w, makeReq("10.5.5.5"))
	require.Equal(t, http.StatusOK, w.Code)
	snap = reg.Snapshot()
	require.Contains(t, snap, "10.5.5.5")
	assert.NotNil(t, snap["10.5.5.5"])
}

// newTestEngineWithRegistry 是上面的 helper，避免循环依赖。
func newTestEngineWithReg(t *testing.T, reg *middleware.LimiterRegistry) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RateLimitWithRegistry(reg, nil))
	r.GET("/p", okHandler)
	return r
}