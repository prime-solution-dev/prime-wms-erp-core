package deliveryService

import (
	"fmt"
	"strings"

	"prime-erp-core/internal/models"
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
