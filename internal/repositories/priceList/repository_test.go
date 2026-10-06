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
	"gorm.io/gorm"
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
            update_dtm timestamp,
            seq integer
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_sub_group (
            id uuid PRIMARY KEY,
            price_list_group_id uuid REFERENCES price_list_group(id),
            subgroup_code text,
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
		`CREATE TABLE IF NOT EXISTS price_list_group_extra (
            id uuid PRIMARY KEY,
            price_list_group_id uuid REFERENCES price_list_group(id),
            extra_key text,
            condition_code text,
            value_int double precision,
            length_extra_key integer,
            operator text,
            cond_range_min double precision,
            cond_range_max double precision,
            create_by text,
            create_dtm timestamp,
            update_by text,
            update_dtm timestamp
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_group_extra_key (
            id uuid PRIMARY KEY,
            group_extra_id uuid REFERENCES price_list_group_extra(id) ON DELETE CASCADE,
            code text,
            value text,
            seq integer
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_formulas (
            id uuid PRIMARY KEY,
            formula_code text UNIQUE NOT NULL,
            name text NOT NULL,
            uom text NOT NULL,
            formula_type text,
            expression text,
            params jsonb,
            rounding integer,
            create_dtm timestamp
        );`,
		`CREATE TABLE IF NOT EXISTS price_list_subgroup_formulas_map (
            id uuid PRIMARY KEY,
            price_list_subgroup_code text NOT NULL,
            price_list_formulas_code text NOT NULL,
            is_default boolean DEFAULT false,
            create_dtm timestamp
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

// schema ที่เขียนมือใน createSchema ต้องมีครบทุกคอลัมน์ที่ GORM model ประกาศ
//
// เคยพังมาแล้วตอนเพิ่ม seq เข้า price_list_group: migration กับ model อัปเดต
// แต่ DDL ในเทสต์ไม่ได้อัปเดตตาม เทสต์เลยล้มด้วย SQLSTATE 42703 ที่อ่านไม่รู้เรื่อง
// และล้มต่อเป็นลูกโซ่ไปที่ FK ของตารางลูก
//
// เทสต์นี้เทียบ field ของ model กับคอลัมน์จริงในตาราง เพื่อให้ครั้งหน้าที่มีใคร
// เพิ่มคอลัมน์ในโมเดล เทสต์บอกตรง ๆ ว่าขาดคอลัมน์ไหนในตารางไหน
func TestCreateSchemaCoversModelColumns(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	defer db.CloseGORM(gormx)

	migrator := gormx.Migrator()
	subjects := []interface{}{
		&models.PriceListGroup{},
		&models.PriceListSubGroup{},
	}

	for _, model := range subjects {
		stmt := &gorm.Statement{DB: gormx}
		if err := stmt.Parse(model); err != nil {
			t.Fatalf("parse model ไม่ได้: %v", err)
		}

		table := stmt.Schema.Table
		for _, field := range stmt.Schema.Fields {
			// ข้าม association ที่ไม่ได้เป็นคอลัมน์ของตารางนี้
			if field.DBName == "" {
				continue
			}
			if !migrator.HasColumn(model, field.DBName) {
				t.Errorf("ตาราง %s ขาดคอลัมน์ %q ที่ model ประกาศไว้ — เพิ่มใน createSchema ด้วย",
					table, field.DBName)
			}
		}
	}
}

// ลบ rule extra ออกแล้ว subgroup ที่ rule นั้นเคยบวกให้ต้องกลับเป็น 0
//
// calculateExtraForSubGroup คืนค่าเดิมเมื่อไม่มี rule ไหน key ตรง (กันค่าที่อัปโหลดมา
// หาย) ค่าที่ rule ที่ถูกลบเคยเขียนไว้จึงค้างตลอดไป ถ้า UpdateExtra ไม่ล้างให้ตอนลบ
// ข้อมูลจริง: GROUP_1_ITEM_2 ขนาด PG04_59 ค้าง extra 1.00 หลังลบ rule
func TestUpdateExtra_ResetsSubGroupsOrphanedByRemovedRule(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	defer db.CloseGORM(gormx)

	now := time.Now()
	groupID := uuid.New()
	assert.NoError(t, gormx.Create(&models.PriceListGroup{
		ID: groupID, CompanyCode: "TEST", SiteCode: "TEST", GroupCode: "TEST_EXTRA_ORPHAN",
		CreateDtm: now, UpdateDtm: now,
	}).Error)

	newSubGroup := func(size string, extra float64) uuid.UUID {
		id := uuid.New()
		assert.NoError(t, gormx.Create(&models.PriceListSubGroup{
			ID: id, PriceListGroupID: groupID, SubgroupKey: "PG01_7|" + size,
			ExtraPriceUnit: extra, ExtraPriceWeight: extra, CreateDtm: &now, UpdateDtm: &now,
			PriceListSubGroupKeys: []models.PriceListSubGroupKey{
				{ID: uuid.New(), SubGroupID: id, Code: "PG01", Value: "PG01_7", Seq: 1},
				{ID: uuid.New(), SubGroupID: id, Code: "PG04", Value: size, Seq: 2},
			},
		}).Error)
		return id
	}
	removedRuleSub := newSubGroup("PG04_59", 1.0) // rule ถูกลบ → ต้องเป็น 0
	keptRuleSub := newSubGroup("PG04_65", 0.6)    // rule ยังอยู่ → ไม่แตะ
	uploadedSub := newSubGroup("PG04_80", 2.5)    // ไม่เคยมี rule (ค่าจากอัปโหลด) → ไม่แตะ

	newExtra := func(size string, value float64) models.PriceListGroupExtra {
		id := uuid.New()
		return models.PriceListGroupExtra{
			ID: id, PriceListGroupID: groupID, ExtraKey: "PG01_7|" + size, ValueInt: value, LengthExtraKey: 2,
			PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
				{ID: uuid.New(), GroupExtraID: id, Code: "PG01", Value: "PG01_7", Seq: 1},
				{ID: uuid.New(), GroupExtraID: id, Code: "PG04", Value: size, Seq: 2},
			},
		}
	}
	removedRule := newExtra("PG04_59", 1.0)
	keptRule := newExtra("PG04_65", 0.6)
	assert.NoError(t, gormx.Create(&[]models.PriceListGroupExtra{removedRule, keptRule}).Error)

	// หน้าเว็บส่ง rule ทั้งชุดที่เหลือมา (ไม่มี PG04_59 แล้ว)
	assert.NoError(t, UpdateExtra([]models.PriceListGroupExtra{keptRule}))

	get := func(id uuid.UUID) models.PriceListSubGroup {
		var sg models.PriceListSubGroup
		assert.NoError(t, gormx.Where("id = ?", id).First(&sg).Error)
		return sg
	}

	removed := get(removedRuleSub)
	assert.Equal(t, 0.0, removed.ExtraPriceUnit)
	assert.Equal(t, 0.0, removed.ExtraPriceWeight)
	assert.Equal(t, 1.0, removed.BeforeExtraPriceUnit)
	assert.Equal(t, 1.0, removed.BeforeExtraPriceWeight)

	assert.Equal(t, 0.6, get(keptRuleSub).ExtraPriceUnit)
	assert.Equal(t, 2.5, get(uploadedSub).ExtraPriceUnit)

	var remaining int64
	assert.NoError(t, gormx.Model(&models.PriceListGroupExtra{}).Where("price_list_group_id = ?", groupID).Count(&remaining).Error)
	assert.Equal(t, int64(1), remaining)
}

// ค่าที่อัปโหลดมาต้องไม่ถูกล้างเมื่อไม่มี rule ไหนถูกลบ
func TestUpdateExtra_KeepsUploadedExtraWhenNoRuleRemoved(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	defer db.CloseGORM(gormx)

	setup := func(groupCode string) (uuid.UUID, uuid.UUID, models.PriceListGroupExtra) {
		now := time.Now()
		groupID := uuid.New()
		assert.NoError(t, gormx.Create(&models.PriceListGroup{
			ID: groupID, CompanyCode: "TEST", SiteCode: "TEST", GroupCode: groupCode, CreateDtm: now, UpdateDtm: now,
		}).Error)
		subID := uuid.New()
		assert.NoError(t, gormx.Create(&models.PriceListSubGroup{
			ID: subID, PriceListGroupID: groupID, SubgroupKey: "PG01_7|PG04_59",
			ExtraPriceUnit: 2.5, ExtraPriceWeight: 2.5, CreateDtm: &now, UpdateDtm: &now,
			PriceListSubGroupKeys: []models.PriceListSubGroupKey{
				{ID: uuid.New(), SubGroupID: subID, Code: "PG01", Value: "PG01_7", Seq: 1},
				{ID: uuid.New(), SubGroupID: subID, Code: "PG04", Value: "PG04_59", Seq: 2},
			},
		}).Error)
		extraID := uuid.New()
		extra := models.PriceListGroupExtra{
			ID: extraID, PriceListGroupID: groupID, ExtraKey: "PG01_7|PG04_59", ValueInt: 1, LengthExtraKey: 2,
			PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
				{ID: uuid.New(), GroupExtraID: extraID, Code: "PG01", Value: "PG01_7", Seq: 1},
				{ID: uuid.New(), GroupExtraID: extraID, Code: "PG04", Value: "PG04_59", Seq: 2},
			},
		}
		return groupID, subID, extra
	}
	extraOf := func(id uuid.UUID) float64 {
		var sg models.PriceListSubGroup
		assert.NoError(t, gormx.Where("id = ?", id).First(&sg).Error)
		return sg.ExtraPriceUnit
	}

	t.Run("first save with no previous rules", func(t *testing.T) {
		_, subID, extra := setup("TEST_EXTRA_FIRST_SAVE")
		assert.NoError(t, UpdateExtra([]models.PriceListGroupExtra{extra}))
		assert.Equal(t, 2.5, extraOf(subID))
	})

	t.Run("re-save the same rules", func(t *testing.T) {
		_, subID, extra := setup("TEST_EXTRA_RESAVE")
		assert.NoError(t, gormx.Create(&extra).Error)
		assert.NoError(t, UpdateExtra([]models.PriceListGroupExtra{extra}))
		assert.Equal(t, 2.5, extraOf(subID))
	})
}

func TestUpdateExtra_OrphanEdgeCases(t *testing.T) {
	gormx, err := db.ConnectGORM("prime_erp")
	if err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	defer db.CloseGORM(gormx)

	newGroup := func(code string) uuid.UUID {
		now := time.Now()
		id := uuid.New()
		assert.NoError(t, gormx.Create(&models.PriceListGroup{
			ID: id, CompanyCode: "TEST", SiteCode: "TEST", GroupCode: code, CreateDtm: now, UpdateDtm: now,
		}).Error)
		return id
	}
	newSubGroup := func(groupID uuid.UUID, size string, extra, beforeExtra float64) uuid.UUID {
		now := time.Now()
		id := uuid.New()
		assert.NoError(t, gormx.Create(&models.PriceListSubGroup{
			ID: id, PriceListGroupID: groupID, SubgroupKey: "PG01_7|" + size,
			ExtraPriceUnit: extra, ExtraPriceWeight: extra, BeforeExtraPriceUnit: beforeExtra,
			CreateDtm: &now, UpdateDtm: &now,
			PriceListSubGroupKeys: []models.PriceListSubGroupKey{
				{ID: uuid.New(), SubGroupID: id, Code: "PG01", Value: "PG01_7", Seq: 1},
				{ID: uuid.New(), SubGroupID: id, Code: "PG04", Value: size, Seq: 2},
			},
		}).Error)
		return id
	}
	newExtra := func(groupID uuid.UUID, size string) models.PriceListGroupExtra {
		id := uuid.New()
		return models.PriceListGroupExtra{
			ID: id, PriceListGroupID: groupID, ExtraKey: "PG01_7|" + size, ValueInt: 1, LengthExtraKey: 2, UpdateBy: "tester",
			PriceListGroupExtraKeys: []models.PriceListGroupExtraKey{
				{ID: uuid.New(), GroupExtraID: id, Code: "PG01", Value: "PG01_7", Seq: 1},
				{ID: uuid.New(), GroupExtraID: id, Code: "PG04", Value: size, Seq: 2},
			},
		}
	}
	get := func(id uuid.UUID) models.PriceListSubGroup {
		var sg models.PriceListSubGroup
		assert.NoError(t, gormx.Where("id = ?", id).First(&sg).Error)
		return sg
	}

	t.Run("rule key changed resets the old size", func(t *testing.T) {
		g := newGroup("TEST_EXTRA_KEY_CHANGED")
		oldSize := newSubGroup(g, "PG04_59", 1, 0)
		old := newExtra(g, "PG04_59")
		assert.NoError(t, gormx.Create(&old).Error)

		changed := newExtra(g, "PG04_60")
		changed.ID = old.ID
		for i := range changed.PriceListGroupExtraKeys {
			changed.PriceListGroupExtraKeys[i].GroupExtraID = old.ID
		}
		assert.NoError(t, UpdateExtra([]models.PriceListGroupExtra{changed}))

		sg := get(oldSize)
		assert.Equal(t, 0.0, sg.ExtraPriceUnit)
		assert.Equal(t, "tester", sg.UpdateBy)
	})

	t.Run("rules of another group in the payload do not cross match", func(t *testing.T) {
		a := newGroup("TEST_EXTRA_MULTI_A")
		b := newGroup("TEST_EXTRA_MULTI_B")
		orphanA := newSubGroup(a, "PG04_59", 1, 0)     // rule ของ A ถูกลบ → 0 แม้ B ยังมี rule PG04_59
		uploadedB := newSubGroup(b, "PG04_65", 2.5, 0) // B ไม่เคยมี rule PG04_65 แม้ A มี → ไม่แตะ
		assert.NoError(t, gormx.Create(&[]models.PriceListGroupExtra{
			newExtra(a, "PG04_59"), newExtra(a, "PG04_65"), newExtra(b, "PG04_59"),
		}).Error)

		assert.NoError(t, UpdateExtra([]models.PriceListGroupExtra{
			newExtra(a, "PG04_65"), newExtra(b, "PG04_59"),
		}))

		assert.Equal(t, 0.0, get(orphanA).ExtraPriceUnit)
		assert.Equal(t, 2.5, get(uploadedB).ExtraPriceUnit)
	})

	t.Run("orphan already at zero keeps its before snapshot", func(t *testing.T) {
		g := newGroup("TEST_EXTRA_ALREADY_ZERO")
		zero := newSubGroup(g, "PG04_59", 0, 0.7)
		assert.NoError(t, gormx.Create(&[]models.PriceListGroupExtra{newExtra(g, "PG04_59")}).Error)

		assert.NoError(t, UpdateExtra([]models.PriceListGroupExtra{newExtra(g, "PG04_65")}))

		assert.Equal(t, 0.7, get(zero).BeforeExtraPriceUnit)
	})
}
