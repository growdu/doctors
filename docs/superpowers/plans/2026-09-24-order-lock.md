# 陪诊师确认锁单 30s Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `docs/superpowers/specs/2026-09-24-order-matching-redesign.md` 落地 —— 把抢单锁单改为「患者从候选列表选 escort → 陪诊师 30s 内 confirm / reject / 超时自动回退」。落地项：状态机新增 `selecting_escort` + `escort_pending_acceptance`；订单字段 `selected_escort_id` + `escort_pending_expire_at` 替换 `lock_owner` / `lock_expire_at`；Redis SETNX `orders:confirm:{order_id}` 30s 防止 select-escort 重复触发；`EscortPendingScanner` 5s 扫一次超时单自动回退 `selecting_escort` 并发 `OrderEscortRejectedEvent`；3 个新事件替换 `OrderMatchingEvent`。

**Architecture:** 复用 `shared/lock.RedisLocker`（v1 抢单锁单 plan 落地）+ `shared/lock.NopLocker`（dev / unit test passthrough）；service 层三段式 `SelectEscort`（患者选 → escort_pending_acceptance）→ `ConfirmAccept`（陪诊师 confirm → accepted）→ `RejectAccept`（陪诊师拒或超时 → selecting_escort）；后台 `EscortPendingScanner` 沿用 `ExpiredLockScanner` 文件名与扫描协程模式，但扫描条件改为 `status='escort_pending_acceptance' AND escort_pending_expire_at < NOW()`，回退目标为 `selecting_escort`；事件总线砍掉 `OrderMatchingEvent`，新增 `OrderSelectingEscortEvent` / `OrderEscortConfirmedEvent` / `OrderEscortRejectedEvent`。

**Tech Stack:** Go 1.24+ · go-redis v9.22 · pgx v5.7 · testify v1.11 · segmentio/kafka-go v0.4.51 · alicebob/miniredis v9（集成测试）。

**前置依赖:**
- `2026-09-24-state-machine.md`（修订版）—— 加 `selecting_escort` + `escort_pending_acceptance` 状态、transitions 更新、0009 迁移（`lock_owner` / `lock_expire_at` → `selected_escort_id` / `escort_pending_expire_at`）。本 plan **不重复** 落地状态机 / migration，假定 state-machine 修订版已 commit。
- `shared/lock.RedisLocker` + `shared/lock.NopLocker`（v1 order-lock plan 已落地），本 plan 直接复用。

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1 + go-redis v9.22 + segmentio/kafka-go v0.4.51 + alicebob/miniredis v9
- Redis 7+（锁单 key：`orders:confirm:{order_id}`，30s TTL，TTL 兜底超时回退）
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`
- Redis 不可用 = 服务降级（DB version 乐观锁 + `escort_pending_expire_at` 兜底），不 panic
- 30s TTL 与 `escort_pending_expire_at` 一致值
- 通知 "best-effort"，失败不阻塞主业务
- 状态机是 pure function，无副作用（service 层负责副作用）
- 状态字符串与 DB CHECK 约束**一一对应**，禁止拼写漂移
- service 层所有"修改 DB + 发事件"两步：DB 在事务里，事件 best-effort 在事务外

---

## File Structure

| 路径 | 变更 | 职责 |
|------|------|------|
| `services/order/internal/state/machine.go` | 假定 state-machine 修订版已交付 | 新增 `StatusSelectingEscort` / `StatusEscortPendingAcceptance`；transitions 调整（不在本 plan 落地） |
| `services/order/internal/state/machine_test.go` | 假定 state-machine 修订版已交付 | 新状态 / 新转换测试（不在本 plan 落地） |
| `migrations/0009_orders_select_escort.up.sql` | 假定 state-machine 修订版已交付 | drop `lock_owner` / `lock_expire_at`，加 `selected_escort_id` / `escort_pending_expire_at`，新索引（不在本 plan 落地） |
| `shared/contracts/events.go` | Modify | 删 `TopicOrderMatching` / `OrderMatchingEvent`；加 `TopicOrderSelectingEscort` / `TopicOrderEscortConfirmed` / `TopicOrderEscortRejected` + 对应 Event 结构 |
| `shared/contracts/contracts_test.go` | Modify | 加 3 个新事件 RoundTrip + Topic 常量测试 |
| `services/order/internal/events/publisher.go` | Modify | `Publisher` 接口：删 `PublishOrderMatching`；加 `PublishOrderSelectingEscort` / `PublishOrderEscortConfirmed` / `PublishOrderEscortRejected`；`KafkaPublisher` / `NopPublisher` 同步更新 |
| `services/order/internal/events/publisher_test.go` | Modify | 新方法计数 / Kafka mock 调用测试 |
| `services/order/internal/repo/order_repo.go` | Modify | `Order` struct：`LockOwner` / `LockExpireAt` → `SelectedEscortID` / `EscortPendingExpireAt`；删 `LockForAccept` / `ReleaseLock` / `LockExpired`；加 `SelectForEscort` / `ConfirmByEscort` / `RejectByEscort` / `PendingExpired` |
| `services/order/internal/repo/order_repo_integration_test.go` | Modify | 加 `SelectForEscort` / `ConfirmByEscort` / `RejectByEscort` / `PendingExpired` 集成测试 |
| `services/order/internal/service/order_service_test.go` | Modify | fakeOrderRepo 的 `LockForAccept` / `ReleaseLock` / `LockExpired` stub 同步改为新方法（防止编译失败） |
| `services/order/internal/service/accept.go` | Modify | 删 `Accept`（Deprecated 注释也清理）/ `TryLock` / `ReleaseAcceptLock` / `ConfirmAccept`（v1 抢单锁单）；加 `SelectEscort` / `ConfirmAccept`（新语义）/ `RejectAccept`；错误码同步：保留 `ErrLockTaken` 但语义改为"select-escort 重复触发" |
| `services/order/internal/service/accept_integration_test.go` | Modify | 删旧集成测试；加 SelectEscort / ConfirmAccept / RejectAccept 集成测试（含 Redis SETNX + DB） |
| `services/order/internal/scheduler/expired_lock_scanner.go` | Modify | 沿用文件名（避免无意义 rename）；scanner 改用 `PendingExpired` + 扫 `status='escort_pending_acceptance'`；发 `OrderEscortRejectedEvent(reason='lock_expired')`；release 方法名 `ReleaseAcceptLock` 替换为 `RejectAccept` |
| `services/order/internal/scheduler/expired_lock_scanner_test.go` | Modify | 更新 fakeRepo / fakePublisher / fakeReleaseSvc；改测 `PendingExpired` + `RejectAccept` + `OrderEscortRejectedEvent` |
| `services/order/internal/cmd/main.go` | Modify | 装配 RedisLocker + KafkaPublisher + 启动 scanner（接口变更后字段名同步） |
| `docs/04-业务流程.md` §4.3 | Modify | 替换"抢单锁单流程"为"选 escort + 陪诊师 30s 确认流程" |
| `docs/07-数据模型.md` | Modify | orders 表 schema 同步（`selected_escort_id` / `escort_pending_expire_at` 替换 `lock_owner` / `lock_expire_at`），CHECK 加入 `selecting_escort` / `escort_pending_acceptance` |
| `dev.md` | Modify | 追加 "3.11 陪诊师确认锁单 30s" 一行 |

---

### Task 1: shared/contracts 3 个新事件 + 删 OrderMatchingEvent

**Files:**
- Modify: `shared/contracts/events.go`
- Modify: `shared/contracts/contracts_test.go`

**Step 1: 写 RoundTrip 失败测试**

在 `shared/contracts/contracts_test.go` 末尾追加：

```go
// TestOrderSelectingEscortEvent_RoundTrip 验证 JSON 序列化可逆。
func TestOrderSelectingEscortEvent_RoundTrip(t *testing.T) {
    now := time.Now().Truncate(time.Second)
    ev := OrderSelectingEscortEvent{
        OrderID:     100,
        Candidates:  []int64{7, 8, 9},
        GeneratedAt: now,
    }
    data, err := json.Marshal(ev)
    require.NoError(t, err)
    var got OrderSelectingEscortEvent
    require.NoError(t, json.Unmarshal(data, &got))
    assert.Equal(t, ev, got)
    assert.Equal(t, []int64{7, 8, 9}, got.Candidates)
}

// TestOrderEscortConfirmedEvent_RoundTrip 验证 JSON 序列化可逆。
func TestOrderEscortConfirmedEvent_RoundTrip(t *testing.T) {
    now := time.Now().Truncate(time.Second)
    ev := OrderEscortConfirmedEvent{
        OrderID:     100,
        EscortID:    7,
        ConfirmedAt: now,
    }
    data, err := json.Marshal(ev)
    require.NoError(t, err)
    var got OrderEscortConfirmedEvent
    require.NoError(t, json.Unmarshal(data, &got))
    assert.Equal(t, ev, got)
}

// TestOrderEscortRejectedEvent_RoundTrip 验证 JSON 序列化可逆（含 reason 枚举）。
func TestOrderEscortRejectedEvent_RoundTrip(t *testing.T) {
    now := time.Now().Truncate(time.Second)
    cases := []string{"escort_declined", "lock_expired"}
    for _, reason := range cases {
        t.Run(reason, func(t *testing.T) {
            ev := OrderEscortRejectedEvent{
                OrderID:    100,
                EscortID:   7,
                Reason:     reason,
                RejectedAt: now,
            }
            data, err := json.Marshal(ev)
            require.NoError(t, err)
            var got OrderEscortRejectedEvent
            require.NoError(t, json.Unmarshal(data, &got))
            assert.Equal(t, ev, got)
        })
    }
}

