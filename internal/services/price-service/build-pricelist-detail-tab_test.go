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
		if columnIndex(tab.Columns, code+groupCodeColumnSuffix) == -1 {
			t.Fatalf("expected a %s code column", code)
		}
	}

	// header ของคอลัมน์ชื่อมาจาก DB ส่วนคอลัมน์รหัสใช้ code ดิบ
	nameCol := tab.Columns[columnIndex(tab.Columns, "PG06")]
	if nameCol.HeaderName != "หนา" {
		t.Fatalf("expected PG06 header หนา, got %q", nameCol.HeaderName)
	}
	codeCol := tab.Columns[columnIndex(tab.Columns, "PG06"+groupCodeColumnSuffix)]
	if codeCol.HeaderName != "PG06" {
		t.Fatalf("expected the code column header to be PG06, got %q", codeCol.HeaderName)
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
		"PG01" + groupCodeColumnSuffix,
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
	if row["PG01"+groupCodeColumnSuffix] != "PG01_3" {
		t.Fatalf("expected the code column to hold the raw code, got %v", row["PG01"+groupCodeColumnSuffix])
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

func TestCollectGroupColumns_MergesNameAndKeepsLowestSeq(t *testing.T) {
	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				SubGroups: []SubGroup{
					// แถวแรก resolve ชื่อไม่ได้และ seq สูงกว่า
					{GroupKeys: []GroupKey{{Code: "PG01", Value: "PG01_9", Seq: 7}}},
					// แถวที่สองมีชื่อและ seq ต่ำกว่า ต้องชนะทั้งคู่
					{GroupKeys: []GroupKey{{Code: "PG01", Value: "PG01_3", Seq: 2}}},
					// code ว่างต้องถูกข้าม ไม่กลายเป็นคอลัมน์
					{GroupKeys: []GroupKey{{Code: "", Value: "ignored", Seq: 1}}},
				},
			},
		},
	}

	nameByCode := func(code string) string {
		if code == "PG01" {
			return "  หมวดหลัก  "
		}
		return ""
	}

	cols := collectGroupColumns(groups, nameByCode)

	if len(cols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(cols))
	}
	if cols[0].code != "PG01" {
		t.Fatalf("unexpected code: %q", cols[0].code)
	}
	// ชื่อถูก trim ช่องว่างหัวท้าย
	if cols[0].name != "หมวดหลัก" {
		t.Fatalf("expected the name to be trimmed, got %q", cols[0].name)
	}
	if cols[0].minSeq != 2 {
		t.Fatalf("expected the lowest seq to win, got %d", cols[0].minSeq)
	}
}

func TestCollectGroupColumns_OrdersMissingSeqLast(t *testing.T) {
	groups := []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				SubGroups: []SubGroup{
					{
						GroupKeys: []GroupKey{
							{Code: "ZZ_NO_SEQ", Value: "v1", Seq: 0},
							{Code: "AA_NO_SEQ", Value: "v2", Seq: 0},
							{Code: "HAS_SEQ", Value: "v3", Seq: 5},
						},
					},
				},
			},
		},
	}

	cols := collectGroupColumns(groups, func(string) string { return "" })

	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}
	// คอลัมน์ที่มี seq มาก่อนคอลัมน์ที่ไม่มี seq เสมอ
	if cols[0].code != "HAS_SEQ" {
		t.Fatalf("expected the column with a seq first, got %q", cols[0].code)
	}
	// ที่ไม่มี seq เหมือนกันเรียงตาม code
	if cols[1].code != "AA_NO_SEQ" || cols[2].code != "ZZ_NO_SEQ" {
		t.Fatalf("expected seq-less columns sorted by code, got %q then %q", cols[1].code, cols[2].code)
	}
}

func TestIsInactiveSubGroup(t *testing.T) {
	cases := []struct {
		name string
		udf  json.RawMessage
		want bool
	}{
		{"empty payload", nil, false},
		{"malformed json", json.RawMessage(`{not json`), false},
		{"inactive missing", json.RawMessage(`{"stock":1}`), false},
		{"inactive not a bool", json.RawMessage(`{"inactive":"yes"}`), false},
		{"inactive false", json.RawMessage(`{"inactive":false}`), false},
		{"inactive true", json.RawMessage(`{"inactive":true}`), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isInactiveSubGroup(c.udf); got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}
