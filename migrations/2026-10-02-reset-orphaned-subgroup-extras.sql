-- ล้าง extra ที่ค้างจาก rule ที่ถูกลบไปแล้ว (ก่อนมี resetOrphanedSubGroupExtras ใน UpdateExtra)
--
-- calculateExtraForSubGroup คืนค่าเดิมเมื่อไม่มี rule ไหน key ตรง ค่าที่ rule ที่ถูกลบ
-- เคยเขียนไว้จึงค้างตลอดไป (UAT 2026-10-02: GROUP_1_ITEM_2 PG04_59 = 1.00,
-- GROUP_1_ITEM_4 = 0.2 / 0.4 รวม 57 แถว)
--
-- ล้างเฉพาะ subgroup ที่
--   1. ไม่มี rule ของ group ตัวเองที่ key ตรงครบ
--   2. group มี rule อย่างน้อยหนึ่งแถว (group ที่ไม่เคยตั้ง rule = ค่าจากอัปโหลด)
--   3. snapshot แรกใน history มี extra = 0 คือค่าจากอัปโหลดเป็น 0 และถูก rule
--      เขียนทีหลัง · subgroup ที่ไม่มี history ไม่ถูกแตะเพราะแยกไม่ออก
--
-- หลังรัน ให้กด Save ที่หน้า extra ของ group ที่ถูกล้าง (หรือเรียก
-- /price/SubGroup/UpdateLatest) เพื่อคำนวณ total_net_price ใหม่
--
-- รันทีละขั้น: preview ก่อน แล้วค่อย UPDATE

CREATE TEMP TABLE orphaned_subgroup_extra AS
SELECT s.id
FROM price_list_sub_group s
WHERE (s.extra_price_unit <> 0 OR s.extra_price_weight <> 0)
  AND EXISTS (SELECT 1 FROM price_list_group_extra e WHERE e.price_list_group_id = s.price_list_group_id)
  AND NOT EXISTS (
    SELECT 1 FROM price_list_group_extra e
    WHERE e.price_list_group_id = s.price_list_group_id
      AND EXISTS (SELECT 1 FROM price_list_group_extra_key k WHERE k.group_extra_id = e.id)
      AND NOT EXISTS (
        SELECT 1 FROM price_list_group_extra_key k
        WHERE k.group_extra_id = e.id
          AND NOT EXISTS (
            SELECT 1 FROM price_list_sub_group_key sk
            WHERE sk.sub_group_id = s.id AND sk.code = k.code AND sk.value = k.value)))
  AND (
    SELECT h.extra_price_unit
    FROM price_list_sub_group_history h
    WHERE h.price_list_group_id = s.price_list_group_id AND h.subgroup_key = s.subgroup_key
    ORDER BY h.expiry_date
    LIMIT 1
  ) = 0;

-- 1) preview: ตรวจจำนวนแถวต่อ group ก่อน (UAT 2026-10-02 คาดว่า 57)
SELECT g.site_code, g.group_code, g.id AS group_id, s.extra_price_unit, count(*) AS rows
FROM orphaned_subgroup_extra o
JOIN price_list_sub_group s ON s.id = o.id
JOIN price_list_group g ON g.id = s.price_list_group_id
GROUP BY g.site_code, g.group_code, g.id, s.extra_price_unit
ORDER BY g.group_code, s.extra_price_unit;

-- 2) ล้าง · SET ฝั่งขวาอ่านค่าก่อน UPDATE จึงได้ before_* เป็นค่าเดิม
BEGIN;
UPDATE price_list_sub_group s
SET before_extra_price_unit   = s.extra_price_unit,
    before_extra_price_weight = s.extra_price_weight,
    extra_price_unit          = 0,
    extra_price_weight        = 0,
    update_by                 = 'migration-2026-10-02',
    update_dtm                = now() AT TIME ZONE 'UTC'
FROM orphaned_subgroup_extra o
WHERE s.id = o.id;
-- ตรวจว่าจำนวนแถวตรงกับ preview แล้วค่อย COMMIT (ไม่ตรง → ROLLBACK)
COMMIT;
