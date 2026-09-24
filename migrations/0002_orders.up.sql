-- 0002_orders.up.sql
-- 订单表 + 订单事件表（每次状态变更落库，便于审计）。
-- status 11 种与 docs/04 一致；version 用于乐观锁。

CREATE TABLE orders (
  id BIGSERIAL PRIMARY KEY,
  order_no VARCHAR(32) UNIQUE NOT NULL,
  patient_id BIGINT NOT NULL REFERENCES users(id),
  escort_id BIGINT REFERENCES users(id),
  hospital_id BIGINT NOT NULL,
  package_id BIGINT NOT NULL,
  service_start_at TIMESTAMPTZ NOT NULL,
  amount NUMERIC(10,2) NOT NULL,
  final_amount NUMERIC(10,2) NOT NULL,
  status VARCHAR(16) NOT NULL CHECK (status IN (
    'created','paid','matching','accepted','in_service','completed',
    'reviewed','refunding','refunded','closed','canceled'
  )),
  version INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE order_events (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES orders(id),
  from_status VARCHAR(16),
  to_status VARCHAR(16) NOT NULL,
  actor_id BIGINT,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_patient_created ON orders(patient_id, created_at DESC);
CREATE INDEX idx_orders_status_start ON orders(status, service_start_at)
  WHERE status IN ('paid','matching','accepted','in_service');
CREATE INDEX idx_order_events_order ON order_events(order_id, created_at);