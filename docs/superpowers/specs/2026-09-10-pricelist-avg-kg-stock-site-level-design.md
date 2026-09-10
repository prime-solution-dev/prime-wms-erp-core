# Design — `Avg. kg stock` ระดับ site และการแก้สูตรราคาที่ป้อนค่าผิดประเภท

วันที่: 2026-09-10
สถานะ: รออนุมัติ
Branch: `feat/pricelist-avg-kg-stock-site-level` (แตกจาก `origin/Develop`)
Repo ที่กระทบ: `prime-wms-erp-core`, `prime-wms-warehouse-core`

---

## 1. ปัญหา

### นิยามทางธุรกิจที่ยืนยันแล้ว

`Avg. kg stock` = **น้ำหนักรวมของ product นั้นใน site นั้น ÷ จำนวนชิ้นรวมใน site นั้น**

```
avg_kg_stock = SUM(total_weight) / SUM(qty)   GROUP BY company, site, product
```

เป็น **1 ค่าต่อ product ต่อ site** รวมทุก batch และทุก location เข้าด้วยกัน
หน่วย: kg ต่อชิ้น

เมื่อไม่มีสต็อก: **แสดง `0`** ส่วนฝั่งสูตรคง fallback `1.0` ไว้ตามเดิม (ตัดสินใจแล้ว ไม่อยู่ใน scope)

### F1 — ค่าที่ใช้เป็นของ batch เดียวที่ถูกสุ่มเลือก (Critical)

`prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-by-key.go:303`
รวมยอดโดยใส่ `BatchNo` ไว้ใน key:

```go
key := fmt.Sprintf("%s|%s|%s|%s", invW.CompanyCode, invW.SiteCode, invW.ProductCode, invW.BatchNo)
```

บรรทัด 345-348 แปลง map เป็น slice โดยไม่ sort ซึ่ง Go สุ่มลำดับการ iterate map:

```go
result := make([]InventoryWeightDetail, 0, len(invMap))
for _, v := range invMap {
    result = append(result, *v)
}
```

ฝั่งผู้ใช้ค่าหยิบ `[0]` ซึ่งคือ batch แบบสุ่ม:

| ไฟล์ | บรรทัด | ผลกระทบ |
|---|---|---|
| `erp-core/internal/services/price-service/update-latest-pricelist-subgroup.go` | 240 | **บันทึกราคาลง DB** |
| `erp-core/internal/services/price-service/get-calculated-pricelist-subgroup.go` | 226 | ราคาที่แสดงก่อนบันทึก |
| `erp-core/internal/services/price-service/get-price-export-table.go` | 637 | ค่าใน Excel |

`get-price-detail.go:306-327` ไม่โดนบั๊กสุ่มโดยตรงเพราะ expand 1 row ต่อ 1 batch entry แต่เกิดอาการเดียวกันปลายทางที่
`patterns/shared.go:1267` ซึ่งเขียน `row["avg_kg_stock"]` ต่อ subgroup ขณะที่ `buildDynamicRows` ยุบหลาย subgroup
เข้า row เดียวกัน (group ตาม `PG06`) ทำให้ subgroup ที่ประมวลผลทีหลังเขียนทับตัวก่อน และลำดับนั้นสุ่มเช่นกัน

#### หลักฐาน (ข้อมูลจริง `18.138.69.85` / `prime_wms_warehouse`)

`RBB610SR24TEF` @ site `TMI_WH` company `09dcb573-c95d-4dc1-9a1e-c478cdbfc1c3`

ค่าที่ถูกต้องตามนิยาม: `30,434.1126 / 10,436 = 2.9969`

ค่าที่โค้ดอาจได้ ขึ้นกับว่า Go สุ่มหยิบ batch ไหน:

| batch_no | qty | total_weight | avg ของ batch นั้น |
|---|---|---|---|
| `b1` | 1 | 0.0000 | **0.0000** → ตกไป fallback `1.0` |
| `12341241` | 105 | 233.1000 | 2.2200 |
| `100-001` | 103 | 233.1000 | 2.2631 |
| *(ว่าง)* | 10,183 | 30,434.1126 | 2.9887 |
| `AS` | 31 | 233.4300 | 7.5300 |
| `TEST_THEO` | 12 | 110.0000 | 9.1667 |
| `b2` | 1 | 32.0000 | **32.0000** |

