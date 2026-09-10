package priceService

import (
	"encoding/json"
	"fmt"

	"prime-erp-core/internal/models"
)

// sortFormulasByDependency เรียงสูตรให้สูตรที่ผลิตค่าอยู่ก่อนสูตรที่บริโภคค่านั้น
//
// สูตรใน price_list_formulas อ้างถึงผลลัพธ์ของกันเอง เช่น
// Pcs = [kg] x [Avg. kg stock] มี expression kg*avg_kg_stock ซึ่ง kg คือผลลัพธ์
// ของสูตร uom = "kg" ในคู่เดียวกัน ไม่ใช่ราคาตั้งของกลุ่ม
//
// repository ดึงสูตรมาด้วย ORDER BY create_dtm DESC ซึ่งเป็นลำดับที่ผูกสูตร
// ไม่ใช่ลำดับ dependency ทำให้บาง subgroup ประเมินสูตร pcs ก่อนที่ kg จะถูก
// คำนวณใหม่ แล้วอ่านค่าของรอบคำนวณก่อนหน้า
//
// ตัดสิน dependency จาก params.required ที่เก็บไว้ใน DB อยู่แล้ว
// ถ้าสูตร A มี required ที่ตรงกับ uom ของสูตร B แล้ว B ต้องมาก่อน A
//
// กรณีอ้างอิงวนกันให้คงลำดับเดิมและไม่คืน error เพราะตอนนี้ไม่มีข้อมูลเช่นนั้น
// และการทำให้ request ล้มจะแย่กว่าการคำนวณด้วยลำดับเดิม
func sortFormulasByDependency(formulas []models.PriceListSubGroupFormulasMap) []models.PriceListSubGroupFormulasMap {
	if len(formulas) < 2 {
		return formulas
	}

	// uom ของสูตรแต่ละตัวคือชื่อตัวแปรที่สูตรนั้นผลิต ("pcs" หรือ "kg")
	producedBy := make(map[string][]int)
	for i, f := range formulas {
		uom := f.PriceListFormulas.Uom
		if uom != "" {
			producedBy[uom] = append(producedBy[uom], i)
		}
	}

	// needs[i] = เซ็ตของ index ที่สูตร i ต้องรอ
	needs := make([]map[int]bool, len(formulas))
	for i, f := range formulas {
		needs[i] = make(map[int]bool)
		for _, varName := range formulaRequiredVars(f.PriceListFormulas) {
			for _, producer := range producedBy[varName] {
				if producer != i {
					needs[i][producer] = true
				}
			}
		}
	}

	// topological sort แบบคงลำดับเดิมไว้มากที่สุด
	done := make([]bool, len(formulas))
	result := make([]models.PriceListSubGroupFormulasMap, 0, len(formulas))

	for len(result) < len(formulas) {
		progressed := false
		for i := range formulas {
			if done[i] {
				continue
			}
			ready := true
			for dep := range needs[i] {
				if !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				result = append(result, formulas[i])
				done[i] = true
				progressed = true
			}
		}
		if !progressed {
			// อ้างอิงวนกัน เติมตัวที่เหลือตามลำดับเดิมแล้วจบ
			fmt.Printf("Warning: พบการอ้างอิงวนกันระหว่างสูตร คงลำดับเดิมไว้\n")
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
