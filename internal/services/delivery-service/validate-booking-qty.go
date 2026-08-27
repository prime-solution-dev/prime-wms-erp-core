package deliveryService

import (
	"fmt"
	"strings"

	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/models"

	"gorm.io/gorm"
)

// closedOutboundStatuses คือสถานะที่ถือว่าคลังปิดงานบรรทัดนั้นแล้ว ไม่มีของค้างจองอยู่อีก
// ต้องตรงกับ CLOSED_OUTBOUND_STATUSES ใน wms-web/src/utils/helper/deliveryRemaining.ts
var closedOutboundStatuses = map[string]bool{
	"COMPLETED": true,
	"CANCELED":  true,
	"CANCELLED": true,
}

// bookingLine คือ 1 บรรทัดที่กำลังจะจอง ผูกกับ sale_item ผ่าน document_ref_item
type bookingLine struct {
	SaleCode        string
	DocumentRefItem string
	ProductCode     string
	Qty             float64
}

// ValidateBookingQty กันไม่ให้ยอดจองรวมของ sale item เกินจำนวนใน sale order
//
// เดิมไม่มีการเช็คฝั่ง server เลย create-delivery.go ไม่เคย query sale/sale_item ด้วยซ้ำ
// การ์ด "เหลือให้จองเท่าไร" อยู่ที่ computed ของหน้าจอที่เดียว ยิง API ตรงก็จองเกินได้
// ในข้อมูล UAT มี sale item ที่ SO 200 แต่จองไปแล้ว 800 ใน 4 ใบ
//
// นับยอดที่ถูกใช้ไปแล้วด้วยกติกาเดียวกับหน้าจอ (ต่อ 1 บรรทัดของใบจองอื่น):
//   - คลังปิดงานครบทุก outbound แล้ว -> นับเฉพาะที่ตัดจริง ส่วนที่ส่งขาดคืนมาให้จองใหม่ได้
//   - ยังมี outbound ค้าง หรือยังไม่เริ่ม -> นับจำนวนที่จองไว้เต็ม เพราะของยังถูกกันอยู่
//
// excludeDeliveryCodes ใช้ตอนแก้ใบเดิม จะได้ไม่นับจำนวนของตัวเองซ้ำ
func ValidateBookingQty(gormx *gorm.DB, lines []bookingLine, excludeDeliveryCodes []string) error {
	if len(lines) == 0 {
		return nil
	}

	// รวม qty ที่ขอจองรอบนี้ ต่อ 1 sale item (payload เดียวอาจมีหลายใบของ SO เดียวกัน)
	requestedQty := map[string]float64{}
	productOf := map[string]string{}
	saleCodes := []string{}
	saleCodeSeen := map[string]bool{}

	for _, line := range lines {
		if line.DocumentRefItem == "" {
			continue
		}

		requestedQty[line.DocumentRefItem] += line.Qty
		productOf[line.DocumentRefItem] = line.ProductCode

		if line.SaleCode != "" && !saleCodeSeen[line.SaleCode] {
			saleCodes = append(saleCodes, line.SaleCode)
			saleCodeSeen[line.SaleCode] = true
		}
	}

	if len(requestedQty) == 0 || len(saleCodes) == 0 {
		return nil
	}

	saleItemQty, err := loadSaleItemQty(gormx, saleCodes)
	if err != nil {
		return err
	}

	usedQty, err := loadBookedQty(gormx, saleCodes, excludeDeliveryCodes)
	if err != nil {
		return err
	}

	// จำนวนที่ใบซึ่งกำลังแก้จองไว้อยู่เดิม ใช้ยอมให้ "ลดหรือคงเดิม" ได้เสมอ
	// ไม่งั้นใบที่ over-booked อยู่แล้ว (เช่นของเก่าที่ SO 200 แต่จองไป 800)
	// จะแก้เพื่อลดจำนวนไม่ได้เลย ต้องยกเลิกแล้วสร้างใหม่อย่างเดียว
	previousQty, err := loadCurrentBookedQty(gormx, excludeDeliveryCodes)
	if err != nil {
		return err
	}

	overBooked := []string{}
	for saleItem, qty := range requestedQty {
		orderQty, exists := saleItemQty[saleItem]
		if !exists {
			// ไม่เจอ sale item ที่อ้างถึง ปล่อยผ่านเพื่อไม่ให้ข้อมูลเก่าที่คีย์ค้างไว้ submit ไม่ได้
			continue
		}

		remain := orderQty - usedQty[saleItem]
		if qty <= remain || qty <= previousQty[saleItem] {
			continue
		}

		overBooked = append(overBooked, fmt.Sprintf(
			"%s (order %v, booked %v, remaining %v, requested %v)",
			productOf[saleItem], orderQty, usedQty[saleItem], remain, qty,
		))
	}

	if len(overBooked) > 0 {
		return fmt.Errorf("booking qty exceeds the sale order for: %s", strings.Join(overBooked, "; "))
	}

	return nil
}

// loadSaleItemQty คืน map ของ sale_item -> qty ที่สั่งไว้ใน sale order
func loadSaleItemQty(gormx *gorm.DB, saleCodes []string) (map[string]float64, error) {
	var saleItems []models.SaleItem

	err := gormx.Model(&models.SaleItem{}).
		Select("sale_item.*").
		Joins("join sale on sale.id = sale_item.sale_id").
		Where("sale.sale_code IN ?", saleCodes).
		Find(&saleItems).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load sale items: %v", err)
	}

	qtyOf := map[string]float64{}
	for _, item := range saleItems {
		if item.SaleItem == "" {
			continue
		}
		qtyOf[item.SaleItem] += item.Qty
	}

	return qtyOf, nil
}

