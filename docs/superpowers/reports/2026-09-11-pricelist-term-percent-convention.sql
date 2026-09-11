-- รายงาน: มาตรฐานหน่วยของ pdc_percent / due_percent ไม่เป็นหนึ่งเดียว (issue 1)
-- วันที่: 2026-09-11
-- มาตรฐานที่ธุรกิจยืนยันแล้ว (2026-09-11): 1 = 1% และ 0.1 = 0.1%
--   คือ pdc_percent / due_percent เก็บเป็นจำนวนเปอร์เซ็นต์ตรง ๆ และ baht = price * percent / 100
-- ใช้ไฟล์นี้เพื่อระบุแถวที่ต้อง backfill และตรวจผลหลัง backfill
--
-- ทุก query ในไฟล์นี้เป็น SELECT เท่านั้น ไม่มีการเขียนลง DB
-- รันกับ database prime_erp
--
-- บริบท
--   ฝั่งหน้าจอถูกต้อง (prime-wms-web/src/components/priceList/BasePriceTable.vue:419-438)
--     baht = percent / 100 * price   และแสดงผลเป็น "{{ percent }}%"
--   ฝั่ง export ถูกต้อง (get-price-export-table.go:490-578) เป็น passthrough ล้วน
--   ฝั่ง import ผิด (upload-pricelist.go:1197-1204) parsePercent หารด้วย 100
--     cell "3.0%" ใน Excel จึงถูกเก็บเป็น 0.03 ทั้งที่ควรเก็บ 3
--
--   คอลัมน์ baht (pdc / due) ถูกต้องอยู่แล้ว ผิดเฉพาะคอลัมน์ *_percent ที่เล็กไป 100 เท่า
--   ยกเว้นแถวที่ผู้ใช้เคยแตะช่อง adjust บนหน้าจอ ซึ่งหน้าจอคำนวณ baht ใหม่จาก percent
--   ที่เพี้ยน (calculateUpdatedPrice บรรทัด 355-368) แล้วเซฟทับ จึง baht เสียหายด้วย


-- =========================================================================
-- 1. มีสองมาตรฐานปนกันจริงกี่แถว
-- =========================================================================
SELECT
    'pdc_percent' AS column_name,
    CASE WHEN pdc_percent = 0 THEN 'ศูนย์'
         WHEN pdc_percent < 1 THEN 'เศษส่วน (<1) — เช่น 0.01 = 1%'
         ELSE 'จำนวนเต็ม (>=1) — เช่น 1 = 1%' END AS convention,
    COUNT(*) AS rows,
    MIN(pdc_percent) AS min_value,
    MAX(pdc_percent) AS max_value
FROM price_list_group_term
GROUP BY 1, 2
UNION ALL
SELECT
    'due_percent',
    CASE WHEN due_percent = 0 THEN 'ศูนย์'
         WHEN due_percent < 1 THEN 'เศษส่วน (<1) — เช่น 0.015 = 1.5%'
         ELSE 'จำนวนเต็ม (>=1) — เช่น 2 = 2%' END,
    COUNT(*), MIN(due_percent), MAX(due_percent)
FROM price_list_group_term
GROUP BY 1, 2
ORDER BY 1, 3 DESC;

-- ผลเมื่อ 2026-09-11
--   pdc_percent : เศษส่วน 46 แถว (0.01–0.03) | ศูนย์ 12 | จำนวนเต็ม 5 แถว (1–5)
--   due_percent : เศษส่วน 48 แถว (0.015–0.04) | ศูนย์ 12 | จำนวนเต็ม 3 แถว (2–6)
--
-- ตีความตามมาตรฐานที่ยืนยันแล้ว
--   แถว "เศษส่วน" = เสียหายจาก parsePercent ต้อง backfill ด้วย percent * 100
--   แถว "จำนวนเต็ม" = ผู้ใช้พิมพ์ผ่านหน้าจอ ถูกต้องแล้ว ห้ามแตะ


-- =========================================================================
-- 2. แถวไหนที่ค่า baht ไม่ตรงกับสูตรใดเลย = ข้อมูลเพี้ยนที่ต้องตัดสินใจแก้
--    if_fraction = price * percent      (มาตรฐานข้างมาก)
--    if_whole    = price * percent/100  (มาตรฐานหน้าจอ)
-- =========================================================================
SELECT
    g.group_name,
    t.term_code,
    g.price_weight,
    t.pdc_percent,
    t.pdc                                                   AS pdc_stored,
    ROUND((g.price_weight * t.pdc_percent)::numeric, 2)     AS pdc_if_fraction,
    ROUND((g.price_weight * t.pdc_percent / 100)::numeric, 2) AS pdc_if_whole,
    CASE
        WHEN ROUND(t.pdc::numeric, 2)
             = ROUND((g.price_weight * t.pdc_percent)::numeric, 2)       THEN 'ตรงสูตรเศษส่วน'
        WHEN ROUND(t.pdc::numeric, 2)
             = ROUND((g.price_weight * t.pdc_percent / 100)::numeric, 2) THEN 'ตรงสูตรหน้าจอ (เขียนโดย web)'
        ELSE 'ไม่ตรงสูตรใดเลย'
    END AS verdict
