# order-service escort 视角扩展 plan (v2：选人模式)

> Spec：`docs/superpowers/specs/2026-09-24-order-matching-redesign.md` §1.2 + §4.1
> Revision history:
> - v1 (2026-09-24)：基于「匹配池+抢单」模型，含 `Accept` + `lock_owner`。
> - v2 (2026-09-24)：切换为「患者选人」模型，删除 `Accept` + `lock_owner`，新增 `SelectEscort` / `ConfirmAccept` / `RejectAccept` 三步流程。配套 state-machine plan (commit 3860cab) 与 order-lock plan (commit bg_e38a70e2) 同步生效。

## Header

### Goal

实现 order-service 中"患者从候选 escort 中选定一人，被邀请方在 30 秒窗口内确认/拒绝"的核心交互链。替换 v1 的"匹配池+抢单 Accept"流程，使订单状态机进入 `selecting_escort` → `escort_pending_acceptance` → `accepted`（或回退）。配套新增四个领域事件、三个 HTTP 端点、两个 service 方法（service 侧另有 `RejectAccept`），并接入 Redis `orders:confirm:{order_id}` SETNX 30s 邀请有效期。

### Architecture

- **DDD 分层**：handler → service → repository，保持现有边界。
- **状态机**：由 state-machine plan (3860cab) 提供，本 plan 仅消费 `selecting_escort` / `escort_pending_acceptance` 两个新状态。
- **邀请有效期**：Redis SETNX key `orders:confirm:{order_id}`，TTL 30s，值 `selected_escort_id`。`ConfirmAccept` 必须先 GET 命中且未过期；`RejectAccept` 不消费 TTL，回退到 `selecting_escort` 即清 key。过期由 scheduler 异步检测并强制回退。
- **依赖接口**：`CandidatesLookup`（按 orderID 返 escortID 列表）与 `AvailabilityLookup`（按 escortID + service_start_at 时间窗返 bool）。两接口由 escort-service 经 gRPC 提供，本 plan 在 `service/port.go` 中定义抽象。
- **事件总线**：在 `OrderCreated` 之后由匹配模块产出 `OrderCandidatesReadyEvent`；`OrderEscortInvitedEvent` / `OrderEscortConfirmedEvent` / `OrderEscortRejectedEvent` 由本 plan 新增。

### Tech Stack

- Go 1.22+ / internal 模块
- Redis 7 (订单邀请锁与过期扫描)
- PostgreSQL (订单持久化 + outbox)
- testify (单测) + testcontainers-go (集成)

### 前置依赖

- state-machine plan 已合并（commit `3860cab`）：状态 `selecting_escort` / `escort_pending_acceptance` 已注册。
- order-lock plan 已合并（commit `bg_e38a70e2`）：`lock_owner` / `lock_expire_at` 字段已删除；`selected_escort_id` / `escort_pending_expire_at` 字段已迁移；`orders:confirm:{order_id}` Redis SETNX 工具方法 `acquireConfirmLock` / `releaseConfirmLock` 已就绪。
- matching 模块能产出 `OrderCandidatesReadyEvent`（独立 plan，不在本 plan 范围）。

## Global Constraints

- 每个 task 必须包含 RED → GREEN → Commit 三步；RED 阶段单测必须先失败。
- 单测必须可独立运行：`go test ./internal/order/service/... -run TestXxx`；集成测试：`go test ./test/integration/... -tags=integration`。
- 任何修改现存文件的 task 必须保留已有 v1 review 关注的 checkin/checkout 路径（`/orders/:id/checkin`、 `/orders/:id/checkout`）与字段 `0006_orders_checkin` 不动。
- 所有时间字段统一 UTC；service 层不直接 `time.Now()`，全部通过 `Clock` 注入，便于单测。
- 错误一律返回 sentinel（`ErrNotInCandidates` / `ErrNotAvailable` / `ErrNotSelected` / `ErrInvitationExpired`），handler 层翻译为 HTTP 4xx。
- 禁止使用 `panic` 处理业务错误；禁止在 service 内直接调用 DB driver（必须经 repository）。
- 不在本 plan 修改：state-machine 表（已在 3860cab）、order-lock 工具方法（已在 bg_e38a70e2）、matching 模块（独立 plan）。
- 每个 commit 信息遵循 Conventional Commits；plan 自身 commit 单独一条 `docs(plan): ...`。

## File Structure

