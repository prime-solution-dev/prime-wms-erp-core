# Spec: ด่านราคา price list ตอน convert QU→SO และคอลัมน์ราคา 2 ช่วงเวลา

วันที่: 2026-09-10 · ลูกค้า: TMI (Thaimetal) · repo: `wms-web` + `erp-core`
branch ทั้งสอง repo: `fix/so-pricelist-approval-columns`

## กติกาที่ SA เคาะ (2026-09-10)

1. convert QU→SO แล้ว **ราคาผ่าน** → ใช้ price list จาก quotation ไปเลย
2. convert แล้ว **ราคาไม่ผ่าน** → ใบเข้าสถานะรออนุมัติ และหน้าจอโชว์ 2 คอลัมน์
   - `Price list at create date` = ราคาจาก **quotation**
   - `Price list at current date` = ราคา **ตอน convert** (SA ตีความว่าเท่ากับ "at convert date")
3. เมื่อกด **approve** แล้ว ช่อง `Price list (THB)` ต้องเป็นราคาจาก **quotation**
   เพราะถือว่าผู้อนุมัติยอมรับราคาที่ห่างจาก price list master แล้ว
4. ใบที่ convert มาแบบ **ไม่ผ่าน** ต้องเห็น 2 คอลัมน์ต่อไป **ตลอดชีวิตใบ** ถึงจะ approve ไปแล้วก็ตาม

## สถานะปัจจุบัน (ตรวจจากโค้ด + ข้อมูลจริงบน TMI UAT 2026-09-10)

| กติกา | โค้ดตอนนี้ | ตรง |
|---|---|---|
| 1 | `sale-order-create.store.ts:293` เขียนราคาที่ดึงใหม่จาก master เสมอ ไม่ดู `isPass` | ❌ |
| 2 (ค่าเก่า) | คอลัมน์ `at create date` ผูก `priceList` = `sale_item.price_list_unit` = ราคาตอน convert | ❌ |
| 2 (ค่าใหม่) | คอลัมน์ `at current date` ยิง `getPriceList()` **สดทุกครั้งที่เปิดหน้า** (`saleOrderCreate/index.vue:762-772`) | ❌ |
| 3 | `update-status-approve-sale.go` (130 บรรทัด) ยุ่งกับ status อย่างเดียว ไม่แตะราคา | ❌ |
| 4 | เงื่อนไขโชว์ 2 คอลัมน์คือ `statusApprove === 'PROCESS'` (`saleOrderDetail/index.vue:635`) พอ approve แล้วกลายเป็น COMPLETED คอลัมน์หาย | ❌ |

**ตัวการที่ทำให้ไม่มีใบไหนถูกจับได้เลย:** `erp-core/internal/services/sale-service/create-sale.go:134-138`

```go
// ตั้งค่า validation flags เป็นค่าเริ่มต้น (หน้าบ้านได้ validate แล้ว)
tempSale.PassPriceList = "Y"
tempSale.PassPriceExpire = "Y"
tempSale.PassCreditLimit = "Y"
tempSale.PassAtpCheck = "Y"
```

backend เขียนทับเป็น `"Y"` ทุกใบไม่ว่า payload ส่งอะไรมา → ข้อมูลจริง **26/26 ใบเป็น `pass_price_list = Y`** ทั้งที่บางใบราคาห่างจาก master หลายพันล้านเท่า

ฝั่ง FE ก็ส่งค่าผิดอยู่แล้ว: `sale-order-create.store.ts:382` ส่ง `pass_price_list: QuotationListTable.passPriceList` (ค่าจาก quotation) ไม่ใช่ผล `validateSaleOrder` ที่เพิ่งยิงไปตอนกด convert

**ของที่มีอยู่แล้วและใช้ได้เลย**
- `sale_item.old_price_list_unit` เก็บราคาจาก quotation ตอน convert อยู่แล้ว (`store.ts:298`) และ **`GetSale` ส่งออก API แล้ว** (ยืนยันด้วยการยิง API จริง)
- ผลตรวจราคาแยกด่านมีอยู่แล้วที่ `quotationCreate/index.vue:1186` → `isPassPrice` (แยกจาก ATP/credit/expiry ที่รวมเป็น `criticalPassed`)
- backend จัดการ `status_approve` ถูกอยู่แล้ว (`create-sale.go:80-132`: PROCESS/PENDING/COMPLETED ตาม `req.Status` + auto-approval) — ที่ FE ส่ง `is_approved: true, status_approve: 'COMPLETED'` มาใน payload ถูก backend ทับทิ้งอยู่แล้ว ไม่ใช่ปัญหา

