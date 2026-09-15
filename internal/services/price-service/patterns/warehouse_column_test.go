package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

func sgWarehouse(id, batch, warehouse string) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:            id,
		BatchNo:       batch,
		WarehouseCode: warehouse,
	}
}

func whCodes(subGroups []models.PriceListSubGroupResponse) []string {
	out := make([]string, 0, len(subGroups))
	for _, sg := range subGroups {
		out = append(out, sg.BatchNo+"/"+sg.WarehouseCode)
	}
	return out
}

// pattern ที่แสดงคลังแต่ไม่แสดง batch (ITEM_3/5/12/14-20) ต้องได้ 1 แถวต่อคลัง
// ไม่ใช่ 1 แถวต่อ site ไม่งั้นของที่อยู่คลังอื่นหายไปจากตารางเงียบ ๆ
func TestCollapseSubGroupRowsSplitsByWarehouseWhenNotPerBatch(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWarehouse("a", "B1", "07"),
		sgWarehouse("a", "B1", "09"),
		sgWarehouse("a", "B2", "07"),
		sgWarehouse("a", "B2", "09"),
	}

	got := collapseSubGroupRows(in, false, true)

	want := []string{"B1/07", "B1/09"}
	if !equalStrings(whCodes(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v (1 แถวต่อคลัง เก็บ record แรกของคลังนั้น)", whCodes(got), want)
	}
}

// pattern ที่แสดง batch แต่ไม่แสดงคลัง ต้องยุบคลังทิ้ง ไม่งั้นได้แถวซ้ำที่หน้าตา
// เหมือนกันทุกช่องเพราะไม่มีคอลัมน์ไหนแยกมันออกจากกัน
func TestCollapseSubGroupRowsCollapsesWarehouseWhenNotShown(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWarehouse("a", "B1", "07"),
		sgWarehouse("a", "B1", "09"),
		sgWarehouse("a", "B2", "07"),
	}

	got := collapseSubGroupRows(in, true, false)

	want := []string{"B1/07", "B2/07"}
	if !equalStrings(whCodes(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", whCodes(got), want)
	}
}

// pattern ที่แสดงทั้ง batch และคลัง (ITEM_7/8/22) ต้องได้ทุกแถวครบ
func TestCollapseSubGroupRowsKeepsEveryRowWhenBothShown(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWarehouse("a", "B1", "07"),
		sgWarehouse("a", "B1", "09"),
		sgWarehouse("a", "B2", "07"),
	}

	got := collapseSubGroupRows(in, true, true)

	want := []string{"B1/07", "B1/09", "B2/07"}
	if !equalStrings(whCodes(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", whCodes(got), want)
	}
}

// pattern ที่ไม่แสดงทั้งคู่ต้องเหลือแถวเดียวต่อ sub_group ตามพฤติกรรมเดิม
func TestCollapseSubGroupRowsKeepsOneRowWhenNeitherShown(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWarehouse("a", "B1", "07"),
		sgWarehouse("a", "B2", "09"),
		sgWarehouse("b", "B1", "07"),
	}

	got := collapseSubGroupRows(in, false, false)

	if want := []string{"a", "b"}; !equalStrings(ids(got), want) {
		t.Fatalf("ได้ %v ต้องเป็น %v", ids(got), want)
	}
}

// batch_no มี "|" อยู่จริงได้ (ดู aggregateInventoryWeights) คีย์ที่ใช้ยุบแถวจึงต้อง
// ไม่ชนกันเมื่อ batch กับรหัสคลังต่อกันแล้วบังเอิญได้ string เดียวกัน
func TestCollapseSubGroupRowsKeyDoesNotCollideOnSeparator(t *testing.T) {
	in := []models.PriceListSubGroupResponse{
		sgWarehouse("a", "B|1", "07"),
		sgWarehouse("a", "B", "1|07"),
	}

	got := collapseSubGroupRows(in, true, true)

	if len(got) != 2 {
		t.Fatalf("สองแถวนี้ต่างกันจริง ต้องไม่ถูกยุบ ได้ %d แถว", len(got))
	}
}