`avg_kg_stock` แกว่งได้ตั้งแต่ `1.0` ถึง `32.0` บนสูตร `(base_price+2.1)*avg_kg_stock*1.02`
คือ **ราคาต่างกันได้ 32 เท่า** และเปลี่ยนทุกครั้งที่กดคำนวณ
สังเกตว่า batch ที่มีของจริง 10,183 ชิ้น (97% ของสต็อก) มีโอกาสถูกเลือกเท่ากับ batch `b2` ที่มีของ 1 ชิ้น

### F2 — ตัวแปร `pcs` / `kg` ในสูตรถูกป้อนจำนวนสต็อกแทนราคา (Critical)

`update-latest-pricelist-subgroup.go:242-243` (และ `get-calculated-pricelist-subgroup.go:228-229`):

```go
pcs = inventoryWeight[0].TotalQty      // จำนวนชิ้นในสต็อก
kg  = inventoryWeight[0].TotalWeight   // น้ำหนักรวมในสต็อก
```

แต่สูตรใน `price_list_formulas` ต้องการ **ราคา** ไม่ใช่จำนวน หลักฐานคือคู่สูตรผกผันที่มีอยู่จริงใน DB:

| name | expression |
|---|---|
| `Pcs = [kg] x [Weight Spec]` | `kg*weight_spec` |
| `kg = [Pcs] / [Weight Spec]` | `pcs*weight_spec` |

`weight_spec` มีหน่วย kg ต่อชิ้น ดังนั้น `ราคาต่อชิ้น = ราคาต่อกิโล × kg ต่อชิ้น` จึงสมเหตุสมผล
และผลลัพธ์ของสูตรเหล่านี้ถูก assign ลง `total_net_price_unit` / `total_net_price_weight`
จึงไม่มีทางตีความ `pcs` และ `kg` เป็นจำนวนสต็อกได้

### F3 — expression ใน DB ไม่ตรงกับชื่อสูตร

`kg = [Pcs] / [Weight Spec]` มี expression เป็น `pcs*weight_spec` ทั้งที่ชื่อระบุว่าเป็นการหาร
ผิดทั้งใน `prime_erp.price_list_formulas` และใน seed `internal/scripts/price_list_formulas/price-list-formulas.json`

### ขนาดผลกระทบ

จาก `prime_erp.price_list_subgroup_formulas_map`:

| expression | subgroups | F1 | F2 |
|---|---|---|---|
| `base_price+extra` | 1,187 | | |
| `kg*avg_kg_stock` | 712 | ✓ | ✓ |
| `kg*weight_spec` | 475 | | ✓ |
| `(base_price+2.1)*avg_kg_stock*1.02` | 262 | ✓ | |
| `pcs = input` | 48 | | |
| `pcs/avg_kg_stock` | 16 | ✓ | ✓ |

**F1 กระทบ 990 subgroup · F2 กระทบ 1,203 subgroup**
(`(base_price+1.4)*avg_kg_stock*1.02` ไม่ถูกผูกกับ subgroup ใดในขณะนี้)

---

## 2. ข้อจำกัดที่ต้องเคารพ

### pattern ที่ใช้ avg ระดับ batch ต้องไม่พัง

3 pattern แสดง `batch_no` เป็นคอลัมน์ โดย `batch_no` ถูกใช้สื่อความหมายอื่น:

| pattern config | คอลัมน์ `batch_no` | คอลัมน์ avg |
|---|---|---|
| `GROUP_1_ITEM_7_PATTERN.json` | "โรงงาน" | `avg_weight_ton` → "Avg.kg stock (Tons)" |
| `GROUP_1_ITEM_8_PATTERN.json` | "Ship No." | `avg_weight` → "Avg.kg stock" |
| `GROUP_1_ITEM_22_PATTERN.json` | "โรงงาน" | `avg_weight_ton` → "Avg.kg stock (Tons)" |

ทั้งสาม pattern อ่านค่าผ่าน `dataMapping: "avg_weight"` เหมือนกัน (`avg_weight_ton` เป็นแค่ชื่อ field
ปลายทาง ไม่ใช่ค่าคนละตัว) จึงต้องคงค่า avg ของ batch ตัวเองไว้
`GROUP_1_ITEM_1-6` และ `9-20` ไม่มี `batch_no` จึงต้องเป็นค่าระดับ site

