package deliveryService

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	orderExternalService "prime-erp-core/external/order-service"
	"prime-erp-core/internal/models"
	systemConfigRepository "prime-erp-core/internal/repositories/systemConfig"
	saleService "prime-erp-core/internal/services/sale-service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// โหมดตัดสินว่า "ส่งครบ" ของ 1 บรรทัดขาย ดูจากจำนวนชิ้นหรือจากน้ำหนัก
const (
	completionModeQty    = "QTY"
	completionModeWeight = "WEIGHT"
)

// saleItemCompletionMode บอกว่าบรรทัดนี้ต้องวัดความครบด้วยชิ้นหรือด้วยน้ำหนัก
//
// กติกาเดียวกับฝั่งจอ (wms-web packingCreate mapUnitCodeByOrder และ deliverySlotCreate:401)
//   - ขายเป็นกิโล (sale_unit = KG) -> วัดด้วยน้ำหนัก
//   - ยกเว้น KG_SPEC ที่เป็นของน้ำหนักต่อชิ้นคงที่ ลูกค้าสั่งเป็นชิ้น -> วัดด้วยชิ้น
//   - นอกนั้น (PC) -> วัดด้วยชิ้น
func saleItemCompletionMode(saleUnit string, saleUnitType string) string {
	unit := strings.ToUpper(strings.TrimSpace(saleUnit))
	unitType := strings.ToUpper(strings.TrimSpace(saleUnitType))
	unitType = strings.ReplaceAll(unitType, "-", "_")

	if unit == "KG" && unitType != "KG_SPEC" {
		return completionModeWeight
	}

	return completionModeQty
}

// completionTarget คือยอดที่ต้องส่งให้ถึงของบรรทัดนั้น ตามโหมดที่ใช้วัด
func completionTarget(item models.SaleItem) float64 {
	if saleItemCompletionMode(item.SaleUnit, item.SaleUnitType) == completionModeWeight {
		return item.TotalWeight
	}

	return item.Qty
}

// isFullyDelivered ถือว่าครบเมื่อยอดที่ตัดจ่ายจริงถึงขอบล่างของระยะผ่อนผัน
//
// ไม่มีเพดานบน — ส่งเกินก็ถือว่าจบงานแล้ว (โค้ดเดิมฝั่ง wms-outbound ใช้กรอบ min..max
// ทำให้ใบที่ส่งเกินไม่ถูกปิดตลอดกาล)
func isFullyDelivered(target float64, issued float64, tolerancePercent float64) bool {
	if target <= 0 {
		return false
	}

	if tolerancePercent < 0 {
		tolerancePercent = 0
	}

	return issued >= target*(1-tolerancePercent/100)
}

// isCanceledStatus บอกว่าสถานะนั้นคือ "ยกเลิก" หรือไม่ รับทุกวิธีสะกดที่เจอในระบบ
// (CANCELED / CANCELLED / ตัวพิมพ์เล็ก / มีช่องว่าง / เขียนด้วยขีดกลาง)
// normalise แบบเดียวกับ saleItemCompletionMode เพื่อไม่ให้สองที่เพี้ยนออกจากกัน
func isCanceledStatus(status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	normalized = strings.ReplaceAll(normalized, "-", "_")

	return normalized == "CANCELED" || normalized == "CANCELLED"
}

