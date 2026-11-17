package priceService

import (
	"testing"

	"prime-erp-core/internal/models"
)

func TestRoundWeight(t *testing.T) {
	tests := []struct {
		v     float64
		digit int
		want  float64
	}{
		{2.99690876, 3, 2.997},
		{9.16666666, 2, 9.17},
		{133.105, 2, 133.11},
		{1.0005, 3, 1.001},
		{-1.2345, 3, -1.235},
		{0, 3, 0},
	}
	for _, tt := range tests {
		if got := roundWeight(tt.v, tt.digit); got != tt.want {
			t.Errorf("roundWeight(%v, %d) = %v ต้องเป็น %v", tt.v, tt.digit, got, tt.want)
		}
	}
}

func TestRoundInventoryWeight(t *testing.T) {
	got := roundInventoryWeight(models.InventoryWeightResponse{
		AvgProduct: 2.99690876, AvgWeight: 9.16666666, WeightSpec: 12.34567, TotalQty: 7,
	}, 3)
	if got.AvgProduct != 2.997 || got.AvgWeight != 9.167 || got.WeightSpec != 12.346 {
		t.Errorf("ปัดไม่ตรง: %+v", got)
	}
	if got.TotalQty != 7 {
		t.Errorf("ห้ามแตะฟิลด์อื่น: TotalQty = %v", got.TotalQty)
	}
}
