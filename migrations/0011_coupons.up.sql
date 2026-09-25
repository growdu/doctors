-- 0011_coupons.up.sql
-- user-service 优惠券：coupons（券模板，平台发券）+ user_coupons（用户领取实例）。
--
-- 设计要点：
--   - coupons：券模板（type / value / threshold / valid_from / valid_until / status / stock）；
--     type: amount_off | percent_off | full_off；value 与 type 配套（CHECK 约束）；
--     threshold ≥ 0 表示满 X 减。
--   - user_coupons：用户领取实例（FK: coupon_id + user_id）；
--     status: unused | used | expired；used_at 非空即 used；expires_at < now() 即 expired。
--   - 平台发券：直接 INSERT 到 coupons（v1 admin / 运营后台手工）；用户领取 = INSERT user_coupons。
--   - v1 不存 order_id（下单抵扣 v2 接 order-service 再加）。
--   - 每个用户对同一 coupon 限领 1 次（partial unique index）。

-- 券模板
CREATE TABLE coupons (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  type VARCHAR(16) NOT NULL CHECK (type IN ('amount_off','percent_off','full_off')),
  value NUMERIC(10,2) NOT NULL,
  threshold NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (threshold >= 0),
  valid_from TIMESTAMPTZ NOT NULL,
  valid_until TIMESTAMPTZ NOT NULL,
  stock INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
  status VARCHAR(16) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','inactive')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_coupons_status ON coupons(status);
CREATE INDEX idx_coupons_valid ON coupons(valid_until);

-- CHECK：value 合法性
ALTER TABLE coupons ADD CONSTRAINT coupons_value_check CHECK (
  (type = 'amount_off' AND value > 0)
  OR (type = 'percent_off' AND value > 0 AND value <= 9.99)
  OR (type = 'full_off' AND value = 0)
);

-- 用户领取实例
CREATE TABLE user_coupons (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  coupon_id BIGINT NOT NULL REFERENCES coupons(id),
  status VARCHAR(16) NOT NULL DEFAULT 'unused'
    CHECK (status IN ('unused','used','expired')),
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_coupons_user_status ON user_coupons(user_id, status, claimed_at DESC);
CREATE INDEX idx_user_coupons_expires ON user_coupons(expires_at)
  WHERE status = 'unused';

-- 同一用户限领同一 coupon 一次（partial unique index）
CREATE UNIQUE INDEX uq_user_coupons_user_coupon ON user_coupons(user_id, coupon_id);