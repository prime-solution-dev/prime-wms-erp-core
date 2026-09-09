package deliveryService

import (
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
