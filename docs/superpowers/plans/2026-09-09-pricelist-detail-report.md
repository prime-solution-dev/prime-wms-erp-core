# Pricelist Detail Report Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** เพิ่ม report ตัวที่สอง "Pricelist Detail Report" ที่หน้า Base Price โดยเปลี่ยนปุ่ม Print เป็น dropdown 2 ตัวเลือก และให้ตัวใหม่กรองด้วย Product Group 1 ได้

**Architecture:** ขยาย pipeline export เดิมด้วย flag `report_type` ที่ไหลจาก frontend → document-core (`c.Decode`) → erp-core `/price/GetPriceExportTable` เมื่อ flag เป็น `PRICELIST_DETAIL` service คืน `Tabs` เดียวชื่อ `Template` ที่ประกอบคอลัมน์ตาม spec แทน 2 tabs เดิม excel generator ฝั่ง document-core ไม่ต้องแก้เลยเพราะเขียน metadata แถว 1-3 + header แถว 5 อยู่แล้ว

**Tech Stack:** Go 1.25 + Gin + GORM/sqlx (erp-core, document-core), excelize (document-core), Vue 3 + TypeScript + Ant Design Vue (prime-wms-web)

**Spec:** `prime-wms-erp-core/docs/superpowers/specs/2026-09-09-pricelist-detail-report-design.md`

---

## Repo / Branch

งานกระจาย 3 repo แต่ละ repo ต้องแตก branch ใหม่จาก `Develop` ชื่อ `feat/pricelist-detail-report`

| Repo | สถานะตอนเริ่ม |
|---|---|
| `prime-wms-erp-core` | branch `feat/pricelist-detail-report` สร้างแล้ว (มี spec commit อยู่) |
| `prime-wms-document-core` | ค้างบน `fix/ib-gr-warehouse-name-from-item` — ต้อง checkout Develop ก่อน |
| `prime-wms-web` | อยู่บน `Develop` |

## File Structure

**prime-wms-erp-core**
- Modify `internal/services/price-service/get-pricelist.go` — เพิ่ม `ReportType` ใน `GetPriceListGroupRequest`
- Create `internal/services/price-service/build-pricelist-detail-tab.go` — สร้าง tab `Template` (แยกไฟล์เพราะ `get-price-export-table.go` ยาว ~500 บรรทัดแล้ว)
- Create `internal/services/price-service/build-pricelist-detail-tab_test.go` — unit test
- Modify `internal/services/price-service/get-price-export-table.go` — แตกทางตาม `ReportType`
- Modify `internal/repositories/priceList/repository.go` — เพิ่ม `GetFormulasBySubgroupCodes`
- Create `internal/repositories/priceList/formulas_integration_test.go` — integration test

**prime-wms-document-core**
- Modify `external/services/erp/get-price-export-table.go` — เพิ่ม `ReportType` ใน request struct
- Create `scripts/migrations/data/pricelist_detail_report_button.json` — button record ใหม่

**prime-wms-web**
- Modify `src/utils/helper/priceListExport.ts` — รับ `buttonCode` / `groupCodes` / `reportType`
- Create `src/components/priceList/PricelistDetailExportModal.vue` — modal เลือก Product Group 1
- Modify `src/views/price-list/BasePrice.vue` — Print เป็น dropdown + ต่อ modal

---

## Task 1: เพิ่ม `ReportType` ใน request struct ของ erp-core

**Files:**
- Modify: `internal/services/price-service/get-pricelist.go` (struct `GetPriceListGroupRequest`)

- [ ] **Step 1: หา struct แล้วเพิ่ม field**

หา `type GetPriceListGroupRequest struct` ใน `internal/services/price-service/get-pricelist.go` แล้วเพิ่มบรรทัด `ReportType` เข้าไปเป็น field สุดท้าย:

```go
type GetPriceListGroupRequest struct {
	CompanyCode       string     `json:"company_code"`
	SiteCodes         []string   `json:"site_codes"`
	GroupCodes        []string   `json:"group_codes"`
	EffectiveDateFrom *time.Time `json:"effective_date_from"`
	EffectiveDateTo   *time.Time `json:"effective_date_to"`
	SubGroupCodes     []string   `json:"sub_group_codes"` // TODO: อาจจะต้อง filter ละเอียดขึ้น หรือ แยกเส้น
	// ReportType เลือกรูปแบบ tab ที่ GetPriceExportTable คืน
	// ค่าว่าง = พฤติกรรมเดิม (Detail + Based price), "PRICELIST_DETAIL" = tab Template ตัวเดียว
	ReportType string `json:"report_type"`
}
```

- [ ] **Step 2: ประกาศค่าคงที่ของ report type**

เพิ่มไว้ใต้ struct ในไฟล์เดียวกัน:

```go
// ReportTypePricelistDetail คือค่า report_type ที่ทำให้ GetPriceExportTable
// คืน tab "Template" ตัวเดียวตามรูปแบบ Pricelist Detail Report
const ReportTypePricelistDetail = "PRICELIST_DETAIL"
```

- [ ] **Step 3: ยืนยันว่ายัง compile ผ่าน**

```bash
cd prime-wms-erp-core && go build ./...
```
Expected: ไม่มี output (สำเร็จ)

- [ ] **Step 4: Commit**

```bash
git add internal/services/price-service/get-pricelist.go
git commit -m "feat(price): add report_type to GetPriceListGroupRequest"
```

---

## Task 2: repository ดึง formula ต่อ subgroup

**Files:**
- Modify: `internal/repositories/priceList/repository.go`
- Test: `internal/repositories/priceList/formulas_integration_test.go`

บริบท: ตาราง `price_list_subgroup_formulas_map` มี FK `price_list_subgroup_code` → `price_list_sub_group.subgroup_code`
และ `price_list_formulas_code` → `price_list_formulas.formula_code` ส่วนตาราง `price_list_formulas`
มีคอลัมน์ `uom` ที่เป็น `kg` หรือ `pcs` ใช้แยกว่าสูตรนั้นเป็นราคาต่อกิโลกรัมหรือต่อหน่วย

หมายเหตุ: struct `models.PriceListSubGroupFormulasMap` ที่มีอยู่ประกาศ gorm `references:SubgroupKey`
ซึ่งไม่ตรงกับ FK จริงใน SQL (`subgroup_code`) จึง**ไม่ใช้ Preload ของ struct นั้น** แต่เขียน join เองเพื่อความถูกต้อง

- [ ] **Step 1: เขียน integration test ที่ยังไม่ผ่าน**

สร้าง `internal/repositories/priceList/formulas_integration_test.go`:

