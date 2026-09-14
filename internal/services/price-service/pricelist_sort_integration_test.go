//go:build integration

package priceService

import (
	"testing"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
)

// ยืนยันว่าค่าที่ group_item เก็บเป็น varchar พร้อม thousand separator ถูกอ่านผ่าน
// getGroupAndItemMappings() แล้วแปลงเป็นตัวเลขได้ถูกต้อง — เป็นต้นทางของลำดับ
// ทุกแกนในหน้า Price List Detail
//
// ค่าที่ seed ดึงจาก UAT จริง (thaimetal-wms-uat, site TMI_WH)
//
// ไม่ parallel-safe — เขียนลงตาราง group/group_item ที่ทั้งแพ็กเกจใช้ร่วมกัน
// ห้ามใส่ t.Parallel()
func TestIntegration_GroupItemValueFeedsSortOrder(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)
	seedGroupItemsForSortTest(t)

	_, groupItemMap, _, err := getGroupAndItemMappings()
	if err != nil {
		t.Fatalf("getGroupAndItemMappings: %v", err)
	}

	cases := []struct {
		code    string
		wantVal float64
		wantHas bool
	}{
		{"PG05_22", 10000, true}, // 1250x8'  value = "10,000.00"
		{"PG05_3", 32, true},     // 4' x 8'  value = "32.00"
		{"PG06_4", 1.2, true},    // 1.2      value = "1.20"
		{"PG06_78", 100, true},   // 100      value = "100.00"
		{"PG03_10", 10, true},    // SS400    value = "10.00"
		{"PG03_21", 21, true},    // LT       value = "21.00"
		{"PG05_40", 0, false},    // ESP      value = ""
	}

	for _, c := range cases {
		gotVal, gotHas := parseGroupItemValue(groupItemMap, c.code)
		if gotVal != c.wantVal || gotHas != c.wantHas {
			t.Fatalf("%s ได้ (%v, %v) ต้องเป็น (%v, %v)", c.code, gotVal, gotHas, c.wantVal, c.wantHas)
		}
	}

	// ลำดับที่ได้ต้องเป็นตัวเลข ไม่ใช่ string — 100 ต้องอยู่หลัง 12 ไม่ใช่หน้า
	twelve, _ := parseGroupItemValue(groupItemMap, "PG06_62")
	hundred, _ := parseGroupItemValue(groupItemMap, "PG06_78")
	if !(twelve < hundred) {
		t.Fatalf("12 (%v) ต้องน้อยกว่า 100 (%v)", twelve, hundred)
	}
}

// seedGroupItemsForSortTest เขียนค่าจริงจาก UAT ลงตาราง group / group_item
func seedGroupItemsForSortTest(t *testing.T) {
	t.Helper()

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("connect gorm: %v", err)
	}
	defer db.CloseGORM(gormx)

	groups := []struct {
		code string
		name string
	}{
		{"PG03", "เกรด/รูปแบบ"},
		{"PG05", "ขนาดหน้ากว้าง"},
		{"PG06", "ความหนา"},
	}

	groupIDs := map[string]uuid.UUID{}
	for _, g := range groups {
		id := uuid.New()
		groupIDs[g.code] = id
		if err := gormx.Exec(
			`INSERT INTO "group" (id, group_code, group_name, value, value_int, seq) VALUES (?, ?, ?, '', 0, 0)`,
			id, g.code, g.name,
		).Error; err != nil {
			t.Fatalf("insert group %s: %v", g.code, err)
		}
	}

	items := []struct {
		group string
		code  string
		name  string
		value string
	}{
		{"PG05", "PG05_22", "1250x8'", "10,000.00"},
		{"PG05", "PG05_3", "4' x 8'", "32.00"},
		{"PG05", "PG05_40", "ESP", ""},
		{"PG06", "PG06_4", "1.2", "1.20"},
		{"PG06", "PG06_62", "12", "12.00"},
		{"PG06", "PG06_78", "100", "100.00"},
		{"PG03", "PG03_10", "SS400", "10.00"},
		{"PG03", "PG03_21", "LT", "21.00"},
	}

	for _, it := range items {
		// value_int ตั้งเป็น 0 ตามที่ SyncGroupMaster ทำจริง เพื่อพิสูจน์ว่าเราไม่ได้พึ่งมัน
		if err := gormx.Exec(
			`INSERT INTO group_item (id, item_code, group_id, item_name, value, value_int) VALUES (?, ?, ?, ?, ?, 0)`,
			uuid.New(), it.code, groupIDs[it.group], it.name, it.value,
		).Error; err != nil {
			t.Fatalf("insert group_item %s: %v", it.code, err)
		}
	}

	t.Cleanup(func() {
		cleanup, err := db.ConnectGORM("prime_erp")
		if err != nil {
			return
		}
		defer db.CloseGORM(cleanup)
		for _, it := range items {
			cleanup.Exec(`DELETE FROM group_item WHERE item_code = ?`, it.code)
		}
		for _, g := range groups {
			cleanup.Exec(`DELETE FROM "group" WHERE id = ?`, groupIDs[g.code])
		}
	})
}
