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

	// ยืนยันว่าราคาของทั้งสอง subgroup ยังอยู่ ไม่ถูกทับ
	seen := map[float64]bool{}
	for _, row := range merged {
		for _, value := range row {
			if v, ok := value.(float64); ok && (v == 101 || v == 202) {
				seen[v] = true
			}
		}
	}
	if !seen[101] || !seen[202] {
		t.Fatalf("ราคาของ subgroup ถูกทับหาย: เห็น %v ต้องเห็นทั้ง 101 และ 202", seen)
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
