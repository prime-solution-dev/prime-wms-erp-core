# Price List Export Excel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ทำให้ไฟล์ Excel ที่ออกจากปุ่ม Export Excel ของ Price List แสดงเวลาเป็นเวลาไทย และมีข้อมูลถูกต้อง (ไม่มีคอลัมน์ซ้ำ ตัวเลขเป็นตัวเลข ชื่อไฟล์มาจาก config)

**Architecture:** งานกระจายอยู่ 3 service ที่คุยกันผ่าน HTTP — `prime-wms-erp-core` ประกอบข้อมูลเป็น Tabs/Columns/Rows, `prime-wms-document-core` แปลงเป็นไฟล์ xlsx, `prime-wms-web` เป็นคนกดปุ่มและตั้งชื่อไฟล์ ทั้งสามไม่แชร์ Go module กัน จึงต้องประกาศ timezone helper แยกกันในแต่ละ repo Task 1-5 อยู่ที่ erp-core, Task 6-7 อยู่ที่ document-core, Task 8 อยู่ที่ web แต่ละกลุ่มไม่ขึ้นต่อกัน ทำสลับลำดับได้

**Tech Stack:** Go 1.24 (erp-core) / Go 1.25 (document-core), sqlx + lib/pq, excelize/v2, Vue 3 + TypeScript

**Spec:** `docs/superpowers/specs/2026-09-03-pricelist-export-excel-design.md`

---

## Worktree

งานทั้งหมดทำใน worktree เท่านั้น branch `feature/pricelist-export-tz` (แตกจาก `origin/Develop`):

```
/home/chonlatee/Desktop/Work/prime-wms/wt-pricelist-export/prime-wms-erp-core
/home/chonlatee/Desktop/Work/prime-wms/wt-pricelist-export/prime-wms-document-core
/home/chonlatee/Desktop/Work/prime-wms/wt-pricelist-export/prime-wms-web
```

ห้ามแก้ไฟล์ใน `/home/chonlatee/Desktop/Work/prime-wms/prime-wms-*` (working copy เดิมของผู้ใช้ มีงานค้างอยู่)

ทุกคำสั่ง `cd` ในแผนนี้ย่อจาก `WT=/home/chonlatee/Desktop/Work/prime-wms/wt-pricelist-export`

---

## File Structure

### prime-wms-erp-core

| ไฟล์ | หน้าที่ | การเปลี่ยนแปลง |
|---|---|---|
| `internal/services/price-service/get-price-export-table.go` | ประกอบ response ของ export | เพิ่ม `bangkok`, แก้ `formatTimestamp`, dedupe columns, รับ `lastUpdated` |
| `internal/services/price-service/get-price-last-updated.go` | **ใหม่** — query หาเวลาที่ราคาถูกแก้ล่าสุด | สร้างใหม่ แยกไฟล์เพราะเป็นคนละความรับผิดชอบกับการประกอบตาราง |
| `internal/services/price-service/get-price-export-table_test.go` | unit test ของการประกอบตาราง | เพิ่ม test ของ timezone / dedupe / lastUpdated |
| `internal/services/price-service/get-price-last-updated_integration_test.go` | **ใหม่** — integration test ของ query | สร้างใหม่ ใช้ testcontainers ตาม pattern `internal/repositories/priceList/repository_test.go` |
| `Makefile` | คำสั่งรัน test | เพิ่ม target `test-integration-price-export` |

### prime-wms-document-core

| ไฟล์ | หน้าที่ | การเปลี่ยนแปลง |
|---|---|---|
| `internal/services/module-report-buttons-service/pdfgen/excel_generator.go` | แปลง response เป็น xlsx | เพิ่ม `bangkok`, แก้ `GenerateFilename` และ `extractValue` |
| `internal/services/module-report-buttons-service/pdfgen/excel_generator_test.go` | **ใหม่** — unit test ของ generator | สร้างใหม่ |

### prime-wms-web

| ไฟล์ | หน้าที่ | การเปลี่ยนแปลง |
|---|---|---|
| `src/types/document/document-buttons.type.ts` | type ของ response จาก document-core | เพิ่ม `template_config?: string` |
| `src/utils/helper/priceListExport.ts` | เรียก export และคำนวณชื่อไฟล์ | คืน `{ blob, filename }` |
| `src/views/price-list/BasePrice.vue` | หน้าจอ | ใช้ filename ที่ helper คืนมา |

---

## Task 1: Timezone ของ erp-core (F1 ส่วนแรก)

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-export-table.go:524-527`
- Test: `prime-wms-erp-core/internal/services/price-service/get-price-export-table_test.go`

- [ ] **Step 1: เขียน failing test**

เปิด `get-price-export-table_test.go` เพิ่ม `"time"` เข้าไปใน import block ให้เป็น:

```go
import (
	"encoding/json"
	"prime-erp-core/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
)
```

แล้วเพิ่มสองฟังก์ชันนี้ท้ายไฟล์:

```go
// update_dtm ถูกเขียนเป็น UTC (update-pricelist.go:27) และ container เป็น alpine
// ที่ไม่มี tzdata เวลาที่แสดงในรายงานจึงต้องถูกแปลงเป็นเวลาไทยอย่างชัดเจน
func TestFormatTimestamp_ConvertsToBangkok(t *testing.T) {
	got := formatTimestamp(time.Date(2026, 9, 3, 2, 30, 0, 0, time.UTC))
	if got != "3/9/2026 09:30" {
		t.Fatalf("expected %q, got %q", "3/9/2026 09:30", got)
	}
}

