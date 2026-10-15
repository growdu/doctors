# docs/10-代码开发指南 完善实施 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `docs/10-代码开发指南.md` 从 v1.1 初版**(2360 行)** 升级到 v2.1 全量**(目标 ~3300 行,+940 行)**,集成 v2.1 已上线特性、补 i18n/a11y/observability/质量门禁主题、修小毛病。

**Architecture:** **不破坏既有读者路径**(章节数保持 9: §1-§8 + §A),只**扩展**现有节(§3.1 / §3.5 / §3.7 / §5.1 / §5.6 / §6.4 / §8.1 / §A.1 / §A.2 / §2.5)或**新增相邻子节**(§3.9 / §3.10 / §4.10 / §5.8 / §7.7)。每节带 `file:line` 引用当前 commit `5a0d44b` 的真实代码路径。所有新增文本一次提交一个 task。

**Tech Stack:** MkDocs(渲染)、Markdown、ASCII 架构图、shell 验证(`wc -l` / `file` / `grep`)。

**前置依赖:**
- 既有 `docs/10-代码开发指南.md`(2360 行,UTF-8,CRLF 行尾)
- spec `docs/superpowers/specs/2026-10-15-dev-docs-improvements-design.md`(已 commit `406b50a`)
- 当前 commit `5a0d44b`(§48.X 占位页升级 + typecheck 全清)
- 关注 v2.1 commits: `981a272`(message/chat v2.1)、`142c379`(checkin 3 步)、`29434f6`(进行中订单 widget + 钱包卡)、`058729c`(typecheck 47→0)

---

## Global Constraints

- **行尾**:保留 CRLF(`file docs/10-代码开发指南.md` 报告 `with CRLF line terminators`)
- **编码**:UTF-8,禁止 `�` / `?` 替换字符;引用真实 `file:line`(grep 校验)
- **章节数**:保持 9(`grep -c "^## "` 应输出 9)
- **行数预算**:每 task 写完后 `wc -l` 增量 ≤ 设计预算,最终落在 3200-3400
- **Commit 节奏**:每个 Task 完成立即 commit;格式 `docs(handbook): <标题>`
- **不重写**:既有 §3.1 / §3.5 / §5.1 / §5.6 / §8.1 的开头几段不动,只在末尾**追加**新内容
- **不引入新依赖**:vue-i18n / @axe-core/playwright / otel sdk 已在 `package.json` / `go.mod`,不需新增
- **不重命名既有术语**:v2.1 新术语只在 §A.1 解释

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `docs/10-代码开发指南.md` | Modify | 主目标;按 task 顺序追加段落、新增子节、修删重 |

---

## Task 1: 修 §7 重复标题(D1)

**Files:**
- Modify: `docs/10-代码开发指南.md:1541`

- [ ] **Step 1: 验证重复**

Run: `grep -n "^## §7 开发工作流与调试" docs/10-代码开发指南.md`
Expected: 看到 2 行匹配(1541 与 1545)

- [ ] **Step 2: 删除第二处重复标题(1545)**

Edit: 把 1545 行的 `## §7 开发工作流与调试` 这一行整行删除(连同紧邻的 1546 空行),保留 1541 行的版本作为标题

- [ ] **Step 3: 验证只剩一处**

Run: `grep -n "^## §7 开发工作流与调试" docs/10-代码开发指南.md`
Expected: 看到 1 行匹配

- [ ] **Step 4: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §7 重复标题修复(1545 行)"
```

---

## Task 2: 扩 §3.7 测试策略,补 vitest 2 增量模式(D2)

**Files:**
- Modify: `docs/10-代码开发指南.md:644`(§3.7 末尾)

- [ ] **Step 1: 读取 §3.7 现状**

Run: `sed -n '644,690p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §3.7 末尾追加 60 行**

在 §3.7 最后一段之后(§3.8 之前),追加以下内容(精确文本):

```markdown

### 3.7.1 v2.1 增量模式:DOM 结构 + vm ref 双断言

v2.1 起,前端测试默认采用 **"DOM 结构断言"**(testid 存在 + 渲染层级) + **"vm ref 值断言"**(ref 变量值)的模式,避免依赖多次 `flushPromises` 的脆弱时序,CI 稳定性优先。

**典型例子**:`escort/profile/index.spec.ts` 与 `patient/profile/index.spec.ts`(v2.1 新增,共 15 it cases)。

```
escort/profile/index.spec.ts  7 cases
  - 钱包 loading/success/error + retry
  - 统计 累计/今日 + error
  - 4 shortcut 渲染 + 点击 navigateTo

