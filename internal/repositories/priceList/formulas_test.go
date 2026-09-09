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