FROM price_list_group_term t
JOIN price_list_group g ON g.id = t.price_list_group_id
WHERE t.pdc_percent <> 0
  AND g.price_weight <> 0
ORDER BY verdict, g.group_name, t.term_code;

-- ผลเมื่อ 2026-09-11 — 4 แถวที่ pdc ถูกเขียนด้วยสูตรหน้าจอจนค่าผิดไป 100 เท่า
--   หมวดเหล็กแผ่น  T3  pdc=0.01  ควรเป็น 0.62
--   หมวดเหล็กแบน   T3  pdc=0.01  ควรเป็น 0.64
--   หมวดฉาก ราง    T3  pdc=0.01  ควรเป็น 0.60
--   หมวดตัวซี      T3  pdc=0.01  ควรเป็น 0.58
-- และ 8 แถวที่ percent ถูกเขียนกลับเป็นจำนวนเต็ม (ตัวซี GI T1/T2/T3 เป็นต้น)


-- =========================================================================
-- 3. เช่นเดียวกันสำหรับ due
-- =========================================================================
SELECT
    g.group_name,
    t.term_code,
    g.price_weight,
    t.due_percent,
    t.due                                                     AS due_stored,
    ROUND((g.price_weight * t.due_percent)::numeric, 2)        AS due_if_fraction,
    ROUND((g.price_weight * t.due_percent / 100)::numeric, 2)  AS due_if_whole,
    CASE
        WHEN ROUND(t.due::numeric, 2)
             = ROUND((g.price_weight * t.due_percent)::numeric, 2)       THEN 'ตรงสูตรเศษส่วน'
        WHEN ROUND(t.due::numeric, 2)
             = ROUND((g.price_weight * t.due_percent / 100)::numeric, 2) THEN 'ตรงสูตรหน้าจอ (เขียนโดย web)'
        ELSE 'ไม่ตรงสูตรใดเลย'
    END AS verdict
FROM price_list_group_term t
JOIN price_list_group g ON g.id = t.price_list_group_id
WHERE t.due_percent <> 0
  AND g.price_weight <> 0
ORDER BY verdict, g.group_name, t.term_code;


-- =========================================================================
-- 4. ค้างจากรอบก่อน: subgroup ที่ total_net_price_unit = 0
--    ค่าเหล่านี้จะถูกคำนวณใหม่เมื่อผู้ใช้กดคำนวณเท่านั้น ไม่มี backfill
--    ผลเมื่อ 2026-09-11: 1,194 จาก 1,350 subgroup ทั้งระบบ
-- =========================================================================
SELECT
    g.group_code,
    g.group_name,
    COUNT(*) AS subgroups_with_zero_unit_price
FROM price_list_sub_group s
JOIN price_list_group g ON g.id = s.price_list_group_id
WHERE s.total_net_price_unit = 0
GROUP BY g.group_code, g.group_name
ORDER BY subgroups_with_zero_unit_price DESC;


-- =========================================================================
-- 5. หลักฐาน issue 4a: GROUP_1_ITEM_9 ยุบ 80 subgroup เหลือ 4 แถว
--    (เก็บไว้ที่นี่เพื่อให้ตรวจซ้ำได้หลังแก้ — หลังแก้ต้องได้ 21 แถว collision 0)
-- =========================================================================
WITH sg AS (
    SELECT s.id
    FROM price_list_sub_group s
    JOIN price_list_group g ON g.id = s.price_list_group_id
    WHERE g.group_code = 'GROUP_1_ITEM_9'
),
k AS (
    SELECT sg.id,
           MAX(CASE WHEN kk.code = 'PG02' THEN kk.value END) AS pg02,
           MAX(CASE WHEN kk.code = 'PG07' THEN kk.value END) AS pg07,
           MAX(CASE WHEN kk.code = 'PG06' THEN kk.value END) AS pg06,
           MAX(CASE WHEN kk.code = 'PG03' THEN kk.value END) AS pg03
    FROM sg
    LEFT JOIN price_list_sub_group_key kk ON kk.sub_group_id = sg.id
    GROUP BY sg.id
)
SELECT 'subgroup ทั้งหมด'                        AS metric, COUNT(*)::text AS value FROM k
UNION ALL
SELECT 'แถวที่แสดง — row key = PG02|PG07 (ตอนนี้)',
       COUNT(DISTINCT COALESCE(pg02,'~')||'|'||COALESCE(pg07,'~'))::text FROM k
