-- BACKFILL: แก้ pdc_percent / due_percent ที่เล็กไป 100 เท่า
-- วันที่: 2026-09-11
-- ที่มา: spec docs/superpowers/specs/2026-09-11-pricelist-issues-triage-design.md
-- รายงานประกอบ: docs/superpowers/reports/2026-09-11-pricelist-term-percent-convention.sql
--
-- ####################################################################
-- #  ไฟล์นี้มีคำสั่ง UPDATE — ผู้ใช้ต้องตรวจและรันเอง                      #
-- #  ผู้เขียนแผนไม่ได้รันไฟล์นี้                                          #
-- #  ต้องรันขั้นที่ 0 ก่อนทุกครั้ง และรันทั้งหมดใน transaction เดียว        #
-- ####################################################################
--
-- บริบท
--   มาตรฐานที่ธุรกิจยืนยัน: 1 = 1% และ 0.1 = 0.1%
--   คือ pdc_percent / due_percent เก็บเป็นจำนวนเปอร์เซ็นต์ และ baht = price * percent / 100
--   upload-pricelist.go parsePercent หารด้วย 100 → cell "3.0%" ถูกเก็บเป็น 0.03
--   คอลัมน์ baht ถูกต้องอยู่แล้ว ยกเว้นแถวที่ผู้ใช้เคยแตะช่อง adjust บนหน้าจอ
--
-- ลำดับในไฟล์นี้สำคัญ: ต้องแก้ baht ก่อนแก้ percent
--   เพราะการคำนวณ baht ใหม่อ้าง percent ค่าเดิมที่ยังไม่ได้คูณ 100


-- ####################################################################
-- ขั้นที่ 0 — ตรวจก่อนแก้ (ต้องรันและอ่านผลก่อนไปขั้นต่อไป)
-- ####################################################################
--
-- ข้อควรระวังที่ต้องตรวจด้วยตาก่อน
--   เกณฑ์ที่ใช้คือ "percent > 0 AND percent < 1 = เพี้ยน"
--   เกณฑ์นี้จะผิดถ้ามีแถวที่ตั้งใจให้เป็นอัตราต่ำกว่า 1% จริง (เช่น 0.5 = 0.5%)
--   ข้อมูลเมื่อ 2026-09-11 มีค่าอยู่ในช่วง 0.01–0.04 เท่านั้น ซึ่งอ่านเป็น 0.01%–0.04%
--   ไม่ได้ในทางธุรกิจ จึงถือว่าเพี้ยนทั้งหมด
--   **ถ้าผลของ query ข้างล่างมีค่าที่ดูเป็นอัตราจริง (เช่น 0.5 หรือ 0.75) ให้หยุด
--     และคัดแถวนั้นออกด้วยมือก่อน**

SELECT 'pdc_percent' AS col, pdc_percent AS value, COUNT(*) AS rows
FROM price_list_group_term
WHERE pdc_percent > 0 AND pdc_percent < 1
GROUP BY 1, 2
UNION ALL
SELECT 'due_percent', due_percent, COUNT(*)
FROM price_list_group_term
WHERE due_percent > 0 AND due_percent < 1
GROUP BY 1, 2
ORDER BY 1, 2;

-- ผลที่คาดไว้เมื่อ 2026-09-11
--   pdc_percent : 0.010 0.015 0.020 0.025 0.030          รวม 46 แถว
--   due_percent : 0.015 0.020 0.025 0.030 0.035 0.040    รวม 48 แถว
--   ไม่มีค่าใดที่อ่านเป็นอัตราจริงได้ → เดินต่อได้


-- ####################################################################
-- ขั้นที่ 1-3 — แก้ข้อมูล (รันทั้งบล็อกใน transaction เดียว)
-- ####################################################################

BEGIN;

-- ขั้นที่ 1 : คำนวณ baht ใหม่เฉพาะแถวที่ baht เสียหาย
--   เงื่อนไขคัด: baht ปัจจุบันไม่ตรงกับค่าที่ควรเป็นเมื่อตีความ percent เป็น percent*100
--   แถวที่ baht ยังถูกอยู่จะไม่ถูกแตะ เพื่อไม่ให้เกิดความต่างจากการปัดเลข
--   คาดว่าแก้ pdc 13 แถว

