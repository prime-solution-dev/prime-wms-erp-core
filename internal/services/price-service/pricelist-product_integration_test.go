//go:build integration

package priceService

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prime-erp-core/config"
	externalProductService "prime-erp-core/external/product-service"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedProductPricelistGroup seeds one price_list_group / price_list_sub_group / keys row
// (company CPP, site S1, group keys PG01=PG01_3 + PG02=PG02_19) — reused by both the
// direct buildPricelistProductTab test and the GetPriceExportTable test below.
func seedProductPricelistGroup(t *testing.T, gormx *gorm.DB, groupCode, subgroupCode string) {
	t.Helper()
	now := time.Now()
	groupID := uuid.New()
	subID := uuid.New()

	if err := gormx.Table("price_list_group").Create(map[string]any{
		"id": groupID, "company_code": "CPP", "site_code": "S1", "group_code": groupCode,
		"group_name": "IT Group", "create_dtm": now, "update_dtm": now,
	}).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}

	if err := gormx.Table("price_list_sub_group").Create(map[string]any{
		"id": subID, "price_list_group_id": groupID, "subgroup_code": subgroupCode,
		"subgroup_key": subgroupCode + "_KEY", "total_net_price_weight": 21.5, "total_net_price_unit": 22.0,
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
}

// TestIntegration_PricelistProductTab_FromDB seeds one price list group/subgroup/keys
// via the real schema (TestMain in upload-pricelist_integration_test.go), reads it back
// through getGroupSubGroup, and feeds that into buildPricelistProductTab to make sure the
// DB → key-matching → export-row pipeline lines up end to end.
func TestIntegration_PricelistProductTab_FromDB(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)
	t.Cleanup(func() { truncateAll(t, gormx) })
	sqlxDB := connectSqlxForTest(t)

	seedProductPricelistGroup(t, gormx, "GRP_IT", "SG_IT")

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

// ensureGroupServiceSchema creates the "group" / "group_item" / "payment_term" tables that
// getGroupAndItemMappings (called from GetPriceExportTable) reads. Despite living under
// external/warehouse-service naming conventions elsewhere, group-service and payment-term
// resolution in THIS repo are plain GORM/sqlx queries against prime_erp, not HTTP calls —
// so the fixture here is schema, not an httptest server.
func ensureGroupServiceSchema(t *testing.T, gormx *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "group" (
			id uuid PRIMARY KEY, group_code text, group_name text, value text, value_int double precision,
			seq integer, create_dtm timestamp, update_by text, update_dtm timestamp, create_by text
		);`,
		`CREATE TABLE IF NOT EXISTS group_item (
			id uuid PRIMARY KEY, item_code text, group_id uuid REFERENCES "group"(id), item_name text, value text,
			parent_group_code text NULL, parent_group_item_code text NULL, value_int double precision,
			create_dtm timestamp, update_by text, update_dtm timestamp, create_by text
		);`,
		`CREATE TABLE IF NOT EXISTS payment_term (term_code text, term_type text, term_name text);`,
	}
	for _, s := range stmts {
		if err := gormx.Exec(s).Error; err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

// TestIntegration_GetPriceExportTable_PricelistProduct drives the Product Pricelist Report
// end to end through the public entrypoint (GetPriceExportTable), not just buildPricelistProductTab:
// DB (price list group/subgroup/keys, group/group_item/payment_term) + stubbed product master +
// a failing inventory-weight HTTP dependency that must not fail the whole export.
func TestIntegration_GetPriceExportTable_PricelistProduct(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)
	t.Cleanup(func() { truncateAll(t, gormx) })
	ensureGroupServiceSchema(t, gormx)

	seedProductPricelistGroup(t, gormx, "GRP_IT", "SG_IT")

	origGetProducts := getProducts
	origInvEndpoint := config.GET_INVENTORY_BY_KEY_ENDPOINT
	t.Cleanup(func() {
		getProducts = origGetProducts
		config.GET_INVENTORY_BY_KEY_ENDPOINT = origInvEndpoint
	})

	// externalService.GetInventoryWeightByKey ล้มเหลวแล้วต้อง log แล้วไปต่อ ไม่ทำให้ export ทั้งไฟล์ล้ม
	invServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(invServer.Close)
	config.GET_INVENTORY_BY_KEY_ENDPOINT = invServer.URL

	products := []externalProductService.GetProductsComponent{
		{ProductCode: "AA", ProductName: "SS", ProductGroup: []models.ProductGroup{
			{GroupCode: "PG01", GroupValue: "PG01_3", Seq: 1, ActiveFlg: true},
			{GroupCode: "PG02", GroupValue: "PG02_19", Seq: 2, ActiveFlg: true},
		}},
		{ProductCode: "ZZ", ProductName: "No match", ProductGroup: []models.ProductGroup{
			{GroupCode: "PG01", GroupValue: "PG01_5", Seq: 1, ActiveFlg: true},
		}},
	}
	stubProducts := func(externalProductService.GetProductRequest) (externalProductService.GetProductsResponse, error) {
		return externalProductService.GetProductsResponse{Products: products}, nil
	}

	t.Run("All", func(t *testing.T) {
		getProducts = stubProducts
		payload := `{"company_code":"CPP","site_codes":["S1"],"report_type":"PRICELIST_PRODUCT"}`

		res, err := GetPriceExportTable(nil, payload)
		if err != nil {
			t.Fatalf("GetPriceExportTable: %v", err)
		}
		resp, ok := res.(GetPriceExportTableResponse)
		if !ok || len(resp.Tabs) != 1 {
			t.Fatalf("unexpected response: %+v", res)
		}
		tab := resp.Tabs[0]
		if tab.Name != "Template" || tab.Headers.Report != "Pricelist Detail By Product" {
			t.Fatalf("unexpected tab shape: %+v", tab.Headers)
		}
		codes := map[string]bool{}
		for _, row := range tab.Rows {
			codes[fmt.Sprintf("%v", row["product_code"])] = true
		}
		if !codes["AA"] || !codes["ZZ"] {
			t.Fatalf("want AA (matched) and ZZ (unmatched) rows, got %+v", tab.Rows)
		}
	})

	t.Run("FilteredByGroupCodes", func(t *testing.T) {
		getProducts = stubProducts
		payload := `{"company_code":"CPP","site_codes":["S1"],"group_codes":["GRP_IT"],"report_type":"PRICELIST_PRODUCT"}`

		res, err := GetPriceExportTable(nil, payload)
		if err != nil {
			t.Fatalf("GetPriceExportTable: %v", err)
		}
		resp := res.(GetPriceExportTableResponse)
		if len(resp.Tabs[0].Rows) != 1 || resp.Tabs[0].Rows[0]["product_code"] != "AA" {
			t.Fatalf("want only AA (group_codes filter drops unmatched), got %+v", resp.Tabs[0].Rows)
		}
	})

	t.Run("GetProductsError", func(t *testing.T) {
		getProducts = func(externalProductService.GetProductRequest) (externalProductService.GetProductsResponse, error) {
			return externalProductService.GetProductsResponse{}, errors.New("boom")
		}
		payload := `{"company_code":"CPP","site_codes":["S1"],"report_type":"PRICELIST_PRODUCT"}`

		_, err := GetPriceExportTable(nil, payload)
		if err == nil || !strings.Contains(err.Error(), "failed to get products") {
			t.Fatalf("want error containing %q, got %v", "failed to get products", err)
		}
	})
}