**ข้อสังเกตที่พบระหว่างทาง — ค้างรอคำตอบจากธุรกิจ ไม่แก้ในงานนี้:**
คอลัมน์ `avg_weight_ton` ของ `GROUP_1_ITEM_7` และ `GROUP_1_ITEM_22` มี headerName ว่า
"Avg.kg stock (Tons)" แต่ `dataMapping` ชี้ไป `avg_weight` ซึ่งมีหน่วย kg

สิ่งที่ตรวจแล้ว:
- ฝั่ง Go ไม่มีการอ้างถึง `avg_weight_ton` เลย (`grep` ใน `internal/services/price-service/` ไม่พบผลลัพธ์)
  คอลัมน์นี้ได้ค่ามาทาง `dataMapping` เท่านั้น
- ฝั่ง web `avg_weight_ton` ปรากฏแห่งเดียวคือ `utils/helper/priceListNumberFormat.ts:19`
  ซึ่งเป็นลิสต์ field ที่ต้อง format ด้วยตัวคั่นหลักพัน ไม่ใช่การแปลงหน่วย
- ค่าจำนวนตันที่ผู้ใช้เห็นบนหน้าจอมาจาก UDF field ชื่อ `ton` ซึ่งผู้ใช้กรอกเอง
  (อยู่ใน `UDF_FIELDS` และ editable suffixes เช่น `PriceListGROUP1ITEM7Detail.vue:1525`)
  เป็นค่าคนละตัวกับ `avg_weight` ไม่ได้คำนวณจากกัน

สรุปสถานะ: ไม่พบการแปลง kg เป็น ton ที่ใดในระบบ ค่าใต้หัวข้อ "(Tons)" จึงน่าจะเป็น kg
แต่เนื่องจากมี field `ton` ที่ผู้ใช้กรอกอยู่แยกอีกตัว ยังสรุปเจตนาไม่ได้ รอคำตอบจากธุรกิจ

### ขอบเขตของ endpoint ปลอดภัย

`GetInventoryWeightByKey` มีผู้เรียกเพียง 4 จุด ทั้งหมดอยู่ใน price-service ของ `erp-core`:

```
get-price-detail.go:279 · get-price-export-table.go:150
update-latest-pricelist-subgroup.go:199 · get-calculated-pricelist-subgroup.go:181
```

ระบบที่ใช้ค่าเฉลี่ยระดับ batch จริง (Transfer, Production Fabrication, Picking, Packing) ใช้ endpoint
คนละตัวคือ `get-inventory-weight` ซึ่งคืน `avgProduct` / `avgBatch` / `avgSerial` เช่น
`prime-wms-web/src/views/production/fabrication/index.vue:720-722`:

```javascript
const avgWeight = avgSerial || avgBatch || avgProduct || 0;
```

การแก้ `get-inventory-weight-by-key.go` จึงไม่กระทบระบบเหล่านั้น

### กฎโปรเจกต์

- ห้าม merge หรือแก้ `Develop` ตรง ๆ · ห้ามแตะ branch `Crossmax-uat`, `shi-sit`, `Pacifica-uat`, `Pacifica-main`, `Thaimetal-uat`, `shi-main`
- unit test + integration test coverage ≥ 80%
- ห้ามรัน `gofmt -w` ทั้งไฟล์ (repo มี pre-existing violation จะทำให้ diff บวม)
- polyrepo: ต้อง `cd` เข้า service ก่อนรันคำสั่ง go

---

## 3. แนวทางที่พิจารณา

### ทางเลือก A — ตัด `BatchNo` ออกจาก key

รวมยอดเป็น `company|site|product` ตรงตามนิยามที่สุด และ response เล็กลง
**ตกไป** เพราะทำให้ `GROUP_1_ITEM_7/8/22` สูญเสีย row ต่อ batch ที่ต้องใช้แสดง "โรงงาน" / "Ship No."

### ทางเลือก B — เติมค่าระดับ site ลงทุก entry โดยคง entry ต่อ batch ไว้ (เลือกทางนี้)

