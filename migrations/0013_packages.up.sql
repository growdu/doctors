-- 0013_packages.up.sql
-- user-service 服务包：挂在医院下，提供半日 / 全日 / 单项等服务包。
--
-- 设计要点：
--   - hospital_id FK → hospitals.id（必须先建 0012_hospitals）；
--   - type：half_day | full_day | single_item（v1 三档）；
--   - duration_min：服务时长（分钟）；v1 给参考值；下单时校验时长窗口；
--   - price：单价（元；DECIMAL 精度避免浮点漂移）；
--   - status：active | inactive（inactive 不在列表展示）；
--   - 索引：hospital_id + status（按医院筛）；idx_packages_type（按 type）。

CREATE TABLE packages (
  id BIGSERIAL PRIMARY KEY,
  hospital_id BIGINT NOT NULL REFERENCES hospitals(id),
  name VARCHAR(128) NOT NULL,
  type VARCHAR(16) NOT NULL CHECK (type IN ('half_day','full_day','single_item')),
  duration_min INT NOT NULL CHECK (duration_min > 0),
  price NUMERIC(10,2) NOT NULL CHECK (price >= 0),
  status VARCHAR(16) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','inactive')),
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_packages_hospital_status ON packages(hospital_id, status);
CREATE INDEX idx_packages_type ON packages(type);

ALTER TABLE packages ADD CONSTRAINT packages_name_nonempty
  CHECK (length(trim(name)) > 0);