// foldWmsDelivered พับความคืบหน้าฝั่งคลังสำหรับ "เส้นปิด SO" โดยเฉพาะ
// คืนรูปเดียวกับ foldWmsIssued: issuedQty, issuedWeight, closed คีย์ "<delivery_code>|<delivery_item>"
//
// ต่างจาก foldWmsIssued ที่จุดเดียว: ตัวนี้ "ไม่ดู outbound.Status" เลย
//
// เหตุผล — hook ORDER/DELIVERY/UPDATE ที่พาเรามาถึง UpdateStatusDelivery ถูกยิงจาก
// wms-outbound-service externalConfirmOrderOutbound (update-flow-tracking-packing.go:394)
// ขณะที่ tx ซึ่งเขียน `update outbound_item set status = ?` (:279) ยังไม่ commit
// (tx เปิดที่ :127 commit ใน defer :135-147) erp-core อ่านสถานะกลับมาทาง HTTP
// บน session ใหม่ จึงเห็น outbound_item.status = 'PENDING' เสมอ
// ถ้า gate ด้วย outbound.Status ยอดที่ตัดจ่ายจะเป็น 0 ทุกบรรทัด และไม่มี SO ไหนถูกปิดเลย
//
// สิ่งที่เชื่อได้ตอน hook ยิงคือ order_item.status ซึ่งถูก commit ใน tx แรกของ
// wms-order-service ConfirmOrderOutbound ก่อน hook จะทำงาน จึงยังใช้ closedOrderItemStatuses ได้
//
// ระดับบรรทัด GI นับทุกบรรทัดที่ไม่ถูกยกเลิก — สถานะว่างก็ต้องนับ เพราะข้อมูลเก่าเว้นช่องนี้ไว้
//
// ห้ามย้ายกติกานี้ไปใส่ foldWmsIssued/foldWmsProgress เพราะ ValidateBookingQty ต้องคง
// การ์ด outbound เดิมไว้ตามที่ SA เคาะ (ดูคอมเมนต์ validate-booking-qty.go:13-19)
func foldWmsDelivered(orders []orderExternalService.GetOrderDeliveryResponse) (map[string]float64, map[string]float64, map[string]bool) {
	issuedQty := map[string]float64{}
	issuedWeight := map[string]float64{}
	closed := map[string]bool{}

	for _, order := range orders {
		for _, orderItem := range order.OrderItem {
			if !closedOrderItemStatuses[orderItem.Status] {
				continue
			}

			key := fmt.Sprintf("%s|%s", order.DocumentRef, orderItem.DocumentRefItem)
			closed[key] = true

			for _, outbound := range orderItem.OutboundItem {
				for _, issued := range outbound.GoodsIssueItem {
					if isCanceledStatus(issued.Status) {
						continue
					}

					issuedQty[key] += issued.Qty
					issuedWeight[key] += issued.Weight
				}
			}
		}
	}

	return issuedQty, issuedWeight, closed
}

// deliveryLine คือ 1 บรรทัดของใบจอง ที่ผูกกับบรรทัดขายผ่าน document_ref_item
type deliveryLine struct {
	DeliveryCode string
	DeliveryItem string
	SaleItemCode string
}

// issuedBySaleItem รวมยอดที่ตัดจ่ายจริงของทุกใบจอง กลับมาเป็นยอดต่อ 1 บรรทัดขาย
//
// นับเฉพาะบรรทัดที่ฝั่งคลังปิดงานแล้ว บรรทัดที่ยังเดินอยู่แปลว่าของยังไม่ออก
// (ใบจองใบอื่นของ SO เดียวกันที่ยังไม่ถึงคิว ต้องไม่ทำให้ SO ถูกปิดก่อนเวลา)
func issuedBySaleItem(lines []deliveryLine, issuedQty map[string]float64, issuedWeight map[string]float64, closed map[string]bool) (map[string]float64, map[string]float64) {
	qtyOf := map[string]float64{}
	weightOf := map[string]float64{}

	for _, line := range lines {
		if line.SaleItemCode == "" {
			continue
		}

		key := fmt.Sprintf("%s|%s", line.DeliveryCode, line.DeliveryItem)
		if !closed[key] {
			continue
		}

		qtyOf[line.SaleItemCode] += issuedQty[key]
		weightOf[line.SaleItemCode] += issuedWeight[key]
	}

	return qtyOf, weightOf
}

// defaultToleranceSOPercent ใช้เมื่ออ่าน config ไม่ได้ — ต้องเป็น 0 (fail closed)
//
// ค่าจริงของ TMI ตอนนี้คือ 3 ค่า default ที่ "หลวมกว่าค่าจริง" (เดิม 5) แปลว่าเวลา
// system_config อ่านไม่ได้หรือ parse ไม่ผ่าน ระยะผ่อนผันจะกว้างขึ้นเอง แล้วเส้นนี้
// ไปเขียน sale/sale_item จริง ปิดใบที่ยังส่งไม่ครบได้ ย้อนคืนไม่ได้
// 0 = ต้องส่งครบเป๊ะจึงปิด อ่าน config ไม่ได้ก็แค่ปิดช้ากว่าที่ควร ไม่ปิดผิด
const defaultToleranceSOPercent = 0.0

