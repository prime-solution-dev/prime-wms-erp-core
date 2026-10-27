# Migrations

คำสั่ง, การเชื่อมต่อ DB, seed และการใช้ใน CI ดูที่ [`../README.md`](../README.md)

## รูปแบบไฟล์

สร้างด้วย `make migrate-create NAME=description` → `migrations/YYYYMMDDHHMMSS_description.sql`

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE approval ADD COLUMN IF NOT EXISTS note TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE approval DROP COLUMN IF EXISTS note;
-- +goose StatementEnd
```

- หนึ่งไฟล์มีทั้ง Up และ Down
- ใช้ `IF NOT EXISTS` / `IF EXISTS` เมื่อทำได้
- ถ้า Down ย้อนแล้วข้อมูลหาย (เช่น ลดขนาด numeric) ให้ Down เป็น `SELECT 1;` พร้อมคอมเมนต์เหตุผล
- ไฟล์ที่รันไปแล้วห้ามแก้ ให้สร้างไฟล์ใหม่

## Seed ตาม branch / ลูกค้า

- โฟลเดอร์ใต้ `seeds/` แต่ละอันเป็น set แยก มีตาราง version ของตัวเอง `goose_seed_<slug>_version` (slug = ตัวพิมพ์เล็ก แทนอักขระอื่นด้วย `_`)
- Jenkins รัน `seeds/<SEED_SET>/` โดยค่าเริ่มต้นคือชื่อ `TARGET_BRANCH` เช่น `seeds/Pacifica-uat/` ไม่ได้รัน `seeds/master/`
- สร้างไฟล์: `make seed-create SET=Pacifica-uat NAME=add_x`
- รันเอง: `make seed-up SET=Pacifica-uat DB_URL='postgres://...'`
- `seeds/dev/` ยังต้องส่ง `CONFIRM=yes` และ Jenkins ไม่รัน dev
- ชื่อ folder ต้องตรงกับ `TARGET_BRANCH` ทุกตัวอักษร (case-sensitive) และมีแค่ `up` ที่ข้ามเมื่อไม่มี folder — คำสั่งอื่น (`status`, `down`, `reset`, `fix`) จะ error
