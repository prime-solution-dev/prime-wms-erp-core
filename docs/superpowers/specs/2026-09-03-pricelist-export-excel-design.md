# Price List Export Excel — แก้ timezone และความถูกต้องของข้อมูล

วันที่: 2026-09-03
Branch: `feature/pricelist-export-tz` (แตกจาก `origin/Develop` ทั้ง 3 repo)

## บริบท

ปุ่ม Export Excel ในหน้า Base Price ของ Price List ทำงานผ่าน record ในตาราง
`module_report_buttons` ของ document-core:

```json
{
  "module_code": "PRICELIST",
  "module_item_code": "BASE_PRICE",
  "module_topic_code": "PRICE_LIST",
  "button_code": "EXPORT_EXCEL",
  "button_action": "EXPORT_CSV",
  "template_config": "{\"filename\": \"TMI_Pricelist_Report\"}"
}
```

### Flow ปัจจุบัน

```
prime-wms-web  BasePrice.vue:146
  -> utils/helper/priceListExport.ts
     POST /module-report-buttons/list  (document-core :9106)
       filter: module_code=PRICELIST, module_item_code=BASE_PRICE, module_topic_code=PRICE_LIST
       หา button_code == "EXPORT_EXCEL" -> ได้ id
     POST /module-report-buttons/action  { id, data: { company_code, site_codes: [site] } }
       (ไม่ส่ง group_codes = export ทุก group)

prime-wms-document-core
  routes/routes.go:550         /module-report-buttons/{list,action}
  module_report_button.go:51   ButtonAction == "EXPORT_CSV" -> NewBareContext -> ExcelDispatch
  handlers/excel_dispatch.go:8 key "PRICE_LIST/BASE_PRICE" -> ExcelExport
  handlers/excel_export.go:13  เรียก erp-core GetPriceExportTable แล้วสร้าง xlsx
  pdfgen/excel_generator.go    2 worksheet: "Detail" + "Based price"

prime-wms-erp-core
  price-service/get-price-export-table.go  ประกอบ Tabs/Columns/Rows

กลับมาที่ web -> downloadBlob(blob, "base_price_<company>_<site>.xlsx")
```

เส้นทางเชื่อมกันถูกต้องครบและใช้งานได้จริง ปัญหาอยู่ที่ **เนื้อหาของไฟล์ที่ออกมา**

## ปัญหาที่พบ

### P1 เวลาที่แสดงไม่ใช่เวลาไทย (บั๊กหลัก)

- `get-price-export-table.go:419,502` ใช้ `time.Now()` แล้วส่งเข้า `formatTimestamp` (`:525`)
- `prime-wms-erp-core/Dockerfile:30` ใช้ `FROM alpine:3.20` และติดตั้งแค่ `ca-certificates nano`
  **ไม่มี tzdata และไม่มี `ENV TZ`** -> Go หา zone ไม่เจอ จึง fallback เป็น **UTC**
- `excel_generator.go:215` `GenerateFilename` ใช้ `time.Now()` เช่นกัน

ผล: ช่อง `Last Updated` / `Download` ในไฟล์ Excel ช้ากว่าเวลาไทย 7 ชั่วโมง
และถ้า export ก่อน 07:00 น. ตามเวลาไทย **วันที่จะเพี้ยนไป 1 วัน**

ยืนยันเพิ่มเติม: `update-pricelist.go:27` เขียน `update_dtm` ด้วย `time.Now().UTC()`
ดังนั้นค่าใน DB เป็น UTC จริง ต้องแปลงตอนแสดงผลเสมอ

ในโปรเจกต์มี pattern ที่ถูกอยู่แล้วให้อ้างอิง — `document-core/.../handlers/picking_enrich.go:127`

### P2 `Last Updated` ไม่ได้สื่อความหมาย

`buildDetailTab` (`:419`) และ `buildBasedPriceTab` (`:502`) ตั้ง
`LastUpdated = Download = time.Now()` ทั้งคู่ -> "Last Updated" ไม่ได้บอกว่า
ราคาถูกแก้ล่าสุดเมื่อไหร่ ซึ่งเป็นสิ่งที่ผู้ใช้ต้องการจากช่องนี้

ข้อมูลมีอยู่แล้ว: `price_list_group.update_dtm` และ `price_list_sub_group.update_dtm`
(`internal/models/pricelist.go:26`, `:141`)

### P3 คอลัมน์ซ้ำใน Detail tab

