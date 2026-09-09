//go:build integration

package priceService

import (
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TestMain กับ schema ของแพ็กเกจนี้อยู่ใน upload-pricelist_integration_test.go
// ซึ่งสร้าง price_list_group และ price_list_sub_group พร้อมคอลัมน์ update_dtm ให้แล้ว
// ที่นั่นตั้งแต่ env ของ GORM ไว้อย่างเดียว ไฟล์นี้ใช้ sqlx จึงต้อง map ต่อเอง
//
// เทสต์ในไฟล์นี้ไม่ parallel-safe — แชร์ตารางเดียวกันกับเทสต์อื่นและ cleanup ด้วย
// DELETE FROM ก่อนเริ่มแต่ละเคส ห้ามใส่ t.Parallel()
func connectSqlxForTest(t *testing.T) *sqlx.DB {
	t.Helper()

	if os.Getenv("database_sqlx_url_prime_erp") == "" {
		dsn := os.Getenv("database_gorm_url_prime_erp")
		if dsn == "" {
			t.Fatal("TestMain did not set database_gorm_url_prime_erp")
		}
		os.Setenv("database_sqlx_url_prime_erp", dsn)
	}

	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { sqlxDB.Close() })

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	return sqlxDB
}

func TestGetPriceLastUpdated_ReturnsLatestOfGroupAndSubGroup(t *testing.T) {
	sqlxDB := connectSqlxForTest(t)

	groupID := uuid.New()
	groupUpdated := time.Date(2026, 8, 20, 3, 0, 0, 0, time.UTC)
	subGroupUpdated := time.Date(2026, 8, 22, 9, 30, 0, 0, time.UTC)

	if _, err := sqlxDB.Exec(
		`INSERT INTO price_list_group (id, company_code, site_code, group_code, update_dtm) VALUES ($1,$2,$3,$4,$5)`,
		groupID, "C1", "S1", "G1", groupUpdated,
	); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	if _, err := sqlxDB.Exec(
		`INSERT INTO price_list_sub_group (id, price_list_group_id, update_dtm) VALUES ($1,$2,$3)`,
		uuid.New(), groupID, subGroupUpdated,
	); err != nil {
		t.Fatalf("insert sub group: %v", err)
	}

	got, err := getPriceLastUpdated(sqlxDB, GetPriceListGroupRequest{CompanyCode: "C1", SiteCodes: []string{"S1"}})
	if err != nil {
		t.Fatalf("getPriceLastUpdated: %v", err)
	}
	if got == nil {
		t.Fatal("expected a timestamp, got nil")
	}
	if !got.UTC().Equal(subGroupUpdated) {
		t.Fatalf("expected %v, got %v", subGroupUpdated, got.UTC())
	}
}

func TestGetPriceLastUpdated_NoMatchingRowsReturnsNil(t *testing.T) {
	sqlxDB := connectSqlxForTest(t)

	got, err := getPriceLastUpdated(sqlxDB, GetPriceListGroupRequest{CompanyCode: "NOPE"})
	if err != nil {
		t.Fatalf("getPriceLastUpdated: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// group ที่ยังไม่มี sub group ต้องยังคืน update_dtm ของ group เอง
func TestGetPriceLastUpdated_GroupWithoutSubGroup(t *testing.T) {
	sqlxDB := connectSqlxForTest(t)

	groupUpdated := time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC)
	if _, err := sqlxDB.Exec(
		`INSERT INTO price_list_group (id, company_code, site_code, group_code, update_dtm) VALUES ($1,$2,$3,$4,$5)`,
		uuid.New(), "C1", "S1", "G1", groupUpdated,
	); err != nil {
		t.Fatalf("insert group: %v", err)
	}

	got, err := getPriceLastUpdated(sqlxDB, GetPriceListGroupRequest{CompanyCode: "C1"})
	if err != nil {
		t.Fatalf("getPriceLastUpdated: %v", err)
	}
	if got == nil || !got.UTC().Equal(groupUpdated) {
		t.Fatalf("expected %v, got %v", groupUpdated, got)
	}
}
