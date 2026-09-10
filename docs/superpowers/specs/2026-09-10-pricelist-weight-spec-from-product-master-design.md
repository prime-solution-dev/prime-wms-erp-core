# Price List Weight-spec — ดึงจาก product-master (base unit)

วันที่: 2026-09-10
Branch: `feat/pricelist-weight-spec-from-product-master` (erp-core + warehouse-core, แตกจาก Develop)

## Problem

คอลัมน์ `Weight-spec` ในหน้า price list detail แสดงค่าผิด และสูตรคำนวณราคาก็ใช้ค่าผิดคนละแบบกัน

| จุด | ตอนนี้ดึงจาก | ผลจริง |
|---|---|---|
| กริด / export — `patterns/shared.go:532` `getWeightSpecFromInventory()` | `sg.InventoryWeight[0].WeightSpec` | `warehouse-core` ประกาศ field `WeightSpec` ไว้ที่ `internal/services/inventory-service/get-inventory-weight.go:51` แต่**ไม่มีจุดไหน assign ค่าเลย** → ได้ `0` เสมอ |
| สูตรคำนวณ — `get-calculated-pricelist-subgroup.go:224-228`, `update-latest-pricelist-subgroup.go:230-246` | `inventoryWeight[0].TotalWeight` | เป็น "น้ำหนักรวมของสต็อก" ซึ่งเป็นตัวเดียวกับตัวแปร `kg` เป๊ะ ๆ (บรรทัดถัดไป `kg = inventoryWeight[0].TotalWeight`) → สูตรใช้ค่าผิดชนิด |

**ค่าที่ถูกต้อง:** `product_unit.weight` ของ unit ที่ `flag_base = true` จาก product-master

## Goal

1. Weight-spec ทุกจุดดึงจาก base unit ของ product-master
2. ต้องมีค่าแม้สินค้าไม่มีสต็อกในคลัง
3. แสดงผลได้บน web, return กลับทาง API, และใช้ในสูตรคำนวณได้

## Key discovery — logic ที่ต้องใช้มีอยู่แล้ว

`prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-by-key.go:136-179`
ทำ combination matching ของ subgroup key อยู่แล้ว:

1. เรียก product-core `GetProductDetail` → ได้ product ทั้งหมด พร้อม `product_groups` และ `units[]`
2. sort `ProductGroups` ตาม `Seq`, กรองเฉพาะ `ActiveFlg == true`
3. `strings.Join(..., "|")` → เทียบตรง ๆ กับ `group_code_keys` / `group_value_keys` ของ subgroup
4. ได้ `matchedProduct`

**การ match นี้ไม่พึ่งสต็อก** — stock ถูก query ทีหลังที่บรรทัด 184 และ `ProductCode` ถูก return แม้ไม่มีสต็อก (บรรทัด 244, `inventory_weight: []`)

และ `matchedProduct.Units[]` ที่อยู่ในหน่วยความจำตรงนั้นแล้ว มี `Weight float64` + `FlagBase bool` ครบ
(`external/services/product/get-product-detail-master.go:109-110`) → **ไม่ต้องยิง HTTP เพิ่มแม้แต่ call เดียว**

## Architecture

```
product-core (product_unit.weight, flag_base=true)
  └─ GetProductDetail  ← warehouse-core เรียกอยู่แล้ว
       └─ warehouse-core: matchedProduct.Units → หา FlagBase==true → weight_spec
            └─ GetInventoryWeightByKeyResult.weight_spec   (ระดับ result ไม่ใช่ระดับ batch row)
                 └─ erp-core: PriceListSubGroupResponse.WeightSpec
                      ├─ patterns/shared.go getWeightSpecFromInventory()  → กริด/export ทุก pattern
                      ├─ PriceData.WeightSpec → CalculatePrice()
                      └─ API response: weight_spec
```

**ทำไม weight_spec ต้องอยู่ระดับ result ไม่ใช่ใน `InventoryWeight[]`:**
`InventoryWeight[]` เป็น array ต่อ batch — ถ้าไม่มีสต็อกจะเป็น array ว่าง ค่าจะหายไปทันที
ขัดกับ Goal ข้อ 2 จึงต้องวางไว้ระดับเดียวกับ `ProductCode` / `SupplierCode`

## Changes

