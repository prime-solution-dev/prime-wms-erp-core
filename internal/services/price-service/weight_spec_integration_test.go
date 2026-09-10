//go:build integration

package priceService

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"prime-erp-core/config"
	"prime-erp-core/internal/db"

	"github.com/google/uuid"
)

// เทสนี้พิสูจน์บั๊กที่แก้ไป: weight_spec ต้องอ่านจาก field ระดับ result
// (config.GET_INVENTORY_BY_KEY_ENDPOINT) ไม่ใช่จาก inventory_weight[].total_weight
// และต้องมีค่าแม้ subgroup นั้นไม่มีสต็อก (inventory_weight ว่าง)
//
// เรียก transformToGetPriceListResponse ตรง ๆ (unexported, อยู่ใน package เดียวกัน)
// แทนที่จะเรียก GetPriceDetail เต็ม flow เพราะ GetPriceDetail ต้องพึ่ง getGroupSubGroup/
// getTerms/getExtras ที่อ่านจากตาราง price_list_group แบบเต็ม ซึ่งไม่จำเป็นต่อบั๊กนี้เลย —
// จุดที่ map weight_spec เข้า subgroup อยู่ใน transformToGetPriceListResponse ล้วน ๆ
// ฟังก์ชันนี้ยังคงพึ่ง DB จริงผ่าน getGroupAndItemMappings() (ตาราง group/group_item)
// และ GetPaymentTerm() (ตาราง payment_term) จึงต้องสร้างตารางเปล่าให้ผ่าน ensureGroupPaymentTablesForTest
//
// ไม่ parallel-safe เหมือนไฟล์ integration test อื่นในแพ็กเกจนี้ — แก้ package var
// config.GET_INVENTORY_BY_KEY_ENDPOINT ตรง ๆ (ไม่มี env var แยกต่อ endpoint ให้ตั้งผ่าน
// t.Setenv เพราะ config.Initialize() คำนวณทุก endpoint จาก base_url ตัวเดียว การแก้ package
// var ตรง ๆ จำกัดผลกระทบเฉพาะ endpoint ที่เทสนี้ใช้จริง) ห้ามใส่ t.Parallel()

func fakeWarehouseServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write fake warehouse response: %v", err)
		}
	}))
}

// ensureGroupPaymentTablesForTest สร้างตาราง group / group_item / payment_term แบบเปล่า
// เท่าที่ getGroupAndItemMappings() และ GetPaymentTerm() ต้องใช้ในการ query แบบไม่กรองอะไรเลย
// (ไม่มีอยู่ใน createPricelistSchema ของ upload-pricelist_integration_test.go เพราะเทสอื่น
// ในแพ็กเกจนี้ไม่เคยเรียก transformToGetPriceListResponse ตรง ๆ มาก่อน)
// ensureSqlxEnvForTest ตั้ง database_sqlx_url_prime_erp จาก database_gorm_url_prime_erp
// ที่ TestMain ตั้งไว้ — GetPaymentTerm() ต่อ DB ผ่าน db.ConnectSqlx ซึ่งอ่านคนละ env var
// จาก GORM (รูปแบบเดียวกับ connectSqlxForTest ใน upload-pricelist_integration_test.go)
func ensureSqlxEnvForTest(t *testing.T) {
	t.Helper()
	if os.Getenv("database_sqlx_url_prime_erp") == "" {
		dsn := os.Getenv("database_gorm_url_prime_erp")
		if dsn == "" {
			t.Fatal("TestMain did not set database_gorm_url_prime_erp")
		}
		os.Setenv("database_sqlx_url_prime_erp", dsn)
	}
}