```go
//go:build integration

package priceListRepository

import (
	"testing"
)

func TestGetFormulasBySubgroupCodes(t *testing.T) {
	// ใช้ DB จาก env เดียวกับ integration test ตัวอื่นใน package นี้
	// (ดู test ตัวที่มีอยู่แล้วในโฟลเดอร์นี้เพื่อ reuse helper setup ถ้ามี)
	setupFormulaFixtures(t)

	got, err := GetFormulasBySubgroupCodes([]string{"SG01", "SG02"})
	if err != nil {
		t.Fatalf("GetFormulasBySubgroupCodes error: %v", err)
	}

	sg01 := got["SG01"]
	if len(sg01) != 2 {
		t.Fatalf("expected 2 formulas for SG01, got %d", len(sg01))
	}

	var kg, pcs *SubgroupFormula
	for i := range sg01 {
		switch sg01[i].Uom {
		case "kg":
			kg = &sg01[i]
		case "pcs":
			pcs = &sg01[i]
		}
	}
	if kg == nil {
		t.Fatal("expected a kg formula for SG01")
	}
	if kg.FormulaCode != "FM-8" {
		t.Fatalf("expected FM-8, got %q", kg.FormulaCode)
	}
	if kg.Name != "kg = Base price + Extra" {
		t.Fatalf("unexpected kg formula name: %q", kg.Name)
	}
	if pcs == nil {
		t.Fatal("expected a pcs formula for SG01")
	}
	if pcs.FormulaCode != "FM-7" {
		t.Fatalf("expected FM-7, got %q", pcs.FormulaCode)
	}

	if _, ok := got["SG99"]; ok {
		t.Fatal("expected no entry for a subgroup code that was not requested")
	}
}

func TestGetFormulasBySubgroupCodes_EmptyInput(t *testing.T) {
	got, err := GetFormulasBySubgroupCodes(nil)
	if err != nil {
		t.Fatalf("expected no error for empty input, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(got))
	}
}
```

`setupFormulaFixtures` ต้อง insert แถวเหล่านี้ (เขียนใน test file เดียวกัน ใช้ connection แบบเดียวกับ
integration test ที่มีอยู่แล้วใน package — เปิดไฟล์ `*_integration_test.go` ตัวเดิมในโฟลเดอร์นี้เพื่อ copy pattern การ setup):

```sql
INSERT INTO price_list_formulas (id, formula_code, name, uom, formula_type, expression, params, rounding, create_dtm)
VALUES
  (gen_random_uuid(), 'FM-8', 'kg = Base price + Extra', 'kg', 'price_calc', '', '{}', 2, now()),
  (gen_random_uuid(), 'FM-7', 'Pcs = [kg]  x [Avg. kg stock]', 'pcs', 'price_calc', '', '{}', 2, now());

INSERT INTO price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
VALUES
  (gen_random_uuid(), 'SG01', 'FM-8', true, now()),
  (gen_random_uuid(), 'SG01', 'FM-7', true, now()),
  (gen_random_uuid(), 'SG02', 'FM-8', true, now());
```

- [ ] **Step 2: รัน test ให้เห็นว่าไม่ผ่าน**

```bash
cd prime-wms-erp-core && make test-integration
```
Expected: FAIL — `undefined: GetFormulasBySubgroupCodes` และ `undefined: SubgroupFormula`

- [ ] **Step 3: เขียน implementation**

เพิ่มท้ายไฟล์ `internal/repositories/priceList/repository.go`:

```go
// SubgroupFormula คือสูตรราคา 1 ตัวที่ผูกกับ price_list_sub_group หนึ่งแถว
type SubgroupFormula struct {
	SubgroupCode string `gorm:"column:subgroup_code"`
	FormulaCode  string `gorm:"column:formula_code"`
	Name         string `gorm:"column:name"`
	Uom          string `gorm:"column:uom"`
}

// GetFormulasBySubgroupCodes คืนสูตรราคาของทุก subgroup ที่ขอมาในคิวรีเดียว
// key ของ map คือ subgroup_code — subgroup ที่ไม่มีสูตรจะไม่มี key อยู่ใน map
//
// เขียน join เองแทนการใช้ Preload ของ models.PriceListSubGroupFormulasMap เพราะ
// gorm tag ของ struct นั้นอ้าง references:SubgroupKey ซึ่งไม่ตรงกับ FK จริงในฐานข้อมูล
func GetFormulasBySubgroupCodes(codes []string) (map[string][]SubgroupFormula, error) {
	result := map[string][]SubgroupFormula{}
	if len(codes) == 0 {
		return result, nil
	}

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return nil, err
	}
	defer db.CloseGORM(gormx)

	rows := []SubgroupFormula{}
	if err := gormx.
		Table("price_list_subgroup_formulas_map AS m").
		Select("m.price_list_subgroup_code AS subgroup_code, f.formula_code, f.name, f.uom").
		Joins("JOIN price_list_formulas AS f ON f.formula_code = m.price_list_formulas_code").
		Where("m.price_list_subgroup_code IN ?", codes).
		Order("m.price_list_subgroup_code, f.uom, f.formula_code").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.SubgroupCode] = append(result[r.SubgroupCode], r)
	}
	return result, nil
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd prime-wms-erp-core && make test-integration
```
Expected: PASS ทั้ง `TestGetFormulasBySubgroupCodes` และ `TestGetFormulasBySubgroupCodes_EmptyInput`

- [ ] **Step 5: Commit**

```bash
git add internal/repositories/priceList/repository.go internal/repositories/priceList/formulas_integration_test.go
git commit -m "feat(price): add GetFormulasBySubgroupCodes repository query"
```

---

## Task 3: สร้าง tab `Template` (unit test ก่อน)

**Files:**
- Create: `internal/services/price-service/build-pricelist-detail-tab.go`
- Test: `internal/services/price-service/build-pricelist-detail-tab_test.go`

บริบทสำคัญ: `buildExportTableTyped` ที่มีอยู่ (`get-price-export-table.go`) รับ `groupNameByCode` และ
`itemNameByCode` เป็น **closure** ไม่ใช่ map — ฟังก์ชันใหม่ใช้ signature เดียวกันเพื่อให้ทดสอบได้โดยไม่ต้องต่อ DB

ลำดับคอลัมน์ที่ต้องได้ (ตามชีท `Template` ของไฟล์ตัวอย่าง):

1. `pricelist_group_name`
2. คอลัมน์ชื่อกลุ่ม (dynamic) — field = group code เช่น `PG01`, header = `groupNameByCode(code)`, ค่า = `itemNameByCode(value)`
3. `total_weight`, `avg_weight`, `price_per_kg`, `price_per_unit`, `extra_price`
4. `formula_kg_name`, `formula_unit_name`
5. `pricelist_group_code`
6. คอลัมน์รหัสกลุ่ม (dynamic) — field = `<code>__code` เช่น `PG01__code`, header = code ดิบ, ค่า = `GroupKey.Value`
7. `formula_kg_code`, `formula_unit_code`
8. `subgroup_code`

- [ ] **Step 1: เขียน failing test**

สร้าง `internal/services/price-service/build-pricelist-detail-tab_test.go`:

