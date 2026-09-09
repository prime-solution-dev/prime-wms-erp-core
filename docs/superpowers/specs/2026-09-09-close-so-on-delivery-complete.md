# Spec: ปิด SO อัตโนมัติเมื่อส่งของครบ (erp-core)

วันที่: 2026-09-09 · ลูกค้า: TMI (Thaimetal) · ระบบ: erp-core + wms-outbound-service

## ปัญหา

`sale.status` / `sale_item.status` ค้าง `PENDING` ตลอดไป ทั้งที่ฝั่งคลังทำครบทุกด่านแล้ว

เคสตัวอย่างจริง (TMI UAT 18.138.69.85, 2026-09-09):

| เอกสาร | สถานะ |
|---|---|
| `SO202609-0011` (10 PCS / 51.50 kg) | **PENDING** ← ผิด |
| `DBS202609-0012` | COMPLETED |
| `CO2609-00032` + `CO2609-00032-001` | COMPLETED |
| outbound `OB1788930903273867309-0` | COMPLETED, confirm_qty 10/10 |
| GI `GI20260909051617-0` | COMPLETED, qty 7+3 = 10 PCS / 51.55 kg |

ทั้ง DB: `sale` 16 ใบ PENDING หมด, `sale_item` 21 บรรทัด PENDING หมด — ไม่มีใบไหนเคยขึ้น COMPLETED

## สาเหตุ

ผู้เขียนเดียวของ `sale_item.status = COMPLETED` คือ `erp-core POST /sale/UpdateSaleItemStatus`
(`internal/services/sale-service/update-sale-item-status.go` — ปิดหัวใบให้เองเมื่อทุกบรรทัดครบ)

ผู้เรียกมีที่เดียว: `wms-outbound-service POST /so/update-completed-so`
(`internal/services/so-service/update-complete-so.go`)

**ไม่มีอะไรเรียก `/so/update-completed-so` เลย** — grep ทุก repo ใน `C:\work-prime` ไม่เจอ,
`git log --all -S` ทุก repo ไม่เคยมี caller, และไม่มีแถวใน `prime_wms_document.hook_config`
ชี้ไปที่มัน (23 แถว ตรวจครบ) ⇒ feature เขียนไว้แต่ไม่เคยต่อสาย

ส่วน DBS ปิดได้เพราะมี hook อยู่แล้ว: `hook_config` แถว `ORDER / DELIVERY / UPDATE`
→ `POST /erp/delivery/UpdateStatusDelivery` ยิงจาก `wms-order-service ConfirmOrderOutbound`
(`CallHookCompletedDeliverySlot`) แต่ `UpdateStatusDelivery` ไม่แตะตาราง `sale` เลย

## สิ่งที่โค้ดเดิมใน wms-outbound ทำผิด (เหตุผลที่ไม่ port มาใช้)

1. `deliveryCodes` มาจาก `req.Outbound` ของรอบที่เพิ่ง pack เท่านั้น → `calculateGITotalsForSaleItem`
   นับ GI ได้แค่ DBS ใบนั้น ⇒ SO ที่แบ่งส่งหลายใบไม่ปิดตลอดกาล
   (`SO202609-0009` บรรทัด `94e41ee0…` 50 PCS = DBS-0009 30 + DBS-0010 3 + DBS-0011 17)
2. `if err != nil || len(orders) == 0 { ... err.Error() }` → nil deref panic เมื่อ orders ว่างแต่ err เป็น nil
3. เทียบหน่วยด้วยสตริง `"PC"/"Pcs"/"pcs"/"kg"/"KG"` บน `sale_unit` เดียว ทั้งที่กฎจริงต้องดู
   `sale_unit` + `sale_unit_type` คู่กัน (KG_SPEC = คุมด้วยชิ้น)
