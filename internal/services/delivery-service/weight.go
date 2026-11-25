package deliveryService

import (
	"context"
	"fmt"
	"math"
	goodsReceiveService "prime-erp-core/external/goods-receive-service"
)

const defaultWeightDigit = 2

// roundWeight ปัดน้ำหนัก (KG) เป็น digit ตำแหน่งแบบ half-up ให้ตรงกับที่หน้าบ้านบันทึก
// ลงเอกสาร (quotation -> sale order -> DBS -> CO) จำนวนตำแหน่งเป็นของแต่ละบริษัท
// (company_config.weight_digit) ทุกเอกสารเก็บตามนั้นแล้วคัดลอกต่อ ไม่ให้แต่ละชั้นปัดกันเอง
//
// บวก 1e-9 ก่อนปัด กัน 133.105*100 = 13310.499999 จาก floating point แล้วปัดลงผิด
func roundWeight(v float64, digit int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	scale := math.Pow10(digit)
	return math.Copysign(math.Floor(math.Abs(v)*scale+0.5+1e-9), v) / scale
}

// getWeightDigit คืน weight_digit ของบริษัท ไม่มีค่าหรือไม่เจอบริษัทใช้ 2
// เรียก API ไม่สำเร็จต้องคืน error ห้ามเดา ไม่งั้นน้ำหนักถูกบันทึกผิดตำแหน่ง
func getWeightDigit(ctx context.Context, companyCode string) (int, error) {
	if companyCode == "" {
		return defaultWeightDigit, nil
	}

	res, err := goodsReceiveService.GetCompany(ctx, goodsReceiveService.CompanyFilter{CompanyCode: []string{companyCode}})
	if err != nil {
		return 0, fmt.Errorf("failed to get weight digit of company %s: %w", companyCode, err)
	}

	for _, company := range res.Company {
		if company.CompanyCode == companyCode && company.CompanyConfig.WeightDigit != nil {
			return *company.CompanyConfig.WeightDigit, nil
		}
	}
	return defaultWeightDigit, nil
}

// getWeightDigits เรียก getWeightDigit บริษัทละครั้งต่อ request
func getWeightDigits(ctx context.Context, companyCodes []string) (map[string]int, error) {
	digits := map[string]int{}
	for _, companyCode := range companyCodes {
		if _, done := digits[companyCode]; done {
			continue
		}
		digit, err := getWeightDigit(ctx, companyCode)
		if err != nil {
			return nil, err
		}
		digits[companyCode] = digit
	}
	return digits, nil
}
