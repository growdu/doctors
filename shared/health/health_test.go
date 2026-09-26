package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/metrics"
)

// staticCheck 是测试 helper：返回一个固定 result 的 Checker。
type staticCheck struct {
	name string
	err  error
}

func (s staticCheck) Name() string                    { return s.name }
func (s staticCheck) Check(_ context.Context) error   { return s.err }

// =============================================================================
// Manager / RunAll
// =============================================================================

// TestManager_EmptyRunAll 验证：未注册任何 checker 时 RunAll 直接返回 healthy=true。
func TestManager_EmptyRunAll(t *testing.T) {
	m := health.NewManager()
	rep := m.RunAll(context.Background())
	assert.True(t, rep.Healthy, "空 manager 应 healthy=true")
	assert.Empty(t, rep.Checks, "空 manager 应返回空 checks")
	assert.Equal(t, 0, m.Len())
}

// TestManager_RegisterAndRunAll 验证：注册的 checker 都跑一次，结果按注册顺序输出。
func TestManager_RegisterAndRunAll(t *testing.T) {
	m := health.NewManager()
	m.MustRegister(staticCheck{name: "a", err: nil})
	m.MustRegister(staticCheck{name: "b", err: errors.New("boom")})
	m.MustRegister(staticCheck{name: "c", err: nil})
	rep := m.RunAll(context.Background())
	require.Equal(t, 3, len(rep.Checks))
	assert.Equal(t, []string{"a", "b", "c"}, []string{rep.Checks[0].Name, rep.Checks[1].Name, rep.Checks[2].Name})
	assert.True(t, rep.Checks[0].OK)
	assert.False(t, rep.Checks[1].OK)
	assert.Contains(t, rep.Checks[1].Error, "boom")
	assert.True(t, rep.Checks[2].OK)
	assert.False(t, rep.Healthy, "任一失败 → healthy=false")
}

// TestManager_DuplicateRegisterRejected 验证：同名 checker 第二次注册被拒绝。
func TestManager_DuplicateRegisterRejected(t *testing.T) {
	m := health.NewManager()
	assert.True(t, m.Register(staticCheck{name: "dup", err: nil}))
	assert.False(t, m.Register(staticCheck{name: "dup", err: nil}), "重复名应被拒绝")
	assert.Equal(t, 1, m.Len())
}

// TestManager_RegisterNilIgnored 验证：nil checker 被忽略（避免 panic）。
func TestManager_RegisterNilIgnored(t *testing.T) {
	m := health.NewManager()
	assert.False(t, m.Register(nil))
	assert.Equal(t, 0, m.Len())
}

// TestManager_RunAllTimeout 验证：单个 checker 慢于 timeout 时返回 ErrCheckTimeout 风格
// 的失败；其他 checker 不被拖垮。
func TestManager_RunAllTimeout(t *testing.T) {
	m := health.NewManager(health.WithTimeout(50 * time.Millisecond))
	m.MustRegister(staticCheck{name: "fast", err: nil})
	m.MustRegister(health.CheckFunc{
		NameFn: func() string { return "slow" },
		CheckFn: func(ctx context.Context) error {
			select {
			case <-time.After(500 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	rep := m.RunAll(context.Background())
	require.Equal(t, 2, len(rep.Checks))
	assert.True(t, rep.Checks[0].OK, "fast 应通过")
	assert.False(t, rep.Checks[1].OK, "slow 应超时失败")
	assert.False(t, rep.Healthy)
}

// TestManager_ParallelExecution 验证：多个 checker 并发执行（wall time < sum of sleep）。
func TestManager_ParallelExecution(t *testing.T) {
	m := health.NewManager()
	for i := 0; i < 5; i++ {
		i := i
		m.MustRegister(health.CheckFunc{
			NameFn:  func() string { return fmt.Sprintf("c%d", i) },
			CheckFn: func(ctx context.Context) error { time.Sleep(80 * time.Millisecond); return nil },
		})
	}
	start := time.Now()
	rep := m.RunAll(context.Background())
	elapsed := time.Since(start)
	assert.True(t, rep.Healthy)
	// 并发：5 × 80ms ≈ 80ms（不是 400ms）。给 300ms 容差。
	assert.Less(t, elapsed.Milliseconds(), int64(300),
		"5 个 checker 应并发执行，总时长 < 300ms")
}

// =============================================================================
// ReadyzHandler
// =============================================================================

// TestReadyzHandler_AllOK 验证全部 OK → 200。
func TestReadyzHandler_AllOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := health.NewManager()
	m.MustRegister(staticCheck{name: "ok1", err: nil})
	m.MustRegister(staticCheck{name: "ok2", err: nil})
	r := gin.New()
	r.GET("/readyz", health.ReadyzHandler(m))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var rep health.Report
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rep))
	assert.True(t, rep.Healthy)
	require.Equal(t, 2, len(rep.Checks))
}

// TestReadyzHandler_AnyFailure 验证任一失败 → 503。
func TestReadyzHandler_AnyFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := health.NewManager()
	m.MustRegister(staticCheck{name: "ok", err: nil})
	m.MustRegister(staticCheck{name: "broken", err: errors.New("connection refused")})
	r := gin.New()
	r.GET("/readyz", health.ReadyzHandler(m))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code, "任一失败应 503")
	var rep health.Report
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rep))
	assert.False(t, rep.Healthy)
	broken := rep.Checks[1]
	assert.False(t, broken.OK)
	assert.Contains(t, broken.Error, "connection refused")
}

