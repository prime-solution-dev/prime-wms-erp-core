-- group_item.value_int เป็น 0 ทุกแถวทั้งตาราง ทำให้ extraConditionMatched
-- (update-latest-pricelist-subgroup.go) เทียบเงื่อนไข extra กับ 0 เสมอ
-- เงื่อนไขอย่าง "<> 30..38" จึงไม่เคย match ส่วน ">=" กับ "<" กลับ match ทุกแถว
--
-- ขนาดจริงถูกเก็บไว้ในคอลัมน์ value เป็น text ("38.00") ตรงกับ item_name ทุกแถว
--
-- ตัวกันถาวรอยู่ที่ deriveGroupItemValueInt ใน SyncGroupMasterWithDB เพราะ sync
-- ลบ group_item ทั้งตารางแล้วสร้างใหม่ ไฟล์นี้แก้เฉพาะข้อมูลชุดที่มีอยู่ตอนนี้
-- เพื่อไม่ต้องรอ sync รอบถัดไป
--
-- ============================================================================
-- ต้องรันตรวจก่อน ห้ามข้าม
-- ============================================================================
--
-- 1) ยืนยันว่าคอลัมน์เก็บทศนิยมได้ ชื่อคอลัมน์คือ value_int แต่ Go model เป็น
--    float64 (internal/models/group.go) ถ้าคอลัมน์จริงเป็น integer/bigint ค่า
--    "1.70" จะถูกปัดเป็น 2 เงียบ ๆ โดยไม่ error ทั้งใน migration นี้และใน sync
--
--    SELECT data_type, numeric_precision, numeric_scale
--    FROM information_schema.columns
--    WHERE table_name = 'group_item' AND column_name = 'value_int';
--
--    ถ้าไม่ใช่ numeric/double precision ที่มีทศนิยม ต้อง ALTER ก่อน:
--    ALTER TABLE group_item ALTER COLUMN value_int TYPE numeric(18,4);
--
-- 2) ยืนยันว่า group ที่ถูกใช้เป็นแกนเงื่อนไขมีแต่ group ที่ value เป็น "ขนาด"
--    บาง group เก็บ value เป็นเลขลำดับ (PG01 = "1.00", "2.00") ไม่ใช่ขนาด
--    หลัง backfill ค่าเหล่านั้นจะกลายเป็น 1, 2 ซึ่งไม่มีความหมาย
--    ไม่เป็นอันตรายตราบใดที่ไม่มี extra row ไหนชี้ condition_code ไปที่ group นั้น
--    (ตรวจ ณ 2026-09-14 บน thaimetal UAT พบเฉพาะ PG06)
--
--    SELECT DISTINCT condition_code FROM price_list_group_extra
--    WHERE trim(coalesce(condition_code, '')) <> '';
--
-- ============================================================================

BEGIN;

-- ตารางสำรองสำหรับย้อนกลับ การ backfill นี้กู้คืนเองไม่ได้เพราะแยกไม่ออกว่า
-- แถวไหนเดิมเป็น 0 โดยธรรมชาติ
CREATE TABLE IF NOT EXISTS group_item_value_int_backup_20260914 AS
SELECT id, value_int FROM group_item;

-- trim ให้ตรงกับ strings.TrimSpace ใน deriveGroupItemValueInt ไม่งั้นแถวที่มี
-- ช่องว่างติดมาจะถูกข้ามที่นี่แต่ถูก derive ตอน sync ทำให้ข้อมูลสองทางไม่ตรงกัน
UPDATE group_item
SET value_int = trim(value)::numeric
WHERE coalesce(value_int, 0) = 0
  AND trim(value) ~ '^-?[0-9]+(\.[0-9]+)?$';

COMMIT;
