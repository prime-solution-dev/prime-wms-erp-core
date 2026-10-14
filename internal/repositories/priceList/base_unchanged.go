package priceListRepository

import (
	"math"
	"prime-erp-core/internal/models"
)

// isBasePriceUnchanged บอกว่า payload ของ UpdatePriceListBase จะไม่ทำให้ group เปลี่ยน
// (ราคาเท่าเดิม และ PDC%/DUE% ของทุก term เท่าเดิม) — group แบบนี้ต้องไม่ถูกเขียน
// history / before_* / terms ใหม่ และไม่ต้อง cascade ไปคำนวณ subgroup
//
// เทียบที่ทศนิยม 2 ตำแหน่งให้ตรงกับ roundDecimal(..., 2) ฝั่งหน้าจอ
func isBasePriceUnchanged(old models.PriceListGroup, oldTerms []models.PriceListGroupTerm, group models.PriceListGroup) bool {
	if !equal2(old.PriceWeight, group.PriceWeight) || !equal2(old.PriceUnit, group.PriceUnit) {
		return false
	}
	if len(oldTerms) != len(group.PriceListGroupTerms) {
		return false
	}

	oldByCode := make(map[string]models.PriceListGroupTerm, len(oldTerms))
	for _, term := range oldTerms {
		oldByCode[term.TermCode] = term
	}
	for _, term := range group.PriceListGroupTerms {
		oldTerm, ok := oldByCode[term.TermCode]
		if !ok || !equal2(oldTerm.PdcPercent, term.PdcPercent) || !equal2(oldTerm.DuePercent, term.DuePercent) {
			return false
		}
	}
	return true
}

func equal2(a, b float64) bool {
	return math.Round(a*100) == math.Round(b*100)
}
