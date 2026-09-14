-- แก้ชื่อหน่วยชิ้นที่แสดงบนหน้าจอจาก 'Pcs' เป็น 'PCS'
--
-- unit master เก็บ unit_code = 'PC' (ของเดิม) ส่วนโค้ดฝั่งอื่นอ้างถึงหน่วยชิ้นว่า 'PCS'
-- (get-compare-price.go, purchase-calc.ts PIECE_CODES) หน้าจอ New PO แสดง unit_name /
-- uom_name ไม่ใช่ code จึงแก้ที่ชื่อ ไม่แตะ code เพราะแถว purchase / product ที่มีอยู่
-- อ้าง 'PC' เป็น FK อยู่ การเปลี่ยน code จะทำให้ lookup พังทั้งระบบ
--
-- unit_name  -> คอลัมน์ Unit และ Cost/Base unit ของ PO Big lot (PrePurchaseItemTable)
-- uom_name   -> คอลัมน์ Cost/Base unit ของ PO Normal (PurchaseItemTable getUnitUomOptions)
-- ทั้งสองต้องแก้พร้อมกัน ไม่งั้น Big lot จะขึ้น PCS แต่ Normal ยังขึ้น Pcs ในคอลัมน์เดียวกัน
--
-- ไม่แตะ unit_method.method_name ('Pcs') เพราะเป็นป้ายของ "Purchase Method"
-- ซึ่งเป็นคนละความหมายกับหน่วยนับ ถ้าต้องการให้เปลี่ยนด้วยมีคำสั่งอยู่ท้ายไฟล์
--
-- วิธีรัน:
--   psql -h <HOST> -U <USER> -d prime_erp -f migrations/2026-09-09-unit-pcs-label.sql

BEGIN;

UPDATE unit
SET unit_name = 'Pcs'
WHERE unit_code = 'PC'
  AND unit_name <> 'Pcs';

UPDATE unit_uom
SET uom_name = 'Pcs'
WHERE uom_code = 'PC'
  AND uom_name <> 'Pcs';

COMMIT;

-- ตรวจผล (ต้องไม่เหลือแถวไหนที่ยังเป็น 'Pcs'):
--   SELECT topic, unit_code, unit_name FROM unit WHERE unit_code = 'PC';
--   SELECT uom_code, uom_name FROM unit_uom WHERE uom_code = 'PC';

-- ถ้าต้องการเปลี่ยนป้าย Purchase Method ด้วย (ต้องยืนยันกับ business ก่อน):
--   UPDATE unit_method SET method_name = 'PCS' WHERE method_code = 'PC' AND method_name <> 'PCS';
