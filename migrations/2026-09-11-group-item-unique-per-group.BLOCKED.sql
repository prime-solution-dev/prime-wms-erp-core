-- ============================================================================
--  ห้ามรันไฟล์นี้ตามลำพัง — ต้องแก้ price-service ให้เสร็จก่อน (ดูหัวข้อ "ติดอะไรอยู่")
--  ไฟล์มี guard ที่ทำให้ abort ทันทีถ้าเผลอรัน ต้องลบ guard ด้วยมือเมื่อพร้อมจริง
-- ============================================================================
--
-- ปัญหา
-- -----
-- group_item.item_code ของ ERP เป็น UNIQUE ทั้งตาราง (group_item_item_code_key) ซึ่งเข้มกว่า
-- กติกาฝั่ง WMS ที่ unique เป็น (group_id, UPPER(item_code)) — item_code เดียวกันอยู่คนละกลุ่มได้
--
-- ERP เก็บ group/group_item เป็นกระจกของ WMS (SyncGroupMasterWithDB ลบทั้งตารางแล้วเขียนทับด้วย
-- snapshot ทั้งก้อน คง id เดิมให้ 1:1) พอ schema สองฝั่งไม่ตรงกัน snapshot ที่ถูกต้องตามกติกา WMS
-- จึงผิดกติกา ERP แล้ว rollback ทั้ง request:
--   sync group master: unexpected status code: 500,
--   body: {"error":"failed to create group items: ERROR: duplicate key value violates
--          unique constraint \"group_item_item_code_key\" (SQLSTATE 23505)"}
-- ที่หนักกว่า error คือ product-core แค่ log แล้ว return 200 — ERP ค้างข้อมูลเก่าแบบเงียบ ๆ
--
-- constraint นี้ไม่มี migration รองรับใน repo (ตารางถูกสร้างนอกโค้ด น่าจะผ่าน AutoMigrate สมัยที่
-- struct ยังมี tag unique) ไฟล์นี้จึงมีหน้าที่พามันกลับเข้า repo พร้อมบริบท
--
-- ติดอะไรอยู่ — ห้ามรันจนกว่าจะแก้ครบ
-- ------------------------------------
-- price-service สร้าง map ของ group_item โดย key ด้วย item_code เดี่ยว บนสมมติฐานว่า unique
-- ทั้งตาราง ถ้าปลด constraint ก่อนแก้ map จะเป็น last-write-wins แล้ว price list จะ "แสดงชื่อ item
-- ผิดโดยไม่มี error" ซึ่งอันตรายกว่า 500 ที่กำลังแก้อยู่
--
--   internal/services/price-service/get-price-detail.go:53    groupItemMap[item.ItemCode] = item
--   internal/services/price-service/get-price-detail.go:161   groupItemMap[groupKey]
--   internal/services/price-service/get-price-detail.go:201   resolveGroupItemName(groupItemMap, sgk.Value)
--   internal/services/price-service/get-pricelist.go:638      groupItemMap[item.ItemCode] = item
--   internal/services/price-service/get-pricelist.go:709      groupMasterItemMap[item.ItemCode] = item
--   internal/services/price-service/get-pricelist.go:790,798  groupMasterItemMap[pk.Value]
--   internal/services/price-service/get-pricelist.go:834      groupItemMap[sgk.Value].ItemName
--   internal/services/price-service/get-price-export-table.go:99  groupItemMap[code]
--
-- 6 ใน 8 จุดมี group code อยู่ในมือแล้ว (sgk.Code / pk.Code) แค่ไม่ได้ใช้ประกอบ key — เปลี่ยน key
-- เป็น code|UPPER(item_code) ได้ตรง ๆ ที่เหลือคือ get-price-detail.go:161 ที่ groupKey มาจาก
-- ExtractGroupKey(SubGroupKey) โดยไม่มี group code คู่มา ต้องหาทางส่ง context เพิ่ม
--
-- ทำไมหยุดไว้ตรงนี้ได้
-- --------------------
-- ข้อมูลจริงตอนนี้ไม่มี item_code ซ้ำข้าม group เลย เคสที่พังคือซ้ำ "ใน group เดียวกัน" ซึ่งปิดแล้ว
-- ที่ฝั่ง WMS (assertNoDuplicateItemCodes + db/2026-09-11-group-item-unique-per-group.sql)
-- ข้อแลกคือ item_code ต้อง unique ทั้งระบบไปก่อน จนกว่าจะแก้ price-service เสร็จ
--
-- วิธีรันเมื่อพร้อมจริง: ลบบล็อก guard ข้างล่าง แล้ว
--   psql -h <HOST> -U <USER> -d prime_erp -f migrations/2026-09-11-group-item-unique-per-group.BLOCKED.sql

-- ----------------------------- guard: ลบทั้งบล็อกนี้เมื่อพร้อม -----------------------------
DO $$
BEGIN
    RAISE EXCEPTION 'migration นี้ถูกกันไว้: ต้องแก้ price-service map lookup (get-price-detail.go, get-pricelist.go, get-price-export-table.go) ให้ key ด้วย (group_code, item_code) ก่อน มิฉะนั้น price list จะแสดงชื่อ item ผิดแบบไม่มี error — อ่านหัวไฟล์';
END $$;
-- --------------------------------------------------------------------------------------

-- 1) ตรวจก่อนรัน: ต้องไม่มีตารางอื่นอ้าง group_item(item_code) เป็น FK มิฉะนั้น DROP จะไม่ผ่าน
-- SELECT conrelid::regclass AS from_table, conname, pg_get_constraintdef(oid)
-- FROM pg_constraint
-- WHERE confrelid = 'group_item'::regclass AND conname <> 'group_item_group_id_fkey';

-- 2) ปลด unique ทั้งตาราง แล้วผูกใหม่ให้ unique ต่อกลุ่ม (case-insensitive ให้ตรงกับ WMS)
--    ใช้ unique index ไม่ใช่ table constraint เพราะ ADD CONSTRAINT UNIQUE ไม่รับ expression UPPER()
ALTER TABLE group_item DROP CONSTRAINT IF EXISTS group_item_item_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_group_item_key
    ON group_item (group_id, UPPER(item_code));

-- 3) ตรวจหลังรัน: ต้องเห็น uq_group_item_key และต้องไม่เห็น group_item_item_code_key
-- SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'group_item';