// export ก่อน 07:00 น. เวลาไทย ต้องไม่ทำให้วันที่ย้อนไปหนึ่งวัน
func TestFormatTimestamp_CrossesDateBoundary(t *testing.T) {
	got := formatTimestamp(time.Date(2026, 9, 2, 20, 0, 0, 0, time.UTC))
	if got != "3/9/2026 03:00" {
		t.Fatalf("expected %q, got %q", "3/9/2026 03:00", got)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run TestFormatTimestamp -v
```

Expected: FAIL — `expected "3/9/2026 09:30", got "3/9/2026 02:30"` (ผลนี้คงที่ทุกเครื่องเพราะ input เป็น `time.UTC`)

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

ใน `get-price-export-table.go` แทนที่บล็อกเดิม:

```go
// formatTimestamp formats time as "DD/MM/YYYY HH:MM" (Thai date format).
func formatTimestamp(t time.Time) string {
	return t.Format("2/1/2006 15:04")
}
```

ด้วย:

```go
// bangkok คือ timezone ที่ใช้แสดงผลทุกเวลาในรายงาน
// container ของ service นี้เป็น alpine ที่ไม่ได้ติดตั้ง tzdata และไม่ได้ตั้ง ENV TZ
// LoadLocation จึงล้มเหลวได้ ต้องมี FixedZone สำรองไว้เสมอ
var bangkok = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

// formatTimestamp formats time as "DD/MM/YYYY HH:MM" (Thai date format) in Asia/Bangkok.
func formatTimestamp(t time.Time) string {
	return t.In(bangkok).Format("2/1/2006 15:04")
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run TestFormatTimestamp -v
```

Expected: PASS ทั้งสองเคส

- [ ] **Step 5: รัน test ทั้งแพ็กเกจกันของเดิมพัง**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/
```

Expected: `ok  	prime-erp-core/internal/services/price-service`

- [ ] **Step 6: Commit**

```bash
cd $WT/prime-wms-erp-core
git add internal/services/price-service/get-price-export-table.go internal/services/price-service/get-price-export-table_test.go
git commit -m "fix: แสดงเวลาในรายงาน price list เป็นเวลาไทย

formatTimestamp แปลงเป็น Asia/Bangkok ก่อน format เสมอ
มี FixedZone สำรองเพราะ container alpine ไม่มี tzdata

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 2: dedupe คอลัมน์ใน Detail tab (F3)

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-export-table.go:249-333`
- Test: `prime-wms-erp-core/internal/services/price-service/get-price-export-table_test.go`

**บริบท:** `buildExportTableTyped` สร้างคอลัมน์สามรอบ — static list, dynamic group-key (`PG01`..`PG10` ซ้ำกับ static), dynamic UDF key (`line_bundle` / `coil_id` / `inactive` ซ้ำกับ static) โดยไม่เช็กว่าซ้ำ

- [ ] **Step 1: เขียน failing test**

เพิ่มท้าย `get-price-export-table_test.go`:

```go
// group key code และ UDF key ชนกับ static column ได้ ต้องไม่ทำให้คอลัมน์ซ้ำ
func TestBuildExportTableTyped_NoDuplicateColumns(t *testing.T) {
	udfJson, err := json.Marshal(map[string]interface{}{
		"line_bundle": 10,
		"coil_id":     "C-001",
	})
	if err != nil {
		t.Fatalf("failed to marshal udf json: %v", err)
	}

	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "G1",
				SubGroups: []SubGroup{
					{
						ID:      uuid.New(),
						UdfJson: udfJson,
						GroupKeys: []GroupKey{
							{Code: "PG01", Value: "V1", Seq: 1},
							{Code: "PG02", Value: "V2", Seq: 2},
						},
					},
				},
			},
		},
	}

	data := buildExportTableTyped(groups, func(string) string { return "" }, func(string) string { return "" })

	count := map[string]int{}
	for _, c := range data.Columns {
		count[c.Field]++
	}
	for field, n := range count {
		if n > 1 {
			t.Fatalf("column %q appears %d times, expected exactly once", field, n)
		}
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run TestBuildExportTableTyped_NoDuplicateColumns -v
```

Expected: FAIL — `column "PG01" appears 2 times, expected exactly once`

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

ใน `buildExportTableTyped` หลังบล็อก static `columns := []ExportColumn{ ... }` (จบที่บรรทัด `}` ก่อนคอมเมนต์ `// Then add dynamic group key columns`) แทรก:

```go
	// seen กันคอลัมน์ซ้ำ — group key code และ UDF key ชนกับ static column ได้
	seen := make(map[string]bool, len(columns))
	for _, c := range columns {
		seen[c.Field] = true
	}
```

แก้ loop dynamic group key columns จาก:

```go
	for _, c := range cols {
		header := c.name
		if header == "" {
			header = c.code
		}
		columns = append(columns, ExportColumn{Field: c.code, HeaderName: header})
	}
```

เป็น:

```go
	for _, c := range cols {
		if seen[c.code] {
			continue
		}
		header := c.name
		if header == "" {
			header = c.code
		}
		columns = append(columns, ExportColumn{Field: c.code, HeaderName: header})
		seen[c.code] = true
	}
```

แก้ loop dynamic UDF columns จาก:

```go
	for _, key := range udfKeys {
		// Use key as header (can be enhanced with mapping later if needed)
		columns = append(columns, ExportColumn{Field: key, HeaderName: key})
	}
```

เป็น:

```go
	for _, key := range udfKeys {
		if seen[key] {
			continue
		}
		// Use key as header (can be enhanced with mapping later if needed)
		columns = append(columns, ExportColumn{Field: key, HeaderName: key})
		seen[key] = true
	}
```

> `udfKeyMap` ไม่ถูกแตะ — มันคุมว่า row จะได้ค่า UDF ไหนบ้าง ไม่ใช่คอลัมน์

- [ ] **Step 4: ซ่อม assertion เดิมที่ค้างอยู่**

`TestBuildExportTableTyped_ColumnsAndRows` **fail อยู่แล้วบน `origin/Develop` ก่อนงานนี้เริ่ม**
สาเหตุคือมันยัง assert ว่าคอลัมน์แรกคือ `PRODUCT_GROUP1` ทั้งที่ static column `PG01`
ถูกเพิ่มเข้ามาทีหลังและมาก่อนใน list ตัว assertion ล้าสมัย ไม่ใช่โค้ดผิด

เนื่องจาก Task นี้แก้ตรรกะการสร้างคอลัมน์อยู่แล้ว ให้ซ่อม assertion นี้ไปด้วย
แก้บล็อกนี้ใน `get-price-export-table_test.go`:

```go
	if resp.Columns[0].Field != "PRODUCT_GROUP1" || resp.Columns[0].HeaderName != "หมวดหลัก" {
		t.Fatalf("unexpected first column: %+v", resp.Columns[0])
	}
```

เป็น:

```go
	// static column มาก่อน dynamic group key เสมอ จึงเช็กว่ามีคอลัมน์อยู่ ไม่เช็กลำดับ
	var productGroup1 *ExportColumn
	for i := range resp.Columns {
		if resp.Columns[i].Field == "PRODUCT_GROUP1" {
			productGroup1 = &resp.Columns[i]
			break
		}
	}
	if productGroup1 == nil {
		t.Fatal("expected a PRODUCT_GROUP1 column")
	}
	if productGroup1.HeaderName != "หมวดหลัก" {
		t.Fatalf("unexpected PRODUCT_GROUP1 header: %q", productGroup1.HeaderName)
	}
```

ห้ามแตะ assertion อื่นในฟังก์ชันนี้

- [ ] **Step 5: รัน test ให้ผ่าน**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run TestBuildExportTableTyped -v
```

Expected: PASS ทั้ง `_ColumnsAndRows` เดิมและ `_NoDuplicateColumns` ใหม่

> `TestUpdateLatestPriceListSubGroup_WithFormulas` และ `TestUpdateLatestPriceListSubGroup_WithInventoryData`
> ก็ fail อยู่ก่อนแล้วบน Develop เช่นกัน แต่ไม่เกี่ยวกับโค้ดที่แผนนี้แตะ **ปล่อยไว้** และรายงานให้ผู้ใช้ทราบ

- [ ] **Step 6: Commit**

```bash
cd $WT/prime-wms-erp-core
git add internal/services/price-service/get-price-export-table.go internal/services/price-service/get-price-export-table_test.go
git commit -m "fix: ตัดคอลัมน์ซ้ำในชีต Detail ของ price list export

group key (PG01-PG10) และ UDF key ที่ชนกับ static column
ถูกเติมเข้าไปซ้ำโดยไม่เช็ก

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 3: ตัดคอลัมน์ is_highlight และแยกหัวคอลัมน์จำนวน (F6, F7)

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-export-table.go:267-268,290`
- Test: `prime-wms-erp-core/internal/services/price-service/get-price-export-table_test.go`

**บริบท:** `is_highlight` เป็น flag สำหรับทำสีตัวอักษร (`excel_generator.go:158`) ไม่ควรโชว์เป็นคอลัมน์ `true`/`false` แต่ค่าใน row ต้องอยู่ต่อ ส่วน `stock_quantity` กับ `quantity` ใช้ HeaderName `"จำนวน"` เหมือนกันทั้งคู่

- [ ] **Step 1: เขียน failing test**

เพิ่มท้าย `get-price-export-table_test.go`:

```go
// is_highlight ใช้ทำสีตัวอักษรฝั่ง document-core จึงต้องอยู่ใน row แต่ไม่ใช่คอลัมน์
func TestBuildExportTableTyped_DropsIsHighlightColumnButKeepsValue(t *testing.T) {
	udfJson, err := json.Marshal(map[string]interface{}{"is_highlight": true})
	if err != nil {
		t.Fatalf("failed to marshal udf json: %v", err)
	}

	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "G1",
				SubGroups: []SubGroup{
					{ID: uuid.New(), UdfJson: udfJson},
				},
			},
		},
	}

	data := buildExportTableTyped(groups, func(string) string { return "" }, func(string) string { return "" })

	for _, c := range data.Columns {
		if c.Field == "is_highlight" {
			t.Fatal("is_highlight must not be an exported column")
		}
	}

	if len(data.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(data.Rows))
	}
	if data.Rows[0]["is_highlight"] != true {
		t.Fatalf("is_highlight value must survive for cell styling, got %v", data.Rows[0]["is_highlight"])
	}
}