`buildExportTableTyped` สร้างคอลัมน์ 3 รอบโดยไม่ dedupe:
1. static list `:250-296` — มี `PG01`..`PG10`, `is_highlight`, `line_bundle`, `coil_id`, `inactive`, ...
2. dynamic group-key columns `:298-305` — เติม `PG01`..`PG10` ซ้ำอีกรอบ
3. dynamic UDF columns `:326-333` — เติม UDF key ซ้ำอีกรอบ
   (ยืนยันว่า `is_highlight` / `line_bundle` / `coil_id` เป็น UDF key จริง —
   `patterns/shared.go:1145,1152,1179`)

### P4 ตัวเลขในไฟล์เป็น text

`excel_generator.go:174-201` `extractValue` คืน `string` ทุกกรณี
-> ราคา / น้ำหนัก / จำนวน ใน xlsx เป็นข้อความ ผู้ใช้ sum / sort / filter เป็นตัวเลขไม่ได้

### P5 `template_config.filename` ไม่มีผล

`excel_export.go:35` ตั้ง `Content-Disposition` จาก `template_config.filename` ถูกแล้ว
แต่ `BasePrice.vue:150` ตั้งชื่อไฟล์เองเป็น `base_price_<company>_<site>.xlsx` ทับ
-> ค่า `"TMI_Pricelist_Report"` ที่ config ไว้ไม่เคยถูกใช้

สาเหตุ: `useAxios.ts:52` interceptor `return response.data` ทิ้ง header ทั้งหมด
frontend จึงอ่าน `Content-Disposition` ไม่ได้

### P6 คอลัมน์ "จำนวน" ซ้ำหัวกัน

`:267-268` `stock_quantity` (จาก `inv.TotalQty`) กับ `quantity` (จาก `inv.SumQty`)
ใช้ HeaderName เดียวกันคือ `"จำนวน"` และ `quantity` ยังมีค่าเท่ากับ `stock` เป๊ะ (`:381,383`)

### P7 `is_highlight` โผล่เป็นคอลัมน์

`:290` `{Field: "is_highlight", HeaderName: "Highlight สีฟ้า"}` โชว์ `"true"/"false"`
ทั้งที่มันเป็น flag สำหรับทำสีตัวอักษร ซึ่ง `excel_generator.go:158` ทำอยู่แล้ว

## สิ่งที่จะแก้

### F1 Timezone -> เวลาไทย (P1)

**erp-core** `internal/services/price-service/get-price-export-table.go`

```go
// bangkok คือ timezone ที่ใช้แสดงผลทุกเวลาในรายงาน
// container เป็น alpine ที่ไม่มี tzdata จึงต้องมี FixedZone fallback
var bangkok = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

func formatTimestamp(t time.Time) string {
	return t.In(bangkok).Format("2/1/2006 15:04")
}
```

แปลงข้างใน `formatTimestamp` -> จุดเรียก (`:420,421,516,517`) ไม่ต้องแก้
คง layout เดิม `"2/1/2006 15:04"` ไว้ (ไม่ zero-pad) เพื่อไม่เปลี่ยนรูปแบบที่ผู้ใช้เห็นอยู่

**document-core** `pdfgen/excel_generator.go:215` `GenerateFilename`
เปลี่ยน `now := time.Now()` -> `now := time.Now().In(bangkok)` โดยประกาศ helper
`bangkok` แบบเดียวกันในไฟล์นี้ (คนละ module กัน แชร์โค้ดไม่ได้)

### F2 Last Updated ใช้เวลาที่ราคาแก้ล่าสุดจริง (P2)

เพิ่ม query เดียวใน `GetPriceExportTable` ก่อนสร้าง tab:

```sql
SELECT MAX(GREATEST(g.update_dtm, COALESCE(sg.update_dtm, g.update_dtm)))
FROM price_list_group g
LEFT JOIN price_list_sub_group sg ON sg.price_list_group_id = g.id
WHERE g.company_code = :company_code
  AND (:site_codes IS NULL OR g.site_code = ANY(:site_codes))
  AND (:group_codes IS NULL OR g.group_code = ANY(:group_codes))
```

ใช้ `sqlxDB` connection เดิมที่เปิดอยู่แล้ว (`:54`)

ส่งค่าเป็นพารามิเตอร์ `lastUpdated *time.Time` เข้า `buildDetailTab` และ `buildBasedPriceTab`:
- `Headers.LastUpdated` = `formatTimestamp(*lastUpdated)` ถ้าไม่ nil, ไม่งั้นเป็น `""`
  (`excel_generator.go:78` ข้ามการเขียนแถวเมื่อค่าว่างอยู่แล้ว)
