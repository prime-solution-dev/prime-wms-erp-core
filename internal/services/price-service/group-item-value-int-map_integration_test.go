//go:build integration

package priceService

import (
	"testing"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// loadGroupItemValueInts ต้องคืน map ซ้อน group_code -> item_code -> value_int
// จากการยิง query ครั้งเดียว ไม่ใช่ต่อแถวเหมือน GetGroupItemValueInt เดิม
func TestLoadGroupItemValueIntsBuildsNestedMap(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.CloseGORM(gormx)

	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS "group" (
			id uuid PRIMARY KEY, group_code text, group_name text, value text,
			value_int double precision, seq integer,
			create_by text, create_dtm timestamp, update_by text, update_dtm timestamp
		);`,
		`CREATE TABLE IF NOT EXISTS group_item (
			id uuid PRIMARY KEY, item_code text, group_id uuid, item_name text, value text,
			parent_group_code text, parent_group_item_code text, value_int double precision,
			create_by text, create_dtm timestamp, update_by text, update_dtm timestamp
		);`,
		`TRUNCATE group_item, "group";`,
	} {
		if err := gormx.Exec(stmt).Error; err != nil {
			t.Fatalf("setup %q: %v", stmt, err)
		}
	}

	pg06, pg03, unused := uuid.New(), uuid.New(), uuid.New()
	if err := gormx.Exec(
		`INSERT INTO "group" (id, group_code, group_name) VALUES ($1,'PG06','size'), ($2,'PG03','thickness'), ($3,'PG99','unused')`,
		pg06, pg03, unused).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if err := gormx.Exec(
		`INSERT INTO group_item (id, item_code, group_id, value, value_int) VALUES
			($1,'PG06_40',$4,'40',40), ($2,'PG06_0',$4,'0',0), ($3,'PG99_1',$5,'1',1)`,
		uuid.New(), uuid.New(), uuid.New(), pg06, unused).Error; err != nil {
		t.Fatalf("seed group_item: %v", err)
	}

	subGroups := []models.PriceListSubGroup{
		subGroupWithConditions("PG06", ""),
		subGroupWithConditions("PG06", "PG03"),
	}

	values, err := loadGroupItemValueInts(subGroups)
	if err != nil {
		t.Fatalf("โหลดล้มเหลว: %v", err)
	}

	// group ที่ไม่ได้อยู่ใน condition_code ต้องไม่ถูกโหลดมาด้วย
	if _, ok := values["PG99"]; ok {
		t.Fatalf("PG99 ไม่ได้เป็น condition_code ต้องไม่ถูกโหลด: %v", values)
	}

	// PG03 มี group แต่ไม่มี item — ต้องมี entry ว่าง ไม่ใช่หายไป
	if items, ok := values["PG03"]; !ok || len(items) != 0 {
		t.Fatalf("PG03 ต้องได้ map ว่าง แต่ได้ %v (มี=%v)", items, ok)
	}

	for _, tc := range []struct {
		item      string
		wantValue float64
		wantFound bool
	}{
		{"PG06_40", 40, true},
		{"PG06_0", 0, true},
		{"PG06_99", 0, false},
	} {
		v, found := values.lookup("PG06", tc.item)
		if v != tc.wantValue || found != tc.wantFound {
			t.Fatalf("lookup(PG06, %s) = (%v, %v) ต้องเป็น (%v, %v)",
				tc.item, v, found, tc.wantValue, tc.wantFound)
		}
	}
}