UNION ALL
SELECT 'แถวที่แสดง — row key = PG02|PG07|PG06 (หลังแก้)',
       COUNT(DISTINCT COALESCE(pg02,'~')||'|'||COALESCE(pg07,'~')||'|'||COALESCE(pg06,'~'))::text FROM k
UNION ALL
SELECT 'cell (row,PG03) ที่ subgroup ทับกัน — ตอนนี้', COUNT(*)::text FROM (
    SELECT 1 FROM k
    GROUP BY COALESCE(pg02,'~'), COALESCE(pg07,'~'), COALESCE(pg03,'~')
    HAVING COUNT(*) > 1) a
UNION ALL
SELECT 'cell (row,PG03) ที่ subgroup ทับกัน — หลังแก้', COUNT(*)::text FROM (
    SELECT 1 FROM k
    GROUP BY COALESCE(pg02,'~'), COALESCE(pg07,'~'), COALESCE(pg06,'~'), COALESCE(pg03,'~')
    HAVING COUNT(*) > 1) b;

-- ผลเมื่อ 2026-09-11 : 80 | 4 | 21 | 12 | 0


-- =========================================================================
-- 6. แถวที่ต้อง backfill — percent เล็กไป 100 เท่า (ยังไม่ใช่คำสั่ง UPDATE)
--    เงื่อนไข: percent อยู่ระหว่าง 0 กับ 1 (ไม่รวมศูนย์) = นำเข้าผ่าน Excel
--    ตรวจคู่กับ baht: ถ้า baht ตรงกับ price * (percent*100)/100 แสดงว่า baht ยังดี
--                    แก้แค่ percent พอ · ถ้าไม่ตรง baht เสียหายด้วย ต้องคำนวณใหม่
-- =========================================================================
SELECT
    g.group_name,
    t.term_code,
    g.price_weight,
    t.pdc_percent                                                AS pdc_percent_now,
    t.pdc_percent * 100                                          AS pdc_percent_ควรเป็น,
    t.pdc                                                        AS pdc_baht_now,
    ROUND((g.price_weight * (t.pdc_percent * 100) / 100)::numeric, 2) AS pdc_baht_ควรเป็น,
    CASE
        WHEN ROUND(t.pdc::numeric, 2)
             = ROUND((g.price_weight * (t.pdc_percent * 100) / 100)::numeric, 2)
        THEN 'baht ยังดี — แก้แค่ percent'
        ELSE 'baht เสียหายด้วย — ต้องคำนวณใหม่'
    END AS action
FROM price_list_group_term t
JOIN price_list_group g ON g.id = t.price_list_group_id
WHERE t.pdc_percent > 0
  AND t.pdc_percent < 1
ORDER BY action DESC, g.group_name, t.term_code;


-- =========================================================================
-- 7. เช่นเดียวกันฝั่ง due
-- =========================================================================
SELECT
    g.group_name,
    t.term_code,
    g.price_weight,
    t.due_percent                                                AS due_percent_now,
    t.due_percent * 100                                          AS due_percent_ควรเป็น,
    t.due                                                        AS due_baht_now,
    ROUND((g.price_weight * (t.due_percent * 100) / 100)::numeric, 2) AS due_baht_ควรเป็น,
    CASE
        WHEN ROUND(t.due::numeric, 2)
             = ROUND((g.price_weight * (t.due_percent * 100) / 100)::numeric, 2)
        THEN 'baht ยังดี — แก้แค่ percent'
        ELSE 'baht เสียหายด้วย — ต้องคำนวณใหม่'
    END AS action
FROM price_list_group_term t
JOIN price_list_group g ON g.id = t.price_list_group_id
WHERE t.due_percent > 0
  AND t.due_percent < 1
ORDER BY action DESC, g.group_name, t.term_code;

-- ผลเมื่อ 2026-09-11
--   pdc : baht ยังดี 33 แถว | baht เสียหาย 13 แถว
--   due : baht ยังดี 33 แถว | baht เสียหาย 15 แถว
--   percent ที่ backfill แล้วได้ค่าธุรกิจที่สะอาดทุกแถว (1.5% 2.5% 3% 4%)
--   ซึ่งเป็นหลักฐานว่าการคูณ 100 เป็นการตีความที่ถูก
