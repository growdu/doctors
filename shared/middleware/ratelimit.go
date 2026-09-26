// Package middleware 提供跨服务复用的 IP 限流（rate limit）中间件。
//
// RateLimit 基于 golang.org/x/time/rate 的 token-bucket 实现，
// 每个 key（默认 c.ClientIP()）独立维护一个 *rate.Limiter。
//
// 设计要点：
//   - 每个 IP 一个限流器：恶意/突发流量不会拉低正常用户体验。
//   - 默认 100 req/s + burst 200：适合一般 API；高 QPS 服务可在启动时
//     通过 WithRateLimitPerSecond / WithRateLimitBurst 调整。
//   - 超限返回 HTTP 429 + 业务码 errs.CodeRateLimit（13001），便于前端
//     直接识别并退避。
//   - keyFunc 可注入：例如按 X-Tenant-ID 限流 / 按 uid 限流。
//   - 内存：每个活跃 IP 一个 *rate.Limiter（≈ 80B）。需长期运行的服务
//     可结合反向代理（nginx limit_req）或 LRU 淘汰。生产可观察 keySet 大小，
//     超过阈值时按 LRU 淘汰或改为基于 Redis 的分布式限流。
//
// 使用：
//
//	engine.Use(middleware.RateLimit())                                    // 默认
//	engine.Use(middleware.RateLimit(                                       // 自定义
//	    middleware.WithRateLimitPerSecond(50),
//	    middleware.WithRateLimitBurst(100),
//	    middleware.WithRateLimitKeyFunc(func(c *gin.Context) string {
//	        return c.GetHeader("X-Tenant-ID")
//	    }),
//	))
//
// 测试 / 进阶：
//
//	reg := middleware.NewLimiterRegistry(100, 200)
//	r.Use(middleware.RateLimitWithRegistry(reg, nil))
//	// reg.Snapshot() 拿到 (key, *rate.Limiter) 快照，可直接读 Tokens() 校验
//	// "两个 key 的桶互相独立"，不依赖真实时钟推进。
package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// RateLimitOption 配置 RateLimit。
type RateLimitOption func(*rateLimitConfig)

type rateLimitConfig struct {
	perSecond rate.Limit
	burst     int
	keyFunc   func(*gin.Context) string
}

// WithRateLimitPerSecond 设置每秒补充的 token 数（rate）。
//
// 默认 100.0；如果 ≤ 0 会被钳到 1 防止"零速率永不放行"的边界。
func WithRateLimitPerSecond(r float64) RateLimitOption {
	return func(c *rateLimitConfig) {
		if r > 0 {
			c.perSecond = rate.Limit(r)
		}
	}
}

// WithRateLimitBurst 设置桶容量（burst）。
//
// 默认 200；如果 ≤ 0 会被钳到 1。
func WithRateLimitBurst(b int) RateLimitOption {
	return func(c *rateLimitConfig) {
		if b > 0 {
			c.burst = b
		}
	}
}

// WithRateLimitKeyFunc 自定义限流 key 提取函数。
//
// 未注入时默认 c.ClientIP()，便于按 IP 限流；可改为从 header / token 中提取。
func WithRateLimitKeyFunc(fn func(*gin.Context) string) RateLimitOption {
	return func(c *rateLimitConfig) {
		if fn != nil {
			c.keyFunc = fn
		}
	}
}