| 路径 | Mode | 职责 |
|---|---|---|
| `shared/contracts/events.go` | Modify | 删除 `OrderMatchingEvent`；新增 `OrderCandidatesReadyEvent` / `OrderEscortInvitedEvent` / `OrderEscortConfirmedEvent` / `OrderEscortRejectedEvent` |
| `internal/order/service/port.go` | Modify | 新增 `CandidatesLookup` / `AvailabilityLookup` 接口 |
| `internal/order/service/escort_selection.go` | Create | 实现 `SelectEscort` / `ConfirmAccept` / `RejectAccept` |
| `internal/order/service/errors.go` | Modify | 新增 `ErrNotInCandidates` / `ErrNotAvailable` / `ErrNotSelected` / `ErrInvitationExpired` |
| `internal/order/repository/order_repo.go` | Modify | 新增 `UpdateSelectedEscort` / `ClearSelectedEscort` / `MarkEscortConfirmed`（涉及字段 `selected_escort_id` / `escort_pending_expire_at` / `escort_id`，由 order-lock plan 提供迁移） |
| `internal/order/handler/order_handler.go` | Modify | 删除 `Accept`；新增 `SelectEscort` / `ConfirmAccept` / `RejectAccept` handler 方法 |
| `internal/order/handler/router.go` | Modify | 路由注册：删除 `POST /orders/:id/accept`，新增 `POST /orders/:id/select-escort`、`POST /orders/:id/confirm-accept`、`POST /orders/:id/reject-accept` |
| `internal/order/handler/list_filter.go` | Modify | `status=invitations` 过滤改为 `escort_pending_acceptance`；保留原有 `pending` / `accepted` / `in_service` 等 |
| `cmd/order-service/main.go` | Modify | 装配 `CandidatesLookup` / `AvailabilityLookup`（默认实现调 escort-service gRPC stub），注入 `EscortSelectionService` |
| `internal/scheduler/escort_invite_expiry.go` | Create | 每 5s 扫描 `escort_pending_expire_at < now()` 的订单，强制回退到 `selecting_escort` 并发 `OrderEscortRejectedEvent(reason=timeout)` |
| `test/integration/escort_selection_e2e_test.go` | Create | 端到端：选人 → 邀请 → 确认；选人拒收回退；邀请超时回退 |

---

## Task 1: shared/contracts/events.go 事件定义迁移

### 目标

事件层从"匹配池广播"切到"定向邀请"。删除 `OrderMatchingEvent`，新增四个定向事件。

### Step 1.1 RED — 写测试

新建 `shared/contracts/events_test.go`：

```go
package contracts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventNames_Registration(t *testing.T) {
	// 必须存在的四个事件名
	for _, name := range []string{
		"order.candidates.ready",
		"order.escort.invited",
		"order.escort.confirmed",
		"order.escort.rejected",
	} {
		_, ok := EventRegistry[name]
		assert.Truef(t, ok, "event %s must be registered", name)
	}
}

func TestOrderMatchingEvent_Removed(t *testing.T) {
	_, ok := EventRegistry["order.matching"]
	assert.False(t, ok, "OrderMatchingEvent must be removed in v2")
}

func TestOrderEscortInvitedEvent_PayloadShape(t *testing.T) {
	ev := OrderEscortInvitedEvent{
		OrderID:      "ord_123",
		PatientID:    "pat_1",
		EscortID:     "esc_2",
		ExpiresAt:    time.Now().Add(30 * time.Second),
		ServiceStart: time.Now().Add(2 * time.Hour),
	}
	assert.Equal(t, "order.escort.invited", ev.Name())
	assert.NotEmpty(t, ev.SchemaVersion())
	assert.NotZero(t, ev.ExpiresAt)
}
```

Run:

```
go test ./shared/contracts/... -run TestEventNames_Registration
```

Expected: FAIL — `EventRegistry` 不存在；`OrderEscortInvitedEvent` 未定义。

### Step 1.2 GREEN — 实现

修改 `shared/contracts/events.go`：

```go
package contracts

import "time"

// EventRegistry 全局事件名→类型映射，由各事件 init() 注册。
var EventRegistry = map[string]Event{}

// Event 最小事件接口
type Event interface {
	Name() string
	SchemaVersion() string
}

// === 新增事件 ===

type OrderCandidatesReadyEvent struct {
	OrderID      string    `json:"order_id"`
	PatientID    string    `json:"patient_id"`
	CandidateIDs []string  `json:"candidate_ids"`
	GeneratedAt  time.Time `json:"generated_at"`
}

func (OrderCandidatesReadyEvent) Name() string           { return "order.candidates.ready" }
func (OrderCandidatesReadyEvent) SchemaVersion() string  { return "1.0" }
func init() { EventRegistry["order.candidates.ready"] = OrderCandidatesReadyEvent{} }

type OrderEscortInvitedEvent struct {
	OrderID      string    `json:"order_id"`
	PatientID    string    `json:"patient_id"`
	EscortID     string    `json:"escort_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	ServiceStart time.Time `json:"service_start_at"`
}

func (OrderEscortInvitedEvent) Name() string          { return "order.escort.invited" }
func (OrderEscortInvitedEvent) SchemaVersion() string { return "1.0" }
func init() { EventRegistry["order.escort.invited"] = OrderEscortInvitedEvent{} }

type OrderEscortConfirmedEvent struct {
	OrderID   string    `json:"order_id"`
	PatientID string    `json:"patient_id"`
	EscortID  string    `json:"escort_id"`
	AcceptedAt time.Time `json:"accepted_at"`
}

func (OrderEscortConfirmedEvent) Name() string          { return "order.escort.confirmed" }
func (OrderEscortConfirmedEvent) SchemaVersion() string { return "1.0" }
func init() { EventRegistry["order.escort.confirmed"] = OrderEscortConfirmedEvent{} }

type OrderEscortRejectedEvent struct {
	OrderID   string    `json:"order_id"`
	PatientID string    `json:"patient_id"`
	EscortID  string    `json:"escort_id"`
	Reason    string    `json:"reason"` // "patient_rejected" | "timeout" | "explicit_decline"
	RejectedAt time.Time `json:"rejected_at"`
}

func (OrderEscortRejectedEvent) Name() string          { return "order.escort.rejected" }
func (OrderEscortRejectedEvent) SchemaVersion() string { return "1.0" }
func init() { EventRegistry["order.escort.rejected"] = OrderEscortRejectedEvent{} }

