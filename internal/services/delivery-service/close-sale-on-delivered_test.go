package deliveryService

import (
	"testing"

	"prime-erp-core/internal/models"
	orderExternalService "prime-erp-core/external/order-service"
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

// buildOutboundGI สร้าง outbound 1 ตัวพร้อมบรรทัด GI (qty, weight) — ตั้งชื่อไม่ให้ชนกับ
// buildOutbound ใน validate-booking-qty_test.go ที่รับแต่ qty
func buildOutboundGI(status string, lines ...[2]float64) orderExternalService.OutboundItemWithGoodsIssue {
	outbound := orderExternalService.OutboundItemWithGoodsIssue{}
	outbound.Status = status

	for _, line := range lines {
		outbound.GoodsIssueItem = append(outbound.GoodsIssueItem, orderExternalService.GetIssueItemResponse{
			Qty:    line[0],
			Weight: line[1],
		})
	}

	return outbound
}

// เคสจริง GI20260909051617-0 ของ CO2609-00032: 2 บรรทัด 7+3 ชิ้น = 36.05+15.50 kg
func TestFoldWmsIssuedSumsQtyAndWeight(t *testing.T) {
	issuedQty, issuedWeight, closed := foldWmsIssued([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS202609-0012", "ITEM-1", "COMPLETED", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutboundGI("COMPLETED", [2]float64{7, 36.05}, [2]float64{3, 15.5}),
		}),
	})

	key := "DBS202609-0012|ITEM-1"
	if !closed[key] {
		t.Fatalf("closed[%s] = false, want true", key)
	}
	if issuedQty[key] != 10 {
		t.Errorf("issuedQty[%s] = %v, want 10", key, issuedQty[key])
	}
	if issuedWeight[key] != 51.55 {
		t.Errorf("issuedWeight[%s] = %v, want 51.55", key, issuedWeight[key])
	}
}

func TestFoldWmsIssuedIgnoresOpenOrderItemAndUncompletedOutbound(t *testing.T) {
	issuedQty, issuedWeight, closed := foldWmsIssued([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "PENDING", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutboundGI("COMPLETED", [2]float64{8, 80}),
		}),
		buildOrder("DBS-2", "ITEM-2", "COMPLETED", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutboundGI("PENDING", [2]float64{5, 50}),
		}),
	})

	if closed["DBS-1|ITEM-1"] {
		t.Error("CO ยัง PENDING ต้องไม่ถือว่าปิด")
	}
	if issuedQty["DBS-1|ITEM-1"] != 0 || issuedWeight["DBS-1|ITEM-1"] != 0 {
		t.Error("บรรทัดที่ CO ยังไม่ปิด ต้องไม่นับยอด GI")
	}
	if issuedQty["DBS-2|ITEM-2"] != 0 || issuedWeight["DBS-2|ITEM-2"] != 0 {
		t.Error("outbound ที่ยังไม่ COMPLETED ต้องไม่ถูกนับ")
	}
}

// SO202609-0009 บรรทัด 94e41ee0… = 50 PCS ถูกแบ่งจอง 3 ใบ 30 + 3 + 17
// ปิดครบทั้ง 3 ใบ -> รวมได้ 50 ชิ้น
func TestIssuedBySaleItemSumsAcrossEveryBooking(t *testing.T) {
	lines := []deliveryLine{
		{DeliveryCode: "DBS202609-0009", DeliveryItem: "ITEM-A", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0010", DeliveryItem: "ITEM-B", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0011", DeliveryItem: "ITEM-C", SaleItemCode: "SALE-1"},
	}

	issuedQty := map[string]float64{
		"DBS202609-0009|ITEM-A": 30,
		"DBS202609-0010|ITEM-B": 3,
		"DBS202609-0011|ITEM-C": 17,
	}
	issuedWeight := map[string]float64{
		"DBS202609-0009|ITEM-A": 87,
		"DBS202609-0010|ITEM-B": 8.7,
		"DBS202609-0011|ITEM-C": 49.3,
	}
	closed := map[string]bool{
		"DBS202609-0009|ITEM-A": true,
		"DBS202609-0010|ITEM-B": true,
		"DBS202609-0011|ITEM-C": true,
	}

	qty, weight := issuedBySaleItem(lines, issuedQty, issuedWeight, closed)

	if qty["SALE-1"] != 50 {
		t.Errorf("qty[SALE-1] = %v, want 50", qty["SALE-1"])
	}
	if weight["SALE-1"] != 145 {
		t.Errorf("weight[SALE-1] = %v, want 145", weight["SALE-1"])
	}
}

