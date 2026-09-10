# เก็บราคาตอน convert ไว้เป็นฟิลด์ของตัวเอง — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ใบที่ convert มาแบบราคาไม่ผ่าน ต้องเทียบ "ราคาที่เสนอลูกค้า" กับ "ราคาที่ price list master ให้ตอน convert" ได้ต่อไป แม้หลังกด approve

**Architecture:** เพิ่มคอลัมน์ `sale_item.convert_price_list_unit` เก็บราคาที่ดึงจาก master ตอน convert เป็นข้อมูลอ้างอิงล้วน ไม่มีใครคิดเงินจากมัน แล้วให้คอลัมน์ที่สองบนจอ bind ฟิลด์นี้แทน `price_list_unit`

**Tech Stack:** Go + GORM (erp-core), Vue 3 + TS + Pinia (wms-web), Postgres

**Spec:** `C:\work-prime\erp-core\docs\superpowers\specs\2026-09-10-so-pricelist-approval-columns.md` (งานนี้เป็นส่วนต่อจาก finding I3 ของ final review)

**Branch (มีอยู่แล้ว ทั้ง 2 repo, ต่อยอดจากงานเดิม):** `fix/so-pricelist-approval-columns`
- `C:\work-prime\erp-core` HEAD = 88c42f5
- `C:\work-prime\wms-web` HEAD = 6d68ec6f

## ที่มา

กติกา SA ข้อ 3 (approve แล้ว `price_list_unit` = ราคาจาก quotation) ชนกับข้อ 4 (คงคอลัมน์คู่ไว้ตลอดชีวิตใบ):
พอ approve ทับแล้ว `price_list_unit` กับ `old_price_list_unit` เท่ากัน คอลัมน์คู่เลยโชว์เลขซ้ำ และราคาที่ master ให้ตอน convert หายถาวร

เจ้าของเลือกทาง "เพิ่มฟิลด์ที่ 3" แทนทาง "เลิกทับตอน approve" เพราะ `price_list_unit` มีปลายทางอ่านไปใช้ต่อหลายจุด
(`get-sale-pack.go:349` ส่งเข้าใบส่งของ, `get-compare-price.go` คิดส่วนต่างราคา, delivery-service อีก 4 ไฟล์)
ซึ่งต้องได้ราคาที่ผู้อนุมัติยอมรับ ไม่ใช่ราคาที่ master เคยให้

## Global Constraints

- commit message เป็นภาษาอังกฤษเสมอ คอมเมนต์ในโค้ดเขียนไทยได้
- ห้าม `git add -A` — wms-web มีไฟล์ untracked `shi-7-step-list.txt` และ `.env.development` ที่แก้ค้างไว้ ห้ามหลุดเข้า commit
- **ห้ามรัน migration เอง** เจ้าของเป็นคนรันบน UAT
- ห้ามยิง API หรือต่อ DB ของ UAT
- ห้ามแตะ `internal/services/price-service/**` (ทีมอื่นเป็นเจ้าของ)
- erp-core: `go build ./...` + `go test ./internal/services/sale-service/` ผ่าน (`go test ./...` มี 3 suite ที่ต้องใช้ Docker ซึ่งพังอยู่ก่อนแล้ว)
- wms-web: `./node_modules/.bin/vitest run src/stores/saleOrder src/utils/helper` เขียว และ `npx vue-tsc --noEmit` ไม่เกิน baseline **2978** errors (หมายเหตุ: `npx vitest` ใช้ไม่ได้ในเครื่องนี้ ต้องเรียก binary ตรง)

---

## Task 1: erp-core เพิ่มคอลัมน์และฟิลด์

**Files:**
- Create: `migrations/2026-09-10-sale-item-convert-price.sql`
- Modify: `internal/models/sale.go` (ต่อจากบรรทัด 99 ที่ประกาศ `OldPriceListUnit`)

**Interfaces:**
- Produces: คอลัมน์ `sale_item.convert_price_list_unit numeric(20,8) DEFAULT 0` และฟิลด์ `ConvertPriceListUnit float64` ที่มี json tag `convert_price_list_unit`

- [ ] **Step 1: เขียน migration**

ไฟล์ `migrations/2026-09-10-sale-item-convert-price.sql` — เขียนคอมเมนต์นำแบบเดียวกับ `migrations/2026-09-09-unit-pcs-label.sql` (คอมเมนต์ไทย อธิบายว่าทำไม) แล้วตามด้วย