คง key เดิมเพื่อให้ per-batch pattern ใช้ได้ แล้วเติม field ใหม่ที่เป็นค่าระดับ site ลงทุก entry
ผู้ใช้ค่าแต่ละรายเลือกเอาค่าที่ตรงกับความหมายของตัวเอง
ข้อดี: ไม่มี consumer ไหนเสียข้อมูล · แยกสองความหมายออกจากกันชัดเจนในระดับ type · ค่าระดับ site
เหมือนกันทุก entry ทำให้การหยิบ `[0]` ไม่เป็นปัญหาอีกแม้ยังมีโค้ดที่หยิบ `[0]` อยู่

### ทางเลือก C — ให้ `erp-core` รวมยอดเองฝั่ง client

แก้ repo เดียว ไม่ต้องแตะ `warehouse-core`
**ตกไป** เพราะต้องเขียน logic รวมยอดซ้ำใน 4 จุด และ `warehouse-core` มี `safeAverage` กับ `AVGProduct`
ที่ทำงานนี้อยู่แล้ว การคำนวณควรอยู่ที่เจ้าของข้อมูล

---

## 4. การเปลี่ยนแปลงที่จะทำ

### A. `prime-wms-warehouse-core` — `get-inventory-weight-by-key.go`

เพิ่ม field `AvgProduct` ใน `InventoryWeightDetail` (json tag `avg_product`)

ใน `getAggregatedInventoryWeights` เพิ่มการรวมยอดชั้นที่สองด้วย key `company|site|product`
(ไม่มี batch) แล้วเติมค่าเดียวกันนั้นลงทุก entry ที่อยู่ใน product/site เดียวกัน
ใช้ helper `safeAverage` ที่มีอยู่แล้วใน `get-inventory-weight.go:64-75` แทนการเขียน guard ใหม่
เพื่อให้ได้การป้องกัน `Inf` / `NaN` ด้วย

`sort` slice ผลลัพธ์ด้วย `Key` ก่อน return เพื่อให้ลำดับ row นิ่ง แก้อาการ row สลับตำแหน่งทุกครั้งที่เรียก
และทำให้ผลลัพธ์ทดสอบได้

ไม่แตะ `AvgWeight` เดิม ยังเป็นค่าระดับ batch ตามที่ per-batch pattern ต้องใช้

### B. `prime-wms-erp-core` — ฝั่งคำนวณราคา

`update-latest-pricelist-subgroup.go:232-243` และ `get-calculated-pricelist-subgroup.go:218-229`
เปลี่ยนแหล่งของ `avgKgStock` จาก `inventoryWeight[0].AvgWeight` เป็น `inventoryWeight[0].AvgProduct`
คง fallback `1.0` เมื่อค่าเป็น 0 ตามที่ตัดสินใจไว้

`get-price-export-table.go` ฟังก์ชัน `applyInventoryFieldsToRow` บรรทัด 637 เปลี่ยนมาใช้ `AvgProduct`
สำหรับคอลัมน์ `avg_weight`

### C. `prime-wms-erp-core` — `patterns/shared.go`

แยกฟังก์ชันอ่านค่าออกเป็นสองตัวให้ชื่อบอกความหมายตรง ๆ:

- `getAvgKgStockFromInventory(sg)` คืน `roundTo2(InventoryWeight[0].AvgProduct)` — ค่าระดับ site
  ใช้กับ pattern ทั่วไป
- `getAvgKgStockPerBatchFromInventory(sg)` คืน `roundTo2(InventoryWeight[0].AvgWeight)` — ค่าระดับ batch
  ใช้กับ `GROUP_1_ITEM_7`, `GROUP_1_ITEM_8`, `GROUP_1_ITEM_22`

`getAvgProductFromInventory` ตัวเดิมถูกแทนที่ทั้งหมด (ชื่อเดิมสื่อผิดอยู่แล้วเพราะอ่าน `AvgWeight`)
จุดเรียกที่ต้องแก้: บรรทัด 1267, 1399, 1401, 1407, 1715, 1845, 2102, 2104, 2112, 2370, 2372
โดยเลือกฟังก์ชันตาม pattern ที่ code path นั้นรองรับ

ทั้งสองฟังก์ชันคืน `0` เมื่อไม่มี inventory ตามการตัดสินใจเรื่องการแสดงผล