### 1. warehouse-core (~10 บรรทัด)

`internal/services/inventory-service/get-inventory-weight-by-key.go`

- struct `GetInventoryWeightByKeyResult` (บรรทัด 44-50) เพิ่ม:
  ```go
  WeightSpec float64 `json:"weight_spec"`
  ```
- ในบล็อก `if matchedProduct != nil` (~บรรทัด 237) หา base unit:
  ```go
  weightSpec := 0.0
  for _, u := range matchedProduct.Units {
      if u.FlagBase {
          weightSpec = u.Weight
          break
      }
  }
  ```
  แล้ว set ลง result
- branch `else` (ไม่ match product) → `WeightSpec: 0`

`get-inventory-weight.go:51` — field `WeightSpec` เดิมที่ไม่เคยถูก set: ปล่อยไว้ ไม่แตะ (ตาม surgical rule) แต่จะไม่มีใครอ่านมันอีก

### 2. erp-core

**a. รับค่าจาก warehouse**
`external/warehouse-service/get-inventory-by-product-code.go:31-39` — `InventoryByProductCodeResponse` เพิ่ม:
```go
WeightSpec float64 `json:"weight_spec"`
```

**b. เก็บลง map แล้วส่งต่อ — ทำแบบเดียวกับ `supplierCodeMap` ที่มีอยู่แล้ว**
4 จุดที่เรียก `GetInventoryWeightByKey`:
- `internal/services/price-service/get-price-detail.go:279`
- `internal/services/price-service/get-calculated-pricelist-subgroup.go:179`
- `internal/services/price-service/get-price-export-table.go:150`
- `internal/services/price-service/update-latest-pricelist-subgroup.go:197`

**c. พา subgroup**
`internal/models/pricelist.go:304-339` — `PriceListSubGroupResponse` เพิ่ม:
```go
WeightSpec float64 `json:"weight_spec"`
```

**d. แก้จุดร่วมจุดเดียว**
`internal/services/price-service/patterns/shared.go:530-537`
```go
func getWeightSpecFromInventory(sg models.PriceListSubGroupResponse) float64 {
    return sg.WeightSpec
}
```
จุดนี้ถูกเรียกจาก 8 ที่ (shared.go:1265, 1396, 1404, 1712, 1842, 2099, 2109, 2367) ครอบคลุม pattern config
ทั้ง 22 ไฟล์ที่มีคอลัมน์ `Weight-spec` → แก้ที่เดียว ถูกหมด

**e. สูตรคำนวณ**
- `get-calculated-pricelist-subgroup.go:224-228` — เปลี่ยน `weightSpec = inventoryWeight[0].TotalWeight` → ใช้ค่าจาก map ใหม่
- `update-latest-pricelist-subgroup.go:230-246` — เช่นเดียวกัน
- `kg = inventoryWeight[0].TotalWeight` **คงเดิม** (ถูกอยู่แล้ว)

**f. API response**
- `internal/models/pricelist.go:496-505` — `GetCalculatedPriceListSubGroupItem` เพิ่ม `WeightSpec float64 \`json:"weight_spec"\`` (ตอนนี้ยังไม่มี field นี้)
- `internal/services/price-service/domain/domain.go` — `Price` struct เพิ่ม `WeightSpec float64 \`json:"weight_spec"\``
- `update-latest-pricelist-subgroup.go:430` มี `"weight_spec": priceData.WeightSpec` อยู่แล้ว → จะได้ค่าถูกอัตโนมัติ

### 3. web — ไม่ต้องแก้

`src/views/price-list/PriceListGROUP1ITEM*Detail.vue` (16 ไฟล์) สร้าง column จาก `fixedColumns` / `dataMapping` /
`editableSuffixesFromApi` ที่ backend ส่งมา ไม่ได้ hardcode column
คอลัมน์ `Weight-spec` → `field: "total_weight"` มีอยู่แล้วทุก pattern config
เมื่อ backend เติมค่าถูก ค่าก็ขึ้นเอง

## Fallback

หาค่าไม่ได้ (ไม่ match product / ไม่มี unit `flag_base=true` / weight เป็น 0):

