package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

// calculateExtraForSubGroup คืนค่าเรียงเป็น (extraWeight, extraUnit)
// และเมื่อไม่มี group extra แมตช์ จะคงค่าเดิมของ subgroup ไว้ทั้งคู่
// ซึ่งอาจต่างกันได้ — test นี้ตรึงลำดับและความต่างไว้
//
// ความต่างนี้สำคัญเพราะผู้เรียกต้องจับคู่ extra ให้ตรงกับ uom ของสูตร
// สูตร uom pcs ต้องใช้ extraUnit และสูตร uom kg ต้องใช้ extraWeight
func TestCalculateExtraForSubGroupReturnsWeightThenUnit(t *testing.T) {
	subGroup := &models.PriceListSubGroup{
		ExtraPriceWeight: 5.0,
		ExtraPriceUnit:   10.0,
	}

	extraWeight, extraUnit, err := calculateExtraForSubGroup(subGroup)
	if err != nil {
		t.Fatalf("คำนวณ extra ล้มเหลว: %v", err)
	}

	if extraWeight != 5.0 {
		t.Errorf("ค่าที่คืนตัวแรก = %v ต้องเป็น 5.0 (ExtraPriceWeight)", extraWeight)
	}
	if extraUnit != 10.0 {
		t.Errorf("ค่าที่คืนตัวที่สอง = %v ต้องเป็น 10.0 (ExtraPriceUnit)", extraUnit)
	}
}
