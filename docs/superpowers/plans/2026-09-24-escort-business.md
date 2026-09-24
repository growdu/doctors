# 陪诊业务实装 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.2 P0 escort 业务 API：14 个新 API（escort 实名 / 健康证 / 培训 / 上线 + 订单抢单池 + 签到 / 打卡）以及评审 §4 评审 I-03 陪诊师档案状态机。覆盖 escort-app P0 模块 1~6。**注意**：14 个 API 中，本 plan 落 11 个（其中 3 个在 order-service 扩展）；陪诊师钱包 + 提现 4 API 走 wallet plan；SOS 重复触发（patient/escort 均可）走 sos plan（已交付）。

**Architecture:**
- 复用 `services/escort/internal/service` 既有 12 个单测骨架（Register / SetAvailability / UpdateLocation 等基础方法保留；新增业务流方法）。
- 扩展 `escort_profiles` + `health_certs` + `training_records` 3 张表。
- service 层封装状态机：`registering → pending_real_name → pending_health_cert → pending_training → pending_agreement → pending_audit → approved/rejected → online/offline`（11 态）。
- order-service 扩展 `GET /orders?role=escort&status=matching`（抢单池）+ `POST /orders/:id/checkin`（accepted → in_service）+ `POST /orders/:id/checkout`（in_service → completed）。
- 实名 mock：service.RealNameAuth 直接返回成功 + 标记 users.real_name_verified=true。
- 健康证：v1 接收 base64；repo 存 SHA256 + 文件名 + mime；不入 OSS。
- 培训题库：service 内部 hardcode 5 道题（80% 通过 = 4/5）。
- GPS 签到 / 打卡：v1 mock 距离校验（不调 geolocator），只接 `lat` / `lng` 入参；只校验范围合法性，校验通过即落库 `checkin_at` / `checkout_at`。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · gin v1.10 · segmentio/kafka-go v0.4.51。

**前置依赖:**
- `2026-09-24-state-machine.md`（orders 表 + status CHECK 含 accepted/in_service/completed；本 plan 的 checkin/checkout 用此 CHECK）
- `2026-09-24-sos.md`（sos_records 表 + POST /orders/:id/sos 已交付，escort 侧触发复用）
- `2026-09-24-l2-api-gap-design.md` §2.2 escort-app P0 清单 + §3.1 EscortSummary entity
- `2026-09-24-escort-app-design.md` §3.2 11 态状态机 + §5 GPS 签到

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1 + segmentio/kafka-go v0.4.51
- 测试覆盖率：业务包 ≥ 80%
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- 陪诊师档案状态机 11 态（`registering` / `pending_real_name` / `pending_health_cert` / `pending_training` / `pending_agreement` / `pending_audit` / `approved` / `rejected` / `online` / `in_service` / `offline`）；service 暴露 `Transition(state, action) error` 严格白名单
- 状态机转换守门：必须按 registering → pending_real_name → ... 顺序；非法转换返回 `CodeForbidden`
- 实名 mock：service 内部直接返回成功 + 落 `users.real_name_verified=true`（不接第三方）
- 健康证 v1：API 收 `image_base64` + `filename`；repo 算 `sha256(image_base64)` 存 hash + filename + mime；**不存原始 base64**，不入 OSS
- 培训考核：service 内 hardcode 5 道单选题（std[2]{"A","B"}）；通过阈值 80% = 答对 ≥ 4 题
- GPS 校验：checkin 接 `lat` ∈ [-90, 90] + `lng` ∈ [-180, 180]；范围合法即落 `checkin_at`；不调用 geolocator / 不算距离
- 抢单池：`GET /api/v1/orders?role=escort&status=matching` 列出 `status='matching'` + `escort_id IS NULL` 的订单；`role=escort` 强制 role check（escort token 才能查）；patient 查老路径
- 事件发布：best-effort，失败仅 log，不阻塞主流程
- admin 审核（approve / reject）：不在本 plan，留 `2026-09-24-admin-plan.md`；状态机预留 `approved / rejected`，admin plan 用 `UPDATE escort_profiles SET state='approved' WHERE id=$1` 直接落库

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `migrations/0006_escort_profiles.up.sql` | Create | `escort_profiles` + `health_certs` + `training_records` 3 表 + 索引 |
| `migrations/0006_escort_profiles.down.sql` | Create | 逆向 |
| `migrations/migrations_test.go` | Modify | 加 `Test0006EscortProfilesUpDown` |
| `services/escort/internal/state/machine.go` | Create | 陪诊师档案状态机 pure function（`CanTransition(state, action) bool`） |
| `services/escort/internal/state/machine_test.go` | Create | 状态机单测（合法 / 非法转换矩阵） |
| `services/escort/internal/repo/profile_repo.go` | Create | pgx 实现 `ProfileRepo`（CRUD + state 更新 + health_certs 写入 + training_records 查询） |
| `services/escort/internal/repo/profile_repo_integration_test.go` | Create | 集成测试（建表 + 写入 + 状态推进 + 反查） |
| `services/escort/internal/service/profile_flow.go` | Create | 业务流（`RegisterFlow` / `RealNameAuth` / `UploadHealthCert` / `CompleteTraining` / `SignAgreement` / `SetOnline` / `SetOffline`） |
| `services/escort/internal/service/profile_flow_test.go` | Create | 业务流单测（用 fake repo；覆盖状态机非法转换 + 实名 mock + 健康证 hash + 培训通过阈值） |
| `services/escort/internal/service/escort_service.go` | Modify | 加 `GetMyProfile(userID)` + `ListTrainingCourses()` + `ListMyReviews(userID)` |
| `services/escort/internal/service/escort_service_test.go` | Modify | 既有 12 个测试保留；新加 3 个测试 |
| `shared/contracts/events.go` | Modify | 加 `EscortStateChangedEvent` + `TopicEscortStateChanged` |
| `shared/contracts/contracts_test.go` | Modify | 加 `TestEscortStateChangedEvent_RoundTrip` + topic 常量 |
| `services/escort/internal/events/publisher.go` | Create | `EscortPublisher`（`PublishStateChanged`）+ Kafka + Nop |
| `services/escort/internal/events/publisher_test.go` | Create | publisher 单测 |
| `services/escort/internal/handler/escort_business.go` | Create | `Handler` 扩展：8 个 endpoint（register / real-name / health-cert / training/complete / me/profile / me/status / me/training-courses / me/reviews） |
| `services/escort/internal/handler/escort_business_test.go` | Create | handler 单测（fake service + httptest） |
| `services/escort/internal/handler/escort.go` | Modify | `RegisterRoutes` 增挂 8 个 route |
| `services/escort/internal/router/router.go` | Modify | 无需改（handler.RegisterRoutes 已挂 v1 group） |
| `services/escort/cmd/main.go` | Modify | 加 `events.NewKafkaPublisher(cfg.Kafka.Brokers)` + `repo.NewProfileRepo(nil)` 装配 |
| `services/order/internal/repo/order_repo.go` | Modify | 加 `ListForEscort(ctx, status, limit, offset)`（查询 `status=$1 AND escort_id IS NULL AND deleted_at IS NULL`） |
| `services/order/internal/repo/order_repo_integration_test.go` | Modify | 加 `TestOrderRepo_ListForEscort_OK` + `TestOrderRepo_ListForEscort_ExcludesAssigned` |
| `services/order/internal/service/order_service.go` | Modify | 加 `CheckIn(ctx, orderID, escortID, lat, lng)` + `CheckOut(ctx, orderID, escortID, note)` + `ListForEscort(ctx, userID, status, limit, offset)` |
| `services/order/internal/service/order_service.go` | Modify | `Order` struct 加 `CheckinAt *time.Time` + `CheckoutAt *time.Time` |
| `services/order/internal/service/order_service_test.go` | Modify | 加 3 个测试（CheckIn_OK / CheckIn_InvalidLat / CheckOut_OK） |
| `services/order/internal/handler/order.go` | Modify | `List` 支持 `role=escort` 分支；加 `CheckInHandler` + `CheckOutHandler` |
| `services/order/internal/handler/order.go` | Modify | `RegisterRoutes` 加 `POST /:id/checkin` + `POST /:id/checkout` |
| `services/order/internal/handler/order_test.go` | Modify | 加 `TestList_EscortRole` + `TestCheckIn_OK` + `TestCheckOut_OK` |
| `services/order/cmd/main.go` | Modify | 无需改（service.New 已装配） |
| `scripts/smoke-escort.sh` | Create | smoke（build + 启动 + /healthz + 11 个 endpoint 401 拦截） |
| `docs/04-业务流程.md` | Modify | §4.7 加陪诊师档案状态机流程 |
| `dev.md` | Modify | §10.13 加 escort-business plan 落地记录 |

---

### Task 1: 数据库迁移（escort_profiles + health_certs + training_records）

**Files:**
- Create: `migrations/0006_escort_profiles.up.sql`
- Create: `migrations/0006_escort_profiles.down.sql`
- Modify: `migrations/migrations_test.go`

**Step 1: 写集成测试**

在 `migrations_test.go` 末尾追加（参考既有 `Test0005SosRecordsUpDown` 风格）：

```go
// Test0006EscortProfilesUpDown 验证 escort_profiles + health_certs + training_records
// 三表 + CHECK + 索引 + down 可逆。
func Test0006EscortProfilesUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 先建 users + orders（escort_profiles 引用 users.id）
	for _, f := range []string{"0001_users.up.sql", "0002_orders.up.sql", "0003_orders_state.up.sql"} {
		sql, _ := os.ReadFile(f)
		_, err = conn.Exec(ctx, string(sql))
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		cleanConn, err := pgx.Connect(cleanCtx, dsn())
		if err != nil {
			return
		}
		defer cleanConn.Close(cleanCtx)
		for _, f := range []string{"0003_orders_state.down.sql", "0002_orders.down.sql", "0001_users.down.sql"} {
			sql, _ := os.ReadFile(f)
			_, _ = cleanConn.Exec(cleanCtx, string(sql))
		}
	})

	applyUp(t, "0006_escort_profiles.up.sql", []string{"escort_profiles", "health_certs", "training_records"})

	// 列检查：escort_profiles
	for _, col := range []string{"id", "user_id", "state", "city", "rating", "version", "created_at", "updated_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='escort_profiles' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "escort_profiles.%s should exist", col)
	}

	// 列检查：health_certs
	for _, col := range []string{"id", "user_id", "filename", "sha256", "mime", "status", "created_at", "reviewed_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='health_certs' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "health_certs.%s should exist", col)
	}

	// 列检查：training_records
	for _, col := range []string{"id", "user_id", "course_id", "score", "passed", "completed_at"} {
		var found bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='training_records' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "training_records.%s should exist", col)
	}

	// 索引检查
	for _, idx := range []string{"idx_escort_profiles_user", "idx_health_certs_user_created", "idx_training_records_user"} {
		var idxExists bool
		err = conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, idx).
			Scan(&idxExists)
		require.NoError(t, err)
		assert.True(t, idxExists, "%s should exist", idx)
	}

	// CHECK 约束：state 11 态
	var hasCheck bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.check_constraints
		               WHERE constraint_name LIKE 'escort_profiles_state_check')`).
		Scan(&hasCheck)
	require.NoError(t, err)
	assert.True(t, hasCheck)

	// down 校验
	downCtx, downCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer downCancel()
	downSQL, err := os.ReadFile("0006_escort_profiles.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(downCtx, string(downSQL))
	require.NoError(t, err, "apply 0006_escort_profiles.down.sql")

	for _, tbl := range []string{"escort_profiles", "health_certs", "training_records"} {
		var gone bool
		err = conn.QueryRow(downCtx,
			`SELECT NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1)`, tbl).
			Scan(&gone)
		require.NoError(t, err)
		assert.True(t, gone, "%s should be gone after down", tbl)
	}
}
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0006EscortProfilesUpDown ./migrations/
```
Expected: FAIL — `Test0006EscortProfilesUpDown` undefined

**Step 3: 写 `0006_escort_profiles.up.sql`**

```sql
-- 0006_escort_profiles.up.sql
-- 陪诊师档案主表 + 健康证 + 培训记录（评审 I-03 + escort-app P0 模块 1~3）。
-- 11 态状态机：registering / pending_real_name / pending_health_cert / pending_training /
--              pending_agreement / pending_audit / approved / rejected / online / in_service / offline
-- 与 escort-app-design.md §3.2 对齐；admin 审核通过/拒绝直接 UPDATE state。

CREATE TABLE escort_profiles (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id),
  state VARCHAR(24) NOT NULL DEFAULT 'registering'
    CHECK (state IN (
      'registering','pending_real_name','pending_health_cert','pending_training',
      'pending_agreement','pending_audit','approved','rejected',
      'online','in_service','offline'
    )),
  city VARCHAR(64),
  rating NUMERIC(3,2) NOT NULL DEFAULT 5.00,
  bad_rate NUMERIC(5,4) NOT NULL DEFAULT 0.0000,
  level VARCHAR(16) NOT NULL DEFAULT 'bronze',
  version INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_escort_profiles_user ON escort_profiles(user_id);
CREATE INDEX idx_escort_profiles_state ON escort_profiles(state, updated_at)
  WHERE state IN ('online','in_service');

-- 健康证表：v1 存 hash + 文件名 + mime；不入 OSS。
-- status：pending（已上传待审）/ approved / rejected
CREATE TABLE health_certs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  filename VARCHAR(255) NOT NULL,
  sha256 CHAR(64) NOT NULL,
  mime VARCHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','approved','rejected')),
  rejection_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  reviewed_by BIGINT REFERENCES users(id)
);

-- 同一用户最新一条健康证（审核查"这个陪诊师现在挂的是哪张"）
CREATE INDEX idx_health_certs_user_created ON health_certs(user_id, created_at DESC);
CREATE INDEX idx_health_certs_status ON health_certs(status, created_at) WHERE status = 'pending';

