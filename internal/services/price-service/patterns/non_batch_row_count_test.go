package patterns

import (
	"strings"
	"testing"

	"prime-erp-core/internal/models"
)

// เทสต์ระดับ build path ของบั๊ก "แถวซ้ำ" — ห้ามอ้าง symbol ภายในของตัวแก้
// เรียกผ่าน buildDynamicRows / buildDirectRows / buildDirectRowsWithProductGroup2WithCode
// เท่านั้น เพื่อให้ cherry-pick ไปวางบน commit ก่อนแก้แล้ว fail ด้วย assertion ได้
//
// fixture จำลองสิ่งที่ get-price-detail.go ทำ คือ copy subgroup เดิม (ID เดียวกัน)
// 1 ชุดต่อ 1 inventory record ที่ warehouse-core คืนมา

// nbCopies สร้าง subgroup ID เดียวกัน n ชุด ต่างกันที่ค่าที่มาจาก inventory record
// avgProduct ตั้งให้ต่างกันเพื่อใช้เป็น probe ว่าโค้ดอ่าน record ตัวไหน
// (ข้อมูลจริงค่านี้เท่ากันทุกชุดเพราะเป็นค่าระดับ site — ที่นี่ทำให้ต่างเพื่อจับลำดับ)
func nbCopies(id string, n int, keys []models.PriceListSubGroupKeyResponse) []models.PriceListSubGroupResponse {
	out := make([]models.PriceListSubGroupResponse, 0, n)
	for i := 1; i <= n; i++ {
		v := float64(i * 10)
		out = append(out, models.PriceListSubGroupResponse{
			ID:           id,
			SubgroupCode: id,
			BatchNo:      strings.Repeat("B", i),
			SubGroupKeys: keys,
			InventoryWeight: []models.InventoryWeightResponse{
				{AvgWeight: v, AvgProduct: v, TotalWeight: v},
			},
		})
	}
	return out
}

func nbLoadPattern(t *testing.T, groupCode string) (*PriceTableConfiguration, *PatternConfig) {
	t.Helper()
	cfg, err := LoadConfiguration(groupCode)
	if err != nil {
		t.Fatalf("โหลด config %s ไม่ได้: %v", groupCode, err)
	}
	if len(cfg.Patterns) == 0 {
		t.Fatalf("config %s ไม่มี pattern", groupCode)
	}
	return cfg, &cfg.Patterns[0]
}

// nbMaxRowNumber คืนค่าสูงสุดของคอลัมน์ *_row_number ในทุกแถว
// buildDynamicRows นับ 1 ครั้งต่อ 1 subgroup ที่วนเจอ จึงบอกได้ว่ามีกี่ชุดถูกประมวลผล
func nbMaxRowNumber(rows []AGGridRowData) int {
	max := 0
	for _, row := range rows {
		for k, v := range row {
			if !strings.HasSuffix(k, "_row_number") {
				continue
			}
			if n, ok := v.(int); ok && n > max {
				max = n
			}
		}
	}
	return max
}

// nbHasValue บอกว่ามีช่องไหนในแถวถือค่า want อยู่หรือไม่
func nbHasValue(row AGGridRowData, want float64) bool {
	for _, v := range row {
		if f, ok := v.(float64); ok && f == want {
			return true
		}
	}
	return false
}

// buildDirectRows สร้าง 1 แถวต่อ 1 subgroup ตรง ๆ
// pattern ที่ไม่มีคอลัมน์ batch_no จึงได้แถวหน้าตาเหมือนกันเป๊ะตามจำนวน inventory record
func TestBuildDirectRows_DuplicateSubGroupRowCount(t *testing.T) {
	tests := []struct {
		name      string
		groupCode string
		wantBatch bool
		wantRows  int
	}{
		{name: "pattern ไม่มี batch_no ต้องเหลือแถวเดียว", groupCode: "GROUP_1_ITEM_6", wantBatch: false, wantRows: 1},
		{name: "pattern มี batch_no ต้องได้ครบทุก batch", groupCode: "GROUP_1_ITEM_8", wantBatch: true, wantRows: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, pattern := nbLoadPattern(t, tt.groupCode)
			if got := patternHasBatchColumn(pattern); got != tt.wantBatch {
				t.Fatalf("fixture ผิด: %s patternHasBatchColumn = %v", tt.groupCode, got)
			}

			rows := buildDirectRows(cfg, pattern, nbCopies("sg-1", 3, nil))
			if len(rows) != tt.wantRows {
				t.Fatalf("ต้องได้ %d แถว แต่ได้ %d", tt.wantRows, len(rows))
			}
		})
	}
}

