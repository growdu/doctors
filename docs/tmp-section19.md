---
## 19. user-service 5 模块落地（2026-09-24 address-coupon + hospital-package + virtual-number plan）

**目标**：按 3 份 plan 落地 user-service 5 个新模块——address（地址簿）、coupon（优惠券双表）、hospital（医院库）、package（服务包，按医院挂载）、virtual-number（虚拟号），覆盖 patient 端核心 CRUD。

**5 个 commit**：

| commit | 模块 | 关键能力 | endpoint |
| :-- | :-- | :-- | :-- |
| `ec50e6a` | `feat(address)` migration 0010 + 5 API | 5 地址/默认地址/partial unique | GET/POST/PUT/DELETE `/api/v1/addresses` + `PUT /:id/default` |
| `baeef7d` | `feat(coupon)` migration 0011 + 5 API | 平台发券 + 用户领取/核销双表 | GET/claim/use `/api/v1/coupons` + `/me/coupons` |
| `56a54fb` | `feat(hospital)` migration 0012 + 2 API | 城市/级别/状态过滤 | GET `/api/v1/hospitals` + `/:id` |
| `0694a01` | `feat(package)` migration 0013 + 2 API | FK → hospitals + 3 type | GET `/api/v1/hospitals/:id/packages` + `/api/v1/packages/:id` |
| `ccd33ad` | `feat(virtual-number)` migration 0014 + 2 events v1.3 | partial unique 同 order_id 单活 + 17+9 位号段 mock | POST `/allocate` + GET `/:id` |

**关键设计**：

1. **5 模块同放 user-service**（与 plan 拆 `services/catalog/` 不同）：共享 auth/JWT/配置；按 task brief 简化架构
2. **coupon 双表设计**（platform 模板 + user 实例）：`coupons` + `user_coupons` 通过外键关联；`partial unique (user_id, coupon_id)` 防一人多次领同券
3. **work_orders polymorphic 已存在**（admin-service）+ **virtual-number partial unique**：同 order_id 只能有 1 个 active 虚拟号（防号段泄漏）
4. **价格格式化**：handler 端 `price` 转 string 防 JS 浮点漂移（与 wallet 一致）
5. **虚拟号 mock 生成**：v1 用 17+9 位号段占位（避免与真实运营商冲突）
6. **Kafka 广播**：contracts 加 `VirtualNumberAllocatedEvent/ReleasedEvent` + topic；service 层 v1 未发布（v2 接 notification 时补）

**累计测试用例**：99 个 user-service 单测 + 22 个集成测试（`//go:build integration` 隔离） = **121 测试**（含 8 router + 9 service 既有）。

**全量回归**：`go test ./...` **48 包 0 FAIL**（含 user 5 个新包）。

**Plan 偏差**：

1. **地址字段简化**：plan 用 `province/city/district/detail`，按 task brief 用 `detail + lat/lng`（避免行政区划白名单争议）
2. **coupon 双表**：plan 1 张表，按 task brief 拆 `coupons` + `user_coupons`（平台模板 + 用户实例）
3. **hospital.service 同进程**：plan 单独 `services/catalog/`，按 task brief 放 user-service
4. **package 字段**：plan 用 `duration_hours/amount`，按 task brief 用 `duration_min/price`（细粒度更友好）
5. **virtual-number 号段**：plan 未指定，v1 用 17+9 位号段 mock
6. **migrations_test 0010~0014 集成测试未追加**：聚焦 module 单测；现有 framework 可直接加

**未做（留给后续）**：

- 接 pgxpool：main.go 仍用 nilRepo 占位（按 task 约束"不要 docker up"）
- migrations_test.go 加 Test0010~Test0014（参照 Test0008 风格）
- virtual-number service 层发布 Kafka 事件（contracts 已加 type）
- address/coupon admin 端 CRUD（admin-service 接管）
- 接 `migrations/0010~0014` 真实 PG 跑集成测试

**端到端联通（v1.3 目标）**：

- patient-miniapp 选陪诊师 → order-service → 后续 patient 选地址/优惠券 → admin-web 监控（address/coupon/escorts/orders 全链路 mock 数据已就绪）
- 后端 11 个 Go 服务 48 包 0 FAIL + user-service 121 测试覆盖
- contracts v1.3 加 2 个虚拟号事件
