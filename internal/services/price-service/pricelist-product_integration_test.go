//go:build integration

package priceService

import (
	"errors"
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

// TestIntegration_GetPriceExportTable_PricelistProduct drives the Product Pricelist Report
// end to end through the public entrypoint (GetPriceExportTable), not just buildPricelistProductTab:
// DB (price list group/subgroup/keys, group/group_item/payment_term) + stubbed product master +
// a failing inventory-weight HTTP dependency that must not fail the whole export.
func TestIntegration_GetPriceExportTable_PricelistProduct(t *testing.T) {
	gormx := openTestDB(t)
	truncateAll(t, gormx)
	t.Cleanup(func() { truncateAll(t, gormx) })
	// ensureGroupPaymentTablesForTest มาจาก weight_spec_integration_test.go — สร้างตาราง
	// group/group_item/payment_term ที่ getGroupAndItemMappings ต้องใช้ (query DB ตรง ๆ ไม่ใช่ HTTP)
	// และเรียก ensureSqlxEnvForTest ให้ด้วย ซึ่งเป็นสาเหตุที่เทสนี้ล้มเมื่อรันเดี่ยว: GetPriceExportTable
	// เรียก db.ConnectSqlx ซึ่งอ่านคนละ env var จาก GORM ที่ TestMain ตั้งไว้ ต้อง map เองก่อนใช้
	ensureGroupPaymentTablesForTest(t)

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
		}, Units: unitsWithBaseWeight(10)},
		{ProductCode: "ZZ", ProductName: "No match", ProductGroup: []models.ProductGroup{
			{GroupCode: "PG01", GroupValue: "PG01_5", Seq: 1, ActiveFlg: true},
		}},
	}
	stubProducts := func(externalProductService.GetProductRequest) (externalProductService.GetProductsResponse, error) {
		return externalProductService.GetProductsResponse{Products: products}, nil
	}

	t.Run("All", func(t *testing.T) {
		getProducts = func(req externalProductService.GetProductRequest) (externalProductService.GetProductsResponse, error) {
			if req.CompanyCode[0] != "CPP" || req.SiteCode[0] != "S1" {
				t.Fatalf("unexpected product request: %+v", req)
			}
			return externalProductService.GetProductsResponse{Products: products}, nil
		}
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
		if len(tab.Rows) != 2 {
			t.Fatalf("rows = %d, want 2 (AA matched + ZZ unmatched): %+v", len(tab.Rows), tab.Rows)
		}
		aa := tab.Rows[0]
		if aa["product_code"] != "AA" || aa["total_weight"] != float64(10) || aa["pricelist_group_code"] != "GRP_IT" {
			t.Fatalf("AA row wrong: %+v", aa)
		}
		if tab.Rows[1]["product_code"] != "ZZ" {
			t.Fatalf("want ZZ unmatched row second, got %+v", tab.Rows[1])
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