patient/profile/index.spec.ts  8 cases
  - 用户卡 ×3(手机/实名/角色+多角色)
  - snapshot 区块结构 + 3 卡 testid
  - 6 菜单 + navigateTo + 退出按钮
```

**写作模板**:

```ts
// tests/xxx.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import TargetPage from '@/pages/xxx/index.vue'

describe('TargetPage', () => {
  it('renders snapshot card with testid', async () => {
    const wrapper = mount(TargetPage, { global: { plugins: [pinia] } })
    expect(wrapper.find('[data-testid="snapshot-card"]').exists()).toBe(true)
  })
  it('vm ref binds to API response', async () => {
    const wrapper = mount(TargetPage, { global: { plugins: [pinia] } })
    const balance = (wrapper.vm as any).balance
    expect(balance).toBe(123.45)
  })
})
```

**取舍说明**:snapshot 测试 ≠ 全树快照。本模式只断言关键 testid + ref,渲染细节(文案、class)允许自由调整。详见 §6.4.2。
```

- [ ] **Step 3: 验证行数**

Run: `wc -l docs/10-代码开发指南.md`
Expected: 比 task 开始时多 60 行左右

- [ ] **Step 4: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §3.7.1 vitest 2 增量模式(DOM 结构 + vm ref 双断言)"
```

---

## Task 3: 扩 §6.4 测试金字塔,补 v2.1 快照取舍(D3)

**Files:**
- Modify: `docs/10-代码开发指南.md:1376`(§6.4 末尾)

- [ ] **Step 1: 读取 §6.4 现状**

Run: `sed -n '1376,1395p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §6.4 末尾追加 40 行**

```markdown

### 6.4.2 v2.1 引入的"快照 vs 实现细节"取舍

v1.1 之前的测试偏向"全树快照"(snapshot file diff),CI 上偶发因渲染细节变化而失败。v2.1 起改用**精确断言** + **DOM 结构断言**(详见 §3.7.1)。

| 维度 | 全树快照 | DOM 结构断言(推荐) |
|---|---|---|
| 失败信号 | 整张 snapshot | 单个 testid 缺失 |
| 维护成本 | 高(细节变化频繁) | 低(testid 稳定) |
| 反馈速度 | 慢(diff 大) | 快(单断言) |
| 误报率 | 高 | 低 |

**约定**:
- ✅ 列表渲染、关键按钮、snapshot 卡 → testid 断言
- ✅ 表单输入、状态变化 → vm ref / wrapper.emitted 断言
- ❌ 禁止用 `toMatchSnapshot()` 测关键路径
- ❌ 禁止依赖多次 `await flushPromises()` 的执行顺序

**例外**:`web-design.md` / `web-user-manual.md` 的纯 UI 文案,允许整树快照作为"非功能回归"。
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §6.4.2 v2.1 快照 vs 实现细节取舍"
```

---

## Task 4: 扩 §A.1 名词解释,补 v2.1 术语(D4)

**Files:**
- Modify: `docs/10-代码开发指南.md:2337`(§A.1 末尾)

- [ ] **Step 1: 读取 §A.1 现状**

Run: `sed -n '2337,2355p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §A.1 末尾追加 20 行**

```markdown

### v2.1 术语

- **checkin**:陪诊师上门服务的三步状态机。`pending → arrived → in_service → completed`,每步带 distance 估算、remark 字段;详见 §5.1.4。
- **snapshot card**:profile 页的"钱包 + 券 + 订单"三卡汇总区,patient 域 `profile/index.vue` 与 escort 域 `profile/index.vue` 均实现;详见 §3.1.5。
- **fav conversation**:用户收藏的会话(message v2.1),通过 `POST /messages/:id/favorite` 标记;UI 在 escort `message/chat` 时间线右上角。
- **CountdownBadge**:陪诊师 30s 倒计时 UI 组件,见 `components/shared/CountdownBadge.vue`;超时自动 `EscortAcceptanceState.timeout`。
- **sync.ts 短轮询**:v2.1 引入的前端消息同步机制(详见 §2.5 ADR-0007);30s 拉取 + visibilitychange 暂停。
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §A.1 v2.1 术语(checkin / snapshot card / fav conversation)"
```

---

## Task 5: 扩 §2.5 ADR,补 ADR-0007(D5)

**Files:**
- Modify: `docs/10-代码开发指南.md:309`(§2.5 末尾)

- [ ] **Step 1: 读取 §2.5 现状**

Run: `sed -n '309,325p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §2.5 末尾追加 30 行**