// === 删除 ===
// OrderMatchingEvent 整段删除；迁移指南见 docs/migrations/v1-to-v2-events.md
```

Run:

```
go test ./shared/contracts/...
```

Expected: PASS。

### Step 1.3 Commit

```
git add shared/contracts/events.go shared/contracts/events_test.go
git commit -m "feat(events): replace OrderMatchingEvent with directed escort invitation events

- Remove OrderMatchingEvent (replaced by OrderCandidatesReadyEvent)
- Add OrderEscortInvitedEvent / OrderEscortConfirmedEvent / OrderEscortRejectedEvent
- EventRegistry now exposes 4 v2 events with schema version 1.0"
```

---

## Task 2: service.SelectEscort / ConfirmAccept / RejectAccept 三步实现

### 目标

在 service 层完成"选人→邀请→确认/拒绝"完整业务逻辑，配套 sentinel 错误与仓储更新。

### Step 2.1 RED — service 单测先写

新建 `internal/order/service/escort_selection_test.go`：

```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCandidatesLookup struct{ mock.Mock }
func (c MockCandidatesLookup) List(ctx context.Context, orderID string) ([]string, error) {
	args := c.Called(ctx, orderID)
	return args.Get(0).([]string), args.Error(1)
}
type MockAvailabilityLookup struct{ mock.Mock }
func (a MockAvailabilityLookup) IsAvailable(ctx context.Context, escortID string, at time.Time) (bool, error) {
	args := a.Called(ctx, escortID, at)
	return args.Bool(0), args.Error(1)
}

func TestSelectEscort_RejectsEscortNotInCandidates(t *testing.T) {
	svc := newTestService(TestDeps{Candidates: MockCandidatesLookup{}})
	svc.candidates.On("List", mock.Anything, "ord_1").Return([]string{"esc_a"}, nil)
	err := svc.SelectEscort(context.Background(), "ord_1", "pat_1", "esc_z", time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrNotInCandidates)
}

func TestSelectEscort_RejectsUnavailableEscort(t *testing.T) {
	deps := TestDeps{
		Candidates: MockCandidatesLookup{},
		Avail:      MockAvailabilityLookup{},
	}
	deps.Candidates.On("List", mock.Anything, "ord_1").Return([]string{"esc_a"}, nil)
	deps.Avail.On("IsAvailable", mock.Anything, "esc_a", mock.Anything).Return(false, nil)
	svc := newTestService(deps)
	err := svc.SelectEscort(context.Background(), "ord_1", "pat_1", "esc_a", time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrNotAvailable)
}

func TestConfirmAccept_RejectsNotSelectedEscort(t *testing.T) {
	svc := newTestService(TestDeps{})
	_, err := svc.ConfirmAccept(context.Background(), "ord_1", "esc_x")
	assert.ErrorIs(t, err, ErrNotSelected)
}

func TestConfirmAccept_RejectsExpired(t *testing.T) {
	svc := newTestService(TestDeps{Now: func() time.Time { return time.Unix(200, 0) }})
	svc.selectedEscort = "esc_a"
	svc.selectedExpireAt = time.Unix(100, 0)
	_, err := svc.ConfirmAccept(context.Background(), "ord_1", "esc_a")
	assert.ErrorIs(t, err, ErrInvitationExpired)
}

func TestRejectAccept_FallsBackToSelectingEscort(t *testing.T) {
	svc := newTestService(TestDeps{})
	svc.selectedEscort = "esc_a"
	err := svc.RejectAccept(context.Background(), "ord_1", "esc_a", "explicit_decline")
	assert.NoError(t, err)
	assert.Equal(t, "selecting_escort", svc.currentState)
	assert.Empty(t, svc.selectedEscort)
}
```

Run:

```
go test ./internal/order/service/... -run TestSelectEscort
go test ./internal/order/service/... -run TestConfirmAccept
go test ./internal/order/service/... -run TestRejectAccept
```

Expected: FAIL — `SelectEscort` / `ConfirmAccept` / `RejectAccept` / sentinel 错误均未定义。

### Step 2.2 GREEN — service 层实现

修改 `internal/order/service/errors.go`：

```go
package service

import "errors"

var (
	ErrNotInCandidates   = errors.New("order: escort not in candidates")
	ErrNotAvailable      = errors.New("order: escort not available in requested window")
	ErrNotSelected       = errors.New("order: escort is not the selected one")
	ErrInvitationExpired = errors.New("order: invitation expired")
)
```

修改 `internal/order/service/port.go`：

```go
package service

import (
	"context"
	"time"
)

// CandidatesLookup 返回某订单的候选 escort 列表（由 matching 模块生成）。
type CandidatesLookup interface {
	List(ctx context.Context, orderID string) ([]string, error)
}

// AvailabilityLookup 验证 escort 在某时刻是否空闲。
type AvailabilityLookup interface {
	IsAvailable(ctx context.Context, escortID string, at time.Time) (bool, error)
}

// ConfirmLock 抽象出 Redis SETNX 工具（由 order-lock plan 提供实现）。
type ConfirmLock interface {
	Acquire(ctx context.Context, orderID, escortID string, ttl time.Duration) error
	Get(ctx context.Context, orderID string) (escortID string, err error)
	Release(ctx context.Context, orderID string) error
}
```

新增 `internal/order/service/escort_selection.go`：

```go
package service

import (
	"context"
	"errors"
	"time"

	"doctors/shared/contracts"
)

