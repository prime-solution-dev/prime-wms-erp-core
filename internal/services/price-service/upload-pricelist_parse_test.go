package priceService

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// sheets maps a sheet name to its rows (row 0 is the header).
type sheets map[string][][]string

func buildXlsx(t *testing.T, sh sheets) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	for name, rows := range sh {
		if _, err := f.NewSheet(name); err != nil {
			t.Fatalf("new sheet %s: %v", name, err)
		}
		for i, row := range rows {
			cell, err := excelize.CoordinatesToCellName(1, i+1)
			if err != nil {
				t.Fatalf("cell name: %v", err)
			}
			vals := make([]interface{}, len(row))
			for j, v := range row {
				vals[j] = v
			}
			if err := f.SetSheetRow(name, cell, &vals); err != nil {
				t.Fatalf("set row: %v", err)
			}
		}
	}
	f.DeleteSheet("Sheet1")

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("write xlsx: %v", err)
	}
	return bytes.NewReader(buf.Bytes())
}

const (
	testCompany = "COMP-1"
	testSite    = "SITE-1"
)

func groupSheet(rows ...[]string) [][]string {
	out := [][]string{{"company_code", "site_code", "group_code", "group_name", "currency", "effective_date", "price_unit", "price_weight", "PG01"}}
	return append(out, rows...)
}

func subGroupSheet(rows ...[]string) [][]string {
	out := [][]string{{"company_code", "site_code", "group_code", "subgroup_code", "PG01", "PG02"}}
	return append(out, rows...)
}

// minimal workbook: the three sheets readSheet treats as mandatory.
func baseSheets() sheets {
	return sheets{
		"price_list_group": groupSheet(
			[]string{testCompany, testSite, "G1", "กลุ่ม 1", "THB", "2026-01-01", "18.5", "18.5", "PG01_1"},
		),
		"price_list_sub_group": subGroupSheet(
			[]string{testCompany, testSite, "G1", "SG01", "PG01_1", "PG02_2"},
		),
		"formulas_map": {{"subgroup_code", "formula_code_default", "formula_code_convert"}},
	}
}

func TestParse_PercentAndDecimalNotTruncated(t *testing.T) {
	sh := baseSheets()
	sh["price_list_group_term"] = [][]string{
		{"company_code", "site_code", "group_code", "term_code", "pdc", "pdc_percent", "due", "due_percent"},
		{testCompany, testSite, "G1", "T1", "0.19", "1.0%", "0.28", "1.5%"},
		{testCompany, testSite, "G1", "T2", "0.37", "0.02", "0.56", "3.0%"},
		{testCompany, testSite, "G1", "T3", "0", "", "0", ""},
	}
	sh["price_list_group_extra"] = [][]string{
		{"company_code", "site_code", "group_code", "condition_code", "operator", "value_int", "cond_range_min", "cond_range_max", "PG01"},
		{testCompany, testSite, "G1", "PG06", "to", "0.1", "3.2", "4.5", "PG01_1"},
	}

	req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(req.Terms) != 3 {
		t.Fatalf("terms = %d, want 3", len(req.Terms))
	}
	for i, want := range []struct{ pdc, due float64 }{{0.01, 0.015}, {0.02, 0.03}, {0, 0}} {
		if got := req.Terms[i].PdcPercent; got != want.pdc {
			t.Errorf("term[%d].PdcPercent = %v, want %v", i, got, want.pdc)
		}
		if got := req.Terms[i].DuePercent; got != want.due {
			t.Errorf("term[%d].DuePercent = %v, want %v", i, got, want.due)
		}
	}
	if len(req.Extras) != 1 || req.Extras[0].ValueInt != 0.1 {
		t.Fatalf("extra value_int = %+v, want 0.1", req.Extras)
	}
}

func TestParse_ExtraRowsWithSameKeyAreKept(t *testing.T) {
	sh := baseSheets()
	sh["price_list_group_extra"] = [][]string{
		{"company_code", "site_code", "group_code", "condition_code", "operator", "value_int", "cond_range_max", "PG01", "PG02"},
		{testCompany, testSite, "G1", "PG06", "=", "1.2", "1.2", "PG01_8", "PG02_17"},
		{testCompany, testSite, "G1", "PG06", "=", "0.5", "1.4", "PG01_8", "PG02_17"},
		{testCompany, testSite, "G1", "PG06", "=", "0.4", "1.5", "PG01_8", "PG02_17"},
		{testCompany, testSite, "G1", "PG06", "=", "0.3", "1.6", "PG01_8", "PG02_17"},
	}

	req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(req.Extras) != 4 {
		t.Fatalf("extras = %d, want 4 (same extra_key + condition_code must not collapse)", len(req.Extras))
	}
	seenRow := map[int]bool{}
	for i, e := range req.Extras {
		if e.ExtraKey != "PG01_8|PG02_17" {
			t.Errorf("extra[%d].ExtraKey = %q", i, e.ExtraKey)
		}
		if seenRow[e.RowNo] {
			t.Errorf("extra[%d] duplicate RowNo %d", i, e.RowNo)
		}
		seenRow[e.RowNo] = true
	}
	if req.Extras[0].RowNo != 2 || req.Extras[3].RowNo != 5 {
		t.Errorf("RowNo not 1-based sheet rows: %d..%d", req.Extras[0].RowNo, req.Extras[3].RowNo)
	}
}

