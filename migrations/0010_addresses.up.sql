-- 0010_addresses.up.sql
-- user-service 地址簿：每个用户最多 5 个地址，1 个默认地址。
--
-- 设计要点：
--   - 地址字段极简（detail + lat/lng）：v1 不做行政区划白名单（v2 加）；
--     业务校验放在 handler/service 层（detail ≤ 200 字、phone 校验）。
--   - 默认地址用 partial unique index 保证唯一；
--   - 应用层走事务切换默认（先清同用户其他默认 → 再 UPDATE）。
--   - 删除地址：v1 hard delete（无 deleted_at，简化）。

CREATE TABLE addresses (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  recipient VARCHAR(32) NOT NULL,
  phone VARCHAR(20) NOT NULL,
  detail VARCHAR(200) NOT NULL,
  lat DOUBLE PRECISION,
  lng DOUBLE PRECISION,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 用户地址列表（按 created_at DESC 列表；带 is_default 排序的子查询）
CREATE INDEX idx_addresses_user_created ON addresses(user_id, created_at DESC);

-- 默认地址唯一（partial unique index）
CREATE UNIQUE INDEX uq_addresses_user_default ON addresses(user_id) WHERE is_default = TRUE;

-- CHECK：非空 + 长度
ALTER TABLE addresses ADD CONSTRAINT addresses_recipient_nonempty
  CHECK (length(trim(recipient)) > 0);
ALTER TABLE addresses ADD CONSTRAINT addresses_detail_nonempty
  CHECK (length(trim(detail)) > 0);