```sql
ALTER TABLE sale_item
    ADD COLUMN IF NOT EXISTS convert_price_list_unit numeric(20,8) DEFAULT 0;
```

คอมเมนต์ต้องบอกอย่างน้อย 3 เรื่อง: ช่องนี้เก็บราคาที่ master ให้ตอน convert, เป็นข้อมูลอ้างอิงล้วนไม่มีใครคิดเงินจากมัน (ปลายทางยังอ่าน `price_list_unit` เหมือนเดิม), และใบเก่าจะได้ 0 เพราะไม่เคยเก็บค่านี้

- [ ] **Step 2: เพิ่มฟิลด์ใน model**

`internal/models/sale.go` ต่อจาก `OldPriceListUnit` — จัดคอลัมน์ให้ตรงกับบรรทัดข้างเคียง

```go
	ConvertPriceListUnit           float64        `json:"convert_price_list_unit"`
```

- [ ] **Step 3: build + test**

```bash
cd /c/work-prime/erp-core && go build ./... && go test ./internal/services/sale-service/
```

Expected: PASS (ขั้นนี้เพิ่ม field ล้วน ไม่มีเทสใหม่)

- [ ] **Step 4: Commit**

```bash
cd /c/work-prime/erp-core
git add migrations/2026-09-10-sale-item-convert-price.sql internal/models/sale.go docs/superpowers/plans/2026-09-10-so-convert-price-field.md
git commit -m "feat(sale): keep the convert-time price list in its own column"
```

---

## Task 2: wms-web ส่งค่าและแสดงในคอลัมน์ที่สอง

**Files:**
- Modify: `src/stores/saleOrder/sale-order-create.store.ts` (`mapSaleOrderItem` ~บรรทัด 300-315)
- Modify: `src/types/saleOrder/sale-order-create.type.ts` (`CreateSaleOrderItemRequest`)
- Modify: `src/types/saleOrder/sale-order-list.type.ts` (ข้าง `oldPriceListUnit`)
- Modify: `src/components/saleOrder/saleOrderCreate/index.vue` (จุด map แถว ข้างบรรทัด `oldPriceList:`)
- Modify: `src/components/saleOrder/saleOrderCreate/saleOrderDetail/saleOrderColumns.ts`
- Modify: `src/components/saleOrder/saleOrderCreate/saleOrderDetail/index.vue` (template cell)
- Test: `src/stores/saleOrder/sale-order-create.store.spec.ts`

**Interfaces:**
- Consumes: `resolvePriceListByUom` และ `resolveConvertPriceList` ที่ `mapSaleOrderItem` เรียกอยู่แล้ว
- Produces: payload key `convert_price_list_unit`, ฟิลด์แถวตาราง `convertPriceList`, คอลัมน์ที่สอง bind `convertPriceList`

- [ ] **Step 1: เขียนเทสที่ยังไม่ผ่าน**

เพิ่มเทสใน `sale-order-create.store.spec.ts` โดยใช้ fixture/ helper แบบเดียวกับเทส `mapSaleOrderItem` ที่มีอยู่แล้วในไฟล์ (อย่าสร้าง harness ใหม่) ยืนยัน 2 อย่าง:
- `isPassPrice = true` → `convert_price_list_unit` = ราคาจาก `adjustedPriceMap` (ราคาที่ดึงจาก master) ขณะที่ `price_list_unit` = ราคาจาก quotation
- `isPassPrice = false` → `convert_price_list_unit` = ราคาจาก master **เท่ากับ** `price_list_unit`

ทั้งสองกรณี `old_price_list_unit` ต้องยังเป็นราคาจาก quotation

- [ ] **Step 2: รันเทสให้เห็นว่าพัง**

```bash
cd /c/work-prime/wms-web && ./node_modules/.bin/vitest run src/stores/saleOrder
```

Expected: FAIL — `convert_price_list_unit` เป็น undefined

- [ ] **Step 3: ส่งค่าใน `mapSaleOrderItem`**

ตอนนี้ราคาที่ดึงจาก master ถูกคำนวณ inline อยู่ในอาร์กิวเมนต์ของ `resolveConvertPriceList` ให้ดึงออกมาเป็นตัวแปรก่อนแล้วใช้ซ้ำสองที่:

