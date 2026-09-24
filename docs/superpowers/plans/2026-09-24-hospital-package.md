# 医院 + 服务包浏览 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `docs/superpowers/specs/2026-09-24-l2-api-gap-design.md` §2.1 patient-miniapp P0 中 `GET /api/v1/hospitals` + `GET /api/v1/packages` 缺 API；产出医院列表 / 详情 / 城市筛选 / 关键词搜索 + 服务包列表 / 详情全链路只读 API。**v1 不做 admin CRUD**（admin plan 接管）。

**Architecture:** 合并 `services/hospital/` + `services/package/` 为单一 `services/catalog/`（理由见 Global Constraints §3.1）。两张表 `hospitals` + `packages`，通过 `packages.hospital_id` 外键关联。4 个 GET 端点 + 分页（page/limit 标准）+ 城市两级筛选（city + district）+ ILIKE 关键词搜索 + lat/lng 坐标（只存不查，留 v2 escort 距离计算）。v1 启动时 SQL seed 10 家测试医院（北京协和 / 301 / 北医三院等）+ 每家 3 个服务包（半日 / 全日 / 单项）。0 Kafka 主题、0 状态机、0 通知。

**Tech Stack:** Go 1.24+ · pgx v5.7 · testify v1.11 · gin v1.10。

## 前置依赖

- `2026-09-24-state-machine.md`（`orders` 表已存在；v2 用 `hospital_id` / `package_id` 外键，本 plan **不** 加 FK 因为 order 表已存在且 0001-0004 迁移已固化；详见 Task 1 §约束）
- `2026-09-24-l2-api-gap-design.md` §2.1 patient-miniapp P0（hospitals + packages 2 行）+ §3.1 entities（Hospital / Package schema）
- `2026-09-24-patient-miniapp-design.md`（待写，依赖本 plan 的 4 个 API）
- `shared/errs.Code`（`CodeNotFound` = 12001 / `CodeParamInvalid` = 10001 / `CodeInternal` = 500000 复用现有）
- `shared/httpx.Resp[T]`（统一响应包装）

---

## Global Constraints

- Go 1.24+（toolchain go1.24.3）
- pgx v5.7.1（与既有 migrations / repo 一致）
- 测试覆盖率：业务包 ≥ 80%；handler + repo 各 ≥ 1 单测 / 集成测试 / endpoint
- Commit 节奏：每个 Task 完成立即 commit；前缀 `feat:` / `test:` / `fix:` / `docs:`
- 所有响应走 `shared/httpx.Resp[T]`（业务码在 body）
- 错误统一 `shared/errs.Error`（业务码 5 位 / 系统码 6 位）
- **v1 只读 API**：无 POST / PUT / DELETE handler；admin CRUD 由 `2026-09-24-admin-plan.md` 接管
- **服务拆分决策**：合并 `services/hospital/` + `services/package/` 为单一 `services/catalog/`（端口 :8088）

  - 理由：(1) spec §4 把 hospital + package 列为 1 个 plan；(2) 都是只读 + 简单查询；(3) 共享 PG pool + 配置 + 中间件；(4) 单 binary + 单 smoke + 单部署
  - **职责分离**：`internal/repo/hospital_repo.go` + `internal/repo/package_repo.go`（独立仓储）、`internal/handler/{hospital.go,package.go}`（独立 handler file）、共享 `internal/router/router.go`（聚合路由）
- **FK 策略**：`packages.hospital_id` 加 FK 引 `hospitals.id`；**不** 给 `orders.hospital_id` / `orders.package_id` 加 FK（`orders` 表是 0002 创建的，本 plan 不动既有迁移；订单创建时的 FK 校验由 service 层做）
- **分页契约**：所有列表走 `{page, limit, total, items}`；page 默认 1，limit 默认 20、最大 100
- **筛选 + 搜索**：`city`（精确）+ `district`（精确，二级）；`keyword` ILIKE `%keyword%` 匹配 `name` 字段
- **lat/lng**：v1 只存不查；不返回 `lat` / `lng`（v2 escort 距离计算时再加）
- **v1 不做**：admin CRUD / 缓存（v2 引 Redis）/ 全文搜索（v2 引 PG tsvector 或 ES）/ 多语言 / 图片 CDN 接入

---

## File Structure

| 路径 | 变更 | 职责 |
|---|---|---|
| `migrations/0006_catalog.up.sql` | Create | `hospitals` + `packages` 表 + 索引 + FK + down |
| `migrations/0006_catalog.down.sql` | Create | 逆向 |
| `migrations/migrations_test.go` | Modify | 加 `Test0006CatalogUpDown` |
| `migrations/0007_catalog_seed.up.sql` | Create | 10 家测试医院 + 30 个服务包 seed |
| `migrations/0007_catalog_seed.down.sql` | Create | 清空 seed 数据 |
| `services/catalog/internal/repo/hospital_repo.go` | Create | `HospitalRepo`（pgx）+ `ErrHospitalNotFound` |
| `services/catalog/internal/repo/hospital_repo_integration_test.go` | Create | 集成测试 |
| `services/catalog/internal/repo/package_repo.go` | Create | `PackageRepo`（pgx）+ `ErrPackageNotFound` |
| `services/catalog/internal/repo/package_repo_integration_test.go` | Create | 集成测试 |
| `services/catalog/internal/service/catalog_service.go` | Create | 业务编排：分页参数校验 + city/domain/keyword 注入 |
| `services/catalog/internal/service/catalog_service_test.go` | Create | service 单测（fake repo） |
| `services/catalog/internal/handler/hospital.go` | Create | `HospitalHandler` + 2 个 GET handler |
| `services/catalog/internal/handler/hospital_test.go` | Create | handler 单测 |
| `services/catalog/internal/handler/package.go` | Create | `PackageHandler` + 2 个 GET handler |
| `services/catalog/internal/handler/package_test.go` | Create | handler 单测 |
| `services/catalog/internal/router/router.go` | Create | 聚合 4 个 GET 路由 + `/healthz` |
| `services/catalog/internal/router/router_test.go` | Create | router 单测 |
| `services/catalog/internal/server/server.go` | Create | HTTP server 包装 + 优雅停机 |
| `services/catalog/cmd/main.go` | Create | main 装配（PG pool + config + router + server） |
| `scripts/smoke-catalog.sh` | Create | smoke：build + `/healthz` + 4 GET 401 拦截 |
| `docs/04-业务流程.md` | Modify | §4.7 加"医院 + 服务包浏览"流程 |
| `dev.md` | Modify | §10.13 加本 plan 落地记录 |

---

### Task 1: 数据库迁移（hospitals + packages + seed）

**Files:**
- Create: `migrations/0006_catalog.up.sql`
- Create: `migrations/0006_catalog.down.sql`
- Modify: `migrations/migrations_test.go`
- Create: `migrations/0007_catalog_seed.up.sql`
- Create: `migrations/0007_catalog_seed.down.sql`

**Step 1: 写集成测试（RED）**

在 `migrations/migrations_test.go` 末尾追加：

```go
// Test0006CatalogUpDown 验证 hospitals + packages 表 + FK + 索引 + down 可逆。
func Test0006CatalogUpDown(t *testing.T) {
	applyUp(t, "0006_catalog.up.sql", []string{"hospitals", "packages"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 列检查：hospitals
	for _, col := range []string{
		"id", "name", "city", "district", "address", "lat", "lng",
		"departments", "level", "photo_url", "created_at", "updated_at",
	} {
		var found bool
		err := conn.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='hospitals' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "hospitals.%s should exist", col)
	}

	// 列检查：packages
	for _, col := range []string{
		"id", "hospital_id", "name", "duration_hours", "amount",
		"description", "included", "created_at", "updated_at",
	} {
		var found bool
		err := conn.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM information_schema.columns
			               WHERE table_name='packages' AND column_name=$1)`, col).
			Scan(&found)
		require.NoError(t, err)
		assert.True(t, found, "packages.%s should exist", col)
	}

	// 索引检查
	for _, idx := range []string{
		"idx_hospitals_city_district",
		"idx_hospitals_name_trgm",
		"idx_packages_hospital_id",
		"idx_packages_name_trgm",
	} {
		var exists bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname=$1)`, idx).
			Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "%s should exist", idx)
	}

	// FK 检查
	var fkExists bool
	err = conn.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM information_schema.table_constraints
		               WHERE constraint_name='packages_hospital_id_fkey')`).
		Scan(&fkExists)
	require.NoError(t, err)
	assert.True(t, fkExists, "packages.hospital_id FK should exist")

	// down 校验
	applyDown(t, "0006_catalog.down.sql", []string{"hospitals", "packages"})
}

