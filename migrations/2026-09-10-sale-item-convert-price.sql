-- เก็บราคาจาก price list master ตอน convert ใบ sale order
--
-- คอลัมน์นี้บันทึกราคาที่ master ให้ตอน convert เอาไว้เทียบกับราคาที่ผู้อนุมัติยอมรับ (ราคาจาก quotation)
-- เป็นข้อมูลอ้างอิงล้วน ไม่มีใครคิดเงินจากคอลัมน์นี้ ปลายทางยังอ่าน price_list_unit เหมือนเดิม
-- ใบเก่าที่สร้างมาก่อนจะได้ค่า DEFAULT 0 เนื่องจากไม่เคยบันทึกค่านี้
--
-- วิธีรัน:
--   psql -h <HOST> -U <USER> -d prime_erp -f migrations/2026-09-10-sale-item-convert-price.sql

ALTER TABLE sale_item
    ADD COLUMN IF NOT EXISTS convert_price_list_unit numeric(20,8) DEFAULT 0;

-- ตรวจผล (ต้องเห็นคอลัมน์ convert_price_list_unit มี type numeric):
--   SELECT column_name, data_type FROM information_schema.columns
--   WHERE table_name = 'sale_item' AND column_name = 'convert_price_list_unit';