// loadCurrentBookedQty คืน map ของ sale_item -> จำนวนที่ใบเหล่านี้จองไว้อยู่ตอนนี้
// ใช้เฉพาะตอนแก้ใบเดิม เพื่อรู้ว่า "ของเดิม" คือเท่าไร จะได้ยอมให้ลดจำนวนได้เสมอ
func loadCurrentBookedQty(gormx *gorm.DB, deliveryCodes []string) (map[string]float64, error) {
	qtyOf := map[string]float64{}
	if len(deliveryCodes) == 0 {
		return qtyOf, nil
	}

	var bookedItems []models.DeliveryItem
	err := gormx.Model(&models.DeliveryItem{}).
		Select("delivery_booking_item.*").
		Joins("join delivery_booking on delivery_booking.id = delivery_booking_item.delivery_id").
		Where("delivery_booking.delivery_code IN ?", deliveryCodes).
		Find(&bookedItems).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load current delivery booking items: %v", err)
	}

	for _, item := range bookedItems {
		if item.DocumentRefItem == "" {
			continue
		}
		qtyOf[item.DocumentRefItem] += item.Qty
	}

	return qtyOf, nil
}

// loadBookedQty คืน map ของ sale_item -> จำนวนที่ใบจองอื่นใช้ไปแล้ว
func loadBookedQty(gormx *gorm.DB, saleCodes []string, excludeDeliveryCodes []string) (map[string]float64, error) {
	var deliveries []models.Delivery

	query := gormx.Model(&models.Delivery{}).
		Where("document_ref IN ?", saleCodes).
		Where("status IN ?", []string{"PENDING", "COMPLETED"})

	if len(excludeDeliveryCodes) > 0 {
		query = query.Where("delivery_code NOT IN ?", excludeDeliveryCodes)
	}

	if err := query.Find(&deliveries).Error; err != nil {
		return nil, fmt.Errorf("failed to load existing delivery bookings: %v", err)
	}

	usedQty := map[string]float64{}
	if len(deliveries) == 0 {
		return usedQty, nil
	}

	deliveryIDs := make([]string, 0, len(deliveries))
	deliveryCodeOf := map[string]string{}
	deliveryCodes := make([]string, 0, len(deliveries))
	for _, delivery := range deliveries {
		deliveryIDs = append(deliveryIDs, delivery.ID.String())
		deliveryCodeOf[delivery.ID.String()] = delivery.DeliveryCode
		deliveryCodes = append(deliveryCodes, delivery.DeliveryCode)
	}

	var bookedItems []models.DeliveryItem
	if err := gormx.Model(&models.DeliveryItem{}).
		Where("delivery_id IN ?", deliveryIDs).
		Find(&bookedItems).Error; err != nil {
		return nil, fmt.Errorf("failed to load existing delivery booking items: %v", err)
	}

	if len(bookedItems) == 0 {
		return usedQty, nil
	}

	issuedQty, closed := loadWmsProgress(deliveryCodes)

	for _, item := range bookedItems {
		if item.DocumentRefItem == "" {
			continue
		}

		key := fmt.Sprintf("%s|%s", deliveryCodeOf[item.DeliveryID.String()], item.DeliveryItem)
		if closed[key] {
			usedQty[item.DocumentRefItem] += issuedQty[key]
			continue
		}

		usedQty[item.DocumentRefItem] += item.Qty
	}

	return usedQty, nil
}

// loadWmsProgress ถาม WMS ว่าแต่ละ delivery item ตัดของไปแล้วเท่าไรและปิดงานหรือยัง
// คีย์ของทั้งสอง map คือ "<delivery_code>|<delivery_item>"
//
// ถ้าถาม WMS ไม่ได้จะคืน map ว่าง ซึ่งทำให้ทุกบรรทัดถูกนับเป็น "ยังจองค้าง" เต็มจำนวน
// คือเข้มกว่าความจริง ยอมให้จองไม่ได้ ดีกว่าปล่อยให้จองเกินตอน WMS ล่ม
func loadWmsProgress(deliveryCodes []string) (map[string]float64, map[string]bool) {
	orderRes, err := orderExternalService.GetOrdersDelivery(orderExternalService.GetOrderDeliveryRequest{
		DeliveryCode: deliveryCodes,
	})
	if err != nil {
		fmt.Printf("ValidateBookingQty: cannot read WMS progress, treating every booking as still held: %v\n", err)
		return map[string]float64{}, map[string]bool{}
	}

	return foldWmsProgress(orderRes.Orders)
}

// foldWmsProgress แยกออกมาเพื่อให้เทสกติกา closed/issued ได้โดยไม่ต้องมี WMS จริง
func foldWmsProgress(orders []orderExternalService.GetOrderDeliveryResponse) (map[string]float64, map[string]bool) {
	issuedQty := map[string]float64{}
	closed := map[string]bool{}

	for _, order := range orders {
		for _, orderItem := range order.OrderItem {
			outbounds := orderItem.OutboundItem
			if len(outbounds) == 0 {
				continue
			}

			key := fmt.Sprintf("%s|%s", order.DocumentRef, orderItem.DocumentRefItem)

			isClosed := true
			for _, outbound := range outbounds {
				if !closedOutboundStatuses[outbound.Status] {
					isClosed = false
					break
				}
			}

			if !isClosed {
				continue
			}

			closed[key] = true
			for _, outbound := range outbounds {
				if outbound.Status != "COMPLETED" {
					continue
				}
				for _, issued := range outbound.GoodsIssueItem {
					issuedQty[key] += issued.Qty
				}
			}
		}
	}

	return issuedQty, closed
}