-- 培训记录表：每次考核落 1 行（v1 题库固定，可重考）。
CREATE TABLE training_records (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  course_id VARCHAR(32) NOT NULL,
  score INT NOT NULL CHECK (score >= 0 AND score <= 100),
  passed BOOLEAN NOT NULL,
  answers JSONB NOT NULL,            -- [{"q":1,"a":"A"},...]
  completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 同一用户同课程通过记录（admin 查"通过了没"）
CREATE INDEX idx_training_records_user ON training_records(user_id, course_id, completed_at DESC);
```

**Step 4: 写 `0006_escort_profiles.down.sql`**

```sql
-- 0006_escort_profiles.down.sql
-- 撤销 0006：删索引 + 删表。

DROP INDEX IF EXISTS idx_training_records_user;
DROP INDEX IF EXISTS idx_health_certs_status;
DROP INDEX IF EXISTS idx_health_certs_user_created;
DROP INDEX IF EXISTS idx_escort_profiles_state;
DROP INDEX IF EXISTS idx_escort_profiles_user;

DROP TABLE IF EXISTS training_records;
DROP TABLE IF EXISTS health_certs;
DROP TABLE IF EXISTS escort_profiles;
```

**Step 5: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run Test0006EscortProfilesUpDown ./migrations/
```
Expected: PASS

**Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0006 escort_profiles + health_certs + training_records (11 态 CHECK + 3 表 + 集成测试)"
```

---

### Task 2: 陪诊师状态机 pure function

**Files:**
- Create: `services/escort/internal/state/machine.go`
- Create: `services/escort/internal/state/machine_test.go`

**Step 1: 写单测（RED）**

`services/escort/internal/state/machine_test.go`：

```go
package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCanTransition_Legal 验证所有合法转换（11 态 × 主路径）。
func TestCanTransition_Legal(t *testing.T) {
	legal := []struct {
		from, action string
	}{
		// 注册流（注册后即转 pending_real_name）
		{"registering", "submit_real_name"},
		{"pending_real_name", "real_name_approved"},
		{"pending_health_cert", "health_cert_approved"},
		{"pending_training", "training_passed"},
		{"pending_agreement", "agreement_signed"},
		{"pending_audit", "audit_approved"},
		{"pending_audit", "audit_rejected"},
		// admin 审核后的二跳
		{"approved", "go_online"},
		{"approved", "go_offline"},
		{"online", "go_offline"},
		{"online", "in_service_start"},
		{"in_service", "service_done"},
		{"in_service", "go_offline"},
		// 审核拒绝后重新提交
		{"rejected", "resubmit_health_cert"},
	}
	for _, c := range legal {
		assert.True(t, CanTransition(c.from, c.action),
			"%s --%s--> ? should be legal", c.from, c.action)
	}
}

// TestCanTransition_Illegal 验证非法转换被拒。
func TestCanTransition_Illegal(t *testing.T) {
	illegal := []struct {
		from, action string
	}{
		// 跳级
		{"registering", "go_online"},           // 不能跳过实名
		{"pending_real_name", "go_online"},     // 不能跳过健康证
		// 倒序
		{"approved", "submit_real_name"},
		// 终态再转换
		{"offline", "go_online"}, // offline → online 需经 approved 中间，单独测
		// 角色错位
		{"pending_real_name", "audit_approved"}, // 没经过健康证 / 培训 / 签署
		// 未知动作
		{"registering", "unknown_action"},
	}
	for _, c := range illegal {
		assert.False(t, CanTransition(c.from, c.action),
			"%s --%s--> ? should be illegal", c.from, c.action)
	}
}

// TestOffline_ToOnline_RequiresApproved 验证 offline → online 必须先 approved。
func TestOffline_ToOnline_RequiresApproved(t *testing.T) {
	assert.True(t, CanTransition("approved", "go_online"))
	assert.False(t, CanTransition("offline", "go_online"))
}

// TestAllStates_HaveAtLeastOneTransition 验证 11 态都注册。
func TestAllStates_HaveAtLeastOneTransition(t *testing.T) {
	all := []string{
		"registering", "pending_real_name", "pending_health_cert", "pending_training",
		"pending_agreement", "pending_audit", "approved", "rejected",
		"online", "in_service", "offline",
	}
	for _, s := range all {
		assert.True(t, IsValid(s), "state %s should be valid", s)
	}
}
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/state/
```
Expected: FAIL — `state` package not exists

**Step 3: 写 machine.go**

`services/escort/internal/state/machine.go`：

```go
// Package state 实现陪诊师档案状态机（pure function）。
//
// 设计要点：
//   - State 用 string 常量，与 DB CHECK 约束对齐。
//   - transitions 是静态白名单；CanTransition 是纯函数，无副作用、易测试。
//   - 状态机变更的副作用（写 order_events、发 Kafka）由 service 层负责，
//     本包只回答"这个动作能不能做"。
//
// 11 态流转：
//   registering → pending_real_name → pending_health_cert → pending_training
//   → pending_agreement → pending_audit → approved/rejected → online/offline
//   online → in_service → online/offline
package state

// State 是陪诊师档案状态枚举，与 escort_profiles.state CHECK 对齐。
type State string

const (
	StateRegistering        State = "registering"
	StatePendingRealName    State = "pending_real_name"
	StatePendingHealthCert  State = "pending_health_cert"
	StatePendingTraining    State = "pending_training"
	StatePendingAgreement   State = "pending_agreement"
	StatePendingAudit       State = "pending_audit"
	StateApproved           State = "approved"
	StateRejected           State = "rejected"
	StateOnline             State = "online"
	StateInService          State = "in_service"
	StateOffline            State = "offline"
)

// transitions[from][action] = to。空 map 表示"这个状态不允许此动作"。
var transitions = map[State]map[string]State{
	StateRegistering: {
		"submit_real_name": StatePendingRealName,
	},
	StatePendingRealName: {
		"real_name_approved": StatePendingHealthCert,
	},
	StatePendingHealthCert: {
		"health_cert_approved": StatePendingTraining,
	},
	StatePendingTraining: {
		"training_passed": StatePendingAgreement,
	},
	StatePendingAgreement: {
		"agreement_signed": StatePendingAudit,
	},
	StatePendingAudit: {
		"audit_approved": StateApproved,
		"audit_rejected": StateRejected,
	},
	StateRejected: {
		"resubmit_health_cert": StatePendingHealthCert,
	},
	StateApproved: {
		"go_online":  StateOnline,
		"go_offline": StateOffline,
	},
	StateOnline: {
		"go_offline":    StateOffline,
		"in_service_start": StateInService,
	},
	StateInService: {
		"service_done": StateOnline,
		"go_offline":   StateOffline,
	},
	StateOffline: {}, // 终态（重新上线需先 approved，admin plan 可重置 approved）
}

// CanTransition 判定 from 下执行 action 是否合法；返回 true 时附带目标状态。
func CanTransition(from State, action string) (State, bool) {
	m, ok := transitions[from]
	if !ok {
		return "", false
	}
	to, ok := m[action]
	return to, ok
}

// IsValid 检查字符串是否为合法的 State 值。
func IsValid(s State) bool {
	_, ok := transitions[s]
	return ok
}

// AllStates 返回全部 11 态（调试 / 文档用）。
func AllStates() []State {
	return []State{
		StateRegistering, StatePendingRealName, StatePendingHealthCert, StatePendingTraining,
		StatePendingAgreement, StatePendingAudit, StateApproved, StateRejected,
		StateOnline, StateInService, StateOffline,
	}
}
```

> **注**：单测里 `assert.True(t, CanTransition(c.from, c.action))` 与新签名 `(State, bool)` 不匹配；实施时把单测改为：
>
> ```go
> _, ok := CanTransition(c.from, c.action)
> assert.True(t, ok, ...)
> ```
>
> 同步调整 `TestCanTransition_Illegal` 用 `_, ok := ...; assert.False(t, ok)`。

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/state/
```
Expected: PASS（4 个测试）

**Step 5: Commit**

```bash
git add services/escort/internal/state/
git commit -m "feat(escort): 陪诊师 11 态状态机 (registering→approved→online/offline) + 4 个单测"
```

---

### Task 3: profile_repo（pgx 实现 + 哨兵错误 + 集成测试）

**Files:**
- Create: `services/escort/internal/repo/profile_repo.go`
- Create: `services/escort/internal/repo/profile_repo_integration_test.go`

**Step 1: 写集成测试（RED）**

`services/escort/internal/repo/profile_repo_integration_test.go`：

```go
//go:build integration
// +build integration

package repo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDSN() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

// setupPool 起连接池并准备 users + escort_profiles + health_certs + training_records 表。
// 复用 migrations 0001/0006（+ 0002/0003 简化起见）。
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS training_records CASCADE;
		DROP TABLE IF EXISTS health_certs CASCADE;
		DROP TABLE IF EXISTS escort_profiles CASCADE;
		DROP TABLE IF EXISTS order_events;
		DROP TABLE IF EXISTS orders;
		DROP TABLE IF EXISTS users;
		CREATE TABLE users (
		  id BIGSERIAL PRIMARY KEY,
		  phone VARCHAR(20) UNIQUE NOT NULL,
		  role VARCHAR(16) NOT NULL CHECK (role IN ('patient','escort','admin')),
		  real_name_verified BOOLEAN NOT NULL DEFAULT FALSE,
		  wx_unionid VARCHAR(64),
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  deleted_at TIMESTAMPTZ
		);
		CREATE TABLE escort_profiles (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id),
		  state VARCHAR(24) NOT NULL DEFAULT 'registering'
		    CHECK (state IN (
		      'registering','pending_real_name','pending_health_cert','pending_training',
		      'pending_agreement','pending_audit','approved','rejected',
		      'online','in_service','offline'
		    )),
		  city VARCHAR(64),
		  rating NUMERIC(3,2) NOT NULL DEFAULT 5.00,
		  bad_rate NUMERIC(5,4) NOT NULL DEFAULT 0.0000,
		  level VARCHAR(16) NOT NULL DEFAULT 'bronze',
		  version INT NOT NULL DEFAULT 0,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE health_certs (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  filename VARCHAR(255) NOT NULL,
		  sha256 CHAR(64) NOT NULL,
		  mime VARCHAR(64) NOT NULL,
		  status VARCHAR(16) NOT NULL DEFAULT 'pending'
		    CHECK (status IN ('pending','approved','rejected')),
		  rejection_reason TEXT,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  reviewed_at TIMESTAMPTZ,
		  reviewed_by BIGINT REFERENCES users(id)
		);
		CREATE TABLE training_records (
		  id BIGSERIAL PRIMARY KEY,
		  user_id BIGINT NOT NULL REFERENCES users(id),
		  course_id VARCHAR(32) NOT NULL,
		  score INT NOT NULL CHECK (score >= 0 AND score <= 100),
		  passed BOOLEAN NOT NULL,
		  answers JSONB NOT NULL,
		  completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS training_records;
			DROP TABLE IF EXISTS health_certs;
			DROP TABLE IF EXISTS escort_profiles;
			DROP TABLE IF EXISTS order_events;
			DROP TABLE IF EXISTS orders;
			DROP TABLE IF EXISTS users;
		`)
		pool.Close()
	})
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool, phone, role string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone, role) VALUES ($1, $2) RETURNING id`, phone, role).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestProfileRepo_CreateAndGetByUser 验证创建 + 按 user_id 查询。
func TestProfileRepo_CreateAndGetByUser(t *testing.T) {
	pool := setupPool(t)
	userID := seedUser(t, pool, "13800138000", "escort")
	r := NewProfileRepo(pool)

	p := &Profile{UserID: userID, State: "registering"}
	require.NoError(t, r.Create(context.Background(), p))
	assert.NotZero(t, p.ID)
	assert.NotZero(t, p.Version)

	got, err := r.GetByUserID(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, "registering", got.State)
}

// TestProfileRepo_GetByUser_None 验证用户无 escort_profile 时返回 nil。
func TestProfileRepo_GetByUser_None(t *testing.T) {
	pool := setupPool(t)
	r := NewProfileRepo(pool)
	got, err := r.GetByUserID(context.Background(), 99999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestProfileRepo_UpdateState_OK 验证状态推进 + version 自增。
func TestProfileRepo_UpdateState_OK(t *testing.T) {
	pool := setupPool(t)
	userID := seedUser(t, pool, "13800138001", "escort")
	r := NewProfileRepo(pool)

	p := &Profile{UserID: userID, State: "registering"}
	require.NoError(t, r.Create(context.Background(), p))

	require.NoError(t, r.UpdateState(context.Background(), p.ID, "pending_real_name", p.Version))
	got, _ := r.GetByUserID(context.Background(), userID)
	require.NotNil(t, got)
	assert.Equal(t, "pending_real_name", got.State)
	assert.Equal(t, p.Version+1, got.Version)
}

// TestProfileRepo_UpdateState_VersionConflict 验证乐观锁冲突。
func TestProfileRepo_UpdateState_VersionConflict(t *testing.T) {
	pool := setupPool(t)
	userID := seedUser(t, pool, "13800138002", "escort")
	r := NewProfileRepo(pool)

	p := &Profile{UserID: userID, State: "registering"}
	require.NoError(t, r.Create(context.Background(), p))

	err := r.UpdateState(context.Background(), p.ID, "pending_real_name", p.Version+999)
	assert.ErrorIs(t, err, ErrVersionConflict)
}

// TestProfileRepo_InsertHealthCert 验证健康证写入。
func TestProfileRepo_InsertHealthCert(t *testing.T) {
	pool := setupPool(t)
	userID := seedUser(t, pool, "13800138003", "escort")
	r := NewProfileRepo(pool)

	cert := &HealthCert{
		UserID:   userID,
		Filename: "cert.jpg",
		SHA256:   "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		MIME:     "image/jpeg",
		Status:   "pending",
	}
	require.NoError(t, r.InsertHealthCert(context.Background(), cert))
	assert.NotZero(t, cert.ID)
	assert.NotZero(t, cert.CreatedAt)

	latest, err := r.LatestHealthCertByUser(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, "cert.jpg", latest.Filename)
}

// TestProfileRepo_LatestHealthCert_None 验证无健康证返回 nil。
func TestProfileRepo_LatestHealthCert_None(t *testing.T) {
	pool := setupPool(t)
	userID := seedUser(t, pool, "13800138004", "escort")
	r := NewProfileRepo(pool)

	latest, err := r.LatestHealthCertByUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Nil(t, latest)
}

// TestProfileRepo_InsertTrainingRecord 验证培训记录写入。
func TestProfileRepo_InsertTrainingRecord(t *testing.T) {
	pool := setupPool(t)
	userID := seedUser(t, pool, "13800138005", "escort")
	r := NewProfileRepo(pool)

	rec := &TrainingRecord{
		UserID:   userID,
		CourseID: "escort-basics",
		Score:    100,
		Passed:   true,
		Answers:  []byte(`[{"q":1,"a":"A"}]`),
	}
	require.NoError(t, r.InsertTrainingRecord(context.Background(), rec))
	assert.NotZero(t, rec.ID)

	records, err := r.ListTrainingByUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "escort-basics", records[0].CourseID)
	assert.True(t, records[0].Passed)
}

// 兜底编译（os 包用于 testDSN 环境变量）。
var _ = os.Getenv
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestProfileRepo_' ./services/escort/internal/repo/
```
Expected: FAIL — `undefined: NewProfileRepo`, `undefined: Profile`, `undefined: HealthCert`, `undefined: TrainingRecord`, `undefined: ErrVersionConflict`

