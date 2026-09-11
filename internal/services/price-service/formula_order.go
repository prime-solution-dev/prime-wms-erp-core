package priceService

import (
	"encoding/json"
	"fmt"

	"prime-erp-core/internal/models"
)

// sortFormulasByDependency เรียงสูตรให้สูตรที่ผลิตค่าอยู่ก่อนสูตรที่บริโภคค่านั้น
// ตัดสิน dependency จาก params.required ที่เก็บไว้ใน DB: ถ้าสูตร A ต้องใช้ตัวแปรที่
// ตรงกับ uom ของสูตร B แล้ว B ต้องมาก่อน A
//
// ORDER BY create_dtm DESC ใช้แทนไม่ได้ เพราะมีคู่ที่ kg อ้าง pcs ด้วย ลำดับจึงไม่ใช่
// ฟังก์ชันของ uom อย่างเดียว
// กรณีอ้างอิงวนกันคงลำดับเดิมและ log warning แทนคืน error เพราะทำให้ request ล้มจะแย่กว่า
// คืน slice ใหม่เสมอ ไม่แก้ไข formulas ที่รับเข้ามา
func sortFormulasByDependency(subGroupCode string, formulas []models.PriceListSubGroupFormulasMap) []models.PriceListSubGroupFormulasMap {
	if len(formulas) < 2 {
		return append([]models.PriceListSubGroupFormulasMap(nil), formulas...)
	}

	// อ่าน required ครั้งเดียว ไม่ parse JSON ซ้ำทุกรอบของ Kahn loop
	reqs := make([][]string, len(formulas))
	for i, f := range formulas {
		reqs[i] = formulaRequiredVars(f.PriceListFormulas)
	}

	done := make([]bool, len(formulas))

	// สูตร i ต้องรอ ถ้ามีสูตรอื่นที่ยังไม่เสร็จผลิตตัวแปรที่ i ต้องใช้
	waiting := func(i int) bool {
		for _, v := range reqs[i] {
			for j := range formulas {
				if j != i && !done[j] && formulas[j].PriceListFormulas.Uom == v {
					return true
				}
			}
		}
		return false
	}

	result := make([]models.PriceListSubGroupFormulasMap, 0, len(formulas))
	for len(result) < len(formulas) {
		progressed := false
		for i := range formulas {
			if done[i] || waiting(i) {
				continue
			}
			result = append(result, formulas[i])
			done[i] = true
			progressed = true
		}
		if !progressed {
			fmt.Printf("Warning: สูตรของ subgroup %s อ้างอิงวนกัน คงลำดับเดิมไว้\n", subGroupCode)
			for i := range formulas {
				if !done[i] {
					result = append(result, formulas[i])
					done[i] = true
				}
			}
		}
	}

	return result
}

// formulaRequiredVars อ่านรายชื่อตัวแปรที่สูตรต้องใช้จาก params.required
// คืน nil เมื่อ params ไม่มีหรือ parse ไม่ได้ เพื่อไม่ให้ข้อมูลเสียทำให้ request ล้ม
func formulaRequiredVars(formula models.PriceListFormulas) []string {
	if len(formula.Params) == 0 {
		return nil
	}
	var parsed struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(formula.Params, &parsed); err != nil {
		return nil
	}
	return parsed.Required
}