// TestReadyzHandler_NilManager 验证：nil manager → fail-closed（永远 503）。
func TestReadyzHandler_NilManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", health.ReadyzHandler(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"nil manager 必须 fail-closed，避免'忘装配就放行'")
}

// =============================================================================
// Adapters
// =============================================================================

// fakePGPool 是 pgxpool.Pool.Ping 的替身：实现 health.pgPoolPinger 接口。
type fakePGPool struct {
	pingErr   error
	pingCalls int32
}

func (f *fakePGPool) Ping(_ context.Context) error {
	atomic.AddInt32(&f.pingCalls, 1)
	return f.pingErr
}

// fakeRedisClient 是 redis.Client 的替身：实现 health.redisClientPinger 接口。
type fakeRedisClient struct {
	err    error
	calls  int32
}

func (f *fakeRedisClient) Ping(_ context.Context) *redis.StatusCmd {
	atomic.AddInt32(&f.calls, 1)
	cmd := redis.NewStatusCmd(context.Background())
	if f.err != nil {
		cmd.SetErr(f.err)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

// TestNewPGPoolChecker_OK 验证：真实 OK 路径。
func TestNewPGPoolChecker_OK(t *testing.T) {
	pool := &fakePGPool{pingErr: nil}
	c := health.NewPGPoolChecker("pg", pool, time.Second)
	assert.Equal(t, "pg", c.Name())
	require.NoError(t, c.Check(context.Background()))
	assert.Equal(t, int32(1), atomic.LoadInt32(&pool.pingCalls))
}

// TestNewPGPoolChecker_Fail 验证：Ping 返回 error → Check 也返回 error。
func TestNewPGPoolChecker_Fail(t *testing.T) {
	pool := &fakePGPool{pingErr: errors.New("pg down")}
	c := health.NewPGPoolChecker("pg", pool, time.Second)
	err := c.Check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pg down")
}

// TestNewPGPoolChecker_NilPool 验证：pool == nil → 返回 resourceNilError。
func TestNewPGPoolChecker_NilPool(t *testing.T) {
	c := health.NewPGPoolChecker("pg", nil, time.Second)
	err := c.Check(context.Background())
	require.Error(t, err)
	assert.True(t, health.IsResourceNil(err), "nil pool 应返回 resourceNilError")
}

// TestNewRedisChecker_OK / Fail 验证 Redis 适配器两个分支。
func TestNewRedisChecker_OK(t *testing.T) {
	rdb := &fakeRedisClient{err: nil}
	c := health.NewRedisChecker("redis", rdb, time.Second)
	require.NoError(t, c.Check(context.Background()))
}

func TestNewRedisChecker_Fail(t *testing.T) {
	rdb := &fakeRedisClient{err: errors.New("redis down")}
	c := health.NewRedisChecker("redis", rdb, time.Second)
	err := c.Check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis down")
}

// fakeDialer 是 kafka TCP dial 的替身：通过 map[addr]err 注入结果。
type fakeDialer struct {
	addrs map[string]error
	calls int32
}

func (d *fakeDialer) DialContext(_ context.Context, _, addr string) (net.Conn, error) {
	atomic.AddInt32(&d.calls, 1)
	if err, ok := d.addrs[addr]; ok {
		return nil, err
	}
	return nil, errors.New("no entry for addr " + addr)
}

// TestNewKafkaBrokerChecker_OneOK 验证：任一 broker 连通即 OK。
func TestNewKafkaBrokerChecker_OneOK(t *testing.T) {
	d := &fakeDialer{addrs: map[string]error{
		"broker-a:9092": errors.New("conn refused"),
		"broker-b:9092": nil, // 故意没在 map 里覆盖——会落入默认 fake 路径；改用 listen
	}}
	d.addrs["broker-b:9092"] = nil
	// 启一个 TCP listener，模拟 broker-b 可达
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	addr := ln.Addr().String()
	// 用真 dialer 替换 fake，便于连真 listener
	c := health.NewKafkaBrokerChecker("kafka", []string{"127.0.0.1:1", addr}, 500*time.Millisecond)
	require.NoError(t, c.Check(context.Background()), "任一 broker 可达即 OK")
	_ = d
}

// TestNewKafkaBrokerChecker_AllFail 验证：所有 broker 都失败 → 返回最后一个 error。
func TestNewKafkaBrokerChecker_AllFail(t *testing.T) {
	c := health.NewKafkaBrokerChecker("kafka",
		[]string{"127.0.0.1:1", "127.0.0.1:2"}, 200*time.Millisecond)
	err := c.Check(context.Background())
	require.Error(t, err, "全失败应返回 error")
	assert.NotContains(t, err.Error(), "health:", "应是 dial error，不是内部包装")
}

// TestNewKafkaBrokerChecker_EmptyBrokers 验证：brokers 为空 → resourceNil。
func TestNewKafkaBrokerChecker_EmptyBrokers(t *testing.T) {
	c := health.NewKafkaBrokerChecker("kafka", nil, time.Second)
	err := c.Check(context.Background())
	require.Error(t, err)
	assert.True(t, health.IsResourceNil(err))
}

// =============================================================================
// CheckFunc adapter
// =============================================================================

// TestCheckFunc_Adapter 验证：CheckFunc 满足 Checker 接口（compile-time + 行为）。
func TestCheckFunc_Adapter(t *testing.T) {
	called := 0
	c := health.CheckFunc{
		NameFn: func() string { return "inline" },
		CheckFn: func(_ context.Context) error {
			called++
			return nil
		},
	}
	var iface health.Checker = c
	assert.Equal(t, "inline", iface.Name())
	require.NoError(t, iface.Check(context.Background()))
	assert.Equal(t, 1, called)
}

// =============================================================================
// ReadyzHandler / Prometheus counter（readyz_check_total）
// =============================================================================

// readCounter 读 readyz_check_total{check,status} 当前值；label 不存在 → 0。
func readCounter(t *testing.T, check, status string) float64 {
	t.Helper()
	return testutil.ToFloat64(metrics.ReadyzCheckTotal.WithLabelValues(check, status))
}

// TestReadyzHandler_IncrementsCounter 验证：每次 /readyz 调用 → readyz_check_total
// 在请求级（_request/status=request）+ 每个 checker 维度各 +1。
//
// 断言路径：
//  1. 启动前快照 _request / ok / fail 计数
//  2. 一次请求（1 个 ok checker + 1 个 fail checker）
//  3. 断言 _request=request +1；ok checker 维度的 status=ok +1；fail checker 维度的 status=fail +1
func TestReadyzHandler_IncrementsCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := health.NewManager()
	m.MustRegister(staticCheck{name: "pg-main", err: nil})
	m.MustRegister(staticCheck{name: "redis-main", err: errors.New("redis down")})

	beforeRequest := readCounter(t, "_request", "request")
	beforeOK := readCounter(t, "pg-main", "ok")
	beforeFail := readCounter(t, "redis-main", "fail")

	r := gin.New()
	r.GET("/readyz", health.ReadyzHandler(m))

	// 触发 /readyz 一次（应 503，因为 redis-main fail）。
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"任一 checker fail → 503")

	// 1 次请求 → _request/request +1。
	assert.Equal(t, beforeRequest+1, readCounter(t, "_request", "request"),
		"1 次 /readyz 调用 → readyz_check_total{check=_request,status=request} +1")
	// ok checker 维度 +1。
	assert.Equal(t, beforeOK+1, readCounter(t, "pg-main", "ok"),
		"ok checker → readyz_check_total{check=pg-main,status=ok} +1")
	// fail checker 维度 +1。
	assert.Equal(t, beforeFail+1, readCounter(t, "redis-main", "fail"),
		"fail checker → readyz_check_total{check=redis-main,status=fail} +1")
}

