package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// subgroup ที่ต่างกันแค่ PG06 ต้องเป็นคนละแถว
// PG06 เป็น pinned fixedColumn ระดับแถวของ GROUP_1_ITEM_9 แต่ไม่ได้อยู่ใน grouping.rows
// จึงถูกยุบเข้าแถวเดียวกันแบบ last-write-wins ทำให้ข้อมูลหาย
func TestGroup1Item9_SubGroupsDifferingOnlyByPG06_StayInSeparateRows(t *testing.T) {
	cfg, err := LoadConfiguration("GROUP_1_ITEM_9")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if len(cfg.Patterns) == 0 {
		t.Fatal("config ไม่มี pattern เลย")
	}
	pattern := &cfg.Patterns[0]

	subGroups := []models.PriceListSubGroupResponse{
		item9SubGroup("sg-1", "PG06_1", "ความยาว 6 เมตร", 101),
		item9SubGroup("sg-2", "PG06_2", "ความยาว 9 เมตร", 202),
	}

	rows := buildDynamicRows(cfg, pattern, subGroups)
	merged := mergeGroup1Item9Rows(rows)

	if len(merged) != 2 {
		t.Fatalf("ต้องได้ 2 แถว (คนละ PG06) แต่ได้ %d แถว — ข้อมูลถูกยุบทับกัน", len(merged))
	}

	// ราคาต้องอยู่กับแถวที่ถูกต้อง ไม่ใช่แค่ยังปรากฏอยู่ที่ไหนก็ได้
	// ผูก PG06 กับราคาที่คาดไว้เป็นคู่ เพื่อจับกรณีราคาสลับแถว
	wantByPG06 := map[string]float64{
		"ความยาว 6 เมตร": 101,
		"ความยาว 9 เมตร": 202,
	}

	for _, row := range merged {
		// shared.go เขียนลงแถวเฉพาะ field ที่อยู่ใน rowFields
		// คอลัมน์ PG06 ที่ pinned ไว้จะมีข้อมูลก็เมื่อ PG06 อยู่ใน grouping.rows
		pg06, ok := row["pg_06"].(string)
		if !ok || pg06 == "" {
			t.Fatalf("คอลัมน์ PG06 ไม่มีข้อมูลป้อน: row = %v", row)
		}

		want, known := wantByPG06[pg06]
		if !known {
			t.Fatalf("เจอแถวที่ PG06 = %q ซึ่งไม่ได้สร้างไว้ใน fixture", pg06)
		}

		found := false
		for _, value := range row {
			if v, isFloat := value.(float64); isFloat && v == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("แถว PG06 = %q ต้องมีราคา %v แต่หาไม่เจอ: row = %v", pg06, want, row)
		}
		delete(wantByPG06, pg06)
	}

	if len(wantByPG06) != 0 {
		t.Fatalf("ยังมี subgroup ที่ไม่ปรากฏเป็นแถวของตัวเอง: %v", wantByPG06)
	}
}

// item9SubGroup สร้าง subgroup ที่มี key ครบตามที่ GROUP_1_ITEM_9 ต้องใช้
// PG02 กับ PG07 เหมือนกันทุกตัว ต่างกันแค่ PG06 เพื่อแยกให้ชัดว่า PG06 คือมิติที่หายไป
// buildCompositeKey อ่านจาก ValueName ส่วน buildCompositeCodeKey อ่านจาก ValueCode
// จึงต้องตั้งทั้งสอง field
func item9SubGroup(id, pg06Code, pg06Name string, priceWeight float64) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:                  id,
		SubgroupCode:        id,
		TotalNetPriceWeight: priceWeight,
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			{GroupCode: "PG02", ValueCode: "PG02_10", ValueName: "หมวดเหล็กเส้น", Seq: 2},
			{GroupCode: "PG07", ValueCode: "PG07_8", ValueName: "ขนาด 12 มม.", Seq: 7},
			{GroupCode: "PG06", ValueCode: pg06Code, ValueName: pg06Name, Seq: 6},
			{GroupCode: "PG03", ValueCode: "PG03_1", ValueName: "เกรด SD40", Seq: 3},
		},
	}
}
