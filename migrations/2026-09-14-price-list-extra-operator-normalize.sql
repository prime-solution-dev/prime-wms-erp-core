-- ปรับข้อมูล extra ให้ตรงกับ contract ใหม่ของ extraConditionMatched
--
-- (1) operator "to" คือ label ของ BETWEEN บนหน้าจอ (OPERATOR_OPTIONS ฝั่ง web:
--     { label: 'to', value: '<>' }) ไม่ใช่ operator ที่ extraConditionMatched
--     รองรับ ผลคือเงื่อนไขเหล่านั้นตกเข้า default แล้วคืน false เสมอ และ
--     validateExtras reject ทั้งหน้าตอนกด Update
--     (พบที่ GROUP_1_ITEM_4 หมวดเหล็กท่อ 12 แถว ทุกแถวมี min กับ max ครบคู่
--     ซึ่งตรงกับความหมายของ BETWEEN)
--     ตัวกันถาวรอยู่ที่ normalizeExtraOperator ใน upload-pricelist.go
--
-- (2) operator ">" และ ">=" เดิมอ่านค่าจาก cond_range_min แต่หน้าจอล็อคช่องนั้น
--     ไว้ให้กรอกได้เฉพาะ BETWEEN (ExtraPriceTable.vue) ค่าที่ผู้ใช้พิมพ์จึงลง
--     cond_range_max เสมอ ตอนนี้ backend อ่าน max ให้ตรงกับหน้าจอแล้ว
--     แถว legacy ที่เก็บค่าไว้ใน min โดยมี max = 0 จะกลายเป็น "> 0" คือ match
--     ทุกขนาด จึงต้องย้ายค่ามาไว้ที่ max
--
--     ตรวจ ณ 2026-09-14 บน thaimetal UAT ไม่พบแถวรูปนี้ (มีแถว ">" เดียวคือ
--     min=0 max=38 ซึ่งเป็นรูปแบบใหม่อยู่แล้ว) แต่ site/company อื่นอาจมี
--     คำสั่งนี้จึงเขียนให้เป็น no-op เมื่อไม่มีข้อมูลที่เข้าเงื่อนไข
--
--     ตรวจก่อนรันได้ด้วย:
--     SELECT operator, cond_range_min, cond_range_max, count(*)
--     FROM price_list_group_extra WHERE operator IN ('>', '>=')
--     GROUP BY 1,2,3 ORDER BY 4 DESC;

BEGIN;

CREATE TABLE IF NOT EXISTS price_list_group_extra_backup_20260914 AS
SELECT id, operator, cond_range_min, cond_range_max FROM price_list_group_extra;

UPDATE price_list_group_extra
SET operator = '<>'
WHERE lower(trim(operator)) = 'to';

UPDATE price_list_group_extra
SET cond_range_max = cond_range_min,
    cond_range_min = 0
WHERE operator IN ('>', '>=')
  AND coalesce(cond_range_max, 0) = 0
  AND coalesce(cond_range_min, 0) <> 0;

COMMIT;
