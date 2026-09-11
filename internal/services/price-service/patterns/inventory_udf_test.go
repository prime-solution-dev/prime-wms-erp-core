package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

func sgWithInventory(inv models.InventoryWeightResponse) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		InventoryWeight: []models.InventoryWeightResponse{inv},
	}
}

// คอลัมน์ "Weight-spec" ต้องอ่านจาก sg.WeightSpec ซึ่งเป็นน้ำหนักของ base unit
// จาก product master ไม่ใช่จาก InventoryWeight ที่ผูกกับสต็อก
//
// เดิมอ่านจาก InventoryWeight[0].WeightSpec ซึ่ง warehouse-core ไม่เคย set ค่าเลย
// จึงได้ 0 เสมอ และค่าต้องมีแม้ subgroup นั้นไม่มีสต็อก (InventoryWeight ว่าง)
func TestGetWeightSpecFromInventory(t *testing.T) {
	sg := models.PriceListSubGroupResponse{WeightSpec: 12.5}
	if got := getWeightSpecFromInventory(sg); got != 12.5 {
		t.Fatalf("want WeightSpec 12.5, got %v", got)
	}

	// ไม่มีสต็อกเลยแต่ยังต้องได้ค่า weight spec
	noStock := models.PriceListSubGroupResponse{
		WeightSpec:      12.5,
		InventoryWeight: []models.InventoryWeightResponse{},
	}
	if got := getWeightSpecFromInventory(noStock); got != 12.5 {
		t.Fatalf("want 12.5 with no stock, got %v", got)
	}

	// มีสต็อกน้ำหนักรวมมหาศาล แต่ต้องไม่หลุดมาเป็น weight spec
	withStock := sgWithInventory(models.InventoryWeightResponse{TotalWeight: 1500000})
	withStock.WeightSpec = 12.5
	if got := getWeightSpecFromInventory(withStock); got != 12.5 {
		t.Fatalf("want 12.5, must not read TotalWeight, got %v", got)
	}

	// ไม่มีข้อมูลอะไรเลย → 0 (ฝั่งสูตรจะ fallback เป็น 1.0 เอง)
	if got := getWeightSpecFromInventory(models.PriceListSubGroupResponse{}); got != 0 {
		t.Fatalf("want 0 with no data, got %v", got)
	}
}

// Avg kg stock must come back as the number 0 when there is no inventory,
// not as an empty string (which rendered as a blank cell).
func TestGetAvgKgStockFromInventoryZeroValues(t *testing.T) {
	if got := getAvgKgStockFromInventory(models.PriceListSubGroupResponse{}, false); got != 0 {
		t.Fatalf("want 0 with no inventory, got %v (%T)", got, got)
	}

	sg := sgWithInventory(models.InventoryWeightResponse{AvgWeight: 22222.224, AvgProduct: 11111.114})
	if got := getAvgKgStockFromInventory(sg, false); got != 11111.11 {
		t.Fatalf("want 11111.11, got %v", got)
	}

	zero := sgWithInventory(models.InventoryWeightResponse{AvgWeight: 0, AvgProduct: 0})
	if got := getAvgKgStockFromInventory(zero, false); got != 0 {
		t.Fatalf("want 0 for zero AvgProduct, got %v", got)
	}
}

func TestGetQtyFromInventory(t *testing.T) {
	if got := getQtyFromInventory(models.PriceListSubGroupResponse{}); got != 0 {
		t.Fatalf("want 0 with no inventory, got %v", got)
	}

	sumQty := sgWithInventory(models.InventoryWeightResponse{SumQty: 7, TotalQty: 99})
	if got := getQtyFromInventory(sumQty); got != 7 {
		t.Fatalf("want SumQty 7, got %v", got)
	}

	totalOnly := sgWithInventory(models.InventoryWeightResponse{TotalQty: 4})
	if got := getQtyFromInventory(totalOnly); got != 4 {
		t.Fatalf("want TotalQty fallback 4, got %v", got)
	}
}

// udf_json holds the same key as a JSON number or a JSON string depending on
// which AG Grid cell editor saved it. Asserting only on float64/int silently
// dropped every string-saved value, which users saw as "Save doesn't work".
func TestUdfNumeric(t *testing.T) {
	cases := []struct {
		name string
		udf  map[string]interface{}
		want interface{}
	}{
		{"json number", map[string]interface{}{"line_bundle": float64(444)}, float64(444)},
		{"go int", map[string]interface{}{"line_bundle": 12}, float64(12)},
		{"numeric string", map[string]interface{}{"line_bundle": "444"}, float64(444)},
		{"decimal string", map[string]interface{}{"line_bundle": "12.5"}, 12.5},
		{"padded string", map[string]interface{}{"line_bundle": " 33 "}, float64(33)},
		{"non-numeric string passes through", map[string]interface{}{"line_bundle": "1Test"}, "1Test"},
		{"empty string", map[string]interface{}{"line_bundle": ""}, nil},
		{"blank string", map[string]interface{}{"line_bundle": "   "}, nil},
		{"explicit null", map[string]interface{}{"line_bundle": nil}, nil},
		{"absent key", map[string]interface{}{}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := udfNumeric(tc.udf, "line_bundle")
			if got != tc.want {
				t.Fatalf("want %v (%T), got %v (%T)", tc.want, tc.want, got, got)
			}
		})
	}
}
