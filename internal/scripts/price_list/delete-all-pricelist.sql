-- ลบข้อมูล price list ทั้งหมด ก่อนอัปโหลดไฟล์ใหม่ผ่าน POST /price/UploadPriceList
--
-- ชุดตารางตรงกับ deleteAllPriceListByScope (upload-pricelist.go:1096-1107)
-- ต่างกันตรงที่ตัวนั้นลบเฉพาะ scope (company_code, site_code) ที่อยู่ในไฟล์
-- ส่วนไฟล์นี้ลบทุก company/site
--
-- ไม่ลบ price_list_formulas (master สูตร) เพราะ formulas_map มี FK ไปหา
-- และการอัปโหลดจะ upsert สูตรจาก sheet price_list_formulars ให้เองอยู่แล้ว
--
--   psql -h <HOST> -U <USER> -d prime_erp -f delete-all-pricelist.sql

\echo '=== ก่อนลบ ==='
SELECT
    (SELECT count(*) FROM price_list_group)                  AS grp,
    (SELECT count(*) FROM price_list_group_key)              AS grp_key,
    (SELECT count(*) FROM price_list_group_term)             AS term,
    (SELECT count(*) FROM price_list_group_extra)            AS extra,
    (SELECT count(*) FROM price_list_group_extra_key)        AS extra_key,
    (SELECT count(*) FROM price_list_sub_group)              AS sub,
    (SELECT count(*) FROM price_list_sub_group_key)          AS sub_key,
    (SELECT count(*) FROM price_list_subgroup_formulas_map)  AS fmap,
    (SELECT count(*) FROM price_list_group_history)          AS grp_hist,
    (SELECT count(*) FROM price_list_sub_group_history)      AS sub_hist;

-- TRUNCATE ทุกตาราง price_list_* ยกเว้น price_list_formulas ในคำสั่งเดียว
-- สร้างลิสต์จาก catalog แทนไล่พิมพ์เอง เพราะยังมีตาราง history/key ที่ผูก FK กันเองอยู่
-- เช่น price_list_sub_group_key_history -> price_list_sub_group_history
-- ไม่ใช้ CASCADE เพื่อไม่ให้ลามไปตารางนอก prefix
DO $$
DECLARE tables text;
BEGIN
    SELECT string_agg(format('%I.%I', schemaname, tablename), ', ')
      INTO tables
      FROM pg_tables
     WHERE schemaname = 'public'
       AND tablename LIKE 'price\_list\_%'
       AND tablename <> 'price_list_formulas';

    RAISE NOTICE 'truncating: %', tables;
    EXECUTE 'TRUNCATE TABLE ' || tables;
END $$;

\echo '=== หลังลบ (ต้องเป็น 0 ทั้งหมด) ==='
SELECT
    (SELECT count(*) FROM price_list_group)                  AS grp,
    (SELECT count(*) FROM price_list_group_key)              AS grp_key,
    (SELECT count(*) FROM price_list_group_term)             AS term,
    (SELECT count(*) FROM price_list_group_extra)            AS extra,
    (SELECT count(*) FROM price_list_group_extra_key)        AS extra_key,
    (SELECT count(*) FROM price_list_sub_group)              AS sub,
    (SELECT count(*) FROM price_list_sub_group_key)          AS sub_key,
    (SELECT count(*) FROM price_list_subgroup_formulas_map)  AS fmap,
    (SELECT count(*) FROM price_list_group_history)          AS grp_hist,
    (SELECT count(*) FROM price_list_sub_group_history)      AS sub_hist;