// stock_quantity (TotalQty) กับ quantity (SumQty) เคยใช้หัวเดียวกันคือ "จำนวน"
func TestBuildExportTableTyped_QuantityHeadersAreDistinct(t *testing.T) {
	data := buildExportTableTyped(nil, func(string) string { return "" }, func(string) string { return "" })

	headers := map[string]string{}
	for _, c := range data.Columns {
		headers[c.Field] = c.HeaderName
	}

	if headers["stock_quantity"] != "จำนวนรวม" {
		t.Fatalf("expected stock_quantity header %q, got %q", "จำนวนรวม", headers["stock_quantity"])
	}
	if headers["quantity"] != "จำนวน" {
		t.Fatalf("expected quantity header %q, got %q", "จำนวน", headers["quantity"])
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run 'TestBuildExportTableTyped_(DropsIsHighlight|QuantityHeaders)' -v
```

Expected: FAIL ทั้งสอง — `is_highlight must not be an exported column` และ `expected stock_quantity header "จำนวนรวม", got "จำนวน"`

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

ใน static column list ของ `buildExportTableTyped`:

เปลี่ยน

```go
		{Field: "stock_quantity", HeaderName: "จำนวน"},
```

เป็น

```go
		{Field: "stock_quantity", HeaderName: "จำนวนรวม"},
```

และลบบรรทัดนี้ออกทั้งบรรทัด

```go
		{Field: "is_highlight", HeaderName: "Highlight สีฟ้า"},
```

จากนั้นหลังบล็อก `seen := make(...)` ที่เพิ่มไว้ใน Task 2 ให้เติม:

```go
	// is_highlight เป็น flag สำหรับทำสีตัวอักษร ไม่ใช่คอลัมน์ที่ผู้ใช้ต้องเห็น
	// ต้องกันไว้ที่นี่ ไม่งั้น loop UDF ด้านล่างจะเติมกลับเข้ามา
	seen["is_highlight"] = true
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run TestBuildExportTableTyped -v
```

Expected: PASS ทุกเคสของ `TestBuildExportTableTyped*`

> ทั้งแพ็กเกจจะยังมี `TestUpdateLatestPriceListSubGroup_WithFormulas` และ
> `TestUpdateLatestPriceListSubGroup_WithInventoryData` fail อยู่ ซึ่ง fail มาก่อนงานนี้
> และไม่เกี่ยวกับโค้ดที่แผนนี้แตะ **อย่าพยายามซ่อม** ให้รายงานไว้เฉยๆ

- [ ] **Step 5: Commit**

```bash
cd $WT/prime-wms-erp-core
git add internal/services/price-service/get-price-export-table.go internal/services/price-service/get-price-export-table_test.go
git commit -m "fix: ตัดคอลัมน์ is_highlight และแยกหัวคอลัมน์จำนวนใน price list export

is_highlight เป็น flag ทำสีตัวอักษร ไม่ควรโชว์ true/false เป็นคอลัมน์
stock_quantity เปลี่ยนหัวเป็น จำนวนรวม แยกจาก quantity

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 4: รับ lastUpdated เข้า tab builder (F2 ส่วนแรก)

**Files:**
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-export-table.go:405-424,426-518`
- Test: `prime-wms-erp-core/internal/services/price-service/get-price-export-table_test.go`

**บริบท:** ตอนนี้ `LastUpdated` และ `Download` เป็น `time.Now()` ทั้งคู่ ทำให้ `Last Updated` ไม่ได้บอกว่าราคาถูกแก้ล่าสุดเมื่อไหร่ Task นี้เปลี่ยนแค่ signature ให้รับค่าจากข้างนอก Task 5 จะเป็นคนไป query ค่าจริงมาให้

- [ ] **Step 1: เขียน failing test**

เพิ่มท้าย `get-price-export-table_test.go`:

```go
// Last Updated ต้องเป็นเวลาที่ราคาถูกแก้ล่าสุด ไม่ใช่เวลาที่กด export
func TestBuildDetailTab_UsesProvidedLastUpdated(t *testing.T) {
	lastUpdated := time.Date(2026, 8, 20, 3, 15, 0, 0, time.UTC)

	tab := buildDetailTab(nil, func(string) string { return "" }, func(string) string { return "" }, &lastUpdated)

	if tab.Headers.LastUpdated != "20/8/2026 10:15" {
		t.Fatalf("expected LastUpdated %q, got %q", "20/8/2026 10:15", tab.Headers.LastUpdated)
	}
	if tab.Headers.Download == tab.Headers.LastUpdated {
		t.Fatal("Download must be the export time, not the last-updated time")
	}
}

// ไม่มีข้อมูลราคาเลย -> ปล่อยหัวเรื่องว่าง (excel_generator ข้ามแถวที่ค่าว่าง)
func TestBuildDetailTab_NilLastUpdatedLeavesHeaderEmpty(t *testing.T) {
	tab := buildDetailTab(nil, func(string) string { return "" }, func(string) string { return "" }, nil)

	if tab.Headers.LastUpdated != "" {
		t.Fatalf("expected empty LastUpdated, got %q", tab.Headers.LastUpdated)
	}
}

func TestBuildBasedPriceTab_UsesProvidedLastUpdated(t *testing.T) {
	lastUpdated := time.Date(2026, 8, 20, 3, 15, 0, 0, time.UTC)

	tab := buildBasedPriceTab(nil, map[string]GetPaymentTermResponse{}, &lastUpdated)

	if tab.Headers.LastUpdated != "20/8/2026 10:15" {
		t.Fatalf("expected LastUpdated %q, got %q", "20/8/2026 10:15", tab.Headers.LastUpdated)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -run 'TestBuild(DetailTab|BasedPriceTab)' -v
```

Expected: FAIL ตอน compile — `too many arguments in call to buildDetailTab`

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

ใน `get-price-export-table.go` เพิ่ม helper ถัดจาก `formatTimestamp`:

```go
// formatOptionalTimestamp คืนสตริงว่างเมื่อไม่มีค่า
// excel_generator ฝั่ง document-core ข้ามการเขียนหัวเรื่องที่ค่าว่างอยู่แล้ว
func formatOptionalTimestamp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTimestamp(*t)
}
```

แก้ `buildDetailTab` เป็น:

```go
// buildDetailTab wraps the existing buildExportTableTyped logic and adds headers.
func buildDetailTab(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
	lastUpdated *time.Time,
) ExportTab {
	// Reuse existing logic but get the old response structure.
	oldResponse := buildExportTableTyped(groups, groupNameByCode, itemNameByCode)

	return ExportTab{
		Name: "Detail",
		Headers: ExportTabHeaders{
			Report:      "Pricelist",
			LastUpdated: formatOptionalTimestamp(lastUpdated),
			Download:    formatTimestamp(time.Now()),
		},
		Columns: oldResponse.Columns,
		Rows:    oldResponse.Rows,
	}
}
```

แก้ signature ของ `buildBasedPriceTab` เป็น:

```go
func buildBasedPriceTab(groups []GetPriceListGroupResponse, paymentTermMap map[string]GetPaymentTermResponse, lastUpdated *time.Time) ExportTab {
```

ลบบรรทัด `now := time.Now()` ที่อยู่บรรทัดแรกของฟังก์ชันออก และแก้ ExportTab ที่ return เป็น:

```go
	return ExportTab{
		Name: "Based price",
		Headers: ExportTabHeaders{
			Report:      "Pricelist- Based price",
			LastUpdated: formatOptionalTimestamp(lastUpdated),
			Download:    formatTimestamp(time.Now()),
		},
		Columns: columns,
		Rows:    rows,
	}
```

แก้จุดเรียกใน `GetPriceExportTable` — ชั่วคราวส่ง `nil` ไปก่อน Task 5 จะมาต่อ:

```go
	detailTab := buildDetailTab(
		res,
		func(code string) string {
			if g, ok := groupMap[code]; ok {
				return g.GroupName
			}
			return ""
		},
		func(code string) string {
			if it, ok := groupItemMap[code]; ok {
				return it.ItemName
			}
			return ""
		},
		nil,
	)
```

และ

```go
	basedPriceTab := buildBasedPriceTab(res, paymentTermMap, nil)
```

แก้ test เดิม `TestBuildBasedPriceTab_Structure` ที่บรรทัด `tab := buildBasedPriceTab(groups, paymentTermMap)` เป็น:

```go
	tab := buildBasedPriceTab(groups, paymentTermMap, nil)
```

- [ ] **Step 4: รัน test ทั้งแพ็กเกจให้ผ่าน**

```bash
cd $WT/prime-wms-erp-core && go build ./... && go test ./internal/services/price-service/ -run 'TestFormatTimestamp|TestBuild' -v
```

Expected: build สำเร็จ, PASS ทุกเคสของ `TestFormatTimestamp*` และ `TestBuild*`
(`TestUpdateLatestPriceListSubGroup_*` fail มาก่อนงานนี้ ปล่อยไว้)

- [ ] **Step 5: Commit**

```bash
cd $WT/prime-wms-erp-core
git add internal/services/price-service/get-price-export-table.go internal/services/price-service/get-price-export-table_test.go
git commit -m "refactor: ให้ tab builder ของ price list export รับ lastUpdated จากข้างนอก

แยกเวลาที่ราคาถูกแก้ล่าสุด ออกจากเวลาที่กด export
จุดเรียกยังส่ง nil รอ query ใน commit ถัดไป

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 5: query เวลาที่ราคาถูกแก้ล่าสุด (F2 ส่วนที่สอง)

**Files:**
- Create: `prime-wms-erp-core/internal/services/price-service/get-price-last-updated.go`
- Create: `prime-wms-erp-core/internal/services/price-service/get-price-last-updated_integration_test.go`
- Modify: `prime-wms-erp-core/internal/services/price-service/get-price-export-table.go` (จุดเรียกที่ส่ง `nil` จาก Task 4)
- Modify: `prime-wms-erp-core/Makefile`

**บริบท:** `price_list_group.update_dtm` และ `price_list_sub_group.update_dtm` มีอยู่แล้ว (`internal/models/pricelist.go:26,141`) เขียนด้วย `time.Now().UTC()` ตอน update

- [ ] **Step 1: เขียน integration test ที่ยังล้มเหลว**

สร้าง `internal/services/price-service/get-price-last-updated_integration_test.go`:

```go
//go:build integration

package priceService

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "postgres:16",
		Env:          map[string]string{"POSTGRES_PASSWORD": "test", "POSTGRES_USER": "test", "POSTGRES_DB": "testdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("failed to get mapped port: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	os.Setenv("database_sqlx_url_prime_erp",
		fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mapped.Port()))

	if err := createLastUpdatedSchema(); err != nil {
		fmt.Printf("failed to create schema: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func createLastUpdatedSchema() error {
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		return err
	}
	defer sqlxDB.Close()

	_, err = sqlxDB.Exec(`
		CREATE TABLE IF NOT EXISTS price_list_group (
			id uuid PRIMARY KEY,
			company_code text,
			site_code text,
			group_code text,
			update_dtm timestamptz
		);
		CREATE TABLE IF NOT EXISTS price_list_sub_group (
			id uuid PRIMARY KEY,
			price_list_group_id uuid,
			update_dtm timestamptz
		);
	`)
	return err
}

func TestGetPriceLastUpdated_ReturnsLatestOfGroupAndSubGroup(t *testing.T) {
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlxDB.Close()

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	groupID := uuid.New()
	groupUpdated := time.Date(2026, 8, 20, 3, 0, 0, 0, time.UTC)
	subGroupUpdated := time.Date(2026, 8, 22, 9, 30, 0, 0, time.UTC)

	if _, err := sqlxDB.Exec(
		`INSERT INTO price_list_group (id, company_code, site_code, group_code, update_dtm) VALUES ($1,$2,$3,$4,$5)`,
		groupID, "C1", "S1", "G1", groupUpdated,
	); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	if _, err := sqlxDB.Exec(
		`INSERT INTO price_list_sub_group (id, price_list_group_id, update_dtm) VALUES ($1,$2,$3)`,
		uuid.New(), groupID, subGroupUpdated,
	); err != nil {
		t.Fatalf("insert sub group: %v", err)
	}

	got, err := getPriceLastUpdated(sqlxDB, GetPriceListGroupRequest{CompanyCode: "C1", SiteCodes: []string{"S1"}})
	if err != nil {
		t.Fatalf("getPriceLastUpdated: %v", err)
	}
	if got == nil {
		t.Fatal("expected a timestamp, got nil")
	}
	if !got.UTC().Equal(subGroupUpdated) {
		t.Fatalf("expected %v, got %v", subGroupUpdated, got.UTC())
	}
}

func TestGetPriceLastUpdated_NoMatchingRowsReturnsNil(t *testing.T) {
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlxDB.Close()

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	got, err := getPriceLastUpdated(sqlxDB, GetPriceListGroupRequest{CompanyCode: "NOPE"})
	if err != nil {
		t.Fatalf("getPriceLastUpdated: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// group ที่ยังไม่มี sub group ต้องยังคืน update_dtm ของ group เอง
func TestGetPriceLastUpdated_GroupWithoutSubGroup(t *testing.T) {
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlxDB.Close()

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	groupUpdated := time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC)
	if _, err := sqlxDB.Exec(
		`INSERT INTO price_list_group (id, company_code, site_code, group_code, update_dtm) VALUES ($1,$2,$3,$4,$5)`,
		uuid.New(), "C1", "S1", "G1", groupUpdated,
	); err != nil {
		t.Fatalf("insert group: %v", err)
	}

	got, err := getPriceLastUpdated(sqlxDB, GetPriceListGroupRequest{CompanyCode: "C1"})
	if err != nil {
		t.Fatalf("getPriceLastUpdated: %v", err)
	}
	if got == nil || !got.UTC().Equal(groupUpdated) {
		t.Fatalf("expected %v, got %v", groupUpdated, got)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-erp-core && go test -tags=integration ./internal/services/price-service/ -run TestGetPriceLastUpdated -v
```

Expected: FAIL ตอน compile — `undefined: getPriceLastUpdated`

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

สร้าง `internal/services/price-service/get-price-last-updated.go`:

```go
package priceService

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// getPriceLastUpdated คืนเวลาที่ราคาถูกแก้ล่าสุดในขอบเขตเดียวกับที่ export
// คืน nil เมื่อไม่มีแถวที่ตรงเงื่อนไข ผู้เรียกจะปล่อยหัวเรื่อง Last Updated ให้ว่าง
//
// ใช้ bind parameter แทนการต่อสตริงแบบ getGroupSubGroup เพราะเป็นโค้ดใหม่
// และ cardinality() ทำให้ตัวกรองที่ไม่ได้ส่งมาไม่มีผลกับเงื่อนไข
func getPriceLastUpdated(sqlxDB *sqlx.DB, req GetPriceListGroupRequest) (*time.Time, error) {
	const query = `
		SELECT MAX(GREATEST(g.update_dtm, COALESCE(sg.update_dtm, g.update_dtm)))
		FROM price_list_group g
		LEFT JOIN price_list_sub_group sg ON sg.price_list_group_id = g.id
		WHERE ($1 = '' OR g.company_code = $1)
		  AND (COALESCE(cardinality($2::text[]), 0) = 0 OR g.site_code = ANY($2))
		  AND (COALESCE(cardinality($3::text[]), 0) = 0 OR g.group_code = ANY($3))`

	var result sql.NullTime
	err := sqlxDB.Get(&result, query,
		req.CompanyCode,
		pq.Array(req.SiteCodes),
		pq.Array(req.GroupCodes),
	)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		return nil, nil
	}

	t := result.Time
	return &t, nil
}
```

> `pq.Array(nil)` ให้ `NULL` และ `cardinality(NULL::text[])` คืน `NULL` ไม่ใช่ `0` ทำให้เงื่อนไขทั้งข้อกลายเป็น NULL แล้วกรองทุกแถวทิ้ง `COALESCE(..., 0)` จึงจำเป็น และทำให้ผู้เรียกส่ง slice เป็น nil ได้โดยไม่ต้องแปลงก่อน

- [ ] **Step 4: เชื่อมเข้ากับ GetPriceExportTable**

ใน `get-price-export-table.go` หลังบรรทัด `res, err = getTerms(sqlxDB, res)` และก่อนบล็อก `groupMap, groupItemMap, paymentTermMap, err := ...` ให้แทรก:

```go
	// Last Updated ต้องเป็นเวลาที่ราคาถูกแก้ล่าสุด ไม่ใช่เวลาที่กด export
	// ถ้า query ล้มเหลวไม่ควรทำให้ export ทั้งใบพัง — ปล่อยหัวเรื่องว่างแทน
	// (พฤติกรรมเดียวกับตอน inventory service ล้มเหลวด้านล่าง)
	lastUpdated, err := getPriceLastUpdated(sqlxDB, req)
	if err != nil {
		fmt.Printf("Warning: failed to get price last updated: %v\n", err)
		lastUpdated = nil
	}
```

แล้วเปลี่ยน `nil` สองจุดที่ใส่ไว้ใน Task 4 เป็น `lastUpdated`:

```go
		lastUpdated,
	)
```

```go
	basedPriceTab := buildBasedPriceTab(res, paymentTermMap, lastUpdated)
```

- [ ] **Step 5: รัน integration test ให้ผ่าน**

ต้องมี Docker daemon ทำงานอยู่

```bash
cd $WT/prime-wms-erp-core && go test -tags=integration ./internal/services/price-service/ -run TestGetPriceLastUpdated -v
```

Expected: PASS ทั้งสามเคส

- [ ] **Step 6: รัน unit test และ build**

```bash
cd $WT/prime-wms-erp-core && go build ./... && go vet ./internal/services/price-service/ && go test ./internal/services/price-service/ -run 'TestFormatTimestamp|TestBuild' -v
```

Expected: build ผ่าน, vet ไม่มี output, PASS ทุกเคสของ `TestFormatTimestamp*` และ `TestBuild*`

- [ ] **Step 7: เพิ่ม Makefile target**

ใน `Makefile` แก้บรรทัด `.PHONY` ให้มี target ใหม่ และเพิ่ม target ต่อจาก `test-integration-prepurchase`:

```makefile
test-integration-price-export:
	go test -v -tags=integration ./internal/services/price-service
```

บรรทัด `.PHONY` เปลี่ยนเป็น:

```makefile
.PHONY: test test-integration test-integration-group test-integration-prepurchase test-integration-price-export tidy seed-price-list-test seed-price-list-formulas
```

- [ ] **Step 8: Commit**

```bash
cd $WT/prime-wms-erp-core
git add internal/services/price-service/get-price-last-updated.go internal/services/price-service/get-price-last-updated_integration_test.go internal/services/price-service/get-price-export-table.go Makefile
git commit -m "feat: Last Updated ในรายงาน price list ใช้เวลาที่ราคาถูกแก้ล่าสุดจริง

query MAX(GREATEST(group.update_dtm, sub_group.update_dtm)) ในขอบเขตเดียวกับที่ export
query ล้มเหลวแล้วปล่อยหัวเรื่องว่าง ไม่ทำให้ export ทั้งใบพัง

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 6: Timezone ของชื่อไฟล์ใน document-core (F1 ส่วนที่สอง)

**Files:**
- Modify: `prime-wms-document-core/internal/services/module-report-buttons-service/pdfgen/excel_generator.go:213-237`
- Create: `prime-wms-document-core/internal/services/module-report-buttons-service/pdfgen/excel_generator_test.go`

**บริบท:** คนละ Go module กับ erp-core แชร์ helper ไม่ได้ ต้องประกาศ `bangkok` ซ้ำในไฟล์นี้ pattern เดียวกับ `handlers/picking_enrich.go:127`

- [ ] **Step 1: เขียน failing test**

สร้าง `excel_generator_test.go`:

```go
package pdfgen

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"wms-document-core/internal/models"
)

// ชื่อไฟล์ต้องมี timestamp เป็นเวลาไทย ไม่ใช่ UTC ของ container
func TestGenerateFilename_UsesTemplateConfigAndBangkokTime(t *testing.T) {
	button := &models.ModuleReportButtons{
		ButtonName:     "Export Excel",
		TemplateConfig: json.RawMessage(`{"filename": "TMI_Pricelist_Report"}`),
	}

	got := NewExcelGenerator().GenerateFilename(button)

	if !strings.HasPrefix(got, "TMI_Pricelist_Report_") {
		t.Fatalf("expected filename to start with template config filename, got %q", got)
	}
	if !strings.HasSuffix(got, ".xlsx") {
		t.Fatalf("expected .xlsx suffix, got %q", got)
	}

	stamp := strings.TrimSuffix(strings.TrimPrefix(got, "TMI_Pricelist_Report_"), ".xlsx")
	parsed, err := time.ParseInLocation("20060102_150405", stamp, bangkok)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q: %v", stamp, err)
	}
	if diff := time.Since(parsed); diff < -time.Minute || diff > time.Minute {
		t.Fatalf("filename timestamp %v is not the current Bangkok time (diff %v)", parsed, diff)
	}
}

func TestGenerateFilename_FallsBackToButtonName(t *testing.T) {
	button := &models.ModuleReportButtons{ButtonName: "Export Price List"}

	got := NewExcelGenerator().GenerateFilename(button)

	if !strings.HasPrefix(got, "export-price-list_") {
		t.Fatalf("expected fallback to slugified button name, got %q", got)
	}
}

func TestGenerateFilename_FallsBackToGenericName(t *testing.T) {
	got := NewExcelGenerator().GenerateFilename(&models.ModuleReportButtons{})

	if !strings.HasPrefix(got, "export_") {
		t.Fatalf("expected generic fallback, got %q", got)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-document-core && go test ./internal/services/module-report-buttons-service/pdfgen/ -run TestGenerateFilename -v
```

Expected: FAIL ตอน compile — `undefined: bangkok`

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

ใน `excel_generator.go` เพิ่มถัดจาก `func NewExcelGenerator()`:

```go
// bangkok คือ timezone ที่ใช้แสดงผลทุกเวลาในไฟล์ export
// container ของ service นี้ไม่ได้ตั้ง ENV TZ จึงต้องแปลงเองและมี FixedZone สำรอง
// pattern เดียวกับ bangkokDay ใน handlers/picking_enrich.go
var bangkok = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()
```

ใน `GenerateFilename` เปลี่ยนบรรทัด

```go
	now := time.Now()
```

เป็น

```go
	now := time.Now().In(bangkok)
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd $WT/prime-wms-document-core && go test ./internal/services/module-report-buttons-service/pdfgen/ -run TestGenerateFilename -v
```

Expected: PASS ทั้งสามเคส

- [ ] **Step 5: Commit**

```bash
cd $WT/prime-wms-document-core
git add internal/services/module-report-buttons-service/pdfgen/excel_generator.go internal/services/module-report-buttons-service/pdfgen/excel_generator_test.go
git commit -m "fix: timestamp ในชื่อไฟล์ excel export ใช้เวลาไทย

GenerateFilename เคยใช้ time.Now() ตรงๆ ซึ่งเป็น UTC ใน container
มี FixedZone สำรองตาม pattern เดียวกับ picking_enrich.go

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 7: เขียนตัวเลขลง Excel เป็นตัวเลข (F4)

**Files:**
- Modify: `prime-wms-document-core/internal/services/module-report-buttons-service/pdfgen/excel_generator.go:173-201`
- Test: `prime-wms-document-core/internal/services/module-report-buttons-service/pdfgen/excel_generator_test.go`

**บริบท:** `extractValue` คืน `string` ทุกกรณี ทำให้ราคา น้ำหนัก จำนวน กลายเป็นข้อความใน xlsx ผู้ใช้ sum/sort ไม่ได้ `f.SetCellValue` รับ `interface{}` อยู่แล้ว จุดเรียกไม่ต้องแก้ และ logic ทำสี (`:126-160`) อ่านค่าจาก `row` ตรงๆ ไม่ผ่าน `extractValue` จึงไม่กระทบ

**หมายเหตุผลกระทบข้ามปุ่ม:** `INVENTORY/INV-CYCLE` (`cycle_count_export.go`) ใช้ `ExcelGenerator` ตัวเดียวกัน และ `CUSTOMIZE/CTM-CTM6` ใช้ `ExcelExport` เส้นเดียวกัน — การเปลี่ยนนี้มีผลกับทั้งสอง ต้อง smoke test ตาม Task 9

- [ ] **Step 1: เขียน failing test**

แก้ import block ของ `excel_generator_test.go` ให้เป็น:

```go
import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	erpExternalService "wms-document-core/external/services/erp"
	"wms-document-core/internal/models"

	"github.com/xuri/excelize/v2"
)
```

แล้วเพิ่มท้ายไฟล์:

```go
func TestExtractValue_PreservesNumericAndBoolTypes(t *testing.T) {
	e := NewExcelGenerator()
	row := map[string]interface{}{
		"price":   1234.5,
		"flag":    true,
		"name":    "steel",
		"missing": nil,
	}

	if got := e.extractValue(row, "price"); got != 1234.5 {
		t.Fatalf("expected float64 1234.5, got %#v", got)
	}
	if got := e.extractValue(row, "flag"); got != true {
		t.Fatalf("expected bool true, got %#v", got)
	}
	if got := e.extractValue(row, "name"); got != "steel" {
		t.Fatalf("expected \"steel\", got %#v", got)
	}
	if got := e.extractValue(row, "missing"); got != "" {
		t.Fatalf("expected empty string for nil value, got %#v", got)
	}
	if got := e.extractValue(row, "not_there"); got != "" {
		t.Fatalf("expected empty string for absent key, got %#v", got)
	}
}

func TestExtractValue_FormatsTimeInBangkok(t *testing.T) {
	e := NewExcelGenerator()
	row := map[string]interface{}{"at": time.Date(2026, 9, 3, 2, 30, 0, 0, time.UTC)}

	if got := e.extractValue(row, "at"); got != "2026-09-03 09:30:00" {
		t.Fatalf("expected %q, got %#v", "2026-09-03 09:30:00", got)
	}
}

// ราคาต้องเป็น cell ชนิดตัวเลข ไม่งั้นผู้ใช้ sum ใน Excel ไม่ได้
func TestGenerateExcelFromExportResponse_WritesNumbersAsNumbers(t *testing.T) {
	response := &erpExternalService.GetPriceExportTableResponse{
		Tabs: []erpExternalService.ExportTab{
			{
				Name: "Detail",
				Columns: []erpExternalService.ExportColumn{
					{Field: "total_net_price_weight", HeaderName: "ราคาขาย กก"},
				},
				Rows: []map[string]interface{}{
					{"total_net_price_weight": 1234.5},
				},
			},
		},
	}

	f, err := NewExcelGenerator().GenerateExcelFromExportResponse(response)
	if err != nil {
		t.Fatalf("GenerateExcelFromExportResponse: %v", err)
	}

	// ไม่มี headers -> แถวแรกคือหัวตาราง ค่าอยู่แถวที่ 2
	cellType, err := f.GetCellType("Detail", "A2")
	if err != nil {
		t.Fatalf("GetCellType: %v", err)
	}
	if cellType != excelize.CellTypeNumber {
		t.Fatalf("expected A2 to be a number cell, got %v", cellType)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าล้มเหลว**

```bash
cd $WT/prime-wms-document-core && go test ./internal/services/module-report-buttons-service/pdfgen/ -run 'TestExtractValue|TestGenerateExcelFromExportResponse' -v
```

Expected: FAIL — `expected float64 1234.5, got "1234.5"` และ `expected A2 to be a number cell`

- [ ] **Step 3: เขียนโค้ดให้ผ่าน**

ใน `excel_generator.go` แทนที่ `extractValue` ทั้งฟังก์ชันด้วย:

```go
// extractValue extracts a value from a row map, keeping numeric and bool types
// intact so excelize writes them as real numbers instead of text.
func (e *ExcelGenerator) extractValue(row map[string]interface{}, field string) interface{} {
	value, exists := row[field]
	if !exists || value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		return v
	case int:
		return v
	case int64:
		return v
	case bool:
		return v
	case time.Time:
		return v.In(bangkok).Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}
```

ลบ import `"strconv"` ออกจากหัวไฟล์ถ้าไม่มีที่อื่นใช้แล้ว — ตรวจด้วย:

```bash
cd $WT/prime-wms-document-core && grep -n "strconv\." internal/services/module-report-buttons-service/pdfgen/excel_generator.go
```

ถ้าไม่มีผลลัพธ์ ให้ลบบรรทัด `"strconv"` ออกจาก import block

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd $WT/prime-wms-document-core && go test ./internal/services/module-report-buttons-service/pdfgen/ -v
```

Expected: PASS ทุกเคสในแพ็กเกจ (รวม `TestExtractItemsAcceptsBothSliceShapes` เดิม)

- [ ] **Step 5: build และ vet ทั้ง service**

```bash
cd $WT/prime-wms-document-core && go build ./... && go vet ./internal/services/module-report-buttons-service/... && go test ./...
```

Expected: build ผ่าน, vet ไม่มี output, test ผ่าน

- [ ] **Step 6: Commit**

```bash
cd $WT/prime-wms-document-core
git add internal/services/module-report-buttons-service/pdfgen/excel_generator.go internal/services/module-report-buttons-service/pdfgen/excel_generator_test.go
git commit -m "fix: เขียนตัวเลขลง excel เป็นชนิดตัวเลขไม่ใช่ข้อความ

extractValue เคยแปลงทุกอย่างเป็น string ทำให้ราคาและจำนวนใน xlsx
เป็น text ผู้ใช้ sum และ sort ไม่ได้ time.Time แปลงเป็นเวลาไทยด้วย

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 8: ใช้ชื่อไฟล์จาก template_config (F5)

**Files:**
- Modify: `prime-wms-web/src/types/document/document-buttons.type.ts:1-10`
- Modify: `prime-wms-web/src/utils/helper/priceListExport.ts`
- Modify: `prime-wms-web/src/views/price-list/BasePrice.vue:141-156`

**บริบท:** `/module-report-buttons/list` คืน `template_config` มาให้อยู่แล้ว (`internal/models/module_report_buttons.go:16` มี json tag ปกติ ไม่ได้ใส่ `-`) มีแค่ TS type ที่ไม่ได้ประกาศไว้ จึงแก้ฝั่ง frontend อย่างเดียว ไม่ต้องแตะ backend และไม่แตะ `useAxios.ts` (interceptor คืน `response.data` ทิ้ง header — แก้ตรงนั้นกระทบทุก caller ทั้งแอป)

web ไม่มี test runner ตั้งไว้ จึงยืนยันด้วย `yarn build` (มี `vue-tsc`) แล้วทดสอบด้วยมือ

- [ ] **Step 1: เพิ่ม template_config เข้า type**

แก้ `src/types/document/document-buttons.type.ts` บล็อกแรกจาก:

```typescript
export interface ModuleReportButton {
  id: string;
  module_code: string;
  module_item_code: string;
  module_topic_code: string;
  button_code: string;
  button_name: string;
  button_action: string;
  // template_config excluded as per requirement
}
```

เป็น:

```typescript
export interface ModuleReportButton {
  id: string;
  module_code: string;
  module_item_code: string;
  module_topic_code: string;
  button_code: string;
  button_name: string;
  button_action: string;
  /** JSON string จาก document-core เช่น '{"filename": "TMI_Pricelist_Report"}' */
  template_config?: string;
}
```

- [ ] **Step 2: ให้ helper คืนชื่อไฟล์มาด้วย**

แทนที่ `src/utils/helper/priceListExport.ts` ทั้งไฟล์ด้วย:

```typescript
import { useDocumentApi } from '@/composables/useApi';
import { showError } from '../notification';

const documentApi = useDocumentApi();

/** yyyyMMdd_HHmmss ตามเวลาเครื่องผู้ใช้ รูปแบบเดียวกับที่ document-core ตั้งให้ */
function timestampSuffix(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}` +
    `_${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
  );
}

/** อ่าน filename จาก template_config ของปุ่ม คืน null ถ้าไม่มีหรือ parse ไม่ได้ */
function filenameFromTemplateConfig(templateConfig?: string): string | null {
  if (!templateConfig) return null;
  try {
    const parsed = JSON.parse(templateConfig) as { filename?: string };
    return parsed.filename ? `${parsed.filename}_${timestampSuffix()}.xlsx` : null;
  } catch {
    return null;
  }
}

export async function callExportPriceExcel(
  companyCode: string,
  siteCode: string,
): Promise<{ blob: Blob; filename: string }> {
  // Find the export button configuration
  const buttons = await documentApi.getModuleReportButtons({
    module_code: 'PRICELIST',
    module_item_code: 'BASE_PRICE',
    module_topic_code: 'PRICE_LIST',
  });

  const exportButton = buttons.find((b) => b.button_code === 'EXPORT_EXCEL');
  if (!exportButton) {
    showError('Export Excel button not found');
    throw new Error('Export Excel button not found');
  }

  // Call document action handler for export
  const blob = await documentApi.executeModuleReportAction({
    id: exportButton.id,
    data: {
      company_code: companyCode,
      site_codes: [siteCode],
      // Don't include group_codes to export ALL data
    },
  });

  const filename =
    filenameFromTemplateConfig(exportButton.template_config) ??
    `base_price_${companyCode}_${siteCode}.xlsx`;

  return { blob, filename };
}
```

- [ ] **Step 3: ให้หน้าจอใช้ชื่อไฟล์ที่ได้มา**

ใน `src/views/price-list/BasePrice.vue` แก้ `onPrintDocument` จาก:

```typescript
    const blob = await callExportPriceExcel(
      siteStore.current.companyCode,
      siteStore.current.siteCode
    );
    downloadBlob(blob, `base_price_${siteStore.current.companyCode}_${siteStore.current.siteCode}.xlsx`);
```

เป็น:

```typescript
    const { blob, filename } = await callExportPriceExcel(
      siteStore.current.companyCode,
      siteStore.current.siteCode
    );
    downloadBlob(blob, filename);
```

- [ ] **Step 4: build ให้ผ่าน**

```bash
cd $WT/prime-wms-web && yarn install --frozen-lockfile && yarn build
```

Expected: `vue-tsc` ไม่รายงาน error และ vite build สำเร็จ

- [ ] **Step 5: Commit**

```bash
cd $WT/prime-wms-web
git add src/types/document/document-buttons.type.ts src/utils/helper/priceListExport.ts src/views/price-list/BasePrice.vue
git commit -m "fix: ใช้ชื่อไฟล์จาก template_config ตอน export price list

template_config ถูกส่งมาจาก document-core อยู่แล้ว แต่ TS type ไม่ได้ประกาศไว้
หน้าจอจึง hardcode ชื่อ base_price_... ทับค่าที่ตั้งใน config

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01CJDm4YYEArAiyKLADRMoVE"
```

---

## Task 9: ตรวจ coverage และ smoke test ปุ่มที่ใช้โค้ดร่วมกัน

**Files:** ไม่แก้ไฟล์ ยกเว้นต้องเพิ่ม test ถ้า coverage ไม่ถึงเกณฑ์

**บริบท:** CLAUDE.md กำหนด coverage ไม่ต่ำกว่า 80% และมีสองปุ่มที่ใช้โค้ดที่แก้ร่วมกัน — `CUSTOMIZE/CTM-CTM6` (ใช้ `ExcelExport` เส้นเดียวกันกับ `PRICE_LIST/BASE_PRICE`) และ `INVENTORY/INV-CYCLE` (ใช้ `ExcelGenerator` ตัวเดียวกัน)

- [ ] **Step 1: วัด coverage ของแพ็กเกจที่แก้**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -cover
cd $WT/prime-wms-document-core && go test ./internal/services/module-report-buttons-service/pdfgen/ -cover
```

Expected: บรรทัดท้ายแต่ละคำสั่งแสดง `coverage: NN.N% of statements`

- [ ] **Step 2: ถ้าแพ็กเกจไหนต่ำกว่า 80% ให้หาบรรทัดที่ยังไม่ถูกครอบ**

```bash
cd $WT/prime-wms-erp-core && go test ./internal/services/price-service/ -coverprofile=/tmp/erp.out && go tool cover -func=/tmp/erp.out | grep -E 'formatTimestamp|formatOptionalTimestamp|buildDetailTab|buildBasedPriceTab|buildExportTableTyped|getPriceLastUpdated'
cd $WT/prime-wms-document-core && go test ./internal/services/module-report-buttons-service/pdfgen/ -coverprofile=/tmp/doc.out && go tool cover -func=/tmp/doc.out | grep -E 'extractValue|GenerateFilename|GenerateExcelFromExportResponse'
```

ฟังก์ชันที่แก้ในแผนนี้ต้องขึ้น `100.0%` หรือใกล้เคียง ถ้าฟังก์ชันไหนต่ำ ให้เพิ่ม test ในไฟล์ test ของแพ็กเกจนั้นให้ครอบสาขาที่ขาด แล้ว commit เพิ่ม

> หมายเหตุ: coverage รวมของ `price-service` ทั้งแพ็กเกจจะต่ำเพราะมีโค้ดเดิมที่ไม่มี test อยู่มาก เกณฑ์ 80% ใช้กับโค้ดที่แผนนี้แตะ ไม่ใช่โค้ดเดิมทั้งแพ็กเกจ — ถ้าผู้ใช้ต้องการทั้งแพ็กเกจให้ยกเป็นงานแยก

- [ ] **Step 3: smoke test ด้วยมือ — ปุ่ม Export Excel ของ Price List**

รัน erp-core และ document-core จาก worktree แล้วชี้ web ไปที่สอง service นี้

```bash
cd $WT/prime-wms-erp-core && go run ./cmd
```

```bash
cd $WT/prime-wms-document-core && go run ./cmd
```

```bash
cd $WT/prime-wms-web && yarn dev
```

เปิดหน้า Price List > Base Price แล้วกดปุ่ม Export Excel ตรวจว่า:
- ชื่อไฟล์ขึ้นต้นด้วย `TMI_Pricelist_Report_` และลงท้าย `.xlsx`
- ช่อง `Last Updated` และ `Download` ในชีต Detail เป็นเวลาไทย (ตรงกับนาฬิกาตอนกดปุ่ม)
- `Last Updated` ไม่เท่ากับ `Download` (ยกเว้นเพิ่งแก้ราคา)
- ไม่มีหัวคอลัมน์ซ้ำในชีต Detail
- ไม่มีคอลัมน์ `Highlight สีฟ้า`
- มีคอลัมน์ `จำนวนรวม` และ `จำนวน` แยกกัน
- คลิกช่องราคาแล้ว Excel มองเป็นตัวเลข (คลุมหลายช่องแล้วเห็นผลรวมที่ status bar)
- แถวที่เคยไฮไลต์ยังมีสีตัวอักษรตามเดิม (แดง/เขียว/ฟ้า)

- [ ] **Step 4: smoke test ปุ่มที่ใช้โค้ดร่วมกัน**

- ปุ่ม export ของ `CUSTOMIZE/CTM-CTM6` — ต้องยัง download ได้และไฟล์เปิดได้
- ปุ่ม export ของ `INVENTORY/INV-CYCLE` (Cycle Count) — ต้องยัง download ได้และไฟล์เปิดได้

ถ้าปุ่มใดพัง ให้หยุดและรายงาน อย่าแก้โดยการ revert เฉพาะจุด — สาเหตุน่าจะมาจาก `extractValue` ที่เปลี่ยน return type

- [ ] **Step 5: บันทึกผลและ push**

```bash
cd $WT/prime-wms-erp-core && git log --oneline origin/Develop..HEAD && git push -u origin feature/pricelist-export-tz
cd $WT/prime-wms-document-core && git log --oneline origin/Develop..HEAD && git push -u origin feature/pricelist-export-tz
cd $WT/prime-wms-web && git log --oneline origin/Develop..HEAD && git push -u origin feature/pricelist-export-tz
```

Expected: แต่ละ repo push ขึ้น branch `feature/pricelist-export-tz` สำเร็จ

> ห้าม merge เข้า `Develop` เอง — เปิด PR ให้ผู้ใช้ review

---

## เกณฑ์ความสำเร็จ (ตรวจครบก่อนปิดงาน)

- [ ] `go test ./...` ของ document-core ผ่านทั้งหมด
- [ ] erp-core: `go test ./internal/services/price-service/ -run 'TestFormatTimestamp|TestBuild'` ผ่านทั้งหมด
  (`TestUpdateLatestPriceListSubGroup_WithFormulas` และ `_WithInventoryData` fail อยู่ก่อนงานนี้บน Develop — ไม่ใช่ขอบเขตของแผนนี้)
- [ ] `go test -tags=integration ./internal/services/price-service` ผ่าน (ต้องมี Docker)
- [ ] `yarn build` ของ web ผ่าน
- [ ] Export จริงแล้วเวลาใน Excel ตรงกับเวลาไทยที่กดปุ่ม
- [ ] `Last Updated` แสดงเวลาแก้ราคาล่าสุด ไม่เท่ากับ `Download`
- [ ] ไม่มีหัวคอลัมน์ซ้ำในชีต Detail
- [ ] ไม่มีคอลัมน์ `Highlight สีฟ้า` แต่สีตัวอักษรยังทำงาน
- [ ] `จำนวนรวม` กับ `จำนวน` แยกหัวกัน
- [ ] ราคาและจำนวนในชีตเป็นตัวเลขที่ sum ได้
- [ ] ชื่อไฟล์เป็น `TMI_Pricelist_Report_<timestamp>.xlsx`
- [ ] ปุ่ม `CUSTOMIZE/CTM-CTM6` และ `INVENTORY/INV-CYCLE` ยัง export ได้ปกติ
- [ ] ทั้ง 3 repo push ขึ้น `feature/pricelist-export-tz` แล้ว ยังไม่ merge เข้า Develop