// TestReadyzHandler_NilManagerIncrementsFail 验证：nil manager 走 fail-closed 分支，
// 同时也 Inc 计数器（check=_manager, status=fail）——便于告警识别"忘了装配"事故。
func TestReadyzHandler_NilManagerIncrementsFail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	beforeReq := readCounter(t, "_request", "request")
	beforeMgr := readCounter(t, "_manager", "fail")

	r := gin.New()
	r.GET("/readyz", health.ReadyzHandler(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	assert.Equal(t, beforeReq+1, readCounter(t, "_request", "request"),
		"nil manager 也算一次 /readyz 调用 → _request/request +1")
	assert.Equal(t, beforeMgr+1, readCounter(t, "_manager", "fail"),
		"nil manager 走 fail-closed 分支 → _manager/fail +1")
}

// TestReadyzHandler_AllOK_NoFailCounter 验证：全 OK 路径不应让任何 fail 计数器增长
// （避免误报）。
func TestReadyzHandler_AllOK_NoFailCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := health.NewManager()
	m.MustRegister(staticCheck{name: "pg-main", err: nil})
	m.MustRegister(staticCheck{name: "kafka-brokers", err: nil})

	beforeFail := readCounter(t, "pg-main", "fail")
	beforeFail2 := readCounter(t, "kafka-brokers", "fail")

	r := gin.New()
	r.GET("/readyz", health.ReadyzHandler(m))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, beforeFail, readCounter(t, "pg-main", "fail"),
		"全 OK 时 pg-main/fail 计数器不应增长")
	assert.Equal(t, beforeFail2, readCounter(t, "kafka-brokers", "fail"),
		"全 OK 时 kafka-brokers/fail 计数器不应增长")
}