package patterns

import (
	"encoding/json"
	"strings"
	"testing"
)

// เซลล์ข้อมูลในกริด price list ต้องจัดกลาง
//
// หัวตารางเป็น center อยู่แล้วทั้งจาก defaultColDef ของ DynamicTable และจาก
// applyHeaderAlignment ที่ fallback เป็น center แต่เซลล์ข้อมูลใช้ค่าจาก
// cellStyle.textAlign ใน config นี้ ก่อนหน้านี้มี "left" อยู่ 81 จุดใน 9 ไฟล์
// ทำให้ผู้ใช้เห็นเซลล์ชิดซ้าย
//
// test นี้กันไม่ให้ "left" กลับมาเงียบ ๆ ตอนแก้ config ครั้งหน้า
func TestPatternConfigs_CellsAreCentered(t *testing.T) {
	entries, err := patternConfigs.ReadDir("configs")
	if err != nil {
		t.Fatalf("อ่าน configs ไม่ได้: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("ไม่พบไฟล์ config เลย — embed.FS อาจเปลี่ยน path")
	}

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := patternConfigs.ReadFile("configs/" + entry.Name())
		if err != nil {
			t.Fatalf("%s: อ่านไม่ได้: %v", entry.Name(), err)
		}

		// parse เพื่อยืนยันว่า JSON ยังใช้ได้ ไม่ใช่แค่ค้นสตริง
		var config PriceTableConfiguration
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("%s: parse ไม่ผ่าน: %v", entry.Name(), err)
		}

		if strings.Contains(string(data), `"textAlign": "left"`) {
			t.Errorf("%s: ยังมี \"textAlign\": \"left\" อยู่ — เซลล์ต้องจัดกลาง", entry.Name())
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("ไม่ได้ตรวจไฟล์ config ใดเลย")
	}
	t.Logf("ตรวจ config %d ไฟล์", checked)
}
