package deliveryService

import (
	"testing"

	orderExternalService "prime-erp-core/external/order-service"
)

// flow สร้าง flow tracking entry แบบสั้นๆ สำหรับ test นี้
func flow(processCode, status string) orderExternalService.OutboundFlowTracking {
	return orderExternalService.OutboundFlowTracking{ProcessCode: processCode, Status: status}
}

// outboundWithFlow สร้าง outbound item ที่มี flow tracking ตามที่ระบุ (ไม่ระบุเลย = ไม่มี flow tracking)
func outboundWithFlow(flows ...orderExternalService.OutboundFlowTracking) orderExternalService.OutboundItemWithGoodsIssue {
	item := orderExternalService.OutboundItemWithGoodsIssue{}
	item.FlowTracking = flows
	return item
}

// orderWithOutbounds สร้าง order หนึ่งใบที่มี order item เดียว ถือ outbound items ตามที่ระบุ
func orderWithOutbounds(outbounds ...orderExternalService.OutboundItemWithGoodsIssue) orderExternalService.GetOrderDeliveryResponse {
	return orderExternalService.GetOrderDeliveryResponse{
		OrderItem: []orderExternalService.GetOrderItemDeliveryResponse{
			{OutboundItem: outbounds},
		},
	}
}

func TestComputePickPackStatus(t *testing.T) {
	tests := []struct {
		name           string
		deliveryStatus string
		order          orderExternalService.GetOrderDeliveryResponse
		want           string
	}{
		{
			name:           "delivery status TEMP short-circuits to draft even with order items present",
			deliveryStatus: "TEMP",
			order:          orderWithOutbounds(outboundWithFlow(flow("PICKING", "PENDING"))),
			want:           PickPackStatusDraft,
		},
		{
			name:           "delivery status match is case-insensitive",
			deliveryStatus: "temp",
			order:          orderExternalService.GetOrderDeliveryResponse{},
			want:           PickPackStatusDraft,
		},
		{
			name:           "delivery status CANCELED short-circuits to canceled",
			deliveryStatus: "CANCELED",
			order:          orderWithOutbounds(outboundWithFlow(flow("PICKING", "PENDING"))),
			want:           PickPackStatusCanceled,
		},
		{
			name:           "delivery status COMPLETED short-circuits to completed",
			deliveryStatus: "COMPLETED",
			order:          orderWithOutbounds(outboundWithFlow(flow("PICKING", "PENDING"))),
			want:           PickPackStatusCompleted,
		},
		{
			name:           "no order items at all gives new",
			deliveryStatus: "PENDING",
			order:          orderExternalService.GetOrderDeliveryResponse{},
			want:           PickPackStatusNew,
		},
		{
			name:           "outbound item with picking pending gives pending-pick",
			deliveryStatus: "PENDING",
			order:          orderWithOutbounds(outboundWithFlow(flow("PICKING", "PENDING"))),
			want:           PickPackStatusPendingPick,
		},
		{
			name:           "picking completed with packing pending on the same outbound item gives pending-pack",
			deliveryStatus: "PENDING",
			order: orderWithOutbounds(outboundWithFlow(
				flow("PICKING", "COMPLETED"),
				flow("PACKING", "PENDING"),
			)),
			want: PickPackStatusPendingPack,
		},
		{
			name:           "picking completed with packing completed on the same outbound item gives pending-pack",
			deliveryStatus: "PENDING",
			order: orderWithOutbounds(outboundWithFlow(
				flow("PICKING", "COMPLETED"),
				flow("PACKING", "COMPLETED"),
			)),
			want: PickPackStatusPendingPack,
		},
		{
			name:           "outbound item with no flow tracking is skipped, the next matching one wins",
			deliveryStatus: "PENDING",
			order: orderWithOutbounds(
				outboundWithFlow(), // no flow tracking entries at all
				outboundWithFlow(flow("PICKING", "PENDING")),
			),
			want: PickPackStatusPendingPick,
		},
		{
			name:           "picking-completed flag on one outbound item must not combine with packing on a different outbound item",
			deliveryStatus: "PENDING",
			order: orderWithOutbounds(
				outboundWithFlow(flow("PICKING", "COMPLETED")),
				outboundWithFlow(flow("PACKING", "PENDING")),
			),
			want: PickPackStatusNew,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePickPackStatus(tt.deliveryStatus, tt.order)
			if got != tt.want {
				t.Errorf("ComputePickPackStatus(%q, ...) = %q, want %q", tt.deliveryStatus, got, tt.want)
			}
		})
	}
}
