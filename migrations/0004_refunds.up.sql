-- 0004_refunds.up.sql
-- 退款单据表 + 退款策略配置表（可热加载）。
-- 评审 C-04 / I-05：退款分段留扩展点（4 档默认策略）。
-- 评审 09 §9.2.1：申请退款→自动原路退回（P0）。

CREATE TABLE refunds (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES orders(id),
  payment_id BIGINT,            -- v1 暂不接 payment-service；预留
  amount NUMERIC(10,2) NOT NULL,
  reason VARCHAR(32) NOT NULL,  -- 'user_cancel' / 'admin_cancel' / 'service_failed' / 'dispute'
  status VARCHAR(16) NOT NULL DEFAULT 'created'
    CHECK (status IN ('created','processing','completed','failed','rejected')),
  external_tx_id VARCHAR(64),  -- v2 微信支付退款单号
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  failure_reason VARCHAR(255)
);

CREATE INDEX idx_refunds_order ON refunds(order_id, created_at DESC);
CREATE INDEX idx_refunds_payment ON refunds(payment_id) WHERE payment_id IS NOT NULL;

CREATE TABLE refund_policies (
  id BIGSERIAL PRIMARY KEY,
  scope VARCHAR(32) NOT NULL DEFAULT 'default',  -- 'default' / 'city:xxx' / 'package:yyy'
  trigger_phase VARCHAR(32) NOT NULL,            -- 'before_paid' / 'after_paid_5min' / 'after_accepted' / 'in_service'
  refund_percent NUMERIC(5,2) NOT NULL,         -- 0~100；如 100.00 / 95.00 / 0.00
  escort_compensation_percent NUMERIC(5,2) NOT NULL DEFAULT 0,  -- 给陪诊师的补偿比例
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  effective_to TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uniq_scope_trigger UNIQUE (scope, trigger_phase)
);

-- v1 默认策略：四档退款规则
INSERT INTO refund_policies (scope, trigger_phase, refund_percent, escort_compensation_percent) VALUES
  ('default', 'before_paid',         100.00, 0.00),
  ('default', 'after_paid_5min',     100.00, 0.00),
  ('default', 'after_accepted',       95.00, 5.00),
  ('default', 'in_service',            0.00, 0.00);