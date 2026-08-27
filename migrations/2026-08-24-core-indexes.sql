-- เพิ่ม PRIMARY KEY / UNIQUE / INDEX ให้ 7 ตารางหลักของ quotation -> sale -> delivery booking
--
-- ทำไมต้องทำ: ตรวจ prime_erp แล้วพบว่าทั้ง 7 ตารางนี้ไม่มี index เลยแม้แต่ primary key
-- (schema public มี 27 index จาก 47 ตาราง กองอยู่ที่ purchase / pre_purchase / price_list_*
--  ซึ่งเป็นโมดูลที่ทำทีหลัง)
--
-- ผลที่ตามมา 2 อย่าง:
--   1. delivery_code ซ้ำได้เงียบๆ  ซึ่งมันคือ document_ref ฝั่ง WMS = คีย์ที่ GetOrdersDelivery
--      ใช้ join กลับมา ถ้าซ้ำเมื่อไหร่การคำนวณ remaining qty กับ Pick-pack status จะเพี้ยนทันที
--      (ตัวสร้างเลขเองก็ยังไม่ atomic - ดู get-running-system-config.go)
--   2. ทุก WHERE delivery_code = ? / document_ref = ? / quotation_id = ? เป็น seq scan
--
-- *** ต้องรัน 2026-08-24-core-indexes-preflight.sql ให้ผ่านก่อน ***
-- ไฟล์นี้ไม่ผูก transaction ให้ ถ้าอยากได้ทั้งหมดหรือไม่ได้เลย ให้รันด้วย psql -1
--
--   psql -h <host> -U <user> -d prime_erp -f 2026-08-24-core-indexes.sql
--
-- ไม่ผูกกับการ deploy erp-core - รันก่อนหรือหลังก็ได้ ไม่มีโค้ดตัวไหนต้องแก้ตาม

-- ---------------------------------------------------------------------------
-- 1. PRIMARY KEY
--    ADD PRIMARY KEY จะ set NOT NULL ให้เอง และสร้าง index ชื่อ <table>_pkey
-- ---------------------------------------------------------------------------

ALTER TABLE public.quotation             ADD CONSTRAINT quotation_pkey             PRIMARY KEY (id);
ALTER TABLE public.quotation_item        ADD CONSTRAINT quotation_item_pkey        PRIMARY KEY (id);
ALTER TABLE public.sale                  ADD CONSTRAINT sale_pkey                  PRIMARY KEY (id);
ALTER TABLE public.sale_item             ADD CONSTRAINT sale_item_pkey             PRIMARY KEY (id);
ALTER TABLE public.sale_deposit          ADD CONSTRAINT sale_deposit_pkey          PRIMARY KEY (id);
ALTER TABLE public.delivery_booking      ADD CONSTRAINT delivery_booking_pkey      PRIMARY KEY (id);
ALTER TABLE public.delivery_booking_item ADD CONSTRAINT delivery_booking_item_pkey PRIMARY KEY (id);

-- ---------------------------------------------------------------------------
-- 2. UNIQUE เลขที่เอกสาร
--    นี่คือด่านสุดท้ายที่กันเลขซ้ำจาก running number ที่ยังไม่ atomic
--    ซ้ำเมื่อไหร่จะกลายเป็น insert error ที่มองเห็น แทนที่จะเงียบแล้วไปพังที่ WMS
-- ---------------------------------------------------------------------------

CREATE UNIQUE INDEX quotation_quotation_code_key      ON public.quotation (quotation_code);
CREATE UNIQUE INDEX sale_sale_code_key                ON public.sale (sale_code);
CREATE UNIQUE INDEX delivery_booking_delivery_code_key ON public.delivery_booking (delivery_code);

-- ---------------------------------------------------------------------------
-- 3. INDEX ตามคีย์ที่โค้ดใช้ join / filter จริง
-- ---------------------------------------------------------------------------

-- ทุกครั้งที่อ่านใบ quotation จะดึง item ตาม quotation_id (get-quotation.go)
-- และตอนแปลงเป็น SO จะ UPDATE quotation_item WHERE quotation_id (create-sale.go)
CREATE INDEX quotation_item_quotation_id_idx ON public.quotation_item (quotation_id);

-- เหมือนกันฝั่ง sale (get-sale.go, update-sale.go)
CREATE INDEX sale_item_sale_id_idx    ON public.sale_item (sale_id);
CREATE INDEX sale_deposit_sale_id_idx ON public.sale_deposit (sale_id);