// buildDynamicRows ยุบแถวด้วย rowMap[columnKey|rowKey|sg.ID] อยู่แล้ว จำนวนแถวจึงไม่ต่าง
// สิ่งที่ต่างคือจำนวนรอบที่ subgroup ถูกประมวลผล (เห็นได้จาก *_row_number)
// และค่าที่มาจาก inventory ซึ่งเขียนทับกันแบบ last-write-wins
func TestBuildDynamicRows_DuplicateSubGroupProcessedOnce(t *testing.T) {
	item9Keys := []models.PriceListSubGroupKeyResponse{
		{GroupCode: "PG02", ValueCode: "PG02_10", ValueName: "หมวดเหล็กเส้น", Seq: 2},
		{GroupCode: "PG07", ValueCode: "PG07_8", ValueName: "ขนาด 12 มม.", Seq: 7},
		{GroupCode: "PG06", ValueCode: "PG06_1", ValueName: "ความยาว 6 เมตร", Seq: 6},
		{GroupCode: "PG03", ValueCode: "PG03_1", ValueName: "เกรด SD40", Seq: 3},
	}
	item7Keys := []models.PriceListSubGroupKeyResponse{
		{GroupCode: "PG02", ValueCode: "PG02_10", ValueName: "หมวดเหล็กเส้น", Seq: 2},
		{GroupCode: "PG06", ValueCode: "PG06_1", ValueName: "ความยาว 6 เมตร", Seq: 6},
		{GroupCode: "PG05", ValueCode: "PG05_1", ValueName: "ขนาด 12 มม.", Seq: 5},
	}

	tests := []struct {
		name          string
		groupCode     string
		keys          []models.PriceListSubGroupKeyResponse
		wantBatch     bool
		wantRowNumber int
	}{
		{name: "pattern ไม่มี batch_no ต้องประมวลผลชุดเดียว", groupCode: "GROUP_1_ITEM_9", keys: item9Keys, wantBatch: false, wantRowNumber: 1},
		{name: "pattern มี batch_no ต้องประมวลผลครบทุก batch", groupCode: "GROUP_1_ITEM_7", keys: item7Keys, wantBatch: true, wantRowNumber: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, pattern := nbLoadPattern(t, tt.groupCode)
			if got := patternHasBatchColumn(pattern); got != tt.wantBatch {
				t.Fatalf("fixture ผิด: %s patternHasBatchColumn = %v", tt.groupCode, got)
			}

			rows := buildDynamicRows(cfg, pattern, nbCopies("sg-1", 3, tt.keys))
			if len(rows) != 1 {
				t.Fatalf("ต้องได้ 1 แถว (rowMap ยุบด้วย sg.ID อยู่แล้ว) แต่ได้ %d", len(rows))
			}
			if got := nbMaxRowNumber(rows); got != tt.wantRowNumber {
				t.Fatalf("row_number สูงสุดต้องเป็น %d แต่ได้ %d", tt.wantRowNumber, got)
			}
		})
	}
}

// buildDirectRowsWithProductGroup2WithCode (GROUP_1_ITEM_11) ยุบแถวด้วย rowKey ของตัวเอง
// จำนวนแถวจึงไม่ต่าง แต่ค่าที่มาจาก inventory ถูกเขียนทับตามลำดับ
// pattern ที่ไม่มี batch_no ต้องเห็นค่าของ record แรก (ตรงกับ inventoryWeights[0] ฝั่ง export)
// ส่วน pattern ที่มี batch_no ต้องยังเห็นค่าของ record สุดท้าย คือไม่มีการยุบเกิดขึ้น
func TestBuildDirectRowsWithProductGroup2WithCode_UsesFirstInventoryRecord(t *testing.T) {
	cfg, item11Pattern := nbLoadPattern(t, "GROUP_1_ITEM_11")
	_, item7Pattern := nbLoadPattern(t, "GROUP_1_ITEM_7")

	pg2 := getGroupCodeFromConfig(cfg, item11Pattern, "productGroup2", "PRODUCT_GROUP2")
	pg6 := getGroupCodeFromConfig(cfg, item11Pattern, "productGroup6", "PRODUCT_GROUP6")
	pg7 := getGroupCodeFromConfig(cfg, item11Pattern, "productGroup7", "PRODUCT_GROUP7")
	pg5 := getGroupCodeFromConfig(cfg, item11Pattern, "productGroup5", "PRODUCT_GROUP5")
	pg3 := getGroupCodeFromConfig(cfg, item11Pattern, "productGroup3", "PRODUCT_GROUP3")

	keys := []models.PriceListSubGroupKeyResponse{
		{GroupCode: pg2, ValueCode: "PG02_1", ValueName: "หมวดเหล็กแบน"},
		{GroupCode: pg6, ValueCode: "PG06_1", ValueName: "หนา 3 มม."},
		{GroupCode: pg7, ValueCode: "PG07_1", ValueName: "ยาว 6 เมตร"},
		{GroupCode: pg5, ValueCode: "PG05_1", ValueName: "ขนาด 1"},
		{GroupCode: pg3, ValueCode: "PG03_1", ValueName: "เกรด A"},
	}

	tests := []struct {
		name      string
		pattern   *PatternConfig
		wantBatch bool
		wantValue float64
	}{
		{name: "pattern ไม่มี batch_no ต้องใช้ record แรก", pattern: item11Pattern, wantBatch: false, wantValue: 10},
		{name: "pattern มี batch_no ต้องไม่ถูกยุบ", pattern: item7Pattern, wantBatch: true, wantValue: 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := patternHasBatchColumn(tt.pattern); got != tt.wantBatch {
				t.Fatalf("fixture ผิด: patternHasBatchColumn = %v", got)
			}

			rows := buildDirectRowsWithProductGroup2WithCode(cfg, tt.pattern, nbCopies("sg-1", 3, keys), pg2, pg6, pg7, pg5, pg3)
			if len(rows) != 1 {
				t.Fatalf("ต้องได้ 1 แถว แต่ได้ %d", len(rows))
			}
			if !nbHasValue(rows[0], tt.wantValue) {
				t.Fatalf("แถวต้องถือค่าจาก inventory record ที่ถูกต้อง (%v) แต่ไม่พบ: %v", tt.wantValue, rows[0])
			}
		})
	}
}