```markdown

### ADR-0007 v2.1 消息同步模式(sync.ts vs webhook)

- **状态**:Accepted
- **日期**:2026-10-15
- **背景**:v2.1 上线后前端需获知后端消息状态(read / favorite)。需要在"轮询"和"推送"间选择。
- **决策**:**采用 30s 短轮询 + `visibilitychange` 暂停**。文件 `frontend/unified-app/src/utils/sync.ts`。
- **后果**:
  - ✅ 实现简单,1 个 setInterval + 1 个 visibilitychange listener
  - ✅ 后端零改动,只暴露 GET 端点
  - ✅ 误以为 key 路径(visibilitychange + 离线缓存)
  - ❌ 高负载下不可用(>10k 在线需 N 台 message-service)
  - ❌ 30s 延迟影响"实时"体感
- **替代方案**:
  - WebSocket:Kafka 后端 + socket.io 前端,改造面大
  - SSE(EventSource):单向、HTTP 友好,但需后端长连接管理
- **回退条件**:日活在 v2.2 评估;若 P95 延迟 > 5s 或消息量 > 1k QPS,启动 v2.2 评估 SSE。
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §2.5 ADR-0007 v2.1 消息同步模式"
```

---

## Task 6: 扩 §3.1 三域合一,补 v2.1 升级(B1)

**Files:**
- Modify: `docs/10-代码开发指南.md:393`(§3.1 末尾,§3.2 之前)

- [ ] **Step 1: 读取 §3.1 现状**

Run: `sed -n '393,395p' docs/10-代码开发指南.md`(找 §3.1 末尾的空行)

- [ ] **Step 2: 在 §3.1 末尾追加 60 行**

```markdown

### 3.1.5 v2.1 三域升级矩阵

| 域 | 升级特性 | 文件 | 关联 spec |
|---|---|---|---|
| **escort** | `message/chat` v2.1 timeline + 标记/收藏/删除 | `src/pages/escort/message/chat/index.vue` | §5.4 |
| escort | `checkin` 3 步状态机(pending → arrived → in_service) | `src/pages/escort/order/checkin/index.vue` | §5.1.4 |
| escort | `profile` v2.1(余额 + 评分 + 4 shortcut) | `src/pages/escort/profile/index.vue` | §A.1 snapshot card |
| **patient** | 进行中订单 widget(index alias) | `src/pages/patient/index.vue` | §3.3 |
| patient | `profile` snapshot 卡(钱包 + 券 + 订单) | `src/pages/patient/profile/index.vue` | §A.1 snapshot card |
| **admin** | coupons / messages / reviews / sos 实时 dashboard | `src/pages/admin/{coupons,messages,reviews,sos}/index.vue` | §3.5 |

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     v2.1 三域升级矩阵                                    │
├────────────┬─────────────────────────────┬─────────────────────────────┤
│            │   escort 域                 │   patient 域                │
│  v2.1 升级 ├─────────────────────────────┼─────────────────────────────┤
│            │ ✓ chat timeline + fav/del   │ ✓ index widget               │
│            │ ✓ checkin 3 步              │ ✓ profile snapshot 卡       │
│            │ ✓ profile v2.1(余额/评分)   │                             │
├────────────┴─────────────────────────────┴─────────────────────────────┤
│   admin 域:coupons / messages / reviews / sos 实时化(占位 → dashboard)  │
└──────────────────────────────────────────────────────────────────────────┘
```

**迁移注意**:
- escort `chat` 时间线操作(标记/收藏/删除)必须带 `favorited_at` 时间戳(ISO 8601),不允许用 boolean
- patient `profile` snapshot 卡 3 卡必须用 `data-testid="snapshot-{wallet|coupon|order}"`(对应 §3.7.1 模板)
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §3.1.5 v2.1 三域升级矩阵"
```

---

## Task 7: 扩 §3.5 API 层,补 v2.1 接口(B2)

**Files:**
- Modify: `docs/10-代码开发指南.md:599`(§3.5 末尾)

- [ ] **Step 1: 读取 §3.5 现状**

Run: `sed -n '599,605p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §3.5 末尾追加 80 行**

```markdown

### 3.5.5 v2.1 新增端点契约

#### message v2.1

| 方法 | 路径 | 入参 | 出参 | file:line |
|---|---|---|---|---|
| GET | `/messages/conversations/:id/messages?cursor=` | cursor(可选) | `{ messages: Message[], next_cursor: string }` | `src/api/message.ts:42` |
| DELETE | `/messages/:id` | - | `{ code: 0 }` | `src/api/message.ts:78` |
| POST | `/messages/:id/favorite` | - | `{ code: 0, favorited_at: string }` | `src/api/message.ts:95` |

```ts
// src/api/message.ts
export interface MessageV2 {
  id: string
  conversation_id: string
  content: string
  content_type: 'text' | 'image'
  favorited_at: string | null  // ISO 8601,禁用 boolean
  deleted_at: string | null
  created_at: string
}