```ts
          const refreshedPriceList = resolvePriceListByUom(
            element.unitUom,
            adjustedPrice,
            element.priceListUnit || 0,
          );
```

แล้วใน object ที่ return

```ts
            price_list_unit: resolveConvertPriceList(
              isPassPrice,
              element.priceListUnit || 0,
              refreshedPriceList,
            ),
            old_price_list_unit: element.priceListUnit || 0,
            // ราคาที่ master ให้ตอน convert เก็บไว้อ้างอิงอย่างเดียว ตอน approve ไม่ทับช่องนี้
            // จึงยังเทียบกับราคาที่เสนอลูกค้าได้หลังอนุมัติ
            convert_price_list_unit: refreshedPriceList,
```

เพิ่ม `convert_price_list_unit: number;` ใน `CreateSaleOrderItemRequest`

- [ ] **Step 4: รับค่ากลับมาแสดง**

- `sale-order-list.type.ts`: เพิ่ม `convertPriceListUnit: number;` ข้าง `oldPriceListUnit`
- `saleOrderCreate/index.vue` จุด map แถว: เพิ่ม `convertPriceList: item.convertPriceListUnit ?? 0,` ข้าง `oldPriceList`
- `saleOrderColumns.ts`: คอลัมน์ที่สอง (`Price list at current date (THB)`) เปลี่ยน `dataIndex`/`key` จาก `priceList` เป็น `convertPriceList` — คอลัมน์แรก (`oldPriceList`) และ `else` branch คอลัมน์เดียว (`priceList`) คงเดิมทั้งคู่
- `saleOrderDetail/index.vue`: เพิ่ม template cell ของ key ใหม่ ใช้ `formatCurrency` แบบเดียวกับ cell ของ `oldPriceList`

- [ ] **Step 5: รันเทสและ type check**

```bash
cd /c/work-prime/wms-web && ./node_modules/.bin/vitest run src/stores/saleOrder src/utils/helper && npx vue-tsc --noEmit 2>&1 | tail -3
```

Expected: เทสผ่าน, vue-tsc ไม่เกิน baseline 2978

- [ ] **Step 6: Commit** (เพิ่มไฟล์ทีละชื่อ ห้าม `git add -A`)

```bash
cd /c/work-prime/wms-web
git add src/stores/saleOrder/sale-order-create.store.ts src/stores/saleOrder/sale-order-create.store.spec.ts src/types/saleOrder/sale-order-create.type.ts src/types/saleOrder/sale-order-list.type.ts src/components/saleOrder/saleOrderCreate/index.vue src/components/saleOrder/saleOrderCreate/saleOrderDetail/saleOrderColumns.ts src/components/saleOrder/saleOrderCreate/saleOrderDetail/index.vue
git commit -m "feat(sale-order): compare the quoted price against the convert-time price after approval"
```

---

## Task 3: ตรวจ flow หลังเพิ่มฟิลด์

**Files:** ไม่มีการแก้โค้ด

- [ ] **Step 1: build + test ทั้งสอง repo**

```bash
cd /c/work-prime/erp-core && go build ./... && go test ./internal/services/sale-service/
cd /c/work-prime/wms-web && ./node_modules/.bin/vitest run && npx vue-tsc --noEmit 2>&1 | tail -3
```

- [ ] **Step 2: ตอบให้ได้ 3 ข้อ โดยชี้ไฟล์+บรรทัด**

1. convert ทั้งกรณีผ่านและไม่ผ่าน → `convert_price_list_unit` = ราคาที่ดึงจาก master
2. approve → `price_list_unit` เปลี่ยนเป็นราคาจาก quotation แต่ `convert_price_list_unit` **ไม่ถูกแตะ** (ยืนยันจาก `update-status-approve-sale.go`)
3. จอใบ `pass_price_list='N'` หลัง approve → คอลัมน์ซ้าย = ราคาจาก quotation, คอลัมน์ขวา = ราคาตอน convert, **คนละค่า**

- [ ] **Step 3: เขียนสิ่งที่เจ้าของต้องทำลงในรายงาน**

- รัน `migrations/2026-09-10-sale-item-convert-price.sql` บน UAT **ก่อน** deploy erp-core
- ใบเก่าจะได้ 0 ในคอลัมน์ที่สอง
