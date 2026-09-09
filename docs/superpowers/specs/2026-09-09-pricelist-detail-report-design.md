# Pricelist Detail Report — Design

- วันที่: 2026-09-09
- ขอบเขต: `prime-wms-erp-core`, `prime-wms-document-core`, `prime-wms-web`
- อ้างอิงผลลัพธ์ที่ต้องการ: `TMI_Pricelist detail Report_20260908.xlsx` ชีท `Template`

## 1. ปัญหา

หน้า Base Price (`prime-wms-web/src/views/price-list/BasePrice.vue`) มีปุ่ม `Print Document` ปุ่มเดียว
ที่ export Excel ตัวเดิมออกมาแบบไม่มี filter ผู้ใช้ต้องการ report ตัวที่สอง (Pricelist Detail Report)
ที่กรองด้วย Product Group 1 ได้ และให้ปุ่ม Print กลายเป็น dropdown ให้เลือกได้ 2 report

## 2. สิ่งที่ pipeline เดิมมีอยู่แล้ว

```
BasePrice.vue → priceListExport.ts → document-core /module-report-buttons/action
  → excel_dispatch.go → ExcelExport() → erp-core /price/GetPriceExportTable
  → pdfgen/excel_generator.go → ไฟล์ .xlsx
```

- `pdfgen/excel_generator.go:75-115` เขียน metadata แถว 1-3 (`Report` / `Last Updated` / `Download`),
  แถว 4 ว่าง, header แถว 5, data แถว 6+ — ตรงกับ layout ของชีท `Template` อยู่แล้ว
- `GetPriceExportTableResponse.Tabs[]` → 1 tab = 1 worksheet ดังนั้นขอ 1 tab ก็ได้ไฟล์ชีทเดียว
- `GetPriceListGroupRequest` มี `groupCodes []string` รองรับ filter อยู่แล้ว

สรุป: งานจริงเหลือแค่ "ประกอบ tab ให้ถูก" ที่ `erp-core` ที่เดียว

## 3. การ resolve ข้อมูล (ไม่ hardcode PG code)

1 แถวของรายงาน = 1 `price_list_sub_group`

- `price_list_sub_group_key` มี `code` (PG01..PG10) และ `value` (item code เช่น `PG01_3`) อยู่แล้ว
  จึงไม่ต้อง parse string `subgroup_key`
- ตาราง `group`: `group_code` (เช่น `PG01`) → `group_name` = **header** ของคอลัมน์
- ตาราง `group_item`: `item_code` (เช่น `PG01_3`) → `item_name` = **ค่าในเซลล์**
- ชุดคอลัมน์ PG มาจาก union ของ `price_list_sub_group_key.code` ที่พบจริง เรียงตาม `code`
  จึงมีกี่คอลัมน์ก็ได้ ไม่ผูกกับ 9 หรือ 10

### แมปคอลัมน์

| ลำดับ | Field | Header | ที่มา |
|---|---|---|---|
| 1 | `pricelist_group_name` | Pricelist group name | `groupItemMap[group.GroupCode].ItemName` |
| 2..n | `PGxx` | `groupMap[PGxx].GroupName` | ค่า = `groupItemMap[key.Value].ItemName` |
| | `weight_spec` | Weight-spec | `udf_json` / inventory enrichment (โค้ดเดิม) |
| | `avg_weight` | Avg. kg stock | โค้ดเดิม |
| | `price_per_kg` | Price per kg | `total_net_price_weight` |
| | `price_per_unit` | Price per unit | `total_net_price_unit` |
| | `extra_price` | Extra price | `extra_price_weight` |
| | `formula_kg_name` | Price per kg formula | formula ที่ `uom='kg'` → `name` |
| | `formula_unit_name` | Price per unit formula | formula ที่ `uom='pcs'` → `name` |
| | `pricelist_group_code` | Pricelist group code | `price_list_group.group_code` |
| n.. | `PGxx_code` | `PGxx` | `price_list_sub_group_key.value` |
| | `formula_kg_code` | Price per kg formula code | `uom='kg'` → `formula_code` |
| | `formula_unit_code` | Price per unit formula code | `uom='pcs'` → `formula_code` |
| | `subgroup_code` | subgroup_code | `price_list_sub_group.subgroup_code` |

Formula มาจาก `price_list_subgroup_formulas_map` JOIN `price_list_formulas`
(FK: `price_list_subgroup_code` → `price_list_sub_group.subgroup_code`) แยก kg / unit ด้วยคอลัมน์ `uom`

หมายเหตุ: ไฟล์ตัวอย่างเขียน header คอลัมน์รหัสเป็น `PG01..PG09` เรียงตามตำแหน่ง ซึ่งเหลื่อมกับ PG code จริง
ในฐานข้อมูล (ที่ข้าม `PG04` ในบาง group) การออกแบบนี้ยึด PG code จริงจากข้อมูล header ที่ออกจึงอาจ
ไม่ตรงตัวอักษรกับไฟล์ตัวอย่าง แต่ตรงกับข้อมูลจริง — เป็นการตัดสินใจที่ผู้ใช้ยืนยันแล้ว

## 4. erp-core

**`internal/models/pricelist.go`**
- `GetPriceListGroupRequest` เพิ่ม `ReportType string` (json tag `reportType`) — ค่าว่าง = พฤติกรรมเดิม
- struct ใหม่ `SubgroupFormula { SubgroupCode, FormulaCode, Name, Uom }`

**`internal/repositories/priceList/repository.go`**
- `GetFormulasBySubgroupCodes(codes []string) (map[string][]SubgroupFormula, error)`
  — query เดียวสำหรับทุก subgroup (กัน N+1)

