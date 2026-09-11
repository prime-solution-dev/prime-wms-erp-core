package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

// ล็อกกติกาการ resolve ชื่อ item ให้ตรงกับฝั่ง export (itemNameByCode):
// มี record ใน group_item แต่ item_name ว่างโดยตั้งใจ ต้องแสดงว่าง ไม่ fallback ไป code
func TestResolveGroupItemName(t *testing.T) {
	groupItemMap := map[string]models.GetGroupItemResponse{
		"PG03_1": {ItemCode: "PG03_1", ItemName: ""},
		"PG03_2": {ItemCode: "PG03_2", ItemName: "SS400"},
	}

	cases := []struct {
		name string
		code string
		want string
	}{
		{"มี record แต่ชื่อว่าง ต้องได้ค่าว่าง", "PG03_1", ""},
		{"มี record และมีชื่อ ต้องได้ชื่อนั้น", "PG03_2", "SS400"},
		{"ไม่มี record ต้อง fallback ไป code", "PG99_9", "PG99_9"},
		{"code ว่าง ไม่มี record ต้องได้ค่าว่าง", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveGroupItemName(groupItemMap, tc.code); got != tc.want {
				t.Fatalf("resolveGroupItemName(%q) = %q, want %q", tc.code, got, tc.want)
			}
		})
	}
}
