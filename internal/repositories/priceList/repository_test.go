package priceListRepository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"prime-erp-core/internal/db"
	"prime-erp-core/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var postgresContainer tc.Container

// TestMain sets up a Postgres container for all tests in this package
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
	postgresContainer = container

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
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, mapped.Port())

	// Point GORM connection to test DB
	os.Setenv("database_gorm_url_prime_erp", dsn)

	// Create minimal schema required for tests
	if err := createSchema(); err != nil {
		fmt.Printf("failed to create schema: %v\n", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()

	_ = postgresContainer.Terminate(ctx)
	os.Exit(code)
}

func createSchema() error {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormx)

	// Create tables (minimal columns used by repository and history)
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS price_list_group (
            id uuid PRIMARY KEY,
            company_code text,
            site_code text,
            group_code text,
            group_name text,
            price_unit double precision,
            price_weight double precision,
            before_price_unit double precision,
            before_price_weight double precision,
            currency text,
            effective_date timestamp NULL,
            remark text,
            group_key text,
            create_by text,
            create_dtm timestamp,
            update_by text,
            update_dtm timestamp
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group (
            id uuid PRIMARY KEY,
            price_list_group_id uuid REFERENCES price_list_group(id),
            subgroup_key text,
            is_trading boolean,
            price_unit double precision,
            extra_price_unit double precision,
            term_price_unit double precision,
            total_net_price_unit double precision,
            price_weight double precision,
            extra_price_weight double precision,
            term_price_weight double precision,
            total_net_price_weight double precision,
            before_price_unit double precision,
            before_extra_price_unit double precision,
            before_term_price_unit double precision,
            before_total_net_price_unit double precision,
            before_price_weight double precision,
            before_extra_price_weight double precision,
            before_term_price_weight double precision,
            before_total_net_price_weight double precision,
            effective_date timestamp NULL,
            remark text,
            create_by text,
            create_dtm timestamp,
            update_by text,
            update_dtm timestamp,
            udf_json json NULL
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group_history (
            id uuid PRIMARY KEY,
            price_list_group_id uuid,
            subgroup_key text,
            is_trading boolean,
            price_unit double precision,
            extra_price_unit double precision,
            term_price_unit double precision,
            total_net_price_unit double precision,
            price_weight double precision,
            extra_price_weight double precision,
            term_price_weight double precision,
            total_net_price_weight double precision,
            before_price_unit double precision,
            before_extra_price_unit double precision,
            before_term_price_unit double precision,
            before_total_net_price_unit double precision,
            before_price_weight double precision,
            before_extra_price_weight double precision,
            before_term_price_weight double precision,
            before_total_net_price_weight double precision,
            effective_date timestamp NULL,
            expiry_date timestamp NULL,
            remark text,
            create_by text,
            create_dtm timestamp,
            update_by text,
            update_dtm timestamp
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group_key (
            id uuid PRIMARY KEY,
            sub_group_id uuid REFERENCES price_list_sub_group(id),
            code text,
            value text,
            seq integer
        );`,
	}

	for _, s := range stmts {
		if err := gormx.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

func TestUpdatePriceListSubGroup_Integration(t *testing.T) {
	t.Run("Update with udf_json merging", func(t *testing.T) {
		gormx, err := db.ConnectGORM("prime_erp")
		if err != nil {
			t.Fatalf("db connect failed: %v", err)
		}
		defer db.CloseGORM(gormx)

		testGroupID := uuid.New()
		testGroup := models.PriceListGroup{
			ID:          testGroupID,
			CompanyCode: "TEST",
			SiteCode:    "TEST",
			GroupCode:   "TEST_GROUP",
			PriceUnit:   100.0,
			PriceWeight: 10.0,
			Currency:    "THB",
			CreateBy:    "test",
			CreateDtm:   time.Now(),
			UpdateBy:    "test",
			UpdateDtm:   time.Now(),
		}
		err = gormx.Create(&testGroup).Error
		assert.NoError(t, err, "create group")

		testSubGroupID := uuid.New()
		existingUdfJson := json.RawMessage(`{"is_highlight": false, "some_key": ""}`)
		now := time.Now()
		testSubGroup := models.PriceListSubGroup{
			ID:               testSubGroupID,
			PriceListGroupID: testGroupID,
			SubgroupKey:      "TEST_KEY",
			IsTrading:        false,
			PriceUnit:        100.0,
			UdfJson:          existingUdfJson,
			CreateBy:         "test",
			CreateDtm:        &now,
			UpdateBy:         "test",
			UpdateDtm:        &now,
		}
		err = gormx.Create(&testSubGroup).Error
		assert.NoError(t, err, "create subgroup")

		newUdfJson := json.RawMessage(`{"is_highlight": true}`)
		req := models.UpdatePriceListSubGroupRequest{SiteCode: "TEST", Changes: []models.UpdatePriceListSubGroupItem{{SubGroupID: testSubGroupID, UdfJson: newUdfJson}}}
		err = UpdatePriceListSubGroups(req)
		assert.NoError(t, err, "UpdatePriceListSubGroup")

		var updated models.PriceListSubGroup
		err = gormx.Where("id = ?", testSubGroupID).First(&updated).Error
		assert.NoError(t, err, "get updated")
		merged := map[string]interface{}{}
		if err := json.Unmarshal(updated.UdfJson, &merged); err != nil {
			t.Fatalf("unmarshal merged: %v", err)
		}
		v, ok := merged["is_highlight"].(bool)
		assert.True(t, ok && v, "is_highlight expected true, got %v", merged["is_highlight"])
		vs, ok := merged["some_key"].(string)
		assert.True(t, ok && vs == "", "some_key expected '', got %v", merged["some_key"])
	})

	t.Run("Update price fields updates before fields", func(t *testing.T) {
		gormx, err := db.ConnectGORM("prime_erp")
		if err != nil {
			t.Fatalf("db connect failed: %v", err)
		}
		defer db.CloseGORM(gormx)

		testGroupID := uuid.New()
		err = gormx.Create(&models.PriceListGroup{
			ID:          testGroupID,
			CompanyCode: "TEST",
			SiteCode:    "TEST",
			GroupCode:   "TEST_GROUP_2",
			PriceUnit:   100.0,
			PriceWeight: 10.0,
			Currency:    "THB",
			CreateBy:    "test",
			CreateDtm:   time.Now(),
			UpdateBy:    "test",
			UpdateDtm:   time.Now(),
		}).Error
		assert.NoError(t, err, "create group 2")

		testSubGroupID := uuid.New()
		oldPriceUnit := 100.0
		now := time.Now()
		err = gormx.Create(&models.PriceListSubGroup{
			ID:               testSubGroupID,
			PriceListGroupID: testGroupID,
			SubgroupKey:      "TEST_KEY_2",
			PriceUnit:        oldPriceUnit,
			CreateBy:         "test",
			CreateDtm:        &now,
			UpdateBy:         "test",
			UpdateDtm:        &now,
		}).Error
		assert.NoError(t, err, "create subgroup 2")

		newPriceUnit := 200.0
		err = UpdatePriceListSubGroups(models.UpdatePriceListSubGroupRequest{SiteCode: "TEST", Changes: []models.UpdatePriceListSubGroupItem{{SubGroupID: testSubGroupID, PriceUnit: &newPriceUnit}}})
		assert.NoError(t, err, "UpdatePriceListSubGroup 2")

		var updated models.PriceListSubGroup
		err = gormx.Where("id = ?", testSubGroupID).First(&updated).Error
		assert.NoError(t, err, "get updated 2")
		assert.Equal(t, newPriceUnit, updated.PriceUnit)
		assert.Equal(t, oldPriceUnit, updated.BeforePriceUnit)
	})
}

func TestGetPriceListSubGroupByID(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	defer db.CloseGORM(gormx)

	groupID := uuid.New()
	subGroupID := uuid.New()
	now := time.Now()

	// create base group (required for foreign key)
	err = gormx.Create(&models.PriceListGroup{
		ID:          groupID,
		CompanyCode: "TEST",
		SiteCode:    "TEST",
		GroupCode:   "TEST_GROUP_FETCH",
		PriceUnit:   10,
		PriceWeight: 20,
		Currency:    "THB",
		CreateBy:    "tester",
		CreateDtm:   now,
		UpdateBy:    "tester",
		UpdateDtm:   now,
	}).Error
	assert.NoError(t, err, "create price list group")

	// create target sub group
	err = gormx.Create(&models.PriceListSubGroup{
		ID:                        subGroupID,
		PriceListGroupID:          groupID,
		SubgroupKey:               "SUB_1",
		IsTrading:                 true,
		PriceUnit:                 11,
		ExtraPriceUnit:            1,
		TotalNetPriceUnit:         3,
		PriceWeight:               21,
		ExtraPriceWeight:          4,
		TermPriceWeight:           5,
		TotalNetPriceWeight:       6,
		BeforePriceUnit:           9,
		BeforeExtraPriceUnit:      8,
		BeforeTermPriceUnit:       7,
		BeforeTotalNetPriceUnit:   6,
		BeforePriceWeight:         5,
		BeforeExtraPriceWeight:    4,
		BeforeTermPriceWeight:     3,
		BeforeTotalNetPriceWeight: 2,
		Remark:                    "target",
		CreateBy:                  "tester",
		CreateDtm:                 &now,
		UpdateBy:                  "tester",
		UpdateDtm:                 &now,
	}).Error
	assert.NoError(t, err, "create sub group")

	// add key to target subgroup
	err = gormx.Create(&models.PriceListSubGroupKey{
		ID:         uuid.New(),
		SubGroupID: subGroupID,
		Code:       "CODE",
		Value:      "VALUE",
		Seq:        1,
	}).Error
	assert.NoError(t, err, "create subgroup key")

	result, err := GetPriceListSubGroupByID(subGroupID)
	assert.NoError(t, err, "repository fetch")
	if assert.NotNil(t, result, "expected result") {
		assert.Equal(t, subGroupID, result.ID)
		assert.Equal(t, groupID, result.PriceListGroupID)
		assert.Equal(t, groupID, result.PriceListGroup.ID)
		assert.Equal(t, "TEST_GROUP_FETCH", result.PriceListGroup.GroupCode)
		assert.Equal(t, "SUB_1", result.SubgroupKey)
		assert.True(t, result.IsTrading)
		assert.Equal(t, 11.0, result.PriceUnit)
		assert.Len(t, result.PriceListSubGroupKeys, 1, "subgroup keys loaded")
		assert.Equal(t, "CODE", result.PriceListSubGroupKeys[0].Code)
		assert.Equal(t, "VALUE", result.PriceListSubGroupKeys[0].Value)
	}

	// ensure not found case returns nil
	unknownID := uuid.New()
	result, err = GetPriceListSubGroupByID(unknownID)
	assert.NoError(t, err, "not found should not error")
	assert.Nil(t, result, "not found result expected nil")
}