func TestWarehouseForRowPrefersUdfThenSubGroup(t *testing.T) {
	sg := models.PriceListSubGroupResponse{WarehouseCode: "07"}

	if got := warehouseForRow(sg, "99"); got != "99" {
		t.Errorf("ค่าที่บันทึกไว้ใน udf_json ต้องชนะ ได้ %#v", got)
	}
	if got := warehouseForRow(sg, nil); got != "07" {
		t.Errorf("ไม่มี udf ต้อง fallback ไปคลังจาก inventory ได้ %#v", got)
	}
	// ต้องเป็น nil ไม่ใช่ "" เพื่อให้เป็นช่องว่าง ไม่ใช่ค่าที่ดูเหมือนตั้งใจ
	if got := warehouseForRow(models.PriceListSubGroupResponse{}, nil); got != nil {
		t.Errorf("ไม่มีข้อมูลเลยต้องได้ nil ได้ %#v", got)
	}
}

// pattern จริงทุกตัวที่แสดงคลัง ต้องถูกตรวจเจอ ไม่งั้นแถวจะไม่ถูกแตกตามคลัง
// แล้วผู้ใช้เห็นคลังเดียวโดยไม่รู้ว่าของอยู่ที่อื่นด้วย
func TestPatternHasWarehouseColumnMatchesRealConfigs(t *testing.T) {
	withWarehouse := []string{
		"GROUP_1_ITEM_3", "GROUP_1_ITEM_5", "GROUP_1_ITEM_7", "GROUP_1_ITEM_8",
		"GROUP_1_ITEM_12", "GROUP_1_ITEM_14", "GROUP_1_ITEM_15", "GROUP_1_ITEM_16",
		"GROUP_1_ITEM_17", "GROUP_1_ITEM_18", "GROUP_1_ITEM_19", "GROUP_1_ITEM_20",
		"GROUP_1_ITEM_22",
	}
	withoutWarehouse := []string{
		"GROUP_1_ITEM_2", "GROUP_1_ITEM_4", "GROUP_1_ITEM_6",
		"GROUP_1_ITEM_9", "GROUP_1_ITEM_11", "GROUP_1_ITEM_13",
	}

	assert := func(groupCode string, want bool) {
		t.Helper()
		cfg, err := LoadConfiguration(groupCode)
		if err != nil {
			t.Fatalf("%s: โหลด config ไม่ได้: %v", groupCode, err)
		}
		found := false
		for i := range cfg.Patterns {
			if patternHasWarehouseColumn(&cfg.Patterns[i]) {
				found = true
				break
			}
		}
		if found != want {
			t.Errorf("%s: patternHasWarehouseColumn = %v ต้องเป็น %v", groupCode, found, want)
		}
	}

	for _, gc := range withWarehouse {
		assert(gc, true)
	}
	for _, gc := range withoutWarehouse {
		assert(gc, false)
	}
}

// ทั้งสอง pattern ที่ใช้หัวคอลัมน์ต่างกัน ("Stock" กับ "โกดัง") ต้อง map ไป
// dataMapping เดียวกัน ไม่งั้นค่าที่แสดงจะมาคนละทาง
func TestStockHeaderStillMapsToWarehouse(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_8")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}

	header := ""
	for i := range cfg.Patterns {
		p := &cfg.Patterns[i]
		cols := append(append([]ColumnConfigItem{}, p.Columns...), p.FixedColumns...)
		for _, g := range p.ColumnGroups {
			cols = append(cols, g.Children...)
		}
		for _, c := range cols {
			if c.DataMapping == "warehouse" {
				header = c.HeaderName
			}
		}
	}

	if header != "Stock" {
		t.Errorf("ITEM_8 ต้องคงหัวคอลัมน์ว่า Stock ไว้ ได้ %q", header)
	}
}