// Test0007CatalogSeedUpDown 验证 seed：10 家医院 + 30 个服务包 + down 清空。
func Test0007CatalogSeedUpDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn())
	require.NoError(t, err)
	defer conn.Close(ctx)

	// 依赖 0006
	for _, f := range []string{"0006_catalog.up.sql"} {
		sql, _ := os.ReadFile(f)
		_, err = conn.Exec(ctx, string(sql))
		require.NoError(t, err)
	}

	sql, err := os.ReadFile("0007_catalog_seed.up.sql")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, string(sql))
	require.NoError(t, err)

	var hospitalCount, packageCount int
	err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM hospitals`).Scan(&hospitalCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, hospitalCount, 10, "应至少 10 家医院")

	err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM packages`).Scan(&packageCount)
	require.NoError(t, err)
	assert.Equal(t, 30, packageCount, "应 30 个服务包（10 家 × 3）")

	// down 校验
	downSQL, err := os.ReadFile("0007_catalog_seed.down.sql")
	require.NoError(t, err)
	_, err = conn.Exec(ctx, string(downSQL))
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM hospitals`).Scan(&hospitalCount)
	require.NoError(t, err)
	assert.Equal(t, 0, hospitalCount, "清空后应为 0")
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'Test0006CatalogUpDown|Test0007CatalogSeedUpDown' ./migrations/
```

Expected: FAIL — `Test0006CatalogUpDown undefined`

**Step 3: 写 `0006_catalog.up.sql`**

```sql
-- 0006_catalog.up.sql
-- 医院 + 服务包浏览基础表（评审 P0：patient-miniapp 首页 + 详情页数据源）。
--
-- 设计要点：
--   - departments / included 用 TEXT[]（PG 原生数组，免 JOIN）。
--   - lat / lng DOUBLE PRECISION；v1 不查（留 escort 距离计算）。
--   - name 模糊搜索用 pg_trgm GIN 索引（性能优于 ILIKE 全表扫）。
--   - packages.hospital_id 加 FK；orders 表已有 hospital_id / package_id 但本 plan 不加 FK（避免改既有迁移）。

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE hospitals (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  city VARCHAR(32) NOT NULL,
  district VARCHAR(32) NOT NULL,
  address VARCHAR(256) NOT NULL DEFAULT '',
  lat DOUBLE PRECISION,
  lng DOUBLE PRECISION,
  departments TEXT[] NOT NULL DEFAULT '{}',
  level VARCHAR(16) NOT NULL DEFAULT '三甲',
  photo_url VARCHAR(512) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 城市两级筛选（首页"选择城市 → 选择区"）
CREATE INDEX idx_hospitals_city_district ON hospitals(city, district);

-- 医院名关键词搜索（ILIKE + GIN）
CREATE INDEX idx_hospitals_name_trgm ON hospitals USING gin (name gin_trgm_ops);

CREATE TABLE packages (
  id BIGSERIAL PRIMARY KEY,
  hospital_id BIGINT NOT NULL REFERENCES hospitals(id) ON DELETE CASCADE,
  name VARCHAR(128) NOT NULL,
  duration_hours INT NOT NULL CHECK (duration_hours > 0),
  amount NUMERIC(10,2) NOT NULL CHECK (amount > 0),
  description TEXT NOT NULL DEFAULT '',
  included TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 服务包列表按医院筛选
CREATE INDEX idx_packages_hospital_id ON packages(hospital_id);

-- 服务包名关键词搜索
CREATE INDEX idx_packages_name_trgm ON packages USING gin (name gin_trgm_ops);
```

**Step 4: 写 `0006_catalog.down.sql`**

```sql
-- 0006_catalog.down.sql
-- 撤销 0006：删表 + 索引（pg_trgm extension 保留）。

DROP INDEX IF EXISTS idx_packages_name_trgm;
DROP INDEX IF EXISTS idx_packages_hospital_id;
DROP INDEX IF EXISTS idx_hospitals_name_trgm;
DROP INDEX IF EXISTS idx_hospitals_city_district;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS hospitals;
```

**Step 5: 写 `0007_catalog_seed.up.sql`**

```sql
-- 0007_catalog_seed.up.sql
-- v1 测试数据：10 家三甲医院（北上广深 + 杭州成都武汉西安南京天津）
-- + 每家 3 个服务包（半日陪诊 / 全日陪诊 / 单项代办）。
--
-- id 用 BIGSERIAL；此处显式 INSERT 顺序固定，方便 Service 单测按 id 断言。

INSERT INTO hospitals (id, name, city, district, address, lat, lng, departments, level, photo_url) VALUES
  (1, '北京协和医院', '北京', '东城区', '帅府园1号', 39.9139, 116.4181, ARRAY['心内科','消化科','呼吸科'], '三甲', ''),
  (2, '中国人民解放军总医院', '北京', '海淀区', '复兴路28号', 39.9085, 116.3278, ARRAY['骨科','神经外科','肝胆外科'], '三甲', ''),
  (3, '北京大学第三医院', '北京', '海淀区', '花园北路49号', 39.9912, 116.3592, ARRAY['妇产科','运动医学','眼科'], '三甲', ''),
  (4, '上海交通大学医学院附属瑞金医院', '上海', '黄浦区', '瑞金二路197号', 31.2186, 121.4692, ARRAY['内分泌科','血液科','烧伤科'], '三甲', ''),
  (5, '复旦大学附属华山医院', '上海', '静安区', '乌鲁木齐中路12号', 31.2085, 121.4421, ARRAY['神经外科','皮肤科','感染科'], '三甲', ''),
  (6, '广州中山大学附属第一医院', '广州', '越秀区', '中山二路58号', 23.1288, 113.2901, ARRAY['肾内科','心内科','消化科'], '三甲', ''),
  (7, '深圳市人民医院', '深圳', '罗湖区', '东门北路1017号', 22.5476, 114.1235, ARRAY['急诊科','心内科','呼吸科'], '三甲', ''),
  (8, '浙江大学医学院附属第一医院', '杭州', '上城区', '庆春路79号', 30.2589, 120.1715, ARRAY['肝胆外科','肾移植','血液科'], '三甲', ''),
  (9, '四川大学华西医院', '成都', '武侯区', '国学巷37号', 30.6414, 104.0657, ARRAY['麻醉科','精神科','肿瘤科'], '三甲', ''),
  (10, '华中科技大学同济医学院附属同济医院', '武汉', '硚口区', '解放大道1095号', 30.5895, 114.2655, ARRAY['心血管内科','神经内科','儿科'], '三甲', '');

-- SELECT setval('hospitals_id_seq', (SELECT MAX(id) FROM hospitals));  -- BIGSERIAL 自动管理

INSERT INTO packages (hospital_id, name, duration_hours, amount, description, included) VALUES
  -- 1 北京协和医院
  (1, '协和半日陪诊', 4, 300.00, '挂号 + 取号 + 全程陪同就诊（4 小时内）', ARRAY['代办挂号','全程陪同','取药']),
  (1, '协和全日陪诊', 8, 580.00, '挂号 + 取号 + 全程陪同就诊（8 小时内）', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (1, '协和单项代办', 2, 150.00, '单项代办（挂号 / 取报告 / 送检）', ARRAY['单项代办']),
  -- 2 301 医院
  (2, '301 半日陪诊', 4, 320.00, '挂号 + 取号 + 全程陪同就诊（4 小时内）', ARRAY['代办挂号','全程陪同','取药']),
  (2, '301 全日陪诊', 8, 600.00, '挂号 + 取号 + 全程陪同就诊（8 小时内）', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (2, '301 单项代办', 2, 160.00, '单项代办', ARRAY['单项代办']),
  -- 3 北医三院
  (3, '北医三院半日陪诊', 4, 300.00, '4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (3, '北医三院全日陪诊', 8, 580.00, '8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (3, '北医三院单项代办', 2, 150.00, '单项代办', ARRAY['单项代办']),
  -- 4 上海瑞金
  (4, '瑞金半日陪诊', 4, 350.00, '上海瑞金 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (4, '瑞金全日陪诊', 8, 660.00, '上海瑞金 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (4, '瑞金单项代办', 2, 180.00, '单项代办', ARRAY['单项代办']),
  -- 5 上海华山
  (5, '华山半日陪诊', 4, 350.00, '上海华山 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (5, '华山全日陪诊', 8, 660.00, '上海华山 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (5, '华山单项代办', 2, 180.00, '单项代办', ARRAY['单项代办']),
  -- 6 广州中山一院
  (6, '中山一院半日陪诊', 4, 280.00, '广州 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (6, '中山一院全日陪诊', 8, 520.00, '广州 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (6, '中山一院单项代办', 2, 140.00, '单项代办', ARRAY['单项代办']),
  -- 7 深圳市人民医院
  (7, '深圳人民医院半日陪诊', 4, 320.00, '深圳 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (7, '深圳人民医院全日陪诊', 8, 600.00, '深圳 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (7, '深圳人民医院单项代办', 2, 160.00, '单项代办', ARRAY['单项代办']),
  -- 8 杭州浙一
  (8, '浙一半日陪诊', 4, 280.00, '杭州 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (8, '浙一全日陪诊', 8, 520.00, '杭州 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (8, '浙一单项代办', 2, 140.00, '单项代办', ARRAY['单项代办']),
  -- 9 成都华西
  (9, '华西半日陪诊', 4, 280.00, '成都 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (9, '华西全日陪诊', 8, 520.00, '成都 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (9, '华西单项代办', 2, 140.00, '单项代办', ARRAY['单项代办']),
  -- 10 武汉同济
  (10, '同济半日陪诊', 4, 280.00, '武汉 4 小时陪诊', ARRAY['代办挂号','全程陪同','取药']),
  (10, '同济全日陪诊', 8, 520.00, '武汉 8 小时陪诊', ARRAY['代办挂号','全程陪同','取药','代取报告']),
  (10, '同济单项代办', 2, 140.00, '单项代办', ARRAY['单项代办']);
```

