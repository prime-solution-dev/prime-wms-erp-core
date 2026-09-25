//go:build integration

package priceService

import (
	"testing"
	"time"

	externalProductService "prime-erp-core/external/product-service"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
)

// TestIntegration_PricelistProductTab_FromDB seeds one price list group/subgroup/keys
// via the real schema (TestMain in upload-pricelist_integration_test.go), reads it back
// through getGroupSubGroup, and feeds that into buildPricelistProductTab to make sure the
// DB → key-matching → export-row pipeline lines up end to end.
func TestIntegration_PricelistProductTab_FromDB(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)
	sqlxDB := connectSqlxForTest(t)

	now := time.Now()
	groupID := uuid.New()
	subID := uuid.New()

	if err := gormx.Table("price_list_group").Create(map[string]any{
		"id": groupID, "company_code": "CPP", "site_code": "S1", "group_code": "GRP_IT",
		"group_name": "IT Group", "create_dtm": now, "update_dtm": now,
	}).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}

	if err := gormx.Table("price_list_sub_group").Create(map[string]any{
		"id": subID, "price_list_group_id": groupID, "subgroup_code": "SG_IT",
		"subgroup_key": "SG_IT_KEY", "total_net_price_weight": 21.5, "total_net_price_unit": 22.0,
		"create_dtm": now, "update_dtm": now,
	}).Error; err != nil {
		t.Fatalf("seed subgroup: %v", err)
	}

	for _, k := range []struct {
		seq         int
		code, value string
	}{
		{1, "PG01", "PG01_3"},
		{2, "PG02", "PG02_19"},
	} {
		if err := gormx.Table("price_list_sub_group_key").Create(map[string]any{
			"id": uuid.New(), "sub_group_id": subID, "seq": k.seq, "code": k.code, "value": k.value,
		}).Error; err != nil {
			t.Fatalf("seed subgroup key %s: %v", k.code, err)
		}
	}

	res, err := getGroupSubGroup(sqlxDB, GetPriceListGroupRequest{
		CompanyCode: "CPP", SiteCodes: []string{"S1"}, GroupCodes: []string{"GRP_IT"},
	})
	if err != nil {
		t.Fatalf("getGroupSubGroup: %v", err)
	}

	noName := func(string) string { return "" }
	noItem := func(string) (string, bool) { return "", false }

	products := []externalProductService.GetProductsComponent{
		{ProductCode: "AA", ProductName: "SS", ProductGroup: []models.ProductGroup{
			{GroupCode: "PG01", GroupValue: "PG01_3", Seq: 1, ActiveFlg: true},
			{GroupCode: "PG02", GroupValue: "PG02_19", Seq: 2, ActiveFlg: true},
		}},
		{ProductCode: "ZZ", ProductName: "No match", ProductGroup: []models.ProductGroup{
			{GroupCode: "PG01", GroupValue: "PG01_5", Seq: 1, ActiveFlg: true},
		}},
	}

	tab := buildPricelistProductTab(res, products, noName, noItem, nil, nil, nil, true)
	if len(tab.Rows) != 1 {
		t.Fatalf("onlyMatched rows = %d, want 1: %+v", len(tab.Rows), tab.Rows)
	}
	row := tab.Rows[0]
	if row["product_code"] != "AA" || row["subgroup_code"] != "SG_IT" ||
		row["price_per_kg"] != 21.5 || row["pricelist_group_code"] != "GRP_IT" {
		t.Fatalf("matched row wrong: %+v", row)
	}

	tabAll := buildPricelistProductTab(res, products, noName, noItem, nil, nil, nil, false)
	if len(tabAll.Rows) != 2 {
		t.Fatalf("all rows = %d, want 2: %+v", len(tabAll.Rows), tabAll.Rows)
	}
	if tabAll.Rows[0]["product_code"] != "AA" || tabAll.Rows[0]["subgroup_code"] != "SG_IT" {
		t.Fatalf("matched row (all) wrong: %+v", tabAll.Rows[0])
	}
	if tabAll.Rows[1]["product_code"] != "ZZ" || tabAll.Rows[1]["subgroup_code"] != "" {
		t.Fatalf("unmatched row (all) wrong: %+v", tabAll.Rows[1])
	}
}
