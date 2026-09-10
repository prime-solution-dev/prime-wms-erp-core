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

#### `[kg]` และ `[Pcs]` หมายถึงอะไรกันแน่

ทุก subgroup ผูกสูตรไว้ 2 ตัวเสมอ และคู่ที่ผูกกันจริงคือ:

| คู่สูตร | subgroups |
|---|---|
| `kg = Base price + Extra` + `Pcs = [kg] x [Avg. kg stock]` | 712 |
| `kg = Base price + Extra` + `Pcs = [kg] x [Weight Spec]` | 475 |
| `Pcs = (...) x [Avg kg. stock] x (1+2%)` ผูกซ้ำ 2 ครั้ง | 131 |
| `kg = [Pcs] / [Avg. kg stock]` + `pcs = input` | 16 |
| `pcs = input` ผูกซ้ำ 2 ครั้ง | 16 |

คู่หลัก 1,187 ตัวเป็น pipeline 2 ขั้น: คำนวณราคาต่อกิโลก่อน (`base_price+extra`) แล้วแปลงเป็นราคาต่อชิ้น
`[kg]` ในสูตร pcs จึงหมายถึง **ผลลัพธ์ของสูตร `kg` ในคู่เดียวกัน** คือ `total_net_price_weight`
ไม่ใช่ราคาตั้งของกลุ่ม ยืนยันอีกทางจากคู่ `kg = [Pcs] / [Avg. kg stock]` + `pcs = input`
ซึ่ง `[Pcs]` คือราคาที่ผู้ใช้กรอกแล้ว kg คำนวณจากมัน

หลักฐานสนับสนุนในโค้ด: `update-latest-pricelist-subgroup.go:222-223` init ตัวแปรจากค่าปัจจุบันใน DB
ซึ่งจะไม่มีเหตุผลเลยถ้าสูตรไม่ได้อ่านค่าเหล่านี้

```go
totalNetPriceUnit := subGroup.TotalNetPriceUnit
totalNetPriceWeight := subGroup.TotalNetPriceWeight
```

และถ้าเจตนาคือราคาตั้งของกลุ่ม สูตรย่อมเขียน `base_price` ตรง ๆ อย่างที่
`(base_price+2.1)*avg_kg_stock*1.02` ทำอยู่แล้ว

#### ผลกระทบที่เกิดขึ้นจริงใน DB

จาก 712 subgroup ที่ผูกสูตร `Pcs = [kg] x [Avg. kg stock]`:

| สถานะ `total_net_price_unit` | subgroups |
|---|---|
| เท่ากับ `0` | **694** |
| มากกว่า `0` | 18 (สูงสุด **632,016.00**) |

ราคาต่อกิโลของกลุ่มเหล่านี้อยู่ราว `18.30` ถึง `19.60` ราคาต่อชิ้นที่ควรได้จึงอยู่ระดับหลักสิบ
ค่า `0` เกิดจาก `kg` ถูกป้อน `TotalWeight` ซึ่งเป็น 0 เมื่อไม่มีสต็อก ส่วนค่า 632,016
เกิดจากน้ำหนักสต็อกรวมถูกนำไปคูณ avg โดยตรง

ตัวอย่างรูปธรรม `SG711` @ `TMI_WH` — `price_weight = 19.40` `extra_price_weight = 0.20`
`total_net_price_weight = 19.60` `total_net_price_unit = 0.000000`
สมมติ `avg_kg_stock = 3.0000`:

| การตีความของ `[kg]` | ค่าที่ป้อน | ผลลัพธ์ |
|---|---|---|
| โค้ดปัจจุบัน = `TotalWeight` (ไม่มีสต็อก) | 0 | `0.00` |
| โค้ดปัจจุบัน = `TotalWeight` (มีสต็อก 211,000 kg) | 211,000 | `633,000.00` |
| ราคาตั้งของกลุ่ม `PriceListGroup.PriceWeight` | 19.40 | `58.20` ตก `extra` ไป |
| **ค่า running `total_net_price_weight`** | 19.60 | **`58.80`** |

เลือกค่า running เพราะรวม `extra` ที่สูตร `kg` คำนวณไว้แล้ว

### F3 — expression ใน DB ไม่ตรงกับชื่อสูตร