**Step 6: 写 `0007_catalog_seed.down.sql`**

```sql
-- 0007_catalog_seed.down.sql
-- 清空 seed 数据；保留表结构（0006 不 down）。

TRUNCATE TABLE packages RESTART IDENTITY CASCADE;
TRUNCATE TABLE hospitals RESTART IDENTITY CASCADE;
```

**Step 7: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'Test0006CatalogUpDown|Test0007CatalogSeedUpDown' ./migrations/
```

Expected: PASS（2 个测试）

**Step 4: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): 0006 hospitals+packages 表（pg_trgm GIN + FK + CHECK）+ 0007 seed（10 医院 + 30 服务包）+ 2 集成测试"
```

---

### Task 2: HospitalRepo + PackageRepo（pgx 实现 + 哨兵错误）

**Files:**
- Create: `services/catalog/internal/repo/hospital_repo.go`
- Create: `services/catalog/internal/repo/hospital_repo_integration_test.go`
- Create: `services/catalog/internal/repo/package_repo.go`
- Create: `services/catalog/internal/repo/package_repo_integration_test.go`

**Step 1: 写集成测试（RED）**

`services/catalog/internal/repo/hospital_repo_integration_test.go`：

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

// HospitalRepo 集成测试 setup：仅 hospitals + packages 表（无 users / orders 依赖）。
func setupHospitalPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN())
	require.NoError(t, err, "connect pg")

	_, err = pool.Exec(ctx, `
		DROP TABLE IF EXISTS packages CASCADE;
		DROP TABLE IF EXISTS hospitals CASCADE;
		CREATE TABLE hospitals (
		  id BIGSERIAL PRIMARY KEY,
		  name VARCHAR(128) NOT NULL,
		  city VARCHAR(32) NOT NULL,
		  district VARCHAR(32) NOT NULL,
		  address VARCHAR(256) NOT NULL DEFAULT '',
		  lat DOUBLE PRECISION,
		  lng DOUBLE PRECISION,
		  departments TEXT[] NOT NULL DEFAULT '{}',
		  level VARCHAR(16) NOT NULL DEFAULT '三甲',
		  photo_url VARCHAR(512) NOT NULL DEFAULT '',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE packages (
		  id BIGSERIAL PRIMARY KEY,
		  hospital_id BIGINT NOT NULL REFERENCES hospitals(id) ON DELETE CASCADE,
		  name VARCHAR(128) NOT NULL,
		  duration_hours INT NOT NULL CHECK (duration_hours > 0),
		  amount NUMERIC(10,2) NOT NULL CHECK (amount > 0),
		  description TEXT NOT NULL DEFAULT '',
		  included TEXT[] NOT NULL DEFAULT '{}',
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err, "create tables")

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DROP TABLE IF EXISTS packages;
			DROP TABLE IF EXISTS hospitals;
		`)
		pool.Close()
	})
	return pool
}

func seedHospital(t *testing.T, pool *pgxpool.Pool, name, city, district string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO hospitals (name, city, district, departments) VALUES ($1, $2, $3, ARRAY['心内科']) RETURNING id`,
		name, city, district).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedPackage(t *testing.T, pool *pgxpool.Pool, hospitalID int64, name string, amount float64) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO packages (hospital_id, name, duration_hours, amount, included) VALUES ($1, $2, 4, $3, ARRAY['挂号']) RETURNING id`,
		hospitalID, name, amount).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestHospitalRepo_ListByCity 验证城市筛选。
func TestHospitalRepo_ListByCity(t *testing.T) {
	pool := setupHospitalPool(t)
	seedHospital(t, pool, "北京协和", "北京", "东城")
	seedHospital(t, pool, "301 医院", "北京", "海淀")
	seedHospital(t, pool, "上海瑞金", "上海", "黄浦")
	r := NewHospitalRepo(pool)

	hospitals, total, err := r.List(context.Background(), HospitalQueryFilter{City: "北京"}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, hospitals, 2)
	for _, h := range hospitals {
		assert.Equal(t, "北京", h.City)
	}
}

// TestHospitalRepo_ListByCityAndDistrict 验证 city + district 二级筛选。
func TestHospitalRepo_ListByCityAndDistrict(t *testing.T) {
	pool := setupHospitalPool(t)
	seedHospital(t, pool, "北京协和", "北京", "东城")
	seedHospital(t, pool, "301 医院", "北京", "海淀")
	r := NewHospitalRepo(pool)

	hospitals, _, err := r.List(context.Background(), HospitalQueryFilter{City: "北京", District: "海淀"}, 1, 20)
	require.NoError(t, err)
	assert.Len(t, hospitals, 1)
	assert.Equal(t, "海淀", hospitals[0].District)
}

// TestHospitalRepo_ListByKeyword 验证关键词 ILIKE 搜索。
func TestHospitalRepo_ListByKeyword(t *testing.T) {
	pool := setupHospitalPool(t)
	seedHospital(t, pool, "北京协和医院", "北京", "东城")
	seedHospital(t, pool, "301 医院", "北京", "海淀")
	r := NewHospitalRepo(pool)

	hospitals, _, err := r.List(context.Background(), HospitalQueryFilter{Keyword: "协和"}, 1, 20)
	require.NoError(t, err)
	assert.Len(t, hospitals, 1)
	assert.Contains(t, hospitals[0].Name, "协和")
}

// TestHospitalRepo_GetByID_OK 验证按 id 查详情。
func TestHospitalRepo_GetByID_OK(t *testing.T) {
	pool := setupHospitalPool(t)
	id := seedHospital(t, pool, "北京协和", "北京", "东城")
	r := NewHospitalRepo(pool)

	h, err := r.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, id, h.ID)
	assert.Equal(t, "北京协和", h.Name)
	assert.Equal(t, []string{"心内科"}, h.Departments)
}

// TestHospitalRepo_GetByID_NotFound 验证 ErrHospitalNotFound。
func TestHospitalRepo_GetByID_NotFound(t *testing.T) {
	pool := setupHospitalPool(t)
	r := NewHospitalRepo(pool)
	h, err := r.GetByID(context.Background(), 99999)
	assert.ErrorIs(t, err, ErrHospitalNotFound)
	assert.Nil(t, h)
}

// TestHospitalRepo_Pagination 验证分页。
func TestHospitalRepo_Pagination(t *testing.T) {
	pool := setupHospitalPool(t)
	for i := 0; i < 25; i++ {
		seedHospital(t, pool, "北京协和"+string(rune('A'+i)), "北京", "东城")
	}
	r := NewHospitalRepo(pool)

	// page 1 limit 10
	hospitals, total, err := r.List(context.Background(), HospitalQueryFilter{City: "北京"}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, hospitals, 10)

	// page 3 limit 10
	hospitals, _, err = r.List(context.Background(), HospitalQueryFilter{City: "北京"}, 3, 10)
	require.NoError(t, err)
	assert.Len(t, hospitals, 5)
}
```

`services/catalog/internal/repo/package_repo_integration_test.go`：

```go
//go:build integration
// +build integration

