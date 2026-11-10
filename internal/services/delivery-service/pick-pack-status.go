package deliveryService

import (
	"strings"

	orderExternalService "prime-erp-core/external/order-service"
)

// กติกาการคำนวณสถานะ pick/pack ของ delivery booking หนึ่งใบ ก็อปพฤติกรรมมาจากฝั่ง
// browser ที่ฟังก์ชัน getPickPackStatus ใน wms-web:
// src/components/global/StatusTagPickPackDelivery.vue
// ถ้าจะแก้กติกานี้ ต้องแก้ทั้งสองที่พร้อมกัน ไม่งั้นค่าที่ filter ฝั่ง server กับ
// tag ที่ผู้ใช้เห็นในหน้าจอจะไม่ตรงกัน
const (
	PickPackStatusDraft       = "draft"
	PickPackStatusCanceled    = "canceled"
	PickPackStatusCompleted   = "completed"
	PickPackStatusNew         = "new"
	PickPackStatusPendingPick = "pending-pick"
	PickPackStatusPendingPack = "pending-pack"
)

// ComputePickPackStatus คำนวณ pick/pack status ของ delivery หนึ่งใบ จาก delivery status
// กับข้อมูล order ที่ผูกมาด้วย (delivery.Order) เป็น pure function ไม่แตะ GORM/gin
// เพื่อให้ unit test เรียกตรงๆ ได้
func ComputePickPackStatus(deliveryStatus string, order orderExternalService.GetOrderDeliveryResponse) string {
	switch strings.ToUpper(deliveryStatus) {
	case "TEMP":
		return PickPackStatusDraft
	case "CANCELED":
		return PickPackStatusCanceled
	case "COMPLETED":
		return PickPackStatusCompleted
	}

	if len(order.OrderItem) == 0 {
		return PickPackStatusNew // ยังไม่ถูกสร้าง outbound
	}

	for _, orderItem := range order.OrderItem {
		for _, outboundItem := range orderItem.OutboundItem {
			if len(outboundItem.FlowTracking) == 0 {
				continue
			}

			hasPickingPending := false
			hasPickingCompleted := false
			hasPackingSeen := false

			for _, flow := range outboundItem.FlowTracking {
				switch flow.ProcessCode {
				case "PICKING":
					if flow.Status == "PENDING" {
						hasPickingPending = true
					} else if flow.Status == "COMPLETED" {
						hasPickingCompleted = true
					}
				case "PACKING":
					if flow.Status == "PENDING" || flow.Status == "COMPLETED" {
						hasPackingSeen = true
					}
				}
			}

			// ธงเหล่านี้เป็นของ outbound item นี้เท่านั้น ไม่ไหลข้ามไป outbound item ถัดไป
			if hasPickingCompleted && hasPackingSeen {
				return PickPackStatusPendingPack
			}
			if hasPickingPending {
				return PickPackStatusPendingPick
			}
		}
	}

	return PickPackStatusNew // default case
}