### D. `prime-wms-erp-core` — แก้ประเภทของ `pcs` / `kg` ในสูตร

`update-latest-pricelist-subgroup.go:242-243` และ `get-calculated-pricelist-subgroup.go:228-229`
เปลี่ยนจากจำนวนสต็อกเป็นราคาของกลุ่ม:

```go
pcs = subGroup.PriceListGroup.PriceUnit      // ราคาต่อชิ้น
kg  = subGroup.PriceListGroup.PriceWeight    // ราคาต่อกิโล
```

ค่าทั้งสองอ่านได้นอกบล็อก `if ... len(inventoryWeight) > 0` เพราะเป็นข้อมูลราคาของกลุ่ม ไม่ผูกกับสต็อก
(เหตุผลเดียวกับที่ `weight_spec` ถูกย้ายออกมานอกบล็อกในงาน Weight-spec รอบก่อน)

### E. `prime-wms-erp-core` — แก้ expression ที่ผิด

migration SQL ใหม่ใต้ `migrations/` แก้ expression ของสูตร `kg = [Pcs] / [Weight Spec]`
จาก `pcs*weight_spec` เป็น `pcs/weight_spec` และแก้ให้ตรงกันใน
`internal/scripts/price_list_formulas/price-list-formulas.json`

migration ต้องเป็น idempotent และระบุสูตรด้วย `formula_code` ไม่ใช่ `name`

### F. ของแถมที่อยู่บนเส้นทางเดียวกัน

`get-price-detail.go:317-326` ตัดเงื่อนไข `if inv.X > 0` ออก เขียนค่าตรง ๆ
ปัจจุบันเมื่อ batch ใหม่มีค่า 0 field จะคงค่าที่ค้างจาก subgroup ต้นแบบไว้
(เป็นรูปแบบเดียวกับบั๊ก Weight-spec รอบก่อน)

ลบการเขียน `expandedSG.InventoryWeight[0].AvgBatch = inv.AvgWeight` ที่บรรทัด 324 ออก
เป็น dead write — ไม่มีผู้อ่าน `AvgBatch` ในโปรเจกต์เลย ผู้อ่านที่เหลือคือฝั่ง web ซึ่งรับค่าจาก
endpoint `get-inventory-weight` คนละตัว

รวมพฤติกรรมเมื่อไม่มีสต็อกให้เป็น `0` ทั้ง 3 row builder ปัจจุบันต่างกัน 3 แบบ:
grid คืน `0` · `applyInventoryFieldsToRow` ไม่ set key เลยเพราะ early return · `build-pricelist-detail-tab.go`
ใส่ string ว่าง

---

## 5. สิ่งที่ไม่ทำในงานนี้

- ไม่แก้ fallback `1.0` ในสูตร — ตัดสินใจคงไว้ ทั้งที่มีความเสี่ยงว่าสินค้าไม่มีสต็อกจะได้ราคาเสมือน
  หนัก 1 kg ต่อชิ้นโดยไม่มีสัญญาณเตือน บันทึกไว้เป็น known limitation
- ไม่แก้ความต่างระหว่างค่าที่แสดง (ปัด 2 ตำแหน่งที่ `shared.go`) กับค่าที่สูตรใช้ (ค่าดิบ) —
  ต่างกันไม่เกิน 0.005 kg ต่อชิ้น ยังไม่มีหลักฐานว่ากระทบธุรกิจ
- ไม่ย้าย price-service ไปใช้ endpoint `get-inventory-weight` แทน `get-inventory-weight-by-key`
  แม้ endpoint นั้นจะมี `AVGProduct` ที่ตรงนิยามอยู่แล้ว เพราะ contract ต่างกันมาก
  (`key_value` / `group_code_keys`) จะกลายเป็นการ rewrite ทั้ง 4 flow
- ไม่แก้ข้อมูลราคาที่คำนวณผิดไปแล้วใน DB — ต้องยืนยันกับธุรกิจก่อนว่าจะ recalculate ย้อนหลังหรือไม่

---

## 6. แผนการทดสอบ

### `prime-wms-warehouse-core`

unit test ของ `getAggregatedInventoryWeights`:

- product เดียว site เดียว หลาย batch → ทุก entry มี `AvgProduct` เท่ากัน และเท่ากับ
  `SUM(weight)/SUM(qty)` ขณะที่ `AvgWeight` ของแต่ละ entry ยังเป็นค่าของ batch ตัวเอง