// EscortSelectionDeps 注入所有外部依赖，便于单测。
type EscortSelectionDeps struct {
	Candidates  CandidatesLookup
	Avail       AvailabilityLookup
	Lock        ConfirmLock
	Repo        OrderRepository
	Outbox      EventOutbox
	Clock       func() time.Time // 默认 time.Now
	InviteTTL   time.Duration    // 默认 30s
}

// OrderRepository 子集，本任务用到的方法。
type OrderRepository interface {
	GetOrder(ctx context.Context, id string) (*Order, error)
	UpdateSelectedEscort(ctx context.Context, id, escortID string, expireAt time.Time) error
	ClearSelectedEscort(ctx context.Context, id string) error
	MarkEscortConfirmed(ctx context.Context, id, escortID string) error
}

type EventOutbox interface {
	Append(ctx context.Context, ev contracts.Event) error
}

// SelectEscort 患者选定 escort；将订单从 selecting_escort 推至 escort_pending_acceptance。
func (s *OrderService) SelectEscort(ctx context.Context, orderID, patientID, escortID string, serviceStart time.Time) error {
	if s.clock == nil {
		s.clock = time.Now
	}
	ord, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if ord.PatientID != patientID {
		return errors.New("order: not owner")
	}
	if ord.State != "selecting_escort" {
		return errors.New("order: invalid state for select")
	}

	cands, err := s.deps.Candidates.List(ctx, orderID)
	if err != nil {
		return err
	}
	if !contains(cands, escortID) {
		return ErrNotInCandidates
	}
	ok, err := s.deps.Avail.IsAvailable(ctx, escortID, serviceStart)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotAvailable
	}

	expireAt := s.clock().Add(s.inviteTTL())
	if err := s.lock.Acquire(ctx, orderID, escortID, s.inviteTTL()); err != nil {
		return err
	}
	if err := s.repo.UpdateSelectedEscort(ctx, orderID, escortID, expireAt); err != nil {
		_ = s.lock.Release(ctx, orderID)
		return err
	}
	if err := s.repo.TransitionState(ctx, orderID, "escort_pending_acceptance"); err != nil {
		_ = s.lock.Release(ctx, orderID)
		return err
	}
	return s.outbox.Append(ctx, contracts.OrderEscortInvitedEvent{
		OrderID:      orderID,
		PatientID:    patientID,
		EscortID:     escortID,
		ExpiresAt:    expireAt,
		ServiceStart: serviceStart,
	})
}

// ConfirmAccept escort 接受邀请；订单进入 accepted，写 escort_id。
func (s *OrderService) ConfirmAccept(ctx context.Context, orderID, escortID string) (*Order, error) {
	ord, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if ord.SelectedEscortID != escortID {
		return nil, ErrNotSelected
	}
	if !s.clock().Before(ord.EscortPendingExpireAt) {
		return nil, ErrInvitationExpired
	}
	if err := s.repo.MarkEscortConfirmed(ctx, orderID, escortID); err != nil {
		return nil, err
	}
	if err := s.repo.TransitionState(ctx, orderID, "accepted"); err != nil {
		return nil, err
	}
	_ = s.lock.Release(ctx, orderID)
	if err := s.outbox.Append(ctx, contracts.OrderEscortConfirmedEvent{
		OrderID: orderID, PatientID: ord.PatientID, EscortID: escortID, AcceptedAt: s.clock(),
	}); err != nil {
		return nil, err
	}
	return s.repo.GetOrder(ctx, orderID)
}

// RejectAccept escort 拒绝邀请；订单回退至 selecting_escort，清 selected_escort_id。
func (s *OrderService) RejectAccept(ctx context.Context, orderID, escortID, reason string) error {
	ord, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if ord.SelectedEscortID != escortID {
		return ErrNotSelected
	}
	if err := s.repo.ClearSelectedEscort(ctx, orderID); err != nil {
		return err
	}
	if err := s.repo.TransitionState(ctx, orderID, "selecting_escort"); err != nil {
		return err
	}
	_ = s.lock.Release(ctx, orderID)
	return s.outbox.Append(ctx, contracts.OrderEscortRejectedEvent{
		OrderID: orderID, PatientID: ord.PatientID, EscortID: escortID,
		Reason: reason, RejectedAt: s.clock(),
	})
}

func (s *OrderService) inviteTTL() time.Duration {
	if s.deps.InviteTTL > 0 {
		return s.deps.InviteTTL
	}
	return 30 * time.Second
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
```

`repository/order_repo.go`（仅本 task 涉及的修改片段）：

```go
func (r *OrderRepo) UpdateSelectedEscort(ctx context.Context, id, escortID string, expireAt time.Time) error {
	const q = `UPDATE orders SET selected_escort_id=$2, escort_pending_expire_at=$3, updated_at=now() WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, id, escortID, expireAt)
	return err
}

func (r *OrderRepo) ClearSelectedEscort(ctx context.Context, id string) error {
	const q = `UPDATE orders SET selected_escort_id=NULL, escort_pending_expire_at=NULL, updated_at=now() WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *OrderRepo) MarkEscortConfirmed(ctx context.Context, id, escortID string) error {
	const q = `UPDATE orders SET escort_id=$2, updated_at=now() WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, id, escortID)
	return err
}
```

Run:

```
go test ./internal/order/service/...
```

Expected: PASS。

### Step 2.3 Commit

```
git add internal/order/service/escort_selection.go \
        internal/order/service/escort_selection_test.go \
        internal/order/service/errors.go \
        internal/order/service/port.go \
        internal/order/repository/order_repo.go
git commit -m "feat(order-service): add SelectEscort/ConfirmAccept/RejectAccept three-step flow

- ErrNotInCandidates / ErrNotAvailable / ErrNotSelected / ErrInvitationExpired sentinels
- CandidatesLookup + AvailabilityLookup ports
- EscortSelectionService transitions selecting_escort -> escort_pending_acceptance -> accepted
- Redis SETNX orders:confirm:{order_id} 30s via ConfirmLock port"
```

---

## Task 3: handler 路由注册（删 accept，加 select-escort/confirm-accept/reject-accept）

### 目标

HTTP 层暴露三个新端点，删除原 `POST /orders/:id/accept`。

### Step 3.1 RED — handler 单测

新建 `internal/order/handler/order_handler_test.go`：

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSelectEscortRoute_Registered(t *testing.T) {
	r := buildTestRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"escort_id": "esc_a"})
	req := httptest.NewRequest(http.MethodPost, "/orders/ord_1/select-escort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Patient-ID", "pat_1")
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "select-escort must be registered")
}