// TestTopicConstants_NewTopics 验证 3 个新 topic 常量值。
func TestTopicConstants_NewTopics(t *testing.T) {
    assert.Equal(t, "order.selecting_escort", TopicOrderSelectingEscort)
    assert.Equal(t, "order.confirmed", TopicOrderEscortConfirmed)
    assert.Equal(t, "order.rejected", TopicOrderEscortRejected)
}
```

**Step 2: 跑测试确认失败**

Run: `go test -count=1 -run 'TestOrderSelectingEscortEvent_RoundTrip|TestOrderEscortConfirmedEvent_RoundTrip|TestOrderEscortRejectedEvent_RoundTrip|TestTopicConstants_NewTopics' ./shared/contracts/`
Expected: FAIL — `undefined: OrderSelectingEscortEvent` / `undefined: TopicOrderSelectingEscort` 等

**Step 3: 在 events.go 删 OrderMatchingEvent + 加 3 个新 Event**

`shared/contracts/events.go`：

```go
const (
    TopicOrderCreated   = "order.created"
    TopicOrderAccepted  = "order.accepted"
    TopicOrderCancelled = "order.cancelled"
    TopicOrderReviewed  = "order.reviewed"
    // 注意：TopicOrderMatching 已删除（spec §5.2：撤销"重新进入匹配池"语义，
    //      因为新流程是患者选 escort，不需要"重新进入匹配池"事件）。
    TopicOrderSelectingEscort = "order.selecting_escort"  // 新增：候选列表生成
    TopicOrderEscortConfirmed = "order.confirmed"          // 新增：陪诊师 30s 内 confirm
    TopicOrderEscortRejected  = "order.rejected"           // 新增：陪诊师拒 / 超时

    TopicUserRegistered   = "user.registered"
    TopicUserRealNameDone = "user.real_name.done"

    TopicEscortRegistered  = "escort.registered"
    TopicEscortAvailable   = "escort.available"
    TopicEscortUnavailable = "escort.unavailable"

    TopicPaymentCreated   = "payment.created"
    TopicPaymentCompleted = "payment.completed"
    TopicPaymentRefunded  = "payment.refunded"
    TopicRefundCompleted  = "refund.completed"

    TopicSOSRaised   = "sos.raised"
    TopicMessageSent = "message.sent"
)

// OrderSelectingEscortEvent 订单候选陪诊师列表生成（order-service → match-service / notification）。
// 触发：订单 paid → selecting_escort 转换后由 order-service 发布；match-service 接收后无操作
// （候选由 order-service 主动生成并写入），notification 服务推送"请选择陪诊师"给患者。
type OrderSelectingEscortEvent struct {
    OrderID     int64     `json:"order_id"`
    Candidates  []int64   `json:"candidates"`   // 候选 escort_ids（按评分倒序，Top N）
    GeneratedAt time.Time `json:"generated_at"`
}

// OrderEscortConfirmedEvent 陪诊师 30s 内 confirm → accepted（order-service → notification / billing）。
type OrderEscortConfirmedEvent struct {
    OrderID     int64     `json:"order_id"`
    EscortID    int64     `json:"escort_id"`
    ConfirmedAt time.Time `json:"confirmed_at"`
}

// OrderEscortRejectedEvent 陪诊师拒接 或 30s 超时 → 回退 selecting_escort（order-service → notification）。
// Reason 取值：
//   - "escort_declined" 陪诊师主动 reject
//   - "lock_expired"     30s 超时由 scanner 自动回退
type OrderEscortRejectedEvent struct {
    OrderID    int64     `json:"order_id"`
    EscortID   int64     `json:"escort_id"`
    Reason     string    `json:"reason"`
    RejectedAt time.Time `json:"rejected_at"`
}

// 注意：OrderMatchingEvent 已删除（spec §5.2）。
```

**Step 4: 跑测试确认通过**

Run: `go test -count=1 ./shared/contracts/`
Expected: PASS（注意：旧 `OrderMatchingEvent` 相关测试若存在需要由 state-machine plan 负责清理；本 plan 不重复删除旧测试代码——见 §Assumptions）

**Step 5: Commit**

```bash
git add shared/contracts/
git commit -m "feat(contracts): 删 OrderMatchingEvent；加 OrderSelectingEscort/Confirmed/Rejected + 3 个新 topic (4 个新测试)"
```

---

### Task 2: events.Publisher 删 PublishOrderMatching + 加 3 个新方法

**Files:**
- Modify: `services/order/internal/events/publisher.go`
- Modify: `services/order/internal/events/publisher_test.go`

**Step 1: 写新方法的失败测试**

在 `publisher_test.go` 末尾追加：

```go
// TestNopPublisher_CountsSelectingEscort 验证 PublishOrderSelectingEscort 计数。
func TestNopPublisher_CountsSelectingEscort(t *testing.T) {
    p := &NopPublisher{}
    ev := contracts.OrderSelectingEscortEvent{OrderID: 1, Candidates: []int64{7, 8}}
    require.NoError(t, p.PublishOrderSelectingEscort(context.Background(), ev))
    assert.Equal(t, 1, p.SelectingEscortCount)
}

// TestNopPublisher_CountsEscortConfirmed 验证 PublishOrderEscortConfirmed 计数。
func TestNopPublisher_CountsEscortConfirmed(t *testing.T) {
    p := &NopPublisher{}
    ev := contracts.OrderEscortConfirmedEvent{OrderID: 1, EscortID: 7}
    require.NoError(t, p.PublishOrderEscortConfirmed(context.Background(), ev))
    assert.Equal(t, 1, p.EscortConfirmedCount)
}

// TestNopPublisher_CountsEscortRejected 验证 PublishOrderEscortRejected 计数（两种 reason）。
func TestNopPublisher_CountsEscortRejected(t *testing.T) {
    p := &NopPublisher{}
    cases := []string{"escort_declined", "lock_expired"}
    for _, reason := range cases {
        ev := contracts.OrderEscortRejectedEvent{
            OrderID: 1, EscortID: 7, Reason: reason,
        }
        require.NoError(t, p.PublishOrderEscortRejected(context.Background(), ev))
    }
    assert.Equal(t, 2, p.EscortRejectedCount)
}
```

**Step 2: 跑测试确认失败**

Run: `go test -count=1 -run 'TestNopPublisher_CountsSelectingEscort|TestNopPublisher_CountsEscortConfirmed|TestNopPublisher_CountsEscortRejected' ./services/order/internal/events/`
Expected: FAIL — `undefined: NopPublisher.SelectingEscortCount` / `undefined: PublishOrderSelectingEscort` 等

**Step 3: 改写 publisher.go**

```go
// Package events 是 order-service 的事件发布层。
//
// 设计要点：
//   - Publisher 接口只暴露业务事件。
//   - 默认实现是 KafkaPublisher，用 segmentio/kafka-go。
//   - 入参用 shared/contracts 的事件类型（不是 repo.Order），避免暴露 DB 结构。
//   - Topic 名复用 contracts 常量。
//   - Publish 失败只 log，不阻塞业务（事件 best-effort）。
//
// 2026-09-24-order-matching-redesign：
//   - 删 PublishOrderMatching（撤销"重新进入匹配池"事件）。
//   - 加 PublishOrderSelectingEscort / PublishOrderEscortConfirmed / PublishOrderEscortRejected。
package events

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "strconv"
    "time"

    "github.com/segmentio/kafka-go"

    "github.com/growdu/doctors/shared/contracts"
)

// Publisher 抽象订单事件发布。
type Publisher interface {
    PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error
    PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error
    PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error
    PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error  // 新增
    PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error // 新增
    PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error   // 新增
    Close() error
}

// KafkaPublisher 用 kafka-go writer 直接写 topic。
type KafkaPublisher struct {
    writer *kafka.Writer
}

// NewKafkaPublisher 构造 publisher。
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
    return &KafkaPublisher{
        writer: &kafka.Writer{
            Addr:         kafka.TCP(brokers...),
            Balancer:     &kafka.LeastBytes{},
            BatchTimeout: 50 * time.Millisecond,
            RequiredAcks: kafka.RequireOne,
            Async:        false,
        },
    }
}

// Close 关闭底层 writer。
func (p *KafkaPublisher) Close() error { return p.writer.Close() }

// publish 写一条 JSON 到 topic。
func (p *KafkaPublisher) publish(ctx context.Context, topic string, key string, body any) error {
    if p == nil || p.writer == nil {
        return errors.New("publisher: writer is nil")
    }
    data, err := json.Marshal(body)
    if err != nil {
        return fmt.Errorf("marshal %s: %w", topic, err)
    }
    return p.writer.WriteMessages(ctx, kafka.Message{
        Topic: topic,
        Key:   []byte(key),
        Value: data,
        Time:  time.Now(),
    })
}

// PublishOrderCreated 发布 order.created 事件。
func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
    return p.publish(ctx, contracts.TopicOrderCreated, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderAccepted 发布 order.accepted 事件。
func (p *KafkaPublisher) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
    return p.publish(ctx, contracts.TopicOrderAccepted, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderCancelled 发布 order.cancelled 事件。
func (p *KafkaPublisher) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
    return p.publish(ctx, contracts.TopicOrderCancelled, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderSelectingEscort 发布 order.selecting_escort 事件（候选列表生成）。
func (p *KafkaPublisher) PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error {
    return p.publish(ctx, contracts.TopicOrderSelectingEscort, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderEscortConfirmed 发布 order.confirmed 事件（陪诊师 confirm）。
func (p *KafkaPublisher) PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error {
    return p.publish(ctx, contracts.TopicOrderEscortConfirmed, strconv.FormatInt(ev.OrderID, 10), ev)
}

// PublishOrderEscortRejected 发布 order.rejected 事件（陪诊师拒 / 超时）。
func (p *KafkaPublisher) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
    return p.publish(ctx, contracts.TopicOrderEscortRejected, strconv.FormatInt(ev.OrderID, 10), ev)
}

// NopPublisher 是测试或 dev 占位实现。
type NopPublisher struct {
    CreatedCount          int
    AcceptedCount         int
    CancelledCount        int
    SelectingEscortCount  int // 新增
    EscortConfirmedCount  int // 新增
    EscortRejectedCount   int // 新增
}

// PublishOrderCreated 计数 + 返回。
func (p *NopPublisher) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
    p.CreatedCount++
    return nil
}

// PublishOrderAccepted 计数 + 返回。
func (p *NopPublisher) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
    p.AcceptedCount++
    return nil
}

// PublishOrderCancelled 计数 + 返回。
func (p *NopPublisher) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
    p.CancelledCount++
    return nil
}

// PublishOrderSelectingEscort 计数 + 返回。
func (p *NopPublisher) PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error {
    p.SelectingEscortCount++
    return nil
}

// PublishOrderEscortConfirmed 计数 + 返回。
func (p *NopPublisher) PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error {
    p.EscortConfirmedCount++
    return nil
}

// PublishOrderEscortRejected 计数 + 返回。
func (p *NopPublisher) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
    p.EscortRejectedCount++
    return nil
}

