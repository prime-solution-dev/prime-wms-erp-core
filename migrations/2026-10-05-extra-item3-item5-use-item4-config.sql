-- ให้หน้า extra ของ GROUP_1_ITEM_3 และ GROUP_1_ITEM_5 ใช้เงื่อนไขเดียวกับ GROUP_1_ITEM_4
--
-- หน้า extra (Extra.vue → convertExtraConfigToColumns) สร้างคอลัมน์จาก config_json ล้วน ๆ
-- จึงคัดลอก config_json ของ ITEM_4 มาทับตรงตัวอักษร (เดิม ITEM_3/5 ใช้ PG03 แทน PG04)
-- ถ้าไม่มีแถว ITEM_4 คำสั่ง UPDATE จะไม่แก้อะไร
--
-- ข้อมูล extra เดิมของ ITEM_3/5 (price_list_group_extra) ไม่แตะ — ผู้ใช้เลือกค่าใหม่และบันทึกเองในหน้า extra
--
-- ตรวจก่อนรัน:
--   SELECT group_code, config_json FROM price_list_extra_config
--   WHERE group_code IN ('GROUP_1_ITEM_3', 'GROUP_1_ITEM_4', 'GROUP_1_ITEM_5');
--
-- Rollback:
--   UPDATE price_list_extra_config t
--   SET config_json = b.config_json, update_by = b.update_by, update_dtm = b.update_dtm
--   FROM price_list_extra_config_backup_20261005 b
--   WHERE t.id = b.id;

BEGIN;

CREATE TABLE IF NOT EXISTS price_list_extra_config_backup_20261005 AS
SELECT id, group_code, config_json, update_by, update_dtm
FROM price_list_extra_config
WHERE group_code IN ('GROUP_1_ITEM_3', 'GROUP_1_ITEM_5');

UPDATE price_list_extra_config t
SET config_json = src.config_json,
    update_by = 'system',
    update_dtm = now()
FROM price_list_extra_config src
WHERE src.group_code = 'GROUP_1_ITEM_4'
  AND t.group_code IN ('GROUP_1_ITEM_3', 'GROUP_1_ITEM_5');

COMMIT;
