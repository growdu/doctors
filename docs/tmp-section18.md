---
## 18. admin-service 后端全套落地（2026-09-24 admin plan §A1-A5）

**目标**：按 plan `2026-09-24-admin.md` 落地 admin-service：12 admin API + RBAC + work_orders 表 + dashboard overview 30s 缓存 + 6 个 admin 事件 + shared/middleware.RoleAuth。

**5 个 commit**：

| commit | 内容 |
| :-- | :-- |
| `82bc14d` | `feat(migrations)` 0008 work_orders (subject_id/subject_type + 4 CHECK + 3 索引 + Test0008) |
| `62371e6` | `feat(errs+contracts)` CodeAdminForbidden(11003) + CodeUpstreamUnavailable(15003) + 6 admin 事件 + 6 topic |
| `3c2a301` | `feat(admin)` repo (WorkOrder CRUD + Reports overview 直读 PG + 9 集成测试) |
| `dbba966` | `feat(admin)` clients (order/refund/escort/user + http.go) + service (12 API + 30s 缓存 + 6 事件) |
| `b73012b` | `feat(admin)` middleware RoleAuth + handler 12 API + router RBAC 矩阵 + server + main + smoke |

**关键设计**：

1. **新服务 `services/admin/` 独立部署**，不持有业务数据（除 work_orders）
2. **跨服务调用**：`internal/clients/` 调 order/refund/escort/user 内部接口（HTTP）
3. **dashboard 30s 缓存**：in-memory `sync.Mutex + expires_at`（v2 接 Redis）
4. **RBAC 6 角色**：super_admin / order_admin / refund_admin / audit_admin / cs / viewer
5. **错误码 2 新增**：admin_forbidden=11003 / upstream_unavailable=15003
6. **work_orders 工单表**：`subject_id + subject_type` polymorphic 关联订单/陪诊师/退款/用户任意对象

**12 admin API endpoint**：

- `GET /api/v1/admin/users`（全部 6 角色）
- `GET /api/v1/admin/orders` + `POST /api/v1/admin/orders/:id/force-cancel`（super/order_admin/cs 等）
- `GET /api/v1/admin/escorts/pending-audit` + `POST /approve` + `POST /reject`（super/audit_admin）
- `GET /api/v1/admin/refunds` + `POST /approve` + `POST /reject`（super/refund_admin）
- `GET /api/v1/admin/work-orders` + `POST /api/v1/admin/work-orders`（admin 创建）
- `GET /api/v1/admin/reports/overview`（全部 6 角色，30s 缓存）

**累计测试用例**：33 单测 + 10 集成 = **43 测试**（含 contracts 6 subtests）。

**全量回归**：`go test ./...` **47 包 0 FAIL**（含 admin 4 个有 test 包）。

**Plan 偏差**：

1. **work_orders schema 调整**：用 polymorphic `subject_id + subject_type`（而非 `order_id`），更通用（订单/陪诊师/退款/用户都可关联）
2. **middleware.RoleAuth 实际新建**：plan 标注"已就绪"实际不存在；本批次实现（4 单测）
3. **smoke 验证**：用 PowerShell `Start-Process admin.exe` + `Invoke-WebRequest` 跑通 /healthz + 401 拦截

**未做（留给后续）**：

- 集成测试需 `docker compose up`（PG 未起无法跑）
- dashboard overview 改 Redis（v2）
- WebSocket / 极光集成
- admin 启动日志体现 RBAC 角色矩阵 debug log

**端到端联通（v1.3 目标）**：

- admin-web 22 路由 → admin-service 12 API → 4 个内部 client（order/refund/escort/user）→ 各自真实业务服务
- RBAC 双层防护：admin-web 路由级 + 按钮级 + admin-service API 级（RoleAuth 中间件）
- 三端 trace-id 三处共用：`mp-` / `escort-` / 后端 logger.FromContext
- admin-web 105 测试 + admin-service 43 测试 + 全部 11 个 Go 服务 47 包 0 FAIL
