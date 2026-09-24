-- 0001_users.up.sql
-- 用户表：手机号为主键，区分 patient/escort/admin 三种角色。
-- 实名信息只存哈希 + 末四位（呼应评审 I-04 PII 最小必要）。

CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  phone VARCHAR(20) UNIQUE NOT NULL,
  role VARCHAR(16) NOT NULL CHECK (role IN ('patient','escort','admin')),
  nickname VARCHAR(64),
  avatar_url VARCHAR(255),
  real_name_verified BOOLEAN NOT NULL DEFAULT FALSE,
  id_card_hash VARCHAR(64),
  id_card_tail VARCHAR(8),
  wx_unionid VARCHAR(64) UNIQUE,
  wx_openid_mini VARCHAR(64),
  wx_openid_app VARCHAR(64),
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_wx_unionid ON users(wx_unionid);
CREATE INDEX idx_users_role_status ON users(role, status) WHERE deleted_at IS NULL;