```go
package priceService

import (
	"encoding/json"
	"testing"
	"time"

	"prime-erp-core/internal/models"
	priceListRepository "prime-erp-core/internal/repositories/priceList"

	"github.com/google/uuid"
)

func detailTestFixtures() ([]GetPriceListGroupResponse, func(string) string, func(string) string) {
	udf, _ := json.Marshal(map[string]interface{}{"inactive": false})

	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:        uuid.New(),
				GroupCode: "GROUP_1_ITEM_1",
				SubGroups: []SubGroup{
					{
						ID:                  uuid.New(),
						SubgroupCode:        "SG01",
						SubGroupKey:         "PG01_3|PG02_6|PG06_4",
						TotalNetPriceWeight: 18.34,
						TotalNetPriceUnit:   1230,
						ExtraPriceWeight:    1,
						UdfJson:             udf,
						GroupKeys: []GroupKey{
							{Code: "PG01", Value: "PG01_3", Seq: 1},
							{Code: "PG02", Value: "PG02_6", Seq: 2},
							{Code: "PG06", Value: "PG06_4", Seq: 6},
						},
						InventoryWeight: []models.InventoryWeightResponse{
							{TotalWeight: 120.45, AvgWeight: 0},
						},
					},
					{
						ID:           uuid.New(),
						SubgroupCode: "SG02",
						SubGroupKey:  "PG01_3|PG04_9",
						UdfJson:      udf,
						GroupKeys: []GroupKey{
							{Code: "PG01", Value: "PG01_3", Seq: 1},
							{Code: "PG04", Value: "PG04_9", Seq: 4},
						},
					},
				},
			},
		},
	}

	groupNameByCode := func(code string) string {
		switch code {
		case "PG01":
			return "หมวดหลัก"
		case "PG02":
			return "หมวดย่อย"
		case "PG04":
			return "ขนาด"
		case "PG06":
			return "หนา"
		default:
			return ""
		}
	}

	itemNameByCode := func(code string) string {
		switch code {
		case "GROUP_1_ITEM_1":
			return "หมวดเหล็กแผ่น"
		case "PG01_3":
			return "หมวดเหล็กแผ่น"
		case "PG02_6":
			return "เหล็กแผ่น"
		case "PG04_9":
			return "4' x 8'"
		case "PG06_4":
			return "1.2"
		default:
			return ""
		}
	}

	return groups, groupNameByCode, itemNameByCode
}

func columnIndex(cols []ExportColumn, field string) int {
	for i := range cols {
		if cols[i].Field == field {
			return i
		}
	}
	return -1
}

func TestBuildPricelistDetailTab_TabShape(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()
	updated := time.Date(2026, 9, 7, 10, 13, 0, 0, time.UTC)

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, &updated)

	if tab.Name != "Template" {
		t.Fatalf("expected tab name Template, got %q", tab.Name)
	}
	if tab.Headers.Report != "Pricelist Detail" {
		t.Fatalf("unexpected report header: %q", tab.Headers.Report)
	}
	if tab.Headers.LastUpdated == "" {
		t.Fatal("expected a last updated header")
	}
	if tab.Headers.Download == "" {
		t.Fatal("expected a download header")
	}
}

func TestBuildPricelistDetailTab_DynamicGroupColumns(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, nil)

	// PG04 ปรากฏเฉพาะใน subgroup ที่สอง ต้องยังมีคอลัมน์ให้
	for _, code := range []string{"PG01", "PG02", "PG04", "PG06"} {
		if columnIndex(tab.Columns, code) == -1 {
			t.Fatalf("expected a %s name column", code)
		}
		if columnIndex(tab.Columns, code+"__code") == -1 {
			t.Fatalf("expected a %s__code column", code)
		}
	}

	// header ของคอลัมน์ชื่อมาจาก DB ส่วนคอลัมน์รหัสใช้ code ดิบ
	nameCol := tab.Columns[columnIndex(tab.Columns, "PG06")]
	if nameCol.HeaderName != "หนา" {
		t.Fatalf("expected PG06 header หนา, got %q", nameCol.HeaderName)
	}
	codeCol := tab.Columns[columnIndex(tab.Columns, "PG06__code")]
	if codeCol.HeaderName != "PG06" {
		t.Fatalf("expected PG06__code header PG06, got %q", codeCol.HeaderName)
	}

	// เรียงตาม seq: PG01 มาก่อน PG06 เสมอ
	if columnIndex(tab.Columns, "PG01") > columnIndex(tab.Columns, "PG06") {
		t.Fatal("expected group columns ordered by seq")
	}
}

func TestBuildPricelistDetailTab_ColumnOrder(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, nil)

	if columnIndex(tab.Columns, "pricelist_group_name") != 0 {
		t.Fatal("expected pricelist_group_name to be the first column")
	}
	if columnIndex(tab.Columns, "subgroup_code") != len(tab.Columns)-1 {
		t.Fatal("expected subgroup_code to be the last column")
	}

	ordered := []string{
		"PG01",
		"total_weight",
		"avg_weight",
		"price_per_kg",
		"price_per_unit",
		"extra_price",
		"formula_kg_name",
		"formula_unit_name",
		"pricelist_group_code",
		"PG01__code",
		"formula_kg_code",
		"formula_unit_code",
		"subgroup_code",
	}
	for i := 1; i < len(ordered); i++ {
		prev, cur := columnIndex(tab.Columns, ordered[i-1]), columnIndex(tab.Columns, ordered[i])
		if prev == -1 {
			t.Fatalf("missing column %s", ordered[i-1])
		}
		if cur == -1 {
			t.Fatalf("missing column %s", ordered[i])
		}
		if prev > cur {
			t.Fatalf("expected %s before %s", ordered[i-1], ordered[i])
		}
	}
}

func TestBuildPricelistDetailTab_RowValues(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, nil)

	if len(tab.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tab.Rows))
	}
	row := tab.Rows[0]

	if row["pricelist_group_name"] != "หมวดเหล็กแผ่น" {
		t.Fatalf("unexpected group name: %v", row["pricelist_group_name"])
	}
	if row["pricelist_group_code"] != "GROUP_1_ITEM_1" {
		t.Fatalf("unexpected group code: %v", row["pricelist_group_code"])
	}
	if row["subgroup_code"] != "SG01" {
		t.Fatalf("unexpected subgroup code: %v", row["subgroup_code"])
	}
	if row["PG01"] != "หมวดเหล็กแผ่น" {
		t.Fatalf("expected PG01 to hold the item name, got %v", row["PG01"])
	}
	if row["PG01__code"] != "PG01_3" {
		t.Fatalf("expected PG01__code to hold the raw code, got %v", row["PG01__code"])
	}
	if row["price_per_kg"] != 18.34 {
		t.Fatalf("unexpected price_per_kg: %v", row["price_per_kg"])
	}
	if row["price_per_unit"] != float64(1230) {
		t.Fatalf("unexpected price_per_unit: %v", row["price_per_unit"])
	}
	if row["extra_price"] != float64(1) {
		t.Fatalf("unexpected extra_price: %v", row["extra_price"])
	}
	if row["total_weight"] != 120.45 {
		t.Fatalf("unexpected total_weight: %v", row["total_weight"])
	}
	if row["avg_weight"] != float64(0) {
		t.Fatalf("unexpected avg_weight: %v", row["avg_weight"])
	}

	// subgroup ที่ไม่มี PG06 ต้องได้เซลล์ว่าง ไม่ใช่ค่าของแถวอื่น
	second := tab.Rows[1]
	if v, ok := second["PG06"]; ok && v != "" {
		t.Fatalf("expected empty PG06 for the second row, got %v", v)
	}
}

func TestBuildPricelistDetailTab_FormulaMapping(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()

	formulas := map[string][]priceListRepository.SubgroupFormula{
		"SG01": {
			{SubgroupCode: "SG01", FormulaCode: "FM-8", Name: "kg = Base price + Extra", Uom: "kg"},
			{SubgroupCode: "SG01", FormulaCode: "FM-7", Name: "Pcs = [kg]  x [Avg. kg stock]", Uom: "pcs"},
		},
	}

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, formulas, nil)

	row := tab.Rows[0]
	if row["formula_kg_name"] != "kg = Base price + Extra" {
		t.Fatalf("unexpected formula_kg_name: %v", row["formula_kg_name"])
	}
	if row["formula_kg_code"] != "FM-8" {
		t.Fatalf("unexpected formula_kg_code: %v", row["formula_kg_code"])
	}
	if row["formula_unit_name"] != "Pcs = [kg]  x [Avg. kg stock]" {
		t.Fatalf("unexpected formula_unit_name: %v", row["formula_unit_name"])
	}
	if row["formula_unit_code"] != "FM-7" {
		t.Fatalf("unexpected formula_unit_code: %v", row["formula_unit_code"])
	}

	// SG02 ไม่มีสูตร ต้องได้ค่าว่าง ไม่ใช่ error และไม่ใช่ค่าของ SG01
	second := tab.Rows[1]
	if second["formula_kg_name"] != "" {
		t.Fatalf("expected empty formula for SG02, got %v", second["formula_kg_name"])
	}
	if second["formula_unit_code"] != "" {
		t.Fatalf("expected empty formula code for SG02, got %v", second["formula_unit_code"])
	}
}

func TestBuildPricelistDetailTab_FallbackToRawCode(t *testing.T) {
	groups, _, _ := detailTestFixtures()

	// resolve ชื่อไม่ได้เลย — ต้อง fallback เป็น code ดิบ ไม่ใช่เซลล์ว่าง
	none := func(string) string { return "" }
	tab := buildPricelistDetailTab(groups, none, none, nil, nil)

	nameCol := tab.Columns[columnIndex(tab.Columns, "PG01")]
	if nameCol.HeaderName != "PG01" {
		t.Fatalf("expected header to fall back to the code, got %q", nameCol.HeaderName)
	}
	if tab.Rows[0]["PG01"] != "PG01_3" {
		t.Fatalf("expected value to fall back to the raw code, got %v", tab.Rows[0]["PG01"])
	}
	if tab.Rows[0]["pricelist_group_name"] != "GROUP_1_ITEM_1" {
		t.Fatalf("expected group name to fall back to the code, got %v", tab.Rows[0]["pricelist_group_name"])
	}
}

func TestBuildPricelistDetailTab_SkipsInactive(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()
	inactive, _ := json.Marshal(map[string]interface{}{"inactive": true})
	groups[0].SubGroups[1].UdfJson = inactive

	tab := buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, nil, nil)

	if len(tab.Rows) != 1 {
		t.Fatalf("expected inactive subgroups to be skipped, got %d rows", len(tab.Rows))
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าไม่ผ่าน**

```bash
cd prime-wms-erp-core && go test ./internal/services/price-service/ -run TestBuildPricelistDetailTab -v
```
Expected: FAIL — `undefined: buildPricelistDetailTab`

- [ ] **Step 3: เขียน implementation**

สร้าง `internal/services/price-service/build-pricelist-detail-tab.go`:

```go
package priceService

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	priceListRepository "prime-erp-core/internal/repositories/priceList"
)

