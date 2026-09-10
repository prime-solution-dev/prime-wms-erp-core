package priceService

import (
	"encoding/json"
	"testing"

	"prime-erp-core/internal/models"
)

// helper สร้าง mapping ของสูตรหนึ่งตัวสำหรับ test
func formulaEntry(name, uom, expression string, required []string) models.PriceListSubGroupFormulasMap {
	params, _ := json.Marshal(map[string]interface{}{
		"required":    required,
		"description": name,
	})
	return models.PriceListSubGroupFormulasMap{
		PriceListFormulas: models.PriceListFormulas{
			Name:        name,
			Uom:         uom,
			FormulaType: "price_calc",
			Expression:  expression,
			Params:      params,
		},
	}
}

func formulaNames(formulas []models.PriceListSubGroupFormulasMap) []string {
	out := make([]string, len(formulas))
	for i, f := range formulas {
		out[i] = f.PriceListFormulas.Name
	}
	return out
}

// สูตร pcs ที่อ้าง kg ต้องรันหลังสูตร uom kg ไม่ว่า input จะเรียงมาแบบใด
func TestSortFormulasByDependencyPutsProducerFirst(t *testing.T) {
	kgFormula := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price", "extra"})
	pcsFormula := formulaEntry("Pcs = [kg]  x [Avg. kg stock]", "pcs", "kg*avg_kg_stock", []string{"kg", "avg_kg_stock"})

	tests := []struct {
		name  string
		input []models.PriceListSubGroupFormulasMap
	}{
		{name: "input เรียง kg ก่อน", input: []models.PriceListSubGroupFormulasMap{kgFormula, pcsFormula}},
		{name: "input เรียง pcs ก่อน", input: []models.PriceListSubGroupFormulasMap{pcsFormula, kgFormula}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortFormulasByDependency(tt.input)
			gotNames := formulaNames(got)
			if len(gotNames) != 2 {
				t.Fatalf("ต้องได้ 2 สูตร แต่ได้ %d", len(gotNames))
			}
			if gotNames[0] != "kg = Base price + Extra" {
				t.Errorf("สูตรแรกคือ %q ต้องเป็น %q", gotNames[0], "kg = Base price + Extra")
			}
		})
	}
}

// คู่ kg = [Pcs] / [Avg. kg stock] + pcs = input
// สูตร kg อ้าง pcs จึงต้องรันหลังสูตรที่ผลิต pcs
func TestSortFormulasByDependencyInputFormulaFirst(t *testing.T) {
	inputFormula := models.PriceListSubGroupFormulasMap{
		PriceListFormulas: models.PriceListFormulas{
			Name:        "pcs = input",
			Uom:         "pcs",
			FormulaType: "input",
		},
	}
	kgFormula := formulaEntry("kg = [Pcs] / [Avg. kg stock]", "kg", "pcs/avg_kg_stock", []string{"pcs", "avg_kg_stock"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{kgFormula, inputFormula})
	gotNames := formulaNames(got)

	if gotNames[0] != "pcs = input" {
		t.Errorf("สูตรแรกคือ %q ต้องเป็น %q", gotNames[0], "pcs = input")
	}
}

// สูตรที่ไม่อ้างถึงกันต้องคงลำดับเดิม
func TestSortFormulasByDependencyKeepsOrderWhenIndependent(t *testing.T) {
	a := formulaEntry("Pcs = ( [Base price] + 2.1 ) x [Avg kg. stock] x (1+2%)", "pcs",
		"(base_price+2.1)*avg_kg_stock*1.02", []string{"base_price", "avg_kg_stock"})
	b := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price", "extra"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{a, b})
	gotNames := formulaNames(got)

	if gotNames[0] != a.PriceListFormulas.Name || gotNames[1] != b.PriceListFormulas.Name {
		t.Errorf("ลำดับ = %v ต้องคงเดิม", gotNames)
	}
}

// สูตรที่อ้างอิงวนกันต้องคงลำดับเดิมและไม่ panic
func TestSortFormulasByDependencyCircularKeepsOrder(t *testing.T) {
	pcsFormula := formulaEntry("Pcs from kg", "pcs", "kg*avg_kg_stock", []string{"kg"})
	kgFormula := formulaEntry("kg from Pcs", "kg", "pcs/avg_kg_stock", []string{"pcs"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{pcsFormula, kgFormula})
	gotNames := formulaNames(got)

	if len(gotNames) != 2 {
		t.Fatalf("ต้องได้ 2 สูตร แต่ได้ %d", len(gotNames))
	}
	if gotNames[0] != "Pcs from kg" || gotNames[1] != "kg from Pcs" {
		t.Errorf("ลำดับ = %v ต้องคงเดิมเมื่ออ้างอิงวนกัน", gotNames)
	}
}

// input ว่าง nil และสูตรเดียวต้องไม่ panic
func TestSortFormulasByDependencyEdgeCases(t *testing.T) {
	if got := sortFormulasByDependency(nil); len(got) != 0 {
		t.Errorf("input nil ต้องได้ slice ว่าง แต่ได้ %d", len(got))
	}
	if got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{}); len(got) != 0 {
		t.Errorf("input ว่างต้องได้ slice ว่าง แต่ได้ %d", len(got))
	}

	single := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price"})
	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{single})
	if len(got) != 1 || got[0].PriceListFormulas.Name != single.PriceListFormulas.Name {
		t.Errorf("สูตรเดียวต้องคืนตัวเดิม")
	}
}

// params ที่ parse ไม่ได้ต้องไม่ทำให้ panic และคงลำดับเดิม
func TestSortFormulasByDependencyInvalidParams(t *testing.T) {
	broken := models.PriceListSubGroupFormulasMap{
		PriceListFormulas: models.PriceListFormulas{
			Name:        "broken params",
			Uom:         "pcs",
			FormulaType: "price_calc",
			Expression:  "kg*avg_kg_stock",
			Params:      json.RawMessage(`{not json`),
		},
	}
	kgFormula := formulaEntry("kg = Base price + Extra", "kg", "base_price+extra", []string{"base_price"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{broken, kgFormula})
	if len(got) != 2 {
		t.Fatalf("ต้องได้ 2 สูตร แต่ได้ %d", len(got))
	}
}

// สูตรเดียวกันถูกผูกซ้ำ 2 ครั้ง (พบจริง 131 subgroup) ต้องไม่หายและไม่วนไม่จบ
func TestSortFormulasByDependencyDuplicateFormula(t *testing.T) {
	f := formulaEntry("Pcs = ( [Base price] + 2.1 ) x [Avg kg. stock] x (1+2%)", "pcs",
		"(base_price+2.1)*avg_kg_stock*1.02", []string{"base_price", "avg_kg_stock"})

	got := sortFormulasByDependency([]models.PriceListSubGroupFormulasMap{f, f})

	if len(got) != 2 {
		t.Fatalf("ต้องได้ 2 สูตร (ผูกซ้ำ) แต่ได้ %d", len(got))
	}
}