// RateLimit 返回 gin.HandlerFunc：每个 key 一个 token-bucket limiter。
//
// 流程：取 key → 取/创建 *rate.Limiter → Allow() →
//   - true：放行（继续 c.Next()）
//   - false：写 429 + 业务码 errs.CodeRateLimit（13001），调用 c.Abort()
//
// 不变量：
//   - 同一 IP 突发 burst 个允许通过；超过 burst 后等待 perSecond token 补充。
//   - 不同 IP 各自独立：限流不会跨 key 污染。
//
// 推荐挂载顺序：Metrics() → Recovery() → RateLimit() → Auth() → business。
// 限流应在鉴权之前，避免对无效请求消耗 token bucket。
func RateLimit(opts ...RateLimitOption) gin.HandlerFunc {
	cfg := rateLimitConfig{
		perSecond: 100,
		burst:     200,
		keyFunc:   func(c *gin.Context) string { return c.ClientIP() },
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	// 防御性钳值：避免 perSecond / burst 被设为 0 时出现"全部 429"或"全部放行"
	if cfg.perSecond <= 0 {
		cfg.perSecond = 1
	}
	if cfg.burst <= 0 {
		cfg.burst = 1
	}

	reg := NewLimiterRegistry(cfg.perSecond, cfg.burst)
	return RateLimitWithRegistry(reg, cfg.keyFunc)
}

// RateLimitWithRegistry 是 RateLimit 的"外部注入 registry"形态。
//
// 主要用途：
//  1. 测试：通过 reg.Snapshot() 直接读 *rate.Limiter.Tokens()，
//     避免依赖真实 sleep 推进 clock 而出现 flaky。
//  2. 高级场景：跨中间件共享同一个 registry（如同一 IP 既出现在限流 A 又出现在限流 B）。
//
// keyFunc 传 nil 时回退为 c.ClientIP()。
func RateLimitWithRegistry(reg *LimiterRegistry, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	if keyFunc == nil {
		keyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}
	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			// key 为空（如未取到 IP）→ 视为单 key 全局限流，避免 bypass
			key = "_"
		}
		if reg.get(key).Allow() {
			c.Next()
			return
		}
		// 超限：写 429 + 业务码 RateLimit（13001）
		c.AbortWithStatusJSON(http.StatusTooManyRequests, httpx.Resp[any]{
			Code:    int(errs.CodeRateLimit),
			Message: "rate limit exceeded",
			Data:    nil,
			TraceID: httpx.TraceID(c),
		})
	}
}

// NewLimiterRegistry 构造 *LimiterRegistry；RateLimit() 内部隐式调用，
// 也可由测试 / 高级用法显式构造后传入 RateLimitWithRegistry。
func NewLimiterRegistry(perSecond rate.Limit, burst int) *LimiterRegistry {
	if perSecond <= 0 {
		perSecond = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &LimiterRegistry{
		limiters:  make(map[string]*rate.Limiter),
		perSecond: perSecond,
		burst:     burst,
	}
}

// LimiterRegistry 按 key 缓存 *rate.Limiter 的并发安全容器。
//
// 实现要点：
//   - map + sync.RWMutex：读多写少（同一 IP 读 limiter，陌生 IP 写一行）
//   - 每个 limiter 独立，互不影响
//   - 生产可加 LRU 淘汰以防止恶意 IP 撑爆内存；本期先实现功能完整性
//
// 公开 Snapshot() 与 Size()，便于测试断言"两个 key 的桶互相独立"——
// 直接通过 Tokens() 读桶剩余 token 数，不依赖 sleep 推进 clock。
type LimiterRegistry struct {
	mu        sync.RWMutex
	limiters  map[string]*rate.Limiter
	perSecond rate.Limit
	burst     int
}

// get 取出或新建指定 key 的 *rate.Limiter。
//
// 读多写少：读路径用 RLock；首次写入退化为 Lock。
func (r *LimiterRegistry) get(key string) *rate.Limiter {
	r.mu.RLock()
	if lim, ok := r.limiters[key]; ok {
		r.mu.RUnlock()
		return lim
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	// 二次检查：可能在 RUnlock→Lock 之间其它 goroutine 已写入
	if lim, ok := r.limiters[key]; ok {
		return lim
	}
	lim := rate.NewLimiter(r.perSecond, r.burst)
	r.limiters[key] = lim
	return lim
}

// Snapshot 返回当前所有 (key, limiter) 的副本（测试用）。
//
// 测试场景常需要断言"两个不同 key 的桶互相独立"——直接读取
// internal map 易触发 data race；Snapshot 在 RLock 下遍历并复制指针，
// 让测试拿到稳定快照后再校验。
func (r *LimiterRegistry) Snapshot() map[string]*rate.Limiter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*rate.Limiter, len(r.limiters))
	for k, v := range r.limiters {
		out[k] = v
	}
	return out
}

// Size 返回当前注册的 key 数（测试 / 监控用）。
func (r *LimiterRegistry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.limiters)
}