## สิ่งที่ต้องแก้ (6 จุด)

### erp-core

1. **`create-sale.go:134-138`** — เลิก hardcode 4 flags ให้ใช้ค่าที่ payload ส่งมา
   ค่าที่ยอมรับคือ `"Y"` / `"N"` เท่านั้น ค่าว่างหรือค่าอื่นให้ถือเป็น `"Y"` (ของเดิมเป็น Y อยู่แล้ว จึงไม่ทำให้ใบเก่าเปลี่ยนพฤติกรรม)
2. **`update-status-approve-sale.go`** — เมื่อ `req.Status == "COMPLETED"` (อนุมัติผ่าน) ให้เขียน
   `sale_item.price_list_unit = sale_item.old_price_list_unit` ของทุกบรรทัดในใบนั้น
   เฉพาะบรรทัดที่ `old_price_list_unit > 0` (0 = ไม่มีราคาจาก quotation ให้ยึด)
   ทำใน transaction เดียวกับการอัปเดตสถานะ และห้ามแตะเมื่อ REJECT/REVIEW

### wms-web

3. **`quotationCreate/index.vue`** — ส่ง `isPassPrice` เข้า `createSaleOrderStore` เพิ่มจาก `allPassed`
   (ตอนนี้ส่งแค่ `allPassed` ซึ่งรวมทุกด่าน แยกไม่ออกว่าตกที่ด่านไหน)
4. **`sale-order-create.store.ts`**
   - `mapSaleOrderRequest` → `pass_price_list: isPassPrice ? 'Y' : 'N'` (เลิกใช้ค่าจาก quotation)
   - `mapSaleOrderItem` → ถ้า `isPassPrice` เป็นจริง `price_list_unit` ใช้ราคาจาก quotation (`element.priceListUnit`)
     ถ้าไม่ผ่าน ใช้ราคาที่ดึงใหม่ตามเดิม (`resolvePriceListByUom(...)`) เพื่อให้ผู้อนุมัติเห็นตัวเทียบ
   - `old_price_list_unit` = ราคาจาก quotation เหมือนเดิมทั้งสองกรณี
5. **`mapSaleOrderUpdateItem` (`store.ts:160`)** — ห้ามเขียน `old_price_list_unit` ด้วย `item.priceList`
   ต้องส่งค่าเดิมที่โหลดมาจาก DB กลับไป (ต้องเพิ่ม `oldPriceListUnit` ใน type + mapping ตอนโหลดหน้า SO ด้วย
   ตอนนี้ `src/types/saleOrder/sale-order-list.type.ts` ยังไม่มี field นี้)
6. **หน้า SO detail**
   - `saleOrderDetail/index.vue:635` → เปลี่ยนตัวตัดสินจาก `statusApprove === 'PROCESS'` เป็น `passPriceList === 'N'`
   - `at create date` → `old_price_list_unit` (ราคาจาก quotation)
   - `at current date` → `price_list_unit` (ราคาตอน convert) และ **เลิกยิง `getPriceList()` สด** ที่ `saleOrderCreate/index.vue:762-772`

## นอกขอบเขต

- **ราคามั่วใน price list master** (`total_net_price_unit` เป็นตัวเลขมหาศาลเพราะ `kg` กับ `weight_spec` ถูกป้อนจาก `TotalWeight` ตัวเดียวกัน แล้วสูตร FM-6 `kg*weight_spec` คูณกันเอง) — **แจ้ง chonlatee เจ้าของ price-service แล้ว เขารับไปแก้เอง ห้ามแตะในงานนี้**
- ใบเก่าที่ค้างค่ามั่ว/`pass_price_list=Y` ผิด — เจ้าของบอกไม่ต้อง backfill เป็นข้อมูลเทส
- ด่าน ATP / credit / expiry — งานนี้แตะเฉพาะด่านราคา

## เกณฑ์รับงาน

- `go build ./...` + `go test ./...` ผ่านใน erp-core (ยกเว้น 3 suite ที่ต้องใช้ Docker ซึ่งพังอยู่ก่อนแล้ว)
- `vue-tsc` ไม่เพิ่ม error จาก baseline, `vitest` ที่เกี่ยวข้องผ่าน
- unit test ครอบ: ราคาผ่าน → เก็บราคาจาก QU + `pass_price_list='Y'`; ราคาไม่ผ่าน → เก็บราคาที่ดึงใหม่ + `'N'`;
  approve แล้วราคาถูกเขียนกลับเป็นของ QU; update SO แล้ว `old_price_list_unit` ไม่เปลี่ยน
- ห้ามยิง API หรือเขียน DB ของ UAT ระหว่างทำงาน
