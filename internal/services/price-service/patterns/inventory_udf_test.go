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

// The "Weight-spec" column must read WeightSpec, not TotalWeight (the total
// on-hand stock weight). Reading TotalWeight showed values like 1,500,000 kg
// in a column that is meant to hold a per-unit spec weight.
func TestGetWeightSpecFromInventory(t *testing.T) {
	sg := sgWithInventory(models.InventoryWeightResponse{
		WeightSpec:  12.5,
		TotalWeight: 1500000,
	})
	if got := getWeightSpecFromInventory(sg); got != 12.5 {
		t.Fatalf("want WeightSpec 12.5, got %v", got)
	}

	if got := getWeightSpecFromInventory(models.PriceListSubGroupResponse{}); got != 0 {
		t.Fatalf("want 0 with no inventory, got %v", got)
	}
}

// Avg kg stock must come back as the number 0 when there is no inventory,
// not as an empty string (which rendered as a blank cell).
func TestGetAvgProductFromInventory(t *testing.T) {
	if got := getAvgProductFromInventory(models.PriceListSubGroupResponse{}); got != 0 {
		t.Fatalf("want 0 with no inventory, got %v (%T)", got, got)
	}

	sg := sgWithInventory(models.InventoryWeightResponse{AvgWeight: 11111.114})
	if got := getAvgProductFromInventory(sg); got != 11111.11 {
		t.Fatalf("want 11111.11, got %v", got)
	}

	zero := sgWithInventory(models.InventoryWeightResponse{AvgWeight: 0})
	if got := getAvgProductFromInventory(zero); got != 0 {
		t.Fatalf("want 0 for zero AvgWeight, got %v", got)
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
