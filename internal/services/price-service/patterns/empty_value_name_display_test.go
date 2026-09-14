package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// GROUP_1_ITEM_13 คอลัมน์ "ขนาด" (field=size) มี dataMapping "PG04_x_PG03"
// item13SubGroup สร้าง subgroup ที่มี key ครบพอให้ buildDirectRows ประกอบแถวได้
func item13SubGroup(id, pg03Name string) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:                  id,
		SubgroupCode:        id,
		TotalNetPriceWeight: 100,
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			{GroupCode: "PG01", ValueCode: "PG01_1", ValueName: "เหล็กรูปพรรณ", Seq: 1},
			{GroupCode: "PG04", ValueCode: "PG04_82", ValueName: "150x75x5.5x9.5", Seq: 4},
			{GroupCode: "PG03", ValueCode: "PG03_1", ValueName: pg03Name, Seq: 3},
			{GroupCode: "PG06", ValueCode: "PG06_1", ValueName: "5.5", Seq: 6},
			{GroupCode: "PG07", ValueCode: "PG07_1", ValueName: "6", Seq: 7},
		},
	}
}

func item13SizeCell(t *testing.T, pg03Name string) string {
	t.Helper()

	cfg, err := LoadConfiguration("GROUP_1_ITEM_13")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if len(cfg.Patterns) == 0 {
		t.Fatal("config ไม่มี pattern เลย")
	}

	rows := buildDirectRows(cfg, &cfg.Patterns[0], []models.PriceListSubGroupResponse{item13SubGroup("sg-1", pg03Name)})
	if len(rows) == 0 {
		t.Fatal("ไม่ได้แถวเลย")
	}

	size, ok := rows[0]["size"].(string)
	if !ok {
		t.Fatalf("คอลัมน์ size ไม่ใช่ string: %#v", rows[0]["size"])
	}
	return size
}

// ชื่อว่างต้องไม่เหลือ separator " x " ห้อยท้าย
func TestItem13SizeColumn_EmptyValueNameLeavesNoTrailingSeparator(t *testing.T) {
	if got := item13SizeCell(t, ""); got != "150x75x5.5x9.5" {
		t.Fatalf("คอลัมน์ ขนาด = %q, want %q", got, "150x75x5.5x9.5")
	}
}

// happy path: มีชื่อครบต้องประกอบเต็มเหมือนเดิม
func TestItem13SizeColumn_BothNamesPresentKeepsSeparator(t *testing.T) {
	if got := item13SizeCell(t, "ZUB"); got != "150x75x5.5x9.5 x ZUB" {
		t.Fatalf("คอลัมน์ ขนาด = %q, want %q", got, "150x75x5.5x9.5 x ZUB")
	}
}

// column header ต้องว่างเมื่อชื่อว่าง แต่ตัวคอลัมน์ต้องยังอยู่ (ไม่หายทั้งคอลัมน์)
// และ key ยังต้องมาจาก code
func TestBuildSingleLevelColumns_EmptyValueNameKeepsColumnWithEmptyHeader(t *testing.T) {
	pattern := &PatternConfig{
		Columns: []ColumnConfigItem{{Field: "price", HeaderName: "ราคา"}},
	}
	subGroups := []models.PriceListSubGroupResponse{
		{ID: "sg-1", SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			{GroupCode: "PG03", ValueCode: "PG03_1", ValueName: ""},
		}},
	}

	columns := buildSingleLevelColumns(pattern, subGroups, []string{"PG03"})
	if len(columns) == 0 {
		t.Fatal("คอลัมน์หายทั้งคอลัมน์เมื่อชื่อว่าง — ข้อมูลราคาหายจากกริด")
	}
	if columns[0].HeaderName != "" {
		t.Fatalf("HeaderName = %q, want \"\" (ชื่อว่างคือว่าง ไม่ใช่ code)", columns[0].HeaderName)
	}
	if columns[0].GroupID != "group_pg03_1" {
		t.Fatalf("GroupID = %q, want %q (key ต้องยังมาจาก code)", columns[0].GroupID, "group_pg03_1")
	}
}

// column_group_value (ค่าที่แสดงเป็นหัวคอลัมน์) ต้องว่างเมื่อ item_name ว่าง
// ไม่ใช่ตกไปใช้ columnKey ซึ่งเป็น code ที่ผ่าน sanitize มาแล้ว (เช่น "pg03_1")
func TestBuildDynamicRows_EmptyValueNameDoesNotFallBackToCodeLabel(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_9")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if len(cfg.Patterns) == 0 {
		t.Fatal("config ไม่มี pattern เลย")
	}

	sg := item9SubGroup("sg-1", "PG06_1", "ความยาว 6 เมตร", 101)
	for i := range sg.SubGroupKeys {
		if sg.SubGroupKeys[i].GroupCode == "PG03" {
			sg.SubGroupKeys[i].ValueName = "" // มี record แต่ item_name ว่างโดยตั้งใจ
		}
	}

	rows := buildDynamicRows(cfg, &cfg.Patterns[0], []models.PriceListSubGroupResponse{sg})
	if len(rows) == 0 {
		t.Fatal("ไม่ได้แถวเลย")
	}
	if got := rows[0]["column_group_value"]; got != "" {
		t.Fatalf("column_group_value = %#v, want \"\" (ห้าม fallback ไป code)", got)
	}
}

// happy path ของ dynamic rows: มีชื่อต้องได้ชื่อนั้นเป็นหัวคอลัมน์
func TestBuildDynamicRows_ValueNamePresentIsUsedAsColumnLabel(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_9")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}

	sg := item9SubGroup("sg-1", "PG06_1", "ความยาว 6 เมตร", 101)
	rows := buildDynamicRows(cfg, &cfg.Patterns[0], []models.PriceListSubGroupResponse{sg})
	if len(rows) == 0 {
		t.Fatal("ไม่ได้แถวเลย")
	}
	if got := rows[0]["column_group_value"]; got != "เกรด SD40" {
		t.Fatalf("column_group_value = %#v, want %q", got, "เกรด SD40")
	}
}

// เมื่อไม่มี code ของ columnLevels เลยสักตัว columnKey ตกไปใช้ col_<subgroup id>
// ซึ่งเป็นตัวระบุตัวตน ไม่ควรหลุดไปโชว์เป็นหัวคอลัมน์
func TestBuildDynamicRows_MissingAllColumnLevelCodesLeavesLabelEmpty(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_12")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if len(cfg.Patterns) == 0 || len(cfg.Patterns[0].ColumnLevels) == 0 {
		t.Fatal("ต้องใช้ pattern ที่มี columnLevels")
	}

	// ไม่มี key ของ PG09/PG07 ที่ columnLevels อ้างถึงเลย
	sg := models.PriceListSubGroupResponse{
		ID:           "sg-1",
		SubgroupCode: "sg-1",
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			{GroupCode: "PG02", ValueCode: "PG02_1", ValueName: "หมวด A", Seq: 2},
		},
	}

	rows := buildDynamicRows(cfg, &cfg.Patterns[0], []models.PriceListSubGroupResponse{sg})
	if len(rows) == 0 {
		t.Fatal("ไม่ได้แถวเลย")
	}
	if got := rows[0]["column_group_value"]; got != "" {
		t.Fatalf("column_group_value = %#v, want \"\" (ห้ามเอา col_<uuid> ไปโชว์)", got)
	}
}