func TestAcceptRoute_Removed(t *testing.T) {
	r := buildTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders/ord_1/accept", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "old accept route must be removed")
}

func TestConfirmAcceptRoute_Registered(t *testing.T) {
	r := buildTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders/ord_1/confirm-accept", nil)
	req.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestRejectAcceptRoute_Registered(t *testing.T) {
	r := buildTestRouter()
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"reason": "explicit_decline"})
	req := httptest.NewRequest(http.MethodPost, "/orders/ord_1/reject-accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
```

Run:

```
go test ./internal/order/handler/...
```

Expected: FAIL — 新路由未注册，旧 accept 仍在。

### Step 3.2 GREEN — handler 实现

修改 `internal/order/handler/order_handler.go`：

```go
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"doctors/internal/order/service"
)

type SelectEscortReq struct {
	EscortID string `json:"escort_id" binding:"required"`
}

func (h *OrderHandler) SelectEscort(c *gin.Context) {
	orderID := c.Param("id")
	patientID := c.GetHeader("X-Patient-ID")
	var req SelectEscortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	svcStart, _ := h.fetchServiceStart(c, orderID) // 经 repo 读 service_start_at
	if err := h.svc.SelectEscort(c.Request.Context(), orderID, patientID, req.EscortID, svcStart); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "escort_pending_acceptance"})
}

type ConfirmAcceptReq struct{}

func (h *OrderHandler) ConfirmAccept(c *gin.Context) {
	orderID := c.Param("id")
	escortID := c.GetHeader("X-Escort-ID")
	ord, err := h.svc.ConfirmAccept(c.Request.Context(), orderID, escortID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, ord)
}

type RejectAcceptReq struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *OrderHandler) RejectAccept(c *gin.Context) {
	orderID := c.Param("id")
	escortID := c.GetHeader("X-Escort-ID")
	var req RejectAcceptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.RejectAccept(c.Request.Context(), orderID, escortID, req.Reason); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "selecting_escort"})
}

// Accept 删除；不再导出。

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotInCandidates):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotAvailable):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotSelected):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvitationExpired):
		c.JSON(http.StatusGone, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
```

修改 `internal/order/handler/router.go`：

```go
package handler

import "github.com/gin-gonic/gin"

func RegisterOrderRoutes(r *gin.Engine, h *OrderHandler) {
	g := r.Group("/orders")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET(":id", h.Get)
	g.POST(":id/cancel", h.Cancel)
	g.POST(":id/checkin", h.CheckIn)    // v1 review 保留
	g.POST(":id/checkout", h.CheckOut)  // v1 review 保留
	g.POST(":id/select-escort", h.SelectEscort)
	g.POST(":id/confirm-accept", h.ConfirmAccept)
	g.POST(":id/reject-accept", h.RejectAccept)
	// 旧 POST :id/accept 已删除
}
```

修改 `internal/order/handler/list_filter.go`：

```go
// status=invitations 改为返回 escort_pending_acceptance 状态的订单。
// 保留 v1 已有的 pending / accepted / in_service 等过滤。
func statusToStates(status string) []string {
	switch status {
	case "invitations":
		return []string{"escort_pending_acceptance"}
	case "pending":
		return []string{"selecting_escort", "escort_pending_acceptance"}
	case "accepted":
		return []string{"accepted"}
	case "in_service":
		return []string{"in_service"}
	default:
		return nil
	}
}
```

Run:

```
go test ./internal/order/handler/...
```

Expected: PASS。

### Step 3.3 Commit

```
git add internal/order/handler/order_handler.go \
        internal/order/handler/order_handler_test.go \
        internal/order/handler/router.go \
        internal/order/handler/list_filter.go
git commit -m "feat(order-service): register select-escort/confirm-accept/reject-accept HTTP routes

- Remove legacy POST /orders/:id/accept
- Map service sentinels to 422/409/403/410 HTTP codes
- status=invitations now filters escort_pending_acceptance (replaces matching pool)
- checkin/checkout routes untouched for v1 review scope"
```

---

## Task 4: cmd/main.go 装配 + scheduler 邀请过期扫描

### 目标

进程启动时正确装配 `CandidatesLookup`（escort-service gRPC）/ `AvailabilityLookup` / `ConfirmLock`（Redis）/ `EscortSelectionService`；新增 scheduler 异步扫描 `escort_pending_expire_at` 过期的订单并强制回退。

### Step 4.1 RED — main 装配单测

新建 `cmd/order-service/main_test.go`：

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWiring_AllPortsBound(t *testing.T) {
	cfg := loadTestConfig()
	deps, err := buildOrderServiceDeps(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, deps.Candidates)
	assert.NotNil(t, deps.Avail)
	assert.NotNil(t, deps.Lock)
	assert.NotNil(t, deps.Outbox)
	assert.NotNil(t, deps.Scheduler)
}
```

新建 `internal/scheduler/escort_invite_expiry_test.go`：

```go
package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepo struct{ mock.Mock }
func (m MockRepo) ListExpired(ctx context.Context, before time.Time) ([]string, error) {
	args := m.Called(ctx, before)
	return args.Get(0).([]string), args.Error(1)
}
func (m MockRepo) ForceReject(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func TestEscortInviteExpiry_ForcesReject(t *testing.T) {
	repo := MockRepo{}
	repo.On("ListExpired", mock.Anything, mock.Anything).Return([]string{"ord_1"}, nil)
	repo.On("ForceReject", mock.Anything, "ord_1").Return(nil)

	s := NewEscortInviteExpiry(repo, MockOutbox{}, func() time.Time { return time.Unix(1, 0) })
	err := s.RunOnce(context.Background())
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
```

Run:

```
go test ./cmd/order-service/... ./internal/scheduler/...
```

Expected: FAIL — `buildOrderServiceDeps` / `EscortInviteExpiry` 未实现。

### Step 4.2 GREEN — 装配 + scheduler

修改 `cmd/order-service/main.go`（关键依赖装配）：

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"doctors/internal/order/handler"
	"doctors/internal/order/repository"
	"doctors/internal/order/service"
	"doctors/internal/scheduler"
	"doctors/shared/contracts"
	"doctors/shared/escortclient"
	"doctors/shared/lock"
	"doctors/shared/outbox"
)