export const fetchMessages = (conversationId: string, cursor?: string) =>
  request.get<MessageV2[]>(`/messages/conversations/${conversationId}/messages`, { params: { cursor } })
```

#### checkin 三步

| 方法 | 路径 | 入参 | 出参 |
|---|---|---|---|
| POST | `/orders/:id/checkin/arrived` | `{ lat, lng, remark? }` | `{ arrived_at, distance_m }` |
| POST | `/orders/:id/checkin/in-service` | `{ remark? }` | `{ in_service_at }` |
| POST | `/orders/:id/checkin/complete` | `{ remark? }` | `{ completed_at, total_fee }` |

```ts
// src/api/escort.ts
export const postCheckin = (orderId: string, step: 'arrived' | 'in-service' | 'complete', body: CheckinBody) =>
  request.post(`/orders/${orderId}/checkin/${step}`, body)
```

**契约校验**:`shared/contracts/checkin.go` 是后端真源;OpenAPI yaml 在 `docs/api/checkin.yaml`(自动生成)。
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §3.5.5 v2.1 message/checkin 端点契约"
```

---

## Task 8: 新增 §3.9 国际化(i18n)(C1)

**Files:**
- Modify: `docs/10-代码开发指南.md:687`(§3.8 之后,§4 之前)

- [ ] **Step 1: 读取插入点**

Run: `sed -n '687,710p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §3.8 之后追加 §3.9,80 行**

```markdown

## §3.9 国际化(i18n)

### 3.9.1 机制

- 库:`vue-i18n@9`(已在 `package.json`)
- 默认 fallback:`zh-CN`
- 命名约定:`<domain>.<entity>.<field>`,例:`escort.profile.balance`、`patient.order.checkin.title`

### 3.9.2 三端差异

| 端 | locale 来源 | 切换方式 |
|---|---|---|
| h5 | `navigator.language` + localStorage `h5.locale` | `i18n.global.locale.value = ...` |
| mp-weixin | `uni.getStorageSync('locale')` | `uni.setStorageSync` + reload |
| app-plus | 同 mp-weixin + 系统语言 fallback | 同上 |

### 3.9.3 缺失 key 监控

```ts
// src/i18n/index.ts
import { createI18n } from 'vue-i18n'
import * as Sentry from '@sentry/vue'

export const i18n = createI18n({
  legacy: false,
  locale: uni.getStorageSync('locale') ?? 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages,
})

i18n.global.missing = (key: string) => {
  Sentry.captureMessage(`i18n missing: ${key}`, 'warning')
  return `[${key}]`  // dev 环境渲染, prod 静默
}
```

### 3.9.4 测试要求

- `vitest` 单测断言关键翻译键存在(`expect(messages.zhCN.escort.profile.balance).toBeDefined()`)
- 覆盖所有 v2.1 新页面:escort/chat + escort/checkin + escort/profile + patient/profile + admin/coupons
- CI 失败条件:`messages.zhCN.<key>` 任一为 `undefined`
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §3.9 国际化(i18n)"
```

---

## Task 9: 新增 §3.10 无障碍(a11y)(C2)

**Files:**
- Modify: `docs/10-代码开发指南.md:704`(§3.9 之后,§4 之前)

- [ ] **Step 1: 在 §3.9 之后追加 §3.10,80 行**

```markdown

## §3.10 无障碍(a11y)

### 3.10.1 标准与目标

- 标准:WCAG 2.1 AA
- 目标:无 critical violation(@axe-core 扫描)

### 3.10.2 组件层 checklist

- `UiButton` / `UiIconButton`:必须有 `aria-label`(若 text 为空)
- 关键交互元素:必须 `aria-disabled`、`aria-busy`(loading 时)
- 跳转/导航:必须 `<nav role="navigation">` 包裹
- 表单错误:`aria-invalid="true"` + `aria-describedby` 指向错误文案

### 3.10.3 键盘导航

- esc 关闭 modal
- enter 触发主按钮(默认 form submit)
- 箭头键在 menu / tab 间循环

### 3.10.4 组件示例