func ensureGroupPaymentTablesForTest(t *testing.T) {
	t.Helper()
	ensureSqlxEnvForTest(t)

	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("connect gorm: %v", err)
	}
	defer db.CloseGORM(gormx)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS "group" (
			id uuid PRIMARY KEY,
			group_code text, group_name text, value text, value_int double precision,
			seq integer, create_dtm timestamp, update_by text, update_dtm timestamp, create_by text
		);`,
		`CREATE TABLE IF NOT EXISTS group_item (
			id uuid PRIMARY KEY,
			item_code text, group_id uuid, item_name text, value text,
			parent_group_code text, parent_group_item_code text, value_int double precision,
			create_dtm timestamp, update_by text, update_dtm timestamp, create_by text
		);`,
		`CREATE TABLE IF NOT EXISTS payment_term (
			term_code text, term_type text, term_name text
		);`,
	}
	for _, s := range stmts {
		if err := gormx.Exec(s).Error; err != nil {
			t.Fatalf("create support table: %v\n%s", err, s)
		}
	}
}

// pointWarehouseEndpointAt แก้ package var config.GET_INVENTORY_BY_KEY_ENDPOINT ให้ชี้ไปที่
// fake server แล้วคืนค่าเดิมกลับเมื่อจบเทส
func pointWarehouseEndpointAt(t *testing.T, url string) {
	t.Helper()
	original := config.GET_INVENTORY_BY_KEY_ENDPOINT
	config.GET_INVENTORY_BY_KEY_ENDPOINT = url
	t.Cleanup(func() { config.GET_INVENTORY_BY_KEY_ENDPOINT = original })
}

// buildSingleSubGroupResponse สร้าง []GetPriceListGroupResponse ที่มี 1 group / 1 subgroup
// พร้อม GroupKeys 1 ตัว (จำเป็นเพื่อให้ transformToGetPriceListResponse สร้าง keyValues
// แล้วยิงเรียก externalService.GetInventoryWeightByKey จริง)
func buildSingleSubGroupResponse(subGroupID uuid.UUID) []GetPriceListGroupResponse {
	return []GetPriceListGroupResponse{
		{
			PriceListGroup: PriceListGroup{
				ID:          uuid.New(),
				CompanyCode: "C1",
				SiteCode:    "S1",
				GroupCode:   "G1",
				SubGroups: []SubGroup{
					{
						ID:          subGroupID,
						SubGroupKey: "G1|PRODUCT|P001",
						GroupKeys: []GroupKey{
							{Code: "PRODUCT", Value: "P001", Seq: 1},
						},
					},
				},
			},
		},
	}
}

func TestGetPriceDetailWeightSpecIntegration_WithStock(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": [
			{"total_weight": 1500000, "total_qty": 100}
		]
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 1 {
		t.Fatalf("expected 1 group with 1 subgroup, got %#v", result)
	}

	got := result[0].SubGroups[0].WeightSpec
	if got != 12.5 {
		t.Fatalf("WeightSpec = %v, want 12.5 (must come from weight_spec field, not inventory_weight[].total_weight=1500000)", got)
	}
}

func TestGetPriceDetailWeightSpecIntegration_NoStock(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	body := `[{
		"id": "` + subGroupID.String() + `",
		"product_code": "P001",
		"supplier_code": "SUP001",
		"supplier_name": "Supplier 1",
		"weight_spec": 12.5,
		"inventory_weight": []
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 1 {
		t.Fatalf("expected 1 group with 1 subgroup, got %#v", result)
	}

	got := result[0].SubGroups[0].WeightSpec
	if got != 12.5 {
		t.Fatalf("WeightSpec = %v, want 12.5 — สินค้าไม่มีสต็อกต้องยังได้ weight_spec จาก product master (เคสที่พฤติกรรมเดิมพัง เพราะเคยอ่านจาก inventory_weight[] ที่ว่าง)", got)
	}
}

func TestGetPriceDetailWeightSpecIntegration_NoMatch(t *testing.T) {
	ensureGroupPaymentTablesForTest(t)

	subGroupID := uuid.New()
	// id ไม่ตรงกับ subGroupID (ค่าเริ่มต้น "") จึงไม่ match ใน inventoryMap/weightSpecMap เลย
	body := `[{
		"id": "",
		"product_code": "",
		"weight_spec": 0,
		"inventory_weight": []
	}]`
	srv := fakeWarehouseServer(t, body)
	defer srv.Close()
	pointWarehouseEndpointAt(t, srv.URL)

	result, err := transformToGetPriceListResponse(buildSingleSubGroupResponse(subGroupID))
	if err != nil {
		t.Fatalf("transformToGetPriceListResponse: %v", err)
	}

	if len(result) != 1 || len(result[0].SubGroups) != 1 {
		t.Fatalf("expected 1 group with 1 subgroup, got %#v", result)
	}

	got := result[0].SubGroups[0].WeightSpec
	if got != 0 {
		t.Fatalf("WeightSpec = %v, want 0 (no matching inventory record for this subgroup)", got)
	}
}