**Step 3: 写 profile_repo.go**

`services/escort/internal/repo/profile_repo.go`：

```go
// Package repo 是 escort-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc）。
//   - state 字段用 CHECK 约束（DB 层先验）。
//   - 状态推进用乐观锁（version）；冲突返回 ErrVersionConflict。
//   - 健康证只存 hash + 文件名 + mime（不入 OSS）；培训记录 answers 用 JSONB。
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Profile 映射 escort_profiles 表行。
type Profile struct {
	ID        int64
	UserID    int64
	State     string // 11 态字符串
	City      string
	Rating    float64
	BadRate   float64
	Level     string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HealthCert 映射 health_certs 表行。
type HealthCert struct {
	ID              int64
	UserID          int64
	Filename        string
	SHA256          string
	MIME            string
	Status          string // "pending" | "approved" | "rejected"
	RejectionReason string
	CreatedAt       time.Time
	ReviewedAt      *time.Time
	ReviewedBy      *int64
}

// TrainingRecord 映射 training_records 表行。
type TrainingRecord struct {
	ID          int64
	UserID      int64
	CourseID    string
	Score       int
	Passed      bool
	Answers     []byte // JSONB
	CompletedAt time.Time
}

// ErrProfileNotFound 是 escort_profiles 查无结果的哨兵。
var ErrProfileNotFound = errors.New("repo: escort profile not found")

// ErrVersionConflict 是乐观锁冲突的哨兵。
var ErrVersionConflict = errors.New("repo: escort profile version conflict")

// ProfileRepo 是 escort_profiles + health_certs + training_records 表的仓储。
type ProfileRepo struct {
	pool *pgxpool.Pool
}

// NewProfileRepo 构造仓储。
func NewProfileRepo(pool *pgxpool.Pool) *ProfileRepo { return &ProfileRepo{pool: pool} }

// Create 插入陪诊师档案；ID / Version / CreatedAt / UpdatedAt 由 DB 回写。
func (r *ProfileRepo) Create(ctx context.Context, p *Profile) error {
	const q = `
		INSERT INTO escort_profiles (user_id, state)
		VALUES ($1, COALESCE(NULLIF($2,''), 'registering'))
		RETURNING id, version, created_at, updated_at`
	return r.pool.QueryRow(ctx, q, p.UserID, p.State).Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
	)
}

// GetByUserID 按 user_id 查找；不存在时返回 (nil, nil)。
func (r *ProfileRepo) GetByUserID(ctx context.Context, userID int64) (*Profile, error) {
	const q = `
		SELECT id, user_id, state, COALESCE(city,''), rating, bad_rate, level, version, created_at, updated_at
		FROM escort_profiles WHERE user_id = $1`
	p := &Profile{}
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&p.ID, &p.UserID, &p.State, &p.City, &p.Rating, &p.BadRate,
		&p.Level, &p.Version, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get profile by user: %w", err)
	}
	return p, nil
}

// UpdateState 推进状态 + version 自增（乐观锁）；version 不匹配返回 ErrVersionConflict。
func (r *ProfileRepo) UpdateState(ctx context.Context, id int64, to string, expectVersion int) error {
	const q = `
		UPDATE escort_profiles
		   SET state = $1, version = version + 1, updated_at = NOW()
		 WHERE id = $2 AND version = $3`
	tag, err := r.pool.Exec(ctx, q, to, id, expectVersion)
	if err != nil {
		return fmt.Errorf("update profile state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrVersionConflict
	}
	return nil
}

// InsertHealthCert 写入一条健康证；ID / CreatedAt 由 DB 回写。
func (r *ProfileRepo) InsertHealthCert(ctx context.Context, c *HealthCert) error {
	const q = `
		INSERT INTO health_certs (user_id, filename, sha256, mime, status)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($6,''), $5))
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, q,
		c.UserID, c.Filename, c.SHA256, c.MIME, c.Status, c.Status,
	).Scan(&c.ID, &c.CreatedAt)
}

// LatestHealthCertByUser 按 created_at DESC 取最近一条；无记录返回 (nil, nil)。
func (r *ProfileRepo) LatestHealthCertByUser(ctx context.Context, userID int64) (*HealthCert, error) {
	const q = `
		SELECT id, user_id, filename, sha256, mime, status, COALESCE(rejection_reason,''),
		       created_at, reviewed_at, reviewed_by
		FROM health_certs WHERE user_id = $1
		ORDER BY created_at DESC LIMIT 1`
	c := &HealthCert{}
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&c.ID, &c.UserID, &c.Filename, &c.SHA256, &c.MIME, &c.Status, &c.RejectionReason,
		&c.CreatedAt, &c.ReviewedAt, &c.ReviewedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("latest health cert: %w", err)
	}
	return c, nil
}

// InsertTrainingRecord 写入培训记录；ID / CompletedAt 由 DB 回写。
func (r *ProfileRepo) InsertTrainingRecord(ctx context.Context, tr *TrainingRecord) error {
	const q = `
		INSERT INTO training_records (user_id, course_id, score, passed, answers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, completed_at`
	return r.pool.QueryRow(ctx, q,
		tr.UserID, tr.CourseID, tr.Score, tr.Passed, tr.Answers,
	).Scan(&tr.ID, &tr.CompletedAt)
}

// ListTrainingByUser 返回用户的全部培训记录（按 completed_at DESC）。
func (r *ProfileRepo) ListTrainingByUser(ctx context.Context, userID int64) ([]*TrainingRecord, error) {
	const q = `
		SELECT id, user_id, course_id, score, passed, answers, completed_at
		FROM training_records WHERE user_id = $1
		ORDER BY completed_at DESC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list training: %w", err)
	}
	defer rows.Close()
	out := make([]*TrainingRecord, 0)
	for rows.Next() {
		tr := &TrainingRecord{}
		if err := rows.Scan(
			&tr.ID, &tr.UserID, &tr.CourseID, &tr.Score, &tr.Passed, &tr.Answers, &tr.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}
```

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestProfileRepo_' ./services/escort/internal/repo/
```
Expected: PASS（8 个测试）

**Step 5: Commit**

```bash
git add services/escort/internal/repo/
git commit -m "feat(escort): repo 加 ProfileRepo (CRUD + 乐观锁 + health_certs + training_records + 8 个集成测试)"
```

---

### Task 4: profile_flow（业务流方法 + 单测）

**Files:**
- Create: `services/escort/internal/service/profile_flow.go`
- Create: `services/escort/internal/service/profile_flow_test.go`
- Modify: `services/escort/internal/service/escort_service.go`
- Modify: `services/escort/internal/service/escort_service_test.go`

**Step 1: 写业务流单测（RED）**

`services/escort/internal/service/profile_flow_test.go`：

```go
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/escort/internal/repo"
	"github.com/growdu/doctors/services/escort/internal/state"
)

// ---------- fakeProfileRepo 实现 repo.ProfileRepo ----------

type fakeProfileRepo struct {
	profiles  map[int64]*repo.Profile
	byUserID  map[int64]int64
	certs     []*repo.HealthCert
	training  []*repo.TrainingRecord
	nextID    int64
	conflicts []float64
}

func newFakeProfileRepo() *fakeProfileRepo {
	return &fakeProfileRepo{
		profiles: map[int64]*repo.Profile{},
		byUserID: map[int64]int64{},
	}
}

func (r *fakeProfileRepo) Create(ctx context.Context, p *repo.Profile) error {
	r.nextID++
	p.ID = r.nextID
	if p.State == "" {
		p.State = "registering"
	}
	p.Version = 0
	r.profiles[p.ID] = p
	r.byUserID[p.UserID] = p.ID
	return nil
}

func (r *fakeProfileRepo) GetByUserID(ctx context.Context, userID int64) (*repo.Profile, error) {
	id, ok := r.byUserID[userID]
	if !ok {
		return nil, nil
	}
	return r.profiles[id], nil
}

func (r *fakeProfileRepo) UpdateState(ctx context.Context, id int64, to string, expectVersion int) error {
	p, ok := r.profiles[id]
	if !ok {
		return repo.ErrProfileNotFound
	}
	if p.Version != expectVersion {
		return repo.ErrVersionConflict
	}
	p.State = to
	p.Version++
	return nil
}

func (r *fakeProfileRepo) InsertHealthCert(ctx context.Context, c *repo.HealthCert) error {
	r.nextID++
	c.ID = r.nextID
	c.CreatedAt = time.Now()
	r.certs = append(r.certs, c)
	return nil
}

func (r *fakeProfileRepo) LatestHealthCertByUser(ctx context.Context, userID int64) (*repo.HealthCert, error) {
	var latest *repo.HealthCert
	for _, c := range r.certs {
		if c.UserID == userID {
			if latest == nil || c.CreatedAt.After(latest.CreatedAt) {
				latest = c
			}
		}
	}
	return latest, nil
}

func (r *fakeProfileRepo) InsertTrainingRecord(ctx context.Context, tr *repo.TrainingRecord) error {
	r.nextID++
	tr.ID = r.nextID
	tr.CompletedAt = time.Now()
	r.training = append(r.training, tr)
	return nil
}

func (r *fakeProfileRepo) ListTrainingByUser(ctx context.Context, userID int64) ([]*repo.TrainingRecord, error) {
	out := make([]*repo.TrainingRecord, 0)
	for _, tr := range r.training {
		if tr.UserID == userID {
			out = append(out, tr)
		}
	}
	return out, nil
}

// ---------- 业务流测试 ----------

// flowHarness 把 facet 与 Service 包在一起。
type flowHarness struct {
	svc    *ProfileFlowService
	repo   *fakeProfileRepo
	pub    *fakeProfilePub
}

func newFlowHarness() *flowHarness {
	r := newFakeProfileRepo()
	p := &fakeProfilePub{}
	svc := NewProfileFlowService(r, p)
	return &flowHarness{svc: svc, repo: r, pub: p}
}

// fakeProfilePub 接 events.Publisher.
type fakeProfilePub struct {
	lastState StateChangedEvent
}

func (p *fakeProfilePub) PublishStateChanged(ctx context.Context, ev StateChangedEvent) error {
	p.lastState = ev
	return nil
}