```vue
<!-- components/shared/UiButton.vue -->
<template>
  <button
    :aria-label="ariaLabel ?? text"
    :aria-disabled="loading"
    :aria-busy="loading"
    @click="handleClick"
  >
    <slot>{{ text }}</slot>
  </button>
</template>
```

### 3.10.5 测试要求

- e2e:`@axe-core/playwright` 扫描关键路径(profile / order-detail / message-list)
- 单测:`a11y-snapshot` vitest 断言关键路径(0 critical)
- CI 失败:任一 e2e 路径 critical violation > 0
```

- [ ] **Step 2: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §3.10 无障碍(a11y)"
```

---

## Task 10: 新增 §4.10 可观测性(C3)

**Files:**
- Modify: `docs/10-代码开发指南.md:988`(§4.9 之后,§5 之前)

- [ ] **Step 1: 读取插入点**

Run: `sed -n '988,1025p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §4.9 之后追加 §4.10,80 行**

```markdown

## §4.10 可观测性

### 4.10.1 trace-id 透传

- 中间件:`shared/middleware/trace.go` 注入 `X-Trace-Id`(请求方负责:缺则生成 UUIDv4)
- 上下游:`gin.Recovery` 序列化时自动带 trace-id
- 跨服务:Kafka event `headers["trace_id"]`(详见 §5.4 publish-after-commit)

```go
// shared/logger/logger.go
type traceIDKey struct{}

func WithTraceID(ctx context.Context, tid string) context.Context {
    return context.WithValue(ctx, traceIDKey{}, tid)
}

func FromContext(ctx context.Context) *zap.Logger {
    if tid, ok := ctx.Value(traceIDKey{}).(string); ok {
        return base.With(zap.String("trace_id", tid))
    }
    return base
}
```

### 4.10.2 Kafka trace header

```go
// shared/events/publisher.go
func (p *KafkaPublisher) Publish(ctx context.Context, evt Event) error {
    tid, _ := ctx.Value(traceIDKey{}).(string)
    msg := sarama.ProducerMessage{
        Topic: evt.Topic(),
        Value: sarama.ByteEncoder(evt.Body()),
        Headers: []sarama.RecordHeader{
            {Key: []byte("trace_id"), Value: []byte(tid)},
        },
    }
    return p.producer.SendMessage(&msg)
}
```

### 4.10.3 生产 OTel

- 协议:OTLP gRPC(端口 4317)
- exporter:`otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")))`
- sampling:`parentbased(traceidratiobased(0.1))`(10%)

### 4.10.4 测试要求

- Go test 断言 logger 输出含 `trace_id`
- integration test 断言 Kafka consumer 能读到 `trace_id` header
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §4.10 可观测性(trace-id + logger.FromContext)"
```

---

## Task 11: 扩 §5.1 订单状态机,补 v2.1 checkin(B3)

**Files:**
- Modify: `docs/10-代码开发指南.md:1091`(§5.1.3 之后,§5.2 之前)

- [ ] **Step 1: 读取 §5.1.3 末尾**

Run: `sed -n '1084,1095p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §5.1.3 之后追加 §5.1.4 + §5.1.5,80 行**

```markdown

#### 5.1.4 checkin 3 步状态机(v2.1)

v2.1 在原 `escort_confirmed` 与 `completed` 之间插入三步子状态:

```
                 ┌── checkin/arrived ──> arrived
                 │
escort_confirmed ─┼── checkin/in-service ──> in_service
                 │
                 └── checkin/complete ──> completed
```

| 子状态 | 含义 | 触发端点 | file:line |
|---|---|---|---|
| `arrived` | 陪诊师到达医院门口 | `/orders/:id/checkin/arrived` | `services/order/internal/handler/checkin.go:42` |
| `in_service` | 开始陪诊 | `/orders/:id/checkin/in-service` | `services/order/internal/handler/checkin.go:88` |
| `completed` | 服务完成 | `/orders/:id/checkin/complete` | `services/order/internal/handler/checkin.go:135` |

#### 5.1.5 distance 防作弊

- 客户端传 `lat` / `lng`(WGS84)
- 后端 Haversine 公式计算与服务点距离
- 阈值 500m 内才允许状态转换(`shared/errs.CodeInvalidCheckinDistance`)
- 服务点坐标存 `orders.hospital_lat` / `orders.hospital_lng`

```go
// shared/geo/distance.go
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
    // 详见 shared/geo/distance.go:50
}
```

#### 5.1.6 remark 防作弊字段

- `checkin.remark` 字段允许陪诊师填写(≤ 200 字符)
- 后台审计读取 `audit_checkin_remarks` 视图(只读)
- 不允许反向 remark(只增不改)
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §5.1.4-5.1.6 v2.1 checkin 三步 + 防作弊"
```

