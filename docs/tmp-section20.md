---
## 20. 4 服务补全 handler + router + main 接入（2026-09-24 message/sos/review/escort plan）

**目标**：补全 message / sos / review / escort 4 个服务缺失的 handler + router + server + cmd + main + config——让 11 个 Go 服务全部从「service 层骨架」升级为「可启动的真实 HTTP 服务」。

**4 个 commit**：

| commit | 服务 | endpoint | 新增单测 |
| :-- | :-- | :-- | :--: |
| `d59b1b7` | `feat(message-service)` | 4（send/list/detail/broadcast） | 34 |
| `3c695b7` | `feat(sos-service)` | 4（raise/list/detail/resolve） | 32 |
| `4d1797c` | `feat(review-service)` | 4（create/list/detail/reply） | 36 |
| `f12380a` | `feat(escort-service)` | 8 + 4 availability（qualifications/trainings/locations + 子包） | 98 |

**4 服务关键设计**：

1. **统一架构**：handler（按 plan 列 endpoint）+ router（gin Engine + shared/middleware.Auth）+ server（HTTP + 优雅停机）+ cmd/main（装配 entrypoint）+ config/<svc>.yaml（4 块 service/http/db/kafka）+ 单测（service + handler + router）
2. **service 扩展而非重写**：message / sos 在原有方法签名基础上新增 `GetByID` / `List` / `Broadcast`；review 改 endpoint 路径；escort 拆 qualifications + trainings 2 个子类型
3. **escort 路由改造**：原 `/escorts/:id/...` → `/escorts/me/...`（基于 JWT user_id 而非 escort ID）
4. **availability 子包接入**：已有 handler 不动，只在主 router 挂载

**累计测试用例**：**200 个新单测**（message 34 + sos 32 + review 36 + escort 98）

**全量回归**：`go test ./...` **59 个测试包 0 FAIL**（含 4 服务所有有 test 包）

**Plan 偏差**：

1. **escort endpoint 数量**：任务标题写"6+4=10"但清单列了 8 个新接口；按清单做了 8 + 保留 2 个兼容（注册/公开详情），共 14 个 handler 入口
2. **review 重写**：原 `/api/v1/reviews/orders/:orderID` 与 plan §Architecture 不符，按 plan 重写为 `/api/v1/reviews` + `/api/v1/reviews/:id` + `/api/v1/reviews/:id/reply`
3. **escort middleware 改造**：本地 `middleware.Auth` 改为接受 `(secret, userIDKey, roleKey)` 三参，与 `shared/middleware.Auth` 对齐
4. **service 层扩展而非重写**：保留原 service_test 全部，新增方法扩展 8-9 个

**未做（留给后续）**：

- pgxpool 真实数据库接入（main 用 nilRepo 占位，与 admin-service 风格一致）
- Kafka publisher 真实接入（占位 nil）
- 集成测试（按 //go:build integration 隔离）
- qualifications image_url 文件上传

**端到端联通（v1.3 目标）**：

- patient-miniapp 选陪诊师 → order-service → user-service（address/coupon/hospital/package/virtual-number）→ message / sos / review / escort 服务 → admin-web 22 路由 + admin-service 12 API 监控全流程
- 后端 11 个 Go 服务 59 包 0 FAIL + 200 个新单测
- 三端 trace-id 共用：mp-/escort-/后端 logger.FromContext
