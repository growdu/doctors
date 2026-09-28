-- 0016_users_roles_jsonb.up.sql
-- v2 (unified-app) 多角色并发：users 加 roles JSONB 数组列。
--
-- 设计要点：
--   - 保留 role 列（v1 单 role 字段）不动；前 16 版客户端仍按 role 字段判角色
--   - roles JSONB 列：v2 角色数组（如 ["patient","escort"] 兼职陪诊师场景）
--   - 一次迁移：role → roles 数组（兼容期，service 层应优先读 roles，回退 role）
--   - GIN 索引：roles @> '["xxx"]' 查询高效（admin-web 找 escort 列表等）
--
-- 兼容策略：
--   - 服务层先读 roles（len>0 用之）；空数组回退到 role（v1 单角色兼容）
--   - v1.1+ 后端可移除 role 列（plan §4 Task 1.10）
ALTER TABLE users ADD COLUMN roles JSONB NOT NULL DEFAULT '[]'::jsonb;

-- 一次性数据迁移：把现有 role 字段拷到 roles 数组
UPDATE users SET roles = jsonb_build_array(role) WHERE roles = '[]'::jsonb;

-- GIN 索引：按角色反查用户（admin 列表筛选 escort / union 等）
CREATE INDEX idx_users_roles_gin ON users USING GIN (roles);