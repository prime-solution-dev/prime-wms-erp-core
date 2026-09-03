//go:build integration

package priceService

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "postgres:16",
		Env:          map[string]string{"POSTGRES_PASSWORD": "test", "POSTGRES_USER": "test", "POSTGRES_DB": "testdb"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("failed to get mapped port: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	os.Setenv("database_sqlx_url_prime_erp",
		fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mapped.Port()))

	if err := createLastUpdatedSchema(); err != nil {
		fmt.Printf("failed to create schema: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func createLastUpdatedSchema() error {
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		return err
	}
	defer sqlxDB.Close()

	_, err = sqlxDB.Exec(`
		CREATE TABLE IF NOT EXISTS price_list_group (
			id uuid PRIMARY KEY,
			company_code text,
			site_code text,
			group_code text,
			update_dtm timestamptz
		);
		CREATE TABLE IF NOT EXISTS price_list_sub_group (
			id uuid PRIMARY KEY,
			price_list_group_id uuid,
			update_dtm timestamptz
		);
	`)
	return err
}

func TestGetPriceLastUpdated_ReturnsLatestOfGroupAndSubGroup(t *testing.T) {
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlxDB.Close()

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

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
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlxDB.Close()

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

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
	sqlxDB, err := db.ConnectSqlx("prime_erp")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlxDB.Close()

	if _, err := sqlxDB.Exec(`DELETE FROM price_list_sub_group; DELETE FROM price_list_group;`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

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