// groupCodeColumnSuffix ต่อท้าย field ของคอลัมน์ที่เก็บ "รหัส" กลุ่ม
// เพื่อไม่ให้ชนกับคอลัมน์ที่เก็บ "ชื่อ" กลุ่มซึ่งใช้ field เป็น group code ตรง ๆ
const groupCodeColumnSuffix = "__code"

// buildPricelistDetailTab ประกอบ tab "Template" ของ Pricelist Detail Report
//
// คอลัมน์กลุ่มสินค้าไม่ได้ hardcode ไว้ — เก็บจาก GroupKey.Code ที่พบจริงในข้อมูล
// หัวคอลัมน์มาจากตาราง group ผ่าน groupNameByCode และค่าในเซลล์มาจากตาราง group_item
// ผ่าน itemNameByCode ทั้งคู่ fallback เป็นรหัสดิบเมื่อ resolve ไม่ได้
//
// formulas เป็น map จาก subgroup_code ไปยังสูตรของ subgroup นั้น ส่งค่า nil ได้
// เมื่อไม่ต้องการเติมคอลัมน์สูตร (เช่นในเทสต์)
func buildPricelistDetailTab(
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
	formulas map[string][]priceListRepository.SubgroupFormula,
	lastUpdated *time.Time,
) ExportTab {
	type colMeta struct {
		code   string
		name   string
		minSeq int
	}

	// เก็บ group code ทุกตัวที่พบ พร้อม seq ต่ำสุดไว้เรียงลำดับให้คงที่
	colMap := map[string]*colMeta{}
	for _, g := range groups {
		for _, sg := range g.SubGroups {
			for _, k := range sg.GroupKeys {
				if k.Code == "" {
					continue
				}
				name := strings.TrimSpace(groupNameByCode(k.Code))
				if existing, ok := colMap[k.Code]; ok {
					if existing.name == "" && name != "" {
						existing.name = name
					}
					if k.Seq > 0 && (existing.minSeq == 0 || k.Seq < existing.minSeq) {
						existing.minSeq = k.Seq
					}
					continue
				}
				colMap[k.Code] = &colMeta{code: k.Code, name: name, minSeq: k.Seq}
			}
		}
	}

	cols := make([]colMeta, 0, len(colMap))
	for _, m := range colMap {
		cols = append(cols, *m)
	}
	sort.Slice(cols, func(i, j int) bool {
		ai, aj := cols[i], cols[j]
		if ai.minSeq != 0 && aj.minSeq != 0 && ai.minSeq != aj.minSeq {
			return ai.minSeq < aj.minSeq
		}
		if ai.minSeq != 0 && aj.minSeq == 0 {
			return true
		}
		if ai.minSeq == 0 && aj.minSeq != 0 {
			return false
		}
		return ai.code < aj.code
	})

	// ลำดับคอลัมน์ตามชีท Template ของไฟล์ตัวอย่าง
	columns := []ExportColumn{
		{Field: "pricelist_group_name", HeaderName: "Pricelist group name"},
	}
	for _, c := range cols {
		header := c.name
		if header == "" {
			header = c.code
		}
		columns = append(columns, ExportColumn{Field: c.code, HeaderName: header})
	}
	columns = append(columns,
		ExportColumn{Field: "total_weight", HeaderName: "Weight-spec"},
		ExportColumn{Field: "avg_weight", HeaderName: "Avg. kg stock"},
		ExportColumn{Field: "price_per_kg", HeaderName: "Price per kg"},
		ExportColumn{Field: "price_per_unit", HeaderName: "Price per unit"},
		ExportColumn{Field: "extra_price", HeaderName: "Extra price"},
		ExportColumn{Field: "formula_kg_name", HeaderName: "Price per kg formula"},
		ExportColumn{Field: "formula_unit_name", HeaderName: "Price per unit formula"},
		ExportColumn{Field: "pricelist_group_code", HeaderName: "Pricelist group code"},
	)
	for _, c := range cols {
		columns = append(columns, ExportColumn{
			Field:      c.code + groupCodeColumnSuffix,
			HeaderName: c.code,
		})
	}
	columns = append(columns,
		ExportColumn{Field: "formula_kg_code", HeaderName: "Price per kg formula code"},
		ExportColumn{Field: "formula_unit_code", HeaderName: "Price per unit formula code"},
		ExportColumn{Field: "subgroup_code", HeaderName: "subgroup_code"},
	)

	rows := make([]map[string]interface{}, 0)
	for _, g := range groups {
		groupName := itemNameByCode(g.GroupCode)
		if groupName == "" {
			groupName = g.GroupCode
		}

		for _, sg := range g.SubGroups {
			if isInactiveSubGroup(sg.UdfJson) {
				continue
			}

			row := map[string]interface{}{
				"pricelist_group_name": groupName,
				"pricelist_group_code": g.GroupCode,
				"subgroup_code":        sg.SubgroupCode,
				"price_per_kg":         sg.TotalNetPriceWeight,
				"price_per_unit":       sg.TotalNetPriceUnit,
				"extra_price":          sg.ExtraPriceWeight,
				"total_weight":         "",
				"avg_weight":           "",
				"formula_kg_name":      "",
				"formula_kg_code":      "",
				"formula_unit_name":    "",
				"formula_unit_code":    "",
			}

			// เติมเซลล์ว่างให้ทุกคอลัมน์กลุ่มก่อน เพื่อไม่ให้แถวที่ไม่มีกลุ่มนั้น
			// ไปหยิบค่าของแถวก่อนหน้าตอน excelize เขียนไฟล์
			for _, c := range cols {
				row[c.code] = ""
				row[c.code+groupCodeColumnSuffix] = ""
			}
			for _, k := range sg.GroupKeys {
				if k.Code == "" {
					continue
				}
				name := itemNameByCode(k.Value)
				if name == "" {
					name = k.Value
				}
				row[k.Code] = name
				row[k.Code+groupCodeColumnSuffix] = k.Value
			}

			if len(sg.InventoryWeight) > 0 {
				inv := sg.InventoryWeight[0]
				row["total_weight"] = inv.TotalWeight
				row["avg_weight"] = inv.AvgWeight
			}

			for _, f := range formulas[sg.SubgroupCode] {
				switch f.Uom {
				case "kg":
					row["formula_kg_name"] = f.Name
					row["formula_kg_code"] = f.FormulaCode
				case "pcs":
					row["formula_unit_name"] = f.Name
					row["formula_unit_code"] = f.FormulaCode
				}
			}

			rows = append(rows, row)
		}
	}

	return ExportTab{
		Name: "Template",
		Headers: ExportTabHeaders{
			Report:      "Pricelist Detail",
			LastUpdated: formatOptionalTimestamp(lastUpdated),
			Download:    formatTimestamp(time.Now()),
		},
		Columns: columns,
		Rows:    rows,
	}
}