// ใบแรกจบแล้วแต่อีก 2 ใบยังไม่เดิน -> ต้องได้แค่ 30 ชิ้นและ 87 kg (ห้ามเพิ่มยอด pending)
func TestIssuedBySaleItemSkipsBookingsThatAreNotClosedYet(t *testing.T) {
	lines := []deliveryLine{
		{DeliveryCode: "DBS202609-0009", DeliveryItem: "ITEM-A", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0010", DeliveryItem: "ITEM-B", SaleItemCode: "SALE-1"},
		{DeliveryCode: "DBS202609-0011", DeliveryItem: "ITEM-C", SaleItemCode: "SALE-1"},
	}

	issuedQty := map[string]float64{
		"DBS202609-0009|ITEM-A": 30,
		"DBS202609-0010|ITEM-B": 3,
		"DBS202609-0011|ITEM-C": 17,
	}
	issuedWeight := map[string]float64{
		"DBS202609-0009|ITEM-A": 87,
		"DBS202609-0010|ITEM-B": 8.7,
		"DBS202609-0011|ITEM-C": 49.3,
	}
	closed := map[string]bool{"DBS202609-0009|ITEM-A": true}

	qty, weight := issuedBySaleItem(lines, issuedQty, issuedWeight, closed)

	if qty["SALE-1"] != 30 {
		t.Errorf("qty[SALE-1] = %v, want 30", qty["SALE-1"])
	}
	if weight["SALE-1"] != 87 {
		t.Errorf("weight[SALE-1] = %v, want 87", weight["SALE-1"])
	}
}

func TestIssuedBySaleItemIgnoresLinesWithoutSaleItem(t *testing.T) {
	lines := []deliveryLine{
		{DeliveryCode: "DBS-1", DeliveryItem: "ITEM-A", SaleItemCode: ""},
	}

	qty, weight := issuedBySaleItem(lines,
		map[string]float64{"DBS-1|ITEM-A": 9},
		map[string]float64{"DBS-1|ITEM-A": 9},
		map[string]bool{"DBS-1|ITEM-A": true})

	if len(qty) != 0 || len(weight) != 0 {
		t.Errorf("บรรทัดที่ไม่มี sale_item ต้องถูกข้าม ได้ qty=%v weight=%v", qty, weight)
	}
}

// SO202609-0011 = 10 PCS ส่งครบ 10 -> ปิด
// SO202609-0009 บรรทัด 50 PCS เพิ่งส่งไป 30 -> ไม่ปิด
// บรรทัดกิโล 500 kg ส่ง 487 (ขาด 2.6%) -> ปิด เพราะอยู่ในระยะผ่อนผัน 3%
func TestSelectDeliveredSaleItemsPicksOnlyLinesThatMetTheTarget(t *testing.T) {
	items := []models.SaleItem{
		{SaleItem: "SALE-DONE", Qty: 10, TotalWeight: 51.5, SaleUnit: "PC", SaleUnitType: "PC", Status: "PENDING"},
		{SaleItem: "SALE-PARTIAL", Qty: 50, TotalWeight: 145, SaleUnit: "PC", SaleUnitType: "PC", Status: "PENDING"},
		{SaleItem: "SALE-KG", Qty: 53, TotalWeight: 500, SaleUnit: "KG", SaleUnitType: "KG", Status: "PENDING"},
		{SaleItem: "SALE-ALREADY", Qty: 5, TotalWeight: 5, SaleUnit: "PC", SaleUnitType: "PC", Status: "COMPLETED"},
	}

	issuedQty := map[string]float64{
		"SALE-DONE":    10,
		"SALE-PARTIAL": 30,
		"SALE-KG":      53,
		"SALE-ALREADY": 5,
	}
	issuedWeight := map[string]float64{
		"SALE-DONE":    51.55,
		"SALE-PARTIAL": 87,
		"SALE-KG":      487,
		"SALE-ALREADY": 5,
	}

	got := selectDeliveredSaleItems(items, issuedQty, issuedWeight, 3)

	if len(got) != 2 {
		t.Fatalf("selectDeliveredSaleItems = %v, want 2 รายการ (SALE-DONE, SALE-KG)", got)
	}
	if got[0] != "SALE-DONE" || got[1] != "SALE-KG" {
		t.Errorf("selectDeliveredSaleItems = %v, want [SALE-DONE SALE-KG]", got)
	}
}

// บรรทัด KG_SPEC ต้องวัดด้วยชิ้น ไม่ใช่น้ำหนัก (SO202609-0001 = 6 ชิ้น / 120 kg)
func TestSelectDeliveredSaleItemsUsesQtyForKgSpec(t *testing.T) {
	items := []models.SaleItem{
		{SaleItem: "SALE-KGSPEC", Qty: 6, TotalWeight: 120, SaleUnit: "KG", SaleUnitType: "KG_SPEC", Status: "PENDING"},
	}

	got := selectDeliveredSaleItems(items,
		map[string]float64{"SALE-KGSPEC": 6},
		map[string]float64{"SALE-KGSPEC": 0},
		3)

	if len(got) != 1 || got[0] != "SALE-KGSPEC" {
		t.Errorf("selectDeliveredSaleItems = %v, want [SALE-KGSPEC] (ครบ 6 ชิ้นแล้วแม้ไม่มีน้ำหนัก)", got)
	}
}

func TestSelectDeliveredSaleItemsSkipsLinesWithNoIssueAtAll(t *testing.T) {
	items := []models.SaleItem{
		{SaleItem: "SALE-NONE", Qty: 10, SaleUnit: "PC", SaleUnitType: "PC", Status: "PENDING"},
	}

	got := selectDeliveredSaleItems(items, map[string]float64{}, map[string]float64{}, 3)

	if len(got) != 0 {
		t.Errorf("selectDeliveredSaleItems = %v, want ว่าง", got)
	}
}