4. tolerance เป็นกรอบ `>= min && <= max` → ส่งเกินเพดานแล้วใบไม่ปิด
5. ลูปซ้อน 7 ชั้น + คิด GI ใหม่ทุก sale_item + `deliveryMap` ที่สร้างแล้วไม่ใช้ + `fmt.Printf` ~30 จุด/คำขอ
   + เปิด GORM transaction บน `prime_wms_document` ที่ไม่มี query ไหนใช้

## ทางที่เลือก

ย้ายงานปิด SO ไปอยู่ **erp-core** ฝั่งเดียวกับเจ้าของข้อมูล:

- `delivery_booking_item.document_ref_item` = `sale_item.sale_item` (ยืนยันจากข้อมูลจริง) — map ตรง ไม่ต้องเดา
- erp-core อ่าน GI ได้อยู่แล้วผ่าน `GetOrdersDelivery` → `OutboundItemWithGoodsIssue.GoodsIssueItem`
  และมี `foldWmsProgress` (`validate-booking-qty.go:249`) พับยอด GI ต่อบรรทัดอยู่แล้ว
- `TOLERANCE_SO` อยู่ใน `prime_erp.system_config` (`topic_code=SO`, ค่าปัจจุบัน **3**) อ่านตรงจาก DB ตัวเอง
- ได้ตัวเลข "ส่งครบยัง" กับ "จองได้อีกเท่าไร" จากสูตรเดียวกัน (ญาติกับ `ValidateBookingQty`)

## เกณฑ์ปิด (ข้อกำหนดหลัก)

ต่อ 1 `sale_item`:

1. รวม GI จาก **ทุก DBS ของ sale_code นั้น** (`delivery_booking.document_ref = sale_code`, ข้ามใบ `CANCELED`)
2. นับเฉพาะบรรทัดที่ CO ปิดแล้ว (`order_item.status ∈ {COMPLETED, CANCELED, CANCELLED}`)
   **ไม่ดู `outbound_item.status` เลย** และนับบรรทัด GI ทุกบรรทัดที่ตัวมันเองไม่ถูกยกเลิก
   (ข้าม `CANCELED`/`CANCELLED` ไม่สนตัวพิมพ์เล็กใหญ่/ขีดกลาง — สถานะว่างต้องนับ เพราะข้อมูลเก่าเว้นช่องนี้ไว้)

   เหตุผลที่ห้ามดู outbound: hook ที่พามาถึงเส้นนี้ถูกยิงจาก
   `wms-outbound-service update-flow-tracking-packing.go:394` ขณะที่ transaction ซึ่งเขียน
   `outbound_item.status = COMPLETED` (`:279`, tx เปิด `:127` commit ใน defer `:135-147`)
   **ยังไม่ commit** erp-core อ่านสถานะกลับมาทาง HTTP บน session ใหม่จึงเห็นเป็น `PENDING`
   ตลอด ถ้า gate ด้วย outbound ยอดที่ตัดจ่ายจะเป็น 0 ทุกบรรทัดและไม่มี SO ไหนถูกปิดเลย
   ส่วน `order_item.status` เชื่อได้ เพราะถูก commit ใน tx แรกของ `ConfirmOrderOutbound` ก่อน hook ยิง

   ⚠️ กติกานี้อยู่ใน `foldWmsDelivered` (`close-sale-on-delivered.go`) ตัวเดียว
   `foldWmsProgress`/`foldWmsIssued` ที่ `ValidateBookingQty` ใช้ **ต้องคงการ์ด outbound เดิมไว้**
   ตามที่ SA เคาะ 2026-08-31 (ดูคอมเมนต์ `validate-booking-qty.go:13-19`) ห้ามรวมสองตัวเข้าด้วยกัน