- ใช้ข้อมูลจาก repro จริงเป็น fixture: 7 batch ของ `RBB610SR24TEF` → `AvgProduct` ต้องเท่ากับ
  `2.9969` ทุก entry
- product เดียวกันต่าง site → `AvgProduct` แยกกันตาม site ไม่ปนกัน
- `qty` รวมเป็น 0 → `AvgProduct` เป็น 0 ไม่ panic ไม่เป็น `Inf` หรือ `NaN`
- batch ที่มี `qty` 0 ปนอยู่กับ batch ปกติ → ไม่ทำให้ค่ารวมเพี้ยน
- **test ลำดับนิ่ง**: เรียกฟังก์ชันด้วย input เดียวกัน 50 ครั้ง ต้องได้ลำดับ slice เหมือนกันทุกครั้ง
  (test นี้จะ fail บนโค้ดปัจจุบัน คือตัวพิสูจน์ F1)

### `prime-wms-erp-core`

unit test:

- `getAvgKgStockFromInventory` กับ `getAvgKgStockPerBatchFromInventory` คืนค่าจาก field ที่ถูกต้อง
  และคืน `0` เมื่อ `InventoryWeight` ว่าง
- `CalculatePrice` กับทั้ง 4 สูตรที่ใช้ `avg_kg_stock` โดยเทียบกับค่าที่คำนวณมือ
- `CalculatePrice` กับสูตร `pcs/weight_spec` ที่แก้แล้ว ต้องเป็นผกผันของ `kg*weight_spec`
- `avgKgStock` fallback เป็น `1.0` เมื่อ `AvgProduct` เป็น 0
- env ที่ส่งเข้า expression engine มี `pcs` และ `kg` เป็น `PriceUnit` / `PriceWeight`
  ไม่ใช่ `TotalQty` / `TotalWeight`

integration test (testcontainers ตามรูปแบบ `make test-integration` ที่มีอยู่):

- seed product ที่มี 7 batch ตาม repro แล้วเรียก flow คำนวณราคา 20 ครั้ง → ราคาต้องเท่ากันทุกครั้ง
  (test นี้จะ fail บนโค้ดปัจจุบัน)
- pattern `GROUP_1_ITEM_8` → คอลัมน์ `avg_weight` ของแต่ละ row ต้องเป็นค่าของ batch ใน row นั้น
- pattern `GROUP_1_ITEM_3` → คอลัมน์ `avg_weight` ต้องเป็นค่าระดับ site เท่ากันทุก row
- export table และ Pricelist Detail Report ของ product ที่ไม่มีสต็อก → ทั้งคู่ให้ `0`
- migration ของ F3 รันซ้ำได้โดยผลไม่เปลี่ยน

coverage เป้าหมาย ≥ 80% ของไฟล์ที่แก้

### การยืนยันด้วยข้อมูลจริง

หลัง deploy ลง SIT เรียก flow คำนวณราคาของ `RBB610SR24TEF` @ `TMI_WH` ซ้ำ 5 ครั้ง
ราคาต้องเท่ากันทุกครั้ง และค่า `Avg. kg stock` บนหน้าจอต้องเป็น `3.00` (ปัด 2 ตำแหน่งจาก `2.9969`)

---

## 7. ลำดับการส่งมอบ

1. `prime-wms-warehouse-core` — ข้อ A พร้อม unit test แล้วเปิด PR เข้า `Develop`
2. `prime-wms-erp-core` — ข้อ B, C, D, F พร้อม test แล้วเปิด PR เข้า `Develop`
3. `prime-wms-erp-core` — ข้อ E migration แยก commit ให้ revert ได้เดี่ยว ๆ

ข้อ 2 ต้องรอข้อ 1 ขึ้น environment ก่อนเพราะพึ่งพา field `avg_product` จาก response
ระหว่างนั้น `AvgProduct` จะเป็น 0 แล้วตกไป fallback `1.0` ซึ่งเป็นพฤติกรรมเดียวกับกรณีไม่มีสต็อก
จึงไม่ทำให้ระบบล่ม แต่ต้องไม่ปล่อย erp-core ขึ้น production ก่อน warehouse-core