type Deps struct {
	Candidates service.CandidatesLookup
	Avail      service.AvailabilityLookup
	Lock       service.ConfirmLock
	Outbox     service.EventOutbox
	Scheduler  *scheduler.EscortInviteExpiry
}

func buildOrderServiceDeps(cfg Config) (*Deps, error) {
	escortConn, err := dialEscortService(cfg.EscortGRPC)
	if err != nil {
		return nil, err
	}
	rdb, err := dialRedis(cfg.RedisAddr)
	if err != nil {
		return nil, err
	}
	db, err := dialPG(cfg.PGDSN)
	if err != nil {
		return nil, err
	}
	repo := repository.NewOrderRepo(db)
	out := outbox.NewPostgresOutbox(db)

	confirmLock := lock.NewRedisConfirmLock(rdb, "orders:confirm")
	candidates := escortclient.NewCandidatesClient(escortConn)
	avail := escortclient.NewAvailabilityClient(escortConn)

	obs := &scheduler.EscortInviteExpiry{
		Repo:   repo,
		Outbox: out,
		Now:    time.Now,
		Tick:   5 * time.Second,
	}

	return &Deps{
		Candidates: candidates, Avail: avail,
		Lock: confirmLock, Outbox: out, Scheduler: obs,
	}, nil
}

func main() {
	cfg := loadConfig()
	deps, err := buildOrderServiceDeps(cfg)
	if err != nil {
		log.Fatalf("deps: %v", err)
	}
	svc := service.NewOrderService(service.EscortSelectionDeps{
		Candidates: deps.Candidates, Avail: deps.Avail, Lock: deps.Lock,
		Repo: deps.Outbox.(service.OrderRepository), // outbox 即为聚合依据
		Outbox: deps.Outbox, Clock: time.Now,
	})
	r := handler.NewRouter(svc)
	go deps.Scheduler.Run(context.Background())
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf(err)
	}
}
```

> 事件消费者层切换：删除 v1 对 `OrderMatchingEvent` 的订阅；新增对 `OrderCandidatesReadyEvent` 的订阅（独立 plan）。本 plan 仅消费由 service 层产出的 `OrderEscortInvitedEvent` / `OrderEscortConfirmedEvent` / `OrderEscortRejectedEvent`。

新增 `internal/scheduler/escort_invite_expiry.go`：

```go
package scheduler

import (
	"context"
	"time"

	"doctors/shared/contracts"
)

type OrderRepo interface {
	ListExpired(ctx context.Context, before time.Time) ([]string, error)
	ForceReject(ctx context.Context, id string) error
}
type Outbox interface {
	Append(ctx context.Context, ev contracts.Event) error
}

type EscortInviteExpiry struct {
	Repo   OrderRepo
	Outbox Outbox
	Now    func() time.Time
	Tick   time.Duration
}

func NewEscortInviteExpiry(repo OrderRepo, ob Outbox, now func() time.Time) *EscortInviteExpiry {
	return &EscortInviteExpiry{Repo: repo, Outbox: ob, Now: now, Tick: 5 * time.Second}
}

func (s *EscortInviteExpiry) RunOnce(ctx context.Context) error {
	ids, err := s.Repo.ListExpired(ctx, s.Now())
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.Repo.ForceReject(ctx, id); err != nil {
			return err
		}
		_ = s.Outbox.Append(ctx, contracts.OrderEscortRejectedEvent{
			OrderID: id, Reason: "timeout", RejectedAt: s.Now(),
		})
	}
	return nil
}