func TestParse_DuplicateGroupCodeIsRejected(t *testing.T) {
	sh := baseSheets()
	sh["price_list_group"] = groupSheet(
		[]string{testCompany, testSite, "G1", "กลุ่ม 1", "THB", "2026-01-01", "18.5", "18.5", "PG01_1"},
		[]string{testCompany, testSite, "G1", "กลุ่ม 1 ซ้ำ", "THB", "2026-01-01", "30.8", "30.8", "PG01_2"},
	)

	_, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err == nil || !strings.Contains(err.Error(), "group_code") {
		t.Fatalf("err = %v, want duplicate group_code error", err)
	}
}

func TestParse_DuplicateSubGroupCodeIsRejected(t *testing.T) {
	sh := baseSheets()
	sh["price_list_sub_group"] = subGroupSheet(
		[]string{testCompany, testSite, "G1", "SG01", "PG01_1", "PG02_2"},
		[]string{testCompany, testSite, "G1", "SG01", "PG01_1", "PG02_3"},
	)

	_, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err == nil || !strings.Contains(err.Error(), "subgroup_code") {
		t.Fatalf("err = %v, want duplicate subgroup_code error", err)
	}
}

func formulaSheet(rows ...[]string) [][]string {
	out := [][]string{{"id", "formula_code", "name", "uom", "formula_type", "expression", "params", "rounding"}}
	return append(out, rows...)
}

func TestParse_MasterFormulasSheet(t *testing.T) {
	sh := baseSheets()
	sh["price_list_formulars"] = formulaSheet(
		[]string{"8a1148b8-321d-4c6a-a4ca-c6be0b68357d", "FM-1", "kg", "kg", "price_calc", "pcs/avg_kg_stock", `{"required":["pcs"]}`, "2"},
		[]string{"", "FM-2", "pcs", "pcs", "price_calc", "base_price*1.02", "", "0"},
	)
	sh["formulas_map"] = [][]string{
		{"subgroup_code", "formula_code_default", "formula_code_convert"},
		{"SG01", "FM-1", "FM-2"},
	}

	req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(req.Formulas) != 2 {
		t.Fatalf("formulas = %d, want 2", len(req.Formulas))
	}
	if req.Formulas[0].FormulaCode != "FM-1" || req.Formulas[0].Rounding != 2 {
		t.Errorf("formula[0] = %+v", req.Formulas[0])
	}
	if len(req.SubGroupFormulas) != 2 {
		t.Errorf("subgroup formulas = %d, want 2", len(req.SubGroupFormulas))
	}
}

func TestParse_FormulasMapUnknownCodeIsRejected(t *testing.T) {
	sh := baseSheets()
	sh["price_list_formulars"] = formulaSheet(
		[]string{"", "FM-1", "kg", "kg", "price_calc", "a/b", "", "2"},
	)
	sh["formulas_map"] = [][]string{
		{"subgroup_code", "formula_code_default", "formula_code_convert"},
		{"SG01", "FM-1", "FM-9"},
	}

	_, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err == nil || !strings.Contains(err.Error(), "FM-9") {
		t.Fatalf("err = %v, want unknown formula_code error", err)
	}
}

func TestParse_WithoutFormulasSheetIsAccepted(t *testing.T) {
	sh := baseSheets()
	sh["formulas_map"] = [][]string{
		{"subgroup_code", "formula_code_default", "formula_code_convert"},
		{"SG01", "FM-8", "FM-7"},
	}

	req, err := buildCreatePricelistRequestFromExcel(buildXlsx(t, sh))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(req.Formulas) != 0 {
		t.Errorf("formulas = %d, want 0", len(req.Formulas))
	}
	if len(req.SubGroupFormulas) != 2 {
		t.Errorf("subgroup formulas = %d, want 2", len(req.SubGroupFormulas))
	}
}
