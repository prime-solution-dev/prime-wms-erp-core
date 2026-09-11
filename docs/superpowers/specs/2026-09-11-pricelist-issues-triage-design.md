# Design — แก้ issue ชุด Price List 5 กลุ่ม (คัดแยก + หาสาเหตุ + แผนแก้)

วันที่: 2026-09-11
ฐาน: `origin/Develop` ของทั้ง 3 repo (PR #85 erp-core และ #26 warehouse-core merge แล้วเมื่อ 2026-09-11 02:39Z)
ที่มา: handoff `/tmp/claude-handoff/2026-09-11-pricelist-issues-triage.md`

## สรุปผู้บริหาร

ผู้ใช้แจ้ง issue 5 กลุ่ม สอบสวนแล้วพบว่าเป็นบั๊กคนละตัวกัน 8 จุด ไม่ใช่รากเดียว
ทุกข้อมีหลักฐานยืนยันจากโค้ดและจากข้อมูลจริงใน DB (SELECT เท่านั้น) ไม่มีข้อใดเป็นการเดา

**ข้อที่เสียหายหนักสุดคือ 4a: `GROUP_1_ITEM_9` ทำข้อมูลหาย 76 จาก 80 subgroup** และแก้ได้ด้วย config บรรทัดเดียว

**issue 1 ไม่แก้ในรอบนี้** เพราะต้องให้ธุรกิจตัดสินมาตรฐานหน่วยก่อน ส่งมอบเป็นรายงานแทน

## คัดแยก: เกี่ยวกับงานรอบก่อนหรือไม่

| issue | เกี่ยวกับ PR #85/#26 | เหตุผล |
|-------|---------------------|--------|
| 1 ตัวเลขหน้าจอ ≠ เอกสาร | **ไม่เกี่ยว** | สาเหตุคือ `parsePercent` ใน `upload-pricelist.go` หารด้วย 100 ซึ่งไม่มีใครแตะในรอบก่อน (handoff เดาว่าเป็นเรื่องการปัดเลข `avg_kg_stock` — เดาผิด) |
| 2a Reset → before ผิด | ไม่เกี่ยว | ไม่มี save path ไหนเคยเขียน `before_*` มาตั้งแต่ต้น |
| 2b ไม่มี comma | ไม่เกี่ยว | ขอบเขต formatter ฝั่ง web |
| 3 ลำดับเปลี่ยน | **เกี่ยวบางส่วน** | รอบก่อนแก้ลำดับ `inventory_weight[]` ชั้นใน แต่ชั้นนอกยังไม่ได้แก้ (reviewer ชี้ไว้แล้ว) |
| 4a ITEM_9 แสดงไม่หมด | ไม่เกี่ยว | config ขาด `PG06` มาตั้งแต่ต้น |
| 4b `PG08_5` fallback | ไม่เกี่ยว | มี test ตรึงพฤติกรรมไว้ตั้งแต่ก่อนรอบก่อน |
| 4c center หัวตาราง | ไม่เกี่ยว | งาน config/CSS |
| 5 Extra ไม่ validate | ไม่เกี่ยว | ขาด validation มาตั้งแต่ต้น |

## รากสาเหตุที่ยืนยันแล้ว

### 4a — `GROUP_1_ITEM_9` ยุบ 80 subgroup เหลือ 4 แถว

`GROUP_1_ITEM_9_PATTERN.json` ประกาศ `grouping.rows = "PG02|PG07"` แต่ `PG06` เป็น pinned
`fixedColumns` ระดับแถว (`dataMapping: PG06`) ที่ไม่ได้อยู่ใน row key

ผลสองชั้น:

1. `patterns/shared.go:1131-1133` เขียนลง row เฉพาะ field ที่อยู่ใน `rowFields` → **คอลัมน์ PG06
   ที่ pinned ไว้ไม่มีข้อมูลป้อนเลย**
2. `pattern_group_1_item_9.go:104-144` `mergeGroup1Item9Rows` ยุบแถวตาม `row_group_value`
   (= row key จาก `shared.go:1124`) → subgroup ที่ต่างกันแค่ PG06 ทับกันแบบ last-write-wins

หลักฐานจาก DB:

| การนับ | ผล |
|--------|-----|
| subgroup ทั้งหมดของ `GROUP_1_ITEM_9` | 80 |
| แถวที่แสดงตอนนี้ (key = `PG02\|PG07`) | **4** |
| แถวถ้าเพิ่ม PG06 (key = `PG02\|PG07\|PG06`) | **21** |
| cell `(row, PG03)` ที่มี subgroup ทับกัน — ตอนนี้ | **12** |
| cell `(row, PG03)` ที่มี subgroup ทับกัน — ถ้าเพิ่ม PG06 | **0** |

การกระจาย: `PG02_10|PG07_8` 28 ตัว · `PG02_10|PG07_7` 27 ตัว · `PG02_3|PG07_7` 24 ตัว ·
คีย์ว่าง 1 ตัว โดยแต่ละแถวมี PG03 ต่างกัน 4 ค่าและ PG06 ต่างกัน 6-7 ค่า → `28 = 4 × 7`

การที่เพิ่ม PG06 แล้ว collision เหลือศูนย์พอดี ยืนยันว่า **การยุบเป็นบั๊ก ไม่ใช่ feature**
(สมมติฐาน "ยุบโดยตั้งใจแล้วต้อง aggregate" ถูกหักล้าง)

audit config ทั้ง 23 pattern แล้ว มีแค่ ITEM_9 ที่ผิดรูปนี้ — pattern ที่ `rows=""` ไม่ยุบแถว
และ field ที่มี `_x_` เป็นคอลัมน์แสดงผลประกอบ ไม่ใช่ key

### 3 — ลำดับรายการเปลี่ยนหลังกดบันทึก

วน map แล้ว `append` เข้า slice โดยไม่ sort 3 จุด:

- `prime-wms-warehouse-core/internal/services/inventory-service/get-inventory-weight-by-key.go:113`
  — `keyValueGroups` ประกอบจาก `for id, items := range idGroups`
- `prime-wms-erp-core/internal/services/price-service/get-price-detail.go:264` — `companyCodeSet`
- `prime-wms-erp-core/internal/services/price-service/get-price-detail.go:268` — `siteCodeSet`

ฝั่ง web ไม่ sort เองเลย: `BasePriceTable.vue:341-353` `filteredDataTable` filter ด้วย
`searchValue` อย่างเดียว → เชื่อลำดับจาก backend 100%

ฉะนั้นลำดับเปลี่ยน**ทุกครั้งที่เรียก API** ไม่ใช่แค่ตอนกดบันทึก แต่กดบันทึกแล้ว reload
คือจังหวะที่ผู้ใช้เห็นชัด

### 2a — กด Reset แล้วราคาช่อง before ผิด

**แก้ไขการวินิจฉัยเดิม**: รอบแรกสรุปว่า "ไม่มี save path ไหนเขียน `before_*` เลย" ซึ่ง**ผิด**
เพราะสรุปจาก `grep` ที่ถูก `head -20` ตัดผลทิ้ง ไล่ใหม่แบบไม่ตัดพบ 142 จุดที่อ้างถึง
คอลัมน์กลุ่มนี้ และ `internal/repositories/priceList/repository.go:490-516` เขียนจริง

```go
if req.TotalNetPriceWeight != nil {
    updateMap["before_total_net_price_weight"] = oldSubGroup.TotalNetPriceWeight
    updateMap["total_net_price_weight"] = *req.TotalNetPriceWeight
}
```

semantic ตรงตามที่ผู้ใช้ยืนยัน (`before` = ค่าก่อนแก้) แต่**ไม่มีเงื่อนไขว่าค่าต้องเปลี่ยนจริง**

รากสาเหตุจริงอยู่ที่ผู้เรียก `update-latest-pricelist-subgroup.go:321-324`

```go
updateChanges = append(updateChanges, models.UpdatePriceListSubGroupItem{
    SubGroupID:          subGroupID,
    TotalNetPriceUnit:   &totalNetPriceUnit,
    TotalNetPriceWeight: &totalNetPriceWeight,
})
```

ส่ง pointer ของทั้งสอง field **ทุก subgroup ทุกครั้งไม่มีเงื่อนไข** แม้ราคาที่คำนวณได้จะ
เท่ากับค่าเดิม ฉะนั้นทุกครั้งที่ `update-latest` รัน `before_*` จะถูกเลื่อนมาเท่ากับค่าปัจจุบัน

ที่ทำให้อาการกระจายวงกว้างคือ `update-latest` รองรับ `update_type = "group"`
(บรรทัด 105-125) ซึ่งดึง subgroup **ทั้งกลุ่ม** มาคำนวณใหม่ และฝั่ง web เรียกแบบ
fire-and-forget หลังแก้ Extra (`Extra.vue` `onUpdate` →
`updateLatestPriceListSubGroup([groupCode])`) → แก้ extra แถวเดียวทำให้ `before_*` ของ
ทุก subgroup ในกลุ่มถูกทับ

หลักฐานจาก DB เมื่อ 2026-09-11:

| สถานะ | subgroup |
|-------|----------|
| `before_total_net_price_weight = total_net_price_weight` (snapshot ถูกทับจนค่า before หายไป) | **1,257** |
| `before_total_net_price_weight <> total_net_price_weight` (ยังมีค่าจริง) | 93 |
| ทั้งหมด | 1,350 |

**93% ของ subgroup สูญค่า before ไปแล้ว**

อาการที่ผู้ใช้เห็นจึงเป็น: ก่อนกด Reset กริดแสดง `before` จาก calculate response ซึ่งคำนวณสด
และถูกต้อง (`get-calculated-pricelist-subgroup.go:209-210`) · กด Reset →
`handleReset()` (`PriceListGROUP1ITEM6Detail.vue:328-342`) เรียก `loadDetailData()` →
อ่านคอลัมน์ที่ persist ซึ่งถูกทับจนเท่ากับ after แล้ว → ตัวเลขเปลี่ยน

หมายเหตุ: `before_term_price_unit` **ไม่มีผู้เขียนเลย** ใน `updateMap` (เขียนครบ 7 จาก 8
คอลัมน์) พบตอนไล่ใหม่ จึงเป็นบั๊กย่อยที่ต้องแก้พร้อมกันใน PR C

สมมติฐานที่ถูกหักล้าง: เดาว่า 2a เกิดจาก row collapse เหมือน 4a → ตกไป เพราะ
`GROUP_1_ITEM_6_PATTERN.json` มี `rows=""` ไม่ยุบแถว

### 4b — `group_item` ค่าว่างโดยตั้งใจถูก fallback ไป `item_code`

`build-pricelist-detail-tab.go:186-188`

```go
name := itemNameByCode(k.Value)
if name == "" {
    name = k.Value   // → "PG08_5"
}
```

`itemNameByCode` คืน `string` เดี่ยว จึงแยกไม่ออกระหว่าง "ไม่มี record" กับ "มี record
แต่ชื่อว่าง" เพราะทั้งสองกรณีคือ empty string
`TestBuildPricelistDetailTab_FallbackToRawCode` (บรรทัด 277-295) ตรึงพฤติกรรมรวมนี้ไว้

กฎที่ผู้ใช้ยืนยัน: **ถ้ามี record ใน DB แต่ชื่อว่าง = ว่างจริง ห้าม fallback**
fallback ไป code ใช้ได้เฉพาะกรณีหา record ไม่เจอ

จุดเดียวกันมีที่ระดับ group ด้วย (บรรทัด 152-154 `groupName` → `g.GroupCode`)

### 5 — Extra Price List ไม่ตรวจค่าว่าง

ขาด validation ทั้ง 2 ชั้น:

- **backend** `models/pricelist.go:448-469` ไม่มี `binding:"required"` เลย —
  `ExtraKey`, `ConditionCode`, `Operator`, `CondRangeMin`, `CondRangeMax`,
  `PriceListGroupExtraKeys` รับ zero value ได้ทั้งหมด ·
  `update-pricelist.go:164` `UpdateExtras` มีแค่ `checkForOverlappingConditions()` (บรรทัด 171-174)

  **ข้อสำคัญที่พบตอนไล่โค้ด**: route `/UpdatePriceListExtra`
  (`internal/routes/routes.go:81-82`) ใช้ `utils.ProcessRequest` ซึ่งอ่าน raw body แล้วให้
  service ทำ `json.Unmarshal` เอง (`utils/request-handler.go:11-29`) →
  **การใส่ `binding:"required"` tag จะไม่ทำงานเลย** เพราะไม่ได้ผ่าน `ShouldBindJSON`
  และ error ที่ service คืนจะกลายเป็น HTTP **500** ไม่ใช่ 400
  (route อื่นอย่าง `/SubGroup/UpdateLatest` ใช้ `ProcessRequestWithBinding` ซึ่งแปลง
  `*utils.BindingError` เป็น 400 ให้)
- **web** `Extra.vue:103-109` มี `<a-form :model="formState" :rules="rules">` แต่ `rules`
  ไม่ cover field ไหนของ `formState` เลย

### 2b — ไม่แสดง `,` คั่นหลักพัน

`priceListNumberFormat.ts` มี `NUMERIC_SUFFIXES` **16 ตัว** (ไม่ใช่ 26 อย่างที่สรุปไว้รอบแรก)
และ match แบบ `field === suffix || field.endsWith('_' + suffix)` → ครอบคอลัมน์ที่มี prefix
กลุ่มอย่าง `pg09_2_avg_weight` ได้อยู่แล้ว และครอบ `before_total_net_price_weight` ได้ด้วย
เพราะลงท้ายด้วย `total_net_price_weight` กลไกไม่พัง แค่ลิสต์ไม่ครบ

ลิสต์ปัจจุบัน: `price_unit`, `price_weight`, `total_net_price_unit`,
`total_net_price_weight`, `extra_price_unit`, `extra_price_weight`, `extra_thb`,
`avg_weight`, `avg_kg_stock`, `avg_weight_ton`, `total_weight`, `market_weight`,
`stock`, `stock_quantity`, `quantity`, `ton`

ที่ขาดชัดเจนคือ `line_bundle` ซึ่งเป็นคอลัมน์ตัวเลขใน config ของหลาย pattern

ผู้ใช้สั่ง: ให้ครอบ**ทุกคอลัมน์ที่เป็นตัวเลขเงินหรือน้ำหนัก**

### 4c — จัดหัวตาราง / ลำดับให้ center

`DynamicTable.vue:365-374` ตั้ง default `headerClass: 'ag-header-cell-center'` +
`cellClass: 'cell-center'` และ `applyHeaderAlignment()`
(`PriceListGROUP1ITEM6Detail.vue:1426-1444`) ก็ fallback เป็น `'center'` อยู่แล้ว
(`getColumnAlignment(updatedCol) || 'center'`) → **หัวตารางเป็น center อยู่แล้ว**

ที่ไม่ center คือ**เซลล์ข้อมูล** เพราะ config ฝั่ง backend ตั้ง `"textAlign": "left"` ไว้
นับได้ **81 จุดใน 9 ไฟล์** เทียบกับ `"textAlign": "center"` 78 จุด

ไฟล์ที่มี `left`: `GROUP_1_ITEM_2` (10), `GROUP_1_ITEM_3` (14), `GROUP_1_ITEM_4` (9),
`GROUP_1_ITEM_5` (12), `GROUP_1_ITEM_8` (6), `GROUP_1_ITEM_9` (6),
`GROUP_1_ITEM_10` (6), `GROUP_1_ITEM_11` (5), `PG01_3` (13)

→ แก้ที่ config JSON ฝั่ง backend ไม่ต้องแตะ web เลย

### 1 — ตัวเลขหน้าจอไม่ตรงกับเอกสาร

**มาตรฐานที่ธุรกิจยืนยัน (2026-09-11): `1` = 1% และ `0.1` = 0.1%**
คือ `pdc_percent` / `due_percent` เก็บเป็นจำนวนเปอร์เซ็นต์ตรง ๆ และ `บาท = price × percent / 100`

ฉะนั้น:

- ฝั่งหน้าจอ **ถูกต้อง** — `BasePriceTable.vue:419-438` ใช้ `percent/100 × price` และแสดงผล
  เป็น `{{ percent }}%` ตรงตามมาตรฐาน ไม่ต้องแก้
- ฝั่ง export **ถูกต้อง** — `get-price-export-table.go:490-578` `buildBasedPriceTab` เป็น
  passthrough ล้วน ไม่คำนวณ ไม่ปัด (number format ในไฟล์เป็น `General` ไม่ใช่ percent format
  ยืนยันจากไฟล์ export จริงที่ผู้ใช้ส่งมา)
- **ฝั่ง import ผิด** — `upload-pricelist.go:1197-1204` `parsePercent` หารด้วย 100
  เมื่อ cell ใน Excel มี `%` ต่อท้าย

```go
parsePercent := func(s string) float64 {
    s = strings.TrimSpace(s)
    if strings.HasSuffix(s, "%") {
        v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, "%")), 64)
        return v / 100        // ← ผิด: cell "3.0%" ควรเก็บ 3 ไม่ใช่ 0.03
    }
    return parseFloat(s)
}
```

comment เหนือฟังก์ชันระบุเจตนาไว้ว่า "Excel percent cells come back already formatted
("1.0%"), not as "0.01"" ซึ่งถูก แต่ข้อสรุปที่ตามมาว่าต้องหาร 100 ผิด เพราะ DB เก็บเป็น
จำนวนเปอร์เซ็นต์ ไม่ใช่เศษส่วน

#### ความเสียหายในข้อมูลจริง

คอลัมน์ `pdc` / `due` (บาท) **ถูกต้องอยู่แล้ว** ตรวจทวนกับมาตรฐานที่ยืนยันมาแล้ว:

| กลุ่ม | term | price | percent ใน DB | baht ใน DB | `price × (percent×100)/100` |
|-------|------|-------|--------------|-----------|---------------------------|
| หมวดเหล็กท่อ | T3 | 19.4 | 0.03 | 0.58 | 19.4 × 3/100 = **0.58** ✓ |
| หมวดI Beams | T3 | 30.8 | 0.025 | 0.77 | 30.8 × 2.5/100 = **0.77** ✓ |
| หมวดเหล็กเส้น | T3 | 18.3 | 0.03 | 0.55 | 18.3 × 3/100 = **0.55** ✓ |

ผิดเฉพาะคอลัมน์ `*_percent` ที่เล็กไป 100 เท่า: `pdc_percent` 46 แถว, `due_percent` 48 แถว
(ทุกแถวที่นำเข้าผ่าน Excel upload)

ความเสียหายชั้นที่สองเกิดจากหน้าจอ: `calculateUpdatedPrice` (`BasePriceTable.vue:355-368`)
วน **ทุก term** แล้วคำนวณทั้ง PDC และ DUE ใหม่ทุกครั้งที่ผู้ใช้แตะช่อง adjust
(`@change`/`@blur` บรรทัด 84-85) แล้ว `.toFixed(2)` → baht ถูกคำนวณจาก percent ที่เพี้ยน
กลายเป็นเล็กลง 100 เท่าและเซฟทับ DB

นับจาก DB เมื่อ 2026-09-11 ในบรรดาแถวที่ percent เพี้ยน:

| คอลัมน์ | baht ยังถูก (แก้แค่ percent) | baht เสียหายด้วย (ต้องคำนวณใหม่) |
|---------|---------------------------|--------------------------------|
| `pdc` / `pdc_percent` | 33 แถว | **13 แถว** |
| `due` / `due_percent` | 33 แถว | **15 แถว** |

ตัวอย่างแถวที่เสียหายชัด: หมวดเหล็กแผ่น, หมวดเหล็กแบน, หมวดฉาก ราง, หมวดตัวซี ที่ term T3
มี `pdc = 0.01` ทั้งที่ควรเป็น 0.58–0.64

และ 8 แถวที่ผู้ใช้พิมพ์ผ่านหน้าจอโดยตรงจึงเก็บถูกมาตรฐาน (`percent` = 1–6) — ตัวซี GI
T1/T2/T3, หมวดเหล็กแผ่น T1, หมวดเหล็กแบน T1 และอีก 3 แถวฝั่ง due
แถวกลุ่มนี้**ไม่ต้องแก้**

#### ช่องว่างที่ยังเหลือ

ยังไม่ได้จับคู่ตัวเลขบนหน้าจอกับในเอกสารเป็นคู่ ๆ เพราะไม่มี screenshot ของหน้าจอ ณ เวลาที่
export ไฟล์นั้น อาการที่ผู้ใช้เห็นอธิบายได้จากรากสาเหตุข้างต้นทั้งหมดแล้ว แต่ถ้าหลังแก้แล้ว
ยังเห็นไม่ตรง ต้องขอ screenshot คู่กับเวลาที่กด export

## แผนแก้: 6 PR แยกตาม issue

ทุก PR แตกจาก `origin/Develop`

| PR | branch | issue | repo |
|----|--------|-------|------|
| A | `fix/pricelist-item9-row-key` | 4a | erp-core |
| B | `fix/pricelist-deterministic-order` | 3 | warehouse-core + erp-core |
| C | `fix/pricelist-before-price-snapshot` | 2a | erp-core |
| D | `fix/pricelist-extra-validation` | 5 | erp-core + web |
| E | `fix/pricelist-display-fixes` | 2b, 4b, 4c | web + erp-core |
| F | `fix/pricelist-percent-convention` | 1 | erp-core |

เรียงตามความเสียหาย: A ทำข้อมูลหาย 76/80 แถว จึงมาก่อน

### PR A — ITEM_9 row key

`patterns/configs/GROUP_1_ITEM_9_PATTERN.json`: `"rows": "PG02|PG07"` → `"PG02|PG07|PG06"`

บรรทัดเดียว ไม่แตะ Go เพราะ `row_group_value` (`shared.go:1124`) และการเขียนคอลัมน์
(`shared.go:1131-1133`) อ่านจาก `grouping.rows` ทั้งคู่ การแก้จุดเดียวจึงคุมทั้ง merge key
และการป้อนข้อมูลคอลัมน์ PG06

- verify: unit test สอง subgroup ที่ต่างกันแค่ PG06 → ต้องได้ 2 แถว (ตอนนี้ได้ 1)
- verify: integration ยืนยัน 21 แถวสำหรับชุดข้อมูล 80 subgroup และคอลัมน์ PG06 ไม่ว่าง

### PR B — ลำดับ deterministic

sort ที่ backend ให้เป็นแหล่งความจริงเดียว ไม่แก้ฝั่ง web

- `get-inventory-weight-by-key.go:113` — sort `keyValueGroups` ด้วย key หลัง build เสร็จ
- `get-price-detail.go:264, 268` — `sort.Strings(companyCodes)` และ `sort.Strings(siteCodes)`
- verify: unit test เรียกฟังก์ชันซ้ำ 20 รอบด้วย input เดียวกัน ต้องได้ลำดับเท่ากันทุกรอบ

### PR C — เลื่อน snapshot `before_*` เฉพาะเมื่อค่าเปลี่ยนจริง

รากสาเหตุไม่ใช่ "ไม่มีใครเขียน" แต่เป็น "เขียนทุกครั้งแม้ค่าไม่เปลี่ยน" ฉะนั้นการแก้คือ
เพิ่มเงื่อนไข ไม่ใช่เพิ่มการเขียน

**จุดที่แก้ (เลือกทางที่ diff เล็กที่สุดและครอบทุก caller)**:
`internal/repositories/priceList/repository.go:490-516` — ใส่เงื่อนไขว่าค่าใหม่ต่างจากค่าเดิม
ก่อนจะเลื่อน `before_*`

```go
if req.TotalNetPriceWeight != nil {
    if *req.TotalNetPriceWeight != oldSubGroup.TotalNetPriceWeight {
        updateMap["before_total_net_price_weight"] = oldSubGroup.TotalNetPriceWeight
    }
    updateMap["total_net_price_weight"] = *req.TotalNetPriceWeight
}
```

แก้ที่นี่ครอบทุก caller ในทีเดียว (ทั้ง `update-latest` และ
`update-pricelist-subgroup`) ส่วนการแก้ที่ `update-latest-pricelist-subgroup.go:321-324`
ให้ส่ง `nil` เมื่อค่าไม่เปลี่ยน จะแก้ได้แค่ caller เดียวและ caller อื่นยังพังอยู่

ทำให้ครบทั้ง 7 คอลัมน์ที่มีเงื่อนไขอยู่แล้ว และ**เพิ่ม `before_term_price_unit` ที่ยังไม่มี
ผู้เขียนเลย** ให้เข้าชุดกับ `before_term_price_weight`

- verify: unit test — เรียก update ด้วยค่าเท่าเดิม แล้ว `updateMap` ต้องไม่มี key `before_*`
- verify: unit test — เรียก update ด้วยค่าใหม่ แล้ว `before_*` ต้องเท่ากับค่าเดิม
- verify: integration test (testcontainers ที่ `repository_test.go` มีอยู่แล้ว) —
  save ค่าใหม่ 1 ครั้งแล้ว save ค่าเดิมซ้ำอีก 2 ครั้ง `before_*` ต้องยังเท่ากับค่าก่อนแก้ครั้งแรก
- verify: กด Reset บนหน้า Price List Detail แล้วค่าช่อง before ต้องไม่เปลี่ยน — repro ของอาการ

**การแก้นี้ไม่ย้อนไปซ่อม 1,257 แถวที่ `before` หายไปแล้ว** ค่าเดิมสูญไปอย่างถาวรใน
`price_list_sub_group` แต่ยังกู้ได้จาก `price_list_sub_group_history` ซึ่ง
`repository.go:394-547` เขียนไว้ทุกครั้ง — ถ้าธุรกิจต้องการกู้ ให้แยกเป็นงานต่างหาก
พร้อมรายงานก่อนเหมือน issue 1

### PR D — Extra validation

`binding:"required"` tag ใช้ไม่ได้เพราะ route นี้ผ่าน `utils.ProcessRequest` ซึ่งไม่ใช้
`ShouldBindJSON` ฉะนั้นต้อง validate ด้วยโค้ดตรง ๆ

2 ชั้น เพราะ backend เป็น trust boundary การพึ่งฟอร์มฝั่ง web อย่างเดียวไม่พอ

**backend**

1. `internal/utils/request-handler.go` — ให้ `ProcessRequest` แปลง `*BindingError`
   เป็น HTTP 400 เหมือนที่ `ProcessRequestWithBinding` ทำอยู่แล้ว ตอนนี้ error ทุกชนิด
   กลายเป็น 500 ซึ่งทำให้แยก "ผู้ใช้ส่งข้อมูลไม่ครบ" กับ "ระบบพัง" ไม่ออก
   แก้จุดเดียวได้ประโยชน์กับทุก route ที่ใช้ `ProcessRequest`
2. `internal/services/price-service/update-pricelist.go` `UpdateExtras` — เพิ่มการ validate
   หลัง `json.Unmarshal` ก่อน `checkForOverlappingConditions` คืน `*utils.BindingError`
   เมื่อพบข้อผิดพลาด ตรวจต่อรายการและระบุ index ในข้อความ:
   - `PriceListGroupID` ต้องไม่เป็น `uuid.Nil`
   - `ExtraKey` ต้องไม่ว่างหลัง `strings.TrimSpace`
   - `ConditionCode` ต้องไม่ว่างหลัง `strings.TrimSpace`
   - `Operator` ต้องไม่ว่างและต้องเป็นค่าที่ `getEffectiveRange` รู้จัก
     (`<=`, `>=`, `=`, `<>`) เพราะตอนนี้ operator ที่ไม่รู้จักตกไป `default` เงียบ ๆ
   - `CondRangeMin <= CondRangeMax`
   - `PriceListGroupExtraKeys` ต้องไม่เป็น slice ว่าง และแต่ละตัว `Code` / `Value`
     ต้องไม่ว่าง

**web** `Extra.vue:103-109` — `rules` ปัจจุบันมีแค่ key `pass` ซึ่งไม่ตรงกับ field ไหนของ
`formState` เลย จึงไม่ validate อะไร ต้องเปลี่ยนไปตรวจที่ `onUpdate` ก่อนเรียก API
เพราะ `formState` เป็น `PriceListExtra[]` (array) ไม่ใช่ object เดียว `a-form :rules`
จึงใช้กับมันตรง ๆ ไม่ได้ · ตรวจ field ชุดเดียวกับ backend และแสดง message ระบุแถว

- verify: unit test backend — ยิง payload ที่ field ว่างทีละตัว ต้องได้ `*utils.BindingError`
  และ repository ต้องไม่ถูกเรียก (ใช้ pattern การ swap function var เหมือน
  `TestUpdatePriceListSubGroup_Validation_MissingID` ใน
  `update-pricelist-subgroup_service_test.go`)
- verify: unit test `ProcessRequest` — service คืน `*BindingError` ต้องได้ status 400
  ส่วน error ชนิดอื่นต้องยังได้ 500
- verify: vitest ฝั่ง web — `onUpdate` ที่ข้อมูลไม่ครบต้องไม่เรียก API

### PR E — แสดงผล

**2b comma** (`web`) — เพิ่ม `line_bundle` เข้า `NUMERIC_SUFFIXES` ใน
`src/utils/helper/priceListNumberFormat.ts` และไล่ config JSON ฝั่ง backend ทุกไฟล์เพื่อหา
field ตัวเลขอื่นที่ยังไม่อยู่ในลิสต์ กลไก suffix match ใช้ได้อยู่แล้วไม่ต้องแก้
(ครอบ `pg09_2_avg_weight` และ `before_total_net_price_weight` ได้แล้ว)

**4b fallback** (`erp-core`) — `itemNameByCode` ถูกสร้างเป็น closure ที่
`get-price-export-table.go:90-101`

```go
itemNameByCode := func(code string) string {
    if it, ok := groupItemMap[code]; ok {
        return it.ItemName
    }
    return ""
}
```

`ok` มีอยู่แล้วแต่ถูกทิ้ง เปลี่ยน signature เป็น `func(string) (string, bool)` แล้วให้
`build-pricelist-detail-tab.go:186-188` fallback เฉพาะเมื่อ `!ok`

```go
name, found := itemNameByCode(k.Value)
if !found {
    name = k.Value
}
row[k.Code] = name
```

ทำแบบเดียวกันกับ `groupNameByCode` ที่ระดับ group (บรรทัด 152-154) เพราะเป็นบั๊กเดียวกัน
· ต้องแก้ทุก caller ของทั้งสอง closure ให้ครบ (`buildPricelistDetailTab` และ
`selectExportTabs`) · แก้ `TestBuildPricelistDetailTab_FallbackToRawCode`
(`build-pricelist-detail-tab_test.go:277-295`) ซึ่งตอนนี้ส่ง
`none := func(string) string { return "" }` ให้ตรงกับ signature ใหม่ และ**เพิ่มเคสใหม่**
"มี record แต่ `ItemName` ว่าง → เซลล์ต้องว่าง ไม่ fallback"

**4c center** (`erp-core`) — เปลี่ยน `"textAlign": "left"` เป็น `"center"` ใน config 9 ไฟล์
(81 จุด) ไม่ต้องแตะ web เพราะหัวตารางเป็น center อยู่แล้ว

- verify: vitest — `isNumericPriceListField('pg09_2_line_bundle')` ต้องเป็น `true`
- verify: unit test Go — เคส "มี record ชื่อว่าง" ต้องได้เซลล์ว่าง และเคส "ไม่มี record"
  ต้องได้ code ดิบ
- verify: `grep -c '"textAlign": "left"' internal/services/price-service/patterns/configs/*.json`
  ต้องไม่เหลือ

### PR F — มาตรฐาน percent ตอน import

`upload-pricelist.go:1197-1204` `parsePercent`: ตัด `/ 100` ออก ให้ cell `"3.0%"` เก็บเป็น `3`
และแก้ comment ที่อธิบายเจตนาผิดไว้ ไม่แตะฝั่ง web และไม่แตะ export เพราะทั้งสองถูกอยู่แล้ว

- verify: unit test ของ `parsePercent` — `"3.0%"` → 3, `"0.1%"` → 0.1, `"3"` → 3, `""` → 0
- verify: integration test upload ไฟล์ที่มี cell percent แล้วอ่านค่าใน DB กลับมาเทียบ
- ต้องตรวจว่ามี caller อื่นของ `parsePercent` หรือไม่ (ตอนนี้มีแค่ 2 จุด บรรทัด 1275, 1277)
  และคอลัมน์ percent อื่นในไฟล์ upload ใช้ `parseFloat` อยู่ — ต้องไม่ทำให้สองเส้นทางขัดกัน

**การแก้นี้ไม่ย้อนไปซ่อมข้อมูลเก่า** ต้อง backfill แยก ดูหัวข้อสิ่งที่ไม่ทำ

## ลำดับการ deploy และ backfill

**ตัดสินใจแล้ว (2026-09-11): backfill ข้อมูล percent ก่อน แล้วค่อย deploy PR F**

เหตุผล: ถ้า deploy PR F ก่อน การ upload ครั้งถัดไปจะเขียนค่าถูกต้อง แต่ข้อมูลเก่าที่เพี้ยน
ยังอยู่ จึงมีสองมาตรฐานปนกันในช่วงคาบเกี่ยว และผู้ใช้ที่แตะช่อง adjust บนแถวเก่าจะทำ
baht เสียหายเพิ่มอีก

ลำดับที่ต้องทำ:

1. รัน query ข้อ 6 และ 7 ของรายงาน ยืนยันรายการแถวที่จะแก้และค่าที่ควรเป็น
2. คำนวณ baht ใหม่เฉพาะแถวที่รายงานระบุว่า "baht เสียหายด้วย" (`pdc` 13 แถว, `due` 15 แถว)
   — **ต้องทำก่อนขั้นที่ 3** เพราะการคำนวณอ้าง `percent` ค่าเดิมที่ยังไม่ได้คูณ 100
3. backfill `pdc_percent` / `due_percent` ด้วย `× 100` เฉพาะแถวที่ `percent > 0 AND percent < 1`
4. รัน query ข้อ 2 และ 3 ซ้ำ ยืนยันว่าไม่มีแถวที่ขึ้น "ไม่ตรงสูตรใดเลย" เหลืออยู่
5. deploy PR F

**ผู้เขียนแผนนี้ไม่รัน write ลง DB** ตามข้อจำกัดของผู้ใช้ ขั้นที่ 2 และ 3 ส่งมอบเป็นไฟล์
`docs/superpowers/reports/2026-09-11-pricelist-term-percent-backfill.sql` ให้ผู้ใช้ตรวจและ
รันเอง ไฟล์นั้นอยู่ใน transaction และ **ไม่มี `COMMIT` ในตัวโดยเจตนา** เพื่อบังคับให้อ่านผล
การตรวจก่อน

ข้อควรระวังที่บันทึกไว้ในไฟล์ backfill: เกณฑ์ `percent > 0 AND percent < 1` จะผิดถ้ามีแถวที่
ตั้งใจให้เป็นอัตราต่ำกว่า 1% จริง (ซึ่งมาตรฐานอนุญาต เพราะ `0.1` = 0.1%) ข้อมูลเมื่อ
2026-09-11 มีค่าอยู่ในช่วง 0.01–0.04 เท่านั้นซึ่งอ่านเป็นอัตราจริงไม่ได้ จึงปลอดภัย
แต่ขั้นที่ 1 ของไฟล์บังคับให้ตรวจค่าที่มีอยู่ด้วยตาก่อนทุกครั้ง

PR A–E ไม่ผูกกับ backfill ปล่อยได้ทันทีตามลำดับความเสียหาย

## Test

- ทุก PR ต้องมี unit test ที่ **fail ก่อนแก้** — คือ repro ตาม debug mantra ข้อ 1
- integration test: `make test-integration-pricelist` (testcontainers postgres:16 ~6 วินาที
  `TestMain` ตั้ง DSN เอง ต้องมีแค่ docker daemon)
- coverage ≥ 80% ต่อ package ที่แตะ
- `price-service` inject dependency ผ่าน package-level function var
  (`getPriceListSubGroupsByIDsFunc` ฯลฯ) ทดสอบ flow เต็มได้โดยไม่ต้องมี DB —
  ดูตัวอย่างที่ `avg_kg_stock_flow_test.go`

## สิ่งที่ไม่ทำในงานนี้

- **backfill ข้อมูลที่เพี้ยนแล้ว** — `pdc_percent` 46 แถว และ `due_percent` 48 แถว ที่เล็กไป
  100× (ต้อง `× 100`) ในจำนวนนั้น `pdc` 13 แถวและ `due` 15 แถวมี baht เสียหายด้วย
  (ต้องคำนวณใหม่),
  1,194 จาก 1,350 subgroup ที่ `total_net_price_unit = 0` (handoff รอบก่อนบันทึกไว้ 694 — ตัวเลขจริงวันนี้สูงกว่า) ส่งมอบเป็น SELECT report
  ให้ตัดสินก่อน ไม่ write ลง DB
- **`avg_weight_ton`** — คอลัมน์ของ `GROUP_1_ITEM_7`/`GROUP_1_ITEM_22` มี headerName
  "Avg.kg stock (Tons)" แต่ `dataMapping` ชี้ `avg_weight` ซึ่งเป็น kg และไม่พบการหารด้วย
  1000 ที่ใดในระบบ — ค้างจากรอบก่อน รอคำตอบจากผู้ใช้
- **F5** — 147 subgroup ผูกสูตร uom `pcs` ทั้งคู่ ไม่มีสูตร `kg` รอธุรกิจตัดสิน
- **`cmd/.env` ของ warehouse-core ถูก track และมี credential ใน git history** — ควรแยกเป็นงาน
  (ย้ายไป `examples.env` + `.gitignore` + rotate) ไฟล์นี้มีสถานะ Modified ค้างอยู่
  **ห้าม commit ห้าม checkout ทับ**
- **fallback `1.0` เมื่อไม่มีสต็อก** — ผู้ใช้ตัดสินใจคงไว้แล้ว