UPDATE price_list_group_term AS t
SET pdc = ROUND((g.price_weight * (t.pdc_percent * 100) / 100)::numeric, 2)
FROM price_list_group AS g
WHERE g.id = t.price_list_group_id
  AND t.pdc_percent > 0
  AND t.pdc_percent < 1
  AND ROUND(t.pdc::numeric, 2)
      <> ROUND((g.price_weight * (t.pdc_percent * 100) / 100)::numeric, 2);

-- คาดว่าแก้ due 15 แถว

UPDATE price_list_group_term AS t
SET due = ROUND((g.price_weight * (t.due_percent * 100) / 100)::numeric, 2)
FROM price_list_group AS g
WHERE g.id = t.price_list_group_id
  AND t.due_percent > 0
  AND t.due_percent < 1
  AND ROUND(t.due::numeric, 2)
      <> ROUND((g.price_weight * (t.due_percent * 100) / 100)::numeric, 2);


-- ขั้นที่ 2 : แก้ percent ให้เป็นมาตรฐานจำนวนเปอร์เซ็นต์
--   ต้องทำหลังขั้นที่ 1 เพราะขั้นที่ 1 อ้าง percent ค่าเดิม
--   คาดว่าแก้ pdc_percent 46 แถว และ due_percent 48 แถว

UPDATE price_list_group_term
SET pdc_percent = pdc_percent * 100
WHERE pdc_percent > 0
  AND pdc_percent < 1;

UPDATE price_list_group_term
SET due_percent = due_percent * 100
WHERE due_percent > 0
  AND due_percent < 1;


-- ขั้นที่ 3 : ตรวจผลก่อน COMMIT
--   ทั้งสอง query ต้องคืน 0 แถว ถ้าไม่ใช่ ให้ ROLLBACK

-- 3.1 ต้องไม่มีแถวที่ baht ไม่ตรงกับ percent อีก
SELECT g.group_name, t.term_code, g.price_weight,
       t.pdc_percent, t.pdc, t.due_percent, t.due
FROM price_list_group_term t
JOIN price_list_group g ON g.id = t.price_list_group_id
WHERE g.price_weight <> 0
  AND (
        (t.pdc_percent <> 0 AND ROUND(t.pdc::numeric, 2)
             <> ROUND((g.price_weight * t.pdc_percent / 100)::numeric, 2))
     OR (t.due_percent <> 0 AND ROUND(t.due::numeric, 2)
             <> ROUND((g.price_weight * t.due_percent / 100)::numeric, 2))
      );

-- 3.2 ต้องไม่มี percent ที่ยังเล็กกว่า 1 เหลืออยู่ (ยกเว้นศูนย์)
SELECT 'pdc_percent' AS col, COUNT(*) AS rows_remaining
FROM price_list_group_term WHERE pdc_percent > 0 AND pdc_percent < 1
UNION ALL
SELECT 'due_percent', COUNT(*)
FROM price_list_group_term WHERE due_percent > 0 AND due_percent < 1;

-- 3.3 ตรวจว่าค่าที่ได้อยู่ในช่วงที่สมเหตุสมผล (คาดว่า 1–6)
SELECT MIN(pdc_percent) AS pdc_min, MAX(pdc_percent) AS pdc_max,
       MIN(due_percent) AS due_min, MAX(due_percent) AS due_max
FROM price_list_group_term
WHERE pdc_percent <> 0 OR due_percent <> 0;


-- ถ้าผลขั้นที่ 3 ถูกต้องทั้งหมด
--   3.1 คืน 0 แถว
--   3.2 คืน 0 ทั้งสองบรรทัด
--   3.3 ค่าอยู่ในช่วง 1–6
-- ให้รัน
--   COMMIT;
-- ถ้าไม่ใช่ ให้รัน
--   ROLLBACK;
--
-- ไฟล์นี้ไม่มี COMMIT ในตัวโดยเจตนา เพื่อบังคับให้อ่านผลขั้นที่ 3 ก่อน


-- ####################################################################
-- หลัง COMMIT แล้วจึง deploy PR F (ตัด / 100 ออกจาก parsePercent)
-- ####################################################################