- **แสดงผล (กริด, export, API response): `0`** — ให้ user เห็นชัดว่า master data ขาด
- **สูตรคำนวณ: `1.0`** — กัน weight_spec ไปเป็นตัวคูณ/ตัวหารแล้วราคาเพี้ยนเป็น 0 หรือ division by zero
  (ตรงกับ default เดิมในโค้ด `weightSpec := 1.0`)

จุดแปลงค่าอยู่ที่เดียว: ก่อนใส่ `PriceData.WeightSpec` → `if weightSpec == 0 { weightSpec = 1.0 }`

## ตัดออกจาก scope

**`GetComparePrice`** — ตรวจแล้ว (`internal/services/price-service/get-compare-price.go`, 245 บรรทัด)
ไม่ได้เรียก `GetInventoryWeightByKey` และไม่ได้เรียก `CalculatePrice` เลย
`total_weight` ในนั้นคือน้ำหนักของ order ที่ client ส่งเข้ามา ไม่ใช่ Weight-spec ของสินค้า
→ ไม่มีอะไรให้แก้ ถ้าภายหลังต้องการให้หน้าเทียบราคาใช้ Weight-spec ด้วย ให้เปิดเป็นงานแยก

**ไม่ใช้ `product-core /Product/GetProductDefaultWeight`** แม้ชื่อจะตรงงาน:
`internal/services/product-service/get-product-default-weight.go:34` ประกาศ `Weight int` →
น้ำหนัก 12.5 kg จะกลายเป็น 12 และบรรทัด 116 มีบั๊ก `Ratio: getFloat(row, "weight")` (assign weight ลง Ratio)
ไม่แก้ไฟล์นี้ในงานนี้ (นอก scope) แต่บันทึกไว้เป็นหนี้ทางเทคนิค

## Error handling

- warehouse-core เรียก product-core fail → error เดิม (`return nil, errors.New("error getting product details: ...")`) ไม่เปลี่ยน
- erp-core เรียก warehouse fail → พฤติกรรมเดิม: log warning แล้วไปต่อโดยไม่มี inventory data → weight_spec = 0 → สูตรใช้ 1.0
- ไม่ match product → `WeightSpec: 0` ไม่ error (สอดคล้องกับ `ProductCode: ""` ที่ทำอยู่แล้ว)

## Testing (ต้องได้ coverage ≥ 80% ตาม CLAUDE.md)

**warehouse-core — unit test**
- match product ที่มี unit `flag_base=true` weight 12.5 → `weight_spec == 12.5`
- product ไม่มี unit ไหน `flag_base=true` → `weight_spec == 0`
- product มีหลาย unit → หยิบเฉพาะตัว `flag_base=true` ไม่ใช่ตัวแรก
- **product match ได้แต่ไม่มีสต็อก → `weight_spec` ยังมีค่า, `inventory_weight == []`** (เคสหลักของ Goal ข้อ 2)
- ไม่ match product → `weight_spec == 0`

**erp-core — unit test**
- `getWeightSpecFromInventory()` คืน `sg.WeightSpec` (มีเทสเดิมที่ `patterns/inventory_udf_test.go:15-27` ต้องแก้ให้ตรง)
- fallback: weight_spec 0 → `PriceData.WeightSpec == 1.0`, แต่ row ในกริดยังเป็น 0
- สูตรคำนวณด้วย weight_spec จริง (เช่น 12.5) ได้ผลต่างจากเดิมที่ใช้ TotalWeight
- `get-price-export-table_test.go:496` มี `WeightSpec: 2.5` อยู่แล้ว — ต้องย้ายมาที่ subgroup level

**erp-core — integration test**
- `make test-integration` (testcontainers) — flow `GetPriceDetail` end-to-end: subgroup key → weight_spec ปรากฏใน response
- mock warehouse response ทั้งเคสมีสต็อกและไม่มีสต็อก

**Manual**
- เปิดหน้า price list detail (GROUP_1_ITEM_2) เทียบคอลัมน์ Weight-spec กับ `product_unit.weight` ของ base unit ใน product master

## หมายเหตุ

- warehouse-core remote เป็น HTTPS และ auth fail → branch แตกจาก local `Develop` (commit `c4ee578`, 2026-07-14) ควรตรวจว่า local Develop ตรงกับ remote ก่อน push
- ห้าม merge เข้า Develop ตรง ๆ ตาม CLAUDE.md
