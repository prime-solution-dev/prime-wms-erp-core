# Extra GROUP_1_ITEM_3 / GROUP_1_ITEM_5 ใช้เงื่อนไขเดียวกับ GROUP_1_ITEM_4

วันที่: 2026-10-05
Branch: `feature/extra-item3-item5-use-item4-config` (แตกจาก `Develop`)

## เป้าหมาย

หน้า extra `/customize/price-list/extra/GROUP_1_ITEM_3` และ `/customize/price-list/extra/GROUP_1_ITEM_5`
ต้องทำงานและแสดงผลเหมือน `/customize/price-list/extra/GROUP_1_ITEM_4` ทุกอย่าง

## สภาพปัจจุบัน

- หน้า extra (`prime-wms-web/src/views/price-list/Extra.vue`) สร้างคอลัมน์จาก `config_json`
  ของตาราง `price_list_extra_config` ทั้งหมด ผ่าน `convertExtraConfigToColumns()`
  - `is_condition = true` → คอลัมน์ช่วง min/max
  - group ขนาด (PRODUCT_GROUP4) → 2 คอลัมน์ "ขนาด" + "Unit" (hard-code ใน frontend)
  - อื่น ๆ → dropdown คอลัมน์เดียว
- Backend (`get-pricelist.go`) ส่ง `config_json` กลับไปตรง ๆ ไม่มี logic แยกตาม ITEM
- `config_json` ปัจจุบัน:

| group_code | p1 | p2 | p3 | p4 (is_condition) |
|---|---|---|---|---|
| GROUP_1_ITEM_4 | PG01 | PG02 | **PG04** | PG06 |
| GROUP_1_ITEM_3 | PG01 | PG02 | **PG03** | PG06 |
| GROUP_1_ITEM_5 | PG01 | PG02 | **PG03** | PG06 |

## การเปลี่ยนแปลง

ไม่แก้โค้ด เพิ่ม SQL migration เท่านั้น

1. `migrations/2026-10-05-extra-item3-item5-use-item4-config.sql`
   - `UPDATE price_list_extra_config` ของ `GROUP_1_ITEM_3`, `GROUP_1_ITEM_5`
     ให้ `config_json` = `config_json` ของแถว `GROUP_1_ITEM_4` (subquery คัดลอกค่าตรงตัวอักษร
     กันปัญหา `PG04` vs `PRODUCT_GROUP4`)
   - ตั้ง `update_by = 'system'`, `update_dtm = now()`
   - ห่อด้วย transaction; ถ้าไม่มีแถว ITEM_4 จะไม่ทำอะไร (subquery ได้ NULL → ใช้ `WHERE EXISTS` กัน)
2. `migrations/2026-10-05-extra-item3-item5-use-item4-config.down.sql`
   - คืน `config_json` ของ ITEM_3 / ITEM_5 เป็นค่าเดิม (PG01, PG02, PG03, PG06 condition)

ก่อนเขียน migration ต้องยืนยันชื่อคอลัมน์จริงของ `price_list_extra_config` จาก model/repository
และชนิดของ `config_json` (text / json / jsonb)

## ข้อมูล extra เดิม

ไม่แตะ แถว extra เดิมของ ITEM_3 / ITEM_5 ที่ผูก PG03 ยังอยู่ ช่องขนาด (PG04) จะว่าง
ผู้ใช้เลือกค่าแล้วกดบันทึกเองในหน้า extra (ตัดสินใจโดยผู้ใช้ 2026-10-05)

## การทดสอบ

1. Integration test (testcontainers, ตาม pattern ของ `make test-integration`):
   seed `price_list_extra_config` 3 แถว (ITEM_3/4/5 ด้วยค่าปัจจุบัน) → รัน migration up
   → assert `config_json` ของ ITEM_3 และ ITEM_5 เท่ากับ ITEM_4 และแถวอื่นไม่เปลี่ยน
   → รัน down → assert กลับเป็นค่าเดิม
2. Unit test frontend (`prime-wms-web/src/utils/helper/priceListExtra.spec.ts`):
   ส่ง config แบบ ITEM_4 เข้า `convertExtraConfigToColumns()` → assert ได้คอลัมน์ "ขนาด" + "Unit"
   และคอลัมน์ condition PG06 (ถ้า case นี้มี test อยู่แล้ว ไม่เพิ่ม)
3. ตรวจด้วยตาบน UAT หลังรัน migration: เปิดหน้า extra ITEM_3 / 4 / 5 เทียบคอลัมน์

## Deploy

Migration ของ erp-core รันด้วยมือต่อ environment (Thaimetal UAT ก่อน แล้วค่อย production)
