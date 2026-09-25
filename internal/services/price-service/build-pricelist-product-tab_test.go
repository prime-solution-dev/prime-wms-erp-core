package priceService

import (
	"errors"
	"testing"

	externalProductService "prime-erp-core/external/product-service"
	"prime-erp-core/internal/models"
)

func TestProductKey_SortsBySeqAndDropsInactive(t *testing.T) {
	p := externalProductService.GetProductsComponent{
		ProductGroup: []models.ProductGroup{
			{GroupCode: "PG02", GroupValue: "PG02_19", Seq: 2, ActiveFlg: true},
			{GroupCode: "PG09", GroupValue: "PG09_1", Seq: 3, ActiveFlg: false},
			{GroupCode: "PG01", GroupValue: "PG01_3", Seq: 1, ActiveFlg: true},
		},
	}
	got := productKey(p)
	want := "PG01|PG02\x00PG01_3|PG02_19"
	if got != want {
		t.Fatalf("productKey = %q, want %q", got, want)
	}
}

func TestSubGroupKey_SortsBySeqAndKeepsAllKeys(t *testing.T) {
	sg := SubGroup{GroupKeys: []GroupKey{
		{Code: "PG02", Value: "PG02_19", Seq: 2},
		{Code: "", Value: "junk", Seq: 0},
		{Code: "PG01", Value: "PG01_3", Seq: 1},
	}}
	got := subGroupKey(sg)
	want := "|PG01|PG02\x00junk|PG01_3|PG02_19"
	if got != want {
		t.Fatalf("subGroupKey = %q, want %q", got, want)
	}
}

func TestProductKey_EmptyWhenNoActiveGroups(t *testing.T) {
	if got := productKey(externalProductService.GetProductsComponent{}); got != "" {
		t.Fatalf("want empty key, got %q", got)
	}
}

func TestProductKeyMatchesSubGroupKey(t *testing.T) {
	p := externalProductService.GetProductsComponent{
		ProductGroup: []models.ProductGroup{
			{GroupCode: "PG02", GroupValue: "PG02_19", Seq: 2, ActiveFlg: true},
			{GroupCode: "PG01", GroupValue: "PG01_3", Seq: 1, ActiveFlg: true},
		},
	}
	sg := SubGroup{GroupKeys: []GroupKey{
		{Code: "PG01", Value: "PG01_3", Seq: 1},
		{Code: "PG02", Value: "PG02_19", Seq: 2},
	}}
	if productKey(p) != subGroupKey(sg) {
		t.Fatalf("productKey(%q) != subGroupKey(%q)", productKey(p), subGroupKey(sg))
	}

	p.ProductGroup[0].GroupValue = "PG02_20"
	if productKey(p) == subGroupKey(sg) {
		t.Fatalf("productKey(%q) should not equal subGroupKey(%q)", productKey(p), subGroupKey(sg))
	}
}

func TestJoinKey_NoSeparatorCollision(t *testing.T) {
	a := joinKey([]keyPart{{code: "A", value: "B#C", seq: 1}})
	b := joinKey([]keyPart{{code: "A#B", value: "C", seq: 1}})
	if a == b {
		t.Fatalf("joinKey collision: %q == %q", a, b)
	}
}

func productFixtureGroups() []GetPriceListGroupResponse {
	sgA := SubGroup{
		SubgroupCode: "SG624", TotalNetPriceWeight: 21.5, TotalNetPriceUnit: 22,
		GroupKeys: []GroupKey{{Code: "PG01", Value: "PG01_3", Seq: 1}, {Code: "PG02", Value: "PG02_19", Seq: 2}},
	}
	sgInactive := SubGroup{
		SubgroupCode: "SG999", UdfJson: []byte(`{"inactive":true}`),
		GroupKeys: []GroupKey{{Code: "PG01", Value: "PG01_9", Seq: 1}},
	}
	sgB := SubGroup{
		SubgroupCode: "SG700", TotalNetPriceWeight: 30,
		GroupKeys: []GroupKey{{Code: "PG01", Value: "PG01_3", Seq: 1}, {Code: "PG02", Value: "PG02_19", Seq: 2}},
	}
	g1 := GetPriceListGroupResponse{PriceListGroup{GroupCode: "GROUP_1", GroupName: "หมวดเหล็กแผ่น", SubGroups: []SubGroup{sgA, sgInactive}}}
	g2 := GetPriceListGroupResponse{PriceListGroup{GroupCode: "GROUP_2", GroupName: "Group 2", SubGroups: []SubGroup{sgB}}}
	return []GetPriceListGroupResponse{g1, g2}
}

func pg(code, value string, seq int) models.ProductGroup {
	return models.ProductGroup{GroupCode: code, GroupValue: value, Seq: seq, ActiveFlg: true}
}

func productFixtures() []externalProductService.GetProductsComponent {
	return []externalProductService.GetProductsComponent{
		{ProductCode: "ZZ", ProductName: "No match", ProductGroup: []models.ProductGroup{pg("PG01", "PG01_5", 1)}},
		{ProductCode: "AA", ProductName: "SS", ProductGroup: []models.ProductGroup{pg("PG02", "PG02_19", 2), pg("PG01", "PG01_3", 1)}},
		{ProductCode: "INACT", ProductName: "matches inactive only", ProductGroup: []models.ProductGroup{pg("PG01", "PG01_9", 1)}},
	}
}

