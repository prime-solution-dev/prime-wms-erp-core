-- PREFLIGHT สำหรับ 2026-08-24-core-indexes.sql  ***อ่านอย่างเดียว ไม่แก้ข้อมูล***
--
-- รันไฟล์นี้บน DB ที่จะ migrate ก่อนเสมอ (UAT แล้วค่อย prod)
-- ผลลัพธ์ที่ต้องได้: ทุกคอลัมน์ null_id / dup_id / dup_code เป็น 0
-- ถ้ามีตัวไหนไม่เป็น 0 ห้ามรัน migration ต้องเคลียร์ข้อมูลก่อน
--
--   psql -h <host> -U <user> -d prime_erp -f 2026-08-24-core-indexes-preflight.sql

\echo '=== 1. id ต้องไม่ null และไม่ซ้ำ (เงื่อนไขของ PRIMARY KEY) ==='

select 'quotation'             as table_name, count(*) as rows, count(*) filter (where id is null) as null_id, count(*) - count(distinct id) as dup_id from public.quotation
union all select 'quotation_item',        count(*), count(*) filter (where id is null), count(*) - count(distinct id) from public.quotation_item
union all select 'sale',                  count(*), count(*) filter (where id is null), count(*) - count(distinct id) from public.sale
union all select 'sale_item',             count(*), count(*) filter (where id is null), count(*) - count(distinct id) from public.sale_item
union all select 'sale_deposit',          count(*), count(*) filter (where id is null), count(*) - count(distinct id) from public.sale_deposit
union all select 'delivery_booking',      count(*), count(*) filter (where id is null), count(*) - count(distinct id) from public.delivery_booking
union all select 'delivery_booking_item', count(*), count(*) filter (where id is null), count(*) - count(distinct id) from public.delivery_booking_item
order by table_name;

\echo ''
\echo '=== 2. เลขที่เอกสารต้องไม่ซ้ำ (เงื่อนไขของ UNIQUE INDEX) ==='

select 'quotation.quotation_code'      as column_name, count(*) - count(distinct quotation_code) as dup_code, count(*) filter (where coalesce(quotation_code, '') = '') as blank_code from public.quotation
union all select 'sale.sale_code',              count(*) - count(distinct sale_code),     count(*) filter (where coalesce(sale_code, '') = '')     from public.sale
union all select 'delivery_booking.delivery_code', count(*) - count(distinct delivery_code), count(*) filter (where coalesce(delivery_code, '') = '') from public.delivery_booking
order by column_name;

\echo ''
\echo '=== 3. ถ้าข้อ 2 ไม่เป็น 0 ใช้ query นี้หาตัวที่ซ้ำ ==='
\echo '-- select quotation_code, count(*) from public.quotation group by 1 having count(*) > 1;'
\echo '-- select sale_code, count(*) from public.sale group by 1 having count(*) > 1;'
\echo '-- select delivery_code, count(*) from public.delivery_booking group by 1 having count(*) > 1;'

\echo ''
\echo '=== 4. index ที่มีอยู่ตอนนี้ (ถ้าว่าง = ยังไม่มีอะไรเลย ตามที่คาด) ==='

select tablename, indexname, indexdef
from pg_indexes
where schemaname = 'public'
  and tablename in ('quotation', 'quotation_item', 'sale', 'sale_item', 'sale_deposit',
                    'delivery_booking', 'delivery_booking_item')
order by tablename, indexname;

\echo ''
\echo '=== 5. ขนาดตาราง ใช้ตัดสินใจว่าต้องใช้ CONCURRENTLY ไหม ==='
\echo '-- ต่ำกว่า ~100k แถว: รัน migration ปกติได้เลย ล็อกแป๊บเดียว'
\echo '-- มากกว่านั้น: ดูหมายเหตุท้ายไฟล์ migration เรื่อง CREATE INDEX CONCURRENTLY'

select relname as table_name, n_live_tup as approx_rows, pg_size_pretty(pg_total_relation_size(relid)) as size
from pg_stat_user_tables
where schemaname = 'public'
  and relname in ('quotation', 'quotation_item', 'sale', 'sale_item', 'sale_deposit',
                  'delivery_booking', 'delivery_booking_item')
order by n_live_tup desc;
