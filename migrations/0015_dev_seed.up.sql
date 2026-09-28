-- 0015_dev_seed.up.sql
-- dev 环境种子数据：3 家医院 + 3 个服务包（用 subquery 让 seed 可重入）。
-- 配套 0015_dev_seed.down.sql 一并提交；不进 prod（prod 应由 catalog-service 维护）。
--
-- 字段映射（与 0013_packages.up.sql 对齐）：
--   type: 'half_day' / 'full_day' / 'single_item'
--   duration_min: 整数分钟
--   services / departments: 用 text[] 列（packages 表无此列，业务 JSON 走 description）
--
-- 幂等策略：
--   - hospitals 用唯一索引（idx_hospitals_name）+ ON CONFLICT (name) DO NOTHING
--   - packages 用子查询取 hospital_id（FK 一定已存在）

-- 1. hospitals：唯一约束（保证 idempotent）
CREATE UNIQUE INDEX IF NOT EXISTS uniq_hospitals_name_seed ON hospitals(name) WHERE name IN (
  '北京协和医院','北京同仁医院','北京大学第一医院'
);

INSERT INTO hospitals (name, city_id, level, status, address, phone, departments, description)
VALUES
  ('北京协和医院', 1, '3a', 'active', '北京市东城区帅府园 1 号', '010-69155555', '内科,外科,妇产科,儿科,眼科,耳鼻喉科,皮肤科,中医科', '三甲综合医院'),
  ('北京同仁医院', 1, '3a', 'active', '北京市东城区崇文门内大街 8 号', '010-58266611', '眼科,耳鼻喉科,内科,外科', '三甲专科（眼/耳鼻喉）'),
  ('北京大学第一医院', 1, '3a', 'active', '北京市西城区西什库大街 8 号', '010-83572211', '内科,外科,妇产科,儿科,泌尿外科,肾脏内科', '三甲综合医院')
ON CONFLICT (name) WHERE name IN (
  '北京协和医院','北京同仁医院','北京大学第一医院'
) DO NOTHING;

-- 2. packages：用 subquery 取 hospital_id（FK 一定存在）
INSERT INTO packages (hospital_id, name, type, duration_min, price, status, description)
SELECT h.id, '陪诊半日套餐', 'half_day', 240, 299.00, 'active', '取号+导诊+陪检查+代取药'
  FROM hospitals h WHERE h.name = '北京协和医院'
  AND NOT EXISTS (SELECT 1 FROM packages p WHERE p.name = '陪诊半日套餐')
UNION ALL
SELECT h.id, '陪诊全日套餐', 'full_day', 480, 599.00, 'active', '取号+导诊+陪检查+代取药+住院办理协助'
  FROM hospitals h WHERE h.name = '北京协和医院'
  AND NOT EXISTS (SELECT 1 FROM packages p WHERE p.name = '陪诊全日套餐')
UNION ALL
SELECT h.id, '眼科专项陪诊', 'single_item', 180, 399.00, 'active', '视力检查引导+散瞳陪同+术前检查指引'
  FROM hospitals h WHERE h.name = '北京同仁医院'
  AND NOT EXISTS (SELECT 1 FROM packages p WHERE p.name = '眼科专项陪诊');