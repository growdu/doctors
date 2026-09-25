-- 0008_admin_work_orders.up.sql
-- 客服工单表：admin-web 的客服工单模块用。
-- 设计要点（按 admin plan §Architecture）：
--   - admin-service 持有本表（work_orders 是 admin 唯一持有的业务表）。
--   - subject_id / subject_type 表达"投诉对象"（polymorphic：order / user / escort / system）。
--   - status: pending / assigned / in_progress / resolved / closed
--   - priority: P0（首响 5min）/ P1（首响 30min）/ P2（首响 2h）/ P3（首响 24h）
--   - category: complaint / refund_consult / escort_issue / system_bug / other

CREATE TABLE work_orders (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),  -- 投诉人
  category VARCHAR(32) NOT NULL CHECK (category IN (
    'complaint','refund_consult','escort_issue','system_bug','other')),
  priority VARCHAR(2) NOT NULL DEFAULT 'P2'
    CHECK (priority IN ('P0','P1','P2','P3')),
  status VARCHAR(16) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','assigned','in_progress','resolved','closed')),
  subject_id BIGINT,                              -- 关联对象 id（订单 / 用户 / 陪诊师 ...）
  subject_type VARCHAR(16) CHECK (subject_type IN ('order','user','escort','system','other') OR subject_type IS NULL),
  assignee_id BIGINT REFERENCES users(id),       -- 受理客服 / admin
  title VARCHAR(128) NOT NULL,
  content TEXT NOT NULL,
  resolution TEXT,                                -- 处理结论
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  closed_at TIMESTAMPTZ,                          -- 关单时间
  sla_due_at TIMESTAMPTZ                          -- 首响 SLA 截止
);

-- 客服工单队列（首响 SLA 监控）
CREATE INDEX idx_work_orders_status_priority ON work_orders(status, priority, sla_due_at)
  WHERE status IN ('pending','assigned');

-- 我的工单（assignee_id 视图）
CREATE INDEX idx_work_orders_assignee ON work_orders(assignee_id, status, updated_at DESC)
  WHERE assignee_id IS NOT NULL;

-- 按对象回溯工单
CREATE INDEX idx_work_orders_subject ON work_orders(subject_type, subject_id)
  WHERE subject_id IS NOT NULL;