- `Headers.Download` = `formatTimestamp(time.Now())` ตามเดิม

ถ้า query ล้มเหลว ให้ log warning และปล่อย `LastUpdated` ว่าง ไม่ทำให้ export ทั้งใบพัง
(สอดคล้องกับวิธีจัดการ inventory service ที่ `:145`)

### F3 dedupe คอลัมน์ (P3)

ใน `buildExportTableTyped` เพิ่ม `seen map[string]bool`:
- ใส่ field ของ static columns ลง `seen` หลังสร้างเสร็จ
- loop dynamic group-key columns (`:298`) ข้ามถ้า `seen[c.code]`
- loop dynamic UDF columns (`:326`) ข้ามถ้า `seen[key]`

การ dedupe ไม่กระทบ rows เพราะ row เป็น map ที่ key ซ้ำอยู่แล้วในตัว

### F4 เขียนตัวเลขเป็นตัวเลข (P4)

`excel_generator.go` `extractValue` เปลี่ยน return type `string` -> `interface{}`:

```go
case string:  return v
case float64: return v
case int:     return v
case int64:   return v
case bool:    return v
case time.Time: return v.In(bangkok).Format("2006-01-02 15:04:05")
default: return fmt.Sprintf("%v", v)
}
```

`f.SetCellValue` (`:163`) รับ `interface{}` อยู่แล้ว ไม่ต้องแก้จุดเรียก
logic ทำสี (`:126-160`) อ่านค่าจาก `row` โดยตรงไม่ผ่าน `extractValue` จึงไม่กระทบ

### F5 ใช้ filename จาก template_config (P5)

`/module-report-buttons/list` **คืน `template_config` มาให้อยู่แล้ว**
(`internal/models/module_report_buttons.go:16` มี json tag ปกติ)
มีแค่ TS type ที่ไม่ประกาศไว้ จึงแก้ฝั่ง frontend อย่างเดียว ไม่ต้องแตะ backend

1. `src/types/document/document-buttons.type.ts:9`
   ลบคอมเมนต์ `// template_config excluded as per requirement`
   แล้วเพิ่ม `template_config?: string;`

2. `src/utils/helper/priceListExport.ts`
   เปลี่ยน return เป็น `{ blob: Blob; filename: string }`
   โดย parse `exportButton.template_config` เอา `filename` มาต่อเป็น
   `<filename>_<yyyyMMdd_HHmmss>.xlsx` (timestamp เวลาไทยจาก browser)
   ถ้า parse ไม่ได้หรือไม่มี `filename` ให้ fallback เป็นชื่อเดิม
   `base_price_<company>_<site>.xlsx`

3. `src/views/price-list/BasePrice.vue:146-150`
   ใช้ `filename` ที่ helper คืนมาแทนชื่อ hardcode

**ไม่แตะ `useAxios.ts`** — การแก้ interceptor ให้คืน header ด้วยกระทบทุก caller ทั้งแอป
ซึ่งเกินขอบเขตงานนี้

### F6 แยกหัวคอลัมน์ "จำนวน" (P6)

`get-price-export-table.go:267-268`
- `stock_quantity` -> HeaderName `"จำนวนรวม"`
- `quantity` -> HeaderName `"จำนวน"` (คงเดิม)

เก็บทั้งสองคอลัมน์ไว้ตามเดิม เปลี่ยนเฉพาะป้ายหัวเพื่อให้ผู้ใช้แยกออก

### F7 ตัด is_highlight ออกจากไฟล์ (P7)

ลบ `{Field: "is_highlight", HeaderName: "Highlight สีฟ้า"}` (`:290`) ออกจาก static columns
และให้ `seen["is_highlight"] = true` เพื่อกัน UDF loop เติมกลับเข้ามา

ค่า `is_highlight` ยังต้องคงอยู่ใน `row` เพราะ `excel_generator.go:120` ใช้ทำสีตัวอักษร
— ตัดแค่คอลัมน์ ไม่ตัดข้อมูล

## ขอบเขตที่ไม่ทำ

- ไม่แก้ `useAxios.ts` interceptor (กระทบทั้งแอป)
- ไม่แก้ Dockerfile / deploy config เรื่อง TZ — ตัดสินใจแก้ที่ระดับโค้ดเพื่อให้ผลลัพธ์
  ไม่ขึ้นกับ environment ที่รัน