// Close NopPublisher 无资源。
func (p *NopPublisher) Close() error { return nil }
```

**Step 4: 删除 `PublishOrderMatching` 引用**

> 注：v1 抢单锁单 plan 已加 `PublishOrderMatching`，本 plan 必须把它从 `Publisher` 接口删除。
> 全局搜索 `PublishOrderMatching` / `MatchingCount` / `OrderMatchingEvent`，确认调用方都已迁移到新事件。
> 若仍有 v1 调用方（accept.go / scheduler 等），先在本 Task 标记为编译失败留待 Task 3/4 改 service + repo 时一起改。

```bash
# 全局搜索确认没有遗漏调用
grep -rn "PublishOrderMatching\|MatchingCount\|OrderMatchingEvent\|TopicOrderMatching" services/ shared/
# 期望：仅出现在 contracts/events.go（已删）、本 plan（标记 TODO）、以及待改的 service / scheduler 文件
```

**Step 5: 跑测试确认通过**

Run: `go test -count=1 ./services/order/internal/events/`
Expected: 现有 3 个新计数测试 PASS；但 service / scheduler 包可能编译失败（PublishOrderMatching 不存在）——留待 Task 3/4 修复

**Step 6: Commit**

```bash
git add services/order/internal/events/
git commit -m "feat(events): 删 PublishOrderMatching；加 PublishOrderSelectingEscort/Confirmed/Rejected (3 个新测试)"
```

---

### Task 3: repo 适配新字段（Order struct + 4 个新方法）

**Files:**
- Modify: `services/order/internal/repo/order_repo.go`
- Modify: `services/order/internal/repo/order_repo_integration_test.go`

**Step 1: 写新方法的失败集成测试**

在 `order_repo_integration_test.go` 末尾追加：

```go
// TestSelectForEscort_TransitionsToPending 验证从 selecting_escort → escort_pending_acceptance。
func TestSelectForEscort_TransitionsToPending(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    orderID := seedOrder(t, pool, "selecting_escort")
    expire := time.Now().Add(30 * time.Second)

    require.NoError(t, r.SelectForEscort(context.Background(), orderID, 7, expire, 1))

    o, err := r.FindByID(context.Background(), orderID)
    require.NoError(t, err)
    assert.Equal(t, "escort_pending_acceptance", o.Status)
    require.NotNil(t, o.SelectedEscortID)
    assert.Equal(t, int64(7), *o.SelectedEscortID)
    require.NotNil(t, o.EscortPendingExpireAt)
    assert.WithinDuration(t, expire, *o.EscortPendingExpireAt, time.Second)
}

// TestSelectForEscort_RejectsWrongStatus 验证非 selecting_escort 状态拒。
func TestSelectForEscort_RejectsWrongStatus(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    orderID := seedOrder(t, pool, "paid")
    expire := time.Now().Add(30 * time.Second)

    err := r.SelectForEscort(context.Background(), orderID, 7, expire, 1)
    assert.ErrorIs(t, err, ErrInvalidStateForSelect)
}