// selectDeliveredSaleItems คืนรหัสบรรทัดขายที่ส่งของถึงเป้าแล้วและยังไม่ถูกปิด
//
// กติกา "บรรทัดนี้จบงานแล้ว" ใช้ของ sale-service ที่เดียว (saleService.IsSaleItemClosed)
// ไม่ประกาศ map ซ้ำ เพื่อไม่ให้สองที่เพี้ยนออกจากกันภายหลัง
func selectDeliveredSaleItems(items []models.SaleItem, issuedQty map[string]float64, issuedWeight map[string]float64, tolerancePercent float64) []string {
	delivered := []string{}

	for _, item := range items {
		if saleService.IsSaleItemClosed(item.Status) {
			continue
		}

		issued := issuedQty[item.SaleItem]
		if saleItemCompletionMode(item.SaleUnit, item.SaleUnitType) == completionModeWeight {
			issued = issuedWeight[item.SaleItem]
		}

		if !isFullyDelivered(completionTarget(item), issued, tolerancePercent) {
			continue
		}

		delivered = append(delivered, item.SaleItem)
	}

	return delivered
}

// describeCandidates ทำสรุป "บรรทัดที่พิจารณาแล้วยังไม่ปิด" ให้ log อ่านออก
//
// เส้นนี้เงียบมาก คืน nil ตอนไม่มีอะไรเข้าเกณฑ์ ทำให้บั๊กที่ยอดถูกนับเป็น 0 ทุกบรรทัด
// (เคส gate ด้วย outbound.Status) หลุดขึ้น production ไปโดยไม่มีร่องรอยอะไรเลย
// บรรทัดที่ปิดไปแล้วไม่ต้องรายงาน เพราะไม่ใช่ผู้ต้องสงสัย
func describeCandidates(items []models.SaleItem, issuedQty map[string]float64, issuedWeight map[string]float64) string {
	parts := []string{}

	for _, item := range items {
		if saleService.IsSaleItemClosed(item.Status) {
			continue
		}

		mode := saleItemCompletionMode(item.SaleUnit, item.SaleUnitType)
		parts = append(parts, fmt.Sprintf("%s[%s status=%s target=%v issued_qty=%v issued_weight=%v]",
			item.SaleItem, mode, item.Status, completionTarget(item),
			issuedQty[item.SaleItem], issuedWeight[item.SaleItem]))
	}

	if len(parts) == 0 {
		return "no open sale item"
	}

	return strings.Join(parts, " ")
}

// toleranceSOPercent อ่านระยะผ่อนผันจาก system_config ของ ERP เอง (topic SO / TOLERANCE_SO)
func toleranceSOPercent() float64 {
	configs, err := systemConfigRepository.GetSystemConfig([]string{"SO"}, []string{"TOLERANCE_SO"})
	if err != nil || len(configs) == 0 {
		return defaultToleranceSOPercent
	}

	parsed, err := strconv.ParseFloat(strings.TrimSpace(configs[0].Value), 64)
	if err != nil {
		return defaultToleranceSOPercent
	}

	return parsed
}