- ไม่แตะ `InventoryWeight[0]` ที่หยิบแค่รายการแรก (`:157`) — เป็นพฤติกรรมที่ตั้งใจไว้
  ตามคอมเมนต์ "For export, use first inventory record per subgroup"
- ไม่แตะ handler ปุ่มอื่นที่ใช้ `ExcelExport` ร่วมกัน (`CUSTOMIZE/CTM-CTM6`) —
  แต่ต้องรับรู้ว่าการแก้ F1/F4 มีผลกับปุ่มนั้นด้วย ซึ่งเป็นการแก้ในทางที่ถูก

## ผลกระทบข้ามปุ่ม

`CUSTOMIZE/CTM-CTM6` ใช้ `ExcelExport` เส้นเดียวกัน (`excel_dispatch.go:8`)
และ `INVENTORY/INV-CYCLE` (`cycle_count_export.go`) ใช้ `ExcelGenerator` ร่วมกัน
-> F1 (filename timezone) และ F4 (ตัวเลขไม่เป็น text) จะมีผลกับสองปุ่มนี้ด้วย
ต้อง smoke test ทั้งสองปุ่มก่อน merge

## การทดสอบ (ตาม CLAUDE.md coverage >= 80%)

### erp-core

`internal/services/price-service/get-price-export-table_test.go` (มีอยู่แล้ว) เพิ่ม:
- `formatTimestamp` แปลง UTC -> ICT ถูกต้อง (input `2026-09-03T02:30:00Z` -> `"3/9/2026 09:30"`)
- `formatTimestamp` ข้ามวัน (input `2026-09-02T20:00:00Z` -> `"3/9/2026 03:00"`)
- `buildExportTableTyped` ไม่มี field ซ้ำใน columns เมื่อ input มีทั้ง group key `PG01`
  และ UDF key ที่ชนกับ static column
- `buildExportTableTyped` ไม่มีคอลัมน์ `is_highlight` แต่ row ยังมี key `is_highlight`
- `stock_quantity` / `quantity` มี HeaderName ต่างกัน
- `buildDetailTab` / `buildBasedPriceTab` ใช้ `lastUpdated` ที่รับเข้ามา ไม่ใช่ `time.Now()`
- `lastUpdated == nil` -> `Headers.LastUpdated == ""`

Integration test ของ query `MAX(GREATEST(...))` ใช้ testcontainers ตาม pattern
`make test-integration` ที่มีอยู่ (`upload-pricelist_integration_test.go`)

### document-core

`pdfgen/excel_generator_test.go` (ไฟล์ใหม่):
- `extractValue` คืน `float64` สำหรับตัวเลข, `bool` สำหรับ bool, `""` สำหรับ nil/ไม่มี key
- `extractValue` แปลง `time.Time` เป็นเวลาไทย
- `GenerateFilename` ใช้ `template_config.filename` และ timestamp เป็นเวลาไทย
- `GenerateFilename` fallback เป็น ButtonName และ `"export"` ตามลำดับ
- `GenerateExcelFromExportResponse` เขียน cell ราคาเป็น numeric ไม่ใช่ string
  (อ่านกลับด้วย `f.GetCellType` == `CellTypeNumber`)

### web

ไม่มี test runner ตั้งไว้ -> ทดสอบด้วยมือ:
- กดปุ่ม Export Excel แล้วได้ไฟล์ชื่อ `TMI_Pricelist_Report_<timestamp>.xlsx`
- เปิดไฟล์แล้ว `Last Updated` / `Download` เป็นเวลาไทย
- คอลัมน์ไม่ซ้ำ, ราคา sum ได้ใน Excel
- `yarn build` ผ่าน (vue-tsc)

## เกณฑ์ความสำเร็จ

1. `go test ./...` ผ่านทั้ง erp-core และ document-core, coverage แพ็กเกจที่แก้ >= 80%
2. `yarn build` ของ web ผ่าน
3. Export จริงแล้วเวลาใน Excel ตรงกับเวลาไทยที่กดปุ่ม
4. `Last Updated` แสดงเวลาแก้ราคาล่าสุด ไม่เท่ากับ `Download` (เว้นแต่เพิ่งแก้ราคา)
5. ไม่มีหัวคอลัมน์ซ้ำในชีต Detail
6. ราคาและจำนวนในชีตเป็นตัวเลขที่ sum ได้
7. ชื่อไฟล์มาจาก `template_config.filename`
8. ปุ่ม `CUSTOMIZE/CTM-CTM6` และ `INVENTORY/INV-CYCLE` ยัง export ได้ปกติ
