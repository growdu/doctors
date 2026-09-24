-- 0007_wallets.up.sql
-- 钱包 T+7 结算核心 3 表 + orders.completed_at 加列。
--
-- 设计要点：
--   - wallets 1 行 / 用户：balance（可用）+ frozen（T+7 锁定）+ total_earned / total_withdrawn 累计。
--   - withdrawals 提现工单：pending → approved → paid / rejected。
--   - billings 账本（流水）：所有 wallet 余额变动落账，用于前端账单页 + 对账。
--   - orders.completed_at 配套：scanner 用 `completed_at + 7 days <= NOW()` 找待释放订单。
--   - 与 refund.pairs 隔离：退款时 wallet 监听 PaymentRefundedEvent，扣 frozen；不在这里反向 FK。

-- orders 表加 completed_at 列（nullable；订单进入 completed 状态时由 service 写入）
ALTER TABLE orders ADD COLUMN completed_at TIMESTAMPTZ;
CREATE INDEX idx_orders_completed ON orders(completed_at)
  WHERE completed_at IS NOT NULL;

-- 钱包账户表（1 行 / 用户；escort + patient 都可能有）
CREATE TABLE wallets (
  user_id BIGINT PRIMARY KEY REFERENCES users(id),
  balance NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
  frozen NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (frozen >= 0),
  total_earned NUMERIC(10,2) NOT NULL DEFAULT 0,
  total_withdrawn NUMERIC(10,2) NOT NULL DEFAULT 0,
  currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wallets_user ON wallets(user_id);

-- 提现申请表
CREATE TABLE withdrawals (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  amount NUMERIC(10,2) NOT NULL CHECK (amount > 0),
  channel VARCHAR(16) NOT NULL CHECK (channel IN ('wx','alipay','mock')),
  account VARCHAR(64) NOT NULL,                -- 脱敏：138****0000
  status VARCHAR(16) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','approved','paid','rejected')),
  external_tx_id VARCHAR(64),                  -- v1 mock：MOCK-TX-{id}
  failure_reason VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  reviewed_by BIGINT REFERENCES users(id),
  paid_at TIMESTAMPTZ
);
CREATE INDEX idx_withdrawals_user_created ON withdrawals(user_id, created_at DESC);
CREATE INDEX idx_withdrawals_status ON withdrawals(status, created_at) WHERE status = 'pending';

-- 账单流水表（所有 wallet 余额变动落账；用于 GET /api/v1/wallet/transactions）
CREATE TABLE billings (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  order_id BIGINT REFERENCES orders(id),       -- 退款 / 收入类必有；admin_adjust 可能为空
  type VARCHAR(32) NOT NULL CHECK (type IN (
    'order_income',         -- 订单完成入账 frozen+
    'frozen_release',        -- T+7 释放 frozen- balance+
    'withdraw',             -- 提现 balance-
    'withdraw_refund',      -- 提现被拒余额回滚 balance+
    'refund_deduct',        -- 退款扣回 frozen-
    'admin_adjust'          -- 管理员手动调账
  )),
  amount NUMERIC(10,2) NOT NULL,               -- signed：正=入账，负=出账
  balance_after NUMERIC(10,2) NOT NULL,
  frozen_after NUMERIC(10,2) NOT NULL,
  note VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_billings_user_created ON billings(user_id, created_at DESC);