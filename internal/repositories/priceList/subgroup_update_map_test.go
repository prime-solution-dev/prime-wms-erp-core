package priceListRepository

import (
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