// isInactiveSubGroup อ่าน flag inactive จาก udf_json — แถวที่ inactive ไม่ถูก export
// เหมือนพฤติกรรมของ buildExportTableTyped
func isInactiveSubGroup(udfJson json.RawMessage) bool {
	if len(udfJson) == 0 {
		return false
	}
	udfData := map[string]interface{}{}
	if err := json.Unmarshal(udfJson, &udfData); err != nil {
		return false
	}
	val, _ := udfData["inactive"].(bool)
	return val
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd prime-wms-erp-core && go test ./internal/services/price-service/ -run TestBuildPricelistDetailTab -v
```
Expected: PASS ทั้ง 7 เทสต์

- [ ] **Step 5: รัน test ทั้ง package กันของเดิมพัง**

```bash
cd prime-wms-erp-core && go test ./internal/services/price-service/
```
Expected: ok

- [ ] **Step 6: Commit**

```bash
git add internal/services/price-service/build-pricelist-detail-tab.go internal/services/price-service/build-pricelist-detail-tab_test.go
git commit -m "feat(price): build Template tab for Pricelist Detail Report"
```

---

## Task 4: ต่อ `buildPricelistDetailTab` เข้ากับ `GetPriceExportTable`

**Files:**
- Modify: `internal/services/price-service/get-price-export-table.go`
- Test: `internal/services/price-service/build-pricelist-detail-tab_test.go` (เพิ่มเทสต์)

- [ ] **Step 1: เขียน failing regression test**

เพิ่มท้าย `build-pricelist-detail-tab_test.go`:

```go
func TestSelectExportTabs_DefaultKeepsTwoTabs(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()

	tabs := selectExportTabs("", groups, groupNameByCode, itemNameByCode, nil, map[string]GetPaymentTermResponse{}, nil)

	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs for the default report type, got %d", len(tabs))
	}
	if tabs[0].Name != "Detail" {
		t.Fatalf("expected first tab Detail, got %q", tabs[0].Name)
	}
	if tabs[1].Name != "Based price" {
		t.Fatalf("expected second tab Based price, got %q", tabs[1].Name)
	}
}

func TestSelectExportTabs_PricelistDetailReturnsSingleTab(t *testing.T) {
	groups, groupNameByCode, itemNameByCode := detailTestFixtures()

	tabs := selectExportTabs(ReportTypePricelistDetail, groups, groupNameByCode, itemNameByCode, nil, map[string]GetPaymentTermResponse{}, nil)

	if len(tabs) != 1 {
		t.Fatalf("expected exactly 1 tab, got %d", len(tabs))
	}
	if tabs[0].Name != "Template" {
		t.Fatalf("expected the Template tab, got %q", tabs[0].Name)
	}
}
```

- [ ] **Step 2: รัน test ให้เห็นว่าไม่ผ่าน**

```bash
cd prime-wms-erp-core && go test ./internal/services/price-service/ -run TestSelectExportTabs -v
```
Expected: FAIL — `undefined: selectExportTabs`

- [ ] **Step 3: เพิ่มฟังก์ชัน `selectExportTabs`**

เพิ่มท้าย `internal/services/price-service/build-pricelist-detail-tab.go`:

```go
// selectExportTabs เลือกชุด tab ตาม report type
// แยกออกมาเป็นฟังก์ชันเดี่ยวเพื่อให้ทดสอบได้โดยไม่ต้องต่อ DB
func selectExportTabs(
	reportType string,
	groups []GetPriceListGroupResponse,
	groupNameByCode func(code string) string,
	itemNameByCode func(code string) string,
	formulas map[string][]priceListRepository.SubgroupFormula,
	paymentTermMap map[string]GetPaymentTermResponse,
	lastUpdated *time.Time,
) []ExportTab {
	if reportType == ReportTypePricelistDetail {
		return []ExportTab{
			buildPricelistDetailTab(groups, groupNameByCode, itemNameByCode, formulas, lastUpdated),
		}
	}
	return []ExportTab{
		buildDetailTab(groups, groupNameByCode, itemNameByCode, lastUpdated),
		buildBasedPriceTab(groups, paymentTermMap, lastUpdated),
	}
}
```

- [ ] **Step 4: รัน test ให้ผ่าน**

```bash
cd prime-wms-erp-core && go test ./internal/services/price-service/ -run TestSelectExportTabs -v
```
Expected: PASS

- [ ] **Step 5: แก้ `GetPriceExportTable` ให้ใช้ `selectExportTabs`**

ใน `internal/services/price-service/get-price-export-table.go`:

5a. ลบบล็อกที่สร้าง `detailTab` ทิ้ง (บล็อกที่ขึ้นต้นด้วยคอมเมนต์ `// Build "Detail" tab (existing functionality).`
จนถึงวงเล็บปิดของ `buildDetailTab(...)`) เพราะจะย้ายไปสร้างทีหลังพร้อมกันทั้งสองแบบ

5b. หลังบล็อก `getGroupAndItemMappings()` ให้ประกาศ closure สองตัวไว้ใช้ซ้ำ:

```go
	groupNameByCode := func(code string) string {
		if g, ok := groupMap[code]; ok {
			return g.GroupName
		}
		return ""
	}
	itemNameByCode := func(code string) string {
		if it, ok := groupItemMap[code]; ok {
			return it.ItemName
		}
		return ""
	}
```

5c. แทนที่ 4 บรรทัดท้ายฟังก์ชัน (ตั้งแต่ `basedPriceTab := buildBasedPriceTab(...)` จนถึง `return response, nil`) ด้วย:

```go
	// สูตรราคาต้องใช้เฉพาะ Pricelist Detail Report — ไม่ยิงคิวรีเพิ่มให้ report เดิม
	var formulas map[string][]priceListRepository.SubgroupFormula
	if req.ReportType == ReportTypePricelistDetail {
		subgroupCodes := []string{}
		for _, resp := range res {
			for _, sg := range resp.SubGroups {
				if sg.SubgroupCode != "" {
					subgroupCodes = append(subgroupCodes, sg.SubgroupCode)
				}
			}
		}
		formulas, err = priceListRepository.GetFormulasBySubgroupCodes(subgroupCodes)
		if err != nil {
			// สูตรที่หายไปทำให้เซลล์ว่าง ไม่ควรทำให้ export ทั้งไฟล์ล้ม
			fmt.Printf("Warning: failed to get subgroup formulas: %v\n", err)
			formulas = nil
		}
	}

	response := GetPriceExportTableResponse{
		Tabs: selectExportTabs(
			req.ReportType,
			res,
			groupNameByCode,
			itemNameByCode,
			formulas,
			paymentTermMap,
			lastUpdated,
		),
	}
	return response, nil
```

5d. เพิ่ม import `priceListRepository "prime-erp-core/internal/repositories/priceList"` ที่หัวไฟล์

- [ ] **Step 6: build และรัน test ทั้ง package**

```bash
cd prime-wms-erp-core && go build ./... && go vet ./internal/services/price-service/ && go test ./internal/services/price-service/
```
Expected: build ผ่าน, vet ไม่มี output, test `ok`

- [ ] **Step 7: ตรวจ coverage ของโค้ดใหม่**

```bash
cd prime-wms-erp-core && go test ./internal/services/price-service/ -coverprofile=/tmp/cover.out && go tool cover -func=/tmp/cover.out | grep -i "pricelist-detail\|buildPricelistDetailTab\|selectExportTabs\|isInactiveSubGroup"
```
Expected: ทุกฟังก์ชันใหม่ ≥ 80%

- [ ] **Step 8: Commit**

```bash
git add internal/services/price-service/
git commit -m "feat(price): return the Template tab when report_type is PRICELIST_DETAIL"
```

---

## Task 5: ส่ง `report_type` ผ่าน document-core

**Files:**
- Modify: `prime-wms-document-core/external/services/erp/get-price-export-table.go`

บริบท: `ExcelExport()` เรียก `c.Decode("price export request data", &priceReq)` ซึ่ง marshal `c.Data`
(payload ที่ frontend ส่งมาใน `data`) แล้ว unmarshal ลง struct นี้ — การเพิ่ม field จึงทำให้ค่าไหลผ่านได้เอง
ไม่ต้องแตะ `excel_export.go` หรือ `excel_dispatch.go`

- [ ] **Step 1: แตก branch ใหม่จาก Develop**

```bash
cd prime-wms-document-core
git fetch origin Develop
git checkout -b feat/pricelist-detail-report origin/Develop
git status --short --branch
```
Expected: `## feat/pricelist-detail-report...origin/Develop` และไม่มีไฟล์ค้าง

- [ ] **Step 2: เพิ่ม field**

ใน `external/services/erp/get-price-export-table.go` แก้ struct เป็น:

```go
type GetPriceExportTableRequest struct {
	CompanyCode string   `json:"company_code"`
	SiteCodes   []string `json:"site_codes"`
	GroupCodes  []string `json:"group_codes"`
	// ReportType ไหลจาก payload ของปุ่มไปยัง erp-core ตรง ๆ
	// ค่าว่าง = Pricelist Report เดิม, "PRICELIST_DETAIL" = Pricelist Detail Report
	ReportType string `json:"report_type"`
}
```

- [ ] **Step 3: build และรัน test**

```bash
cd prime-wms-document-core && go build ./... && go test ./...
```
Expected: build ผ่าน, test `ok` (ไม่มีอะไรพัง)

- [ ] **Step 4: Commit**

```bash
git add external/services/erp/get-price-export-table.go
git commit -m "feat(price-export): forward report_type to erp-core"
```

---

## Task 6: เพิ่ม button record ของ report ใหม่

**Files:**
- Create: `prime-wms-document-core/scripts/migrations/data/pricelist_detail_report_button.json`

บริบท: `module_report_button.go:51-52` route ด้วย `button.ButtonAction == "EXPORT_CSV"` → `ExcelDispatch`
→ `ExcelExport` และ `excel_dispatch.go` switch ที่ `ModuleTopicCode + "/" + ModuleItemCode` ซึ่งตรงกับ
`PRICE_LIST/BASE_PRICE` ที่มีอยู่แล้ว ปุ่มใหม่จึงต้องใช้ `button_action` เดิมและ topic/item เดิม
เปลี่ยนแค่ `button_code` — ไม่ต้องแก้โค้ด dispatch เลย

`ButtonCode` มี unique constraint ในตาราง (`gorm:"unique;not null"`) จึงต้องไม่ซ้ำกับปุ่มเดิม

- [ ] **Step 1: สร้างไฟล์ seed**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440021",
  "module_code": "PRICELIST",
  "module_item_code": "BASE_PRICE",
  "module_topic_code": "PRICE_LIST",
  "button_code": "EXPORT_EXCEL_DETAIL",
  "button_name": "Pricelist Detail Report",
  "button_action": "EXPORT_CSV",
  "template_config": {
    "filename": "Pricelist detail Report"
  }
}
```

- [ ] **Step 2: ตรวจ `button_action` ของปุ่มเดิมใน DB ให้ตรงกัน**

ปุ่ม `EXPORT_EXCEL` ปัจจุบันไม่มีไฟล์ seed ในโฟลเดอร์นี้ (ถูกใส่เข้า DB มือ) ต้องยืนยันว่าปุ่มเดิมใช้
`button_action = 'EXPORT_CSV'` จริง ถ้าไม่ตรงให้แก้ไฟล์ข้างบนให้ตรงกับของเดิม:

```bash
psql "$database_gorm_url_prime_wms_document" -c "SELECT button_code, button_action, module_code, module_item_code, module_topic_code FROM module_report_buttons WHERE module_topic_code = 'PRICE_LIST';"
```
Expected: แถวของ `EXPORT_EXCEL` แสดง `button_action` = `EXPORT_CSV`

- [ ] **Step 3: รัน seed**

```bash
cd prime-wms-document-core && make db-migrate-buttons FOLDER=data
```
Expected: ไม่มี error และมีข้อความว่าประมวลผลไฟล์ทั้งหมดในโฟลเดอร์

- [ ] **Step 4: ยืนยันว่า record เข้า DB**

```bash
psql "$database_gorm_url_prime_wms_document" -c "SELECT button_code, button_name, button_action FROM module_report_buttons WHERE button_code = 'EXPORT_EXCEL_DETAIL';"
```
Expected: 1 แถว `EXPORT_EXCEL_DETAIL | Pricelist Detail Report | EXPORT_CSV`

- [ ] **Step 5: Commit**

```bash
git add scripts/migrations/data/pricelist_detail_report_button.json
git commit -m "feat(price-export): seed the Pricelist Detail Report button"
```

---

## Task 7: ให้ helper ฝั่ง web รับ report type และ filter

**Files:**
- Modify: `prime-wms-web/src/utils/helper/priceListExport.ts`

- [ ] **Step 1: แตก branch ใหม่จาก Develop**

```bash
cd prime-wms-web
git fetch origin Develop
git checkout -b feat/pricelist-detail-report origin/Develop
git status --short --branch
```
Expected: `## feat/pricelist-detail-report...origin/Develop`

- [ ] **Step 2: แก้ `callExportPriceExcel`**

แทนที่ฟังก์ชัน `callExportPriceExcel` เดิมทั้งฟังก์ชันด้วย:

```typescript
export interface ExportPriceExcelOptions {
  /** button_code ที่จะเรียก — ค่า default คือปุ่ม Pricelist Report เดิม */
  buttonCode?: string;
  /** report_type ที่ส่งต่อไป erp-core — ค่าว่างคือรายงานเดิม */
  reportType?: string;
  /** group code ของ Product Group 1 ที่เลือก — array ว่างคือ All */
  groupCodes?: string[];
}

export async function callExportPriceExcel(
  companyCode: string,
  siteCode: string,
  options: ExportPriceExcelOptions = {},
): Promise<{ blob: Blob; filename: string }> {
  const {
    buttonCode = 'EXPORT_EXCEL',
    reportType = '',
    groupCodes = [],
  } = options;

  const buttons = await documentApi.getModuleReportButtons({
    module_code: 'PRICELIST',
    module_item_code: 'BASE_PRICE',
    module_topic_code: 'PRICE_LIST',
  });

  const exportButton = buttons.find((b) => b.button_code === buttonCode);
  if (!exportButton) {
    showError(`Export button ${buttonCode} not found`);
    throw new Error(`Export button ${buttonCode} not found`);
  }

  const blob = await documentApi.executeModuleReportAction({
    id: exportButton.id,
    data: {
      company_code: companyCode,
      site_codes: [siteCode],
      group_codes: groupCodes,
      report_type: reportType,
    },
  });

  const filename =
    filenameFromTemplateConfig(exportButton.template_config) ??
    `base_price_${companyCode}_${siteCode}.xlsx`;

  return { blob, filename };
}
```

- [ ] **Step 3: ยืนยันว่า type ผ่าน**

```bash
cd prime-wms-web && yarn build
```
Expected: build สำเร็จ — ผู้เรียกเดิมใน `BasePrice.vue` ส่งแค่ 2 อาร์กิวเมนต์จึงยังคอมไพล์ได้เพราะ `options` มีค่า default

- [ ] **Step 4: Commit**

```bash
git add src/utils/helper/priceListExport.ts
git commit -m "feat(price-list): let the export helper take a button code and group filter"
```

---

## Task 8: modal เลือก Product Group 1

**Files:**
- Create: `prime-wms-web/src/components/priceList/PricelistDetailExportModal.vue`

รายการตัวเลือกมาจาก price list group ที่หน้า Base Price โหลดไว้แล้ว (`PriceList.groupCode` / `PriceList.groupName`)
จึงไม่ต้องยิง API เพิ่ม

- [ ] **Step 1: สร้าง component**

```vue
<template>
  <a-modal
    v-model:open="localVisible"
    centered
    width="480px"
    title="Pricelist Detail Report"
    :confirm-loading="loading"
    ok-text="Export"
    @ok="onExport"
    @cancel="onCancel"
  >
    <div class="py-4">
      <div class="text-[#303030] text-base font-semibold mb-2">
        Product Group 1
      </div>
      <a-select
        v-model:value="selectedGroupCode"
        style="width: 100%"
        :options="groupOptions"
      />
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { PriceList } from '@/types/price-list/price-list.model';

const props = defineProps<{
  open: boolean;
  groups: PriceList[] | null;
  loading?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void;
  /** groupCodes ว่างหมายถึงเลือก All */
  (e: 'export', groupCodes: string[]): void;
}>();

/** ค่าว่างคือตัวเลือก All */
const ALL_GROUPS = '';

const selectedGroupCode = ref<string>(ALL_GROUPS);

const localVisible = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value),
});

const groupOptions = computed(() => {
  const options = [{ label: 'All', value: ALL_GROUPS }];
  for (const group of props.groups ?? []) {
    options.push({
      label: group.groupName || group.groupCode,
      value: group.groupCode,
    });
  }
  return options;
});

// เปิด modal ใหม่ทุกครั้งให้กลับไปที่ All ไม่ค้างค่าที่เลือกรอบก่อน
watch(
  () => props.open,
  (open) => {
    if (open) selectedGroupCode.value = ALL_GROUPS;
  },
);

const onExport = () => {
  emit(
    'export',
    selectedGroupCode.value === ALL_GROUPS ? [] : [selectedGroupCode.value],
  );
};

const onCancel = () => {
  emit('update:open', false);
};
</script>
```

- [ ] **Step 2: ยืนยันว่า type ผ่าน**

```bash
cd prime-wms-web && yarn build
```
Expected: build สำเร็จ

- [ ] **Step 3: Commit**

```bash
git add src/components/priceList/PricelistDetailExportModal.vue
git commit -m "feat(price-list): add the Pricelist Detail Report filter modal"
```

---

## Task 9: เปลี่ยนปุ่ม Print เป็น dropdown

**Files:**
- Modify: `prime-wms-web/src/views/price-list/BasePrice.vue`

- [ ] **Step 1: แทนที่ปุ่ม Print ใน template**

แทน block `<GlobalButton ... @click="onPrintDocument">Print Document</GlobalButton>` ด้วย:

```vue
              <a-dropdown placement="topLeft" :trigger="['click']">
                <GlobalButton
                  type="default"
                  size="large"
                  class="bg-slate-400 text-[#444] w-fit !px-[73px]"
                >
                  Print Document
                </GlobalButton>

                <template #overlay>
                  <a-menu>
                    <a-menu-item key="pricelist" @click="onPrintDocument">
                      Pricelist Report
                    </a-menu-item>
                    <a-menu-item key="pricelist-detail" @click="onOpenDetailExport">
                      Pricelist Detail Report
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
```

- [ ] **Step 2: เพิ่ม modal ใน template**

ใส่ไว้ก่อนปิด `</a-form>`:

```vue
            <PricelistDetailExportModal
              v-model:open="detailExportOpen"
              :groups="priceListData"
              :loading="loading"
              @export="onExportDetail"
            />
```

- [ ] **Step 3: เพิ่ม import และ state ใน `<script setup>`**

เพิ่ม import ต่อจาก import ของ `BasePriceTable`:

```typescript
import PricelistDetailExportModal from '@/components/priceList/PricelistDetailExportModal.vue';
```

เพิ่ม state ต่อจากบรรทัด `const deleteIds = ref<string[]>([]);`:

```typescript
const detailExportOpen = ref<boolean>(false);
```

- [ ] **Step 4: เพิ่ม handler ต่อจาก `onPrintDocument`**

```typescript
const onOpenDetailExport = () => {
  detailExportOpen.value = true;
};

const onExportDetail = async (groupCodes: string[]) => {
  if (!siteStore.current) return;

  try {
    loading.value = true;
    const { blob, filename } = await callExportPriceExcel(
      siteStore.current.companyCode,
      siteStore.current.siteCode,
      {
        buttonCode: 'EXPORT_EXCEL_DETAIL',
        reportType: 'PRICELIST_DETAIL',
        groupCodes,
      }
    );
    downloadBlob(blob, filename);
    detailExportOpen.value = false;
  } catch (error) {
    console.error('Error exporting pricelist detail:', error);
  } finally {
    loading.value = false;
  }
};
```

- [ ] **Step 5: build**

```bash
cd prime-wms-web && yarn build
```
Expected: build สำเร็จ ไม่มี type error

- [ ] **Step 6: Commit**

```bash
git add src/views/price-list/BasePrice.vue
git commit -m "feat(price-list): turn Print Document into a two-report dropdown"
```

---

## Task 10: ตรวจผลลัพธ์จริงแบบ end to end

