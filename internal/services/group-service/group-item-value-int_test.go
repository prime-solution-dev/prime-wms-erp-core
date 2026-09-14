package groupService

import (
	"testing"

	"prime-erp-core/internal/models"
)

// WMS เก็บขนาดไว้ในคอลัมน์ value เป็น text และไม่ได้ส่ง value_int มาด้วย
// แต่ extraConditionMatched ฝั่ง price-service ใช้ value_int เป็นตัวเทียบเงื่อนไข extra
// ถ้าไม่ derive ตรงนี้ เงื่อนไขทุกข้อจะถูกเทียบกับ 0
func TestDeriveGroupItemValueInt(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		valueInt float64
		want     float64
	}{
		{"ทศนิยม", "1.70", 0, 1.7},
		{"จำนวนเต็ม", "38.00", 0, 38},
		{"ไม่มีทศนิยม", "38", 0, 38},
		{"มีช่องว่างหน้าหลัง", "  38.00  ", 0, 38},
		{"ค่าติดลบ", "-2.50", 0, -2.5},
		{"value ไม่ใช่ตัวเลขต้องคง 0", "BULK", 0, 0},
		{"value ว่างต้องคง 0", "", 0, 0},
		{"value_int ที่ส่งมาแล้วต้องไม่ถูกทับ", "38.00", 12, 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []models.GroupItem{{
				ItemCode: "PG06_1",
				ItemName: "38",
				Value:    tt.value,
				ValueInt: tt.valueInt,
			}}

			deriveGroupItemValueInt(items)

			if items[0].ValueInt != tt.want {
				t.Fatalf("value=%q value_int เดิม=%v ต้องได้ %v แต่ได้ %v",
					tt.value, tt.valueInt, tt.want, items[0].ValueInt)
			}
		})
	}
}

func TestDeriveGroupItemValueInt_HandlesEmptySlice(t *testing.T) {
	deriveGroupItemValueInt(nil)
	deriveGroupItemValueInt([]models.GroupItem{})
}
