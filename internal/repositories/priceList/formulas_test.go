package priceListRepository

import (
	"testing"

	"prime-erp-core/internal/db"
)

// setupFormulaFixtures เตรียมสูตรราคาสองตัวและผูกเข้ากับ subgroup code สองตัว
// SG01 มีทั้งสูตร kg และ pcs ส่วน SG02 มีแค่สูตร kg
func setupFormulaFixtures(t *testing.T) {
	t.Helper()

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.CloseGORM(gormx)

	stmts := []string{
		`DELETE FROM price_list_subgroup_formulas_map;`,
		`DELETE FROM price_list_formulas;`,
		`INSERT INTO price_list_formulas (id, formula_code, name, uom, formula_type, expression, params, rounding, create_dtm)
         VALUES
           (gen_random_uuid(), 'FM-8', 'kg = Base price + Extra', 'kg', 'price_calc', '', '{}', 2, now()),
           (gen_random_uuid(), 'FM-7', 'Pcs = [kg]  x [Avg. kg stock]', 'pcs', 'price_calc', '', '{}', 2, now());`,
		`INSERT INTO price_list_subgroup_formulas_map (id, price_list_subgroup_code, price_list_formulas_code, is_default, create_dtm)
         VALUES
           (gen_random_uuid(), 'SG01', 'FM-8', true, now()),
           (gen_random_uuid(), 'SG01', 'FM-7', true, now()),
           (gen_random_uuid(), 'SG02', 'FM-8', true, now());`,
	}

	for _, s := range stmts {
		if err := gormx.Exec(s).Error; err != nil {
			t.Fatalf("failed to seed formula fixtures: %v", err)
		}
	}
}

func TestGetFormulasBySubgroupCodes(t *testing.T) {
	setupFormulaFixtures(t)

	got, err := GetFormulasBySubgroupCodes([]string{"SG01", "SG02"})
	if err != nil {
		t.Fatalf("GetFormulasBySubgroupCodes error: %v", err)
	}

	sg01 := got["SG01"]
	if len(sg01) != 2 {
		t.Fatalf("expected 2 formulas for SG01, got %d", len(sg01))
	}

	var kg, pcs *SubgroupFormula
	for i := range sg01 {
		switch sg01[i].Uom {
		case "kg":
			kg = &sg01[i]
		case "pcs":
			pcs = &sg01[i]
		}
	}
	if kg == nil {
		t.Fatal("expected a kg formula for SG01")
	}
	if kg.FormulaCode != "FM-8" {
		t.Fatalf("expected FM-8, got %q", kg.FormulaCode)
	}
	if kg.Name != "kg = Base price + Extra" {
		t.Fatalf("unexpected kg formula name: %q", kg.Name)
	}
	if pcs == nil {
		t.Fatal("expected a pcs formula for SG01")
	}
	if pcs.FormulaCode != "FM-7" {
		t.Fatalf("expected FM-7, got %q", pcs.FormulaCode)
	}

	if len(got["SG02"]) != 1 {
		t.Fatalf("expected 1 formula for SG02, got %d", len(got["SG02"]))
	}

	if _, ok := got["SG99"]; ok {
		t.Fatal("expected no entry for a subgroup code that was not requested")
	}
}

func TestGetFormulasBySubgroupCodes_EmptyInput(t *testing.T) {
	got, err := GetFormulasBySubgroupCodes(nil)
	if err != nil {
		t.Fatalf("expected no error for empty input, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(got))
	}
}

// setupSubGroupKeyFixtures สร้าง price list สองกลุ่มที่ใช้ PG ไม่เท่ากัน
// เพื่อยืนยันว่าการดึงชุดคอลัมน์ไม่ผูกกับกลุ่มใดกลุ่มหนึ่ง
func setupSubGroupKeyFixtures(t *testing.T) {
	t.Helper()

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.CloseGORM(gormx)

	stmts := []string{
		`DELETE FROM price_list_sub_group_key;`,
		`DELETE FROM price_list_sub_group;`,
		`DELETE FROM price_list_group;`,
		`INSERT INTO price_list_group (id, company_code, site_code, group_code, group_name)
         VALUES
           ('11111111-1111-1111-1111-111111111111', 'ACME', 'MAIN', 'GROUP_1_ITEM_1', 'กลุ่มหนึ่ง'),
           ('22222222-2222-2222-2222-222222222222', 'ACME', 'MAIN', 'GROUP_1_ITEM_2', 'กลุ่มสอง'),
           ('33333333-3333-3333-3333-333333333333', 'OTHER', 'MAIN', 'GROUP_1_ITEM_9', 'คนละบริษัท');`,
		`INSERT INTO price_list_sub_group (id, price_list_group_id, subgroup_code, subgroup_key)
         VALUES
           ('aaaaaaaa-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'SGA', 'k1'),
           ('bbbbbbbb-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 'SGB', 'k2'),
           ('cccccccc-3333-3333-3333-333333333333', '33333333-3333-3333-3333-333333333333', 'SGC', 'k3');`,
		`INSERT INTO price_list_sub_group_key (id, sub_group_id, code, value, seq)
         VALUES
           (gen_random_uuid(), 'aaaaaaaa-1111-1111-1111-111111111111', 'PG01', 'PG01_1', 1),
           (gen_random_uuid(), 'aaaaaaaa-1111-1111-1111-111111111111', 'PG02', 'PG02_1', 2),
           (gen_random_uuid(), 'bbbbbbbb-2222-2222-2222-222222222222', 'PG01', 'PG01_2', 1),
           (gen_random_uuid(), 'bbbbbbbb-2222-2222-2222-222222222222', 'PG06', 'PG06_1', 6),
           (gen_random_uuid(), 'cccccccc-3333-3333-3333-333333333333', 'PG09', 'PG09_1', 9);`,
	}

	for _, s := range stmts {
		if err := gormx.Exec(s).Error; err != nil {
			t.Fatalf("failed to seed subgroup key fixtures: %v", err)
		}
	}
}

func TestGetSubGroupKeyColumns(t *testing.T) {
	setupSubGroupKeyFixtures(t)

	got, err := GetSubGroupKeyColumns("ACME", []string{"MAIN"})
	if err != nil {
		t.Fatalf("GetSubGroupKeyColumns error: %v", err)
	}

	// ครอบทุกกลุ่มของบริษัทนี้ ไม่ใช่แค่กลุ่มเดียว และไม่ปนบริษัทอื่น
	codes := []string{}
	for _, c := range got {
		codes = append(codes, c.Code)
	}
	want := []string{"PG01", "PG02", "PG06"}
	if len(codes) != len(want) {
		t.Fatalf("expected %v, got %v", want, codes)
	}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("expected %v ordered by seq, got %v", want, codes)
		}
	}

	if got[0].Seq != 1 || got[2].Seq != 6 {
		t.Fatalf("unexpected seq values: %+v", got)
	}
}

func TestGetSubGroupKeyColumns_UnknownCompany(t *testing.T) {
	setupSubGroupKeyFixtures(t)

	got, err := GetSubGroupKeyColumns("NOPE", []string{"MAIN"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no columns, got %+v", got)
	}
}
