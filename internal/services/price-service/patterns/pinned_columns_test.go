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
		// pattern ที่ fixedColumns ว่าง (เช่น GROUP_1_ITEM_5) ไม่ถูกตรวจ เพราะ columns เป็น template ต่อ column group
		for _, cols := range fixedColumnLists(root) {
			first, _ := cols[0].(map[string]any)
			if first["pinned"] != "left" {
				t.Errorf("%s: คอลัมน์แรก %v ไม่ได้ pinned left", entry.Name(), first["field"])
			}
		}
	}
}

// fixedColumnLists คืน fixedColumns ทุกตัวที่ไม่ว่าง — เป็นทางเดียวที่ builder คัดลอก pinned ไปถึง API
func fixedColumnLists(node any) [][]any {
	var out [][]any
	switch v := node.(type) {
	case map[string]any:
		if cols, ok := v["fixedColumns"].([]any); ok && len(cols) > 0 {
			out = append(out, cols)
		}
		for k, child := range v {
			if k == "fixedColumns" {
				continue
			}
			out = append(out, fixedColumnLists(child)...)
		}
	case []any:
		for _, child := range v {
			out = append(out, fixedColumnLists(child)...)
		}
	}
	return out
}