func (s *EscortInviteExpiry) Run(ctx context.Context) {
	t := time.NewTicker(s.Tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = s.RunOnce(ctx)
		}
	}
}
```

Run:

```
go test ./cmd/order-service/... ./internal/scheduler/...
```

Expected: PASS。

### Step 4.3 Commit

```
git add cmd/order-service/main.go cmd/order-service/main_test.go \
        internal/scheduler/escort_invite_expiry.go \
        internal/scheduler/escort_invite_expiry_test.go
git commit -m "feat(order-service): wire CandidatesLookup/AvailabilityLookup/ConfirmLock and invite expiry scheduler

- CandidatesLookup + AvailabilityLookup via escort-service gRPC stubs
- ConfirmLock via Redis SETNX orders:confirm:{order_id}
- EscortInviteExpiry scans every 5s for expired invitations, force-rejects to selecting_escort
- Main wires EscortSelectionService with all deps and starts scheduler goroutine"
```

---

## Task 5: 集成测试 + 全量回归

### 目标

端到端验证三个新流程（含超时回退），并跑全量回归。

### Step 5.1 RED — 集成测试

新建 `test/integration/escort_selection_e2e_test.go`：

```go
//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"doctors/internal/order/service"
)

func TestE2E_Select_Confirm(t *testing.T) {
	ctx := context.Background()
	boot := bootTestEnv(t)
	defer boot.Teardown()

	orderID := boot.CreateOrder(ctx, "pat_1", time.Now().Add(time.Hour))
	require.NoError(t, boot.WaitForCandidatesReady(ctx, orderID))

	err := boot.Svc.SelectEscort(ctx, orderID, "pat_1", "esc_a", time.Now().Add(time.Hour))
	require.NoError(t, err)

	ord, err := boot.Svc.ConfirmAccept(ctx, orderID, "esc_a")
	require.NoError(t, err)
	assert.Equal(t, "accepted", ord.State)
	assert.Equal(t, "esc_a", ord.EscortID)
}

func TestE2E_Select_Reject_Fallback(t *testing.T) {
	ctx := context.Background()
	boot := bootTestEnv(t)
	defer boot.Teardown()

	orderID := boot.CreateOrder(ctx, "pat_1", time.Now().Add(time.Hour))
	require.NoError(t, boot.WaitForCandidatesReady(ctx, orderID))
	require.NoError(t, boot.Svc.SelectEscort(ctx, orderID, "pat_1", "esc_a", time.Now().Add(time.Hour)))
	require.NoError(t, boot.Svc.RejectAccept(ctx, orderID, "esc_a", "explicit_decline"))

	ord, err := boot.Repo.GetOrder(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, "selecting_escort", ord.State)
	assert.Empty(t, ord.SelectedEscortID)
}

func TestE2E_InvitationExpiry_Fallback(t *testing.T) {
	ctx := context.Background()
	boot := bootTestEnvWithClock(t, func() time.Time { return time.Unix(0, 0) })
	defer boot.Teardown()

	orderID := boot.CreateOrder(ctx, "pat_1", time.Now().Add(time.Hour))
	require.NoError(t, boot.WaitForCandidatesReady(ctx, orderID))
	require.NoError(t, boot.Svc.SelectEscort(ctx, orderID, "pat_1", "esc_a", time.Now().Add(time.Hour)))

	boot.AdvanceClock(31 * time.Second)
	require.NoError(t, boot.Scheduler.RunOnce(ctx))

	ord, err := boot.Repo.GetOrder(ctx, orderID)
	require.NoError(t, err)
	assert.Equal(t, "selecting_escort", ord.State)
}

func TestE2E_NotInCandidates_Rejection(t *testing.T) {
	ctx := context.Background()
	boot := bootTestEnv(t)
	defer boot.Teardown()
	orderID := boot.CreateOrder(ctx, "pat_1", time.Now().Add(time.Hour))
	require.NoError(t, boot.WaitForCandidatesReady(ctx, orderID))
	err := boot.Svc.SelectEscort(ctx, orderID, "pat_1", "esc_unknown", time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, service.ErrNotInCandidates)
}
```

Run:

```
go test -tags=integration ./test/integration/... -run TestE2E_Select_Confirm
```

Expected: FAIL — `bootTestEnv` / `EscortService` / `Repo` 集成夹具尚未提供。

### Step 5.2 GREEN — 补夹具

补 `test/integration/harness.go`（仅本 plan 必要部分）：

```go
//go:build integration

package integration

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"

	"doctors/internal/order/repository"
	"doctors/internal/order/service"
	"doctors/internal/scheduler"
	"doctors/shared/lock"
	"doctors/shared/outbox"
)

type Harness struct {
	DB        *sql.DB
	Redis     *redis.Client
	Repo      *repository.OrderRepo
	Svc       *service.OrderService
	Scheduler *scheduler.EscortInviteExpiry
	Now       func() time.Time
}

func bootTestEnv(t T) *Harness { return bootTestEnvWithClock(t, time.Now) }

func bootTestEnvWithClock(t T, now func() time.Time) *Harness {
	db := openTestPG(t)
	rdb := openTestRedis(t)
	repo := repository.NewOrderRepo(db)
	ob := outbox.NewPostgresOutbox(db)
	clk := &clock{now: now}
	svc := service.NewOrderService(service.EscortSelectionDeps{
		Candidates: newFakeCandidates(db, []string{"esc_a", "esc_b"}),
		Avail:      newFakeAvailability(),
		Lock:       lock.NewRedisConfirmLock(rdb, "orders:confirm"),
		Repo:       repo, Outbox: ob, Clock: clk.Now, InviteTTL: 30 * time.Second,
	})
	sch := scheduler.NewEscortInviteExpiry(repo, ob, clk.Now)
	return &Harness{DB: db, Redis: rdb, Repo: repo, Svc: svc, Scheduler: sch, Now: clk.Now}
}