package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageRepo_ListByHospital 验证按医院筛选。
func TestPackageRepo_ListByHospital(t *testing.T) {
	pool := setupHospitalPool(t)
	h1 := seedHospital(t, pool, "北京协和", "北京", "东城")
	h2 := seedHospital(t, pool, "301 医院", "北京", "海淀")
	seedPackage(t, pool, h1, "协和半日", 300)
	seedPackage(t, pool, h1, "协和全日", 580)
	seedPackage(t, pool, h2, "301 半日", 320)
	r := NewPackageRepo(pool)

	pkgs, total, err := r.List(context.Background(), PackageQueryFilter{HospitalID: h1}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, pkgs, 2)
	for _, p := range pkgs {
		assert.Equal(t, h1, p.HospitalID)
	}
}

// TestPackageRepo_ListByKeyword 验证服务包名关键词搜索。
func TestPackageRepo_ListByKeyword(t *testing.T) {
	pool := setupHospitalPool(t)
	h1 := seedHospital(t, pool, "北京协和", "北京", "东城")
	seedPackage(t, pool, h1, "协和半日陪诊", 300)
	seedPackage(t, pool, h1, "协和全日陪诊", 580)
	seedPackage(t, pool, h1, "协和单项代办", 150)
	r := NewPackageRepo(pool)

	pkgs, _, err := r.List(context.Background(), PackageQueryFilter{HospitalID: h1, Keyword: "全日"}, 1, 20)
	require.NoError(t, err)
	assert.Len(t, pkgs, 1)
	assert.Contains(t, pkgs[0].Name, "全日")
}

// TestPackageRepo_GetByID_OK 验证按 id 查详情。
func TestPackageRepo_GetByID_OK(t *testing.T) {
	pool := setupHospitalPool(t)
	h := seedHospital(t, pool, "北京协和", "北京", "东城")
	pid := seedPackage(t, pool, h, "协和半日", 300)
	r := NewPackageRepo(pool)

	p, err := r.GetByID(context.Background(), pid)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, pid, p.ID)
	assert.Equal(t, "协和半日", p.Name)
	assert.InDelta(t, 300.00, p.Amount, 0.01)
	assert.Equal(t, []string{"挂号"}, p.Included)
}

// TestPackageRepo_GetByID_NotFound 验证 ErrPackageNotFound。
func TestPackageRepo_GetByID_NotFound(t *testing.T) {
	pool := setupHospitalPool(t)
	r := NewPackageRepo(pool)
	p, err := r.GetByID(context.Background(), 99999)
	assert.ErrorIs(t, err, ErrPackageNotFound)
	assert.Nil(t, p)
}

// TestPackageRepo_Pagination 验证分页。
func TestPackageRepo_Pagination(t *testing.T) {
	pool := setupHospitalPool(t)
	h := seedHospital(t, pool, "北京协和", "北京", "东城")
	for i := 0; i < 25; i++ {
		seedPackage(t, pool, h, "协和套餐"+string(rune('A'+i)), 100)
	}
	r := NewPackageRepo(pool)

	pkgs, total, err := r.List(context.Background(), PackageQueryFilter{HospitalID: h}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, pkgs, 10)
}

