// Package health 提供统一的依赖健康检查 + /readyz 端点。
//
// 设计动机：
//   - K8s readinessProbe 需要区分"进程存活"(/healthz，liveness) 与
//     "依赖就绪"(/readyz，readiness)。前者只反映进程没崩溃；后者反映
//     DB / Redis / Kafka 等依赖可达，流量才能安全下发。
//   - 11 个服务都要做这件事，避免每个 service 重新发明轮子。
//
// 设计要点：
//   - Checker 接口：每个外部依赖（DB / Redis / Kafka）实现一个。
//   - Manager 聚合多个 Checker；调用 RunAll 并行 ping，返回 Status 列表。
//   - Status（每个 checker 名字 + err）：用于 JSON 输出 + ReadyzHandler 决策。
//   - ReadyzHandler：所有 checker 通过 → 200；任一失败 → 503 + JSON body。
//   - 适配器：pgxpool / redis.Client / kafka.Brokers 各一个 checker，
//     用 "WithPGPool / WithRedisClient / WithKafkaBrokers" Option 注入。
//
// 使用示例（cmd/main.go）：
//
//	mgr := health.NewManager()
//	if pool != nil {
//	    mgr.Register(health.NewPGPoolChecker("postgres-main", pool, time.Second))
//	}
//	if rdb != nil {
//	    mgr.Register(health.NewRedisChecker("redis-main", rdb, time.Second))
//	}
//	if len(cfg.Kafka.Brokers) > 0 {
//	    mgr.Register(health.NewKafkaBrokerChecker("kafka-brokers", cfg.Kafka.Brokers, time.Second))
//	}
//	r.GET("/readyz", health.ReadyzHandler(mgr))
package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Checker 是单个依赖健康检查的最小契约。
//
// 实现要点：
//   - Name 必须稳定（用于 /readyz JSON 输出 + 监控 label），同一资源永远返回同一字符串。
//   - Check 在 ctx 内完成；超时由调用方控制（Manager.RunAll 注入统一 timeout）。
//   - 返回 nil → 健康；非 nil → 不健康（error message 进 JSON body）。
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckFunc 是 Checker 的函数式适配器（便于写 inline checker）。
//
// 使用：
//
//	mgr.Register(health.CheckFunc{
//	    NameFn: func() string { return "custom" },
//	    CheckFn: func(ctx context.Context) error { ... },
//	})
type CheckFunc struct {
	NameFn  func() string
	CheckFn func(ctx context.Context) error
}

func (c CheckFunc) Name() string                    { return c.NameFn() }
func (c CheckFunc) Check(ctx context.Context) error { return c.CheckFn(ctx) }

// Status 是单个 checker 的运行结果快照。
type Status struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency"` // "1.2ms" 形式；便于排查慢依赖
}

// Report 是 /readyz 整体响应。
//
// Healthy 字段由 Manager.RunAll 聚合（任一 checker 失败 → false）。
type Report struct {
	Healthy bool     `json:"healthy"`
	Checks  []Status `json:"checks"`
}

// Manager 是 Checker 容器 + RunAll 调度器。
//
// 实现要点：
//   - 注册顺序决定 Checks 输出顺序（便于运维对照 yaml 配置）。
//   - RunAll 用 errgroup 模式（fan-out + 第一个 ctx 控制），单个 checker
//     超时不会拖垮其他 checker。
//   - 默认 timeout 1s：与 K8s readinessProbe.timeoutSeconds 默认一致；
//     调用方可传入 WithTimeout Option 覆盖。
type Manager struct {
	mu       sync.RWMutex
	checkers []Checker
	timeout  time.Duration
}

// Option 配置 Manager 行为。
type Option func(*Manager)

// WithTimeout 设置 RunAll 的统一超时（默认 1s）。
func WithTimeout(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.timeout = d
		}
	}
}

// NewManager 构造空的 Manager。
func NewManager(opts ...Option) *Manager {
	m := &Manager{timeout: time.Second}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Register 注册一个 Checker（顺序敏感，输出按注册顺序）。
//
// 重复注册同名 checker：保留最先注册的，后注册的会被跳过并返回 false；
// 这样避免上游误配（同名两 DB 实例）导致覆盖式隐藏问题。
func (m *Manager) Register(c Checker) bool {
	if c == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.checkers {
		if existing.Name() == c.Name() {
			return false
		}
	}
	m.checkers = append(m.checkers, c)
	return true
}

// MustRegister 是 Register 的 panic 版本；常用于 main 启动期装配（fail-fast）。
func (m *Manager) MustRegister(c Checker) {
	if !m.Register(c) {
		panic(fmt.Sprintf("health: duplicate or nil checker %q", c.Name()))
	}
}

// Len 返回已注册 checker 数（测试用）。
func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.checkers)
}

// RunAll 并发执行所有 checker 的 Check，timeout 由 Manager.timeout 控制。
//
// 返回 Report：Healthy = 所有 checker 都 OK；Checks 按注册顺序输出。
// 单个 checker 失败不会中断其他 checker；超时归因到超时的那个。
func (m *Manager) RunAll(ctx context.Context) Report {
	m.mu.RLock()
	checkers := make([]Checker, len(m.checkers))
	copy(checkers, m.checkers)
	m.mu.RUnlock()

	results := make([]Status, len(checkers))
	var wg sync.WaitGroup
	for i, c := range checkers {
		wg.Add(1)
		go func(i int, c Checker) {
			defer wg.Done()
			results[i] = runOne(ctx, c, m.timeout)
		}(i, c)
	}
	wg.Wait()

	healthy := true
	for _, r := range results {
		if !r.OK {
			healthy = false
			break
		}
	}
	return Report{Healthy: healthy, Checks: results}
}

// runOne 执行单个 checker 并收集 latency / error。
func runOne(parent context.Context, c Checker, timeout time.Duration) Status {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	start := time.Now()
	err := c.Check(ctx)
	latency := time.Since(start)

	s := Status{
		Name:    c.Name(),
		OK:      err == nil,
		Latency: latency.String(),
	}
	if err != nil {
		s.Error = err.Error()
	}
	return s
}

// ErrCheckTimeout 标记 checker 自身超时（与 ctx.Cancel 区分）。
//
// 调用方在 Check() 实现里建议：
//
//	if ctx.Err() == context.DeadlineExceeded { return ErrCheckTimeout }
var ErrCheckTimeout = errors.New("health: check timeout")