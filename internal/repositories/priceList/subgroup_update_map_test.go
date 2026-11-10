package priceListRepository

import (
	"strings"
	"testing"

	"prime-erp-core/internal/models"
)

func floatPtr(v float64) *float64 { return &v }

// ตรึงพฤติกรรมเดิม: เมื่อค่าใหม่ต่างจากค่าเดิม ต้องเลื่อน before_* มาเก็บค่าเดิม
func TestBuildSubGroupUpdateMap_ShiftsBeforeWhenValueChanges(t *testing.T) {
	old := models.PriceListSubGroup{
		TotalNetPriceWeight: 18.50,
		TotalNetPriceUnit:   1200,
	}
	item := models.UpdatePriceListSubGroupItem{
		TotalNetPriceWeight: floatPtr(19.75),
		TotalNetPriceUnit:   floatPtr(1300),
	}

	got := buildSubGroupUpdateMap(old, item)

	if got["total_net_price_weight"] != 19.75 {
		t.Fatalf("total_net_price_weight = %v, want 19.75", got["total_net_price_weight"])
	}
	if got["before_total_net_price_weight"] != 18.50 {
		t.Fatalf("before_total_net_price_weight = %v, want 18.50", got["before_total_net_price_weight"])
	}
	if got["total_net_price_unit"] != float64(1300) {
		t.Fatalf("total_net_price_unit = %v, want 1300", got["total_net_price_unit"])
	}
	if got["before_total_net_price_unit"] != float64(1200) {
		t.Fatalf("before_total_net_price_unit = %v, want 1200", got["before_total_net_price_unit"])
	}
}

// ตรึงพฤติกรรมเดิม: field ที่ req ไม่ได้ส่งมา (nil) ต้องไม่ปรากฏใน updateMap เลย
// เพื่อไม่ให้ GORM เขียนทับด้วย zero value
func TestBuildSubGroupUpdateMap_SkipsNilFields(t *testing.T) {
	old := models.PriceListSubGroup{PriceWeight: 10, TotalNetPriceWeight: 20}
	item := models.UpdatePriceListSubGroupItem{TotalNetPriceWeight: floatPtr(21)}

	got := buildSubGroupUpdateMap(old, item)

	if _, exists := got["price_weight"]; exists {
		t.Fatal("price_weight ไม่ควรอยู่ใน updateMap เพราะ req ไม่ได้ส่งมา")
	}
	if _, exists := got["before_price_weight"]; exists {
		t.Fatal("before_price_weight ไม่ควรอยู่ใน updateMap เพราะ req ไม่ได้ส่งมา")
	}
}

// บั๊กจริง: update-latest ส่ง TotalNetPrice* ทุก subgroup ทุกครั้งแม้ราคาไม่เปลี่ยน
// ถ้าเลื่อน before_* โดยไม่ดูว่าค่าเปลี่ยนจริงไหม snapshot จะถูกทับจนเท่ากับค่าปัจจุบัน
// ข้อมูลจริงเมื่อ 2026-09-11 มี 1,257 จาก 1,350 subgroup ที่ before = after ไปแล้ว
func TestBuildSubGroupUpdateMap_DoesNotShiftBeforeWhenValueUnchanged(t *testing.T) {
	old := models.PriceListSubGroup{
		TotalNetPriceWeight: 18.50,
		TotalNetPriceUnit:   1200,
		PriceWeight:         17,
		ExtraPriceWeight:    1.5,
		TermPriceWeight:     0.55,
		PriceUnit:           1100,
		ExtraPriceUnit:      100,
	}
	item := models.UpdatePriceListSubGroupItem{
		TotalNetPriceWeight: floatPtr(18.50),
		TotalNetPriceUnit:   floatPtr(1200),
		PriceWeight:         floatPtr(17),
		ExtraPriceWeight:    floatPtr(1.5),
		TermPriceWeight:     floatPtr(0.55),
		PriceUnit:           floatPtr(1100),
		ExtraPriceUnit:      floatPtr(100),
	}

	got := buildSubGroupUpdateMap(old, item)

	for key := range got {
		if strings.HasPrefix(key, "before_") {
			t.Errorf("ค่าไม่เปลี่ยนแต่ยังเลื่อน %s = %v — snapshot จะถูกทับ", key, got[key])
		}
	}
}

// เลื่อนเฉพาะ field ที่ค่าเปลี่ยนจริง field อื่นในคำขอเดียวกันต้องไม่ถูกเลื่อน
func TestBuildSubGroupUpdateMap_ShiftsOnlyChangedFields(t *testing.T) {
	old := models.PriceListSubGroup{
		TotalNetPriceWeight: 18.50,
		TotalNetPriceUnit:   1200,
	}
	item := models.UpdatePriceListSubGroupItem{
		TotalNetPriceWeight: floatPtr(19.75), // เปลี่ยน
		TotalNetPriceUnit:   floatPtr(1200),  // ไม่เปลี่ยน
	}

	got := buildSubGroupUpdateMap(old, item)

	if got["before_total_net_price_weight"] != 18.50 {
		t.Errorf("field ที่เปลี่ยนต้องเลื่อน before: ได้ %v", got["before_total_net_price_weight"])
	}
	if _, exists := got["before_total_net_price_unit"]; exists {
		t.Error("field ที่ไม่เปลี่ยนต้องไม่เลื่อน before")
	}
	// ค่าปัจจุบันยังต้องถูกเขียนทั้งสอง field
	if got["total_net_price_weight"] != 19.75 || got["total_net_price_unit"] != float64(1200) {
		t.Errorf("ค่าปัจจุบันต้องถูกเขียนครบ: %v", got)
	}
}
