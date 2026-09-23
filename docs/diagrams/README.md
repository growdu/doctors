# 架构图清单 · docs/diagrams

> 本目录是陪诊师平台需求文档（`docs/`）的可视化补充。  
> 所有图都是独立的 HTML 文件，内嵌 SVG + CSS，可直接在浏览器打开、截图或嵌入文档。

## 主题

医疗蓝绿主题（基于 docs/diagrams/style-guide.md）：  
- paper `#fafbfc` · ink `#1f3a4d`（深墨蓝）  
- accent `#1a8f7a`（医疗绿） · link `#2e5aa8`（连接蓝）  
- danger `#c0392b` · warning `#d97706`

## 技术栈

- **后端**：Go 1.22+ · Hertz / Gin · gRPC
- **数据库**：PostgreSQL 16 · Patroni · pgx + sqlc
- **缓存 / MQ**：Redis 7 · Kafka
- **部署**：K8s · distroless 镜像

## 图目录（10 张）

| # | 主题 | 类型 | 文件 | 用途 |
| :--: | :-- | :-- | :-- | :-- |
| 01 | 系统总体架构 | Architecture | [01-system-architecture.html](./01-system-architecture.html) | 三层分层 + 三方集成总览 |
| 02 | 多 AZ 部署 | Deployment | [02-deployment-architecture.html](./02-deployment-architecture.html) | 双 AZ + 异地灾备 |
| 03 | 安全与合规分层 | Layer Stack · Compensating | [03-security-layers.html](./03-security-layers.html) | 六层纵深防御 |
| 04 | 患者陪诊旅程 | User Journey | [04-patient-journey.html](./04-patient-journey.html) | 6 阶段 + 情绪曲线 + 痛点 |
| 05 | 订单匹配数据流 | Data Flow | [05-order-matching-flow.html](./05-order-matching-flow.html) | 支付→抢单→接单全链路 |
| 06 | 服务异常与降级 | Flowchart | [06-degradation-flow.html](./06-degradation-flow.html) | 4 大异常决策树 + SLA |
| 07 | 订单状态机 | State Machine | [07-order-state-machine.html](./07-order-state-machine.html) | 11 状态 + 14 转换 |
| 08 | 陪诊师抢单时序 | Sequence | [08-grab-order-sequence.html](./08-grab-order-sequence.html) | 5 角色 + 15 步交互 |
| 09 | 三端集成对比 | Architecture | [09-three-clients-integration.html](./09-three-clients-integration.html) | iOS / Android / 小程序差异化接入 |
| 10 | 服务依赖图 | Dependency Graph | [10-service-dependencies.html](./10-service-dependencies.html) | 11 服务依赖关系 |

## 阅读顺序建议

- **新人 5 分钟入门**：01 → 04 → 07
- **架构师**：01 → 02 → 05 → 08 → 10
- **安全 / 合规评审**：03 → 06
- **产品 / 业务**：04 → 06 → 07

## 与 docs/ 章节的对应

| 图 | 对应文档章节 |
| :-- | :-- |
| 01 系统架构 | `docs/06-系统架构.md` 6.2 总体架构图 |
| 02 部署架构 | `docs/06-系统架构.md` 6.5 部署架构 |
| 03 安全分层 | `docs/05-非功能需求.md` 5.3 安全 / 5.4 合规 |
| 04 患者旅程 | `docs/03-功能需求.md` 3.2 患者端 / `docs/04-业务流程.md` 4.3 |
| 05 订单匹配数据流 | `docs/04-业务流程.md` 4.5 匹配规则 + `docs/06-系统架构.md` 6.3 |
| 06 降级路径 | `docs/04-业务流程.md` 4.11 异常分支 / `docs/05-非功能需求.md` 5.10 应急 |
| 07 订单状态机 | `docs/04-业务流程.md` 4.4 / `docs/07-数据模型.md` 7.3 |
| 08 抢单时序 | `docs/04-业务流程.md` 4.3 端到端 / 4.6 支付 |
| 09 三端集成 | `docs/01-项目概述.md` 1.3 范围 / `docs/03-功能需求.md` 3.5 三端差异 |
| 10 服务依赖 | `docs/06-系统架构.md` 6.3 服务拆分 |

## 引用方式

```markdown
![系统总体架构](./diagrams/01-system-architecture.html)
```

或将 HTML 直接嵌入到 confluence / 飞书文档时使用 iframe。

## 维护约定

- 每张图命名 `NN-主题-类型.html`，NN 为两位序号
- 改图后同步更新本 README 的"对应文档章节"
- 新增图：复制 `assets/template.html`，按 diagram-design skill 流程做