`kg = [Pcs] / [Weight Spec]` มี expression เป็น `pcs*weight_spec` ทั้งที่ชื่อระบุว่าเป็นการหาร
ผิดทั้งใน `prime_erp.price_list_formulas` และใน seed `internal/scripts/price_list_formulas/price-list-formulas.json`

### F4 — ลำดับการประเมินสูตรไม่ตรงกับลำดับ dependency (Critical)

เมื่อ `[kg]` คือผลลัพธ์ของสูตร `kg` สูตร `kg` ต้องถูกประเมินก่อนสูตร `pcs` เสมอ
แต่ `internal/repositories/priceList/repository.go:641` ดึงสูตรมาด้วย

```sql
ORDER BY psfm.create_dtm DESC
```

คือเรียงตามเวลาที่ผูกสูตรเข้า subgroup ไม่ใช่เรียงตาม dependency
`update-latest-pricelist-subgroup.go:263` วน `priceListFormulas` ตามลำดับที่ได้มาตรง ๆ

สูตรที่ถูกประเมินก่อนในข้อมูลจริง:

| สูตรที่รันก่อน | subgroups | ลำดับถูกต้องหรือไม่ |
|---|---|---|
| `kg = Base price + Extra` | 665 | ถูก (โดยบังเอิญ) |
| `Pcs = [kg] x [Avg. kg stock]` | 307 | **ผิด** |
| `Pcs = [kg] x [Weight Spec]` | 215 | **ผิด** |
| `Pcs = (...) x [Avg kg. stock] x (1+2%)` | 131 | ไม่เกี่ยว (ไม่อ้าง `kg`) |
| `pcs = input` | 20 | ไม่เกี่ยว |
| `kg = [Pcs] / [Avg. kg stock]` | 12 | ไม่เกี่ยว (คู่เป็น input) |

**522 subgroup ประเมินสูตร `pcs` ก่อนที่ `kg` จะถูกคำนวณใหม่** ทำให้อ่าน `total_net_price_weight`
ของรอบคำนวณก่อนหน้า ราคาต่อชิ้นจึงตามหลังราคาต่อกิโลหนึ่งรอบเสมอ ต้องกดคำนวณสองครั้งจึงจะตรง

### F5 — ข้อมูลการผูกสูตรไม่ครบคู่ (รายงานเท่านั้น ไม่แก้ในงานนี้)

147 subgroup ผูกสูตร uom `pcs` ทั้งสองตัว ไม่มีสูตร uom `kg` เลย

- 131 ตัวผูกสูตร `(base_price+2.1)*avg_kg_stock*1.02` **ซ้ำสองครั้ง** คำนวณสิ่งเดียวกันซ้ำลง
  `total_net_price_unit` และ `total_net_price_weight` ไม่ถูกคำนวณใหม่เลย
- 16 ตัวผูก `pcs = input` ซ้ำสองครั้ง

เป็นปัญหาข้อมูลไม่ใช่โค้ด ต้องให้ธุรกิจตัดสินว่าแต่ละกลุ่มควรผูกสูตร `kg` ตัวไหน
งานนี้ส่งมอบเพียง query รายงานรายการ subgroup ที่ผูกไม่ครบคู่

### ขนาดผลกระทบ

จาก `prime_erp.price_list_subgroup_formulas_map`:

| expression | subgroups | F1 | F2 | F4 |
|---|---|---|---|---|
| `base_price+extra` | 1,187 | | | |
| `kg*avg_kg_stock` | 712 | ✓ | ✓ | ✓ (307 ตัวลำดับผิด) |
| `kg*weight_spec` | 475 | | ✓ | ✓ (215 ตัวลำดับผิด) |
| `(base_price+2.1)*avg_kg_stock*1.02` | 262 | ✓ | | |
| `pcs = input` | 48 | | | |
| `pcs/avg_kg_stock` | 16 | ✓ | ✓ | |

**F1 กระทบ 990 subgroup · F2 กระทบ 1,203 subgroup · F4 กระทบ 522 subgroup · F5 กระทบ 147 subgroup**
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

ลบตัวแปร `pcs` และ `kg` ที่อ่านจากสต็อกออก (`update-latest-pricelist-subgroup.go:242-243`
และ `get-calculated-pricelist-subgroup.go:228-229`) แล้วป้อน env จากค่า running ที่โค้ดมีอยู่แล้ว:

```go
priceData := priceDomain.PriceData{
    BasePrice:  subGroup.PriceListGroup.PriceUnit,   // หรือ PriceWeight ตาม uom เดิม
    Extra:      extraPriceUnit,                      // หรือ extraPriceWeight ตาม uom เดิม
    AvgKgStock: avgKgStock,
    WeightSpec: weightSpec,
    Pcs:        totalNetPriceUnit,                   // ราคาต่อชิ้นล่าสุด
    Kg:         totalNetPriceWeight,                 // ราคาต่อกิโลล่าสุด
}
```

ตัวแปรทั้งสองถูก init จากค่าใน DB ที่บรรทัด 222-223 และถูกเขียนทับเมื่อสูตรของ uom นั้นคำนวณเสร็จ
จึงต้องสร้าง `priceData` **ใหม่ในแต่ละรอบของลูปสูตร** ไม่ใช่สร้างครั้งเดียวก่อนเข้าลูป
เพื่อให้สูตรที่รันทีหลังเห็นค่าที่สูตรก่อนหน้าเพิ่งคำนวณ (โค้ดปัจจุบันสร้างใน `case` อยู่แล้ว
จึงได้พฤติกรรมนี้ฟรีเมื่อแก้ร่วมกับข้อ D2)

### D2. `prime-wms-erp-core` — บังคับลำดับการประเมินสูตรตาม dependency (F4)

ก่อนเข้าลูปประเมินสูตร เรียง `priceListFormulas` ใหม่ตาม dependency โดยอ่านจาก
`params.required` ที่เก็บไว้ใน `price_list_formulas` อยู่แล้ว

กฎ: ถ้าสูตร A มี `required` ที่อ้างตัวแปรชื่อเดียวกับ `uom` ของสูตร B แล้ว B ต้องรันก่อน A
เช่น `Pcs = [kg] x [Avg. kg stock]` มี `required: ["kg", "avg_kg_stock"]` และคู่ของมันมี `uom = "kg"`
จึงต้องรันสูตร kg ก่อน

กรณีอ้างอิงวนกัน (ทั้งสองอ้างกันเอง) ให้คงลำดับเดิมจาก query และ log warning ไว้
ไม่ทำให้ request ล้ม เพราะปัจจุบันไม่มีข้อมูลเช่นนั้น แต่ป้องกันการผูกสูตรผิดในอนาคต

ทำที่ชั้น service ไม่แก้ `ORDER BY` ใน repository เพราะ `create_dtm DESC` ยังเป็นลำดับที่ flow อื่นใช้
และการเรียงตาม dependency เป็นตรรกะของการคำนวณ ไม่ใช่ของการดึงข้อมูล

### E. `prime-wms-erp-core` — แก้ expression ที่ผิด

migration SQL ใหม่ใต้ `migrations/` แก้ expression ของสูตร `kg = [Pcs] / [Weight Spec]`
จาก `pcs*weight_spec` เป็น `pcs/weight_spec` และแก้ให้ตรงกันใน
`internal/scripts/price_list_formulas/price-list-formulas.json`

migration ต้องเป็น idempotent และระบุสูตรด้วย `formula_code` ไม่ใช่ `name`

### F. ของแถมที่อยู่บนเส้นทางเดียวกัน

`get-price-detail.go:317-326` ตัดเงื่อนไข `if inv.X > 0` ออก เขียนค่าตรง ๆ

**แก้ความเข้าใจผิดของ spec ฉบับแรก:** ตอนเขียน spec ผมระบุว่าค่าค้างมาจาก subgroup ต้นแบบ
(`expandedSG := sg`) ซึ่ง**ไม่จริง** — การ construct `models.PriceListSubGroupResponse`
ที่ `get-price-detail.go:211-237` ไม่เคยคัดลอก `InventoryWeight` มาจาก input `SubGroup` เลย
ค่าจึงไม่มีทางไหลมาจากทางนั้น

