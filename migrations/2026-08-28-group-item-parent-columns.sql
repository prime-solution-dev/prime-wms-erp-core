-- เพิ่มคอลัมน์สาย parent ของ group_item ให้ตรงกับฝั่ง WMS (prime-wms-product-core)
--
-- WMS เก็บ parent เป็น code คู่ (parent_group_code, parent_group_item_code) ไม่ใช่ FK
-- เพราะ CreateInitGroupItem ลบ-สร้าง item ใหม่ทุกครั้งที่ save — uuid จึงไม่คงที่
-- snapshot ที่ sync มาจาก WMS ส่ง 2 ฟิลด์นี้มาด้วยเสมอ ต้อง apply ก่อน deploy SyncGroupMaster
--
-- วิธีรัน:
--   psql -h <HOST> -U <USER> -d prime_erp -f migrations/2026-08-28-group-item-parent-columns.sql
ALTER TABLE group_item ADD COLUMN IF NOT EXISTS parent_group_code varchar(255);
ALTER TABLE group_item ADD COLUMN IF NOT EXISTS parent_group_item_code varchar(255);
