package patterns

import (
	"encoding/json"
	"strings"
	"testing"
)

// issue log TMI #39: หน้า price list เหล็กฉาก (GROUP_1_ITEM_6) หัวคอลัมน์เขียน "ยี่ห้อ"
// แต่ค่าที่แสดงคือ PG03 ซึ่ง product master เรียกว่า เกรด/รูปแบบ — คอลัมน์ที่ดึง PG03
// ต้องไม่ใช้ชื่อ "ยี่ห้อ" (pattern ส่วนใหญ่ใช้ "เกรด", ITEM_4 ใช้ "spec" มาแต่เดิม)
func TestPatternConfigs_PG03ColumnIsNotBrand(t *testing.T) {
	entries, err := patternConfigs.ReadDir("configs")
	if err != nil {
		t.Fatalf("อ่าน configs ไม่ได้: %v", err)
	}

	found := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := patternConfigs.ReadFile("configs/" + entry.Name())
		if err != nil {
			t.Fatalf("%s: อ่านไม่ได้: %v", entry.Name(), err)
		}
		var config PriceTableConfiguration
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("%s: parse ไม่ผ่าน: %v", entry.Name(), err)
		}
		for _, p := range config.Patterns {
			for _, col := range append(append([]ColumnConfigItem{}, p.FixedColumns...), p.Columns...) {
				if col.DataMapping != "PG03" {
					continue
				}
				found++
				if col.HeaderName == "ยี่ห้อ" {
					t.Errorf("%s pattern %s: คอลัมน์ %q ดึง PG03 (เกรด/รูปแบบ) แต่หัวคอลัมน์เป็น \"ยี่ห้อ\"",
						entry.Name(), p.ID, col.Field)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("ไม่พบคอลัมน์ที่ดึง PG03 เลย — โครง config อาจเปลี่ยน")
	}
}
