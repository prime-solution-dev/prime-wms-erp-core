package patterns

import (
	"encoding/json"
	"strings"
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
			name: "มี batch_no ใน ColumnGroups.Children",
			pattern: PatternConfig{
				ColumnGroups: []ColumnGroupConfig{
					{Children: []ColumnConfigItem{{Field: "batch_no", HeaderName: "โรงงาน"}}},
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

// ยึดกับ config จริงในไฟล์ ไม่ใช่ PatternConfig ที่ประกอบขึ้นในเทสต์
// เพื่อให้จับได้ถ้ามีใครแก้ field batch_no ใน configs/*.json
// หรือมี pattern ใหม่ที่ประกาศ batch_no ผ่านช่องทางที่ patternHasBatchColumn มองไม่เห็น
func TestPatternHasBatchColumnAgainstRealConfigs(t *testing.T) {
	tests := []struct {
		groupCode string
		want      bool
	}{
		{"GROUP_1_ITEM_7", true},
		{"GROUP_1_ITEM_8", true},
		{"GROUP_1_ITEM_22", true},
		{"GROUP_1_ITEM_2", false},
		{"GROUP_1_ITEM_9", false},
		{"GROUP_1_ITEM_13", false},
	}

	for _, tt := range tests {
		t.Run(tt.groupCode, func(t *testing.T) {
			cfg, err := LoadConfiguration(tt.groupCode)
			if err != nil {
				t.Fatalf("โหลด config ไม่ได้: %v", err)
			}
			if len(cfg.Patterns) == 0 {
				t.Fatalf("config ไม่มี pattern เลย")
			}
			for i := range cfg.Patterns {
				got := patternHasBatchColumn(&cfg.Patterns[i])
				if got != tt.want {
					t.Errorf("pattern %q: ได้ %v ต้องเป็น %v", cfg.Patterns[i].ID, got, tt.want)
				}
			}
		})
	}
}

// คอลัมน์ Avg.kg stock ต้องผูกกับ field ของ avg ไม่ใช่ total_weight
//
// PG01_3_PATTERN.json เคย copy บล็อกคอลัมน์ Weight-spec มาทำ Avg.kg stock
// แล้วลืมเปลี่ยน field/dataMapping ทำให้กริดวาดเลข Weight-spec ซ้ำสองคอลัมน์
// ผู้ใช้เห็น Avg.kg stock เป็นเลขที่ดูสมเหตุสมผลทั้งที่ไม่มีสต็อก ส่วน Excel
// ซึ่งผูก avg_weight ถูกอยู่แล้วแสดง 0 ตามความจริง
//
// test นี้เดินทุกไฟล์ใน configs/*.json แบบ generic เพื่อกันเคสเดียวกันกลับมา
// ในไฟล์ไหนก็ตาม ไม่ใช่แค่ไฟล์ที่เคยพัง
func TestPatternConfigs_AvgColumnsBindToAvgField(t *testing.T) {
	entries, err := patternConfigs.ReadDir("configs")
	if err != nil {
		t.Fatalf("อ่าน configs ไม่ได้: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("ไม่พบไฟล์ config เลย — embed.FS อาจเปลี่ยน path")
	}

	// เดิน JSON แบบ generic แทนการ unmarshal เข้า struct
	// เพราะคอลัมน์กระจายอยู่หลาย key (columns, fixedColumns, columnGroups.children, patterns[])
	// การเดินดิบ ๆ จับได้ทุกที่โดยไม่ต้องไล่ตามโครงสร้างที่อาจเพิ่มในอนาคต
	var walk func(file string, node any, t *testing.T)
	walk = func(file string, node any, t *testing.T) {
		switch n := node.(type) {
		case map[string]any:
			header, _ := n["headerName"].(string)
			if strings.Contains(strings.ToLower(header), "avg") {
				field, _ := n["field"].(string)
				mapping, _ := n["dataMapping"].(string)
				// อย่างน้อยหนึ่งในสองต้องชี้ไปที่ avg ถึงจะดึงค่าจริงมาได้
				// (shared.go รับ dataMapping ทั้ง avg_weight และ avg_kg_stock
				// และ fallback ไปอ่าน field เมื่อ dataMapping ว่าง)
				if field != "" || mapping != "" {
					if !strings.Contains(strings.ToLower(field), "avg") &&
						!strings.Contains(strings.ToLower(mapping), "avg") {
						t.Errorf("%s: คอลัมน์ %q ผูกกับ field=%q dataMapping=%q ซึ่งไม่ใช่ค่า avg",
							file, header, field, mapping)
					}
				}
			}
			for _, v := range n {
				walk(file, v, t)
			}
		case []any:
			for _, v := range n {
				walk(file, v, t)
			}
		}
	}

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := patternConfigs.ReadFile("configs/" + entry.Name())
		if err != nil {
			t.Fatalf("%s: อ่านไม่ได้: %v", entry.Name(), err)
		}

		var root any
		if err := json.Unmarshal(data, &root); err != nil {
			t.Fatalf("%s: parse ไม่ผ่าน: %v", entry.Name(), err)
		}

		walk(entry.Name(), root, t)
		checked++
	}

	if checked == 0 {
		t.Fatal("ไม่ได้ตรวจไฟล์ config ใดเลย")
	}
	t.Logf("ตรวจ config %d ไฟล์", checked)
}
