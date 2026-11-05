-- Index สำหรับ FK/filter column ของ GetPriceDetail (ตอนนี้มีแค่ pkey + unique code)
-- หมายเหตุ: CREATE INDEX CONCURRENTLY รันใน transaction ไม่ได้ จึงไม่มี BEGIN/COMMIT
-- และต้องรันทีละ statement (psql -f ได้ แต่ห้ามห่อด้วย -1 / --single-transaction)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_sub_group_price_list_group_id ON price_list_sub_group (price_list_group_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_sub_group_key_sub_group_id ON price_list_sub_group_key (sub_group_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_group_term_price_list_group_id ON price_list_group_term (price_list_group_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_group_extra_price_list_group_id ON price_list_group_extra (price_list_group_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_group_extra_key_group_extra_id ON price_list_group_extra_key (group_extra_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_group_company_site ON price_list_group (company_code, site_code);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_price_list_subgroup_formulas_map_subgroup_code ON price_list_subgroup_formulas_map (price_list_subgroup_code);
