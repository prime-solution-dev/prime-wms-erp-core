-- ซ่อมข้อมูล price_list_group_extra_key ที่เสียจากบั๊กผูกคีย์ด้วย extra_key
--
-- ต้นเหตุ: upload-pricelist.go ผูก price_list_group_extra_key กลับหา extra ด้วยสตริง
-- extra_key ซึ่ง gen มาจากค่า PG01..PG10 · extra สองแถวที่ PG ชุดเดียวกัน
-- (เช่น "30 to 38" กับ "> 38" ของเกรดเดียวกัน) จึงมี extra_key ซ้ำกันเสมอ
-- คีย์ของทุกแถวถูกยกไปให้แถวแรกหมด แถวที่เหลือได้ 0 คีย์
--
-- ผลที่เกิด:
--   1. หน้าเว็บกด Update ไม่ผ่าน — "price_list_group_extra_keys ห้ามว่าง" (ITEM_4, ITEM_7)
--   2. ราคาผิด — extra ที่ไม่มีคีย์ผ่านการจับคู่ "ตรงทุกคีย์" โดยอัตโนมัติ
--      (ITEM_4 บวกให้ 78/207 subgroup · ITEM_7 บวกให้ 2/47)
--
-- โค้ดถูกแก้แล้ว (ผูกด้วย RowNo + guard ข้าม extra ที่ไม่มีคีย์) ไฟล์นี้ซ่อมข้อมูลเก่า
--
-- apply ด้วยมือ: psql "$database_gorm_url_prime_erp" -f migrations/2026-09-14-repair-extra-keys.sql
-- ต้องการ PostgreSQL 13+ (gen_random_uuid built-in)
--
-- *** หลัง apply ต้องเรียก POST /api/erp/price/SubGroup/UpdateLatest
--     {"update_type":"group","group_codes":[...]} ของ group ที่ถูกแก้ เพื่อคิดราคาใหม่ ***

BEGIN;

-- ก่อนแก้: ดูขอบเขตความเสียหาย
-- SELECT g.group_code, count(*) FILTER (WHERE kc.n = 0) AS extra_ไม่มีคีย์,
--        count(*) FILTER (WHERE kc.n > kd.n) AS extra_คีย์ซ้ำ
-- FROM price_list_group_extra e
-- JOIN price_list_group g ON g.id = e.price_list_group_id
-- CROSS JOIN LATERAL (SELECT count(*) n FROM price_list_group_extra_key k WHERE k.group_extra_id = e.id) kc
-- CROSS JOIN LATERAL (SELECT count(DISTINCT (code, value, seq)) n FROM price_list_group_extra_key k WHERE k.group_extra_id = e.id) kd
-- GROUP BY g.group_code ORDER BY 1;

-- 1) ลบคีย์ที่ซ้ำกันบน extra แถวแรก (เก็บไว้ชุดเดียว)
DELETE FROM price_list_group_extra_key k
USING price_list_group_extra_key dup
WHERE k.group_extra_id = dup.group_extra_id
  AND k.code IS NOT DISTINCT FROM dup.code
  AND k.value IS NOT DISTINCT FROM dup.value
  AND k.seq IS NOT DISTINCT FROM dup.seq
  AND k.ctid > dup.ctid;

-- 2) แจกชุดคีย์คืนให้ extra ที่เหลือ 0 คีย์
--    extra_key ถูก gen จากค่า PG ทั้งชุด extra ที่ extra_key เดียวกันใน group เดียวกัน
--    จึงมีชุดคีย์เหมือนกันเสมอ — copy จากพี่น้องได้ตรงตัว
--
--    copy จากพี่น้อง "แถวเดียว" ไม่ใช่ union ข้ามทุกแถว · ถ้าวันหนึ่งมีข้อมูลที่
--    extra_key ชนกันแต่ชุดคีย์ต่างกันจริง การ union จะได้คีย์ปนกันจนไม่มีทาง
--    matchedAllKeys ได้เลย = extra ตายเงียบ
INSERT INTO price_list_group_extra_key (id, group_extra_id, seq, code, value)
SELECT gen_random_uuid(), e.id, src.seq, src.code, src.value
FROM price_list_group_extra e
JOIN LATERAL (
    SELECT k.seq, k.code, k.value
    FROM price_list_group_extra_key k
    WHERE k.group_extra_id = (
        SELECT sib.id
        FROM price_list_group_extra sib
        WHERE sib.price_list_group_id = e.price_list_group_id
          AND sib.extra_key = e.extra_key
          AND EXISTS (SELECT 1 FROM price_list_group_extra_key kk WHERE kk.group_extra_id = sib.id)
        ORDER BY sib.id
        LIMIT 1
    )
) src ON TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM price_list_group_extra_key ke WHERE ke.group_extra_id = e.id
);

-- 3) ตรวจผล — ถ้ายังมี extra ไร้คีย์เหลืออยู่ ให้ abort ทั้ง transaction
--
--    เคสที่ซ่อมไม่ได้คือ extra ที่ไม่มีพี่น้อง extra_key เดียวกันเหลือคีย์อยู่เลย
--    (เช่น แถวที่ถือคีย์ไว้ถูกลบไปตอน re-upload) ปล่อย commit ไปจะยังกด Update
--    ไม่ผ่านเหมือนเดิมแต่ไม่มีใครรู้ · abort เพื่อให้คนรันเห็นและมาดูด้วยมือ
--
--    ถ้าตรวจแล้วยอมรับได้ว่าบาง group ซ่อมไม่ได้ ให้ comment บล็อกนี้ออกแล้วรันใหม่
--    การซ่อมส่วนที่ซ่อมได้จะถูก commit ตามปกติ
DO $$
DECLARE n int;
BEGIN
    SELECT count(*) INTO n
    FROM price_list_group_extra e
    WHERE NOT EXISTS (SELECT 1 FROM price_list_group_extra_key k WHERE k.group_extra_id = e.id);

    IF n > 0 THEN
        RAISE EXCEPTION 'ยังมี extra ที่ไม่มีคีย์เหลืออยู่ % แถว ซ่อมจากพี่น้องไม่ได้ ต้องตรวจด้วยมือ (ดูคอมเมนต์เหนือบล็อกนี้)', n;
    END IF;
END $$;

-- ดูรายตัวว่าเหลือแถวไหนบ้าง (ใช้ตอน block ด้านบน abort)
-- SELECT g.group_code, e.id, e.extra_key
-- FROM price_list_group_extra e
-- JOIN price_list_group g ON g.id = e.price_list_group_id
-- WHERE NOT EXISTS (SELECT 1 FROM price_list_group_extra_key k WHERE k.group_extra_id = e.id);

COMMIT;