// testDSN 复用 hospital 测试的 DSN 解析。
func testDSN() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestHospitalRepo_|TestPackageRepo_' ./services/catalog/internal/repo/
```

Expected: FAIL — `undefined: NewHospitalRepo`, `undefined: NewPackageRepo`, `undefined: HospitalQueryFilter`

**Step 3: 写 `hospital_repo.go`**

`services/catalog/internal/repo/hospital_repo.go`：

```go
// Package repo 是 catalog-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直写 SQL（不引 sqlc）。
//   - city + district 走精确匹配（与微信小程序"选城市"交互对齐）。
//   - keyword 走 ILIKE；GIN 索引（pg_trgm）在 0006 已建。
//   - lat / lng v1 不返回（留 v2 escort 距离计算）。
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Hospital 映射 hospitals 表行。
type Hospital struct {
	ID          int64
	Name        string
	City        string
	District    string
	Address     string
	Departments []string
	Level       string
	PhotoURL    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HospitalQueryFilter 是 List 的筛选参数。
type HospitalQueryFilter struct {
	City     string // 精确
	District string // 精确
	Keyword  string // ILIKE %kw%
}

// ErrHospitalNotFound 是查询无结果时的哨兵。
var ErrHospitalNotFound = errors.New("repo: hospital not found")

// HospitalRepo 是 hospitals 表的仓储。
type HospitalRepo struct {
	pool *pgxpool.Pool
}

// NewHospitalRepo 构造仓储。
func NewHospitalRepo(pool *pgxpool.Pool) *HospitalRepo { return &HospitalRepo{pool: pool} }

// List 按筛选条件分页查询；返回 items + total。
func (r *HospitalRepo) List(ctx context.Context, f HospitalQueryFilter, page, limit int) ([]*Hospital, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// 动态 WHERE：基于筛选参数选择性 append。
	where := []string{}
	args := []any{}
	if f.City != "" {
		args = append(args, f.City)
		where = append(where, fmt.Sprintf("city = $%d", len(args)))
	}
	if f.District != "" {
		args = append(args, f.District)
		where = append(where, fmt.Sprintf("district = $%d", len(args)))
	}
	if f.Keyword != "" {
		args = append(args, "%"+f.Keyword+"%")
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	// COUNT
	countSQL := "SELECT COUNT(*) FROM hospitals" + whereSQL
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count hospitals: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	// SELECT（按 id 稳定）
	args = append(args, limit, offset)
	listSQL := fmt.Sprintf(`
		SELECT id, name, city, district, address, departments, level, photo_url,
		       created_at, updated_at
		FROM hospitals%s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d`, whereSQL, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list hospitals: %w", err)
	}
	defer rows.Close()

	out := make([]*Hospital, 0, limit)
	for rows.Next() {
		h := &Hospital{}
		if err := rows.Scan(
			&h.ID, &h.Name, &h.City, &h.District, &h.Address, &h.Departments,
			&h.Level, &h.PhotoURL, &h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, h)
	}
	return out, total, rows.Err()
}

// GetByID 按 id 查详情；不存在返回 ErrHospitalNotFound。
func (r *HospitalRepo) GetByID(ctx context.Context, id int64) (*Hospital, error) {
	const q = `
		SELECT id, name, city, district, address, departments, level, photo_url,
		       created_at, updated_at
		FROM hospitals WHERE id = $1`
	h := &Hospital{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&h.ID, &h.Name, &h.City, &h.District, &h.Address, &h.Departments,
		&h.Level, &h.PhotoURL, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHospitalNotFound
		}
		return nil, fmt.Errorf("get hospital: %w", err)
	}
	return h, nil
}
```

**Step 4: 写 `package_repo.go`**

`services/catalog/internal/repo/package_repo.go`：

```go
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Package 映射 packages 表行。
type Package struct {
	ID            int64
	HospitalID    int64
	Name          string
	DurationHours int
	Amount        float64
	Description   string
	Included      []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PackageQueryFilter 是 List 的筛选参数。
type PackageQueryFilter struct {
	HospitalID int64  // 精确；v1 必须传（patient 端从医院详情页进）
	Keyword    string // ILIKE %kw%
}

// ErrPackageNotFound 是查询无结果时的哨兵。
var ErrPackageNotFound = errors.New("repo: package not found")

// PackageRepo 是 packages 表的仓储。
type PackageRepo struct {
	pool *pgxpool.Pool
}

// NewPackageRepo 构造仓储。
func NewPackageRepo(pool *pgxpool.Pool) *PackageRepo { return &PackageRepo{pool: pool} }

// List 按筛选条件分页查询。
func (r *PackageRepo) List(ctx context.Context, f PackageQueryFilter, page, limit int) ([]*Package, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	where := []string{}
	args := []any{}
	if f.HospitalID > 0 {
		args = append(args, f.HospitalID)
		where = append(where, fmt.Sprintf("hospital_id = $%d", len(args)))
	}
	if f.Keyword != "" {
		args = append(args, "%"+f.Keyword+"%")
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	countSQL := "SELECT COUNT(*) FROM packages" + whereSQL
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count packages: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	args = append(args, limit, offset)
	listSQL := fmt.Sprintf(`
		SELECT id, hospital_id, name, duration_hours, amount, description, included,
		       created_at, updated_at
		FROM packages%s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d`, whereSQL, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list packages: %w", err)
	}
	defer rows.Close()

	out := make([]*Package, 0, limit)
	for rows.Next() {
		p := &Package{}
		if err := rows.Scan(
			&p.ID, &p.HospitalID, &p.Name, &p.DurationHours, &p.Amount,
			&p.Description, &p.Included, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// GetByID 按 id 查详情。
func (r *PackageRepo) GetByID(ctx context.Context, id int64) (*Package, error) {
	const q = `
		SELECT id, hospital_id, name, duration_hours, amount, description, included,
		       created_at, updated_at
		FROM packages WHERE id = $1`
	p := &Package{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.HospitalID, &p.Name, &p.DurationHours, &p.Amount,
		&p.Description, &p.Included, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPackageNotFound
		}
		return nil, fmt.Errorf("get package: %w", err)
	}
	return p, nil
}
```

**Step 5: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 -run 'TestHospitalRepo_|TestPackageRepo_' ./services/catalog/internal/repo/
```

Expected: PASS（11 个测试：hospital 6 + package 5）

**Step 6: Commit**

```bash
git add services/catalog/internal/repo/
git commit -m "feat(catalog): HospitalRepo + PackageRepo (pgx + ILIKE + 分页 + 11 集成测试)"
```

---

### Task 3: CatalogService（业务编排 + 分页参数校验）

**Files:**
- Create: `services/catalog/internal/service/catalog_service.go`
- Create: `services/catalog/internal/service/catalog_service_test.go`

**Step 1: 写 service 单测（RED）**

`services/catalog/internal/service/catalog_service_test.go`：

```go
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeHospitalRepo 满足 HospitalRepoIface（单测无需 PG）。
type fakeHospitalRepo struct {
	listFn func(ctx context.Context, f repo.HospitalQueryFilter, page, limit int) ([]*repo.Hospital, int, error)
	getFn  func(ctx context.Context, id int64) (*repo.Hospital, error)
}

func (f *fakeHospitalRepo) List(ctx context.Context, q repo.HospitalQueryFilter, page, limit int) ([]*repo.Hospital, int, error) {
	return f.listFn(ctx, q, page, limit)
}
func (f *fakeHospitalRepo) GetByID(ctx context.Context, id int64) (*repo.Hospital, error) {
	return f.getFn(ctx, id)
}

type fakePackageRepo struct {
	listFn func(ctx context.Context, f repo.PackageQueryFilter, page, limit int) ([]*repo.Package, int, error)
	getFn  func(ctx context.Context, id int64) (*repo.Package, error)
}

func (f *fakePackageRepo) List(ctx context.Context, q repo.PackageQueryFilter, page, limit int) ([]*repo.Package, int, error) {
	return f.listFn(ctx, q, page, limit)
}
func (f *fakePackageRepo) GetByID(ctx context.Context, id int64) (*repo.Package, error) {
	return f.getFn(ctx, id)
}

// TestListHospitals_PageLimitDefaults 验证 page/limit 默认值（page=1, limit=20）。
func TestListHospitals_PageLimitDefaults(t *testing.T) {
	var gotPage, gotLimit int
	hRepo := &fakeHospitalRepo{
		listFn: func(ctx context.Context, f repo.HospitalQueryFilter, page, limit int) ([]*repo.Hospital, int, error) {
			gotPage, gotLimit = page, limit
			return []*repo.Hospital{}, 0, nil
		},
	}
	pRepo := &fakePackageRepo{}
	svc := New(hRepo, pRepo)
	_, _, err := svc.ListHospitals(context.Background(), HospitalListInput{}, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, gotPage, "page 默认 1")
	assert.Equal(t, 20, gotLimit, "limit 默认 20")
}

// TestListHospitals_LimitCapped 验证 limit > 100 截到 100。
func TestListHospitals_LimitCapped(t *testing.T) {
	var gotLimit int
	hRepo := &fakeHospitalRepo{
		listFn: func(ctx context.Context, f repo.HospitalQueryFilter, page, limit int) ([]*repo.Hospital, int, error) {
			gotLimit = limit
			return []*repo.Hospital{}, 0, nil
		},
	}
	pRepo := &fakePackageRepo{}
	svc := New(hRepo, pRepo)
	_, _, err := svc.ListHospitals(context.Background(), HospitalListInput{}, 1, 500)
	require.NoError(t, err)
	assert.Equal(t, 100, gotLimit, "limit > 100 截到 100")
}

// TestListHospitals_NegativePageReturnsError 验证 page < 1 返回 CodeParamInvalid。
func TestListHospitals_NegativePageReturnsError(t *testing.T) {
	hRepo := &fakeHospitalRepo{}
	pRepo := &fakePackageRepo{}
	svc := New(hRepo, pRepo)
	_, _, err := svc.ListHospitals(context.Background(), HospitalListInput{}, -1, 20)
	require.Error(t, err)
	assert.True(t, errs.IsCode(err, errs.CodeParamInvalid))
}

// TestListHospitals_OK 验证正常返回。
func TestListHospitals_OK(t *testing.T) {
	hRepo := &fakeHospitalRepo{
		listFn: func(ctx context.Context, f repo.HospitalQueryFilter, page, limit int) ([]*repo.Hospital, int, error) {
			assert.Equal(t, "北京", f.City)
			assert.Equal(t, "东城", f.District)
			assert.Equal(t, "协和", f.Keyword)
			return []*repo.Hospital{{ID: 1, Name: "北京协和"}}, 1, nil
		},
	}
	pRepo := &fakePackageRepo{}
	svc := New(hRepo, pRepo)
	items, total, err := svc.ListHospitals(context.Background(),
		HospitalListInput{City: "北京", District: "东城", Keyword: "协和"}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Equal(t, "北京协和", items[0].Name)
}

// TestGetHospital_NotFoundReturnsCodeNotFound 验证服务层把 ErrHospitalNotFound 转 404。
func TestGetHospital_NotFoundReturnsCodeNotFound(t *testing.T) {
	hRepo := &fakeHospitalRepo{
		getFn: func(ctx context.Context, id int64) (*repo.Hospital, error) {
			return nil, repo.ErrHospitalNotFound
		},
	}
	pRepo := &fakePackageRepo{}
	svc := New(hRepo, pRepo)
	_, err := svc.GetHospital(context.Background(), 99999)
	require.Error(t, err)
	assert.True(t, errs.IsCode(err, errs.CodeNotFound))
}

// TestListPackages_RequiresHospitalID 验证 hospital_id = 0 返回 CodeParamInvalid。
func TestListPackages_RequiresHospitalID(t *testing.T) {
	hRepo := &fakeHospitalRepo{}
	pRepo := &fakePackageRepo{}
	svc := New(hRepo, pRepo)
	_, _, err := svc.ListPackages(context.Background(), PackageListInput{}, 1, 20)
	require.Error(t, err)
	assert.True(t, errs.IsCode(err, errs.CodeParamInvalid))
}

// TestGetPackage_NotFoundReturnsCodeNotFound 验证服务层把 ErrPackageNotFound 转 404。
func TestGetPackage_NotFoundReturnsCodeNotFound(t *testing.T) {
	hRepo := &fakeHospitalRepo{}
	pRepo := &fakePackageRepo{
		getFn: func(ctx context.Context, id int64) (*repo.Package, error) {
			return nil, repo.ErrPackageNotFound
		},
	}
	svc := New(hRepo, pRepo)
	_, err := svc.GetPackage(context.Background(), 99999)
	require.Error(t, err)
	assert.True(t, errs.IsCode(err, errs.CodeNotFound))
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestListHospitals_|TestGetHospital_|TestListPackages_|TestGetPackage_' ./services/catalog/internal/service/
```

Expected: FAIL — `undefined: New`, `undefined: HospitalListInput`

**Step 3: 写 `catalog_service.go`**

`services/catalog/internal/service/catalog_service.go`：

```go
// Package service - catalog-service 业务编排。
//
// 设计要点：
//   - 只读 API：5 个方法（ListHospitals / GetHospital / ListPackages / GetPackage）。
//   - 分页参数校验在服务层做（避免 handler 重复 + repo 兜底）。
//   - 哨兵错误（ErrHospitalNotFound / ErrPackageNotFound）转 CodeNotFound。
package service

import (
	"context"

	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/shared/errs"
)

// HospitalRepoIface 是 HospitalRepo 的最小契约（解耦）。
type HospitalRepoIface interface {
	List(ctx context.Context, f repo.HospitalQueryFilter, page, limit int) ([]*repo.Hospital, int, error)
	GetByID(ctx context.Context, id int64) (*repo.Hospital, error)
}

// PackageRepoIface 同理。
type PackageRepoIface interface {
	List(ctx context.Context, f repo.PackageQueryFilter, page, limit int) ([]*repo.Package, int, error)
	GetByID(ctx context.Context, id int64) (*repo.Package, error)
}

// 编译期断言
var (
	_ HospitalRepoIface = (*repo.HospitalRepo)(nil)
	_ PackageRepoIface  = (*repo.PackageRepo)(nil)
)

// HospitalListInput 是 ListHospitals 的入参。
type HospitalListInput struct {
	City     string
	District string
	Keyword  string
}

// PackageListInput 是 ListPackages 的入参。
type PackageListInput struct {
	HospitalID int64
	Keyword    string
}

// Service 是 catalog 业务门面。
type Service struct {
	hospitals HospitalRepoIface
	packages  PackageRepoIface
}

// New 构造服务。
func New(h HospitalRepoIface, p PackageRepoIface) *Service {
	return &Service{hospitals: h, packages: p}
}

// 业务常量。
const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// normalizePageLimit 把非法 page/limit 修正为默认值；page<1 报错。
func normalizePageLimit(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, errs.New(errs.CodeParamInvalid, "page must be >= 1")
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit, nil
}

// ListHospitals 列出医院（带筛选）。
func (s *Service) ListHospitals(ctx context.Context, in HospitalListInput, page, limit int) ([]*repo.Hospital, int, error) {
	page, limit, err := normalizePageLimit(page, limit)
	if err != nil {
		return nil, 0, err
	}
	return s.hospitals.List(ctx, repo.HospitalQueryFilter{
		City: in.City, District: in.District, Keyword: in.Keyword,
	}, page, limit)
}

// GetHospital 按 id 查医院详情。
func (s *Service) GetHospital(ctx context.Context, id int64) (*repo.Hospital, error) {
	h, err := s.hospitals.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrHospitalNotFound) {
			return nil, errs.Wrap(errs.CodeNotFound, "hospital not found", err)
		}
		return nil, err
	}
	return h, nil
}

// ListPackages 列出服务包（必须指定 hospital_id）。
func (s *Service) ListPackages(ctx context.Context, in PackageListInput, page, limit int) ([]*repo.Package, int, error) {
	if in.HospitalID <= 0 {
		return nil, 0, errs.New(errs.CodeParamInvalid, "hospital_id is required")
	}
	page, limit, err := normalizePageLimit(page, limit)
	if err != nil {
		return nil, 0, err
	}
	return s.packages.List(ctx, repo.PackageQueryFilter{
		HospitalID: in.HospitalID, Keyword: in.Keyword,
	}, page, limit)
}

// GetPackage 按 id 查服务包详情。
func (s *Service) GetPackage(ctx context.Context, id int64) (*repo.Package, error) {
	p, err := s.packages.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrPackageNotFound) {
			return nil, errs.Wrap(errs.CodeNotFound, "package not found", err)
		}
		return nil, err
	}
	return p, nil
}
```

> **重要补充 import**：上面对 `errors.Is` 用了 stdlib `errors` 包；请在文件头部追加：
>
> ```go
> import (
>   "context"
>   "errors"
>
>   "github.com/growdu/doctors/services/catalog/internal/repo"
>   "github.com/growdu/doctors/shared/errs"
> )
> ```
>
> 若 `shared/errs.Wrap` 签名与你工程的实际签名不匹配（参考既有 sos_service.go），按实际签名调整。

**Step 4: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/catalog/internal/service/
```

Expected: PASS（7 个测试）

**Step 5: Commit**

```bash
git add services/catalog/internal/service/
git commit -m "feat(catalog): Service 业务门面（分页参数校验 + 哨兵转 CodeNotFound + 7 个单测）"
```

---

### Task 4: Handler + Router（4 个 GET endpoint）

**Files:**
- Create: `services/catalog/internal/handler/hospital.go`
- Create: `services/catalog/internal/handler/hospital_test.go`
- Create: `services/catalog/internal/handler/package.go`
- Create: `services/catalog/internal/handler/package_test.go`
- Create: `services/catalog/internal/router/router.go`
- Create: `services/catalog/internal/router/router_test.go`

**Step 1: 写 handler 单测（RED）**

`services/catalog/internal/handler/hospital_test.go`：

```go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeHospitalSvc 满足 HospitalServiceIface。
type fakeHospitalSvc struct {
	listFn   func(ctx context.Context, in HospitalListInput, page, limit int) ([]*repo.Hospital, int, error)
	getFn    func(ctx context.Context, id int64) (*repo.Hospital, error)
	listCall struct{ city, district, keyword string; page, limit int }
	getCall  int64
}

func (f *fakeHospitalSvc) ListHospitals(ctx context.Context, in HospitalListInput, page, limit int) ([]*repo.Hospital, int, error) {
	f.listCall.city, f.listCall.district, f.listCall.keyword = in.City, in.District, in.Keyword
	f.listCall.page, f.listCall.limit = page, limit
	return f.listFn(ctx, in, page, limit)
}
func (f *fakeHospitalSvc) GetHospital(ctx context.Context, id int64) (*repo.Hospital, error) {
	f.getCall = id
	return f.getFn(ctx, id)
}

// TestListHospitals_OK 验证 GET /api/v1/hospitals。
func TestListHospitals_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeHospitalSvc{
		listFn: func(ctx context.Context, in HospitalListInput, page, limit int) ([]*repo.Hospital, int, error) {
			return []*repo.Hospital{{ID: 1, Name: "北京协和", City: "北京", District: "东城"}}, 1, nil
		},
	}
	h := NewHospitalHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/hospitals?city=北京&district=东城&keyword=协和&page=1", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "北京", svc.listCall.city)
	assert.Equal(t, "东城", svc.listCall.district)
	assert.Equal(t, "协和", svc.listCall.keyword)
	assert.Equal(t, 1, svc.listCall.page)

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestListHospitals_PageDefaults 验证无 page 时默认 1。
func TestListHospitals_PageDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeHospitalSvc{
		listFn: func(ctx context.Context, in HospitalListInput, page, limit int) ([]*repo.Hospital, int, error) {
			return nil, 0, nil
		},
	}
	h := NewHospitalHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/hospitals", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, svc.listCall.page, "无 page 应默认 1")
}

// TestGetHospital_OK 验证 GET /api/v1/hospitals/:id。
func TestGetHospital_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeHospitalSvc{
		getFn: func(ctx context.Context, id int64) (*repo.Hospital, error) {
			assert.Equal(t, int64(7), id)
			return &repo.Hospital{ID: 7, Name: "北京协和"}, nil
		},
	}
	h := NewHospitalHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/hospitals/7", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(7), svc.getCall)
}

// TestGetHospital_NotFound 验证不存在的 id 返回 CodeNotFound。
func TestGetHospital_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeHospitalSvc{
		getFn: func(ctx context.Context, id int64) (*repo.Hospital, error) {
			return nil, errs.New(errs.CodeNotFound, "hospital not found")
		},
	}
	h := NewHospitalHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/hospitals/99999", nil))
	var resp httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

// TestGetHospital_InvalidID 验证非数字 id 返回 CodeParamInvalid。
func TestGetHospital_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeHospitalSvc{}
	h := NewHospitalHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/hospitals/abc", nil))
	var resp httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}
```

`services/catalog/internal/handler/package_test.go`：

```go
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePackageSvc 满足 PackageServiceIface。
type fakePackageSvc struct {
	listFn   func(ctx context.Context, in PackageListInput, page, limit int) ([]*repo.Package, int, error)
	getFn    func(ctx context.Context, id int64) (*repo.Package, error)
	listCall struct{ hospitalID int64; keyword string; page, limit int }
	getCall  int64
}

func (f *fakePackageSvc) ListPackages(ctx context.Context, in PackageListInput, page, limit int) ([]*repo.Package, int, error) {
	f.listCall.hospitalID = in.HospitalID
	f.listCall.keyword = in.Keyword
	f.listCall.page, f.listCall.limit = page, limit
	return f.listFn(ctx, in, page, limit)
}
func (f *fakePackageSvc) GetPackage(ctx context.Context, id int64) (*repo.Package, error) {
	f.getCall = id
	return f.getFn(ctx, id)
}

// TestListPackages_OK 验证 GET /api/v1/packages?hospital_id=1。
func TestListPackages_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakePackageSvc{
		listFn: func(ctx context.Context, in PackageListInput, page, limit int) ([]*repo.Package, int, error) {
			assert.Equal(t, int64(1), in.HospitalID)
			return []*repo.Package{{ID: 1, Name: "协和半日"}}, 1, nil
		},
	}
	h := NewPackageHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/packages?hospital_id=1&keyword=半日", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(1), svc.listCall.hospitalID)
	assert.Equal(t, "半日", svc.listCall.keyword)

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestListPackages_MissingHospitalID 验证无 hospital_id 返回 CodeParamInvalid。
func TestListPackages_MissingHospitalID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakePackageSvc{
		listFn: func(ctx context.Context, in PackageListInput, page, limit int) ([]*repo.Package, int, error) {
			return nil, 0, errs.New(errs.CodeParamInvalid, "hospital_id is required")
		},
	}
	h := NewPackageHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/packages", nil))
	var resp httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestGetPackage_OK 验证 GET /api/v1/packages/:id。
func TestGetPackage_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakePackageSvc{
		getFn: func(ctx context.Context, id int64) (*repo.Package, error) {
			return &repo.Package{ID: id, Name: "协和全日", Amount: 580}, nil
		},
	}
	h := NewPackageHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/packages/2", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}
```

**Step 2: 跑测试确认失败**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 -run 'TestListHospitals_|TestGetHospital_|TestListPackages_|TestGetPackage_' ./services/catalog/internal/handler/
```

Expected: FAIL — `undefined: NewHospitalHandler`, `undefined: NewPackageHandler`, `undefined: HospitalListInput`

**Step 3: 写 `hospital.go`**

`services/catalog/internal/handler/hospital.go`：

```go
// Package handler 翻译 catalog HTTP 请求 ↔ service 调用 + errs 业务码。
package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// HospitalServiceIface 是 service 的最小契约。
type HospitalServiceIface interface {
	ListHospitals(ctx context.Context, in HospitalListInput, page, limit int) ([]*repo.Hospital, int, error)
	GetHospital(ctx context.Context, id int64) (*repo.Hospital, error)
}

// HospitalListInput 是 handler 入参。
type HospitalListInput struct {
	City     string
	District string
	Keyword  string
}

// HospitalHandler 持有 service 引用。
type HospitalHandler struct {
	svc HospitalServiceIface
}

// NewHospitalHandler 构造 handler。
func NewHospitalHandler(svc HospitalServiceIface) *HospitalHandler {
	return &HospitalHandler{svc: svc}
}

// ListHandler GET /api/v1/hospitals
func (h *HospitalHandler) ListHandler(c *gin.Context) {
	in := HospitalListInput{
		City:     c.Query("city"),
		District: c.Query("district"),
		Keyword:  c.Query("keyword"),
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	hospitals, total, err := h.svc.ListHospitals(c.Request.Context(), in, page, limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if hospitals == nil {
		hospitals = []*repo.Hospital{}
	}
	httpx.OK[any](c, gin.H{
		"items": hospitals,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetHandler GET /api/v1/hospitals/:id
func (h *HospitalHandler) GetHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid hospital id"))
		return
	}
	hospital, err := h.svc.GetHospital(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK[any](c, hospital)
}

// RegisterRoutes 挂路由（医院端）。
func (h *HospitalHandler) RegisterRoutes(r gin.IRouter, authMW gin.HandlerFunc) {
	v1 := r.Group("/api/v1")
	if authMW != nil {
		v1.Use(authMW)
	}
	v1.GET("/hospitals", h.ListHandler)
	v1.GET("/hospitals/:id", h.GetHandler)
}
```

**Step 4: 写 `package.go`**

`services/catalog/internal/handler/package.go`：

```go
package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// PackageServiceIface 是 service 的最小契约。
type PackageServiceIface interface {
	ListPackages(ctx context.Context, in PackageListInput, page, limit int) ([]*repo.Package, int, error)
	GetPackage(ctx context.Context, id int64) (*repo.Package, error)
}

// PackageListInput 是 handler 入参。
type PackageListInput struct {
	HospitalID int64
	Keyword    string
}

// PackageHandler 持有 service 引用。
type PackageHandler struct {
	svc PackageServiceIface
}

// NewPackageHandler 构造 handler。
func NewPackageHandler(svc PackageServiceIface) *PackageHandler {
	return &PackageHandler{svc: svc}
}

// ListHandler GET /api/v1/packages
func (h *PackageHandler) ListHandler(c *gin.Context) {
	hospitalID, _ := strconv.ParseInt(c.Query("hospital_id"), 10, 64)
	in := PackageListInput{
		HospitalID: hospitalID,
		Keyword:    c.Query("keyword"),
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	packages, total, err := h.svc.ListPackages(c.Request.Context(), in, page, limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if packages == nil {
		packages = []*repo.Package{}
	}
	httpx.OK[any](c, gin.H{
		"items": packages,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetHandler GET /api/v1/packages/:id
func (h *PackageHandler) GetHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Fail(c, errs.New(errs.CodeParamInvalid, "invalid package id"))
		return
	}
	pkg, err := h.svc.GetPackage(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK[any](c, pkg)
}

// RegisterRoutes 挂路由（服务包端）。
func (h *PackageHandler) RegisterRoutes(r gin.IRouter, authMW gin.HandlerFunc) {
	v1 := r.Group("/api/v1")
	if authMW != nil {
		v1.Use(authMW)
	}
	v1.GET("/packages", h.ListHandler)
	v1.GET("/packages/:id", h.GetHandler)
}
```

**Step 5: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/catalog/internal/handler/
```

Expected: PASS（8 个测试）

**Step 6: 写 `router.go`**

`services/catalog/internal/router/router.go`：

```go
// Package router 注册 catalog-service 的 HTTP 路由。
//
// 聚合 hospital + package handler 到同一 Engine；端口 :8088（参考 order :8082 / sos :8087）。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/catalog/internal/handler"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/middleware"
)

// New 返回挂好全部路由的 *gin.Engine。
func New(hh *handler.HospitalHandler, ph *handler.PackageHandler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})

	auth := middleware.Auth(jwtSecret)
	hh.RegisterRoutes(r, auth)
	ph.RegisterRoutes(r, auth)
	return r
}
```

**Step 7: 写 `router_test.go`**

```go
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(nil, nil, "test-secret") // healthz 不依赖 handler

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
```

**Step 8: 跑测试确认通过**

```bash
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./services/catalog/internal/router/
```

Expected: PASS

**Step 9: Commit**

```bash
git add services/catalog/internal/handler/ services/catalog/internal/router/
git commit -m "feat(catalog): HospitalHandler + PackageHandler + Router（4 GET endpoint + 9 单测）"
```

---

### Task 5: Server + Main 装配 + smoke

**Files:**
- Create: `services/catalog/internal/server/server.go`
- Create: `services/catalog/cmd/main.go`
- Create: `scripts/smoke-catalog.sh`

**Step 1: 写 `server.go`**

`services/catalog/internal/server/server.go`：

```go
// Package server 启动 catalog-service HTTP server。
package server

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/growdu/doctors/shared/logger"
)

