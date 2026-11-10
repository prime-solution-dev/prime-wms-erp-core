package deliveryService

import (
	"testing"

	orderExternalService "prime-erp-core/external/order-service"
)

func outboundState(status string, confirmQty float64) orderExternalService.OutboundItemWithGoodsIssue {
	item := orderExternalService.OutboundItemWithGoodsIssue{}
	item.Status = status
	item.ConfirmQty = confirmQty
	return item
}

func orderOf(ref string, outbounds ...orderExternalService.OutboundItemWithGoodsIssue) orderExternalService.GetOrderDeliveryResponse {
	order := orderExternalService.GetOrderDeliveryResponse{DocumentRef: ref}
	order.OrderItem = []orderExternalService.GetOrderItemDeliveryResponse{{OutboundItem: outbounds}}
	return order
}

// cancel pick / cancel pack ปิด outbound เป็น COMPLETED qty 0 ไม่ใช่ CANCELED
// เคส DBS202610-0004: เคยมี pick แล้ว cancel ทิ้ง ปุ่ม Cancel DBS หายถาวร
func TestOutboundBlocksCancel(t *testing.T) {
	cases := []struct {
		name    string
		items   []orderExternalService.OutboundItemWithGoodsIssue
		blocked bool
	}{
		{"no outbound", nil, false},
		{"canceled outbound", []orderExternalService.OutboundItemWithGoodsIssue{outboundState("CANCELED", 0)}, false},
		{"closed by cancel pick/pack", []orderExternalService.OutboundItemWithGoodsIssue{outboundState("COMPLETED", 0)}, false},
		{"goods already shipped", []orderExternalService.OutboundItemWithGoodsIssue{outboundState("COMPLETED", 5)}, true},
		{"pick/pack still running", []orderExternalService.OutboundItemWithGoodsIssue{outboundState("PENDING", 0)}, true},
		{"one closed one running", []orderExternalService.OutboundItemWithGoodsIssue{outboundState("COMPLETED", 0), outboundState("PICKING", 0)}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := outboundBlocksCancel(tc.items); got != tc.blocked {
				t.Errorf("outboundBlocksCancel = %v, want %v", got, tc.blocked)
			}
		})
	}
}

func TestSummarizeDeliveryWarehouse(t *testing.T) {
	orders := []orderExternalService.GetOrderDeliveryResponse{
		orderOf("DBS-1"),
		orderOf("DBS-2", outboundState("COMPLETED", 0)),
		orderOf("DBS-3", outboundState("PENDING", 0)),
	}

	got := summarizeDeliveryWarehouse(orders)

	if s := got["DBS-1"]; !s.HasOrder || s.Blocked {
		t.Errorf("DBS-1 = %+v, want order and not blocked", s)
	}
	if s := got["DBS-2"]; !s.HasOrder || s.Blocked {
		t.Errorf("DBS-2 = %+v, want order and not blocked", s)
	}
	if s := got["DBS-3"]; !s.HasOrder || !s.Blocked {
		t.Errorf("DBS-3 = %+v, want order and blocked", s)
	}
	// DBS ที่ไม่มี CO ใน WMS ต้องไม่ถูกส่งไปยกเลิก ไม่งั้น CancelOrderByCode ตอบ no order found
	if s := got["DBS-4"]; s.HasOrder || s.Blocked {
		t.Errorf("DBS-4 = %+v, want no order", s)
	}
}