// TestConfirmByEscort_WritesFinalEscortID 验证 confirm 后写 escort_id（最终字段）。
func TestConfirmByEscort_WritesFinalEscortID(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    orderID := seedOrder(t, pool, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() + INTERVAL '30 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    o, _ := r.FindByID(context.Background(), orderID)

    require.NoError(t, r.ConfirmByEscort(context.Background(), orderID, 7, o.Version))

    o2, _ := r.FindByID(context.Background(), orderID)
    assert.Equal(t, "accepted", o2.Status)
    require.NotNil(t, o2.EscortID)
    assert.Equal(t, int64(7), *o2.EscortID)
    assert.Nil(t, o2.SelectedEscortID, "确认后清空 selected_escort_id")
    assert.Nil(t, o2.EscortPendingExpireAt, "确认后清空 escort_pending_expire_at")
}

// TestConfirmByEscort_RejectsMismatchedEscort 验证 selected_escort_id ≠ caller 拒。
func TestConfirmByEscort_RejectsMismatchedEscort(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    orderID := seedOrder(t, pool, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1 WHERE id = $2`, 7, orderID)
    require.NoError(t, err)
    o, _ := r.FindByID(context.Background(), orderID)

    err = r.ConfirmByEscort(context.Background(), orderID, 99, o.Version) // 99 ≠ 7
    assert.ErrorIs(t, err, ErrSelectedEscortMismatch)
}

// TestRejectByEscort_RevertsToSelectingEscort 验证 reject 回退 selecting_escort。
func TestRejectByEscort_RevertsToSelectingEscort(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    orderID := seedOrder(t, pool, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() + INTERVAL '30 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    o, _ := r.FindByID(context.Background(), orderID)

    require.NoError(t, r.RejectByEscort(context.Background(), orderID, 7, o.Version))

    o2, _ := r.FindByID(context.Background(), orderID)
    assert.Equal(t, "selecting_escort", o2.Status)
    assert.Nil(t, o2.SelectedEscortID, "拒接后清空 selected_escort_id")
    assert.Nil(t, o2.EscortPendingExpireAt, "拒接后清空 escort_pending_expire_at")
}

// TestPendingExpired_ReturnsExpired 验证扫描超时 escort_pending_acceptance 单。
func TestPendingExpired_ReturnsExpired(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    now := time.Now()
    expiredID := seedOrder(t, pool, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = $2 WHERE id = $3`,
        7, now.Add(-1*time.Hour), expiredID)
    require.NoError(t, err)

    freshID := seedOrder(t, pool, "escort_pending_acceptance")
    _, err = pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = $2 WHERE id = $3`,
        8, now.Add(time.Hour), freshID)
    require.NoError(t, err)

    rows, err := r.PendingExpired(context.Background(), now, 10)
    require.NoError(t, err)
    require.Len(t, rows, 1)
    assert.Equal(t, expiredID, rows[0].ID)
    require.NotNil(t, rows[0].SelectedEscortID)
    assert.Equal(t, int64(7), *rows[0].SelectedEscortID)
}

// TestPendingExpired_SkipsNilExpireAt 验证 escort_pending_expire_at=nil 不在扫描内。
func TestPendingExpired_SkipsNilExpireAt(t *testing.T) {
    pool := setupPool(t)
    defer pool.Close()
    r := newTestRepo(pool)
    orderID := seedOrder(t, pool, "escort_pending_acceptance")
    // 不写 escort_pending_expire_at（= NULL）

    rows, err := r.PendingExpired(context.Background(), time.Now(), 10)
    require.NoError(t, err)
    assert.Empty(t, rows, "nil expire_at 不应被扫到")
    _ = orderID
}
```

**Step 2: 跑测试确认失败**

Run: `go test -tags=integration -run 'TestSelectForEscort|TestConfirmByEscort|TestRejectByEscort|TestPendingExpired' ./services/order/internal/repo/`
Expected: FAIL — `undefined: (*OrderRepo).SelectForEscort` / `undefined: ErrInvalidStateForSelect` / `undefined: (*Order).SelectedEscortID` 等

**Step 3: 改写 order_repo.go**

```go
// Package repo 提供 order-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc 工具链）。
//   - UpdateStatus 用 version 做乐观锁；并发安全靠 DB 不靠 Redis。
//   - 状态变更走 UpdateStatus + InsertEvent；service 层负责把"原 status"先读出再调用，
//     同一事务由 shared/db.WithTx 包装。
//
// 2026-09-24-order-matching-redesign：
//   - 删 LockForAccept / ReleaseLock / LockExpired（v1 抢单锁单）。
//   - 加 SelectForEscort / ConfirmByEscort / RejectByEscort / PendingExpired（陪诊师确认锁单）。
//   - Order 字段：LockOwner/LockExpireAt → SelectedEscortID/EscortPendingExpireAt。
package repo

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Order 映射 orders 表行。
type Order struct {
    ID                     int64
    OrderNo                string
    PatientID              int64
    EscortID               *int64
    HospitalID             int64
    PackageID              int64
    ServiceStartAt         any
    Amount                 float64
    FinalAmount            float64
    Status                 string
    Version                int
    SelectedEscortID       *int64     // escort_pending_acceptance 期间填；nil 表示未选
    EscortPendingExpireAt  *time.Time // 30s 超时时间；nil 表示未选
}

// OrderEvent 映射 order_events 表行。
type OrderEvent struct {
    ID         int64
    OrderID    int64
    FromStatus *string
    ToStatus   string
    ActorID    *int64
    Payload    []byte
}

// ErrOrderNotFound 是查询无结果时的哨兵。
var ErrOrderNotFound = errors.New("repo: order not found")

// ErrVersionConflict 是乐观锁冲突（version 不匹配）。
var ErrVersionConflict = errors.New("repo: order version conflict")

// ErrInvalidStateForSelect 是 SelectForEscort 时订单状态不在 selecting_escort 的哨兵。
var ErrInvalidStateForSelect = errors.New("repo: order not in selecting_escort state for select")

// ErrSelectedEscortMismatch 是 ConfirmByEscort / RejectByEscort 时 selected_escort_id ≠ caller 的哨兵。
var ErrSelectedEscortMismatch = errors.New("repo: selected escort mismatch")

// OrderRepo 是 orders + order_events 表的仓储。
type OrderRepo struct {
    pool *pgxpool.Pool
}

// NewOrderRepo 构造仓储。
func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo { return &OrderRepo{pool: pool} }

// Create 插入新订单；ID / version 由 DB 回写。
func (r *OrderRepo) Create(ctx context.Context, o *Order) error {
    const q = `
        INSERT INTO orders (order_no, patient_id, hospital_id, package_id,
                            service_start_at, amount, final_amount, status)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING id, version`
    return r.pool.QueryRow(ctx, q,
        o.OrderNo, o.PatientID, o.HospitalID, o.PackageID,
        o.ServiceStartAt, o.Amount, o.FinalAmount, o.Status,
    ).Scan(&o.ID, &o.Version)
}

// FindByID 按主键查找。
func (r *OrderRepo) FindByID(ctx context.Context, id int64) (*Order, error) {
    const q = baseSelect + ` WHERE id = $1 AND deleted_at IS NULL`
    return r.scanOne(r.pool.QueryRow(ctx, q, id))
}

// ListByPatient 按患者分页查订单，按 created_at DESC。
func (r *OrderRepo) ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*Order, error) {
    const q = baseSelect + ` WHERE patient_id = $1 AND deleted_at IS NULL
                             ORDER BY created_at DESC LIMIT $2 OFFSET $3`
    rows, err := r.pool.Query(ctx, q, patientID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("list orders: %w", err)
    }
    defer rows.Close()
    out := make([]*Order, 0)
    for rows.Next() {
        o, err := r.scanRow(rows)
        if err != nil {
            return nil, err
        }
        out = append(out, o)
    }
    return out, rows.Err()
}

// UpdateStatus 用乐观锁更新状态；escortID 可空（confirmed 时填）。
func (r *OrderRepo) UpdateStatus(ctx context.Context, id int64, toStatus string, expectVersion int, escortID *int64) error {
    const q = `
        UPDATE orders
           SET status = $1, version = version + 1, updated_at = NOW(),
               escort_id = COALESCE($2, escort_id)
         WHERE id = $3 AND version = $4 AND deleted_at IS NULL`
    tag, err := r.pool.Exec(ctx, q, toStatus, escortID, id, expectVersion)
    if err != nil {
        return fmt.Errorf("update status: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return ErrVersionConflict
    }
    return nil
}

// InsertEvent 写入一条 order_events。
func (r *OrderRepo) InsertEvent(ctx context.Context, orderID int64, fromStatus *string, toStatus string, actorID *int64, payload []byte) error {
    const q = `
        INSERT INTO order_events (order_id, from_status, to_status, actor_id, payload)
        VALUES ($1, $2, $3, $4, $5)`
    _, err := r.pool.Exec(ctx, q, orderID, fromStatus, toStatus, actorID, payload)
    if err != nil {
        return fmt.Errorf("insert event: %w", err)
    }
    return nil
}

// ListEvents 取一个订单的所有事件，按时间正序。
func (r *OrderRepo) ListEvents(ctx context.Context, orderID int64) ([]*OrderEvent, error) {
    const q = `
        SELECT id, order_id, from_status, to_status, actor_id, payload
          FROM order_events WHERE order_id = $1 ORDER BY created_at ASC, id ASC`
    rows, err := r.pool.Query(ctx, q, orderID)
    if err != nil {
        return nil, fmt.Errorf("list events: %w", err)
    }
    defer rows.Close()
    out := make([]*OrderEvent, 0)
    for rows.Next() {
        var e OrderEvent
        if err := rows.Scan(&e.ID, &e.OrderID, &e.FromStatus, &e.ToStatus, &e.ActorID, &e.Payload); err != nil {
            return nil, err
        }
        out = append(out, &e)
    }
    return out, rows.Err()
}

// SelectForEscort 患者从候选列表选 → 切 escort_pending_acceptance + 写 selected_escort_id + escort_pending_expire_at。
//
// 校验：
//   - status = 'selecting_escort'
//   - version 匹配（乐观锁）
//   - selected_escort_id IS NULL（避免重复 select）
//
// 失败语义：
//   - version 不匹配 → ErrVersionConflict
//   - status 不在 selecting_escort → ErrInvalidStateForSelect
func (r *OrderRepo) SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
    const q = `
        UPDATE orders
           SET status = 'escort_pending_acceptance',
               selected_escort_id = $1,
               escort_pending_expire_at = $2,
               version = version + 1,
               updated_at = NOW()
         WHERE id = $3 AND version = $4
           AND status = 'selecting_escort' AND deleted_at IS NULL
           AND selected_escort_id IS NULL`
    tag, err := r.pool.Exec(ctx, q, escortID, expireAt, id, expectVersion)
    if err != nil {
        return fmt.Errorf("select for escort: %w", err)
    }
    if tag.RowsAffected() == 0 {
        var curStatus string
        var curVer int
        _ = r.pool.QueryRow(ctx, "SELECT status, version FROM orders WHERE id=$1", id).Scan(&curStatus, &curVer)
        if curVer != expectVersion {
            return ErrVersionConflict
        }
        return ErrInvalidStateForSelect
    }
    return nil
}

// ConfirmByEscort 陪诊师 30s 内 confirm → accepted + 写 escort_id（最终）。
//
// 校验：
//   - status = 'escort_pending_acceptance'
//   - selected_escort_id = caller（service 层在调用前再校验，这里兜底）
//   - version 匹配
//   - escort_pending_expire_at >= NOW()（30s 内才能 confirm，超时由 scanner 处理）
//
// 副作用：清空 selected_escort_id + escort_pending_expire_at。
func (r *OrderRepo) ConfirmByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
    const q = `
        UPDATE orders
           SET status = 'accepted',
               escort_id = $1,
               selected_escort_id = NULL,
               escort_pending_expire_at = NULL,
               version = version + 1,
               updated_at = NOW()
         WHERE id = $2 AND version = $3
           AND status = 'escort_pending_acceptance'
           AND selected_escort_id = $1
           AND deleted_at IS NULL
           AND escort_pending_expire_at >= NOW()`
    tag, err := r.pool.Exec(ctx, q, escortID, id, expectVersion)
    if err != nil {
        return fmt.Errorf("confirm by escort: %w", err)
    }
    if tag.RowsAffected() == 0 {
        // 区分 selected_escort 不匹配 / version 不匹配 / 超时
        var curStatus string
        var curSelected *int64
       	var curVer int
        _ = r.pool.QueryRow(ctx,
            "SELECT status, selected_escort_id, version FROM orders WHERE id=$1", id).
            Scan(&curStatus, &curSelected, &curVer)
        if curVer != expectVersion {
            return ErrVersionConflict
        }
        if curSelected == nil || *curSelected != escortID {
            return ErrSelectedEscortMismatch
        }
        return ErrInvalidStateForSelect // 超时 / 状态已变更（scanner 已回退）
    }
    return nil
}

// RejectByEscort 陪诊师拒接 → 回退 selecting_escort + 清空 selected_escort_id + escort_pending_expire_at。
//
// 校验：
//   - status = 'escort_pending_acceptance'
//   - selected_escort_id = caller
//   - version 匹配
func (r *OrderRepo) RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
    const q = `
        UPDATE orders
           SET status = 'selecting_escort',
               selected_escort_id = NULL,
               escort_pending_expire_at = NULL,
               version = version + 1,
               updated_at = NOW()
         WHERE id = $1 AND version = $2
           AND status = 'escort_pending_acceptance'
           AND selected_escort_id = $3
           AND deleted_at IS NULL`
    tag, err := r.pool.Exec(ctx, q, id, expectVersion, escortID)
    if err != nil {
        return fmt.Errorf("reject by escort: %w", err)
    }
    if tag.RowsAffected() == 0 {
        var curStatus string
        var curSelected *int64
        var curVer int
        _ = r.pool.QueryRow(ctx,
            "SELECT status, selected_escort_id, version FROM orders WHERE id=$1", id).
            Scan(&curStatus, &curSelected, &curVer)
        if curVer != expectVersion {
            return ErrVersionConflict
        }
        if curSelected == nil || *curSelected != escortID {
            return ErrSelectedEscortMismatch
        }
        return ErrInvalidStateForSelect
    }
    return nil
}

// PendingExpired 返回 escort_pending_expire_at < now 的 escort_pending_acceptance 单（按时间升序）。
// 给 §4.2 scanner 定时任务用。
func (r *OrderRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*Order, error) {
    const q = baseSelect + ` WHERE status = 'escort_pending_acceptance'
                              AND escort_pending_expire_at IS NOT NULL
                              AND escort_pending_expire_at < $1
                              ORDER BY escort_pending_expire_at ASC
                              LIMIT $2`
    rows, err := r.pool.Query(ctx, q, now, limit)
    if err != nil {
        return nil, fmt.Errorf("find pending expired: %w", err)
    }
    defer rows.Close()
    out := make([]*Order, 0)
    for rows.Next() {
        o, err := r.scanRow(rows)
        if err != nil {
            return nil, err
        }
        out = append(out, o)
    }
    return out, rows.Err()
}

// baseSelect 是 SELECT 子句。
const baseSelect = `
    SELECT id, order_no, patient_id, escort_id, hospital_id, package_id,
           service_start_at, amount, final_amount, status, version,
           selected_escort_id, escort_pending_expire_at
    FROM orders`

// scanOne 把单行扫描为 *Order。
func (r *OrderRepo) scanOne(row pgx.Row) (*Order, error) {
    o := &Order{}
    if err := row.Scan(
        &o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
        &o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
        &o.SelectedEscortID, &o.EscortPendingExpireAt,
    ); err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrOrderNotFound
        }
        return nil, fmt.Errorf("scan order: %w", err)
    }
    return o, nil
}

// scanRow 把 pgx.Rows 的一行扫描为 *Order。
func (r *OrderRepo) scanRow(rows pgx.Rows) (*Order, error) {
    o := &Order{}
    if err := rows.Scan(
        &o.ID, &o.OrderNo, &o.PatientID, &o.EscortID, &o.HospitalID, &o.PackageID,
        &o.ServiceStartAt, &o.Amount, &o.FinalAmount, &o.Status, &o.Version,
        &o.SelectedEscortID, &o.EscortPendingExpireAt,
    ); err != nil {
        return nil, err
    }
    return o, nil
}
```

> ⚠️ `accept.go` 的 `Service.TryLock` / `ReleaseAcceptLock` / `ConfirmAccept` 仍用老方法 `LockForAccept` / `ReleaseLock` / `LockExpired`，编译失败 —— 留待 Task 4 改 service 时同步替换。

**Step 4: 跑集成测试确认通过**

Run: `go test -tags=integration -run 'TestSelectForEscort|TestConfirmByEscort|TestRejectByEscort|TestPendingExpired' ./services/order/internal/repo/`
Expected: PASS

**Step 4.5: 同步更新 `order_service_test.go` 的 fakeOrderRepo stub**

> 注：删 `LockForAccept` / `ReleaseLock` / `LockExpired` 会导致 `services/order/internal/service/order_service_test.go` 的 `fakeOrderRepo` 编译失败。
> 在 `order_service_test.go` 把 3 个旧 stub 方法替换为：

```go
func (r *fakeOrderRepo) SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
    for _, o := range r.orders {
        if o.ID == id && o.Version == expectVersion && o.Status == "selecting_escort" {
            o.SelectedEscortID = &escortID
            t := expireAt
            o.EscortPendingExpireAt = &t
            o.Status = "escort_pending_acceptance"
            o.Version = expectVersion + 1
            return nil
        }
    }
    return repo.ErrInvalidStateForSelect
}

func (r *fakeOrderRepo) RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
    for _, o := range r.orders {
        if o.ID == id && o.Version == expectVersion && o.Status == "escort_pending_acceptance" &&
            o.SelectedEscortID != nil && *o.SelectedEscortID == escortID {
            o.SelectedEscortID = nil
            o.EscortPendingExpireAt = nil
            o.Status = "selecting_escort"
            o.Version = expectVersion + 1
            return nil
        }
    }
    return repo.ErrSelectedEscortMismatch
}

func (r *fakeOrderRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
    out := make([]*repo.Order, 0)
    for _, o := range r.orders {
        if o.Status == "escort_pending_acceptance" && o.EscortPendingExpireAt != nil && o.EscortPendingExpireAt.Before(now) {
            out = append(out, o)
            if len(out) >= limit { break }
        }
    }
    return out, nil
}
```

**Step 5: Commit**

```bash
git add services/order/internal/repo/ services/order/internal/service/order_service_test.go
git commit -m "feat(repo): 删 LockForAccept/ReleaseLock/LockExpired；加 SelectForEscort/ConfirmByEscort/RejectByEscort/PendingExpired + 字段 LockOwner→SelectedEscortID (7 个集成测试 + fakeOrderRepo stub)"
```

---

### Task 4: service 三段式（SelectEscort / ConfirmAccept / RejectAccept）+ Redis SETNX

**Files:**
- Modify: `services/order/internal/service/accept.go`
- Modify: `services/order/internal/service/accept_integration_test.go`

**Step 1: 写三个新方法的失败集成测试**

> 注：删旧 `TestAccept_*` / `TestTryLock_*` / `TestReleaseAcceptLock_*` / `TestConfirmAccept_*`（v1 抢单锁单），换为新流程的 `TestSelectEscort_*` / `TestConfirmAccept_*` / `TestRejectAccept_*`。

```go
import (
    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
    sharedlock "github.com/growdu/doctors/shared/lock"
)

// TestSelectEscort_RedisSetNX_BlocksDuplicate 验证 Redis SETNX 防止 select-escort 重复触发。
func TestSelectEscort_RedisSetNX_BlocksDuplicate(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "selecting_escort")
    r := repo.NewOrderRepo(pool)

    mr, _ := miniredis.Run()
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    defer rdb.Close()
    locker := sharedlock.NewRedisLocker(rdb)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker)

    require.NoError(t, svc.SelectEscort(context.Background(), orderID, patient, 7, 30*time.Second))
    err := svc.SelectEscort(context.Background(), orderID, patient, 8, 30*time.Second)
    assert.ErrorIs(t, err, ErrLockTaken)
}

// TestSelectEscort_NopLockerPassesThrough 验证 NopLocker 让 select 走 DB（Redis 不可用降级）。
func TestSelectEscort_NopLockerPassesThrough(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "selecting_escort")
    r := repo.NewOrderRepo(pool)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(sharedlock.NopLocker{})

    require.NoError(t, svc.SelectEscort(context.Background(), orderID, patient, 7, 30*time.Second))

    o, _ := r.FindByID(context.Background(), orderID)
    assert.Equal(t, "escort_pending_acceptance", o.Status)
    require.NotNil(t, o.SelectedEscortID)
    assert.Equal(t, int64(7), *o.SelectedEscortID)
}

// TestSelectEscort_RedisExpires_AllowsReselect 验证 TTL 到期后允许重新 select。
func TestSelectEscort_RedisExpires_AllowsReselect(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "selecting_escort")
    r := repo.NewOrderRepo(pool)

    mr, _ := miniredis.Run()
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    defer rdb.Close()
    locker := sharedlock.NewRedisLocker(rdb)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker)

    require.NoError(t, svc.SelectEscort(context.Background(), orderID, patient, 7, 100*time.Millisecond))
    time.Sleep(150 * time.Millisecond)
    // TTL 到期后允许重新 select
    require.NoError(t, svc.SelectEscort(context.Background(), orderID, patient, 8, 30*time.Second))
}

// TestSelectEscort_RejectsNonSelectingEscort 验证非 selecting_escort 状态拒。
func TestSelectEscort_RejectsNonSelectingEscort(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "paid")
    r := repo.NewOrderRepo(pool)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(sharedlock.NopLocker{})

    err := svc.SelectEscort(context.Background(), orderID, patient, 7, 30*time.Second)
    assert.Error(t, err)
}

// TestSelectEscort_PublishesOrderSelectingEscort 验证成功 select 后发布事件。
func TestSelectEscort_PublishesOrderSelectingEscort(t *testing.T) {
    // 注意：OrderSelectingEscortEvent 实际由 match-service 在生成候选时发布，
    // 而不是由 select 时发布。本测试改为验证 ConfirmAccept / RejectAccept 的事件发布。
    t.Skip("OrderSelectingEscortEvent 由 match-service 发布；order-service 不直接发")
}

// TestConfirmAccept_PublishesOrderEscortConfirmed 验证 confirm 后发 OrderEscortConfirmedEvent。
func TestConfirmAccept_PublishesOrderEscortConfirmed(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() + INTERVAL '30 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    r := repo.NewOrderRepo(pool)

    mr, _ := miniredis.Run()
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    defer rdb.Close()
    locker := sharedlock.NewRedisLocker(rdb)
    pub := &events.NopPublisher{}
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker).WithPublisher(pub)

    _, err = svc.ConfirmAccept(context.Background(), orderID, 7)
    require.NoError(t, err)

    assert.Equal(t, 1, pub.EscortConfirmedCount, "应发布 OrderEscortConfirmedEvent 一次")
    assert.Equal(t, 0, pub.EscortRejectedCount, "不应发布 Rejected 事件")
}

// TestConfirmAccept_ReleasesRedisLock 验证 confirm 成功后释放 Redis SETNX。
func TestConfirmAccept_ReleasesRedisLock(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "selecting_escort")
    r := repo.NewOrderRepo(pool)

    mr, _ := miniredis.Run()
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    defer rdb.Close()
    locker := sharedlock.NewRedisLocker(rdb)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker)

    require.NoError(t, svc.SelectEscort(context.Background(), orderID, patient, 7, 30*time.Second))
    _, err := svc.ConfirmAccept(context.Background(), orderID, 7)
    require.NoError(t, err)

    // Redis 锁应被释放（key 不存在）
    key := fmt.Sprintf("orders:confirm:%d", orderID)
    exists, _ := mr.Exists(key)
    assert.Equal(t, 0, exists, "confirm 后应释放 Redis SETNX")
}

// TestConfirmAccept_RejectsMismatchedEscort 验证 selected_escort_id ≠ caller 拒。
func TestConfirmAccept_RejectsMismatchedEscort(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() + INTERVAL '30 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    r := repo.NewOrderRepo(pool)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(sharedlock.NopLocker{})

    _, err = svc.ConfirmAccept(context.Background(), orderID, 99)
    assert.Error(t, err)
}

// TestConfirmAccept_RejectsAfterExpiry 验证 30s 超时后 confirm 拒（scanner 已回退）。
func TestConfirmAccept_RejectsAfterExpiry(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "escort_pending_acceptance")
    // 模拟已超时
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() - INTERVAL '1 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    r := repo.NewOrderRepo(pool)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(sharedlock.NopLocker{})

    _, err = svc.ConfirmAccept(context.Background(), orderID, 7)
    assert.Error(t, err, "超时后 confirm 应被业务层拒")
}

// TestRejectAccept_PublishesOrderEscortRejected 验证 reject 后发 OrderEscortRejectedEvent(reason='escort_declined')。
func TestRejectAccept_PublishesOrderEscortRejected(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() + INTERVAL '30 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    r := repo.NewOrderRepo(pool)

    mr, _ := miniredis.Run()
    defer mr.Close()
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    defer rdb.Close()
    locker := sharedlock.NewRedisLocker(rdb)
    pub := &capturingPublisher{}
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(locker).WithPublisher(pub)

    require.NoError(t, svc.SelectEscort(context.Background(), orderID, patient, 7, 30*time.Second))
    require.NoError(t, svc.RejectAccept(context.Background(), orderID, 7))

    require.NotNil(t, pub.rejectedEv, "应发布 OrderEscortRejectedEvent")
    assert.Equal(t, "escort_declined", pub.rejectedEv.Reason)
    assert.Equal(t, int64(7), pub.rejectedEv.EscortID)
}

// TestRejectAccept_RevertsToSelectingEscort 验证 reject 后 status 回退 selecting_escort。
func TestRejectAccept_RevertsToSelectingEscort(t *testing.T) {
    pool, patient := setupPool(t)
    defer pool.Close()
    orderID := seedOrder(t, pool, patient, "escort_pending_acceptance")
    _, err := pool.Exec(context.Background(),
        `UPDATE orders SET selected_escort_id = $1, escort_pending_expire_at = NOW() + INTERVAL '30 seconds' WHERE id = $2`,
        7, orderID)
    require.NoError(t, err)
    r := repo.NewOrderRepo(pool)
    svc := New(r, nilUserLookup{}).WithTx(&PGPoolTxRunner{Pool: pool}).WithLocker(sharedlock.NopLocker{})

    require.NoError(t, svc.RejectAccept(context.Background(), orderID, 7))

    o, _ := r.FindByID(context.Background(), orderID)
    assert.Equal(t, "selecting_escort", o.Status)
    assert.Nil(t, o.SelectedEscortID)
    assert.Nil(t, o.EscortPendingExpireAt)
}

// capturingPublisher 是测试用的 publisher，记录最近一次 RejectedEvent 用于断言 reason。
type capturingPublisher struct {
    events.NopPublisher
    rejectedEv *contracts.OrderEscortRejectedEvent
}
func (p *capturingPublisher) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
    p.rejectedEv = &ev
    return nil
}
```

**Step 2: 跑测试确认失败**

Run: `go test -tags=integration -run 'TestSelectEscort|TestConfirmAccept|TestRejectAccept' ./services/order/internal/service/`
Expected: FAIL — `undefined: (*Service).SelectEscort` / `undefined: (*Service).RejectAccept` 等

**Step 3: 改写 accept.go（删旧 + 加新）**

```go
// Package service - 陪诊师确认锁单核心 (SelectEscort / ConfirmAccept / RejectAccept)
//
// 设计要点：
//   - 2026-09-24-order-matching-redesign：业务从"抢单锁单"改为"陪诊师确认锁单"。
//     删 v1 的 Accept / TryLock / ReleaseAcceptLock / ConfirmAccept。
//   - 新三段式：
//       SelectEscort(ctx, orderID, patientID, escortID) - 患者从候选列表选 escort
//         → 验证 escort 可用（service.SelectEscort 内部校验 escort_availabilities，v2）
//         → Redis SETNX orders:confirm:{order_id} 30s（防止 select 重复触发）
//         → DB 切 escort_pending_acceptance + 写 selected_escort_id + escort_pending_expire_at
//       ConfirmAccept(ctx, orderID, escortID) - 陪诊师 30s 内 confirm
//         → 校验 escort 是 selected_escort_id + 未超时
//         → DB 切 accepted + 写 escort_id（最终）+ 清 selected_escort_id / escort_pending_expire_at
//         → Redis Release（best-effort）
//         → 发 OrderEscortConfirmedEvent
//       RejectAccept(ctx, orderID, escortID) - 陪诊师拒接
//         → 校验 escort 是 selected_escort_id
//         → DB 回退 selecting_escort + 清 selected_escort_id / escort_pending_expire_at
//         → Redis Release（best-effort）
//         → 发 OrderEscortRejectedEvent(reason='escort_declined')
//   - Redis SETNX 用 shared/lock.Key (best-effort)：失败 = 跳过；DB 兜底
//   - 整段 SQL 在一个事务里；事件 best-effort 在事务外
//   - 失败语义：
//       ErrLockTaken              → select-escort 已触发（Redis SETNX 已存在）
//       ErrInvalidStateForSelect  → 订单不在 selecting_escort
//       ErrSelectedEscortMismatch → confirm/reject 的 escort 不是 selected_escort_id
//       ErrVersionConflict        → version 不匹配
package service

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/growdu/doctors/services/order/internal/events"
    "github.com/growdu/doctors/services/order/internal/repo"
    "github.com/growdu/doctors/services/order/internal/state"
    "github.com/growdu/doctors/shared/contracts"
    "github.com/growdu/doctors/shared/errs"
)

// SelectEscort / ConfirmAccept / RejectAccept 错误码。
var (
    ErrLockTaken             = errors.New("service: order confirm lock taken (select already triggered)")
    ErrInvalidStateForSelect = repo.ErrInvalidStateForSelect
    ErrSelectedEscortMismatch = repo.ErrSelectedEscortMismatch
    ErrVersionConflict       = repo.ErrVersionConflict
)

// AcceptResult 是 ConfirmAccept 成功时的返回值（保留兼容）。
type AcceptResult struct {
    OrderID int64
    Version int
}

// TxRunner 是事务抽象；业务层只关心 fn(tx) 是否能跑。
type TxRunner interface {
    WithTx(ctx context.Context, fn func(pgx.Tx) error) error
}

// PGPoolTxRunner 是基于 pgxpool 的默认实现。
type PGPoolTxRunner struct {
    Pool *pgxpool.Pool
}

func (r *PGPoolTxRunner) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
    tx, err := r.Pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("service: begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback(ctx) }()
    if err := fn(tx); err != nil {
        _ = tx.Rollback(ctx)
        return err
    }
    return tx.Commit(ctx)
}

// SelectEscort 患者从候选列表选 escort → 切 escort_pending_acceptance。
//
// 三道防线：
//   1. Redis SETNX orders:confirm:{order_id} 30s（防止 select-escort 重复触发）
//   2. DB 校验 status='selecting_escort' + version
//   3. DB UPDATE status='escort_pending_acceptance' + selected_escort_id + escort_pending_expire_at
//
// 失败语义：
//   - Redis SETNX 已存在 → ErrLockTaken（fast path）
//   - 订单不在 selecting_escort → ErrInvalidStateForSelect
//   - version 不匹配 → ErrVersionConflict
//
// token 简化：random；Release 用 best-effort（不命中靠 TTL 兜底；v2 引入 lock_token DB 列做精确释放）。
func (s *Service) SelectEscort(ctx context.Context, orderID, patientID, escortID int64, ttl time.Duration) error {
    if orderID == 0 || patientID == 0 || escortID == 0 {
        return errs.New(errs.CodeParamInvalid, "order_id / patient_id / escort_id required")
    }

    // 第一道闸：Redis SETNX
    tokenBytes := make([]byte, 16)
    _, _ = rand.Read(tokenBytes)
    token := hex.EncodeToString(tokenBytes)
    key := fmt.Sprintf("orders:confirm:%d", orderID)
    ok, err := s.locker.TryLock(ctx, key, token, ttl)
    if err != nil {
        // Redis 异常：降级放行，由 DB version 乐观锁兜底
        _ = err
    }
    if err == nil && !ok {
        return ErrLockTaken
    }

    o, err := s.orders.FindByID(ctx, orderID)
    if err != nil {
        if errors.Is(err, repo.ErrOrderNotFound) {
            return errs.New(errs.CodeNotFound, "order not found")
        }
        return errs.Wrap(errs.CodeInternal, "find order", err)
    }
    if state.Status(o.Status) != state.StatusSelectingEscort {
        return errs.New(errs.CodeConflict, fmt.Sprintf("cannot select from status %s", o.Status))
    }
    // 简化：service 层不校验 escort 可用性（escort_availabilities 校验留给 escort-service，
    // 本 plan 不重复 escort-business 的工作；spec §7.2 候选推荐由 match-service 在生成候选时过滤）。

    expireAt := s.clockNow().Add(ttl)
    if err := s.orders.SelectForEscort(ctx, orderID, escortID, expireAt, o.Version); err != nil {
        if errors.Is(err, repo.ErrInvalidStateForSelect) {
            return ErrInvalidStateForSelect
        }
        if errors.Is(err, repo.ErrVersionConflict) {
            return ErrVersionConflict
        }
        return errs.Wrap(errs.CodeInternal, "select for escort", err)
    }
    return nil
}

// ConfirmAccept 陪诊师 30s 内 confirm → accepted；escort_id 写入最终字段。
//
// 校验：
//   - DB 已 SelectForEscort（service.SelectEscort 已写过 selected_escort_id）
//   - selected_escort_id = caller
//   - escort_pending_expire_at >= NOW()（超时由 scanner 处理）
//
// 副作用：
//   - status = 'accepted' + escort_id = escortID
//   - 清空 selected_escort_id + escort_pending_expire_at
//   - 释放 Redis SETNX（best-effort）
//   - 发 OrderEscortConfirmedEvent
func (s *Service) ConfirmAccept(ctx context.Context, orderID, escortID int64) (*repo.Order, error) {
    if orderID == 0 || escortID == 0 {
        return nil, errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
    }
    o, err := s.orders.FindByID(ctx, orderID)
    if err != nil {
        return nil, errs.Wrap(errs.CodeInternal, "find order", err)
    }
    if state.Status(o.Status) != state.StatusEscortPendingAcceptance {
        return nil, errs.New(errs.CodeConflict, fmt.Sprintf("cannot confirm from status %s", o.Status))
    }
    if err := s.orders.ConfirmByEscort(ctx, orderID, escortID, o.Version); err != nil {
        switch {
        case errors.Is(err, repo.ErrSelectedEscortMismatch):
            return nil, errs.New(errs.CodeForbidden, "selected escort mismatch")
        case errors.Is(err, repo.ErrVersionConflict):
            return nil, ErrVersionConflict
        case errors.Is(err, repo.ErrInvalidStateForSelect):
            return nil, errs.New(errs.CodeConflict, "order already expired or transitioned")
        }
        return nil, errs.Wrap(errs.CodeInternal, "confirm by escort", err)
    }
    // 写 order_event
    from := string(state.StatusEscortPendingAcceptance)
    to := string(state.StatusAccepted)
    actor := escortID
    if err := s.orders.InsertEvent(ctx, orderID, &from, to, &actor, nil); err != nil {
        return nil, errs.Wrap(errs.CodeInternal, "insert event", err)
    }
    // 释放 Redis SETNX（best-effort）
    s.releaseConfirmLock(ctx, orderID)
    // 发 OrderEscortConfirmedEvent
    if s.publisher != nil {
        if err := s.publisher.PublishOrderEscortConfirmed(ctx, contracts.OrderEscortConfirmedEvent{
            OrderID:     orderID,
            EscortID:    escortID,
            ConfirmedAt: s.clockNow(),
        }); err != nil {
            // best-effort：仅 log
            _ = err
        }
    }
    return s.orders.FindByID(ctx, orderID)
}

// RejectAccept 陪诊师拒接 → 回退 selecting_escort。
//
// 校验：
//   - status = 'escort_pending_acceptance'
//   - selected_escort_id = caller
//
// 副作用：
//   - status = 'selecting_escort'
//   - 清空 selected_escort_id + escort_pending_expire_at
//   - 释放 Redis SETNX（best-effort）
//   - 发 OrderEscortRejectedEvent(reason='escort_declined')
func (s *Service) RejectAccept(ctx context.Context, orderID, escortID int64) error {
    if orderID == 0 || escortID == 0 {
        return errs.New(errs.CodeParamInvalid, "order_id / escort_id required")
    }
    o, err := s.orders.FindByID(ctx, orderID)
    if err != nil {
        return errs.Wrap(errs.CodeInternal, "find order", err)
    }
    if state.Status(o.Status) != state.StatusEscortPendingAcceptance {
        return errs.New(errs.CodeConflict, fmt.Sprintf("cannot reject from status %s", o.Status))
    }
    if err := s.orders.RejectByEscort(ctx, orderID, escortID, o.Version); err != nil {
        switch {
        case errors.Is(err, repo.ErrSelectedEscortMismatch):
            return errs.New(errs.CodeForbidden, "selected escort mismatch")
        case errors.Is(err, repo.ErrVersionConflict):
            return ErrVersionConflict
        }
        return errs.Wrap(errs.CodeInternal, "reject by escort", err)
    }
    // 写 order_event
    from := string(state.StatusEscortPendingAcceptance)
    to := string(state.StatusSelectingEscort)
    actor := escortID
    if err := s.orders.InsertEvent(ctx, orderID, &from, to, &actor, nil); err != nil {
        return errs.Wrap(errs.CodeInternal, "insert event", err)
    }
    // 释放 Redis SETNX（best-effort）
    s.releaseConfirmLock(ctx, orderID)
    // 发 OrderEscortRejectedEvent(reason='escort_declined')
    if s.publisher != nil {
        if err := s.publisher.PublishOrderEscortRejected(ctx, contracts.OrderEscortRejectedEvent{
            OrderID:    orderID,
            EscortID:   escortID,
            Reason:     "escort_declined",
            RejectedAt: s.clockNow(),
        }); err != nil {
            _ = err
        }
    }
    return nil
}

// releaseConfirmLock 释放 Redis SETNX orders:confirm:{order_id}（best-effort）。
// token 简化：random；不命中靠 TTL 兜底。
func (s *Service) releaseConfirmLock(ctx context.Context, orderID int64) {
    tokenBytes := make([]byte, 16)
    _, _ = rand.Read(tokenBytes)
    key := fmt.Sprintf("orders:confirm:%d", orderID)
    _ = s.locker.Release(ctx, key, hex.EncodeToString(tokenBytes))
}

// 编译期确保 repo.Order 与 service.OrderRepo 兼容。
var _ OrderRepo = (*repo.OrderRepo)(nil)

// 编译期确保 events.Publisher 接口实现完整。
var _ events.Publisher = (*events.KafkaPublisher)(nil)
```

**Step 4: 跑测试确认通过**

Run: `go test -tags=integration -run 'TestSelectEscort|TestConfirmAccept|TestRejectAccept' ./services/order/internal/service/`
Expected: PASS

**Step 5: 删旧测试 + Commit**

```bash
# 删 v1 抢单锁单集成测试
rm services/order/internal/service/accept_integration_test.go
# 用 Task 4 Step 1 的新测试代码重写（已包含全部新场景）
git add services/order/internal/service/
git commit -m "feat(service): 删 v1 Accept/TryLock；加 SelectEscort/ConfirmAccept/RejectAccept + Redis SETNX orders:confirm:{id} (10 个集成测试)"
```

---

### Task 5: EscortPendingScanner（沿用 ExpiredLockScanner 文件名 + 改语义）

**Files:**
- Modify: `services/order/internal/scheduler/expired_lock_scanner.go`
- Modify: `services/order/internal/scheduler/expired_lock_scanner_test.go`

**Step 1: 写新语义的失败测试**

> 注：删旧 `TestScanner_*` 测试代码（v1 用 LockOwner / LockExpireAt / PublishOrderMatching）；换为新流程的 `PendingExpired` + `RejectAccept` + `OrderEscortRejectedEvent`。

`expired_lock_scanner_test.go`：

```go
package scheduler

import (
    "context"
    "sync"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/growdu/doctors/services/order/internal/repo"
    "github.com/growdu/doctors/services/order/internal/state"
    "github.com/growdu/doctors/shared/contracts"
)

// fakeRepo 提供最小仓储契约（只实现 PendingExpired）。
type fakeRepo struct {
    mu     sync.Mutex
    orders map[int64]*repo.Order
}

func newFakeRepo() *fakeRepo {
    return &fakeRepo{orders: map[int64]*repo.Order{}}
}

func (f *fakeRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    out := make([]*repo.Order, 0)
    for _, o := range f.orders {
        if o.Status != string(state.StatusEscortPendingAcceptance) || o.EscortPendingExpireAt == nil {
            continue
        }
        if o.EscortPendingExpireAt.Before(now) {
            out = append(out, o)
        }
        if len(out) >= limit {
            break
        }
    }
    return out, nil
}

// fakePublisher 记录最近一次发布的 OrderEscortRejectedEvent。
type fakePublisher struct {
    mu     sync.Mutex
    last   *contracts.OrderEscortRejectedEvent
}

func (f *fakePublisher) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
    return nil
}
func (f *fakePublisher) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
    return nil
}
func (f *fakePublisher) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
    return nil
}
func (f *fakePublisher) PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error {
    return nil
}
func (f *fakePublisher) PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error {
    return nil
}
func (f *fakePublisher) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    cp := ev
    f.last = &cp
    return nil
}
func (f *fakePublisher) Close() error { return nil }

// fakeRejectSvc 记录被调用的 orderID + escortID。
type fakeRejectSvc struct {
    mu     sync.Mutex
    called []int64
    escorts []int64
}

func (f *fakeRejectSvc) RejectAccept(ctx context.Context, orderID, escortID int64) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.called = append(f.called, orderID)
    f.escorts = append(f.escorts, escortID)
    return nil
}

// TestScanner_PublishesEscortRejectedOnExpire 验证扫描超时 escort_pending_acceptance 单后发 OrderEscortRejectedEvent。
func TestScanner_PublishesEscortRejectedOnExpire(t *testing.T) {
    expire := time.Now().Add(-1 * time.Hour)
    selectedID := int64(7)
    fr := newFakeRepo()
    fr.orders[100] = &repo.Order{
        ID:                    100,
        Status:                string(state.StatusEscortPendingAcceptance),
        SelectedEscortID:      &selectedID,
        EscortPendingExpireAt: &expire,
    }
    pub := &fakePublisher{}
    svc := &fakeRejectSvc{}
    s := NewExpiredLockScanner(fr, svc, pub, time.Second)
    s.ScanOnce(context.Background())

    require.NotNil(t, pub.last, "应发布 OrderEscortRejectedEvent")
    assert.Equal(t, int64(100), pub.last.OrderID)
    assert.Equal(t, int64(7), pub.last.EscortID)
    assert.Equal(t, "lock_expired", pub.last.Reason, "scanner 超时 → reason=lock_expired")

    require.Len(t, svc.called, 1, "RejectAccept 应被调用一次")
    assert.Equal(t, int64(100), svc.called[0])
    assert.Equal(t, int64(7), svc.escorts[0])
}

// TestScanner_ScanOnce_SkipsNonExpired 验证未超时的 escort_pending_acceptance 不被处理。
func TestScanner_ScanOnce_SkipsNonExpired(t *testing.T) {
    future := time.Now().Add(time.Hour)
    selectedID := int64(7)
    fr := newFakeRepo()
    fr.orders[200] = &repo.Order{
        ID:                    200,
        Status:                string(state.StatusEscortPendingAcceptance),
        SelectedEscortID:      &selectedID,
        EscortPendingExpireAt: &future,
    }
    pub := &fakePublisher{}
    svc := &fakeRejectSvc{}
    s := NewExpiredLockScanner(fr, svc, pub, time.Second)
    s.ScanOnce(context.Background())

    assert.Nil(t, pub.last, "未超时不应发布")
    assert.Empty(t, svc.called)
}

// TestScanner_ScanOnce_SkipsNilSelectedEscort 验证 selected_escort_id=nil 时跳过（异常保护）。
func TestScanner_ScanOnce_SkipsNilSelectedEscort(t *testing.T) {
    expire := time.Now().Add(-1 * time.Hour)
    fr := newFakeRepo()
    fr.orders[300] = &repo.Order{
        ID:                    300,
        Status:                string(state.StatusEscortPendingAcceptance),
        SelectedEscortID:      nil, // 异常：DB 状态不一致
        EscortPendingExpireAt: &expire,
    }
    pub := &fakePublisher{}
    svc := &fakeRejectSvc{}
    s := NewExpiredLockScanner(fr, svc, pub, time.Second)
    s.ScanOnce(context.Background())

    assert.Nil(t, pub.last)
    assert.Empty(t, svc.called)
}

// TestScanner_Run_RespectsCtxCancel 验证 Run 在 ctx 取消后退出。
func TestScanner_Run_RespectsCtxCancel(t *testing.T) {
    fr := newFakeRepo()
    pub := &fakePublisher{}
    svc := &fakeRejectSvc{}
    s := NewExpiredLockScanner(fr, svc, pub, 50*time.Millisecond)

    ctx, cancel := context.WithCancel(context.Background())
    done := make(chan struct{})
    go func() {
        s.Run(ctx)
        close(done)
    }()
    time.Sleep(100 * time.Millisecond)
    cancel()
    select {
    case <-done:
        // OK
    case <-time.After(2 * time.Second):
        t.Fatal("Run did not return within 2s after ctx cancel")
    }
}
```

**Step 2: 跑测试确认失败**

Run: `go test -count=1 -v ./services/order/internal/scheduler/`
Expected: FAIL — `undefined: (*ExpiredLockScanner).RepoLockExpired`（接口签名不一致）/ `undefined: fakePublisher.PublishOrderSelectingEscort`（接口不一致）

**Step 3: 改写 expired_lock_scanner.go**

```go
// Package scheduler 跑后台定时任务。
//
// 设计要点：
//   - 每个 scheduler 都是 Run(ctx) 阻塞循环；ctx cancel 即退出。
//   - 用接口注入 repo / publisher / service；scheduler 只关心"扫 + 通知"，
//     不直接操作 DB（与业务 service 一致）。
//   - 失败 best-effort：单条订单处理失败不影响下一条。
//
// 2026-09-24-order-matching-redesign：
//   - ExpiredLockScanner 保留文件名（避免无意义 churn）。
//   - 扫描条件：status='escort_pending_acceptance' AND escort_pending_expire_at < NOW()
//   - 动作：调 service.RejectAccept 回退 selecting_escort + 发 OrderEscortRejectedEvent(reason='lock_expired')
//   - 间隔：5s（保留）
package scheduler

import (
    "context"
    "log"
    "time"

    "github.com/growdu/doctors/services/order/internal/events"
    "github.com/growdu/doctors/services/order/internal/repo"
    "github.com/growdu/doctors/shared/contracts"
)

// RepoPendingExpired 是 scheduler 调用的最小仓储契约。
type RepoPendingExpired interface {
    PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error)
}

// RejectService 是 service 暴露给 scheduler 的最小契约。
type RejectService interface {
    RejectAccept(ctx context.Context, orderID, escortID int64) error
}

// ExpiredLockScanner 每 interval 秒扫一次 escort_pending_expromise 超时单，
// 自动调 RejectAccept 回退 selecting_escort 并发 OrderEscortRejectedEvent(reason='lock_expired')。
//
// 文件名沿用 "expired_lock_scanner.go" 以避免 churn；类型名也是 ExpiredLockScanner
// （在 order-lock v1 后已成为本服务的固定组件）。
type ExpiredLockScanner struct {
    repo      RepoPendingExpired
    svc       RejectService
    publisher events.Publisher
    interval  time.Duration
    limit     int
}

// NewExpiredLockScanner 构造 scanner；interval <= 0 默认 5s。
func NewExpiredLockScanner(r RepoPendingExpired, s RejectService, p events.Publisher, interval time.Duration) *ExpiredLockScanner {
    if interval <= 0 {
        interval = 5 * time.Second
    }
    return &ExpiredLockScanner{
        repo:      r,
        svc:       s,
        publisher: p,
        interval:  interval,
        limit:     50,
    }
}

// Run 阻塞扫描直到 ctx 取消。
func (s *ExpiredLockScanner) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.ScanOnce(ctx)
        }
    }
}

// ScanOnce 扫描一次；可单测。
func (s *ExpiredLockScanner) ScanOnce(ctx context.Context) {
    expired, err := s.repo.PendingExpired(ctx, time.Now(), s.limit)
    if err != nil {
        log.Printf("[expired-lock-scanner] scan failed: %v", err)
        return
    }
    for _, o := range expired {
        // selected_escort_id 为 nil 视为异常（DB 状态不一致）；跳过
        if o.SelectedEscortID == nil {
            log.Printf("[expired-lock-scanner] order %d has nil selected_escort_id, skip", o.ID)
            continue
        }
        // 用 selected_escort_id 作为 escortID 调用 RejectAccept
        if err := s.svc.RejectAccept(ctx, o.ID, *o.SelectedEscortID); err != nil {
            log.Printf("[expired-lock-scanner] reject %d failed: %v", o.ID, err)
            continue
        }
        // 发 OrderEscortRejectedEvent(reason='lock_expired')（best-effort，失败仅 log）
        if s.publisher != nil {
            if err := s.publisher.PublishOrderEscortRejected(ctx, contracts.OrderEscortRejectedEvent{
                OrderID:    o.ID,
                EscortID:   *o.SelectedEscortID,
                Reason:     "lock_expired",
                RejectedAt: time.Now(),
            }); err != nil {
                log.Printf("[expired-lock-scanner] publish %d failed: %v", o.ID, err)
            }
        }
        log.Printf("[expired-lock-scanner] order %d lock_expired, reverted to selecting_escort", o.ID)
    }
}
```

**Step 4: 跑测试确认通过**

Run: `go test -count=1 -v ./services/order/internal/scheduler/`
Expected: PASS（4 个测试）

**Step 5: Commit**

```bash
git add services/order/internal/scheduler/
git commit -m "feat(scheduler): ExpiredLockScanner 改扫 escort_pending_expire_at + 发 OrderEscortRejectedEvent(reason='lock_expired') (4 个测试)"
```

---

### Task 6: main.go 装配 + 文档同步

**Files:**
- Modify: `services/order/cmd/main.go`
- Modify: `docs/04-业务流程.md` §4.3
- Modify: `docs/07-数据模型.md`
- Modify: `dev.md`

**Step 1: 改 main.go 装配**

```go
// services/order/cmd/main.go 装配（替换旧的 OrderRepo / scanner 注入）
pool, err := db.NewPool(ctx, db.Config{DSN: cfg.DB.DSN})
if err != nil { log.Fatal(err) }
orderRepo := repo.NewOrderRepo(pool)

rdb, err := sharedRedis.NewClient(cfg.Redis)
if err != nil { log.Fatal(err) }
locker := sharedLock.NewRedisLocker(rdb)

pub := events.NewKafkaPublisher(cfg.Kafka.Brokers)

svc := service.New(orderRepo, unwiredUsers{}).
    WithTx(&service.PGPoolTxRunner{Pool: pool}).
    WithPublisher(pub).
    WithLocker(locker).
    WithClock(func() time.Time { return time.Now() })

// 启动 5s 一次的 escort_pending_acceptance 超时扫描
go scheduler.NewExpiredLockScanner(orderRepo, svc, pub, 5*time.Second).Run(ctx)

srv := server.New(cfg.HTTP.Addr, handler.New(svc), cfg.Auth.JWTSecret)
srv.Run(ctx)
```

> 注意：scanner 注入的是 `*service.Service`（满足 `RejectService` 接口）；`orderRepo` 满足 `RepoPendingExpired` 接口（`*repo.OrderRepo` 已有 `PendingExpired` 方法）。

**Step 2: 跑全量回归**

Run: `go build ./... && go test -count=1 ./shared/... ./services/order/...`
Expected: PASS（无编译错误，所有测试通过）

**Step 3: 改 docs/04 §4.3（替换抢单流程为选 escort + 30s 确认流程）**

```markdown
### §4.3 选陪诊师 + 陪诊师 30s 确认流程（2026-09-24 重构）

陪诊师不再"抢单"，改为"陪诊师设置空余时段 → 患者从候选列表选 → 陪诊师 30s 内 confirm/reject"。

```
患者：创建订单（service_start_at）→ paid
     → 系统生成候选 escort 列表（status: selecting_escort）
     → 从候选列表选 1 位（POST /orders/{id}/select-escort body: {escort_id}）
     → 状态切 escort_pending_acceptance；selected_escort_id + escort_pending_expire_at（= now + 30s）

陪诊师：收到邀请（OrderEscortSelectedEvent via Kafka / 推送）→ 30s 内 confirm 或 reject
       → confirm：POST /orders/{id}/confirm-accept → accepted；escort_id（最终）写入
       → reject：POST /orders/{id}/reject-accept → 回退 selecting_escort
       → 超时：scanner 自动 RejectAccept → selecting_escort

订单：paid → selecting_escort → escort_pending_acceptance → accepted
                                                  ↘ selecting_escort（拒 / 超时）
```

锁单三道防线（SelectEscort 入口）：

1. Redis SETNX orders:confirm:{order_id} 30s（防止 select-escort 重复触发；NopLocker 跳过）
2. SELECT orders WHERE id=$1 AND version=$2（验证 status='selecting_escort' + version）
3. UPDATE orders SET status='escort_pending_acceptance', selected_escort_id=$1, escort_pending_expire_at=NOW()+30s

超时回退（scanner）：

- ExpiredLockScanner 5s 扫一次 `status='escort_pending_acceptance' AND escort_pending_expire_at < NOW()`
- 对每条命中单：调 service.RejectAccept → DB 回退 selecting_escort + 发 OrderEscortRejectedEvent(reason='lock_expired')
```

**Step 4: 改 docs/07-数据模型.md**

```markdown
### orders 表（0009 迁移修订）

字段（关键）：

| 字段 | 类型 | 说明 |
|------|------|------|
| status | VARCHAR(32) | 状态枚举（CHECK）：created / paid / matching / selecting_escort / escort_pending_acceptance / accepted / in_service / completed / reviewed / refunding / refunded / settling / disputed / closed / canceled |
| version | INT | 乐观锁 |
| escort_id | BIGINT NULL | 最终接单陪诊师（accepted 时填） |
| selected_escort_id | BIGINT NULL | 患者选的陪诊师（escort_pending_acceptance 时填；confirm → 移到 escort_id；reject → 清空） |
| escort_pending_expire_at | TIMESTAMPTZ NULL | 30s 超时时间（scanner 扫这个） |

CHECK 约束：

```sql
CHECK (status IN (
  'created','paid','matching','selecting_escort','escort_pending_acceptance','accepted',
  'in_service','completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
))
```

索引：

```sql
CREATE INDEX idx_orders_selecting ON orders(service_start_at)
  WHERE status IN ('selecting_escort','escort_pending_acceptance');
```

> 注：0003 加的 `lock_owner` / `lock_expire_at` / `idx_orders_lock` 在 0009 中被删除。
```

**Step 5: dev.md 追加**

```markdown
| 3.11 | 陪诊师确认锁单 30s (select-escort + confirm-accept + reject-accept + Redis SETNX orders:confirm + scanner) | ~6 commits | `feat(service): 删 v1 Accept；加 SelectEscort/ConfirmAccept/RejectAccept` |
```

**Step 6: Commit**

```bash
git add services/order/cmd/ docs/04-业务流程.md docs/07-数据模型.md dev.md
git commit -m "docs(order): main 装配 + 04 §4.3 改选 escort 流程 + 07 数据模型同步 + dev.md 落地"
```

---

## Self-Review

- ✅ Spec 覆盖：选 escort + 30s 陪诊师确认锁单全链路落地（spec §1.2 / §5.1 / §7.1 / §7.3）。
- ✅ 删的：v1 抢单锁单 `Accept` / `TryLock` / `ReleaseAcceptLock` / `LockForAccept` / `ReleaseLock` / `LockExpired` / `OrderMatchingEvent` / `PublishOrderMatching` / `lock_owner` / `lock_expire_at`。
- ✅ 加的：3 个新事件（`OrderSelectingEscortEvent` / `OrderEscortConfirmedEvent` / `OrderEscortRejectedEvent`）+ 3 个 Publisher 方法；3 个新 service 方法（`SelectEscort` / `ConfirmAccept` / `RejectAccept`）+ 4 个 repo 方法（`SelectForEscort` / `ConfirmByEscort` / `RejectByEscort` / `PendingExpired`）；Redis SETNX `orders:confirm:{order_id}` 30s。
- ✅ Scanner 语义变更：扫 `escort_pending_expire_at` + 发 `OrderEscortRejectedEvent(reason='lock_expired')`；文件名沿用 `expired_lock_scanner.go`（避免 churn）。
- ✅ 类型一致：`ErrLockTaken` 复用（语义改为"select-escort 重复触发"）；`ErrSelectedEscortMismatch` 新增；`ErrInvalidStateForSelect` 重命名复用；`state.StatusSelectingEscort` / `StatusEscortPendingAcceptance` 假定 state-machine 修订版已交付。
- ✅ 测试矩阵：Task 1（contracts 4）+ Task 2（publisher 3）+ Task 3（repo 集成 7）+ Task 4（service 集成 10）+ Task 5（scanner 4）+ Task 6（编译 + 全量回归）。
- ✅ YAGNI：service 层不校验 escort 可用性（留 escort-business 计划）；token 简化处理（best-effort + TTL 兜底）；selected escort availability 校验留给 escort-service。

## Assumptions

- `2026-09-24-state-machine.md`（修订版）已先 commit —— 加 `StatusSelectingEscort` / `StatusEscortPendingAcceptance`、transitions 更新、0009 迁移（删 `lock_owner` / `lock_expire_at`，加 `selected_escort_id` / `escort_pending_expire_at`）。本 plan 不重复落地，假定上游已交付。
- `escort-business` / `escort-order-ext` plans 也已先 commit —— 加 `GET /api/v1/escorts/{id}/availabilities` + escort 端 `confirm-accept` / `reject-accept` + patient 端 `select-escort`。本 plan 只关注 order-service 内部实现，HTTP handler 由 `escort-order-ext` plan 接入。
- `shared/lock.RedisLocker` / `shared/lock.NopLocker` 已是 v1 order-lock plan 落地状态，本 plan 直接复用。
- v1 `OrderMatchingEvent` 相关旧测试（如 `TestOrderMatchingEvent_RoundTrip`）由 state-machine 修订版 plan 负责删除，本 plan 不重复。

## 执行选项

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-order-lock.md`。
> 仍在 plan_all 模式，**不自动执行**。

**下一步**：
1. **继续 plan_all**：立即产出下一份 plan（escort-order-ext：拆 select-escort / confirm-accept / reject-accept handler）
2. **暂停 plan**：你 review 此 plan 后告诉我调整，或暂停出新 plan