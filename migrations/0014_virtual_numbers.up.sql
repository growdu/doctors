-- 0014_virtual_numbers.up.sql
-- user-service 虚拟号：订单 / 服务阶段对陪诊师 + 患者屏蔽真实号，
-- 通过中间号路由（v1 mock：生成 11 位号段，写表即生效）。
--
-- 设计要点：
--   - 一次分配：1 个订单（order_id）→ 1 条 virtual_number（patient ↔ escort）；
--   - phone：11 位 mock 号段（如 17000000001），由 DB / 应用生成；
--   - expire_at：默认 = service_start_at + duration_hours（应用层传）；
--   - status：active | expired | released；service 层自动算；
--   - 同一订单仅 1 条 active（partial unique）；
--   - patient_id / escort_id 引用 users.id（不要求 FK 强制，因为 order_id 在外部服务）。

CREATE TABLE virtual_numbers (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL,
  patient_id BIGINT NOT NULL REFERENCES users(id),
  escort_id BIGINT NOT NULL REFERENCES users(id),
  phone VARCHAR(20) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','expired','released')),
  expire_at TIMESTAMPTZ NOT NULL,
  released_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 同一订单仅 1 条 active（partial unique index）
CREATE UNIQUE INDEX uq_virtual_numbers_order_active ON virtual_numbers(order_id)
  WHERE status = 'active';

-- 按患者 / 陪诊师 / 订单 索引（用于查询）
CREATE INDEX idx_virtual_numbers_patient ON virtual_numbers(patient_id, status);
CREATE INDEX idx_virtual_numbers_escort ON virtual_numbers(escort_id, status);

-- CHECK：phone 格式（11 位数字 / 固话，宽松）
ALTER TABLE virtual_numbers ADD CONSTRAINT virtual_numbers_phone_check
  CHECK (phone ~ '^[0-9\-\+]{7,20}$');

-- CHECK：expire_at > created_at
ALTER TABLE virtual_numbers ADD CONSTRAINT virtual_numbers_expire_check
  CHECK (expire_at > created_at);