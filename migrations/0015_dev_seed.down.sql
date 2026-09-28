-- 0015_dev_seed.down.sql
-- 配套 0015_dev_seed.up.sql 回滚（dev-only seed；生产由 catalog-service 维护）。

DELETE FROM packages WHERE name IN ('陪诊半日套餐','陪诊全日套餐','眼科专项陪诊');
DELETE FROM hospitals WHERE name IN ('北京协和医院','北京同仁医院','北京大学第一医院');