// StateChangedEvent 是 publisher 发布的最小事件（events 包导类型不同，测试用 alias）。
type StateChangedEvent = struct {
	UserID    int64     `json:"user_id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	OccurredAt time.Time `json:"occurred_at"`
}

// TestRegisterFlow_OK 验证注册 → state=registering。
func TestRegisterFlow_OK(t *testing.T) {
	h := newFlowHarness()
	p, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, int64(100), p.UserID)
	assert.Equal(t, state.StateRegistering, state.State(p.State))
}

// TestRegisterFlow_Duplicate 验证重复注册返回错误。
func TestRegisterFlow_Duplicate(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	_, err = h.svc.RegisterFlow(context.Background(), 100)
	assert.Error(t, err)
}

// TestRealNameAuth_AdvancesState 验证实名 → pending_health_cert。
func TestRealNameAuth_AdvancesState(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)

	require.NoError(t, h.svc.RealNameAuth(context.Background(), 100, "张三", "110101199001011234"))
	got, _ := h.repo.GetByUserID(context.Background(), 100)
	require.NotNil(t, got)
	assert.Equal(t, "pending_health_cert", got.State)
}

// TestRealNameAuth_WrongState 验证非 registering 状态不能实名。
func TestRealNameAuth_WrongState(t *testing.T) {
	h := newFlowHarness()
	// 没注册直接实名 → CodeForbidden
	err := h.svc.RealNameAuth(context.Background(), 100, "张三", "110101199001011234")
	assert.Error(t, err)
}

// TestUploadHealthCert_HashesBase64 验证 SHA256 计算正确。
func TestUploadHealthCert_HashesBase64(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	require.NoError(t, h.svc.RealNameAuth(context.Background(), 100, "张三", "110101199001011234"))

	img := "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD"
	filename := "health_cert.jpg"
	require.NoError(t, h.svc.UploadHealthCert(context.Background(), 100, filename, img))

	cert, _ := h.repo.LatestHealthCertByUser(context.Background(), 100)
	require.NotNil(t, cert)
	sum := sha256.Sum256([]byte(img))
	expected := hex.EncodeToString(sum[:])
	assert.Equal(t, expected, cert.SHA256)
	assert.Equal(t, filename, cert.Filename)
	assert.Equal(t, "image/jpeg", cert.MIME)
}

// TestUploadHealthCert_WrongState 验证非 pending_real_name 不能上传。
func TestUploadHealthCert_WrongState(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	// 还在 registering，不能上传
	err = h.svc.UploadHealthCert(context.Background(), 100, "x.jpg", "data:image/jpeg;base64,xx")
	assert.Error(t, err)
}

// TestCompleteTraining_Pass 验证 5 题答对 4 题通过。
func TestCompleteTraining_Pass(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	require.NoError(t, h.svc.RealNameAuth(context.Background(), 100, "张三", "110101199001011234"))
	require.NoError(t, h.svc.UploadHealthCert(context.Background(), 100, "x.jpg", "data:image/jpeg;base64,xx"))
	// 手动推状态到 pending_training（v1 不接 health_cert 自动审核，admin plan 才有）
	// 这里用 fake repo 直接改
	h.repo.profiles[h.repo.byUserID[100]].State = "pending_training"

	answers := []Answer{
		{QuestionID: 1, Choice: "A"}, // 正确
		{QuestionID: 2, Choice: "B"}, // 正确
		{QuestionID: 3, Choice: "A"}, // 正确
		{QuestionID: 4, Choice: "C"}, // 正确（4/5 = 80%）
		{QuestionID: 5, Choice: "X"}, // 错误
	}
	require.NoError(t, h.svc.CompleteTraining(context.Background(), 100, "escort-basics", answers))
	got, _ := h.repo.GetByUserID(context.Background(), 100)
	require.NotNil(t, got)
	assert.Equal(t, "pending_agreement", got.State)
}

// TestCompleteTraining_Fail 验证 5 题答对 3 题不通过（60% < 80%）。
func TestCompleteTraining_Fail(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	// 直接推到 pending_training
	h.repo.profiles[h.repo.byUserID[100]].State = "pending_training"

	answers := []Answer{
		{QuestionID: 1, Choice: "A"},
		{QuestionID: 2, Choice: "B"},
		{QuestionID: 3, Choice: "A"},
		{QuestionID: 4, Choice: "X"},
		{QuestionID: 5, Choice: "X"},
	}
	err = h.svc.CompleteTraining(context.Background(), 100, "escort-basics", answers)
	assert.Error(t, err)
	got, _ := h.repo.GetByUserID(context.Background(), 100)
	assert.Equal(t, "pending_training", got.State, "失败不应推进状态")
}

// TestSignAgreement_AdvancesState 验证签署 → pending_audit。
func TestSignAgreement_AdvancesState(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	h.repo.profiles[h.repo.byUserID[100]].State = "pending_agreement"

	require.NoError(t, h.svc.SignAgreement(context.Background(), 100, "signature_base64_data"))
	got, _ := h.repo.GetByUserID(context.Background(), 100)
	assert.Equal(t, "pending_audit", got.State)
}

// TestSetOnline_RequiresApproved 验证非 approved 不能上线。
func TestSetOnline_RequiresApproved(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	// 还在 registering
	err = h.svc.SetOnline(context.Background(), 100)
	assert.Error(t, err)
}

// TestSetOnline_FromApproved 验证 approved → online。
func TestSetOnline_FromApproved(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	h.repo.profiles[h.repo.byUserID[100]].State = "approved"

	require.NoError(t, h.svc.SetOnline(context.Background(), 100))
	got, _ := h.repo.GetByUserID(context.Background(), 100)
	assert.Equal(t, "online", got.State)
}

// TestSetOffline_FromOnline 验证 online → offline。
func TestSetOffline_FromOnline(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	h.repo.profiles[h.repo.byUserID[100]].State = "online"

	require.NoError(t, h.svc.SetOffline(context.Background(), 100))
	got, _ := h.repo.GetByUserID(context.Background(), 100)
	assert.Equal(t, "offline", got.State)
}

// TestStateChange_PublishesEvents 验证每次状态推进发事件。
func TestStateChange_PublishesEvents(t *testing.T) {
	h := newFlowHarness()
	_, err := h.svc.RegisterFlow(context.Background(), 100)
	require.NoError(t, err)
	require.NoError(t, h.svc.RealNameAuth(context.Background(), 100, "张三", "110101199001011234"))

	assert.Equal(t, int64(100), h.pub.lastState.UserID)
	assert.Equal(t, "registering", h.pub.lastState.From)
	assert.Equal(t, "pending_health_cert", h.pub.lastState.To)
}

// 兜底编译（time 包在生产代码用到，引用防 unused）。
var _ = errors.New
var _ = strings.TrimSpace
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestRegisterFlow_|TestRealNameAuth_|TestUploadHealthCert_|TestCompleteTraining_|TestSignAgreement_|TestSetOnline_|TestSetOffline_|TestStateChange_' \
  ./services/escort/internal/service/
```
Expected: FAIL — `undefined: NewProfileFlowService`, `undefined: Answer`

**Step 3: 写 profile_flow.go**

`services/escort/internal/service/profile_flow.go`：

```go
// Package service 是 escort-service 的业务编排层（含陪诊师档案业务流）。
//
// 设计要点：
//   - 业务流方法（RegisterFlow / RealNameAuth / ...）操作 escort_profiles 表；
//     老的 Register / SetAvailability / UpdateLocation 走老 escort 表（match 兼容）。
//   - 状态机由 escort/internal/state 守门；非法转换返回 errs.CodeForbidden。
//   - 实名 v1 mock：直接返回通过 + 落 users.real_name_verified。
//   - 健康证 v1：base64 → SHA256；不入 OSS。
//   - 培训 v1：hardcode 5 道题，80% 通过 = 4/5。
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/growdu/doctors/services/escort/internal/repo"
	"github.com/growdu/doctors/services/escort/internal/state"
	"github.com/growdu/doctors/shared/errs"
)

// ProfileRepo 是 escort_profiles + health_certs + training_records 表的最小契约。
type ProfileRepo interface {
	Create(ctx context.Context, p *repo.Profile) error
	GetByUserID(ctx context.Context, userID int64) (*repo.Profile, error)
	UpdateState(ctx context.Context, id int64, to string, expectVersion int) error
	InsertHealthCert(ctx context.Context, c *repo.HealthCert) error
	LatestHealthCertByUser(ctx context.Context, userID int64) (*repo.HealthCert, error)
	InsertTrainingRecord(ctx context.Context, tr *repo.TrainingRecord) error
	ListTrainingByUser(ctx context.Context, userID int64) ([]*repo.TrainingRecord, error)
}

// 编译期确保 repo.ProfileRepo 满足 ProfileRepo 接口。
var _ ProfileRepo = (*repo.ProfileRepo)(nil)

// ProfilePublisher 是事件发布抽象；与 events 包解耦（便于单测 fake）。
type ProfilePublisher interface {
	PublishStateChanged(ctx context.Context, ev StateChangedEvent) error
}

// StateChangedEvent 是状态变更事件（escort-service → 内部 notification / admin audit）。
// 与 shared/contracts.EscortStateChangedEvent 字段一致；本类型用于 service 间解耦。
type StateChangedEvent struct {
	UserID    int64     `json:"user_id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Answer 是单次考核的一道题答案。
type Answer struct {
	QuestionID int    `json:"question_id"`
	Choice     string `json:"choice"`
}

// standardQuiz 是 v1 hardcode 的 5 道题（每题 1 分；4/5 = 80% 通过）。
var standardQuiz = map[string]string{
	"1": "A",
	"2": "B",
	"3": "A",
	"4": "C",
	"5": "B",
}

// ProfileFlowService 是陪诊师档案业务流。
type ProfileFlowService struct {
	profiles ProfileRepo
	pub      ProfilePublisher
	now      func() time.Time
}

// NewProfileFlowService 构造 Service。
func NewProfileFlowService(p ProfileRepo, pub ProfilePublisher) *ProfileFlowService {
	return &ProfileFlowService{profiles: p, pub: pub, now: time.Now}
}

// WithClock 注入时钟（测试用）。
func (s *ProfileFlowService) WithClock(now func() time.Time) *ProfileFlowService {
	s.now = now
	return s
}

// RegisterFlow 注册一个 escort；默认 state=registering。
//   - userID 必须 > 0
//   - 同一 userID 重复注册返回 CodeConflict
func (s *ProfileFlowService) RegisterFlow(ctx context.Context, userID int64) (*repo.Profile, error) {
	if userID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "user_id required")
	}
	if existing, _ := s.profiles.GetByUserID(ctx, userID); existing != nil {
		return nil, errs.New(errs.CodeConflict, "user already registered as escort")
	}
	p := &repo.Profile{UserID: userID, State: string(state.StateRegistering)}
	if err := s.profiles.Create(ctx, p); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create escort profile", err)
	}
	s.publishState(ctx, userID, "", string(state.StateRegistering))
	return p, nil
}

// RealNameAuth 实名认证（v1 mock：直接通过）。
//   - state 必须为 registering
//   - 返回通过 → 推进到 pending_health_cert
func (s *ProfileFlowService) RealNameAuth(ctx context.Context, userID int64, name, idCard string) error {
	name = strings.TrimSpace(name)
	idCard = strings.TrimSpace(idCard)
	if name == "" || len(idCard) < 4 {
		return errs.New(errs.CodeParamInvalid, "name / idCard required")
	}
	return s.transition(ctx, userID, state.StateRegistering, "submit_real_name",
		nil, errs.CodeForbidden, "real name only allowed in registering state")
}

// UploadHealthCert 上传健康证（v1 接收 base64；存 SHA256）。
//   - state 必须为 pending_real_name（自动审核：v1 直接 approved 并推进到 pending_training）
func (s *ProfileFlowService) UploadHealthCert(ctx context.Context, userID int64, filename, imageBase64 string) error {
	filename = strings.TrimSpace(filename)
	imageBase64 = strings.TrimSpace(imageBase64)
	if filename == "" || imageBase64 == "" {
		return errs.New(errs.CodeParamInvalid, "filename / image_base64 required")
	}
	if len(imageBase64) < 16 {
		return errs.New(errs.CodeParamInvalid, "image_base64 too short")
	}
	mime := detectMIME(imageBase64)
	if mime == "" {
		return errs.New(errs.CodeParamInvalid, "unsupported image format")
	}
	// 落 health_certs（status=approved，v1 mock 自动通过）
	sum := sha256.Sum256([]byte(imageBase64))
	cert := &repo.HealthCert{
		UserID: userID, Filename: filename, SHA256: hex.EncodeToString(sum[:]),
		MIME: mime, Status: "approved",
	}
	if err := s.profiles.InsertHealthCert(ctx, cert); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert health cert", err)
	}
	// 推进：pending_real_name → pending_health_cert（提交）→ pending_training（自动审核通过）
	if err := s.transitionQuiet(ctx, userID, state.StatePendingRealName, "real_name_approved"); err != nil {
		return err
	}
	return s.transitionQuiet(ctx, userID, state.StatePendingHealthCert, "health_cert_approved")
}

// CompleteTraining 完成培训考核（v1 hardcode 题库）。
//   - state 必须为 pending_training
//   - 5 道题答对 ≥ 4 道通过；通过则推进到 pending_agreement
func (s *ProfileFlowService) CompleteTraining(ctx context.Context, userID int64, courseID string, answers []Answer) error {
	if len(answers) != len(standardQuiz) {
		return errs.New(errs.CodeParamInvalid, fmt.Sprintf("answers must be %d items", len(standardQuiz)))
	}
	correct := 0
	answersJSON := make([]map[string]any, 0, len(answers))
	for _, a := range answers {
		expected, ok := standardQuiz[fmt.Sprintf("%d", a.QuestionID)]
		if ok && strings.EqualFold(a.Choice, expected) {
			correct++
		}
		answersJSON = append(answersJSON, map[string]any{"q": a.QuestionID, "a": a.Choice})
	}
	score := correct * 100 / len(standardQuiz)
	passed := score >= 80
	answersBytes, _ := jsonMarshal(answersJSON)
	rec := &repo.TrainingRecord{
		UserID: userID, CourseID: courseID, Score: score, Passed: passed, Answers: answersBytes,
	}
	if err := s.profiles.InsertTrainingRecord(ctx, rec); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert training record", err)
	}
	if !passed {
		return errs.New(errs.CodeForbidden, fmt.Sprintf("training failed with %d%% (need 80%%)", score))
	}
	return s.transitionQuiet(ctx, userID, state.StatePendingTraining, "training_passed")
}

// SignAgreement 签署电子协议 → pending_audit。
func (s *ProfileFlowService) SignAgreement(ctx context.Context, userID int64, signatureBase64 string) error {
	if strings.TrimSpace(signatureBase64) == "" {
		return errs.New(errs.CodeParamInvalid, "signature required")
	}
	return s.transition(ctx, userID, state.StatePendingAgreement, "agreement_signed",
		nil, errs.CodeForbidden, "agreement signing only allowed in pending_agreement state")
}

// SetOnline 上线：approved → online。
func (s *ProfileFlowService) SetOnline(ctx context.Context, userID int64) error {
	return s.transition(ctx, userID, state.StateApproved, "go_online",
		nil, errs.CodeForbidden, "go online only allowed in approved state")
}

// SetOffline 下线：online/in_service → offline。
func (s *ProfileFlowService) SetOffline(ctx context.Context, userID int64) error {
	p, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil || p == nil {
		return errs.New(errs.CodeNotFound, "escort profile not found")
	}
	from := state.State(p.State)
	if from == state.StateOnline {
		return s.transitionQuiet(ctx, userID, state.StateOnline, "go_offline")
	}
	if from == state.StateInService {
		return s.transitionQuiet(ctx, userID, state.StateInService, "go_offline")
	}
	return errs.New(errs.CodeForbidden, fmt.Sprintf("go offline only from online/in_service (got %s)", from))
}

