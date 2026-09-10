package patterns

import (
	"testing"

	"prime-erp-core/internal/models"
)

// Avg. kg stock ที่ pattern ทั่วไปต้องแสดงคือค่าระดับ site (AvgProduct)
// ส่วน pattern ที่แสดงคอลัมน์ batch_no ต้องแสดงค่าระดับ batch (AvgWeight)
func TestGetAvgKgStockFromInventory(t *testing.T) {
	sg := models.PriceListSubGroupResponse{
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: 32.0, AvgProduct: 2.9969},
		},
	}

	tests := []struct {
		name     string
		perBatch bool
		want     float64
	}{
		{name: "pattern ทั่วไปใช้ค่าระดับ site", perBatch: false, want: 3.0},
		{name: "pattern ที่แสดง batch ใช้ค่าระดับ batch", perBatch: true, want: 32.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAvgKgStockFromInventory(sg, tt.perBatch)
			if got != tt.want {
				t.Errorf("ได้ %v ต้องเป็น %v", got, tt.want)
			}
		})
	}
}

// ไม่มีสต็อกต้องคืน 0 เพื่อให้กริดแสดงเลข 0 ไม่ใช่ช่องว่าง
func TestGetAvgKgStockFromInventoryNoStock(t *testing.T) {
	empty := models.PriceListSubGroupResponse{}

	if got := getAvgKgStockFromInventory(empty, false); got != 0 {
		t.Errorf("ไม่มีสต็อก (site): ได้ %v ต้องเป็น 0", got)
	}
	if got := getAvgKgStockFromInventory(empty, true); got != 0 {
		t.Errorf("ไม่มีสต็อก (batch): ได้ %v ต้องเป็น 0", got)
	}
}

// ค่าที่แสดงต้องถูกปัดเป็น 2 ตำแหน่งเหมือนพฤติกรรมเดิม
func TestGetAvgKgStockFromInventoryRoundsToTwoDecimals(t *testing.T) {
	sg := models.PriceListSubGroupResponse{
		InventoryWeight: []models.InventoryWeightResponse{
			{AvgWeight: 9.16666666, AvgProduct: 2.99690876},
		},
	}

	if got := getAvgKgStockFromInventory(sg, false); got != 3.0 {
		t.Errorf("site: ได้ %v ต้องเป็น 3", got)
	}
	if got := getAvgKgStockFromInventory(sg, true); got != 9.17 {
		t.Errorf("batch: ได้ %v ต้องเป็น 9.17", got)
	}
}

// pattern ที่มีคอลัมน์ batch_no ต้องถูกจัดเป็น per-batch
// GROUP_1_ITEM_7 และ GROUP_1_ITEM_22 ใช้ headerName "โรงงาน"
// GROUP_1_ITEM_8 ใช้ "Ship No." แต่ field ยังเป็น batch_no
func TestPatternHasBatchColumn(t *testing.T) {
	tests := []struct {
		name    string
		pattern PatternConfig
		want    bool
	}{
		{
			name: "มี batch_no ใน Columns",
			pattern: PatternConfig{
				Columns: []ColumnConfigItem{
					{Field: "batch_no", HeaderName: "โรงงาน"},
					{Field: "avg_weight_ton", HeaderName: "Avg.kg stock (Tons)"},
				},
			},
			want: true,
		},
		{
			name: "มี batch_no ใน FixedColumns",
			pattern: PatternConfig{
				FixedColumns: []ColumnConfigItem{
					{Field: "batch_no", HeaderName: "Ship No."},
				},
			},
			want: true,
		},
		{
			name: "มี batch_no เป็น dataMapping เท่านั้น",
			pattern: PatternConfig{
				Columns: []ColumnConfigItem{
					{Field: "factory", HeaderName: "โรงงาน", DataMapping: "batch_no"},
				},
			},
			want: true,
		},
		{
			name: "ไม่มี batch_no เลย",
			pattern: PatternConfig{
				Columns: []ColumnConfigItem{
					{Field: "avg_weight", HeaderName: "Avg.kg stock"},
					{Field: "total_weight", HeaderName: "Weight-spec"},
				},
			},
			want: false,
		},
		{
			name:    "pattern ว่าง",
			pattern: PatternConfig{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patternHasBatchColumn(&tt.pattern)
			if got != tt.want {
				t.Errorf("ได้ %v ต้องเป็น %v", got, tt.want)
			}
		})
	}
}