**Files:** ไม่มีไฟล์ที่ต้องแก้ — เป็นขั้นตอนตรวจสอบ

- [ ] **Step 1: รัน erp-core และ document-core**

```bash
cd prime-wms-erp-core && go run ./cmd
```
อีกเทอร์มินัล:
```bash
cd prime-wms-document-core && make run
```
Expected: erp-core ฟังที่ 9115, document-core ฟังที่ 9106

- [ ] **Step 2: ยิง endpoint ของ erp-core ตรง ๆ ดูรูป response**

```bash
curl -s -X POST http://localhost:9115/price/GetPriceExportTable \
  -H 'Content-Type: application/json' \
  -d '{"company_code":"TMI","site_codes":["MAIN"],"group_codes":[],"report_type":"PRICELIST_DETAIL"}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); t=d['tabs']; print('tabs:',[x['name'] for x in t]); print('cols:',[c['headerName'] for c in t[0]['columns']]); print('rows:',len(t[0]['rows']))"
```
Expected: `tabs: ['Template']`, รายชื่อ header ขึ้นต้นด้วย `Pricelist group name` และลงท้ายด้วย `subgroup_code`

แทน `TMI` / `MAIN` ด้วย company/site จริงในฐานข้อมูลที่ใช้ทดสอบ

- [ ] **Step 3: ยืนยันว่ารายงานเดิมไม่พัง**

```bash
curl -s -X POST http://localhost:9115/price/GetPriceExportTable \
  -H 'Content-Type: application/json' \
  -d '{"company_code":"TMI","site_codes":["MAIN"],"group_codes":[]}' \
  | python3 -c "import sys,json; print([x['name'] for x in json.load(sys.stdin)['tabs']])"
```
Expected: `['Detail', 'Based price']`

- [ ] **Step 4: ทดสอบผ่านหน้าเว็บ**

```bash
cd prime-wms-web && yarn dev
```
เปิดหน้า Price List → Base Price แล้วตรวจ:
1. ปุ่ม Print Document กดแล้วมีเมนู 2 บรรทัด
2. `Pricelist Report` ดาวน์โหลดไฟล์ทันที ไม่มี modal และเนื้อไฟล์เหมือนเดิม (2 ชีท)
3. `Pricelist Detail Report` เปิด modal ที่มี `All` เป็นตัวเลือกแรก
4. เลือก `All` แล้ว Export ได้ไฟล์ที่มีชีทเดียวชื่อ `Template`
5. เลือกกลุ่มเดียวแล้ว Export ได้ไฟล์ที่มีเฉพาะแถวของกลุ่มนั้น

- [ ] **Step 5: เทียบไฟล์ที่ได้กับ template ตัวอย่าง**

```bash
python3 -c "
import openpyxl, sys
got = openpyxl.load_workbook(sys.argv[1])
want = openpyxl.load_workbook('TMI_Pricelist detail Report_20260908.xlsx')
g, w = got['Template'], want['Template']
print('sheets:', got.sheetnames)
print('A1/A2/A3:', g['A1'].value, '|', g['A2'].value, '|', g['A3'].value)
print('got headers :', [c.value for c in g[5] if c.value])
print('want headers:', [c.value for c in w[5] if c.value])
" ~/Downloads/'Pricelist detail Report_*.xlsx'
```
Expected: ชีทเดียวชื่อ `Template`, `A1` = `Report`, `A2` = `Last Updated`, `A3` = `Download`,
header แถว 5 เริ่มด้วย `Pricelist group name` และจบด้วย `subgroup_code`

หมายเหตุ: header ของคอลัมน์รหัสกลุ่มจะเป็น PG code จริงจากฐานข้อมูล ซึ่งอาจไม่ตรงกับ `PG01..PG09`
ที่พิมพ์ไว้ในไฟล์ตัวอย่าง — เป็นพฤติกรรมที่ตั้งใจตาม spec ข้อ 3

- [ ] **Step 6: บันทึกผลการตรวจ**

ถ้าข้อใดไม่ตรง ให้กลับไปแก้ task ที่เกี่ยวข้องก่อนถือว่างานเสร็จ

---

## Task 11: อัปเดตเอกสาร contract ของ document-core

**Files:**
- Modify: `prime-wms-document-core/CONTRACT.md`

`prime-wms-document-core/CLAUDE.md` หัวข้อ 5.1 บังคับว่าทุกครั้งที่แก้ endpoint หรือ payload
ที่ repo อื่นเรียกใช้ ต้องอัปเดต `CONTRACT.md` ในรอบแก้ไขเดียวกัน — งานนี้เพิ่ม field `report_type`
เข้า payload ของ `/module-report-buttons/action` สำหรับปุ่ม `PRICE_LIST/BASE_PRICE`

- [ ] **Step 1: หาหัวข้อของ endpoint นี้ใน CONTRACT.md**

```bash
cd prime-wms-document-core && grep -n "module-report-buttons\|EXPORT_CSV\|BASE_PRICE" CONTRACT.md
```

- [ ] **Step 2: เพิ่มคำอธิบาย field ใหม่ในหัวข้อ 3 ของ CONTRACT.md**

เพิ่มลงในส่วนที่อธิบาย payload ของ `/module-report-buttons/action`:

```markdown
สำหรับปุ่ม `PRICE_LIST/BASE_PRICE` ช่อง `data` รับ field เพิ่มเติม:

| field | type | ความหมาย |
|---|---|---|
| `group_codes` | `string[]` | filter Product Group 1 — array ว่างคือทุกกลุ่ม |
| `report_type` | `string` | `""` = Pricelist Report (2 ชีท), `"PRICELIST_DETAIL"` = Pricelist Detail Report (ชีท `Template` ชีทเดียว) |

ปุ่มที่รองรับ: `EXPORT_EXCEL` (Pricelist Report) และ `EXPORT_EXCEL_DETAIL` (Pricelist Detail Report)
ทั้งคู่ใช้ `button_action = EXPORT_CSV`
```

- [ ] **Step 3: บันทึกใน Version History (หัวข้อ 6)**

```markdown
- 2026-09-09: เพิ่ม `report_type` และ `group_codes` ใน payload ของปุ่ม `PRICE_LIST/BASE_PRICE`
  และเพิ่มปุ่ม `EXPORT_EXCEL_DETAIL` — เป็น backward compatible field ทั้งคู่ (ค่าว่าง = พฤติกรรมเดิม)
```

- [ ] **Step 4: Commit**

```bash
git add CONTRACT.md
git commit -m "docs: document report_type and group_codes on the price list export button"
```

---

## Definition of Done

- [ ] `cd prime-wms-erp-core && go build ./... && go vet ./... && go test ./...` ผ่าน
- [ ] `cd prime-wms-erp-core && make test-integration` ผ่าน
- [ ] coverage ของ `buildPricelistDetailTab`, `selectExportTabs`, `isInactiveSubGroup`, `GetFormulasBySubgroupCodes` ≥ 80%
- [ ] `cd prime-wms-document-core && go build ./... && go test ./...` ผ่าน
- [ ] `cd prime-wms-web && yarn build` ผ่าน
- [ ] Task 10 ตรวจครบทุกข้อ
- [ ] `CONTRACT.md` ของ document-core อัปเดตแล้ว
- [ ] ทั้ง 3 repo อยู่บน branch `feat/pricelist-detail-report` ที่แตกจาก `Develop` และไม่มีใครแตะ Develop ตรง ๆ
- [ ] ไม่มี `TODO`, `test.skip`, `.only`, หรือ stub ค้างในไฟล์ที่แก้
