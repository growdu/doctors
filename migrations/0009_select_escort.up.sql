-- 0009_select_escort.up.sql
-- 订单匹配模式重构 v1.1（抢单→选人）：
--   - 删除 orders.lock_owner / lock_expire_at（v1 抢单锁单用）
--   - 新增 orders.selected_escort_id / escort_pending_expire_at（v1.1 选人 +30s 确认窗口用）
--   - 状态 CHECK 约束新增 selecting_escort / escort_pending_acceptance 两状态（与旧 matching / pending_acceptance 并存）
--   - 新增 idx_orders_selecting（查 escort_pending_expire_at 即将到期）
--   - 新增 escort_availabilities 表（陪诊师空余时段；order-service 状态机驱动 book/release）
--
-- 与 state-machine plan v1.1（selecting_escort / escort_pending_acceptance 两状态）配套。
-- 与 order-lock plan v1.1（Redis SETNX orders:confirm: + 3 新事件）配套。
-- 与 escort-availability plan v1.1（sub-package services/escort/internal/availability）配套。
--
-- 兼容性说明：
--   - 老数据 status='matching'/'pending_acceptance' 仍保留在 CHECK 内（v1 流程仍可用）
--   - 老 orders.lock_owner / lock_expire_at 数据若存在则丢失（v1 数据迁移另行排期；本迁移加 IF EXISTS 防御）

-- 1. 删除旧的抢单锁单字段
ALTER TABLE orders DROP COLUMN IF EXISTS lock_owner;
ALTER TABLE orders DROP COLUMN IF EXISTS lock_expire_at;
DROP INDEX IF EXISTS idx_orders_lock;

-- 2. 新增选人 + 30s 确认窗口字段
ALTER TABLE orders ADD COLUMN selected_escort_id BIGINT REFERENCES users(id);
ALTER TABLE orders ADD COLUMN escort_pending_expire_at TIMESTAMPTZ;

-- 3. 索引服务于"查 30s 确认窗口即将到期"任务（escort_pending_expire_at scanner）
CREATE INDEX idx_orders_selecting ON orders(escort_pending_expire_at)
  WHERE selected_escort_id IS NOT NULL;

-- 4. 替换 CHECK 约束（含 selecting_escort + escort_pending_acceptance 两新状态）
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN (
  'created','paid','matching','pending_acceptance',
  'selecting_escort','escort_pending_acceptance',
  'accepted','in_service',
  'completed','reviewed','refunding','refunded','settling','disputed',
  'closed','canceled'
));

-- 5. 陪诊师空余时段表（escort-service 拥有；order-service 触发 Book/Release）
CREATE TABLE escort_availabilities (
  id BIGSERIAL PRIMARY KEY,
  escort_id BIGINT NOT NULL REFERENCES users(id),
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  status VARCHAR(16) NOT NULL CHECK (status IN ('available','booked','canceled')),
  booked_order_id BIGINT REFERENCES orders(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_escort_avail ON escort_availabilities(escort_id, start_at);
CREATE INDEX idx_escort_avail_time ON escort_availabilities(start_at, end_at)
  WHERE status = 'available';