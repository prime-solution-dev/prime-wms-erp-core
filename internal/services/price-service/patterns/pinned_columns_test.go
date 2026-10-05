package patterns

import (
	"encoding/json"
	"testing"
)

// ทุก tab ต้อง pin คอลัมน์แรก ไม่งั้นเลื่อนซ้าย/ขวาแล้วคอลัมน์ระบุแถวหาย
func TestPatternConfigs_FirstColumnPinnedLeft(t *testing.T) {
	entries, err := patternConfigs.ReadDir("configs")
	if err != nil {
		t.Fatalf("อ่าน configs ไม่ได้: %v", err)
	}
	for _, entry := range entries {
		data, err := patternConfigs.ReadFile("configs/" + entry.Name())
		if err != nil {
			t.Fatalf("%s: อ่านไม่ได้: %v", entry.Name(), err)
		}
		var root any
		if err := json.Unmarshal(data, &root); err != nil {
			t.Fatalf("%s: parse ไม่ผ่าน: %v", entry.Name(), err)
		}
		for _, cols := range leadingColumnLists(root) {
			first, _ := cols[0].(map[string]any)
			if first["pinned"] != "left" {
				t.Errorf("%s: คอลัมน์แรก %v ไม่ได้ pinned left", entry.Name(), first["field"])
			}
		}
	}
}

// leadingColumnLists คืนรายการคอลัมน์นำหน้าของแต่ละ tab: fixedColumns ถ้ามี ไม่งั้น columns ที่เป็นคอลัมน์แสดงผล (มี headerName)
func leadingColumnLists(node any) [][]any {
	var out [][]any
	switch v := node.(type) {
	case map[string]any:
		if cols, ok := v["fixedColumns"].([]any); ok && len(cols) > 0 {
			out = append(out, cols)
		} else if cols, ok := v["columns"].([]any); ok && len(cols) > 0 {
			if first, ok := cols[0].(map[string]any); ok && first["headerName"] != nil {
				out = append(out, cols)
			}
		}
		for k, child := range v {
			if k == "fixedColumns" || k == "columns" {
				continue
			}
			out = append(out, leadingColumnLists(child)...)
		}
	case []any:
		for _, child := range v {
			out = append(out, leadingColumnLists(child)...)
		}
	}
	return out
}