// transition 通用状态推进；非法转换返回 forbiddenCode。
func (s *ProfileFlowService) transition(
	ctx context.Context, userID int64, expectFrom state.State, action string,
	_ any, forbiddenCode errs.Code, forbiddenMsg string,
) error {
	to, ok := state.CanTransition(expectFrom, action)
	if !ok {
		return errs.New(errs.CodeInternal, fmt.Sprintf("action %s not declared from %s", action, expectFrom))
	}
	p, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil || p == nil {
		return errs.New(errs.CodeNotFound, "escort profile not found")
	}
	if state.State(p.State) != expectFrom {
		return errs.New(forbiddenCode, forbiddenMsg)
	}
	if err := s.profiles.UpdateState(ctx, p.ID, string(to), p.Version); err != nil {
		if errors.Is(err, repo.ErrVersionConflict) {
			return errs.New(errs.CodeConflict, "version conflict; please retry")
		}
		return errs.Wrap(errs.CodeInternal, "update state", err)
	}
	s.publishState(ctx, userID, p.State, string(to))
	return nil
}

// transitionQuiet 是 transition 的简化版（不带自定义 forbiddenMsg）。
func (s *ProfileFlowService) transitionQuiet(ctx context.Context, userID int64, expectFrom state.State, action string) error {
	to, ok := state.CanTransition(expectFrom, action)
	if !ok {
		return errs.New(errs.CodeInternal, fmt.Sprintf("action %s not declared from %s", action, expectFrom))
	}
	p, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil || p == nil {
		return errs.New(errs.CodeNotFound, "escort profile not found")
	}
	if state.State(p.State) != expectFrom {
		return errs.New(errs.CodeForbidden, fmt.Sprintf("expected state %s, got %s", expectFrom, p.State))
	}
	if err := s.profiles.UpdateState(ctx, p.ID, string(to), p.Version); err != nil {
		if errors.Is(err, repo.ErrVersionConflict) {
			return errs.New(errs.CodeConflict, "version conflict; please retry")
		}
		return errs.Wrap(errs.CodeInternal, "update state", err)
	}
	s.publishState(ctx, userID, p.State, string(to))
	return nil
}

// publishState 发布状态变更事件（best-effort）。
func (s *ProfileFlowService) publishState(ctx context.Context, userID int64, from, to string) {
	if s.pub == nil {
		return
	}
	_ = s.pub.PublishStateChanged(ctx, StateChangedEvent{
		UserID: userID, From: from, To: to, OccurredAt: s.now(),
	})
}

// detectMIME 从 base64 前缀嗅探 image 类型（v1 简版）。
func detectMIME(b64 string) string {
	if strings.HasPrefix(b64, "data:image/jpeg") || strings.HasPrefix(b64, "/9j/") {
		return "image/jpeg"
	}
	if strings.HasPrefix(b64, "data:image/png") || strings.HasPrefix(b64, "iVBOR") {
		return "image/png"
	}
	return ""
}

// jsonMarshal 把 v 序列化为 JSON 字节；内部 helper 避免顶层引 encoding/json。
func jsonMarshal(v any) ([]byte, error) {
	// 用 fmt.Sprintf 简化；只用于内部 answers 落库（结构简单）。
	// 真要严谨可换 encoding/json；本文件已在 stdlib json encoding context 下。
	type buf []byte
	_ = buf(nil)
	// 直接调 encoding/json：
	return jsonMarshalStd(v)
}
```

> **注意**：上面 `jsonMarshal` 用了占位实现。实施时**改用** `encoding/json`：
>
> ```go
> import "encoding/json"
> ...
> func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
> ```
>
> 单测中 `answersJSON` 类型 `[]map[string]any` 可直接序列化。

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/service/
```
Expected: PASS（既有 12 个 + 新加 13 个 = 25 个）

**Step 5: 扩展既有 service 加 GetMyProfile + ListTrainingCourses + ListMyReviews**

修改 `services/escort/internal/service/escort_service.go` 末尾追加：

```go
// ---------- 2026-09-24 escort-business plan 扩展：me/* endpoints ----------

// TrainingCourse 是 v1 培训课程静态视图（前端 hardcode 也行；后端提供便于统一）。
type TrainingCourse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DurationMin int    `json:"duration_min"`
}

// standardCourses 是 v1 课程列表（hardcode；admin 可后续配置化）。
var standardCourses = []TrainingCourse{
	{ID: "escort-basics", Title: "陪诊师基础", Description: "服务流程 + 注意事项", DurationMin: 30},
	{ID: "first-aid", Title: "急救常识", Description: "SOS 触发条件 + 现场处理", DurationMin: 20},
}

// Review 是陪诊师收到的评价摘要（v1 stub；review-service 完整版在 review plan）。
type Review struct {
	ID         int64     `json:"id"`
	OrderID    int64     `json:"order_id"`
	ReviewerID int64     `json:"reviewer_id"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ProfileFlowReader 是 escort_service 用来读 profile 数据的接口（解耦）。
type ProfileFlowReader interface {
	GetByUserID(ctx context.Context, userID int64) (*repo.Profile, error)
	ListTrainingByUser(ctx context.Context, userID int64) ([]*repo.TrainingRecord, error)
}

// SetProfileReader 注入 reader（cmd/main.go 装配）。
func (s *Service) SetProfileReader(r ProfileFlowReader) *Service { s.profileReader = r; return s }

// 在 Service struct 加字段：
//   profileReader ProfileFlowReader

// GetMyProfile 取当前陪诊师档案。
func (s *Service) GetMyProfile(ctx context.Context, userID int64) (*repo.Profile, error) {
	if s.profileReader == nil {
		return nil, errs.New(errs.CodeUnavailable, "profile reader not wired")
	}
	p, err := s.profileReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "get profile", err)
	}
	if p == nil {
		return nil, errs.New(errs.CodeNotFound, "escort profile not found")
	}
	return p, nil
}

// ListTrainingCourses 返回 v1 课程列表（hardcode）。
func (s *Service) ListTrainingCourses() []TrainingCourse { return standardCourses }

// ListMyReviews 返回陪诊师收到的评价（v1 stub：从 mock 列表返回；review plan 接通）。
func (s *Service) ListMyReviews(ctx context.Context, userID int64) ([]Review, error) {
	// v1 stub：返回空列表；review plan 完成后改为读 review-service
	return []Review{}, nil
}
```

> 修改 Service struct + New 时给字段默认值 nil；既有 12 个测试不动。

修改 `services/escort/internal/service/escort_service_test.go` 末尾追加：

```go
// TestGetMyProfile_NotFound 验证未注册时返回 NotFound。
func TestGetMyProfile_NotFound(t *testing.T) {
	s := New(newFakeRepo(), &fakePub{})
	s.SetProfileReader(newFakeProfileReader())
	_, err := s.GetMyProfile(context.Background(), 99999)
	assert.Error(t, err)
}

// TestListTrainingCourses_OK 验证返回 2 门课程。
func TestListTrainingCourses_OK(t *testing.T) {
	s := New(newFakeRepo(), &fakePub{})
	courses := s.ListTrainingCourses()
	assert.Len(t, courses, 2)
	assert.Equal(t, "escort-basics", courses[0].ID)
}

// TestListMyReviews_Empty 验证 v1 stub 返回空列表。
func TestListMyReviews_Empty(t *testing.T) {
	s := New(newFakeRepo(), &fakePub{})
	reviews, err := s.ListMyReviews(context.Background(), 100)
	require.NoError(t, err)
	assert.Empty(t, reviews)
}

// fakeProfileReader 满足接口最小实现。
type fakeProfileReader struct{}

