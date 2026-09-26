---
## 32. 11 服务真实 Kafka producer/consumer 接入（v1.4 事件流闭环）

**目标**：把 11 个服务 main.go 中占位的 `nilPublisher` / `NopPublisher` 全部替换为真实 Kafka publisher——Brokers 缺失时降级为 NopPublisher + warn log（保留 v1.0 dev 模式）。

**12 个 commit**（1 auth 确认 + 9 publisher 接入 + 1 wallet consumer 改造 + 1 admin 重构 + 1 docs）：

| commit | 内容 | 关键改动 |
| :-- | :-- | :-- |
| `3b11093` | `feat(auth)` 确认 auth-service 不发 Kafka（纯 JWT） | 注释 + TestAuth_NoKafkaPublisher |
| `9789ae4` | `feat(order)` buildPublisher 加 2 单测（空 / 非空 Brokers） | TestBuildPublisher_EmptyBrokersReturnsNop + Kafka |
| `e781428` | `feat(match)` consumer 装配加 TestMatch_KafkaConsumerConfigGating | cfg 解析契约 + groupID 兜底 |
| `20ccc0e` | `feat(message)` 接入 kafkapublisher + buildPublisher + shutdown hook | 新建 kafkapublisher 包（TopicMessageSent）|
| `19a70ca` | `feat(payment)` 接入 kafkapublisher（completed/refunded）+ shutdown hook | 新建 kafkapublisher 包 |
| `ada49da` | `feat(review)` 接入 kafkapublisher（TopicOrderReviewed）+ shutdown hook | 新建 kafkapublisher 包 |
| `e22177d` | `feat(sos)` 接入 kafkapublisher（TopicSOSRaised）+ shutdown hook | 新建 kafkapublisher 包 |
| `b563f34` | `feat(user)` virtualnumber 接入 kafkapublisher + shutdown hook | 新建 kafkapublisher 包 + Service 加 Publisher 接口 |
| `f40dc42` | `feat(escort)` 接入 kafkapublisher + buildPublisher + shutdown hook | 新建 kafkapublisher 包（available/unavailable）|
| `739ae93` | `feat(wallet)` consumer 改用 shared/kafka.NewReader | 统一校验 + TestConsumeKafka |
| `b92996c` | `feat(admin)` buildPublisher 重构为函数 + 2 单测 | 函数式封装，Brokers 空 → NopPublisher + nil kafkaPub |
| `de031af` | `chore(docs)` dev.md §32 11 服务真实 Kafka 接入记录 | 7 子节 |

**关键设计（公共模式）**：

```go
func buildPublisher(cfg *config.Config) (service.Publisher, *events.KafkaPublisher, error) {
    if len(cfg.Kafka.Brokers) == 0 {
        logger.L().Warn("kafka brokers empty; falling back to NopPublisher")
        return &events.NopPublisher{}, nil, nil
    }
    kp := &events.KafkaPublisher{Writer: ...}
    return kp, kp, nil
}

// in main:
pub, kafkaPub, _ := buildPublisher(cfg)
svc := service.New(repo, pub, nilChannel)
if kafkaPub != nil {
    srv.RegisterShutdownHook("kafka-producer", func() error { return kafkaPub.Close() })
}
```

**7 个新增 kafkapublisher 包**（message / payment / review / sos / user-virtualnumber / escort / 共用 events）：
- 构造：`kafkapublisher.New(brokers, topic)` 返回 `Publisher` 接口
- Close：写 grace shutdown hook
- 测试：cfg 解析 + nil safety + topic 正确

**累计测试**：80 包 0 FAIL（基线 71 → +9 包：6 个 kafkapublisher + 3 个服务 main_test 新增）

**Plan 偏差**：

1. **order.events.KafkaPublisher 复用现有 events 包**：多 topic 走 SetTopic；不引入 shared/kafka.NewWriter
2. **admin.events.KafkaPublisher 同样复用现有 events 包**：Publish(ctx, topic, ev) 通用接口
3. **user 虚拟号 publisher 在 virtualnumber 子包内**：贴近 service 接口定义
4. **escort.AvailabilityEvent 字段名是 `Available` 而非 `Online`**：按真实字段路由
5. **escort kafkapublisher 简化构造**：不广播 UserID（v2 扩展）
6. **wallet consumer 改用 shared/kafka.NewReader**：统一校验
7. **payment kafkapublisher 借用 TopicPaymentCompleted 占位**：因 shared/kafka.NewWriter 强制 topic 不空，每次覆写 Topic
8. **auth 不引入 buildPublisher**：纯 JWT，注释 + TestAuth_NoKafkaPublisher 双重声明

**未做**：

- docker build / docker compose up（按要求跳过）
- 集成测试（`//go:build integration`）：需 docker-compose 起 Kafka；本轮单测覆盖 cfg 解析 + nil safety
- push（按要求跳过）
- 服务间端到端联调：仅在 dev.md §32.6 文档化验证流程

**端到端 v1.4 收官**：

```
docker compose -f docker-compose.deploy.yml up -d
  ↓
18 容器启动
  ├─ postgres 接收 shareddb.NewPool 的 11 服务连接
  ├─ kafka 接收 9 个 publisher（order/message/payment/review/sos/user-virtualnumber/escort/admin/wallet）
  └─ jaeger 接收 OTel exporter
  ↓
业务调用 → 真实 PG 读/写 → Kafka publisher 发布 event → 下游 service consumer 接收
  ↓
  ├─ order: OrderCreated → match 推邀请 → escort.confirm → order.accepted
  ├─ order: completed → wallet.OnOrderCompleted → frozen += amount
  ├─ payment: completed → order.escort_pending_acceptance → order.accepted
  ├─ order: refund → payment.OnRefund → wallet.DeductFrozenForRefund
  └─ T+7 触发 → wallet scanner 1 分钟扫到 → frozen -= amount; balance += amount
  ↓
SIGTERM → Server.Shutdown(15s) → LIFO hooks 释放
  ├─ otel-tracer flush
  ├─ db-pool close
  ├─ kafka-producer close
  └─ kafka-consumer reader close
  ↓
80 包 0 FAIL + Prometheus /metrics + Jaeger trace
```

### 累计交付（v1.4 收官）

| 维度 | 状态 |
| :-- | :--: |
| 后端 11 Go 服务 | ✅ 真实 PG + 真实 Kafka + 完整 HTTP + OTel + 采样 + 指标 + Recovery + RateLimit + 优雅停机 |
| 前端 3 端 | ✅ 完整骨架 + 业务页 |
| 部署 | ✅ 14 Dockerfile + docker-compose + Jaeger |
| CI | ✅ 4 job + Pages + 仓库维护 + 0.1 采样 |
| 可观测性 | ✅ OTel + Jaeger + Prometheus + zap 日志关联 |
| 稳定性 | ✅ Recovery + RateLimit + 优雅停机 (15s) + LIFO hooks |
| 测试 | ✅ 80 包 0 FAIL + 跨平台一键脚本 + golangci-lint v2 |
| 文档 | ✅ dev.md 32 章节 + README + REVIEW |
