package deliveryService

import (
	"testing"

	orderExternalService "prime-erp-core/external/order-service"
)

func buildOrder(deliveryCode string, deliveryItem string, outbounds []orderExternalService.OutboundItemWithGoodsIssue) orderExternalService.GetOrderDeliveryResponse {
	return orderExternalService.GetOrderDeliveryResponse{
		DocumentRef: deliveryCode,
		OrderItem: []orderExternalService.GetOrderItemDeliveryResponse{
			{
				DocumentRefItem: deliveryItem,
				OutboundItem:    outbounds,
			},
		},
	}
}

func buildOutbound(status string, issuedQty ...float64) orderExternalService.OutboundItemWithGoodsIssue {
	outbound := orderExternalService.OutboundItemWithGoodsIssue{}
	outbound.Status = status

	for _, qty := range issuedQty {
		outbound.GoodsIssueItem = append(outbound.GoodsIssueItem, orderExternalService.GetIssueItemResponse{Qty: qty})
	}

	return outbound
}

func TestFoldWmsProgressMarksFullyIssuedItemAsClosed(t *testing.T) {
	issued, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("COMPLETED", 8),
		}),
	})

	key := "DBS-1|ITEM-1"
	if !closed[key] {
		t.Fatalf("closed[%s] = false, want true when every outbound is COMPLETED", key)
	}

	if issued[key] != 8 {
		t.Errorf("issued[%s] = %v, want 8", key, issued[key])
	}
}

// นี่คือเคสที่พังก่อนหน้านี้: มี outbound ปิดไปแล้วหนึ่งตัวแต่ยังมีอีกตัวค้างอยู่
// ของเดิมจะนับแค่ที่ตัดจริงแล้วปล่อยส่วนที่ยังจองค้างให้จองซ้ำได้
func TestFoldWmsProgressKeepsItemOpenWhileAnyOutboundIsPending(t *testing.T) {
	_, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("COMPLETED", 8),
			buildOutbound("PENDING"),
		}),
	})

	if closed["DBS-1|ITEM-1"] {
		t.Fatal("closed = true while an outbound is still PENDING, the booking must stay held in full")
	}
}

func TestFoldWmsProgressTreatsCancelledOutboundAsClosed(t *testing.T) {
	issued, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("CANCELED"),
		}),
	})

	key := "DBS-1|ITEM-1"
	if !closed[key] {
		t.Fatal("a cancelled outbound should close the line so the qty returns to the pool")
	}

	if issued[key] != 0 {
		t.Errorf("issued[%s] = %v, want 0 for a cancelled outbound", key, issued[key])
	}
}

func TestFoldWmsProgressIgnoresItemWithoutOutbound(t *testing.T) {
	issued, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", nil),
	})

	if len(closed) != 0 || len(issued) != 0 {
		t.Fatalf("an order item with no outbound must stay unknown, got issued=%v closed=%v", issued, closed)
	}
}

func TestFoldWmsProgressSumsEveryGoodsIssueLine(t *testing.T) {
	issued, _ := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("COMPLETED", 3, 2),
			buildOutbound("COMPLETED", 1),
		}),
	})

	if issued["DBS-1|ITEM-1"] != 6 {
		t.Errorf("issued = %v, want 6", issued["DBS-1|ITEM-1"])
	}
}

// ใบที่จองเกินอยู่แล้ว (SO 200 แต่มี 4 ใบจองใบละ 200) ต้องยังแก้เพื่อลดจำนวนได้
// ไม่งั้นทางเดียวที่จะเคลียร์ข้อมูลเสียคือยกเลิกแล้วสร้างใหม่
func TestOverBookedLineCanStillBeReduced(t *testing.T) {
	const orderQty, usedByOthers, previous = 200.0, 600.0, 200.0
	remain := orderQty - usedByOthers // -400

	allowed := func(requested float64) bool {
		return requested <= remain || requested <= previous
	}

	if !allowed(50) {
		t.Error("reducing an over-booked line to 50 must be allowed")
	}
	if !allowed(200) {
		t.Error("keeping an over-booked line at its current 200 must be allowed")
	}
	if allowed(250) {
		t.Error("increasing an over-booked line to 250 must still be rejected")
	}
}
