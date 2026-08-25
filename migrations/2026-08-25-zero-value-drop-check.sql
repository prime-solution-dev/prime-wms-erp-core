-- ตรวจว่าบั๊ก Updates(struct) กินข้อมูลจริงไปแค่ไหน  (READ ONLY ไม่มี write สักบรรทัด)
--
-- ที่มา: update-sale.go และ update-quotation.go ส่ง struct เข้า GORM Updates()
-- GORM v2 จะข้ามทุกฟิลด์ที่เป็น zero value ทั้งที่หน้าบ้านส่งค่ามาครบ
-- ผู้ใช้ล้างส่วนลดเป็น 0 ล้าง remark เป็นค่าว่าง หรือตั้งค่าขนส่งเป็น 0 จึงไม่ถูกบันทึก
-- คอลัมน์นั้นค้างค่าเดิมไว้ ส่วนคอลัมน์ข้างๆ ที่ค่าไม่เป็นศูนย์ถูกเขียนทับตามปกติ
--
-- แถวที่ query ที่ 2 กับ 3 หาเจอ = แถวที่สองฝั่งนี้ไม่ตรงกันแล้ว
--
-- วิธีรัน:
--   export PGPASSWORD='...'
--   psql -h <HOST> -U dev_theo -d prime_erp -f 2026-08-25-zero-value-drop-check.sql

\echo '=== 1) ประชากรที่เสี่ยง: ใบที่เคยถูกแก้หลังสร้าง ==='

SELECT
    count(*)                                                   AS sale_total,
    count(*) FILTER (WHERE update_date > create_date)           AS sale_edited,
    round(100.0 * count(*) FILTER (WHERE update_date > create_date)
          / nullif(count(*), 0), 1)                             AS pct_edited
FROM sale;

SELECT
    count(*)                                                   AS quotation_total,
    count(*) FILTER (WHERE update_date > create_date)           AS quotation_edited
FROM quotation;


\echo ''
\echo '=== 2) sale_item ที่ส่วนลดค้างไม่ตรงกับยอดก่อน VAT ==='
-- หน้าบ้านคิด subtotal_excl_vat = ราคาก่อนส่วนลด - total_discount แล้วส่งมาทั้งคู่
-- ถ้า total_discount ถูกล้างเป็น 0 แล้วไม่ถูกบันทึก ยอดสองตัวนี้จะไม่ลงรอยกัน
-- ราคาก่อนส่วนลดคิดตาม unit_uom เหมือน calculateBasePrice ฝั่งหน้าบ้าน

WITH item_check AS (
    SELECT
        s.sale_code,
        s.status,
        s.update_date,
        si.sale_item,
        si.unit_uom,
        si.qty,
        si.total_weight,
        si.price_unit,
        si.total_discount,
        si.subtotal_excl_vat,
        CASE WHEN si.unit_uom = 'KG'
             THEN si.total_weight * si.price_unit
             ELSE si.qty * si.price_unit
        END AS base_price
    FROM sale_item si
    JOIN sale s ON s.id = si.sale_id
)
SELECT
    sale_code, status, sale_item, unit_uom,
    base_price,
    total_discount,
    subtotal_excl_vat,
    round((base_price - total_discount - subtotal_excl_vat)::numeric, 2) AS diff,
    update_date
FROM item_check
WHERE abs(base_price - total_discount - subtotal_excl_vat) > 0.01
ORDER BY update_date DESC NULLS LAST
LIMIT 40;

-- นับรวมว่ามีกี่แถว เทียบกับทั้งหมด
WITH item_check AS (
    SELECT
        si.total_discount,
        si.subtotal_excl_vat,
        CASE WHEN si.unit_uom = 'KG'
             THEN si.total_weight * si.price_unit
             ELSE si.qty * si.price_unit
        END AS base_price
    FROM sale_item si
)
SELECT
    count(*)                                                            AS item_total,
    count(*) FILTER (WHERE abs(base_price - total_discount - subtotal_excl_vat) > 0.01)
                                                                        AS item_mismatch
FROM item_check;


\echo ''
\echo '=== 3) หัวใบกับผลรวม item ไม่ตรงกัน ==='
-- หัวใบกับ item เดินผ่าน Updates(struct) คนละรอบ จึงค้างเป็นอิสระจากกัน
-- ไม่ตรงกันเมื่อไหร่แปลว่าฝั่งหนึ่งบันทึก อีกฝั่งไม่บันทึก

SELECT
    s.sale_code,
    s.status,
    s.total_discount                          AS head_discount,
    coalesce(sum(si.total_discount), 0)       AS item_discount,
    s.total_weight                            AS head_weight,
    coalesce(sum(si.total_weight), 0)         AS item_weight,
    s.subtotal_excl_vat                       AS head_subtotal,
    coalesce(sum(si.subtotal_excl_vat), 0)    AS item_subtotal,
    s.update_date
FROM sale s
LEFT JOIN sale_item si ON si.sale_id = s.id
GROUP BY s.id, s.sale_code, s.status, s.total_discount, s.total_weight,
         s.subtotal_excl_vat, s.update_date
HAVING abs(s.total_discount   - coalesce(sum(si.total_discount), 0))   > 0.01
    OR abs(s.subtotal_excl_vat - coalesce(sum(si.subtotal_excl_vat), 0)) > 0.01
ORDER BY s.update_date DESC NULLS LAST
LIMIT 40;


\echo ''
\echo '=== 4) เผื่อไว้: ฟิลด์ที่ควรล้างได้แต่ยังมีค่าอยู่ในใบที่เคยแก้ ==='
-- อ่านประกอบเฉยๆ ดูจากข้อมูลอย่างเดียวบอกไม่ได้ว่าผู้ใช้ตั้งใจล้างหรือตั้งใจให้มีค่า
-- แต่ถ้าตัวเลขตรงนี้เป็นศูนย์ แปลว่าไม่มีอะไรให้ค้างตั้งแต่แรก

SELECT
    count(*) FILTER (WHERE total_discount      > 0) AS has_discount,
    count(*) FILTER (WHERE total_transport_cost > 0) AS has_transport_cost,
    count(*) FILTER (WHERE coalesce(remark, '')        <> '') AS has_remark,
    count(*) FILTER (WHERE coalesce(remark_approval,'') <> '') AS has_remark_approval,
    count(*) FILTER (WHERE coalesce(ref_po_doc, '')    <> '') AS has_ref_po
FROM sale
WHERE update_date > create_date;
