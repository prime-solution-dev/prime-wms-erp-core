package deliveryService

import (
	"testing"

	"prime-erp-core/internal/models"
)

func TestSaleItemCompletionModeFollowsSaleUnitAndType(t *testing.T) {
	cases := []struct {
		name     string
		saleUnit string
		unitType string
		wantMode string
	}{
		{"ขายเป็นชิ้น", "PC", "PC", completionModeQty},
		{"ขายเป็นกิโลจริง", "KG", "KG", completionModeWeight},
		{"KG_SPEC คุมด้วยชิ้น", "KG", "KG_SPEC", completionModeQty},
		{"KG_SPEC เขียนด้วยขีดกลาง", "KG", "kg-spec", completionModeQty},
		{"KG ที่ตั้งราคาแบบชิ้น ยังคิดเป็นน้ำหนัก", "KG", "PC", completionModeWeight},
		{"ค่าว่างถือเป็นชิ้น", "", "", completionModeQty},
		{"มีช่องว่างและตัวพิมพ์เล็ก", " kg ", " kg ", completionModeWeight},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := saleItemCompletionMode(tc.saleUnit, tc.unitType); got != tc.wantMode {
				t.Errorf("saleItemCompletionMode(%q, %q) = %q, want %q", tc.saleUnit, tc.unitType, got, tc.wantMode)
			}
		})
	}
}

// SO202609-0011 = 10 PCS / 51.50 kg ขายเป็นชิ้น -> เป้าคือ 10
// SO202609-0017 = 53 ชิ้น / 500 kg ขายเป็นกิโล -> เป้าคือ 500
func TestCompletionTargetPicksQtyOrWeight(t *testing.T) {
	pieces := models.SaleItem{Qty: 10, TotalWeight: 51.5, SaleUnit: "PC", SaleUnitType: "PC"}
	if got := completionTarget(pieces); got != 10 {
		t.Errorf("completionTarget(ชิ้น) = %v, want 10", got)
	}

	weighted := models.SaleItem{Qty: 53, TotalWeight: 500, SaleUnit: "KG", SaleUnitType: "KG"}
	if got := completionTarget(weighted); got != 500 {
		t.Errorf("completionTarget(กิโล) = %v, want 500", got)
	}

	kgSpec := models.SaleItem{Qty: 6, TotalWeight: 120, SaleUnit: "KG", SaleUnitType: "KG_SPEC"}
	if got := completionTarget(kgSpec); got != 6 {
		t.Errorf("completionTarget(KG_SPEC) = %v, want 6", got)
	}
}

func TestIsFullyDeliveredUsesLowerBoundOnly(t *testing.T) {
	const tolerance = 3.0 // ค่าจริงของ TMI ใน system_config

	cases := []struct {
		name   string
		target float64
		issued float64
		want   bool
	}{
		{"ส่งครบพอดี", 10, 10, true},
		{"ขาดแต่ยังอยู่ในระยะผ่อนผัน", 100, 97, true},
		{"ขาดเกินระยะผ่อนผัน", 100, 96.9, false},
		{"ส่งเกินถือว่าครบ", 10, 11, true},
		{"ยังไม่ได้ส่งเลย", 10, 0, false},
		{"เป้าเป็นศูนย์ไม่ปิด", 0, 0, false},
		{"เป้าติดลบไม่ปิด", -5, 10, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFullyDelivered(tc.target, tc.issued, tolerance); got != tc.want {
				t.Errorf("isFullyDelivered(%v, %v, %v) = %v, want %v", tc.target, tc.issued, tolerance, got, tc.want)
			}
		})
	}
}

func TestIsFullyDeliveredTreatsNegativeToleranceAsZero(t *testing.T) {
	if isFullyDelivered(100, 97, -3) {
		t.Error("tolerance ติดลบต้องถือเป็น 0 -> ส่ง 97 จาก 100 ยังไม่ครบ")
	}
}