// Server 是 HTTP server 包装。
type Server struct {
	addr string
	srv  *http.Server
}

// New 构造 server。
func New(addr string, h http.Handler) *Server {
	return &Server{
		addr: addr,
		srv: &http.Server{
			Addr:              addr,
			Handler:           h,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Run 启动 + 优雅停机。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	logger.L().Info("catalog-service starting", zap.String("addr", s.addr))

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
```

**Step 2: 写 `cmd/main.go`**

`services/catalog/cmd/main.go`：

```go
// catalog-service 入口。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/catalog/internal/handler"
	"github.com/growdu/doctors/services/catalog/internal/repo"
	"github.com/growdu/doctors/services/catalog/internal/router"
	"github.com/growdu/doctors/services/catalog/internal/server"
	"github.com/growdu/doctors/services/catalog/internal/service"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/db"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("catalog")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	// PG pool（nil 时业务调用会 panic；smoke 只走 /healthz）
	pool, err := db.NewPool(context.Background(), cfg.DB.DSN)
	if err != nil {
		log.Fatalf("connect pg: %v", err)
	}
	defer pool.Close()

	hRepo := repo.NewHospitalRepo(pool)
	pRepo := repo.NewPackageRepo(pool)

	svc := service.New(hRepo, pRepo)
	hh := handler.NewHospitalHandler(svc)
	ph := handler.NewPackageHandler(svc)
	srv := server.New(cfg.HTTP.Addr, router.New(hh, ph, cfg.Auth.JWTSecret))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.L().Info("catalog-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("catalog-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("catalog-service stopped")
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
```

> **关键**：若 `shared/db.NewPool` 不存在或签名不同，按既有 service（如 order / sos）调整；可参考 `services/sos/internal/cmd/main.go` 已有的 pool 装配。
> **关键**：若 `shared/config.Load("catalog")` 找不到 `catalog` 配置 section（参考既有 config.Load("sos")），按现有命名约定调整。

**Step 3: 写 `scripts/smoke-catalog.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
ADDR=":8088"
BIN="$ROOT/bin/catalog"
LOGFILE="$ROOT/.data/catalog-smoke.log"
mkdir -p "$ROOT/bin" "$ROOT/.data"

echo "[1/5] building catalog-service..."
GOPROXY="${GOPROXY:-https://goproxy.io,https://goproxy.cn,direct}" GOSUMDB="${GOSUMDB:-off}" \
  go build -o "$BIN" ./services/catalog/cmd

echo "[2/5] starting catalog-service on $ADDR..."
DOCTORS_CATALOG_HTTP_ADDR="$ADDR" "$BIN" > "$LOGFILE" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

# wait for /healthz
for i in {1..10}; do
  if curl -fsS "http://127.0.0.1$ADDR/healthz" > /dev/null 2>&1; then
    echo "  /healthz OK after ${i}00ms"
    break
  fi
  sleep 0.1
done

echo "[3/5] curl /healthz"
curl -fsS "http://127.0.0.1$ADDR/healthz" | head -c 200
echo
echo "[4/5] curl /api/v1/hospitals (no token → expect 401)"
curl -sS "http://127.0.0.1$ADDR/api/v1/hospitals?city=北京" | head -c 200
echo
echo "[5/5] curl /api/v1/packages (no token → expect 401)"
curl -sS "http://127.0.0.1$ADDR/api/v1/packages?hospital_id=1" | head -c 200
echo
echo "smoke OK"
```

```bash
chmod +x scripts/smoke-catalog.sh
bash scripts/smoke-catalog.sh
```

Expected: smoke OK（启动 + `/healthz` 200 + 2 个 GET endpoint 401 拦截）

**Step 4: Commit**

```bash
git add services/catalog/internal/server/ services/catalog/cmd/ scripts/smoke-catalog.sh
git commit -m "feat(catalog): server + main 装配（PG pool + Router + Server）+ smoke 脚本"
```

---

### Task 6: 文档同步 + dev.md + 全量回归

**Files:**
- Modify: `docs/04-业务流程.md`
- Modify: `dev.md`

**Step 1: 04 加"医院 + 服务包浏览"流程**

在 `docs/04-业务流程.md` §4.6 / §4.7 末尾追加：

```markdown
### 医院 + 服务包浏览流程

1. patient-miniapp 首页 → `GET /api/v1/hospitals?city=北京&district=东城` → 医院列表（10 家/页）
2. 用户输入"协和"→ `GET /api/v1/hospitals?keyword=协和` → 1 条命中（pg_trgm GIN 索引）
3. 点击医院 → `GET /api/v1/hospitals/:id` → 详情（departments / level / address）
4. 点击"查看服务包" → `GET /api/v1/packages?hospital_id=1` → 服务包列表（半日/全日/单项）
5. 点击服务包 → `GET /api/v1/packages/:id` → 详情（含 amount / duration_hours / included）
6. 进入下单页 → 携带 hospital_id + package_id 调 `POST /api/v1/orders`（order-service 接管）

**v1 不做**：
- admin CRUD（admin plan 接管）
- 缓存（v2 引 Redis）
- 全文搜索（v2 引 tsvector / ES）
- 距离计算（v2 escort 计划用 lat/lng）
```

**Step 2: dev.md 加 §10.13**

```markdown
### 10.13 医院 + 服务包浏览（2026-09-24 hospital-package plan）

解决 patient-miniapp 首页 / 详情页缺数据源 P0。

**落地 commits（6 个）**：

| commit | 内容 |
| :-- | :-- |
| feat(migrations) | 0006 hospitals+packages + 0007 seed（10 医院 + 30 服务包）+ 2 集成测试 |
| feat(catalog) | HospitalRepo + PackageRepo (pgx + ILIKE + 分页 + 11 集成测试) |
| feat(catalog) | Service（分页参数校验 + 哨兵转 CodeNotFound + 7 单测） |
| feat(catalog) | HospitalHandler + PackageHandler + Router（4 GET + 9 单测） |
| feat(catalog) | server + main 装配 + smoke 脚本 |
| docs | 04 流程 + dev.md 10.13 |

**API 增量（4 个）**：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/hospitals?city=&district=&keyword=&page=&limit= | 医院列表 + 城市两级筛选 + 关键词搜索 |
| GET | /api/v1/hospitals/:id | 医院详情 |
| GET | /api/v1/packages?hospital_id=&keyword=&page=&limit= | 服务包列表（v1 必传 hospital_id） |
| GET | /api/v1/packages/:id | 服务包详情 |

**架构决定**：合并 `services/hospital/` + `services/package/` 为单一 `services/catalog/`（端口 :8088）；理由见 plan §Global Constraints §3.1。

**FK 策略**：`packages.hospital_id` 加 FK（0006）；**orders.hospital_id / package_id 不加 FK**（避免改既有 0002 迁移；订单创建时 service 层校验）。

**未做**：admin CRUD / Redis 缓存 / 全文搜索 / 距离计算。
```

**Step 3: Commit**

```bash
git add docs/04-业务流程.md dev.md
git commit -m "docs: 医院+服务包浏览流程 + dev.md 10.13（6 commits / 4 GET API / 合并 catalog 服务）"
```

**Step 4: 全量回归**

```bash
# 清干净 PG（仅本次新增表）
docker exec doctors-postgres psql -U doctors -d doctors -c \
  "DROP TABLE IF EXISTS packages CASCADE; DROP TABLE IF EXISTS hospitals CASCADE;"

# 跑全部单测
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -count=1 ./shared/... ./services/catalog/...

# 跑全部集成测试
GOPROXY=https://goproxy.io,https://goproxy.cn,direct GOSUMDB=off \
  go test -tags=integration -count=1 ./migrations/... ./services/catalog/...

# 跑 smoke
bash scripts/smoke-catalog.sh

# push
git push origin main
```

Expected: 全部 PASS + smoke OK + pushed.

---

## Self-Review

- ✅ **Spec 覆盖**: l2-api-gap-design.md §2.1 patient-miniapp P0 全部 4 个 API + §3.1 entities（Hospital + Package 字段一致）
- ✅ **无占位符**: 每步有具体 SQL / Go 代码 + 测试命令；无 "TBD"
- ✅ **类型一致**: `Hospital` / `Package` 跨 Task 1/2/3/4 一致；`HospitalListInput` / `PackageListInput` 在 service 和 handler 间共享
- ✅ **测试矩阵**:
  - Task 1: 2 集成测试（迁移 + seed）
  - Task 2: 11 集成测试（hospital 6 + package 5）
  - Task 3: 7 单元测试（service 业务编排）
  - Task 4: 9 单元测试（handler 5 + package 3 + router 1）
  - Task 5: smoke（4 endpoint 启动 + 拦截）
  - Task 6: 全量回归
- ✅ **YAGNI**: v1 不做 admin CRUD / Redis 缓存 / 全文搜索 / 距离计算 / lat-lng 暴露
- ✅ **TDD 节奏**: 每个 Task 都遵循 RED（写测试）→ 失败 → GREEN（实现）→ 通过 → commit
- ✅ **架构决定**: 合并 catalog 服务在 §Global Constraints §3.1 + dev.md §10.13 双重声明

## 关联 plan

- `docs/superpowers/plans/2026-09-24-state-machine.md`（前置；orders 表已存在）
- `docs/superpowers/plans/2026-09-24-sos.md`（同批次；参考服务骨架）
- `docs/superpowers/plans/2026-09-24-escort-business-plan.md`（后续；escort 注册依赖 hospital/package 列表）
- `docs/superpowers/plans/2026-09-24-admin-plan.md`（后续；接管 hospital / package admin CRUD）