package saleService

import "testing"

// เงื่อนไข SQL (NOT IN) ของ markSaleItems กับ predicate IsSaleItemClosed ต้องอ้างลิสต์เดียวกัน
// ไม่งั้นเส้นอัตโนมัติจะเขียนทับบรรทัดที่สะกด CANCELLED สองแอลได้
func TestClosedSaleItemStatusListCoversEverySpelling(t *testing.T) {
	if len(closedSaleItemStatusList) != 3 {
		t.Fatalf("closedSaleItemStatusList = %v, want 3 ค่า", closedSaleItemStatusList)
	}

	for _, status := range []string{"COMPLETED", "CANCELED", "CANCELLED"} {
		if !IsSaleItemClosed(status) {
			t.Errorf("IsSaleItemClosed(%q) = false, want true", status)
		}
	}

	for _, status := range []string{"PENDING", "TEMP", ""} {
		if IsSaleItemClosed(status) {
			t.Errorf("IsSaleItemClosed(%q) = true, want false", status)
		}
	}

	// ทุกค่าในลิสต์ต้องอยู่ใน map ที่ predicate ใช้จริง (กันสองที่เพี้ยนออกจากกัน)
	for _, status := range closedSaleItemStatusList {
		if !closedSaleItemStatuses[status] {
			t.Errorf("closedSaleItemStatuses ไม่มี %q ที่อยู่ใน closedSaleItemStatusList", status)
		}
	}
}

func TestAllSaleItemsClosedCountsCanceledLinesAsDone(t *testing.T) {
	cases := []struct {
		name     string
		statuses []string
		want     bool
	}{
		{"ครบทุกบรรทัด", []string{"COMPLETED", "COMPLETED"}, true},
		{"มีบรรทัดที่ถูกยกเลิกปนอยู่", []string{"COMPLETED", "CANCELED"}, true},
		{"สะกดแบบสองแอล", []string{"COMPLETED", "CANCELLED"}, true},
		{"ยังมีบรรทัดค้าง", []string{"COMPLETED", "PENDING"}, false},
		{"ไม่มีบรรทัดเลย", []string{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allSaleItemsClosed(tc.statuses); got != tc.want {
				t.Errorf("allSaleItemsClosed(%v) = %v, want %v", tc.statuses, got, tc.want)
			}
		})
	}
}