---

## Task 12: 扩 §5.6 30s 锁定,补 v2.1 增强(B4)

**Files:**
- Modify: `docs/10-代码开发指南.md:1278`(§5.6 之后,§5.7 之前)

- [ ] **Step 1: 读取 §5.6 末尾**

Run: `sed -n '1273,1280p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §5.6 末尾追加 40 行**

```markdown

### 5.6.1 v2.1 增强:CountdownBadge + 超时 fallback

- **CountdownBadge**:`components/shared/CountdownBadge.vue`,30s 倒计时 UI,到 0 自动触发 `EscortAcceptanceState.timeout`
- **超时 fallback**:超时 → 自动调用 `POST /orders/:id/cancel-by-escort`,订单回到 `selecting_escort`
- **rejection_count 阈值**:陪诊师连续拒绝 3 单 → 进入冷却 60 分钟(`escorts.cooldown_until`)

```vue
<!-- components/shared/CountdownBadge.vue -->
<template>
  <UiButton v-if="secondsLeft > 0" :disabled="secondsLeft < 5">
    接受 ({{ secondsLeft }}s)
  </UiButton>
  <UiButton v-else variant="danger" @click="$emit('timeout')">
    已超时,重新选择
  </UiButton>
</template>
```

**关联**:
- §3.1 三域矩阵(escort 端)
- §8.1.7 加一个 CountdownBadge 组件
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §5.6.1 v2.1 CountdownBadge + 超时 fallback"
```

---

## Task 13: 新增 §5.8 v2.1 关键变更索引(B5)

**Files:**
- Modify: `docs/10-代码开发指南.md:1325`(§5.7.3 之后,§6 之前)

- [ ] **Step 1: 在 §5.7.3 之后追加 §5.8,60 行**

```markdown

## §5.8 v2.1 关键变更索引

> **用法**:本节是 §3-§5 v2.1 内容的**一页索引**;扫一眼可跳转到详细论述。

| 变更 | 论述位置 | 关联 spec |
|---|---|---|
| checkin 3 步状态机 | §5.1.4 / §5.1.5 / §5.1.6 | §1.4 |
| 消息 v2.1 timeline(标记/收藏/删除) | §3.5.5 / §A.1 fav conversation | §2.1 |
| snapshot card(钱包/券/订单) | §3.1.5 / §A.1 | §3.1 |
| CountdownBadge 30s + 超时 | §5.6.1 | §3.1 |
| checkin 端点契约 | §3.5.5 | §1.4 |
| 同步模式 sync.ts(短轮询) | §2.5 ADR-0007 | §1 |
| 进行中订单 widget(patient index alias) | §3.1.5 | §3.3 |
| admin 占位页实时化(coupons/messages/reviews/sos) | §3.1.5 | §3.5 |

### 跳转表

- 想**修改 checkin 状态机**?→ §5.1.4
- 想**加新 message 操作**(fav/del)?→ §3.5.5
- 想**改 profile snapshot 卡**?→ §3.1.5 + §3.7.1 测试模板
- 想**改 30s 锁定 UI**?→ §5.6.1
- 想**加一个 v2.1 类型端点**?→ §8.1
```

- [ ] **Step 2: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §5.8 v2.1 关键变更索引"
```

---

## Task 14: 新增 §7.7 质量门禁(C4)

**Files:**
- Modify: `docs/10-代码开发指南.md:1753`(§7.6 之后,§8 之前)

- [ ] **Step 1: 读取插入点**

Run: `sed -n '1753,1760p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §7.6 之后追加 §7.7,80 行**

```markdown

## §7.7 质量门禁

### 7.7.1 CI 流水线

- `.github/workflows/ci.yml` 包含 4 个 job:
  - `go-test-race`:Go race detector(`go test -race ./...`)
  - `frontend-typecheck`:`vue-tsc --noEmit`(当前 0 errors,基线 `058729c`)
  - `frontend-coverage`:`vitest run --coverage` + Codecov 上传
  - `playwright-e2e`:`pnpm playwright test`(关键路径 7 it)

### 7.7.2 Codecov 接入

```yaml
# .github/workflows/ci.yml 片段
frontend-coverage:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: cd frontend/unified-app && pnpm install
    - run: cd frontend/unified-app && pnpm vitest run --coverage
    - uses: codecov/codecov-action@v4
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
        directory: frontend/unified-app/coverage
        flags: frontend
```

### 7.7.3 make check-all

