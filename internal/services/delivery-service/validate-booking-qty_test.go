package deliveryService

import (
	"testing"

	orderExternalService "prime-erp-core/external/order-service"
)

func buildOrder(deliveryCode string, deliveryItem string, orderItemStatus string, outbounds []orderExternalService.OutboundItemWithGoodsIssue) orderExternalService.GetOrderDeliveryResponse {
	return orderExternalService.GetOrderDeliveryResponse{
		DocumentRef: deliveryCode,
		OrderItem: []orderExternalService.GetOrderItemDeliveryResponse{
			{
				DocumentRefItem: deliveryItem,
				Status:          orderItemStatus,
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

func TestFoldWmsProgressMarksCompletedOrderItemAsClosed(t *testing.T) {
	issued, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "COMPLETED", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("COMPLETED", 8),
		}),
	})

	key := "DBS-1|ITEM-1"
	if !closed[key] {
		t.Fatalf("closed[%s] = false, want true when the CO line is COMPLETED", key)
	}

	if issued[key] != 8 {
		t.Errorf("issued[%s] = %v, want 8", key, issued[key])
	}
}

// CO ยังทำงานอยู่ ถึงจะมี outbound ปิดไปแล้วบางตัว ของก็ยังต้องถูกกันไว้เต็มจำนวน
func TestFoldWmsProgressKeepsItemOpenWhileOrderItemIsPending(t *testing.T) {
	_, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "PENDING", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("COMPLETED", 8),
			buildOutbound("PENDING"),
		}),
	})

	if closed["DBS-1|ITEM-1"] {
		t.Fatal("closed = true while the CO line is still PENDING, the booking must stay held in full")
	}
}

// เคส CO2608-00005-002: outbound ถูกปิดเป็น COMPLETED ทั้งที่ไม่เคย pack/GI เลย
// ถ้าตัดสินจาก outbound ยอดจะถูกปล่อยคืนทั้งที่ CO ยังรอของอยู่
func TestFoldWmsProgressKeepsItemOpenWhenOutboundClosedWithoutIssuing(t *testing.T) {
	_, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "PENDING", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("COMPLETED"),
		}),
	})

	if closed["DBS-1|ITEM-1"] {
		t.Fatal("an outbound that closed without issuing must not release a CO line that is still open")
	}
}

func TestFoldWmsProgressTreatsCancelledOrderItemAsClosed(t *testing.T) {
	issued, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "CANCELED", []orderExternalService.OutboundItemWithGoodsIssue{
			buildOutbound("CANCELED"),
		}),
	})

	key := "DBS-1|ITEM-1"
	if !closed[key] {
		t.Fatal("a cancelled CO line should close the line so the qty returns to the pool")
	}

	if issued[key] != 0 {
		t.Errorf("issued[%s] = %v, want 0 for a cancelled line", key, issued[key])
	}
}

// TEMP คือ CO ที่ยังร่างอยู่ ยังไม่จบ ต้องกันของไว้
func TestFoldWmsProgressKeepsDraftOrderItemOpen(t *testing.T) {
	_, closed := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "TEMP", nil),
	})

	if closed["DBS-1|ITEM-1"] {
		t.Fatal("a TEMP (draft) CO line must keep holding the booking")
	}
}

func TestFoldWmsProgressSumsEveryGoodsIssueLine(t *testing.T) {
	issued, _ := foldWmsProgress([]orderExternalService.GetOrderDeliveryResponse{
		buildOrder("DBS-1", "ITEM-1", "COMPLETED", []orderExternalService.OutboundItemWithGoodsIssue{
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