**`internal/services/price-service/get-price-export-table.go`**
- แตกทางที่หัวฟังก์ชัน: `reportType == "PRICELIST_DETAIL"` → `Tabs: []ExportTab{ buildPricelistDetailTab(...) }`
  นอกนั้นคืน 2 tabs เดิม (`Detail`, `Based price`)
- `buildPricelistDetailTab()` ประกอบคอลัมน์ตามตารางข้อ 3
- ชื่อ tab = `Template`
- Tab headers: `Report: "Pricelist Detail"`, `LastUpdated` จาก `get-price-last-updated.go`,
  `Download` = เวลาปัจจุบัน
- Filter ใช้ `groupCodes[]` เดิม — ส่ง `[]` หมายถึง All

## 5. document-core

`excel_dispatch.go` switch ที่ `ModuleTopicCode + "/" + ModuleItemCode` ซึ่ง report ใหม่ใช้ค่าเดียวกับตัวเดิม
จึง **ไม่ต้องเพิ่ม case** — ให้ `ExcelExport()` อ่าน `reportType` จาก `template_config` ของ button record
แล้วส่งต่อไป `GetPriceExportTable`

`scripts/migrations/data/pricelist_detail_button.json` — record ใหม่:

```json
{
  "module_code": "PRICELIST",
  "module_item_code": "BASE_PRICE",
  "module_topic_code": "PRICE_LIST",
  "button_code": "EXPORT_EXCEL_DETAIL",
  "button_name": "Pricelist Detail Report",
  "button_action": "EXPORT_EXCEL",
  "template_config": {
    "filename": "Pricelist detail Report",
    "reportType": "PRICELIST_DETAIL"
  }
}
```

## 6. web

- **`src/views/price-list/BasePrice.vue`** — เปลี่ยน `GlobalButton` เดี่ยวเป็น `<a-dropdown>` 2 เมนู
  - `Pricelist Report` → เรียก `onPrintDocument()` เดิมตรง ๆ ไม่ขึ้น modal
  - `Pricelist Detail Report` → เปิด modal
- **`src/components/priceList/PricelistDetailExportModal.vue`** (ใหม่) — `a-select` เลือกได้ตัวเดียว
  option แรก `All` (value `''`) ที่เหลือ derive จาก price list group ที่หน้าโหลดไว้แล้ว
  (`groupCode` + `groupName`) ไม่ต้องยิง API เพิ่ม; ปุ่ม Export / Cancel
- **`src/utils/helper/priceListExport.ts`** — `callExportPriceExcel()` รับพารามิเตอร์เพิ่ม
  `buttonCode` (default `EXPORT_EXCEL`) และ `groupCodes: string[]` (default `[]`) แล้วใส่ลง payload

## 7. Error handling

- ไม่พบ formula สำหรับ subgroup → เซลล์ formula ว่าง ไม่ error
- resolve ชื่อ item ไม่ได้ → fallback แสดง code ดิบ (ดีกว่าเซลล์ว่างเงียบ ๆ)
- `groupCodes` ที่ส่งมาไม่มีในระบบ → คืน tab ที่มี header ครบแต่ไม่มีแถว ไม่ใช่ error
- frontend ใช้ try/catch + loading pattern เดิมของ `onPrintDocument()`

## 8. Test

**Unit — `go test ./internal/services/price-service/...`**
- `buildPricelistDetailTab`: คอลัมน์ PG ไดนามิกถูกต้องเมื่อ subgroup มีชุด PG ไม่เท่ากัน และเมื่อมี `PG04`
- แมป formula: `uom='kg'` → คอลัมน์ kg, `uom='pcs'` → คอลัมน์ unit, ไม่มี formula → ว่าง
- resolve ชื่อจาก `groupMap` / `groupItemMap` และ fallback เมื่อ resolve ไม่ได้
- `reportType` ว่าง → ยังคืน 2 tabs เดิม (regression guard)

**Integration — testcontainers, `make test-integration`**
- seed `group`, `group_item`, `price_list_group`, `price_list_sub_group`, `price_list_sub_group_key`,
  `price_list_formulas`, `price_list_subgroup_formulas_map` แล้วเรียก `GetPriceExportTable`
  เทียบผลกับแถวตัวอย่างในไฟล์ template
- `GetFormulasBySubgroupCodes` คืนครบทุก subgroup ด้วย query เดียว

เป้า coverage ของโค้ดใหม่ ≥ 80% (ฝั่ง web ไม่มี test runner ตาม CLAUDE.md)

## 9. ข้อตัดสินใจที่ผู้ใช้ยืนยันแล้ว

1. ปุ่ม Print กลายเป็น dropdown 2 report — ไม่เพิ่มปุ่ม `Export Detail` แยกอีกปุ่ม
2. Filter Product Group 1 เลือกได้ตัวเดียว + มีตัวเลือก `All`
3. ไฟล์ผลลัพธ์มีชีทเดียว (`Template`) ไม่มี `Parameter` / `Noted`
4. เฉพาะ `Pricelist Detail Report` ที่ขึ้น modal — `Pricelist Report` ตัวเดิมกดแล้ว export เลย
5. ขยาย endpoint เดิมด้วย `reportType` ไม่สร้าง endpoint ใหม่
6. ยึด PG code จริงจากตาราง `group` ไม่ hardcode แมป

## 10. ข้อสมมติ

1. `Extra price` (คอลัมน์ O) = `extra_price_weight` ไม่ใช่ `extra_price_unit`
2. คอลัมน์ PG เรียงตาม `code` และ header ใช้ `group_name` จาก DB
3. ชื่อชีท = `Template`

## 11. Branch

แตก branch ใหม่จาก `Develop` ทั้ง 3 repo — `prime-wms-document-core` ค้างอยู่บน
`fix/ib-gr-warehouse-name-from-item` ต้อง checkout `Develop` ก่อน
