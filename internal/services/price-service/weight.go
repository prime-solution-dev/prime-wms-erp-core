package priceService

import (
	"context"
	"log"
	"math"
	goodsReceiveService "prime-erp-core/external/goods-receive-service"
	"prime-erp-core/internal/models"
)

const defaultWeightDigit = 2

// roundWeight ปัดน้ำหนัก (KG) แบบ half-up ตาม company_config.weight_digit
// สูตรเดียวกับ delivery-service บวก 1e-9 กัน 133.105*100 = 13310.499999
func roundWeight(v float64, digit int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	scale := math.Pow10(digit)
	return math.Copysign(math.Floor(math.Abs(v)*scale+0.5+1e-9), v) / scale
}

// weightDigitForDisplay คืน weight_digit ของบริษัทสำหรับค่าที่แสดงในกริด/export เท่านั้น
// (สูตรราคาอ่านน้ำหนักดิบแยกเอง) เรียก API ไม่ได้ใช้ 2 เหมือนพฤติกรรมเดิม ไม่ให้หน้า price list ล่มทั้งหน้า
func weightDigitForDisplay(ctx context.Context, companyCode string) int {
	if companyCode == "" {
		return defaultWeightDigit
	}
	res, err := goodsReceiveService.GetCompany(ctx, goodsReceiveService.CompanyFilter{CompanyCode: []string{companyCode}})
	if err != nil {
		log.Printf("Warning: failed to get weight digit of company %s, use %d: %v", companyCode, defaultWeightDigit, err)
		return defaultWeightDigit
	}
	for _, company := range res.Company {
		if company.CompanyCode == companyCode && company.CompanyConfig.WeightDigit != nil {
			return *company.CompanyConfig.WeightDigit
		}
	}
	return defaultWeightDigit
}

// roundInventoryWeight ปัดค่าน้ำหนักที่กริด/export แสดง (Avg. kg stock, น.น. ตลาด)
func roundInventoryWeight(inv models.InventoryWeightResponse, digit int) models.InventoryWeightResponse {
	inv.AvgProduct = roundWeight(inv.AvgProduct, digit)
	inv.AvgWeight = roundWeight(inv.AvgWeight, digit)
	inv.WeightSpec = roundWeight(inv.WeightSpec, digit)
	return inv
}