func newFakeProfileReader() *fakeProfileReader { return &fakeProfileReader{} }
func (r *fakeProfileReader) GetByUserID(ctx context.Context, userID int64) (*repo.Profile, error) {
	return nil, nil
}
func (r *fakeProfileReader) ListTrainingByUser(ctx context.Context, userID int64) ([]*repo.TrainingRecord, error) {
	return nil, nil
}
```

**Step 6: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/service/
```
Expected: PASS（既有 12 + 新增 13 业务流 + 新增 3 me/* = 28 个）

**Step 7: Commit**

```bash
git add services/escort/internal/service/
git commit -m "feat(escort): profile_flow 业务流 + 11 态状态机守门 + 实名 mock + 健康证 SHA256 + 培训 80% 通过 + me/* endpoints (25+ 单测)"
```

---

### Task 5: shared/contracts + escort events.Publisher

**Files:**
- Modify: `shared/contracts/events.go`
- Modify: `shared/contracts/contracts_test.go`
- Create: `services/escort/internal/events/publisher.go`
- Create: `services/escort/internal/events/publisher_test.go`

**Step 1: 写 events.go 测试**

修改 `shared/contracts/contracts_test.go`：

```go
// TestEscortStateChangedEvent_RoundTrip 验证序列化可逆。
func TestEscortStateChangedEvent_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ev := EscortStateChangedEvent{
		UserID: 100, From: "registering", To: "pending_real_name", OccurredAt: now,
	}
	data, err := json.Marshal(ev)
	require.NoError(t, err)
	var got EscortStateChangedEvent
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, ev, got)
}

// TestTopicConstants_EscortStateChanged 验证 topic 常量。
// 在 TestTopicConstants 里追加：
//   assert.Equal(t, "escort.state_changed", TopicEscortStateChanged)
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestEscortStateChangedEvent|TestTopicConstants' ./shared/contracts/
```
Expected: FAIL — `undefined: EscortStateChangedEvent`

**Step 3: 修改 events.go**

```go
// 在 const 块追加（与 TopicEscortAvailable 紧邻）：
TopicEscortStateChanged = "escort.state_changed"

// 在 EscortAvailableEvent 定义后追加：
// EscortStateChangedEvent 陪诊师档案状态变更（escort-service → notification / admin audit）。
type EscortStateChangedEvent struct {
	UserID     int64     `json:"user_id"`
	From       string    `json:"from,omitempty"`
	To         string    `json:"to"`
	OccurredAt time.Time `json:"occurred_at"`
}
```

**Step 4: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./shared/contracts/
```
Expected: PASS

**Step 5: 写 publisher.go**

`services/escort/internal/events/publisher.go`：

```go
// Package events 是 escort-service 的事件发布层。
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/contracts"
)

// Publisher 抽象 escort 事件。
type Publisher interface {
	PublishStateChanged(ctx context.Context, ev service.StateChangedEvent) error
	Close() error
}

// KafkaPublisher 用 kafka-go writer 写 topic。
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher 构造 publisher。
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr: kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
			BatchTimeout: 50 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
			Async: false,
		},
	}
}

// Close 关闭 writer。
func (p *KafkaPublisher) Close() error { return p.writer.Close() }

// PublishStateChanged 发 escort.state_changed。
func (p *KafkaPublisher) PublishStateChanged(ctx context.Context, ev service.StateChangedEvent) error {
	if p == nil || p.writer == nil {
		return errors.New("publisher: writer is nil")
	}
	wire := contracts.EscortStateChangedEvent{
		UserID: ev.UserID, From: ev.From, To: ev.To, OccurredAt: ev.OccurredAt,
	}
	data, err := json.Marshal(wire)
	if err != nil {
		return fmt.Errorf("marshal escort state: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: contracts.TopicEscortStateChanged,
		Key:   []byte(strconv.FormatInt(ev.UserID, 10)),
		Value: data,
		Time:  time.Now(),
	})
}

// NopPublisher 是测试 / dev 占位。
type NopPublisher struct {
	Count int
}

// PublishStateChanged 计数。
func (p *NopPublisher) PublishStateChanged(_ context.Context, _ service.StateChangedEvent) error {
	p.Count++
	return nil
}

// Close 无资源。
func (p *NopPublisher) Close() error { return nil }
```

`services/escort/internal/events/publisher_test.go`：

```go
package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/escort/internal/service"
)

// TestNopPublisher_PublishStateChanged_Counts 验证计数。
func TestNopPublisher_PublishStateChanged_Counts(t *testing.T) {
	p := &NopPublisher{}
	require.NoError(t, p.PublishStateChanged(context.Background(), service.StateChangedEvent{
		UserID: 1, From: "registering", To: "pending_real_name", OccurredAt: time.Now(),
	}))
	assert.Equal(t, 1, p.Count)
}

// TestNopPublisher_Close 不报错。
func TestNopPublisher_Close(t *testing.T) {
	assert.NoError(t, (&NopPublisher{}).Close())
}
```

**Step 6: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/events/ ./shared/contracts/
```
Expected: PASS

**Step 7: Commit**

```bash
git add shared/contracts/ services/escort/internal/events/
git commit -m "feat(contracts+escort): EscortStateChangedEvent + TopicEscortStateChanged + KafkaPublisher + NopPublisher (2 单测)"
```

---

### Task 6: escort handler 8 endpoints + 测试

**Files:**
- Create: `services/escort/internal/handler/escort_business.go`
- Create: `services/escort/internal/handler/escort_business_test.go`
- Modify: `services/escort/internal/handler/escort.go`

**Step 1: 写 handler 单测（RED）**

`services/escort/internal/handler/escort_business_test.go`：

```go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/services/escort/internal/repo"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/services/escort/internal/state"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "test-secret-escort"

// fakeFlow 是 ProfileFlowService 的 fake（满足 handler 接口）。
type fakeFlow struct {
	registerCalled   bool
	realNameCalled   bool
	uploadCalled     bool
	trainingCalled   bool
	signCalled       bool
	goOnlineCalled   bool
	goOfflineCalled  bool
	registerErr      error
	realNameErr      error
	uploadErr        error
	trainingErr      error
	signErr          error
	goOnlineErr      error
	goOfflineErr     error
	lastRegisterUser int64
}

func (f *fakeFlow) RegisterFlow(ctx context.Context, userID int64) (*repo.Profile, error) {
	f.registerCalled = true; f.lastRegisterUser = userID
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	return &repo.Profile{ID: 1, UserID: userID, State: "registering", Version: 0}, nil
}
func (f *fakeFlow) RealNameAuth(ctx context.Context, userID int64, name, idCard string) error {
	f.realNameCalled = true; return f.realNameErr
}
func (f *fakeFlow) UploadHealthCert(ctx context.Context, userID int64, filename, b64 string) error {
	f.uploadCalled = true; return f.uploadErr
}
func (f *fakeFlow) CompleteTraining(ctx context.Context, userID int64, courseID string, answers []service.Answer) error {
	f.trainingCalled = true; return f.trainingErr
}
func (f *fakeFlow) SignAgreement(ctx context.Context, userID int64, sig string) error {
	f.signCalled = true; return f.signErr
}
func (f *fakeFlow) SetOnline(ctx context.Context, userID int64) error {
	f.goOnlineCalled = true; return f.goOnlineErr
}
func (f *fakeFlow) SetOffline(ctx context.Context, userID int64) error {
	f.goOfflineCalled = true; return f.goOfflineErr
}

// fakeReader 是 ProfileFlowReader 的 fake（满足 service 测试用接口）。
type fakeReader struct {
	profile *repo.Profile
}

func (r *fakeReader) GetByUserID(ctx context.Context, userID int64) (*repo.Profile, error) {
	if r.profile != nil && r.profile.UserID == userID {
		return r.profile, nil
	}
	return nil, nil
}
func (r *fakeReader) ListTrainingByUser(ctx context.Context, userID int64) ([]*repo.TrainingRecord, error) {
	return nil, nil
}

func signTestToken(t *testing.T, uid int64, role string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uid, "role": role, "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return signed
}

// newTestServer 构造 handler + gin engine + 中间件。
func newBusinessServer(flowSvc *fakeFlow, reader ProfileFlowReader) (*gin.Engine, *service.Service) {
	escSvc := service.New(nil, nil)
	if reader != nil {
		escSvc.SetProfileReader(reader)
	}
	h := NewBusinessHandler(flowSvc, escSvc)
	r := gin.New()
	v1 := r.Group("/api/v1", middleware.Auth(testSecret))
	h.RegisterRoutes(v1)
	return r, escSvc
}

func doJSON(t *testing.T, r *gin.Engine, method, path, token string, payload any) *httpx.Resp[map[string]any] {
	t.Helper()
	var body *bytes.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return &resp
}

// TestRegister_OK 验证 POST /escorts/register。
func TestRegister_OK(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPost, "/api/v1/escorts/register", tok, nil)
	assert.Equal(t, 0, resp.Code)
	assert.True(t, flow.registerCalled)
	assert.Equal(t, int64(100), flow.lastRegisterUser)
}

// TestRealName_OK 验证 POST /escorts/real-name/auth。
func TestRealName_OK(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPost, "/api/v1/escorts/real-name/auth", tok, map[string]any{
		"name": "张三", "id_card": "110101199001011234",
	})
	assert.Equal(t, 0, resp.Code)
	assert.True(t, flow.realNameCalled)
}

// TestRealName_BadJSON 验证缺字段返回错误码。
func TestRealName_BadJSON(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPost, "/api/v1/escorts/real-name/auth", tok, map[string]any{})
	assert.NotEqual(t, 0, resp.Code)
	assert.False(t, flow.realNameCalled)
}

// TestHealthCertUpload_OK 验证 POST /escorts/health-cert/upload。
func TestHealthCertUpload_OK(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPost, "/api/v1/escorts/health-cert/upload", tok, map[string]any{
		"filename": "cert.jpg",
		"image_base64": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD",
	})
	assert.Equal(t, 0, resp.Code)
	assert.True(t, flow.uploadCalled)
}

// TestTrainingComplete_OK 验证 POST /escorts/training/complete。
func TestTrainingComplete_OK(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPost, "/api/v1/escorts/training/complete", tok, map[string]any{
		"course_id": "escort-basics",
		"answers": []map[string]any{
			{"question_id": 1, "choice": "A"},
			{"question_id": 2, "choice": "B"},
			{"question_id": 3, "choice": "A"},
			{"question_id": 4, "choice": "C"},
			{"question_id": 5, "choice": "B"},
		},
	})
	assert.Equal(t, 0, resp.Code)
	assert.True(t, flow.trainingCalled)
}

// TestMeProfile_OK 验证 GET /escorts/me/profile。
func TestMeProfile_OK(t *testing.T) {
	reader := &fakeReader{profile: &repo.Profile{
		ID: 1, UserID: 100, State: string(state.StateApproved), Rating: 5.0, Version: 1,
	}}
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, reader)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodGet, "/api/v1/escorts/me/profile", tok, nil)
	assert.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "approved", resp.Data["state"])
}

// TestMeProfile_NotFound 验证未注册返回 NotFound。
func TestMeProfile_NotFound(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, &fakeReader{}) // reader 永远返回 nil
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodGet, "/api/v1/escorts/me/profile", tok, nil)
	assert.NotEqual(t, 0, resp.Code)
}

// TestMeStatus_Online 验证 PUT /escorts/me/status (online)。
func TestMeStatus_Online(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPut, "/api/v1/escorts/me/status", tok, map[string]any{
		"online": true,
	})
	assert.Equal(t, 0, resp.Code)
	assert.True(t, flow.goOnlineCalled)
}

// TestMeStatus_Offline 验证 PUT /escorts/me/status (offline)。
func TestMeStatus_Offline(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodPut, "/api/v1/escorts/me/status", tok, map[string]any{
		"online": false,
	})
	assert.Equal(t, 0, resp.Code)
	assert.True(t, flow.goOfflineCalled)
}

// TestMeTrainingCourses_OK 验证 GET /escorts/me/training-courses。
func TestMeTrainingCourses_OK(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodGet, "/api/v1/escorts/me/training-courses", tok, nil)
	assert.Equal(t, 0, resp.Code)
}

// TestMeReviews_OK 验证 GET /escorts/me/reviews。
func TestMeReviews_OK(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "escort")
	resp := doJSON(t, r, http.MethodGet, "/api/v1/escorts/me/reviews", tok, nil)
	assert.Equal(t, 0, resp.Code)
}

// TestEscortOnly 验证 patient 不能访问 escort endpoint。
func TestRegister_PatientForbidden(t *testing.T) {
	flow := &fakeFlow{}
	r, _ := newBusinessServer(flow, nil)
	tok := signTestToken(t, 100, "patient")
	resp := doJSON(t, r, http.MethodPost, "/api/v1/escorts/register", tok, nil)
	assert.NotEqual(t, 0, resp.Code)
}

// 兜底编译（errs 包用于编译期保证；time 包同上）。
var _ = errs.CodeOK
var _ = time.Now
```

**Step 2: 跑测试确认失败**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestRegister_OK|TestRealName_|TestHealthCertUpload|TestTrainingComplete|TestMeProfile|TestMeStatus|TestMeTrainingCourses|TestMeReviews|TestEscortOnly' \
  ./services/escort/internal/handler/
```
Expected: FAIL — `undefined: NewBusinessHandler`

**Step 3: 写 escort_business.go**

`services/escort/internal/handler/escort_business.go`：

```go
// Package handler 把 escort-service 暴露为 REST 接口（含 8 个业务流 endpoint）。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// FlowService 是 escort-service 业务流的接口（profile_flow.go 实现 + handler fake）。
type FlowService interface {
	RegisterFlow(ctx interface{ Done() <-chan struct{} }, userID int64) (*profileResult, error)
}

// profileResult 是 handler 用最小结构（避免反向依赖 repo.Profile）。
type profileResult = struct {
	ID      int64  `json:"id"`
	UserID  int64  `json:"user_id"`
	State   string `json:"state"`
	Version int    `json:"version"`
}
```

> **注**：上面 `FlowService` 用 `interface{ Done() <-chan struct{} }` 是为兼容 stdlib `context.Context`。**实施时**改为：
>
> ```go
> import "context"
> type FlowService interface {
>     RegisterFlow(ctx context.Context, userID int64) (*service.ProfileResult, error)
>     RealNameAuth(ctx context.Context, userID int64, name, idCard string) error
>     UploadHealthCert(ctx context.Context, userID int64, filename, b64 string) error
>     CompleteTraining(ctx context.Context, userID int64, courseID string, answers []service.Answer) error
>     SignAgreement(ctx context.Context, userID int64, sig string) error
>     SetOnline(ctx context.Context, userID int64) error
>     SetOffline(ctx context.Context, userID int64) error
> }
> ```
>
> 其中 `service.ProfileResult` 在 `profile_flow.go` 加：
>
> ```go
> type ProfileResult struct { ID, UserID int64; State string; Version int }
> ```
>
> handler 拿到后转 `gin.H{"id": r.ID, ...}` 即可。

继续写 handler：

```go
// BusinessHandler 是 8 个新 endpoint 的 handler。
type BusinessHandler struct {
	flow  FlowService
	escort *service.Service // 复用 me/* 三个
}

// NewBusinessHandler 构造 handler。
func NewBusinessHandler(flow FlowService, escort *service.Service) *BusinessHandler {
	return &BusinessHandler{flow: flow, escort: escort}
}

// RegisterRoutes 把业务流路由挂到 /escorts。
func (h *BusinessHandler) RegisterRoutes(r gin.IRouter) {
	e := r.Group("/escorts")
	e.POST("/register", h.register)
	e.POST("/real-name/auth", h.realNameAuth)
	e.POST("/health-cert/upload", h.uploadHealthCert)
	e.POST("/training/complete", h.trainingComplete)
	e.POST("/agreement/sign", h.signAgreement) // P1 留存口（spec 未列；v1 提供但不进 UI）
	e.GET("/me/profile", h.meProfile)
	e.PUT("/me/status", h.meStatus)
	e.GET("/me/training-courses", h.meTrainingCourses)
	e.GET("/me/reviews", h.meReviews)
}

// escortOnly 校验 role=escort（patient 拒绝）。
func escortOnly(c *gin.Context) (int64, bool) {
	uid := middleware.UserID(c)
	if uid == 0 {
		httpx.Fail(c, int(errs.CodeUnauthorized), "no user")
		return 0, false
	}
	if role := middleware.Role(c); role != "escort" {
		httpx.Fail(c, int(errs.CodeForbidden), "only escort can access")
		return 0, false
	}
	return uid, true
}

// register POST /api/v1/escorts/register
func (h *BusinessHandler) register(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	p, err := h.flow.RegisterFlow(c.Request.Context(), uid)
	if err != nil { respondError(c, err); return }
	httpx.OK(c, gin.H{"id": p.ID, "user_id": p.UserID, "state": p.State, "version": p.Version})
}

type realNameReq struct {
	Name   string `json:"name" binding:"required"`
	IDCard string `json:"id_card" binding:"required"`
}

// realNameAuth POST /api/v1/escorts/real-name/auth
func (h *BusinessHandler) realNameAuth(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	var req realNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.flow.RealNameAuth(c.Request.Context(), uid, req.Name, req.IDCard); err != nil {
		respondError(c, err); return
	}
	httpx.OK[any](c, gin.H{"user_id": uid, "verified": true})
}

type healthCertReq struct {
	Filename    string `json:"filename" binding:"required"`
	ImageBase64 string `json:"image_base64" binding:"required"`
}

// uploadHealthCert POST /api/v1/escorts/health-cert/upload
func (h *BusinessHandler) uploadHealthCert(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	var req healthCertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.flow.UploadHealthCert(c.Request.Context(), uid, req.Filename, req.ImageBase64); err != nil {
		respondError(c, err); return
	}
	httpx.OK[any](c, gin.H{"user_id": uid, "status": "approved"})
}

type trainingReq struct {
	CourseID string             `json:"course_id" binding:"required"`
	Answers  []service.Answer   `json:"answers" binding:"required"`
}

// trainingComplete POST /api/v1/escorts/training/complete
func (h *BusinessHandler) trainingComplete(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	var req trainingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.flow.CompleteTraining(c.Request.Context(), uid, req.CourseID, req.Answers); err != nil {
		respondError(c, err); return
	}
	httpx.OK[any](c, gin.H{"user_id": uid, "passed": true})
}

type signReq struct {
	SignatureBase64 string `json:"signature_base64" binding:"required"`
}

// signAgreement POST /api/v1/escorts/agreement/sign
func (h *BusinessHandler) signAgreement(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	var req signReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.flow.SignAgreement(c.Request.Context(), uid, req.SignatureBase64); err != nil {
		respondError(c, err); return
	}
	httpx.OK[any](c, gin.H{"user_id": uid, "state": "pending_audit"})
}

// meProfile GET /api/v1/escorts/me/profile
func (h *BusinessHandler) meProfile(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	p, err := h.escort.GetMyProfile(c.Request.Context(), uid)
	if err != nil { respondError(c, err); return }
	httpx.OK(c, gin.H{
		"id": p.ID, "user_id": p.UserID, "state": p.State,
		"city": p.City, "rating": p.Rating, "level": p.Level, "version": p.Version,
	})
}

type statusReq struct {
	Online bool `json:"online"`
}

// meStatus PUT /api/v1/escorts/me/status
func (h *BusinessHandler) meStatus(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	var err error
	if req.Online {
		err = h.flow.SetOnline(c.Request.Context(), uid)
	} else {
		err = h.flow.SetOffline(c.Request.Context(), uid)
	}
	if err != nil { respondError(c, err); return }
	state := "online"
	if !req.Online { state = "offline" }
	httpx.OK[any](c, gin.H{"user_id": uid, "state": state})
}

// meTrainingCourses GET /api/v1/escorts/me/training-courses
func (h *BusinessHandler) meTrainingCourses(c *gin.Context) {
	if _, ok := escortOnly(c); !ok { return }
	courses := h.escort.ListTrainingCourses()
	httpx.OK(c, gin.H{"courses": courses})
}

// meReviews GET /api/v1/escorts/me/reviews
func (h *BusinessHandler) meReviews(c *gin.Context) {
	uid, ok := escortOnly(c); if !ok { return }
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	reviews, err := h.escort.ListMyReviews(c.Request.Context(), uid)
	if err != nil { respondError(c, err); return }
	if reviews == nil {
		reviews = []service.Review{}
	}
	httpx.OK(c, gin.H{"reviews": reviews, "limit": limit})
}
```

**Step 4: 修改既有 handler/escort.go 挂新路由**

修改 `services/escort/internal/handler/escort.go` 的 `RegisterRoutes`，**追加**：

```go
// 在 e.GET("/:id", h.Get) 之前或之后追加：
// business 路由（陪诊师档案业务流）
biz := NewBusinessHandler(flowImpl, escSvc)
e.POST("/register", biz.register)
e.POST("/real-name/auth", biz.realNameAuth)
e.POST("/health-cert/upload", biz.uploadHealthCert)
e.POST("/training/complete", biz.trainingComplete)
e.POST("/agreement/sign", biz.signAgreement)
e.GET("/me/profile", biz.meProfile)
e.PUT("/me/status", biz.meStatus)
e.GET("/me/training-courses", biz.meTrainingCourses)
e.GET("/me/reviews", biz.meReviews)
```

> **改造要点**：现有 `RegisterRoutes(r gin.IRouter)` 接收 router group；新增构造 business handler 时需要 flow + escort svc 引用。把 Handler struct 扩字段：

```go
type Handler struct {
	svc    *service.Service
	flow   service.ProfileFlowService  // 业务流（cmd/main.go 装配）
}
```

> **实施时**调整 `New(svc)` 接受 flow 参数；既有 `RegisterRoutes` 调用点（cmd/main.go + 测试）同步调整。

**Step 5: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/escort/internal/handler/
```
Expected: PASS（既有 + 12 个新 endpoint 测试）

**Step 6: Commit**

```bash
git add services/escort/internal/handler/
git commit -m "feat(escort): 8 个业务 endpoint (register/real-name/health-cert/training/me/profile/me/status/me/reviews/me/training-courses) + 12 handler 单测"
```

---

### Task 7: order-service 扩展（抢单池 + checkin/checkout）

**Files:**
- Modify: `services/order/internal/repo/order_repo.go`
- Modify: `services/order/internal/repo/order_repo_integration_test.go`
- Modify: `services/order/internal/service/order_service.go`
- Modify: `services/order/internal/service/order_service_test.go`
- Modify: `services/order/internal/handler/order.go`
- Modify: `services/order/internal/handler/order_test.go`

**Step 1: 加 repo.ListForEscort + 集成测试**

修改 `services/order/internal/repo/order_repo.go` 末尾追加：

```go
// ListForEscort 抢单池：status=matching 且 escort_id IS NULL（未被人接）的订单，按 created_at DESC。
func (r *OrderRepo) ListForEscort(ctx context.Context, status string, limit, offset int) ([]*Order, error) {
	const q = baseSelect + ` WHERE status = $1 AND escort_id IS NULL AND deleted_at IS NULL
	                         ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list for escort: %w", err)
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
```

修改 `OrderRepo` 接口（在 `OrderRepo` 定义里加）：

```go
ListForEscort(ctx context.Context, status string, limit, offset int) ([]*Order, error)
```

修改 `services/order/internal/repo/order_repo_integration_test.go` 末尾追加：

```go
// TestOrderRepo_ListForEscort_OK 验证抢单池列出 matching 且未接的订单。
func TestOrderRepo_ListForEscort_OK(t *testing.T) {
	pool := setupOrderPool(t)
	patient := seedOrderPoolUser(t, pool, "13800222001", "patient")
	escort := seedOrderPoolUser(t, pool, "13800222002", "escort")
	r := NewOrderRepo(pool)

	// 3 单：1 matching / 1 matching 但已接 / 1 paid
	o1 := seedOrderPoolOrder(t, pool, patient, "matching", nil)
	o2 := seedOrderPoolOrder(t, pool, patient, "matching", &escort)
	o3 := seedOrderPoolOrder(t, pool, patient, "paid", nil)

	list, err := r.ListForEscort(context.Background(), "matching", 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, o1, list[0].ID, "只应返回未接的 matching 订单")
	_ = o2; _ = o3
}

// TestOrderRepo_ListForEscort_ExcludesOtherStatuses 验证只查指定 status。
func TestOrderRepo_ListForEscort_ExcludesOtherStatuses(t *testing.T) {
	pool := setupOrderPool(t)
	patient := seedOrderPoolUser(t, pool, "13800222003", "patient")
	r := NewOrderRepo(pool)

	seedOrderPoolOrder(t, pool, patient, "paid", nil)
	seedOrderPoolOrder(t, pool, patient, "created", nil)

	list, err := r.ListForEscort(context.Background(), "matching", 10, 0)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// seedOrderPoolUser / seedOrderPoolOrder 帮助函数（参考既有 seedUser / seedOrder）。
func seedOrderPoolUser(t *testing.T, pool *pgxpool.Pool, phone, role string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (phone, role) VALUES ($1, $2) RETURNING id`, phone, role).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedOrderPoolOrder(t *testing.T, pool *pgxpool.Pool, patientID int64, status string, escortID *int64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO orders (order_no, patient_id, escort_id, hospital_id, package_id,
		                    service_start_at, amount, final_amount, status)
		VALUES (gen_random_uuid()::text, $1, $2, 1, 1, NOW() + INTERVAL '1 day', 100, 100, $3)
		RETURNING id`, patientID, escortID, status).Scan(&id)
	require.NoError(t, err)
	return id
}
```

**Step 2: 加 service.CheckIn / CheckOut / ListForEscort + 单测**

修改 `services/order/internal/service/order_service.go`：

- `Order` struct 在既有基础上加 `CheckinAt *time.Time` + `CheckoutAt *time.Time`
- `baseSelect` 改：增加 `COALESCE(checkin_at, NULL), COALESCE(checkout_at, NULL)`
- 注：当前表没这两列；**Task 7 先不加 DB 字段**（v1 仅作为 service 层概念，handler 返回时硬编码）；如需落库，需追加 `migrations/0007_orders_checkin.up.sql`。本 plan **保持 DB 不变**，service 层用 `time.Now()` 计算响应字段
- `OrderRepo` 接口加 `ListForEscort`
- Service 加方法：

```go
// ListForEscort 抢单池查询。
func (s *Service) ListForEscort(ctx context.Context, status string, limit, offset int) ([]*repo.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if status == "" {
		status = string(state.StatusMatching)
	}
	return s.orders.ListForEscort(ctx, status, limit, offset)
}

// CheckIn 陪诊师到院签到：accepted → in_service；v1 mock GPS 只校验 lat/lng 范围。
func (s *Service) CheckIn(ctx context.Context, orderID, escortID int64, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return errs.New(errs.CodeParamInvalid, "lat/lng out of range")
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.EscortID == nil || *o.EscortID != escortID {
		return errs.New(errs.CodeForbidden, "only assigned escort can check in")
	}
	from := state.Status(o.Status)
	to := state.StatusInService
	if !state.CanTransition(from, to) {
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot check in from %s", from))
	}
	if err := s.orders.UpdateStatus(ctx, orderID, string(to), o.Version, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	fromStr := string(from)
	actor := escortID
	if err := s.orders.InsertEvent(ctx, orderID, &fromStr, string(to), &actor, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}
	return nil
}

// CheckOut 陪诊师完成打卡：in_service → completed；note 可选。
func (s *Service) CheckOut(ctx context.Context, orderID, escortID int64, note string) error {
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			return errs.New(errs.CodeNotFound, "order not found")
		}
		return errs.Wrap(errs.CodeInternal, "find order", err)
	}
	if o.EscortID == nil || *o.EscortID != escortID {
		return errs.New(errs.CodeForbidden, "only assigned escort can check out")
	}
	from := state.Status(o.Status)
	to := state.StatusCompleted
	if !state.CanTransition(from, to) {
		return errs.New(errs.CodeConflict, fmt.Sprintf("cannot check out from %s", from))
	}
	if err := s.orders.UpdateStatus(ctx, orderID, string(to), o.Version, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	fromStr := string(from)
	actor := escortID
	if err := s.orders.InsertEvent(ctx, orderID, &fromStr, string(to), &actor, nil); err != nil {
		return errs.Wrap(errs.CodeInternal, "insert event", err)
	}
	return nil
}
```

修改既有单测 `services/order/internal/service/order_service_test.go` 的 `fakeOrderRepo` 加 `ListForEscort`：

```go
func (r *fakeOrderRepo) ListForEscort(ctx context.Context, status string, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.Status == status && o.EscortID == nil {
			out = append(out, o)
		}
	}
	return out, nil
}
```

末尾追加新测试：

```go
// TestListForEscort_OK 验证抢单池返回 matching 未接的订单。
func TestListForEscort_OK(t *testing.T) {
	svc := New(newFakeOrderRepo(), &fakeUserLookup{users: map[int64]*UserSnapshot{
		1: {ID: 1, Role: "patient", RealNameVerified: true},
	}})
	_ = svc
	// 直接用 fakeOrderRepo 数据
	fr := newFakeOrderRepo()
	fr.Create(context.Background(), &repo.Order{PatientID: 1, Status: "matching", OrderNo: "M1"})
	fr.Create(context.Background(), &repo.Order{PatientID: 1, Status: "paid", OrderNo: "P1"})
	svc2 := New(fr, &fakeUserLookup{users: map[int64]*UserSnapshot{1: {ID: 1, Role: "patient", RealNameVerified: true}}})
	list, err := svc2.ListForEscort(context.Background(), "matching", 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

// TestCheckIn_OK 验证 accepted → in_service。
func TestCheckIn_OK(t *testing.T) {
	fr := newFakeOrderRepo()
	escort := int64(2)
	fr.Create(context.Background(), &repo.Order{PatientID: 1, EscortID: &escort, Status: "accepted", OrderNo: "X"})
	svc := New(fr, &fakeUserLookup{users: map[int64]*UserSnapshot{
		1: {ID: 1, Role: "patient", RealNameVerified: true},
	}})
	require.NoError(t, svc.CheckIn(context.Background(), 1, 2, 39.9, 116.4))
	o, _ := fr.FindByID(context.Background(), 1)
	assert.Equal(t, "in_service", o.Status)
}

// TestCheckIn_InvalidLat 验证非法 lat 返回错误。
func TestCheckIn_InvalidLat(t *testing.T) {
	fr := newFakeOrderRepo()
	escort := int64(2)
	fr.Create(context.Background(), &repo.Order{PatientID: 1, EscortID: &escort, Status: "accepted", OrderNo: "X"})
	svc := New(fr, &fakeUserLookup{users: map[int64]*UserSnapshot{1: {ID: 1, Role: "patient", RealNameVerified: true}}})
	err := svc.CheckIn(context.Background(), 1, 2, 100, 0)
	assert.Error(t, err)
}

// TestCheckOut_OK 验证 in_service → completed。
func TestCheckOut_OK(t *testing.T) {
	fr := newFakeOrderRepo()
	escort := int64(2)
	fr.Create(context.Background(), &repo.Order{PatientID: 1, EscortID: &escort, Status: "in_service", OrderNo: "X"})
	svc := New(fr, &fakeUserLookup{users: map[int64]*UserSnapshot{1: {ID: 1, Role: "patient", RealNameVerified: true}}})
	require.NoError(t, svc.CheckOut(context.Background(), 1, 2, "service done"))
	o, _ := fr.FindByID(context.Background(), 1)
	assert.Equal(t, "completed", o.Status)
}
```

**Step 3: handler 扩展**

修改 `services/order/internal/handler/order.go`：

- `List` 函数增加 role 分支：

```go
// List GET /api/v1/orders?role=escort&status=matching
func (h *Handler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	role := c.Query("role")
	if role == "escort" {
		// 抢单池：列 matching 未接订单
		if middleware.Role(c) != "escort" {
			respondError(c, errs.New(errs.CodeForbidden, "only escort can list matching feed"))
			return
		}
		status := c.Query("status")
		if status == "" { status = "matching" }
		list, err := h.svc.ListForEscort(c.Request.Context(), status, 20, 0)
		if err != nil { respondError(c, err); return }
		httpx.OK(c, gin.H{"orders": list})
		return
	}
	// 老路径：patient 列自己的订单
	list, err := h.svc.List(c.Request.Context(), uid, 20, 0)
	if err != nil { respondError(c, err); return }
	httpx.OK(c, gin.H{"orders": list})
}
```

- 加 checkin / checkout handler：

```go
type checkinReq struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}

// CheckIn POST /api/v1/orders/:id/checkin
func (h *Handler) CheckIn(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c); if !ok { return }
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can check in"))
		return
	}
	var req checkinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.CheckIn(c.Request.Context(), id, uid, req.Lat, req.Lng); err != nil {
		respondError(c, err); return
	}
	httpx.OK[any](c, gin.H{"order_id": id, "state": "in_service"})
}

type checkoutReq struct {
	Note string `json:"note"`
}

// CheckOut POST /api/v1/orders/:id/checkout
func (h *Handler) CheckOut(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c); if !ok { return }
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can check out"))
		return
	}
	var req checkoutReq
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.CheckOut(c.Request.Context(), id, uid, req.Note); err != nil {
		respondError(c, err); return
	}
	httpx.OK[any](c, gin.H{"order_id": id, "state": "completed"})
}
```

- `RegisterRoutes` 末尾追加：

```go
orders.POST("/:id/checkin", h.CheckIn)
orders.POST("/:id/checkout", h.CheckOut)
```

**Step 4: handler 测试**

修改 `services/order/internal/handler/order_test.go` 末尾追加：

```go
// fakeOrderRepo 加 ListForEscort 已在 service test 加；handler test 复用既有 fakeRepo 也加。
func (r *fakeRepo) ListForEscort(ctx context.Context, status string, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.Status == status && o.EscortID == nil {
			out = append(out, o)
		}
	}
	return out, nil
}

// TestList_EscortRole 验证 role=escort 返回抢单池。
func TestList_EscortRole(t *testing.T) {
	r, fr, _ := newTestServer()
	fr.Create(context.Background(), &repo.Order{PatientID: 1, Status: "matching", OrderNo: "M"})
	tok := signTestToken(t, 2, "escort")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders?role=escort&status=matching", tok, nil)
	assert.Equal(t, 0, resp.Code)
}

// TestList_EscortRole_PatientForbidden 验证 patient 不能用 role=escort。
func TestList_EscortRole_PatientForbidden(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders?role=escort&status=matching", tok, nil)
	assert.NotEqual(t, 0, resp.Code)
}

// TestCheckIn_OK 验证签到。
func TestCheckIn_OK(t *testing.T) {
	r, fr, _ := newTestServer()
	escort := int64(2)
	fr.Create(context.Background(), &repo.Order{PatientID: 1, EscortID: &escort, Status: "accepted", OrderNo: "X"})
	tok := signTestToken(t, 2, "escort")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/1/checkin", tok, map[string]any{
		"lat": 39.9, "lng": 116.4,
	})
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestCheckIn_NotAssigned 验证非该订单的 escort 不能签到。
func TestCheckIn_NotAssigned(t *testing.T) {
	r, fr, _ := newTestServer()
	escort := int64(2)
	fr.Create(context.Background(), &repo.Order{PatientID: 1, EscortID: &escort, Status: "accepted", OrderNo: "X"})
	tok := signTestToken(t, 99, "escort") // 不同 user_id
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/1/checkin", tok, map[string]any{
		"lat": 39.9, "lng": 116.4,
	})
	assert.NotEqual(t, 0, resp.Code)
}

// TestCheckOut_OK 验证打卡。
func TestCheckOut_OK(t *testing.T) {
	r, fr, _ := newTestServer()
	escort := int64(2)
	fr.Create(context.Background(), &repo.Order{PatientID: 1, EscortID: &escort, Status: "in_service", OrderNo: "X"})
	tok := signTestToken(t, 2, "escort")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/1/checkout", tok, map[string]any{
		"note": "service done",
	})
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestCheckIn_BadLat 验证参数错误。
func TestCheckIn_BadLat(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 2, "escort")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/1/checkin", tok, map[string]any{
		"lat": 200, "lng": 0,
	})
	assert.NotEqual(t, 0, resp.Code)
}
```

**Step 5: 跑测试确认通过**

Run:
```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/order/...
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestOrderRepo_ListForEscort' ./services/order/internal/repo/
```
Expected: PASS（既有 service + handler + 集成 + 新加测试）

**Step 6: Commit**

```bash
git add services/order/internal/
git commit -m "feat(order): 抢单池 ListForEscort + checkin/checkout (accepted→in_service→completed) + 8 个测试"
```

---

### Task 8: main 装配 + smoke 脚本

**Files:**
- Modify: `services/escort/cmd/main.go`
- Create: `scripts/smoke-escort.sh`

**Step 1: 修改 escort cmd/main.go**

```go
// 在 import 块追加：
import (
	"github.com/growdu/doctors/services/escort/internal/events"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/repo"
	"github.com/growdu/doctors/services/escort/internal/service"
)

// 在 main 里替换 svc := service.New(nilRepo{}, nilPublisher{}) 为：
profileRepo := repo.NewProfileRepo(nil)
var pub events.Publisher
if len(cfg.Kafka.Brokers) > 0 {
	pub = events.NewKafkaPublisher(cfg.Kafka.Brokers)
} else {
	pub = &events.NopPublisher{}
}
defer func() { _ = pub.Close() }()
flow := service.NewProfileFlowService(profileRepo, pub)
svc := service.New(nilRepo{}, nilPublisher{})
svc.SetProfileReader(profileRepo)
bizH := handler.NewBusinessHandler(flow, svc)

// nilRepo / nilPublisher 既有；保留给老接口（match 兼容）。
```

> 既有 `handler.New(svc)` 调用应改为 `handler.NewWithBusiness(svc, bizH)` 或在原 Handler 内挂 BusinessHandler；按实际既有 `handler.New` 签名调整。

**Step 2: 写 smoke 脚本**

`scripts/smoke-escort.sh`：

```bash
#!/usr/bin/env bash
# escort-service 端到端 smoke：
#   1. build
#   2. 启动（后台）
#   3. /healthz 通
#   4. 11 个 P0 endpoint 401 拦截（无 token）
#
# 前置：go build 通过；业务调用需 main 装配 PG / Kafka；smoke 只测启动 + 路由 + 鉴权。

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ADDR=":8086"  # escort-service 默认端口
BIN="$ROOT/bin/escort"
LOGFILE="$ROOT/.data/escort-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/4] building escort-service..."
GOFLAGS="-mod=mod" GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/escort/cmd

echo "[2/4] starting escort-service on $ADDR..."
DOCTORS_ESCORT_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/4] curl /healthz"
HEALTH=$(curl -fsS "http://127.0.0.1$ADDR/healthz")
echo "$HEALTH" | grep -q '"status":"ok"' || { echo "healthz unexpected: $HEALTH"; exit 1; }
echo "  -> $HEALTH"

echo "[4/4] 11 个 P0 endpoint 401 拦截"
for path in \
  "/api/v1/escorts/register" \
  "/api/v1/escorts/real-name/auth" \
  "/api/v1/escorts/health-cert/upload" \
  "/api/v1/escorts/training/complete" \
  "/api/v1/escorts/me/profile" \
  "/api/v1/escorts/me/status" \
  "/api/v1/escorts/me/training-courses" \
  "/api/v1/escorts/me/reviews" \
  "/api/v1/orders?role=escort&status=matching" \
  "/api/v1/orders/1/checkin" \
  "/api/v1/orders/1/checkout" \
; do
  RESP=$(curl -sS "http://127.0.0.1$ADDR$path")
  echo "$RESP" | grep -q '"code":11001' || { echo "  $path unexpected: $RESP"; exit 1; }
  echo "  $path -> 401 OK"
done

echo "smoke OK"
```

**Step 3: 跑 smoke**

```bash
chmod +x scripts/smoke-escort.sh
bash scripts/smoke-escort.sh
```
Expected: smoke OK

**Step 4: Commit**

```bash
git add services/escort/cmd/main.go scripts/smoke-escort.sh
git commit -m "feat(escort): main 装配 ProfileFlowService + events.Publisher + smoke 脚本（11 个 endpoint 401 拦截）"
```

---

### Task 9: 文档同步 + dev.md

**Files:**
- Modify: `docs/04-业务流程.md`
- Modify: `dev.md`

**Step 1: 04 加陪诊师档案状态机流程**

```markdown
### 陪诊师档案状态机（11 态）

1. 陪诊师注册 → `escort_profiles.state='registering'`
2. POST `/escorts/real-name/auth` → `pending_real_name` → `pending_health_cert`（mock 自动通过）
3. POST `/escorts/health-cert/upload`（base64 + SHA256）→ 自动审核 → `pending_training`
4. POST `/escorts/training/complete`（5 题 80% 通过）→ `pending_agreement`
5. POST `/escorts/agreement/sign` → `pending_audit`
6. admin POST `/admin/escorts/{id}/approve`（admin plan）→ `approved`
7. PUT `/escorts/me/status`（online=true）→ `online`；online=false → `offline`
8. 服务中：`accepted → in_service`（checkin）/ `in_service → completed`（checkout）

事件 `escort.state_changed` 发到 Kafka；admin audit + notification 消费。
```

**Step 2: dev.md 加 §10.13**

```markdown
### 10.13 陪诊业务实装（2026-09-24 escort-business plan）

解决 `l2-api-gap-design.md` §2.2 P0 escort 14 API 中的 11 个（钱包 + 提现走 wallet plan）。

**落地 commits（8 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(migrations) | 0006 escort_profiles + health_certs + training_records |
| feat(escort) | 11 态状态机 pure function |
| feat(escort) | repo ProfileRepo (CRUD + 乐观锁 + 8 集成测试) |
| feat(escort) | profile_flow 业务流（实名 mock + 健康证 SHA256 + 培训 80%）+ me/* |
| feat(contracts+escort) | EscortStateChangedEvent + KafkaPublisher |
| feat(escort) | handler 8 endpoint + 12 handler 单测 |
| feat(order) | 抢单池 ListForEscort + checkin/checkout |
| feat(escort) | main 装配 + smoke 脚本 |

**API 增量（11 个）**：
- POST /api/v1/escorts/register
- POST /api/v1/escorts/real-name/auth
- POST /api/v1/escorts/health-cert/upload
- POST /api/v1/escorts/training/complete
- GET  /api/v1/escorts/me/profile
- PUT  /api/v1/escorts/me/status
- GET  /api/v1/escorts/me/training-courses
- GET  /api/v1/escorts/me/reviews
- GET  /api/v1/orders?role=escort&status=matching（order-service 扩展）
- POST /api/v1/orders/:id/checkin（accepted → in_service）
- POST /api/v1/orders/:id/checkout（in_service → completed）

**状态机**：11 态（registering / pending_real_name / pending_health_cert / pending_training / pending_agreement / pending_audit / approved / rejected / online / in_service / offline）。

**未做**：
- 实名真实接入（v1 mock "已通过"；v2 接公安二要素）
- 健康证 OSS 上传（v1 收 base64 + 存 SHA256；v2 接 OSS）
- 培训题库管理后台（v1 hardcode 5 题；v2 admin plan 配置）
- admin 审核（approve/reject）走 admin plan 直接 `UPDATE state`
- checkin/checkout 落库字段（v1 仅状态推进；v2 加 `checkin_at` / `checkout_at` 列）
```

**Step 3: Commit**

```bash
git add docs/ dev.md
git commit -m "docs: 陪诊师档案状态机流程 + dev.md 10.13 escort-business plan 落地记录"
```

---

### Task 10: 全量回归 + push

```bash
# 清干净 PG
docker exec doctors-postgres psql -U doctors -d doctors -c \
  "DROP TABLE IF EXISTS training_records CASCADE; DROP TABLE IF EXISTS health_certs CASCADE; DROP TABLE IF EXISTS escort_profiles CASCADE; DROP TABLE IF EXISTS refunds CASCADE; DROP TABLE IF EXISTS refund_policies CASCADE; DROP TABLE IF EXISTS order_events CASCADE; DROP TABLE IF EXISTS orders CASCADE; DROP TABLE IF EXISTS users CASCADE;"

# 跑全部单测
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off go test -count=1 ./shared/... ./services/...

# 跑全部集成测试
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 ./migrations/... ./services/...

# 跑 smoke
bash scripts/smoke-order.sh
bash scripts/smoke-escort.sh

# push
git push origin main
```

Expected: 全部 PASS + smoke OK + pushed.

---

## Self-Review

- ✅ **Spec 覆盖**: `l2-api-gap-design.md` §2.2 P0 escort 14 API 中落 11 个（钱包 + 提现 4 个走 wallet plan；escort 触发 SOS 复用 sos plan）；§3.1 EscortSummary 与 escort ProFiles 表字段对齐
- ✅ **状态机完整**: 11 态 `registering → pending_real_name → pending_health_cert → pending_training → pending_agreement → pending_audit → approved/rejected → online/offline` + `online → in_service → online/offline`，含 rejected → pending_health_cert 重新提交路径
- ✅ **关键边界**:
  - 实名 mock：service.RealNameAuth 直接通过 + 落 `users.real_name_verified=true`
  - 健康证 v1：API 收 base64 → repo 算 SHA256 + filename + mime；不入 OSS
  - 培训 v1：hardcode 5 题；80% 通过（4/5）
  - GPS mock：service.CheckIn 只校验 lat/lng 范围，**不调 geolocator**；**不算距离**
- ✅ **类型一致**: `Profile` / `HealthCert` / `TrainingRecord` 与 SQL CHECK 对齐；`State` 字符串与 `escort_profiles.state` 对齐；`StateChangedEvent` 与 `shared/contracts.EscortStateChangedEvent` 字段一致
- ✅ **测试矩阵**: Task 1 集成（migrations）+ Task 2 单元（state）+ Task 3 集成（repo）+ Task 4 单元（service）+ Task 5 单元（publisher）+ Task 6 单元（handler）+ Task 7 集成 + 单元（order）+ Task 8 smoke + Task 10 全量回归
- ✅ **YAGNI**: v1 不接真实实名 / OSS / 题库管理；admin 审核留 admin plan；不做 checkin/checkout 落库字段（v1 状态推进足够；v2 加列）

## 关联 spec

- `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.2 + §3.1
- `docs/superpowers/specs/2026-09-24-escort-app-design.md` §3.1 + §3.2 + §5
- `docs/superpowers/plans/2026-09-24-state-machine.md`（状态机 CHECK）
- `docs/superpowers/plans/2026-09-24-sos.md`（SOS 复用）

## Execution Options

> Plan 已 commit 到 `docs/superpowers/plans/2026-09-24-escort-business.md`。
> 当前为 plan_all 模式 → 进入实施阶段需要用户决策。

**下一步选项**：

1. **立即执行**（subagent-driven 或 inline 执行）—— 我开始实施 Task 1~10
2. **暂停 + review** —— 你 review 此 plan 后告诉我调整
3. **继续产 plan** —— 接着出 8 个后端 plan + 3 个前端 plan（virtual-number / wallet / review / message / address-coupon / admin / hospital-package / escort-order-ext + patient-miniapp / admin-web）