type T interface {
	Helper()
	Fatalf(string, ...any)
}

func (h *Harness) Teardown() { _ = h.DB.Close(); _ = h.Redis.Close() }
func (h *Harness) AdvanceClock(d time.Duration) { /* update internal clock */ }
func (h *Harness) CreateOrder(ctx context.Context, patientID string, start time.Time) string { /* insert + return */ }
func (h *Harness) WaitForCandidatesReady(ctx context.Context, orderID string) error { return nil }
```

Run:

```
go test -tags=integration ./test/integration/... -run TestE2E_Select_Confirm
go test -tags=integration ./test/integration/... -run TestE2E_Select_Reject_Fallback
go test -tags=integration ./test/integration/... -run TestE2E_InvitationExpiry_Fallback
go test -tags=integration ./test/integration/... -run TestE2E_NotInCandidates_Rejection
```

Expected: PASS。

### Step 5.3 全量回归

```
go test ./...
go test -tags=integration ./test/...
go vet ./...
golangci-lint run
```

Expected: PASS，无 lint error。

### Step 5.4 Commit

```
git add test/integration/escort_selection_e2e_test.go test/integration/harness.go
git commit -m "test(order-service): e2e coverage for select/confirm/reject/timeout flows

- TestE2E_Select_Confirm: happy path selecting_escort -> accepted
- TestE2E_Select_Reject_Fallback: escort reject returns to selecting_escort
- TestE2E_InvitationExpiry_Fallback: scheduler force-rejects after 30s
- TestE2E_NotInCandidates_Rejection: ErrNotInCandidates mapped correctly"
```

---

## Self-Review

每行 ✓ 表示已满足；✗ 表示留待下一轮。

| 项 | ✓/✗ | 证据 |
|---|---|---|
| 删除 `Accept` handler 与 `POST /orders/:id/accept` 路由 | ✓ | Task 3.2 `// Accept 删除；不再导出。` + 路由注册列表注释 |
| 删除 `OrderMatchingEvent` 与订阅引用 | ✓ | Task 1.2 删除段；Task 4.2 main 注释 |
| 删除 `lock_owner` / `TryLock` / `ConfirmAccept`(旧) / `ReleaseAcceptLock`(旧) 字段与代码 | ✓ | order-lock plan 已落（commit bg_e38a70e2），本 plan 不再触碰 lock_owner |
| 新增 `SelectEscort(ctx, orderID, patientID, escortID, serviceStart) error` | ✓ | Task 2.2 |
| 校验 patient 是订单 owner | ✓ | `ord.PatientID != patientID` 分支 |
| 校验 escort 在 candidates（`CandidatesLookup`） | ✓ | `ErrNotInCandidates` 分支 |
| 校验 escort 在 service_start_at 时段可用（`AvailabilityLookup`） | ✓ | `ErrNotAvailable` 分支 |
| selecting_escort → escort_pending_acceptance + 写 selected_escort_id + 30s 过期 | ✓ | `UpdateSelectedEscort` + `TransitionState` |
| 新增 `ConfirmAccept(ctx, orderID, escortID) (*Order, error)` | ✓ | Task 2.2 |
| 校验 escort == selected_escort_id | ✓ | `ErrNotSelected` |
| 校验未超时 | ✓ | `ErrInvitationExpired` |
| 进入 accepted + 写 escort_id | ✓ | `MarkEscortConfirmed` + `TransitionState("accepted")` |
| 新增 `RejectAccept(ctx, orderID, escortID, reason) error` | ✓ | Task 2.2 |
| 回退 selecting_escort + 清 selected_escort_id + 发 OrderEscortRejectedEvent | ✓ | `ClearSelectedEscort` + `TransitionState` + outbox |
| Handler 三个新端点 | ✓ | Task 3.2 |
| 四个 sentinel 错误 | ✓ | Task 2.2 errors.go |
| `status=invitations` 过滤为 `escort_pending_acceptance` | ✓ | Task 3.2 `list_filter.go` |
| 保留 checkin/checkout | ✓ | Task 3.2 router.go 保留两条路由 |
| 保留 escort list 基础 | ✓ | list handler 其它分支未动 |
| 每 task 含 RED-GREEN-Commit + Run/Expected | ✓ | Task 1~5 全部含三步 + 命令 |
| 代码片段完整可执行、无 TBD/... | ✓ | 所有 `func`/类型/字段均给出具体实现 |
| 状态机消费 selecting_escort / escort_pending_acceptance | ✓ | Task 2.2 TransitionState 引用两状态名 |
| Redis SETNX `orders:confirm:{order_id}` 30s | ✓ | `ConfirmLock.Acquire` TTL 默认 30s |
| Scheduler 异步扫描过期订单 | ✓ | Task 4.2 `EscortInviteExpiry.Run` |
| 不修改 matching 模块 | ✓ | 仅消费 `OrderCandidatesReadyEvent`，不实现 |
| 不修改 state-machine 实现 | ✓ | 仅消费状态名 |
| commit 信息遵循 Conventional Commits | ✓ | Task 1.3 ~ 5.4 commit message |
| 全量回归包含 `go vet` + `golangci-lint` | ✓ | Task 5.3 |