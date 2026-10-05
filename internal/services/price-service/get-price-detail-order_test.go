package priceService

import (
	"strings"
	"testing"
)

// companyCodeSet และ siteCodeSet ถูกแปลงเป็น slice ด้วยการวน map ซึ่ง Go สุ่มลำดับ
// get-price-detail ใช้ companyCodes[0] เป็นค่าที่ส่งไป inventory service
// ถ้าไม่ sort ค่าที่ถูกเลือกจะเปลี่ยนทุกครั้งที่เรียก
func TestSortedSetKeys_IsDeterministic(t *testing.T) {
	set := map[string]bool{
		"C003": true,
		"C001": true,
		"C002": true,
	}

	var first string
	for round := 0; round < 50; round++ {
		got := strings.Join(sortedSetKeys(set), ",")
		if round == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("รอบที่ %d ได้ %q ต่างจากรอบแรก %q — ลำดับไม่นิ่ง", round, got, first)
		}
	}

	if first != "C001,C002,C003" {
		t.Fatalf("ต้อง sort ได้ %q ต้องเป็น \"C001,C002,C003\"", first)
	}
}

func TestSortedSetKeys_Empty(t *testing.T) {
	if got := sortedSetKeys(map[string]bool{}); len(got) != 0 {
		t.Fatalf("set ว่างต้องได้ slice ว่าง แต่ได้ %v", got)
	}
}

func TestSortedSetKeys_Nil(t *testing.T) {
	if got := sortedSetKeys(nil); len(got) != 0 {
		t.Fatalf("set nil ต้องได้ slice ว่าง แต่ได้ %v", got)
	}
}