3. โหมดเทียบหน่วย:

   | `sale_unit` | `sale_unit_type` | โหมด | เป้า (target) | เทียบกับ |
   |---|---|---|---|---|
   | `KG` | `KG_SPEC` | QTY | `sale_item.qty` (ชิ้น) | GI `qty` |
   | `KG` | อื่น (`KG`, `PC`) | WEIGHT | `sale_item.total_weight` | GI `weight` |
   | อื่น (`PC`) | — | QTY | `sale_item.qty` | GI `qty` |

   ที่มา: `wms-web packingCreate mapUnitCodeByOrder` (KG_SPEC → PCS) + `deliverySlotCreate:401`
   (KG_SPEC ใช้ `weightUnit` คงที่) — ช่อง `KG` + `PC` กฎเดิมไม่ได้ระบุ ตัดสินให้เป็น WEIGHT
   เพราะ `sale_qty` ของแถวนั้นเป็นกิโล (`SO202609-0003` = 100 kg / 34 ชิ้น) — **ต้องให้ SA ยืนยัน**

4. ปิดเมื่อ `issued >= target * (1 - TOLERANCE_SO/100)` — **ไม่มีเพดานบน** (ส่งเกินถือว่าครบ)
5. `target <= 0` → ไม่ปิด (กันหารศูนย์/บรรทัดขยะ)
6. บรรทัดที่ `status` เป็น `COMPLETED` หรือ `CANCELED` อยู่แล้ว ไม่แตะ
7. ทุกบรรทัดของใบปิดครบ (`COMPLETED`/`CANCELED`) → `sale.status = COMPLETED`

## จุดแทรก

`internal/services/delivery-service/update-status-delivery.go` — **หลัง** `tx.Commit()`,
ทำเฉพาะ `req.Status == "COMPLETED"`, **error ต้อง log ห้าม return**

เหตุผลที่ห้าม return: hook นี้ถูกยิงจาก `wms-order-service` ขณะที่ `tx2` ยังไม่ commit
ถ้าคืน error จะ rollback แล้ว **pack confirm ล้มทั้งใบ ทั้งที่สต็อกกับ GI ตัดไปแล้ว**
(คอมเมนต์เตือนไว้ที่ `confirm-order-outbound.go:205-208`)

จังหวะข้อมูลปลอดภัย: GI ถูกสร้างก่อน confirm order เสมอ
(`wms-outbound-service update-flow-tracking-packing.go:377` → `:394`)
และ `order_item.status = COMPLETED` ถูก commit ใน tx แรกก่อน hook ยิง (`confirm-order-outbound.go:117-142`)

## ของที่ต้องลบ (wms-outbound-service)

- `internal/services/so-service/update-complete-so.go` (492 บรรทัด, ไม่เคยรัน)
- route `soRoutes.POST("/so/update-completed-so")` ใน `internal/routes/routes.go:116-120`
- `external/services/sale/update-sale-item-status.go` + `UPDATE_SALE_ITEM_STATUS_ENDPOINT` ใน `config/constants.go:52`

`erp-core POST /sale/UpdateSaleItemStatus` **คงไว้** (ใช้ปิดด้วยมือ/ทดสอบ) และ refactor
ให้ใช้ helper ตัวเดียวกับเส้นอัตโนมัติ

## นอกขอบเขต

- SO ที่ส่งไปบางส่วนแล้วยกเลิก DBS ที่เหลือ (ส่ง 30 / ยกเลิก 20) จะยังค้าง PENDING — ช่องโหว่เดิมของดีไซน์
- `status_payment` / invoice ไม่แตะ
- ไม่เพิ่ม cron ทุกกรณี

## เกณฑ์รับงาน

- `go build ./...` + `go test ./...` ผ่านทั้ง 2 repo
- unit test ครอบ: PC ครบ/ขาด, KG ครบ/ขาด, KG_SPEC, tolerance ขอบ (3%), ส่งเกิน, target=0,
  SO แบ่ง 3 DBS (30+3+17 = ปิด), SO แบ่ง 3 DBS ที่เพิ่งจบใบแรก (30/50 = ไม่ปิด), CO ยัง PENDING = ไม่ปิด
- ยิงจริงบน UAT ต้องขออนุญาตเจ้าของก่อน (เขียน `sale`/`sale_item`)