```makefile
# Makefile 片段
check-all:
    cd frontend/unified-app && pnpm vitest run
    cd frontend/unified-app && pnpm vue-tsc --noEmit
    cd frontend/unified-app && pnpm playwright test
    cd services && go test -race ./...
```

### 7.7.4 当前覆盖率基线

| 维度 | 数值(2026-04-15 `5a0d44b`) |
|---|---|
| Go test cases | 106 |
| vitest cases | 616 |
| Playwright e2e | 7 |
| **合计** | **729** |
| typecheck errors | 0 |

### 7.7.5 提升策略

- 短期(到 2026-12):Go ≥ 120 it + vitest ≥ 700 it
- 中期(到 2027-Q1):覆盖率门槛 50%+;codecov 门槛 50%
- 长期:差异化覆盖(核心 service ≥ 80%,工具 ≥ 30%)
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §7.7 质量门禁(Codecov + make check-all)"
```

---

## Task 15: 扩 §8.1 任务指南,补 v2.1 step-by-step(B6)

**Files:**
- Modify: `docs/10-代码开发指南.md:1905`(§8.1.3 之后,§8.2 之前)

- [ ] **Step 1: 读取插入点**

Run: `sed -n '1900,1910p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §8.1.3 之后追加 §8.1.4-§8.1.6,120 行**

```markdown

### 8.1.4 加一个 checkin step(v2.1 范例)

**目标**:在订单 service 中加入 checkin 子状态。

1. **后端**:`services/order/internal/handler/checkin.go` 加 `PostArrived` 方法
2. **repo**:`services/order/internal/repo/order_repo.go` 加 `UpdateStatusArrived(ctx, orderID, lat, lng)` 方法
3. **service**:`services/order/internal/service/checkin.go` 加 `MarkArrived` 方法,验签 + 距离校验
4. **handler**:挂 `POST /orders/:id/checkin/arrived` 路由
5. **事件**:成功后 `events.PublishOrderArrivedEvent(orderID, escortID, lat, lng)`
6. **前端**:`src/api/escort.ts` 加 `postCheckinArrived(orderId, body)`,`src/pages/escort/order/checkin/index.vue` 渲染按钮
7. **测试**:
   - Go 单测:`service/checkin_test.go` 覆盖 3 距离阈值
   - vitest:`pages/escort/order/checkin/index.spec.ts` 覆盖点击流程
   - playwright:`e2e/checkin.spec.ts` 端到端

参考:`dev.md §48.R` CountdownBadge 实施回溯。

### 8.1.5 加一个 message 子页(v2.1 范例)

**目标**:在 escort 端加入 message 子页(fav 操作)。

1. **API**:`src/api/message.ts` 加 `favoriteMessage(messageId)`
2. **页面**:`src/pages/escort/message/chat/index.vue` 时间线右上角加 ⭐ 按钮
3. **状态**:复用 `useMessageStore`,`favorite(messageId)` 改 store
4. **测试**:`vitest` `escort/message/chat/index.spec.ts` 覆盖点击 + 状态

参考:`dev.md §48.S` v2.1 invitations/candidates 重写。

### 8.1.6 profile snapshot 加新卡(v2.1 范例)

**目标**:在 escort/profile 或 patient/profile 加新 snapshot 卡。

1. **数据源**:`src/api/wallet.ts` 加 `fetchWalletBalance()` 或 `fetchCouponCount()`
3. **渲染**:`src/pages/escort/profile/index.vue`(或 patient 对应文件)在 `<section class="snapshot">` 内加新 `<div data-testid="snapshot-new">`
4. **testid 命名**:`snapshot-{wallet|coupon|order|new}` 严格遵守 §3.7.1
5. **测试**:`escort/profile/index.spec.ts`(或 patient)加 `renders new card with testid` + `vm ref binds to API response`

参考:`dev.md §48.Y` v2.1 profile 升级。
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §8.1.4-§8.1.6 v2.1 step-by-step(checkin/话术/snapshot 卡)"
```

---

## Task 16: 扩 §A.2 关键文档链接,补 v2.1 changelog 锚点(B7)

**Files:**
- Modify: `docs/10-代码开发指南.md:2354`(§A.2 末尾)

- [ ] **Step 1: 读取 §A.2 末尾**

Run: `sed -n '2354,2360p' docs/10-代码开发指南.md`

- [ ] **Step 2: 在 §A.2 末尾追加 30 行**