กลไกจริงคือ `models.InventoryWeightResponse` มีทั้ง field เก่า (`sum_qty`, `sum_weight`,
`avg_batch`) และใหม่ (`total_qty`, `total_weight`, `avg_weight`) อยู่ใน struct เดียวกัน
บรรทัด `expandedSG.InventoryWeight = []models.InventoryWeightResponse{inv}` copy ทั้ง struct
จาก `inv` ซึ่งรวม field เก่าที่ backend ส่งมาใน JSON แล้วเงื่อนไข `> 0` เขียนทับแค่บาง field
ทำให้ค่าเก่าค้างอยู่เมื่อค่าใหม่เป็น 0 จริง ๆ

(ยังเป็นรูปแบบเดียวกับบั๊ก Weight-spec รอบก่อนในแง่ที่การเขียนแบบมีเงื่อนไขปล่อยค่าเก่าให้ค้าง)

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
  (694 subgroup มี `total_net_price_unit = 0` อยู่ตอนนี้ และจะถูกคำนวณใหม่ให้ถูกเมื่อผู้ใช้กดคำนวณ
  แต่ยังไม่มีการ backfill อัตโนมัติ)
- ไม่แก้ F5 — ส่งมอบเพียง query รายงาน subgroup ที่ผูกสูตรไม่ครบคู่ ให้ธุรกิจตัดสินว่าควรผูกสูตร `kg` ตัวไหน
- ไม่แก้คอลัมน์ `avg_weight_ton` — รอคำตอบจากธุรกิจ

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
- env ที่ส่งเข้า expression engine มี `pcs` และ `kg` เป็น `totalNetPriceUnit` / `totalNetPriceWeight`
  ไม่ใช่ `TotalQty` / `TotalWeight`
- เคสตามตัวอย่างใน spec: `price_weight = 19.40` `extra = 0.20` `avg_kg_stock = 3.0` กับคู่สูตร
  `base_price+extra` + `kg*avg_kg_stock` → `total_net_price_weight = 19.60` และ
  `total_net_price_unit = 58.80` (test นี้จะ fail บนโค้ดปัจจุบันซึ่งให้ `0`)

unit test ของการเรียงลำดับสูตร (F4):

- สูตร `pcs` ที่ `required` มี `kg` คู่กับสูตร uom `kg` → สูตร kg ต้องอยู่ก่อนในลำดับที่เรียงแล้ว
  ไม่ว่า input จะเรียงมาแบบใด (ทดสอบทั้งสองลำดับ input)
- สูตรที่ไม่อ้างถึงกัน เช่น `(base_price+2.1)*avg_kg_stock*1.02` คู่กับ `pcs = input` → คงลำดับเดิม
- สูตรที่อ้างอิงวนกัน → คงลำดับเดิมและไม่ panic
- คู่ `kg = [Pcs] / [Avg. kg stock]` + `pcs = input` → ลำดับต้องให้ค่า `pcs` พร้อมก่อนสูตร kg
- integration: คู่สูตร `base_price+extra` + `kg*avg_kg_stock` ที่ผูกโดยให้สูตร pcs มี `create_dtm`
  ใหม่กว่า (เลียนแบบ 522 subgroup ที่ลำดับผิด) → กดคำนวณครั้งเดียวต้องได้ราคาต่อชิ้นที่ถูกต้อง
  ไม่ต้องกดสองครั้ง (test นี้จะ fail บนโค้ดปัจจุบัน)

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
4. ส่ง query รายงาน F5 ให้ทีม/ธุรกิจ (ไม่มีการแก้โค้ดหรือข้อมูล)

ข้อ D และ D2 ต้องอยู่ใน PR เดียวกัน แก้ D อย่างเดียวโดยไม่แก้ D2 จะทำให้ 522 subgroup
อ่านค่า running ของรอบก่อนหน้า ซึ่งผิดในรูปแบบใหม่แทนที่จะหายไป

ข้อ 2 ต้องรอข้อ 1 ขึ้น environment ก่อนเพราะพึ่งพา field `avg_product` จาก response
ระหว่างนั้น `AvgProduct` จะเป็น 0 แล้วตกไป fallback `1.0` ซึ่งเป็นพฤติกรรมเดียวกับกรณีไม่มีสต็อก
จึงไม่ทำให้ระบบล่ม แต่ต้องไม่ปล่อย erp-core ขึ้น production ก่อน warehouse-core