// CloseSalesFullyDelivered ปิดบรรทัดขาย (และหัวใบเมื่อครบทุกบรรทัด) ของ SO ที่ส่งของครบแล้ว
//
// เรียกหลัง commit ของ UpdateStatusDelivery เท่านั้น และผู้เรียกต้อง "log ทิ้ง" ถ้าพัง
// ห้ามคืน error ขึ้นไปให้ hook เพราะ hook ORDER/DELIVERY/UPDATE ถูกยิงระหว่างที่
// wms-order-service ยังไม่ commit การคืน error จะทำให้ pack confirm ทั้งใบล้ม
// ทั้งที่สต็อกกับ GI ตัดไปแล้ว (ดูคอมเมนต์ที่ confirm-order-outbound.go:205-208)
func CloseSalesFullyDelivered(gormx *gorm.DB, saleCodes []string, user string) error {
	codes := []string{}
	seen := map[string]bool{}
	for _, saleCode := range saleCodes {
		if saleCode == "" || seen[saleCode] {
			continue
		}
		seen[saleCode] = true
		codes = append(codes, saleCode)
	}

	if len(codes) == 0 {
		return nil
	}

	// อ่านทุกใบแล้วคัดด้วย saleService.IsSaleItemClosed ไม่คัดใน SQL
	// เพราะ SQL literal เดิม ("COMPLETED","CANCELED") ไม่รู้จัก CANCELLED สองแอล
	// กติกา "จบงานแล้ว" ต้องมีที่เดียวคือ predicate ของ sale-service
	var sales []models.Sale
	if err := gormx.Where("sale_code IN ?", codes).Find(&sales).Error; err != nil {
		return fmt.Errorf("failed to load sales %v: %v", codes, err)
	}

	saleIDs := make([]uuid.UUID, 0, len(sales))
	openSaleCodes := make([]string, 0, len(sales))
	for _, sale := range sales {
		if saleService.IsSaleItemClosed(sale.Status) {
			continue
		}

		saleIDs = append(saleIDs, sale.ID)
		openSaleCodes = append(openSaleCodes, sale.SaleCode)
	}

	if len(openSaleCodes) == 0 {
		return nil
	}

	var saleItems []models.SaleItem
	if err := gormx.Where("sale_id IN ?", saleIDs).Find(&saleItems).Error; err != nil {
		return fmt.Errorf("failed to load sale items of %v: %v", openSaleCodes, err)
	}

	if len(saleItems) == 0 {
		return nil
	}

	// ใบจองทุกใบของ SO เหล่านี้ ใบที่ยกเลิกไม่นับเพราะของถูกคืนไปแล้ว
	// ต้องกันทั้งสองวิธีสะกด CANCELED/CANCELLED เหมือนที่ predicate ฝั่ง Go ทำ
	var deliveries []models.Delivery
	if err := gormx.Where("document_ref IN ? AND status NOT IN ?", openSaleCodes, []string{"CANCELED", "CANCELLED"}).
		Find(&deliveries).Error; err != nil {
		return fmt.Errorf("failed to load delivery bookings of %v: %v", openSaleCodes, err)
	}

	if len(deliveries) == 0 {
		return nil
	}

	deliveryIDs := make([]uuid.UUID, 0, len(deliveries))
	deliveryCodes := make([]string, 0, len(deliveries))
	deliveryCodeOf := map[uuid.UUID]string{}
	for _, delivery := range deliveries {
		deliveryIDs = append(deliveryIDs, delivery.ID)
		deliveryCodes = append(deliveryCodes, delivery.DeliveryCode)
		deliveryCodeOf[delivery.ID] = delivery.DeliveryCode
	}

	var bookedItems []models.DeliveryItem
	if err := gormx.Where("delivery_id IN ?", deliveryIDs).Find(&bookedItems).Error; err != nil {
		return fmt.Errorf("failed to load delivery booking items of %v: %v", openSaleCodes, err)
	}

	lines := make([]deliveryLine, 0, len(bookedItems))
	for _, item := range bookedItems {
		lines = append(lines, deliveryLine{
			DeliveryCode: deliveryCodeOf[item.DeliveryID],
			DeliveryItem: item.DeliveryItem,
			SaleItemCode: item.DocumentRefItem,
		})
	}

	orderRes, err := orderExternalService.GetOrdersDelivery(orderExternalService.GetOrderDeliveryRequest{
		DeliveryCode: deliveryCodes,
	})
	if err != nil {
		return fmt.Errorf("failed to read WMS progress of %v: %v", deliveryCodes, err)
	}

	// foldWmsDelivered ไม่ใช่ foldWmsIssued — เส้นปิด SO ห้าม gate ด้วย outbound.Status
	// เพราะแถว outbound_item ยังไม่ commit ตอน hook ยิง (ดูคอมเมนต์ที่ foldWmsDelivered)
	issuedQtyByLine, issuedWeightByLine, closed := foldWmsDelivered(orderRes.Orders)
	issuedQty, issuedWeight := issuedBySaleItem(lines, issuedQtyByLine, issuedWeightByLine, closed)

	tolerance := toleranceSOPercent()

	toClose := selectDeliveredSaleItems(saleItems, issuedQty, issuedWeight, tolerance)
	if len(toClose) == 0 {
		fmt.Printf("CloseSalesFullyDelivered: nothing to close for sales %v (tolerance %v%%), candidates: %s\n",
			openSaleCodes, tolerance, describeCandidates(saleItems, issuedQty, issuedWeight))
		return nil
	}

	now := time.Now()
	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tx := gormx.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// เส้นอัตโนมัติต้องใช้ตัวที่มีการ์ดสถานะ (แตะแค่บรรทัดที่ยังเปิดอยู่)
	// ระหว่างอ่านกับเขียนอาจมีคนยกเลิกบรรทัดนั้น หรือใบจองอีกใบของ SO เดียวกันปิดไปแล้ว
	_, completedSaleCodes, err := saleService.MarkOpenSaleItemsCompleted(tx, toClose, user, nowDateOnly)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	fmt.Printf("CloseSalesFullyDelivered: closed %d sale items %v, completed sales %v\n",
		len(toClose), toClose, completedSaleCodes)

	return nil
}
