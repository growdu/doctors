-- 0012_hospitals.up.sql
-- user-service 医院库：医院主数据，供 patient 浏览 + order-service 引用。
--
-- 设计要点：
--   - city_id：v1 用 city_id（指向 city 表的简化版——v1 内嵌；v2 接外部行政区划）；
--   - level：三甲 / 三乙 / 二甲 / 二乙 / 一级 / 其他（CHECK 约束）；
--   - status：active | inactive（v1 默认 active；inactive 不在列表展示）；
--   - lat / lng：v1 只存不查（v2 escort 距离计算用）；
--   - 索引：city_id + status（按城市筛选）；ILIKE 模糊查询通过 name 字段 + GIN 扩展（v1 跳过 GIN，仅普通 B-tree）；
--   - FK：orders.hospital_id 不加（订单表 0002 已固化，service 层校验）。

CREATE TABLE hospitals (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  city_id BIGINT NOT NULL,
  level VARCHAR(16) NOT NULL CHECK (level IN ('3a','3b','2a','2b','1','other')),
  status VARCHAR(16) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','inactive')),
  address VARCHAR(255) NOT NULL,
  lat DOUBLE PRECISION,
  lng DOUBLE PRECISION,
  phone VARCHAR(20),
  departments TEXT,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 城市筛选（按城市 + 状态）
CREATE INDEX idx_hospitals_city_status ON hospitals(city_id, status);

-- 名称搜索（B-tree 仅 prefix 匹配；ILIKE 全匹配 v1 在 service 层实现）
CREATE INDEX idx_hospitals_name ON hospitals(name);

-- CHECK：非空
ALTER TABLE hospitals ADD CONSTRAINT hospitals_name_nonempty
  CHECK (length(trim(name)) > 0);
ALTER TABLE hospitals ADD CONSTRAINT hospitals_address_nonempty
  CHECK (length(trim(address)) > 0);