-- delivery_booking_item.document_ref_item = sale_item.sale_item
-- คือ join ที่ใช้นับว่า item ไหนถูก book slot ไปแล้วเท่าไร (repositories/sale/repository.go:259)
CREATE INDEX sale_item_sale_item_idx ON public.sale_item (sale_item);
CREATE INDEX delivery_booking_item_document_ref_item_idx ON public.delivery_booking_item (document_ref_item);

-- GetDelivery กรองด้วยเลข SO (document_ref) และดึง item ตาม delivery_id
CREATE INDEX delivery_booking_document_ref_idx    ON public.delivery_booking (document_ref);
CREATE INDEX delivery_booking_item_delivery_id_idx ON public.delivery_booking_item (delivery_id);

-- หน้า list กรองตาม company/site เกือบทุกครั้ง
CREATE INDEX quotation_company_site_idx        ON public.quotation (company_code, site_code);
CREATE INDEX sale_company_site_idx             ON public.sale (company_code, site_code);
CREATE INDEX delivery_booking_company_site_idx ON public.delivery_booking (company_code, site_code);

-- ---------------------------------------------------------------------------
-- ถ้าตารางไหนใหญ่ (preflight ข้อ 5 เกิน ~100k แถว) และห้าม downtime
-- ให้เปลี่ยนวิธีเป็นแบบไม่ล็อกยาว ทีละตาราง เช่น:
--
--   CREATE UNIQUE INDEX CONCURRENTLY quotation_pkey_idx ON public.quotation (id);
--   ALTER TABLE public.quotation ALTER COLUMN id SET NOT NULL;
--   ALTER TABLE public.quotation ADD CONSTRAINT quotation_pkey PRIMARY KEY USING INDEX quotation_pkey_idx;
--
--   CREATE UNIQUE INDEX CONCURRENTLY quotation_quotation_code_key ON public.quotation (quotation_code);
--   CREATE INDEX CONCURRENTLY quotation_item_quotation_id_idx ON public.quotation_item (quotation_id);
--
-- ข้อควรระวังของ CONCURRENTLY:
--   - รันใน transaction ไม่ได้ ห้ามใช้ psql -1 และห้ามครอบ BEGIN/COMMIT
--   - ถ้าล้มกลางทางจะเหลือ index สถานะ INVALID ต้อง DROP INDEX ทิ้งแล้วสร้างใหม่
--     ตรวจด้วย: select indexrelid::regclass from pg_index where not indisvalid;
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------
-- rollback (ถ้าต้องถอย)
-- ---------------------------------------------------------------------------
-- ALTER TABLE public.quotation             DROP CONSTRAINT IF EXISTS quotation_pkey;
-- ALTER TABLE public.quotation_item        DROP CONSTRAINT IF EXISTS quotation_item_pkey;
-- ALTER TABLE public.sale                  DROP CONSTRAINT IF EXISTS sale_pkey;
-- ALTER TABLE public.sale_item             DROP CONSTRAINT IF EXISTS sale_item_pkey;
-- ALTER TABLE public.sale_deposit          DROP CONSTRAINT IF EXISTS sale_deposit_pkey;
-- ALTER TABLE public.delivery_booking      DROP CONSTRAINT IF EXISTS delivery_booking_pkey;
-- ALTER TABLE public.delivery_booking_item DROP CONSTRAINT IF EXISTS delivery_booking_item_pkey;
-- DROP INDEX IF EXISTS public.quotation_quotation_code_key;
-- DROP INDEX IF EXISTS public.sale_sale_code_key;
-- DROP INDEX IF EXISTS public.delivery_booking_delivery_code_key;
-- DROP INDEX IF EXISTS public.quotation_item_quotation_id_idx;
-- DROP INDEX IF EXISTS public.sale_item_sale_id_idx;
-- DROP INDEX IF EXISTS public.sale_deposit_sale_id_idx;
-- DROP INDEX IF EXISTS public.sale_item_sale_item_idx;
-- DROP INDEX IF EXISTS public.delivery_booking_item_document_ref_item_idx;
-- DROP INDEX IF EXISTS public.delivery_booking_document_ref_idx;
-- DROP INDEX IF EXISTS public.delivery_booking_item_delivery_id_idx;
-- DROP INDEX IF EXISTS public.quotation_company_site_idx;
-- DROP INDEX IF EXISTS public.sale_company_site_idx;
-- DROP INDEX IF EXISTS public.delivery_booking_company_site_idx;
