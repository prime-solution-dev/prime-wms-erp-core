package patterns

import (
	"testing"

	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

func item11SubGroup(id, pg06Name, pg07Name string) models.PriceListSubGroupResponse {
	return models.PriceListSubGroupResponse{
		ID:                  id,
		SubgroupCode:        id,
		TotalNetPriceWeight: 100,
		SubGroupKeys: []models.PriceListSubGroupKeyResponse{
			{GroupCode: "PG02", ValueCode: "PG02_1", ValueName: "หมวด A", Seq: 2},
			{GroupCode: "PG05", ValueCode: "PG05_1", ValueName: "100", Seq: 5},
			{GroupCode: "PG03", ValueCode: "PG03_9", ValueName: "x200", Seq: 3},
			{GroupCode: "PG06", ValueCode: "PG06_1", ValueName: pg06Name, Seq: 6},
			{GroupCode: "PG07", ValueCode: "PG07_1", ValueName: pg07Name, Seq: 7},
		},
	}
}

func item11Config(t *testing.T) *PriceTableConfiguration {
	t.Helper()
	cfg, err := LoadConfiguration("GROUP_1_ITEM_11")
	if err != nil {
		t.Fatalf("โหลด config ไม่ได้: %v", err)
	}
	if len(cfg.Patterns) == 0 {
		t.Fatal("config ไม่มี pattern เลย")
	}
	return cfg
}

func item11Row(t *testing.T, pg06Name, pg07Name string) AGGridRowData {
	t.Helper()
	cfg := item11Config(t)
	rows := buildDirectRowsWithProductGroup2WithCode(
		cfg, &cfg.Patterns[0],
		[]models.PriceListSubGroupResponse{item11SubGroup("sg-1", pg06Name, pg07Name)},
		"PG02", "PG06", "PG07", "PG05", "PG03",
	)
	if len(rows) == 0 {
		t.Fatal("ไม่ได้แถวเลย")
	}
	return rows[0]
}

// "หนา x ยาว" ต้องใช้กติกาเดียวกับ compositeMappingValue คือข้ามค่าว่าง
// TrimSpace เดิมตัดได้แค่ช่องว่าง ตัว "x" ยังค้าง
func TestItem11ThicknessLength_SkipsEmptyValueName(t *testing.T) {
	cases := []struct {
		name   string
		pg06   string
		pg07   string
		want   string
		wantSz string
	}{
		{"มีครบทั้งคู่", "6", "12", "6 x 12", "100x200"},
		{"ยาวว่าง", "6", "", "6", "100x200"},
		{"หนาว่าง", "", "6", "6", "100x200"},
		{"ว่างทั้งคู่", "", "", "", "100x200"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := item11Row(t, tc.pg06, tc.pg07)
			if got := row["thickness_x_length"]; got != tc.want {
				t.Fatalf("thickness_x_length = %#v, want %q", got, tc.want)
			}
			// ขนาด (PG05+PG03) ต่อกันตรง ๆ ไม่มี separator — ต้องไม่เปลี่ยน
			if got := row["size"]; got != tc.wantSz {
				t.Fatalf("size = %#v, want %q", got, tc.wantSz)
			}
		})
	}
}

// comparator ของ sort ต้องเรียงตามค่าเดียวกับที่แสดง
// "6" (ยาวว่าง) ต้องมาก่อน "6 x 12" เหมือนที่ผู้ใช้เห็นบนจอ
func TestItem11SortOrder_MatchesDisplayedThicknessLength(t *testing.T) {
	resp, err := BuildGroup1Item11Response([]models.GetPriceListResponse{{
		ID: uuid.New().String(),
		SubGroups: []models.PriceListSubGroupResponse{
			item11SubGroup("sg-full", "6", "12"),
			item11SubGroup("sg-empty-length", "6", ""),
		},
	}}, "GROUP_1_ITEM_11")
	if err != nil {
		t.Fatalf("สร้าง response ไม่ได้: %v", err)
	}
	if len(resp.Tabs) == 0 || len(resp.Tabs[0].TableData) < 2 {
		t.Fatalf("ต้องได้อย่างน้อย 2 แถว แต่ได้ %#v", resp.Tabs)
	}

	got := []interface{}{
		resp.Tabs[0].TableData[0]["thickness_x_length"],
		resp.Tabs[0].TableData[1]["thickness_x_length"],
	}
	if got[0] != "6" || got[1] != "6 x 12" {
		t.Fatalf("ลำดับแถว = %#v, want [\"6\" \"6 x 12\"] — sort ต้องเรียงตามค่าที่แสดง", got)
	}
}
