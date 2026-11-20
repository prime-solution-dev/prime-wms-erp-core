-- คืน unique key / index ที่ baseline (20261026000000) ไม่มี แต่ migration เดิมบน Crossmax-main เคยสร้าง:
--   2026-08-24-core-indexes.sql        → quotation/sale/delivery_booking unique เลขที่เอกสาร
--   2026-10-15-system-config-running-dedupe.sql → dedupe system_config + ux_system_config_running
--   2026-10-16-price-list-fk-indexes.sql → 2 index ที่ baseline ขาด
-- index ที่มีอยู่แล้ว (เช่น env ที่เคยรันไฟล์เดิม) จะข้าม; index ที่ไฟล์นี้สร้างจะติด COMMENT ไว้ให้ Down ถอดเฉพาะตัวเอง
-- ถ้ามีเลขที่เอกสารซ้ำ CREATE UNIQUE INDEX จะ fail ทั้งไฟล์ (rollback) — ต้องเคลียร์ข้อมูลเองก่อน ไฟล์นี้ไม่แก้ข้อมูลธุรกิจ
-- +goose Up
-- +goose StatementBegin
SET LOCAL lock_timeout = '10s';

-- dedupe system_config (logic เดิมจาก 2026-10-15-system-config-running-dedupe.sql) ไม่แตะเลข running ที่เหลือ
-- 1) RUNNING: เหลือ 1 แถวต่อ config_code (period ล่าสุด แล้วเลขมากสุด เพื่อไม่ให้ออกเลขซ้ำ)
DELETE FROM public.system_config t
USING (
  SELECT ctid, row_number() OVER (
    PARTITION BY config_code
    ORDER BY "json"->>'year' DESC, "json"->>'month' DESC,
             ("json"->>'current_running')::int DESC
  ) rn
  FROM public.system_config WHERE topic_code = 'RUNNING'
) d
WHERE t.ctid = d.ctid AND d.rn > 1;

-- 2) config อื่นที่ซ้ำกันทุกคอลัมน์ ให้เหลือแถวเดียว
DELETE FROM public.system_config t
USING (
  SELECT ctid, row_number() OVER (
    PARTITION BY topic_code, config_code, tenant_id, config_name,
                 cond1, cond2, value, "sequence", remark, "json"::text
  ) rn
  FROM public.system_config
) d
WHERE t.ctid = d.ctid AND d.rn > 1;

DO $m$
DECLARE
  ix text[];
BEGIN
  FOREACH ix SLICE 1 IN ARRAY ARRAY[
    ['quotation_quotation_code_key',                 'CREATE UNIQUE INDEX quotation_quotation_code_key ON public.quotation (quotation_code)'],
    ['sale_sale_code_key',                           'CREATE UNIQUE INDEX sale_sale_code_key ON public.sale (sale_code)'],
    ['delivery_booking_delivery_code_key',           'CREATE UNIQUE INDEX delivery_booking_delivery_code_key ON public.delivery_booking (delivery_code)'],
    ['ux_system_config_running',                     'CREATE UNIQUE INDEX ux_system_config_running ON public.system_config (config_code) WHERE topic_code = ''RUNNING'''],
    ['idx_price_list_sub_group_price_list_group_id', 'CREATE INDEX idx_price_list_sub_group_price_list_group_id ON public.price_list_sub_group (price_list_group_id)'],
    ['idx_price_list_sub_group_key_sub_group_id',    'CREATE INDEX idx_price_list_sub_group_key_sub_group_id ON public.price_list_sub_group_key (sub_group_id)']
  ] LOOP
    IF to_regclass('public.' || ix[1]) IS NULL THEN
      EXECUTE ix[2];
      EXECUTE format('COMMENT ON INDEX public.%I IS %L', ix[1], 'goose:20261120120000');
    END IF;
  END LOOP;
END $m$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- ถอดเฉพาะ index ที่ไฟล์นี้สร้าง (ดูจาก COMMENT) แถว system_config ที่ dedupe ไปแล้วไม่คืน (เป็นแถวซ้ำ/period เก่าของ config_code เดียวกัน)
DO $m$
DECLARE
  r record;
BEGIN
  FOR r IN
    SELECT c.relname FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
    WHERE c.relkind = 'i' AND obj_description(c.oid, 'pg_class') = 'goose:20261120120000'
  LOOP
    EXECUTE format('DROP INDEX IF EXISTS public.%I', r.relname);
  END LOOP;
END $m$;
-- +goose StatementEnd