var testGroupName = func(code string) string {
	return map[string]string{"PG01": "หมวดหลัก", "PG02": "หมวดย่อย"}[code]
}

var testItemName = func(code string) (string, bool) {
	n, ok := map[string]string{"PG01_3": "หมวดเหล็กแผ่น", "PG01_5": "หมวดท่อ", "PG02_19": "เหล็กแผ่น special"}[code]
	return n, ok
}

func TestBuildPricelistProductTab_ColumnsPrefixProduct(t *testing.T) {
	tab := buildPricelistProductTab(productFixtureGroups(), productFixtures(), testGroupName, testItemName, nil, nil, nil, false)
	if tab.Name != "Template" || tab.Headers.Report != "Pricelist Detail By Product" {
		t.Fatalf("unexpected tab shape: %+v", tab.Headers)
	}
	wantHead := []string{"Product Code", "Product Name", "Pricelist group name", "หมวดหลัก", "หมวดย่อย", "Weight-spec"}
	for i, h := range wantHead {
		if tab.Columns[i].HeaderName != h {
			t.Fatalf("col %d = %q, want %q", i, tab.Columns[i].HeaderName, h)
		}
	}
	if last := tab.Columns[len(tab.Columns)-1].Field; last != "subgroup_code" {
		t.Fatalf("last column = %q, want subgroup_code", last)
	}
}

func TestBuildPricelistProductTab_AllIncludesUnmatchedSortedByProductCode(t *testing.T) {
	tab := buildPricelistProductTab(productFixtureGroups(), productFixtures(), testGroupName, testItemName, nil, nil, nil, false)
	// AA ตรง 2 subgroup (GROUP_1/SG624, GROUP_2/SG700), INACT ตรงแค่ subgroup inactive → ไม่ตรง, ZZ ไม่ตรง
	want := [][2]string{{"AA", "SG624"}, {"AA", "SG700"}, {"INACT", ""}, {"ZZ", ""}}
	if len(tab.Rows) != len(want) {
		t.Fatalf("rows = %d, want %d: %+v", len(tab.Rows), len(want), tab.Rows)
	}
	for i, w := range want {
		if tab.Rows[i]["product_code"] != w[0] || tab.Rows[i]["subgroup_code"] != w[1] {
			t.Fatalf("row %d = %v/%v, want %v", i, tab.Rows[i]["product_code"], tab.Rows[i]["subgroup_code"], w)
		}
	}
	if tab.Rows[0]["price_per_kg"] != 21.5 || tab.Rows[0]["product_name"] != "SS" || tab.Rows[0]["PG01"] != "หมวดเหล็กแผ่น" {
		t.Fatalf("matched row values wrong: %+v", tab.Rows[0])
	}
}

func TestBuildPricelistProductTab_UnmatchedRowUsesProductGroups(t *testing.T) {
	tab := buildPricelistProductTab(productFixtureGroups(), productFixtures(), testGroupName, testItemName, nil, nil, nil, false)
	zz := tab.Rows[3]
	if zz["PG01"] != "หมวดท่อ" || zz["PG01__code"] != "PG01_5" {
		t.Fatalf("unmatched PG cols wrong: %+v", zz)
	}
	// ZZ ไม่มี Units ในฟิกซ์เจอร์ → ไม่มี base unit → Weight-spec = 0, Avg. kg stock = 0 เสมอ
	if zz["total_weight"] != float64(0) || zz["avg_weight"] != float64(0) {
		t.Fatalf("unmatched weight cols wrong: total_weight=%v avg_weight=%v", zz["total_weight"], zz["avg_weight"])
	}
	for _, f := range []string{"pricelist_group_name", "pricelist_group_code", "price_per_kg", "PG02", "formula_kg_code"} {
		if zz[f] != "" {
			t.Fatalf("unmatched %s = %v, want empty", f, zz[f])
		}
	}
}

func unitsWithBaseWeight(w float64) []externalProductService.GetUnitsComponent {
	return []externalProductService.GetUnitsComponent{{FlagBase: true, Weight: w}}
}

func TestProductWeightSpec(t *testing.T) {
	cases := []struct {
		name  string
		units []externalProductService.GetUnitsComponent
		want  float64
	}{
		{"base unit wins over non-base", []externalProductService.GetUnitsComponent{{FlagBase: false, Weight: 5}, {FlagBase: true, Weight: 12.5}}, 12.5},
		{"no base unit", []externalProductService.GetUnitsComponent{{FlagBase: false, Weight: 5}}, 0},
		{"no units", nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := externalProductService.GetProductsComponent{Units: c.units}
			if got := productWeightSpec(p); got != c.want {
				t.Fatalf("productWeightSpec = %v, want %v", got, c.want)
			}
		})
	}
}