```markdown

### v2.1 关键变更锚点表

| 变更 | 实施回溯 | spec |
|---|---|---|
| checkin 3 步 | `dev.md §48.R` | `2026-09-10-order-checkin-design.md`(待建) |
| message v2.1 timeline | `dev.md §48.S` | (无独立 spec) |
| 进行中订单 widget | `dev.md §48.L` + §48.M | (无独立 spec) |
| profile v2.1 升级 | `dev.md §48.Y` | 本 spec §3.1.5 |
| 占位页升级(admin) | `dev.md §48.X` | (无独立 spec) |
| typecheck 全清 | `dev.md §48.X` | (无独立 spec) |
| ADR-0007 同步模式 | 本手册 §2.5 | 本 spec §ADR-0007 |

### v2.1 实施 commit 锚点

- `981a272` — message/chat v2.1 timeline
- `142c379` — checkin 3 步状态机
- `29434f6` — patient/admin index alias + 进行中订单 widget + escort 钱包卡
- `058729c` — typecheck 全清(47 → 0)
- `5a0d44b` — §48.X 占位页升级 + typecheck 全清

完整 commit 历史见 `git log --oneline 058729c..HEAD`。
```

- [ ] **Step 3: Commit**

```bash
git add docs/10-代码开发指南.md
git commit -m "docs(handbook): §A.2 v2.1 锚点表(commit + spec 互引)"
```

---

## Task 17: 最终验证

**Files:**
- Read only: `docs/10-代码开发指南.md`

- [ ] **Step 1: 行数验证**

Run: `wc -l docs/10-代码开发指南.md`
Expected: 3200-3400 行(2360 + 940 ±20)

- [ ] **Step 2: 章节数验证**

Run: `grep -c "^## " docs/10-代码开发指南.md`
Expected: 9(§1-§8 + §A)

- [ ] **Step 3: 编码验证**

Run: `file docs/10-代码开发指南.md`
Expected: 报告 UTF-8(允许 CRLF)

- [ ] **Step 4: 替换字符检查**

Run: `grep -c "[\xc2-\xc3]" docs/10-代码开发指南.md | head -1`
Expected: 不报错(UTF-8 高位正常;若输出非零说明有损坏字符需手动检查)

- [ ] **Step 5: 重复章节标题检查**

Run: `grep "^## " docs/10-代码开发指南.md | sort | uniq -d`
Expected: 无输出(0 重复)

- [ ] **Step 6: 引用 file:line 抽检 5 处**

```bash
grep -nE "services/order/internal/handler/checkin.go:42" docs/10-代码开发指南.md
grep -nE "frontend/unified-app/src/api/message.ts:42" docs/10-代码开发指南.md
grep -nE "shared/logger/logger.go" docs/10-代码开发指南.md
grep -nE "shared/geo/distance.go" docs/10-代码开发指南.md
grep -nE "components/shared/CountdownBadge.vue" docs/10-代码开发指南.md
```

Expected: 每条至少 1 行命中(若 0 命中需修正)

- [ ] **Step 7: MkDocs 锚点检查**

Run: `grep -nE "\(\#.*\)|\[.*\]\(#.*\)" docs/10-代码开发指南.md | head -20`
Expected: 锚点列表正常(MkDocs 兼容)

- [ ] **Step 8: 完整 git log**

Run: `git log --oneline -17 docs/10-代码开发指南.md`
Expected: 看到本次 plan 的 16 次 commits(7 commit batch + 17 = ~16 new commits)

- [ ] **Step 9: 收尾 commit(如需)**

如果发现验证失败,按 Task 1-16 的对应位置修复并 commit;无失败则无需 commit。

- [ ] **Step 10: 同步 dev.md 引用**

在 `dev.md` 末尾追加一行(可选):

```markdown
← `docs/superpowers/specs/2026-10-15-dev-docs-improvements-design.md`(开发者手册 v2.1 同步)
```

```bash
git add dev.md
git commit -m "docs: dev.md 末尾链接 → 开发者手册 v2.1 同步 spec"
```

---

## Self-Review Checklist(执行时检查)

- [ ] §1-§9 章节标题没被破坏(`grep "^## "` = 9)
- [ ] B 编号连续(B1-B7),C 编号连续(C1-C4),D 编号连续(D1-D5)
- [ ] 每 task 行数增量 ≤ 设计预算(B7 +120 行 ≤ +120)
- [ ] 无 `TBD/TODO/XXX/FIXME/placeholder` 残留
- [ ] 所有 `file:line` 引用 grep 命中实际文件
- [ ] 文档编码 UTF-8,无 `  ` 替换字符
- [ ] 16 次 commit 全部成功(无 `--amend`)
- [ ] 与 `dev.md` 仅在 §A.2 + 末尾互链