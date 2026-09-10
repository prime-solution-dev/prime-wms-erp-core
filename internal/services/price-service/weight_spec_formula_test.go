package priceService

import (
	"testing"

	priceDomain "prime-erp-core/internal/services/price-service/domain"
)

// สูตรคู่ Pcs = [kg] x [Weight Spec] และ kg = [Pcs] / [Weight Spec] ต้องเป็นผกผันกัน
//
// weight_spec มีหน่วย kg ต่อชิ้น ดังนั้น
//   ราคาต่อชิ้น = ราคาต่อกิโล x kg ต่อชิ้น
//   ราคาต่อกิโล = ราคาต่อชิ้น / kg ต่อชิ้น
//
// expression ของสูตรที่สองเคยเป็น pcs*weight_spec ซึ่งเป็นการคูณ ไม่ตรงกับชื่อสูตร
// แก้เป็น pcs/weight_spec ที่ migration 2026-09-11-fix-pcs-weight-spec-expression.sql
func TestWeightSpecFormulasAreInverse(t *testing.T) {
	const weightSpec = 3.0
	const kgPrice = 19.60

	pcsFromKg, err := CalculatePrice(priceDomain.PriceFormula{
		Expression: "kg*weight_spec",
		Rounding:   2,
	}, priceDomain.PriceData{Kg: kgPrice, WeightSpec: weightSpec})
	if err != nil {
		t.Fatalf("คำนวณราคาต่อชิ้นล้มเหลว: %v", err)
	}
	if pcsFromKg != 58.80 {
		t.Fatalf("ราคาต่อชิ้น = %v ต้องเป็น 58.80 (19.60 x 3)", pcsFromKg)
	}

	kgFromPcs, err := CalculatePrice(priceDomain.PriceFormula{
		Expression: "pcs/weight_spec",
		Rounding:   2,
	}, priceDomain.PriceData{Pcs: pcsFromKg, WeightSpec: weightSpec})
	if err != nil {
		t.Fatalf("คำนวณราคาต่อกิโลล้มเหลว: %v", err)
	}
	if kgFromPcs != kgPrice {
		t.Errorf("ราคาต่อกิโล = %v ต้องกลับมาเป็น %v", kgFromPcs, kgPrice)
	}
}

// expression เดิมที่ผิดต้องให้ผลที่ไม่ใช่ผกผัน — ตรึงไว้เพื่อกันการถอยกลับ
func TestWeightSpecFormulaWrongExpressionIsNotInverse(t *testing.T) {
	wrong, err := CalculatePrice(priceDomain.PriceFormula{
		Expression: "pcs*weight_spec",
		Rounding:   2,
	}, priceDomain.PriceData{Pcs: 58.80, WeightSpec: 3.0})
	if err != nil {
		t.Fatalf("คำนวณล้มเหลว: %v", err)
	}
	if wrong == 19.60 {
		t.Error("expression pcs*weight_spec ไม่ควรให้ผลเท่ากับผกผันที่ถูกต้อง")
	}
	if wrong != 176.40 {
		t.Errorf("ได้ %v ต้องเป็น 176.40 (58.80 x 3) ซึ่งเป็นค่าที่ผิด", wrong)
	}
}

// สูตรทั้ง 4 ตัวที่ใช้ avg_kg_stock ต้องคำนวณตรงกับการคิดมือ
func TestCalculatePriceAvgKgStockFormulas(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		rounding   int
		data       priceDomain.PriceData
		want       float64
	}{
		{
			name:       "Pcs = [kg] x [Avg. kg stock]",
			expression: "kg*avg_kg_stock",
			rounding:   2,
			data:       priceDomain.PriceData{Kg: 19.60, AvgKgStock: 3.0},
			want:       58.80,
		},
		{
			name:       "kg = [Pcs] / [Avg. kg stock]",
			expression: "pcs/avg_kg_stock",
			rounding:   2,
			data:       priceDomain.PriceData{Pcs: 58.80, AvgKgStock: 3.0},
			want:       19.60,
		},
		{
			name:       "Pcs = ( [Base price] + 1.4 ) x [Avg kg. stock] x (1+2%)",
			expression: "(base_price+1.4)*avg_kg_stock*1.02",
			rounding:   0,
			data:       priceDomain.PriceData{BasePrice: 18.30, AvgKgStock: 3.0},
			want:       60,
		},
		{
			name:       "Pcs = ( [Base price] + 2.1 ) x [Avg kg. stock] x (1+2%)",
			expression: "(base_price+2.1)*avg_kg_stock*1.02",
			rounding:   0,
			data:       priceDomain.PriceData{BasePrice: 18.30, AvgKgStock: 3.0},
			want:       62,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculatePrice(priceDomain.PriceFormula{
				Expression: tt.expression,
				Rounding:   tt.rounding,
			}, tt.data)
			if err != nil {
				t.Fatalf("คำนวณล้มเหลว: %v", err)
			}
			if got != tt.want {
				t.Errorf("ได้ %v ต้องเป็น %v", got, tt.want)
			}
		})
	}
}