func TestBuildPricelistProductTab_MatchedRowsUseEachProductsOwnWeight(t *testing.T) {
	groups := []GetPriceListGroupResponse{
		{PriceListGroup{GroupCode: "G", SubGroups: []SubGroup{{
			SubgroupCode: "SG1",
			GroupKeys:    []GroupKey{{Code: "PG01", Value: "PG01_3", Seq: 1}},
		}}}},
	}
	products := []externalProductService.GetProductsComponent{
		{ProductCode: "P1", ProductGroup: []models.ProductGroup{pg("PG01", "PG01_3", 1)}, Units: unitsWithBaseWeight(10)},
		{ProductCode: "P2", ProductGroup: []models.ProductGroup{pg("PG01", "PG01_3", 1)}, Units: unitsWithBaseWeight(25)},
	}

	tab := buildPricelistProductTab(groups, products, testGroupName, testItemName, nil, nil, nil, false)
	if len(tab.Rows) != 2 {
		t.Fatalf("rows = %d, want 2: %+v", len(tab.Rows), tab.Rows)
	}
	if tab.Rows[0]["product_code"] != "P1" || tab.Rows[0]["total_weight"] != float64(10) {
		t.Fatalf("P1 row wrong: %+v", tab.Rows[0])
	}
	if tab.Rows[1]["product_code"] != "P2" || tab.Rows[1]["total_weight"] != float64(25) {
		t.Fatalf("P2 row wrong: %+v", tab.Rows[1])
	}
}

func TestBuildPricelistProductTab_UnmatchedIgnoresCodesOutsideColumns(t *testing.T) {
	products := []externalProductService.GetProductsComponent{
		{ProductCode: "X", ProductGroup: []models.ProductGroup{pg("PRODUCT_TYPE", "T1", 1)}},
	}
	tab := buildPricelistProductTab(productFixtureGroups(), products, testGroupName, testItemName, nil, nil, nil, false)
	if _, ok := tab.Rows[0]["PRODUCT_TYPE"]; ok {
		t.Fatalf("PRODUCT_TYPE is not a pricelist column and must not be written")
	}
}

func TestBuildPricelistProductTab_FilteredDropsUnmatched(t *testing.T) {
	groups := productFixtureGroups()[:1] // สมมติว่า query กรองเหลือ GROUP_1
	tab := buildPricelistProductTab(groups, productFixtures(), testGroupName, testItemName, nil, nil, nil, true)
	if len(tab.Rows) != 1 || tab.Rows[0]["product_code"] != "AA" || tab.Rows[0]["subgroup_code"] != "SG624" {
		t.Fatalf("filtered rows wrong: %+v", tab.Rows)
	}
}

func TestFetchAllProducts_SingleCallAllRowsAndDedupes(t *testing.T) {
	orig := getProducts
	defer func() { getProducts = orig }()
	calls := 0
	getProducts = func(req externalProductService.GetProductRequest) (externalProductService.GetProductsResponse, error) {
		calls++
		if req.Page != 0 || req.PageSize != 0 {
			t.Fatalf("want no paging (page=0, page_size=0), got page=%d page_size=%d", req.Page, req.PageSize)
		}
		if len(req.ActiveFlg) != 1 || !req.ActiveFlg[0] || req.CompanyCode[0] != "C1" || req.SiteCode[0] != "S1" {
			t.Fatalf("unexpected request: %+v", req)
		}
		return externalProductService.GetProductsResponse{
			Products: []externalProductService.GetProductsComponent{
				{ProductCode: "A"}, {ProductCode: "B"}, {ProductCode: "B"},
			},
		}, nil
	}
	got, err := fetchAllProducts("C1", []string{"S1"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(got) != 2 {
		t.Fatalf("calls=%d products=%d, want 1/2", calls, len(got))
	}
}

func TestFetchAllProducts_ReturnsError(t *testing.T) {
	orig := getProducts
	defer func() { getProducts = orig }()
	getProducts = func(externalProductService.GetProductRequest) (externalProductService.GetProductsResponse, error) {
		return externalProductService.GetProductsResponse{}, errors.New("boom")
	}
	if _, err := fetchAllProducts("C1", nil); err == nil {
		t.Fatal("want error")
	}
}

func TestBuildPricelistProductTab_SubGroupWithoutKeysNeverMatches(t *testing.T) {
	groups := []GetPriceListGroupResponse{
		{PriceListGroup{GroupCode: "G", SubGroups: []SubGroup{{SubgroupCode: "SG_EMPTY"}}}},
	}
	products := []externalProductService.GetProductsComponent{{ProductCode: "P0"}}

	tab := buildPricelistProductTab(groups, products, testGroupName, testItemName, nil, nil, nil, false)
	if len(tab.Rows) != 1 || tab.Rows[0]["product_code"] != "P0" || tab.Rows[0]["subgroup_code"] != "" {
		t.Fatalf("unmatched rows wrong: %+v", tab.Rows)
	}

	tabMatched := buildPricelistProductTab(groups, products, testGroupName, testItemName, nil, nil, nil, true)
	if len(tabMatched.Rows) != 0 {
		t.Fatalf("onlyMatched rows = %d, want 0: %+v", len(tabMatched.Rows), tabMatched.Rows)
	}
}
